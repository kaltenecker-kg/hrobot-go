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

		if got := r.FormValue("type"); got != "month" {
			t.Errorf("expected type 'month', got '%s'", got)
		}
		if got := r.FormValue("from"); got != "2019-01-01" {
			t.Errorf("expected from '2019-01-01', got '%s'", got)
		}
		if got := r.FormValue("to"); got != "2019-01-07" {
			t.Errorf("expected to '2019-01-07', got '%s'", got)
		}
		if got := r.FormValue("ip"); got != "123.123.123.123" {
			t.Errorf("expected ip '123.123.123.123', got '%s'", got)
		}
		if got := r.FormValue("single_values"); got != "true" {
			t.Errorf("expected single_values 'true', got '%s'", got)
		}

		// Doc-verbatim response shape for "Query traffic data grouped by
		// days for one IP" (POST /traffic with single_values=true).
		const body = `{
			"traffic": {
				"type": "month",
				"from": "2019-01-01",
				"to": "2019-01-07",
				"data": {
					"123.123.123.123": {
						"01": {
							"in": 0.0023,
							"out": 0.0102,
							"sum": 0.0125
						},
						"02": {
							"in": 229.7502,
							"out": 10.7187,
							"sum": 240.4689
						},
						"03": {
							"in": 97.8517,
							"out": 1.53,
							"sum": 99.3817
						},
						"04": {
							"in": 191.0187,
							"out": 0.153,
							"sum": 191.1717
						},
						"05": {
							"in": 0.0021,
							"out": 0.0022,
							"sum": 0.0043
						},
						"06": {
							"in": 0,
							"out": 0.0021,
							"sum": 0.0021
						},
						"07": {
							"in": 0,
							"out": 0,
							"sum": 0
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
		Type:         TrafficTypeMonth,
		From:         "2019-01-01",
		To:           "2019-01-07",
		IP:           "123.123.123.123",
		SingleValues: true,
	})
	if err != nil {
		t.Fatalf("Traffic.Get returned error: %v", err)
	}

	if traffic == nil {
		t.Fatal("expected non-nil traffic result")
	}

	if traffic.Type != "month" {
		t.Errorf("expected type 'month', got '%s'", traffic.Type)
	}
	if traffic.From != "2019-01-01" {
		t.Errorf("expected from '2019-01-01', got '%s'", traffic.From)
	}
	if traffic.To != "2019-01-07" {
		t.Errorf("expected to '2019-01-07', got '%s'", traffic.To)
	}
	if traffic.Data != nil {
		t.Errorf("expected Data to be nil in single_values mode, got %v", traffic.Data)
	}

	ipData, ok := traffic.SingleValues["123.123.123.123"]
	if !ok {
		t.Fatal("expected SingleValues to contain IP '123.123.123.123'")
	}

	day2, ok := ipData["02"]
	if !ok {
		t.Fatal("expected SingleValues[ip] to contain interval '02'")
	}
	if day2.In != 229.7502 {
		t.Errorf("expected In 229.7502, got %v", day2.In)
	}
	if day2.Out != 10.7187 {
		t.Errorf("expected Out 10.7187, got %v", day2.Out)
	}
	if day2.Sum != 240.4689 {
		t.Errorf("expected Sum 240.4689, got %v", day2.Sum)
	}

	day7, ok := ipData["07"]
	if !ok {
		t.Fatal("expected SingleValues[ip] to contain interval '07'")
	}
	if day7.In != 0 {
		t.Errorf("expected In 0, got %v", day7.In)
	}
	if day7.Out != 0 {
		t.Errorf("expected Out 0, got %v", day7.Out)
	}
	if day7.Sum != 0 {
		t.Errorf("expected Sum 0, got %v", day7.Sum)
	}
}

func TestTrafficService_Get_NoSingleValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("type"); got != "month" {
			t.Errorf("expected type 'month', got '%s'", got)
		}
		if got := r.FormValue("from"); got != "2010-09-01" {
			t.Errorf("expected from '2010-09-01', got '%s'", got)
		}
		if got := r.FormValue("to"); got != "2010-09-31" {
			t.Errorf("expected to '2010-09-31', got '%s'", got)
		}

		// single_values must NOT be set when false
		if _, ok := r.Form["single_values"]; ok {
			t.Errorf("expected single_values to be absent, got '%s'", r.FormValue("single_values"))
		}

		// ip must NOT be set when empty
		if _, ok := r.Form["ip"]; ok {
			t.Errorf("expected ip to be absent, got '%s'", r.FormValue("ip"))
		}

		// Doc-verbatim response shape for "Query traffic data for one IP"
		// (POST /traffic without single_values).
		const body = `{
			"traffic": {
				"type": "month",
				"from": "2010-09-01",
				"to": "2010-09-31",
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
		From: "2010-09-01",
		To:   "2010-09-31",
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
