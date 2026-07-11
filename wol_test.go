package hrobot

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func wolFixture() string {
	return `{
		"wol": {
			"server_ip": "123.123.123.123",
			"server_ipv6_net": "2a01:4f8:111:4221::",
			"server_number": 321
		}
	}`
}

func TestWOLService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wol/321" {
			t.Errorf("expected path '/wol/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}

		// POST /wol/{server-number} takes no input parameters (doc example
		// sends an empty body via `-d ''`).
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("expected empty request body, got '%s'", body)
		}

		if _, err := w.Write([]byte(wolFixture())); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	wol, err := client.WOL.Send(context.Background(), ServerID(321))
	if err != nil {
		t.Fatalf("WOL.Send returned error: %v", err)
	}
	if wol.ServerIP != "123.123.123.123" {
		t.Errorf("expected server_ip '123.123.123.123', got '%s'", wol.ServerIP)
	}
	if wol.ServerIPv6Net != "2a01:4f8:111:4221::" {
		t.Errorf("expected server_ipv6_net '2a01:4f8:111:4221::', got '%s'", wol.ServerIPv6Net)
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
		if _, err := w.Write([]byte(wolFixture())); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	wol, err := client.WOL.Get(context.Background(), ServerID(321))
	if err != nil {
		t.Fatalf("WOL.Get returned error: %v", err)
	}
	if wol.ServerIP != "123.123.123.123" {
		t.Errorf("expected server_ip '123.123.123.123', got '%s'", wol.ServerIP)
	}
	if wol.ServerIPv6Net != "2a01:4f8:111:4221::" {
		t.Errorf("expected server_ipv6_net '2a01:4f8:111:4221::', got '%s'", wol.ServerIPv6Net)
	}
	if wol.ServerNumber != 321 {
		t.Errorf("expected server_number 321, got %d", wol.ServerNumber)
	}
}
