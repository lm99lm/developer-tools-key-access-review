package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseKeys(t *testing.T) {
	got, err := parseKeys(json.RawMessage(`{"items":[{"key_id":"ifr_...1234","name":"build-key"}],"next_cursor":null,"count":1}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []accountKey{{ID: "ifr_...1234", Name: "build-key"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %#v, want %#v", got, want)
	}
}

func TestEvaluateAccess(t *testing.T) {
	events := []event{{"build", "compile", "build:read"}, {"release", "publish", "release:write"}, {"diagnostic", "trace", "diagnostic:read"}}
	tests := []struct {
		name   string
		scopes []string
		want   []event
	}{
		{"build only", []string{"build:read"}, events[:1]},
		{"build and diagnostics", []string{"diagnostic:read", "build:read"}, []event{events[0], events[2]}},
		{"unknown scope", []string{"other:read"}, []event{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluate([]accountKey{{ID: "key-1", Scopes: tt.scopes}}, events)[0].Reachable
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("reachable = %#v, want %#v", got, tt.want)
			}
		})
	}
}
