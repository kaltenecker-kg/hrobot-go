package hrobot

import (
	"context"
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

		// Doc-verbatim response shape for POST /traffic with single_values.
		const body = `{
			"traffic": {
				"type": "day",
				"from": "2024-01-01",
				"to": "2024-01-31",
				"data": {
					"123.123.123.123": {
						"2010-09-01": {
							"in": 0.2874,
							"out": 0.0481,
							"sum": 0.3355
						}
					}
				}
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
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

	if traffic.Type != "day" {
		t.Errorf("expected type 'day', got '%s'", traffic.Type)
	}
	if traffic.From != "2024-01-01" {
		t.Errorf("expected from '2024-01-01', got '%s'", traffic.From)
	}
	if traffic.To != "2024-01-31" {
		t.Errorf("expected to '2024-01-31', got '%s'", traffic.To)
	}
	if traffic.Data != nil {
		t.Errorf("expected Data to be nil in single_values mode, got %v", traffic.Data)
	}

	ipData, ok := traffic.SingleValues["123.123.123.123"]
	if !ok {
		t.Fatal("expected SingleValues to contain IP '123.123.123.123'")
	}
	stats, ok := ipData["2010-09-01"]
	if !ok {
		t.Fatal("expected SingleValues[ip] to contain interval '2010-09-01'")
	}
	if stats.In != 0.2874 {
		t.Errorf("expected In 0.2874, got %v", stats.In)
	}
	if stats.Out != 0.0481 {
		t.Errorf("expected Out 0.0481, got %v", stats.Out)
	}
	if stats.Sum != 0.3355 {
		t.Errorf("expected Sum 0.3355, got %v", stats.Sum)
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

		// Doc-verbatim response shape for POST /traffic without single_values.
		const body = `{
			"traffic": {
				"type": "month",
				"from": "2024-01-01",
				"to": "2024-01-31",
				"data": {
					"123.123.123.123": {
						"in": 0.2874,
						"out": 0.0481,
						"sum": 0.3355
					}
				}
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	traffic, err := client.Traffic.Get(ctx, TrafficGetParams{
		Type: TrafficTypeMonth,
		From: "2024-01-01",
		To:   "2024-01-31",
	})
	if err != nil {
		t.Fatalf("Traffic.Get returned error: %v", err)
	}

	if traffic.Type != "month" {
		t.Errorf("expected type 'month', got '%s'", traffic.Type)
	}
	if traffic.SingleValues != nil {
		t.Errorf("expected SingleValues to be nil in default mode, got %v", traffic.SingleValues)
	}

	stats, ok := traffic.Data["123.123.123.123"]
	if !ok {
		t.Fatal("expected Data to contain IP '123.123.123.123'")
	}
	if stats.In != 0.2874 {
		t.Errorf("expected In 0.2874, got %v", stats.In)
	}
	if stats.Out != 0.0481 {
		t.Errorf("expected Out 0.0481, got %v", stats.Out)
	}
	if stats.Sum != 0.3355 {
		t.Errorf("expected Sum 0.3355, got %v", stats.Sum)
	}
}
