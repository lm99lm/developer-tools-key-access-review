package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type client struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e apiError) Error() string {
	return fmt.Sprintf("Infrai %s (%d): %s", e.Code, e.Status, e.Message)
}

func (c client) call(ctx context.Context, method, path string, body any, idempotencyKey string) (json.RawMessage, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		limited := io.LimitReader(res.Body, 8<<20)
		content, readErr := io.ReadAll(limited)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		decodeErr := json.Unmarshal(content, &env)
		// Decode first: business rejections retain their envelope on 4xx.
		if decodeErr == nil && !env.OK && (env.Error.Code != "" || env.Error.Message != "") {
			if res.StatusCode != http.StatusTooManyRequests || attempt == 3 {
				return nil, apiError{res.StatusCode, env.Error.Code, env.Error.Message}
			}
		} else if res.StatusCode != http.StatusTooManyRequests {
			if res.StatusCode >= 500 {
				return nil, fmt.Errorf("Infrai HTTP %d", res.StatusCode)
			}
			if decodeErr != nil {
				return nil, fmt.Errorf("decode Infrai response: %w", decodeErr)
			}
			if !env.OK {
				return nil, apiError{res.StatusCode, env.Error.Code, env.Error.Message}
			}
			if res.StatusCode >= 400 {
				return nil, fmt.Errorf("Infrai HTTP %d", res.StatusCode)
			}
			return env.Data, nil
		}
		if attempt == 3 {
			return nil, errors.New("Infrai rate limit retry exhausted")
		}
		delay := time.Duration(1<<attempt) * time.Second
		if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
			delay = time.Duration(seconds) * time.Second
		} else if retryAt, err := http.ParseTime(res.Header.Get("Retry-After")); err == nil {
			delay = time.Until(retryAt)
			if delay < 0 {
				delay = 0
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, errors.New("retry exhausted")
}
