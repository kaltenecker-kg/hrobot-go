package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

// loadSpec loads the vendored OpenAPI spec once per test, failing the test
// immediately if it cannot be loaded or fails validation.
func loadSpec(t *testing.T) *spectest.Spec {
	t.Helper()
	spec, err := spectest.Load("spec/robot.yaml")
	if err != nil {
		t.Fatalf("failed to load spec/robot.yaml: %v", err)
	}
	return spec
}

func TestServerService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server" {
			t.Errorf("expected path '/server', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]any{
			{
				"server": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"server_name":   "server1",
					"product":       "EX41",
					"dc":            "FSN1-DC5",
					"traffic":       "unlimited",
					"status":        "ready",
					"cancelled":     false,
					"paid_until":    "2024-12-31",
					"ip":            []string{"123.123.123.123"},
					"subnet":        []map[string]any{},
				},
			},
			{
				"server": map[string]any{
					"server_ip":     "124.124.124.124",
					"server_number": 456,
					"server_name":   "server2",
					"product":       "AX41",
					"dc":            "NBG1-DC3",
					"traffic":       "5 TB",
					"status":        "ready",
					"cancelled":     false,
					"paid_until":    "2024-11-30",
					"ip":            []string{"124.124.124.124"},
					"subnet":        []map[string]any{},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	servers, err := client.Server.List(ctx)
	if err != nil {
		t.Fatalf("Server.List returned error: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}

	if servers[0].ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", servers[0].ServerNumber)
	}

	if servers[0].ServerName != "server1" {
		t.Errorf("expected server name 'server1', got '%s'", servers[0].ServerName)
	}

	if servers[0].Product != "EX41" {
		t.Errorf("expected product 'EX41', got '%s'", servers[0].Product)
	}

	if !servers[0].Traffic.Unlimited {
		t.Errorf("expected unlimited traffic, got limited")
	}

	if servers[1].ServerNumber != 456 {
		t.Errorf("expected server number 456, got %d", servers[1].ServerNumber)
	}

	// Check the "5 TB" traffic parsing
	if servers[1].Traffic.Bytes != 5497558138880 {
		t.Errorf("expected traffic bytes 5497558138880 (5 TB), got %d", servers[1].Traffic.Bytes)
	}
	if servers[1].Traffic.Raw != "5 TB" {
		t.Errorf("expected traffic Raw '5 TB', got '%s'", servers[1].Traffic.Raw)
	}
}

func TestServerService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321" {
			t.Errorf("expected path '/server/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]any{
			"server": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "test-server",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       "unlimited",
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet": []map[string]any{
					{
						"ip":   "123.123.123.128",
						"mask": "255.255.255.192",
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	srv, err := client.Server.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Server.Get returned error: %v", err)
	}

	if srv.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", srv.ServerNumber)
	}

	if srv.ServerName != "test-server" {
		t.Errorf("expected server name 'test-server', got '%s'", srv.ServerName)
	}

	if srv.DC != "FSN1-DC5" {
		t.Errorf("expected DC 'FSN1-DC5', got '%s'", srv.DC)
	}

	if len(srv.Subnet) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(srv.Subnet))
	}
}

func TestServerService_SetName(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321" {
			t.Errorf("expected path '/server/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("server_name") != "new-name" {
			t.Errorf("expected server_name 'new-name', got '%s'", r.FormValue("server_name"))
		}

		response := map[string]any{
			"server": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "new-name",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       "unlimited",
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet":        []map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	srv, err := client.Server.SetName(ctx, ServerID(321), "new-name")
	if err != nil {
		t.Fatalf("Server.SetName returned error: %v", err)
	}

	if srv.ServerName != "new-name" {
		t.Errorf("expected server name 'new-name', got '%s'", srv.ServerName)
	}
}

func TestServerService_RequestCancellation_DisallowedByPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("RequestCancellation must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))

	err := client.Server.RequestCancellation(context.Background(), Cancellation{
		ServerID:         ServerID(321),
		CancellationDate: "2024-12-31",
	})
	if !IsPolicyError(err) {
		t.Fatalf("expected policy error, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 451 {
		t.Fatalf("expected status 451, got %v", err)
	}
}

func TestServerService_WithdrawCancellation(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321/cancellation" {
			t.Errorf("expected path '/server/321/cancellation', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Server.WithdrawCancellation(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Server.WithdrawCancellation returned error: %v", err)
	}
}

func TestServerService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			method:     "list",
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
		},
		{
			name:       "SetName unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "setname",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"status":  tt.statusCode,
						"code":    "ERROR",
						"message": "test error",
					},
				})
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			var err error
			switch tt.method {
			case "list":
				_, err = client.Server.List(ctx)
			case "get":
				_, err = client.Server.Get(ctx, ServerID(321))
			case "setname":
				_, err = client.Server.SetName(ctx, ServerID(321), "new-name")
			}

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
