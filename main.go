package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("set INFRAI_API_KEY")
	}
	base := os.Getenv("INFRAI_BASE_URL")
	if base == "" {
		base = defaultBaseURL
	}
	c := client{baseURL: base, key: key, http: &http.Client{Timeout: 30 * time.Second}}
	http.HandleFunc("POST /audit", func(w http.ResponseWriter, r *http.Request) {
		var input auditRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil || len(input.Events) == 0 {
			http.Error(w, "provide events", http.StatusBadRequest)
			return
		}
		for _, e := range input.Events {
			if (e.Kind != "build" && e.Kind != "release" && e.Kind != "diagnostic") || e.Name == "" || e.Scope == "" {
				http.Error(w, "events need kind, name and scope", http.StatusBadRequest)
				return
			}
		}
		data, err := c.call(r.Context(), http.MethodGet, "/v1/account/keys/list", nil, "")
		if err != nil {
			respondError(w, err)
			return
		}
		keys, err := parseKeys(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		rows := evaluate(keys, input.Events)
		markdown := document(time.Now(), rows)
		digest := sha256.Sum256([]byte(markdown))
		pdf, err := c.call(r.Context(), http.MethodPost, "/v1/pdf/generate", map[string]any{"markdown": markdown, "store": false}, hex.EncodeToString(digest[:]))
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Keys []reach         `json:"keys"`
			PDF  json.RawMessage `json:"pdf"`
		}{rows, pdf})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func respondError(w http.ResponseWriter, err error) {
	var api apiError
	if errors.As(err, &api) {
		status := http.StatusBadGateway
		if api.Status >= 400 && api.Status < 500 {
			status = api.Status
		}
		http.Error(w, api.Error(), status)
		return
	}
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}
