package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

func TestIPService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip" {
			t.Errorf("expected path '/ip', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		body := `[
			{
				"ip": {
					"ip": "123.123.123.123",
					"server_ip": "123.123.123.123",
					"server_number": 321,
					"locked": false,
					"separate_mac": null,
					"traffic_warnings": false,
					"traffic_hourly": 50,
					"traffic_daily": 50,
					"traffic_monthly": 8
				}
			},
			{
				"ip": {
					"ip": "124.124.124.124",
					"server_ip": "123.123.123.123",
					"server_number": 321,
					"locked": false,
					"separate_mac": null,
					"traffic_warnings": false,
					"traffic_hourly": 200,
					"traffic_daily": 2000,
					"traffic_monthly": 20
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

	ips, err := client.IP.List(ctx)
	if err != nil {
		t.Fatalf("IP.List returned error: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}

	if ips[0].IP.String() != "123.123.123.123" {
		t.Errorf("expected ip '123.123.123.123', got '%s'", ips[0].IP.String())
	}

	if ips[0].ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", ips[0].ServerNumber)
	}

	if ips[0].TrafficWarnings {
		t.Error("expected traffic warnings to be disabled")
	}

	if ips[0].TrafficHourly != 50 || ips[0].TrafficDaily != 50 || ips[0].TrafficMonthly != 8 {
		t.Errorf("unexpected traffic limits for ips[0]: %+v", ips[0])
	}

	if ips[1].IP.String() != "124.124.124.124" {
		t.Errorf("expected ip '124.124.124.124', got '%s'", ips[1].IP.String())
	}

	if ips[1].TrafficWarnings {
		t.Error("expected traffic warnings to be disabled")
	}

	if ips[1].TrafficHourly != 200 || ips[1].TrafficDaily != 2000 || ips[1].TrafficMonthly != 20 {
		t.Errorf("unexpected traffic limits for ips[1]: %+v", ips[1])
	}
}

func TestIPService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123" {
			t.Errorf("expected path '/ip/123.123.123.123', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		body := `{
			"ip": {
				"ip": "123.123.123.123",
				"gateway": "123.123.123.97",
				"mask": 27,
				"broadcast": "123.123.123.127",
				"server_ip": "123.123.123.123",
				"server_number": 321,
				"locked": false,
				"separate_mac": null,
				"traffic_warnings": false,
				"traffic_hourly": 50,
				"traffic_daily": 50,
				"traffic_monthly": 8
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ip := net.ParseIP("123.123.123.123")
	ipAddr, err := client.IP.Get(ctx, ip)
	if err != nil {
		t.Fatalf("IP.Get returned error: %v", err)
	}

	if ipAddr.IP.String() != "123.123.123.123" {
		t.Errorf("expected ip '123.123.123.123', got '%s'", ipAddr.IP.String())
	}

	if ipAddr.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", ipAddr.ServerNumber)
	}

	if ipAddr.SeparateMac != "" {
		t.Errorf("expected separate_mac to be empty, got '%s'", ipAddr.SeparateMac)
	}

	if ipAddr.TrafficWarnings {
		t.Error("expected traffic warnings to be disabled")
	}

	if ipAddr.TrafficHourly != 50 || ipAddr.TrafficDaily != 50 || ipAddr.TrafficMonthly != 8 {
		t.Errorf("unexpected traffic limits: %+v", ipAddr)
	}
}

func TestIPService_SetTrafficWarnings(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enable traffic warnings",
			enabled: true,
		},
		{
			name:    "disable traffic warnings",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := loadSpec(t)
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ip/123.123.123.123" {
					t.Errorf("expected path '/ip/123.123.123.123', got '%s'", r.URL.Path)
				}

				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				expectedValue := "false"
				if tt.enabled {
					expectedValue = "true"
				}

				if r.FormValue("traffic_warnings") != expectedValue {
					t.Errorf("expected traffic_warnings '%s', got '%s'", expectedValue, r.FormValue("traffic_warnings"))
				}

				// Doc-verbatim example response body for
				// POST /ip/{ip} (traffic_warnings substituted per test case).
				response := map[string]any{
					"ip": map[string]any{
						"ip":               "123.123.123.123",
						"gateway":          "123.123.123.97",
						"mask":             27,
						"broadcast":        "123.123.123.127",
						"server_ip":        "123.123.123.123",
						"server_number":    321,
						"locked":           false,
						"separate_mac":     nil,
						"traffic_warnings": tt.enabled,
						"traffic_hourly":   50,
						"traffic_daily":    50,
						"traffic_monthly":  8,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			ip := net.ParseIP("123.123.123.123")
			ipAddr, err := client.IP.SetTrafficWarnings(ctx, ip, tt.enabled)
			if err != nil {
				t.Fatalf("IP.SetTrafficWarnings returned error: %v", err)
			}
			if ipAddr.TrafficWarnings != tt.enabled {
				t.Errorf("expected traffic_warnings %v, got %v", tt.enabled, ipAddr.TrafficWarnings)
			}
			if ipAddr.Gateway.String() != "123.123.123.97" {
				t.Errorf("expected gateway '123.123.123.97', got '%s'", ipAddr.Gateway.String())
			}
		})
	}
}

func TestIPService_NilIPGuard(t *testing.T) {
	client := NewClient("test-user", "test-pass")
	ctx := context.Background()

	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "Get with nil IP",
			testFunc: func() error {
				_, err := client.IP.Get(ctx, nil)
				return err
			},
		},
		{
			name: "SetTrafficWarnings with nil IP",
			testFunc: func() error {
				_, err := client.IP.SetTrafficWarnings(ctx, nil, true)
				return err
			},
		},
		{
			name: "WithdrawIPCancellation with nil IP",
			testFunc: func() error {
				return client.IP.WithdrawIPCancellation(ctx, nil)
			},
		},
		{
			name: "GetMAC with nil IP",
			testFunc: func() error {
				_, err := client.IP.GetMAC(ctx, nil)
				return err
			},
		},
		{
			name: "SetMAC with nil IP",
			testFunc: func() error {
				_, err := client.IP.SetMAC(ctx, nil)
				return err
			},
		},
		{
			name: "DeleteMAC with nil IP",
			testFunc: func() error {
				return client.IP.DeleteMAC(ctx, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if err == nil {
				t.Errorf("expected error for nil IP, got nil")
			}
			var e *Error
			if !errors.As(err, &e) || e.Kind != ErrKindParse {
				t.Errorf("expected parse error, got %T: %v", err, err)
			}
		})
	}
}

