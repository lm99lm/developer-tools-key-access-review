package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type event struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type auditRequest struct {
	Events []event `json:"events"`
}

type accountKey struct {
	ID        string   `json:"key_id"`
	Name      string   `json:"name"`
	ProjectID string   `json:"project_id"`
	Scopes    []string `json:"scopes"`
}

type reach struct {
	KeyID     string   `json:"key_id"`
	KeyName   string   `json:"key_name"`
	ProjectID string   `json:"project_id,omitempty"`
	Scopes    []string `json:"scopes"`
	Reachable []event  `json:"reachable"`
}

func parseKeys(data json.RawMessage) ([]accountKey, error) {
	var keys []accountKey
	if err := json.Unmarshal(data, &keys); err == nil {
		return keys, nil
	}
	var wrapped struct {
		Items []accountKey `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil || wrapped.Items == nil {
		return nil, fmt.Errorf("unexpected key list data")
	}
	return wrapped.Items, nil
}

func evaluate(keys []accountKey, events []event) []reach {
	result := make([]reach, 0, len(keys))
	for _, key := range keys {
		scopes := append([]string(nil), key.Scopes...)
		sort.Strings(scopes)
		row := reach{KeyID: key.ID, KeyName: key.Name, ProjectID: key.ProjectID, Scopes: scopes, Reachable: []event{}}
		for _, e := range events {
			for _, scope := range scopes {
				if scope == e.Scope {
					row.Reachable = append(row.Reachable, e)
					break
				}
			}
		}
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].KeyID < result[j].KeyID })
	return result
}

func document(day time.Time, rows []reach) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Developer-tools access review — %s\n\n", day.UTC().Format("2006-01-02"))
	for _, row := range rows {
		fmt.Fprintf(&b, "## %s (%s)\n\nProject: %s\n\nScopes: %s\n\n", row.KeyName, row.KeyID, row.ProjectID, strings.Join(row.Scopes, ", "))
		if len(row.Reachable) == 0 {
			b.WriteString("Reachable events: none\n\n")
		}
		for _, e := range row.Reachable {
			fmt.Fprintf(&b, "- %s: %s (scope: %s)\n", e.Kind, e.Name, e.Scope)
		}
		b.WriteString("\n")
	}
	return b.String()
}
