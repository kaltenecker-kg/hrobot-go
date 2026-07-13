package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

func TestRDNSService_List(t *testing.T) {
	tests := []struct {
		name     string
		serverIP string
		wantPath string
	}{
		{
			name:     "list all",
			serverIP: "",
			wantPath: "/rdns",
		},
		{
			name:     "list filtered by server IP",
			serverIP: "123.123.123.123",
			wantPath: "/rdns?server_ip=123.123.123.123",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check path and query string together
				fullPath := r.URL.Path
				if r.URL.RawQuery != "" {
					fullPath += "?" + r.URL.RawQuery
				}

				if fullPath != tt.wantPath {
					t.Errorf("expected path '%s', got '%s'", tt.wantPath, fullPath)
				}

				if r.Method != "GET" {
					t.Errorf("expected GET request, got '%s'", r.Method)
				}

				response := []map[string]any{
					{
						"rdns": map[string]any{
							"ip":  "123.123.123.123",
							"ptr": "server1.example.com",
						},
					},
					{
						"rdns": map[string]any{
							"ip":  "124.124.124.124",
							"ptr": "server2.example.com",
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

			entries, err := client.RDNS.List(ctx, tt.serverIP)
			if err != nil {
				t.Fatalf("RDNS.List returned error: %v", err)
			}

			if len(entries) != 2 {
				t.Errorf("expected 2 entries, got %d", len(entries))
			}

			if entries[0].IP != "123.123.123.123" {
				t.Errorf("expected IP '123.123.123.123', got '%s'", entries[0].IP)
			}

			if entries[0].PTR != "server1.example.com" {
				t.Errorf("expected PTR 'server1.example.com', got '%s'", entries[0].PTR)
			}
		})
	}
}

func TestRDNSService_Get(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		wantPath string
		ptr      string
	}{
		{
			name:     "IPv4 address",
			ip:       "123.123.123.123",
			wantPath: "/rdns/123.123.123.123",
			ptr:      "server1.example.com",
		},
		{
			name:     "IPv6 address",
			ip:       "2001:db8::1",
			wantPath: "/rdns/2001:db8::1",
			ptr:      "server2.example.com",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Errorf("expected path '%s', got '%s'", tt.wantPath, r.URL.Path)
				}

				if r.Method != "GET" {
					t.Errorf("expected GET request, got '%s'", r.Method)
				}

				response := map[string]any{
					"rdns": map[string]any{
						"ip":  tt.ip,
						"ptr": tt.ptr,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			entry, err := client.RDNS.Get(ctx, tt.ip)
			if err != nil {
				t.Fatalf("RDNS.Get returned error: %v", err)
			}

			if entry.IP != tt.ip {
				t.Errorf("expected IP '%s', got '%s'", tt.ip, entry.IP)
			}

			if entry.PTR != tt.ptr {
				t.Errorf("expected PTR '%s', got '%s'", tt.ptr, entry.PTR)
			}
		})
	}
}

func TestRDNSService_Create(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		ptr  string
	}{
		{
			name: "create IPv4 PTR",
			ip:   "123.123.123.123",
			ptr:  "new-server.example.com",
		},
		{
			name: "create IPv6 PTR",
			ip:   "2001:db8::1",
			ptr:  "ipv6-server.example.com",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// URL paths are not escaped by httptest, so compare without encoding
				expectedPath := "/rdns/" + tt.ip
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("ptr") != tt.ptr {
					t.Errorf("expected ptr '%s', got '%s'", tt.ptr, r.FormValue("ptr"))
				}

				// Doc: "the status code 201 CREATED is returned" for PUT.
				w.WriteHeader(http.StatusCreated)
				response := map[string]any{
					"rdns": map[string]any{
						"ip":  tt.ip,
						"ptr": tt.ptr,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			entry, err := client.RDNS.Create(ctx, tt.ip, tt.ptr)
			if err != nil {
				t.Fatalf("RDNS.Create returned error: %v", err)
			}

			if entry.IP != tt.ip {
				t.Errorf("expected IP '%s', got '%s'", tt.ip, entry.IP)
			}

			if entry.PTR != tt.ptr {
				t.Errorf("expected PTR '%s', got '%s'", tt.ptr, entry.PTR)
			}
		})
	}
}

func TestRDNSService_Update(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		ptr  string
	}{
		{
			name: "update IPv4 PTR",
			ip:   "123.123.123.123",
			ptr:  "updated-server.example.com",
		},
		{
			name: "update IPv6 PTR",
			ip:   "2001:db8::1",
			ptr:  "updated-ipv6.example.com",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// URL paths are not escaped by httptest, so compare without encoding
				expectedPath := "/rdns/" + tt.ip
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("ptr") != tt.ptr {
					t.Errorf("expected ptr '%s', got '%s'", tt.ptr, r.FormValue("ptr"))
				}

				response := map[string]any{
					"rdns": map[string]any{
						"ip":  tt.ip,
						"ptr": tt.ptr,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			entry, err := client.RDNS.Update(ctx, tt.ip, tt.ptr)
			if err != nil {
				t.Fatalf("RDNS.Update returned error: %v", err)
			}

			if entry.IP != tt.ip {
				t.Errorf("expected IP '%s', got '%s'", tt.ip, entry.IP)
			}

			if entry.PTR != tt.ptr {
				t.Errorf("expected PTR '%s', got '%s'", tt.ptr, entry.PTR)
			}
		})
	}
}

func TestRDNSService_Delete(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{
			name: "delete IPv4 PTR",
			ip:   "123.123.123.123",
		},
		{
			name: "delete IPv6 PTR",
			ip:   "2001:db8::1",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// URL paths are not escaped by httptest, so compare without encoding
				expectedPath := "/rdns/" + tt.ip
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got '%s'", r.Method)
				}

				w.WriteHeader(http.StatusOK)
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			err := client.RDNS.Delete(ctx, tt.ip)
			if err != nil {
				t.Fatalf("RDNS.Delete returned error: %v", err)
			}
		})
	}
}

func TestRDNSService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			method:     "list",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.RDNS.List(ctx, "")
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.RDNS.Get(ctx, "123.123.123.123")
				return err
			},
		},
		{
			name:       "Create conflict",
			statusCode: http.StatusConflict,
			method:     "create",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.RDNS.Create(ctx, "123.123.123.123", "test.example.com")
				return err
			},
		},
		{
			name:       "Update unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "update",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.RDNS.Update(ctx, "123.123.123.123", "test.example.com")
				return err
			},
		},
		{
			name:       "Delete not found",
			statusCode: http.StatusNotFound,
			method:     "delete",
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.RDNS.Delete(ctx, "123.123.123.123")
			},
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

			err := tt.setupFunc(client, ctx)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

// TestRDNSService_List_Empty verifies an empty array response decodes to an empty slice, not an error.
func TestRDNSService_List_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rdns" {
			t.Errorf("expected path '/rdns', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.RDNS.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}
