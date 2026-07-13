package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/v2/internal/spectest"
)

func TestFailoverService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/failover" {
			t.Errorf("expected path '/failover', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		body := `[
			{
				"failover": {
					"ip": "123.123.123.123",
					"netmask": "255.255.255.255",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "78.46.1.93"
				}
			},
			{
				"failover": {
					"ip": "2a01:4f8:fff1::",
					"netmask": "ffff:ffff:ffff:ffff::",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "2a01:4f8:d0a:2003::"
				}
			}
		]`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	failovers, err := client.Failover.List(ctx)
	if err != nil {
		t.Fatalf("Failover.List returned error: %v", err)
	}

	if len(failovers) != 2 {
		t.Errorf("expected 2 failovers, got %d", len(failovers))
	}

	if failovers[0].IP != "123.123.123.123" {
		t.Errorf("expected IP '123.123.123.123', got '%s'", failovers[0].IP)
	}

	if failovers[0].ServerIPv6Net != "2a01:4f8:d0a:2003::" {
		t.Errorf("expected server_ipv6_net '2a01:4f8:d0a:2003::', got '%s'", failovers[0].ServerIPv6Net)
	}

	if failovers[0].ActiveServerIP == nil {
		t.Error("expected active_server_ip to be set")
	} else if *failovers[0].ActiveServerIP != "78.46.1.93" {
		t.Errorf("expected active_server_ip '78.46.1.93', got '%s'", *failovers[0].ActiveServerIP)
	}

	if failovers[1].IP != "2a01:4f8:fff1::" {
		t.Errorf("expected IP '2a01:4f8:fff1::', got '%s'", failovers[1].IP)
	}

	if failovers[1].ActiveServerIP == nil {
		t.Error("expected active_server_ip to be set")
	} else if *failovers[1].ActiveServerIP != "2a01:4f8:d0a:2003::" {
		t.Errorf("expected active_server_ip '2a01:4f8:d0a:2003::', got '%s'", *failovers[1].ActiveServerIP)
	}
}

func TestFailoverService_Get(t *testing.T) {
	tests := []struct {
		name             string
		ip               string
		wantPath         string
		body             string
		wantNetmask      string
		wantActiveServer string
	}{
		{
			name:             "IPv4 failover",
			ip:               "123.123.123.123",
			wantPath:         "/failover/123.123.123.123",
			wantNetmask:      "255.255.255.255",
			wantActiveServer: "78.46.1.93",
			body: `{
				"failover": {
					"ip": "123.123.123.123",
					"netmask": "255.255.255.255",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "78.46.1.93"
				}
			}`,
		},
		{
			name:             "IPv6 failover",
			ip:               "2a01:4f8:fff1::",
			wantPath:         "/failover/2a01:4f8:fff1::",
			wantNetmask:      "ffff:ffff:ffff:ffff::",
			wantActiveServer: "2a01:4f8:d0a:2003::",
			body: `{
				"failover": {
					"ip": "2a01:4f8:fff1::",
					"netmask": "ffff:ffff:ffff:ffff::",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "2a01:4f8:d0a:2003::"
				}
			}`,
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

				if _, err := w.Write([]byte(tt.body)); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			failover, err := client.Failover.Get(ctx, tt.ip)
			if err != nil {
				t.Fatalf("Failover.Get returned error: %v", err)
			}

			if failover.IP != tt.ip {
				t.Errorf("expected IP '%s', got '%s'", tt.ip, failover.IP)
			}

			if failover.Netmask != tt.wantNetmask {
				t.Errorf("expected netmask '%s', got '%s'", tt.wantNetmask, failover.Netmask)
			}

			if failover.ActiveServerIP == nil {
				t.Error("expected active_server_ip to be set")
			} else if *failover.ActiveServerIP != tt.wantActiveServer {
				t.Errorf("expected active_server_ip '%s', got '%s'", tt.wantActiveServer, *failover.ActiveServerIP)
			}
		})
	}
}

func TestFailoverService_Update(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		activeServerIP string
		body           string
	}{
		{
			name:           "IPv4 route switch",
			ip:             "123.123.123.123",
			activeServerIP: "124.124.124.124",
			body: `{
				"failover": {
					"ip": "123.123.123.123",
					"netmask": "255.255.255.255",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "124.124.124.124"
				}
			}`,
		},
		{
			name:           "IPv6 route switch",
			ip:             "2a01:4f8:fff1::",
			activeServerIP: "2a01:4f8:0:5176::",
			body: `{
				"failover": {
					"ip": "2a01:4f8:fff1::",
					"netmask": "ffff:ffff:ffff:ffff::",
					"server_ip": "78.46.1.93",
					"server_ipv6_net": "2a01:4f8:d0a:2003::",
					"server_number": 321,
					"active_server_ip": "2a01:4f8:0:5176::"
				}
			}`,
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/failover/" + tt.ip
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("active_server_ip") != tt.activeServerIP {
					t.Errorf("expected active_server_ip '%s', got '%s'", tt.activeServerIP, r.FormValue("active_server_ip"))
				}

				if _, err := w.Write([]byte(tt.body)); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			failover, err := client.Failover.Update(ctx, tt.ip, tt.activeServerIP)
			if err != nil {
				t.Fatalf("Failover.Update returned error: %v", err)
			}

			if failover.ActiveServerIP == nil {
				t.Error("expected active_server_ip to be set")
			} else if *failover.ActiveServerIP != tt.activeServerIP {
				t.Errorf("expected active_server_ip '%s', got '%s'", tt.activeServerIP, *failover.ActiveServerIP)
			}
		})
	}
}

func TestFailoverService_Delete(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/failover/123.123.123.123" {
			t.Errorf("expected path '/failover/123.123.123.123', got '%s'", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		body := `{
			"failover": {
				"ip": "123.123.123.123",
				"netmask": "255.255.255.255",
				"server_ip": "78.46.1.93",
				"server_ipv6_net": "2a01:4f8:d0a:2003::",
				"server_number": 321,
				"active_server_ip": null
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Failover.Delete(ctx, "123.123.123.123")
	if err != nil {
		t.Fatalf("Failover.Delete returned error: %v", err)
	}
}

func TestFailoverService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.Get(ctx, "123.123.123.100")
				return err
			},
		},
		{
			name:       "Update unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.Update(ctx, "123.123.123.100", "123.123.123.123")
				return err
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.Failover.Delete(ctx, "123.123.123.100")
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

// TestFailoverService_List_Empty verifies an empty array response decodes to an empty slice, not an error.
func TestFailoverService_List_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/failover" {
			t.Errorf("expected path '/failover', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.Failover.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}