func TestIPService_CancelIP_DisallowedByPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("CancelIP must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))

	err := client.IP.CancelIP(context.Background(), net.ParseIP("123.123.123.123"), "2024-12-31")
	if !IsPolicyError(err) {
		t.Fatalf("expected policy error, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 451 {
		t.Fatalf("expected status 451, got %v", err)
	}
}

func TestIPService_WithdrawIPCancellation(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/cancellation" {
			t.Errorf("expected path '/ip/123.123.123.123/cancellation', got '%s'", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Doc-verbatim example response body for
		// DELETE /ip/{ip}/cancellation.
		response := map[string]any{
			"cancellation": map[string]any{
				"ip":                         "123.123.123.123",
				"server_number":              321,
				"earliest_cancellation_date": "2022-02-11",
				"cancelled":                  false,
				"cancellation_date":          nil,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ip := net.ParseIP("123.123.123.123")
	err := client.IP.WithdrawIPCancellation(ctx, ip)
	if err != nil {
		t.Fatalf("IP.WithdrawIPCancellation returned error: %v", err)
	}
}

func TestIPService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.IP.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				ip := net.ParseIP("123.123.123.123")
				_, err := c.IP.Get(ctx, ip)
				return err
			},
		},
		{
			name:       "SetTrafficWarnings unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				ip := net.ParseIP("123.123.123.123")
				_, err := c.IP.SetTrafficWarnings(ctx, ip, true)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"status":  tt.statusCode,
						"code":    "ERROR",
						"message": "test error",
					},
				})
			}))
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

func TestIPService_GetMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/mac" {
			t.Errorf("expected path '/ip/123.123.123.123/mac', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mac": map[string]any{
				"ip":  "123.123.123.123",
				"mac": "00:21:85:62:3e:9c",
			},
		})
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	mac, err := client.IP.GetMAC(context.Background(), net.ParseIP("123.123.123.123"))
	if err != nil {
		t.Fatalf("IP.GetMAC returned error: %v", err)
	}
	if mac.MAC != "00:21:85:62:3e:9c" {
		t.Errorf("expected mac '00:21:85:62:3e:9c', got '%s'", mac.MAC)
	}
}

func TestIPService_SetMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/mac" {
			t.Errorf("expected path '/ip/123.123.123.123/mac', got '%s'", r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mac": map[string]any{
				"ip":  "123.123.123.123",
				"mac": "00:21:85:62:3e:9c",
			},
		})
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	mac, err := client.IP.SetMAC(context.Background(), net.ParseIP("123.123.123.123"))
	if err != nil {
		t.Fatalf("IP.SetMAC returned error: %v", err)
	}
	if mac.MAC == "" {
		t.Error("expected mac to be returned")
	}
}

func TestIPService_DeleteMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/mac" {
			t.Errorf("expected path '/ip/123.123.123.123/mac', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		body := `{
			"mac": {
				"ip": "123.123.123.123",
				"mac": null
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if err := client.IP.DeleteMAC(context.Background(), net.ParseIP("123.123.123.123")); err != nil {
		t.Fatalf("IP.DeleteMAC returned error: %v", err)
	}
}
