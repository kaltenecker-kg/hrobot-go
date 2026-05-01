package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func wolFixture() map[string]any {
	return map[string]any{
		"wol": map[string]any{
			"server_ip":       "123.123.123.123",
			"server_ipv6_net": "2a01:4f8:111:4221::",
			"server_number":   321,
		},
	}
}

func TestWOLService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wol/321" {
			t.Errorf("expected path '/wol/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(wolFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	wol, err := client.WOL.Send(context.Background(), ServerID(321))
	if err != nil {
		t.Fatalf("WOL.Send returned error: %v", err)
	}
	if wol.ServerNumber != 321 {
		t.Errorf("expected server_number 321, got %d", wol.ServerNumber)
	}
}

func TestWOLService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wol/321" {
			t.Errorf("expected path '/wol/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(wolFixture())
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	wol, err := client.WOL.Get(context.Background(), ServerID(321))
	if err != nil {
		t.Fatalf("WOL.Get returned error: %v", err)
	}
	if wol.ServerNumber != 321 {
		t.Errorf("expected server_number 321, got %d", wol.ServerNumber)
	}
}
