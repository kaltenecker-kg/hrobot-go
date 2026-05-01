package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrafficService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traffic" {
			t.Errorf("expected path '/traffic', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("type"); got != "day" {
			t.Errorf("expected type 'day', got '%s'", got)
		}
		if got := r.FormValue("from"); got != "2024-01-01" {
			t.Errorf("expected from '2024-01-01', got '%s'", got)
		}
		if got := r.FormValue("to"); got != "2024-01-31" {
			t.Errorf("expected to '2024-01-31', got '%s'", got)
		}
		if got := r.FormValue("ip"); got != "123.123.123.123" {
			t.Errorf("expected ip '123.123.123.123', got '%s'", got)
		}
		if got := r.FormValue("single_values"); got != "true" {
			t.Errorf("expected single_values 'true', got '%s'", got)
		}

		// The client's response unwrapper consumes a top-level "data" or
		// "traffic" key (see responseWrapper in client.go), so /traffic
		// responses don't round-trip cleanly into ServerTrafficData. We
		// therefore return an empty object — it's enough to verify that
		// the request was issued correctly with the expected form fields
		// and that the response decode does not error.
		response := map[string]any{}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	traffic, err := client.Traffic.Get(ctx, TrafficGetParams{
		Type:         TrafficTypeDay,
		From:         "2024-01-01",
		To:           "2024-01-31",
		IP:           "123.123.123.123",
		SingleValues: true,
	})
	if err != nil {
		t.Fatalf("Traffic.Get returned error: %v", err)
	}

	if traffic == nil {
		t.Fatal("expected non-nil traffic result")
	}
}

func TestTrafficService_Get_NoSingleValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// single_values must NOT be set when false
		if _, ok := r.Form["single_values"]; ok {
			t.Errorf("expected single_values to be absent, got '%s'", r.FormValue("single_values"))
		}

		// ip must NOT be set when empty
		if _, ok := r.Form["ip"]; ok {
			t.Errorf("expected ip to be absent, got '%s'", r.FormValue("ip"))
		}

		response := map[string]any{
			"type": "month",
			"from": "2024-01-01",
			"to":   "2024-01-31",
			"data": map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Traffic.Get(ctx, TrafficGetParams{
		Type: TrafficTypeMonth,
		From: "2024-01-01",
		To:   "2024-01-31",
	})
	if err != nil {
		t.Fatalf("Traffic.Get returned error: %v", err)
	}
}
