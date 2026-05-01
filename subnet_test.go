package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubnetService_Cancel_DisallowedByPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("Cancel must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))

	_, err := client.Subnet.Cancel(context.Background(), "2001:db8::", "2024-12-31")
	if !IsPolicyError(err) {
		t.Fatalf("expected policy error, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 451 {
		t.Fatalf("expected status 451, got %v", err)
	}
}

func subnetFixture() map[string]interface{} {
	return map[string]interface{}{
		"subnet": map[string]interface{}{
			"ip":               "2a01:4f8:111:4221::",
			"mask":             64,
			"gateway":          "2a01:4f8:111:4221::1",
			"server_ip":        "88.198.123.123",
			"server_number":    321,
			"failover":         false,
			"locked":           false,
			"traffic_warnings": false,
			"traffic_hourly":   100,
			"traffic_daily":    500,
			"traffic_monthly":  2,
		},
	}
}

func TestSubnetService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subnet" {
			t.Errorf("expected '/subnet', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{subnetFixture()})
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	subnets, err := client.Subnet.List(context.Background())
	if err != nil {
		t.Fatalf("Subnet.List returned error: %v", err)
	}
	if len(subnets) != 1 || subnets[0].ServerNumber != 321 {
		t.Errorf("unexpected result: %+v", subnets)
	}
}

func TestSubnetService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subnet/2a01:4f8:111:4221::" {
			t.Errorf("expected subnet path, got '%s'", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(subnetFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	subnet, err := client.Subnet.Get(context.Background(), "2a01:4f8:111:4221::")
	if err != nil {
		t.Fatalf("Subnet.Get returned error: %v", err)
	}
	if subnet.Mask != 64 {
		t.Errorf("expected mask 64, got %d", subnet.Mask)
	}
}

func TestSubnetService_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("traffic_warnings") != "true" {
			t.Errorf("traffic_warnings: got '%s'", r.FormValue("traffic_warnings"))
		}
		if r.FormValue("traffic_monthly") != "9" {
			t.Errorf("traffic_monthly: got '%s'", r.FormValue("traffic_monthly"))
		}
		_ = json.NewEncoder(w).Encode(subnetFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if _, err := client.Subnet.Update(context.Background(), "2a01:4f8:111:4221::", true, 1, 5, 9); err != nil {
		t.Fatalf("Subnet.Update returned error: %v", err)
	}
}

func subnetMACFixture() map[string]interface{} {
	return map[string]interface{}{
		"mac": map[string]interface{}{
			"ip":   "2a01:4f8:111:4221::",
			"mask": "64",
			"mac":  "00:21:85:62:3e:9c",
			"possible_mac": map[string]interface{}{
				"123.123.123.123": "00:21:85:62:3e:9c",
				"123.123.123.124": "00:21:85:62:3e:9d",
			},
		},
	}
}

func TestSubnetService_GetMAC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(subnetMACFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	mac, err := client.Subnet.GetMAC(context.Background(), "2a01:4f8:111:4221::")
	if err != nil {
		t.Fatalf("Subnet.GetMAC returned error: %v", err)
	}
	if mac.MAC != "00:21:85:62:3e:9c" {
		t.Errorf("expected mac, got '%s'", mac.MAC)
	}
	if len(mac.PossibleMAC) != 2 {
		t.Errorf("expected 2 possible_mac entries, got %d", len(mac.PossibleMAC))
	}
}

func TestSubnetService_SetMAC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got '%s'", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("mac") != "00:21:85:62:3e:9c" {
			t.Errorf("expected mac form value, got '%s'", r.FormValue("mac"))
		}
		_ = json.NewEncoder(w).Encode(subnetMACFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if _, err := client.Subnet.SetMAC(context.Background(), "2a01:4f8:111:4221::", "00:21:85:62:3e:9c"); err != nil {
		t.Fatalf("Subnet.SetMAC returned error: %v", err)
	}
}

func TestSubnetService_DeleteMAC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if err := client.Subnet.DeleteMAC(context.Background(), "2a01:4f8:111:4221::"); err != nil {
		t.Fatalf("Subnet.DeleteMAC returned error: %v", err)
	}
}

func TestSubnetService_GetCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"cancellation": map[string]interface{}{
				"ip":                         "2a01:4f8:111:4221::",
				"mask":                       "64",
				"server_number":              321,
				"earliest_cancellation_date": "2026-06-01",
				"cancelled":                  false,
				"cancellation_date":          nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	c, err := client.Subnet.GetCancellation(context.Background(), "2a01:4f8:111:4221::")
	if err != nil {
		t.Fatalf("Subnet.GetCancellation returned error: %v", err)
	}
	if c.Cancelled {
		t.Error("expected cancelled false")
	}
	if c.CancellationDate != nil {
		t.Errorf("expected nil cancellation_date, got %v", *c.CancellationDate)
	}
}

func TestSubnetService_WithdrawCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if err := client.Subnet.WithdrawCancellation(context.Background(), "2a01:4f8:111:4221::"); err != nil {
		t.Fatalf("Subnet.WithdrawCancellation returned error: %v", err)
	}
}
