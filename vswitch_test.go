package hrobot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

func TestVSwitchService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch" {
			t.Errorf("expected path '/vswitch', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Doc example (GET /vswitch): top-level array of unwrapped vSwitch
		// objects, no envelope.
		response := []map[string]any{
			{
				"id":        1234,
				"name":      "vswitch 1234",
				"vlan":      4000,
				"cancelled": false,
			},
			{
				"id":        4321,
				"name":      "vswitch test",
				"vlan":      4001,
				"cancelled": false,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vswitches, err := client.VSwitch.List(ctx)
	if err != nil {
		t.Fatalf("VSwitch.List returned error: %v", err)
	}

	if len(vswitches) != 2 {
		t.Errorf("expected 2 vswitches, got %d", len(vswitches))
	}

	if vswitches[0].ID != 1234 {
		t.Errorf("expected ID 1234, got %d", vswitches[0].ID)
	}

	if vswitches[0].Name != "vswitch 1234" {
		t.Errorf("expected name 'vswitch 1234', got '%s'", vswitches[0].Name)
	}

	if vswitches[0].VLAN != 4000 {
		t.Errorf("expected VLAN 4000, got %d", vswitches[0].VLAN)
	}
}

func TestVSwitchService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/4321" {
			t.Errorf("expected path '/vswitch/4321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Doc example (GET /vswitch/{vswitch-id}): the vSwitch object is
		// returned unwrapped at the top level (no "vswitch" envelope key).
		response := map[string]any{
			"id":        4321,
			"name":      "my vSwitch",
			"vlan":      4000,
			"cancelled": false,
			"server": []map[string]any{
				{
					"server_ip":       "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   321,
					"status":          "ready",
				},
				{
					"server_ip":       "123.123.123.124",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   421,
					"status":          "ready",
				},
			},
			"subnet": []map[string]any{
				{
					"ip":      "213.239.252.48",
					"mask":    29,
					"gateway": "213.239.252.49",
				},
			},
			"cloud_network": []map[string]any{
				{
					"id":      123,
					"ip":      "10.0.2.0",
					"mask":    24,
					"gateway": "10.0.2.1",
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

	vswitch, err := client.VSwitch.Get(ctx, 4321)
	if err != nil {
		t.Fatalf("VSwitch.Get returned error: %v", err)
	}

	if vswitch.ID != 4321 {
		t.Errorf("expected ID 4321, got %d", vswitch.ID)
	}

	if vswitch.Name != "my vSwitch" {
		t.Errorf("expected name 'my vSwitch', got '%s'", vswitch.Name)
	}

	if len(vswitch.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(vswitch.Servers))
	}

	if len(vswitch.Servers) > 0 && vswitch.Servers[0].ServerIPv6Net != "2a01:4f8:111:4221::" {
		t.Errorf("expected server_ipv6_net '2a01:4f8:111:4221::', got '%s'", vswitch.Servers[0].ServerIPv6Net)
	}

	if len(vswitch.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(vswitch.Subnets))
	}

	if len(vswitch.CloudNetwork) != 1 {
		t.Errorf("expected 1 cloud network, got %d", len(vswitch.CloudNetwork))
	}
}

func TestVSwitchService_Create(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch" {
			t.Errorf("expected path '/vswitch', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "my vSwitch" {
			t.Errorf("expected name 'my vSwitch', got '%s'", r.FormValue("name"))
		}

		if r.FormValue("vlan") != "4000" {
			t.Errorf("expected vlan '4000', got '%s'", r.FormValue("vlan"))
		}

		// Doc example (POST /vswitch): the created vSwitch object is
		// returned unwrapped at the top level (no "vswitch" envelope key).
		response := map[string]any{
			"id":            4321,
			"name":          "my vSwitch",
			"vlan":          4000,
			"cancelled":     false,
			"server":        []map[string]any{},
			"subnet":        []map[string]any{},
			"cloud_network": []map[string]any{},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vswitch, err := client.VSwitch.Create(ctx, "my vSwitch", 4000)
	if err != nil {
		t.Fatalf("VSwitch.Create returned error: %v", err)
	}

	if vswitch.Name != "my vSwitch" {
		t.Errorf("expected name 'my vSwitch', got '%s'", vswitch.Name)
	}

	if vswitch.VLAN != 4000 {
		t.Errorf("expected VLAN 4000, got %d", vswitch.VLAN)
	}
}

func TestVSwitchService_Update(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/4321" {
			t.Errorf("expected path '/vswitch/4321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "my new name" {
			t.Errorf("expected name 'my new name', got '%s'", r.FormValue("name"))
		}

		if r.FormValue("vlan") != "4001" {
			t.Errorf("expected vlan '4001', got '%s'", r.FormValue("vlan"))
		}

		// Doc: "Output: No output".
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.Update(ctx, 4321, "my new name", 4001)
	if err != nil {
		t.Fatalf("VSwitch.Update returned error: %v", err)
	}
}

func TestVSwitchService_Delete(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/4321" {
			t.Errorf("expected path '/vswitch/4321', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// DeleteWithBody sends form data in body with DELETE method
		// Read body to parse form data
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		// Parse the form-encoded body manually
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("failed to parse query: %v", err)
		}

		cancDate := values.Get("cancellation_date")
		if cancDate != "2018-06-30" {
			t.Errorf("expected cancellation_date '2018-06-30', got '%s' (body: %s)", cancDate, string(body))
		}

		// Doc: "Output: No output".
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.Delete(ctx, 4321, "2018-06-30")
	if err != nil {
		t.Fatalf("VSwitch.Delete returned error: %v", err)
	}
}

func TestVSwitchService_AddServers(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/4321/server" {
			t.Errorf("expected path '/vswitch/4321/server', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		servers := r.Form["server[]"]
		if len(servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(servers))
		}
		if len(servers) == 2 && (servers[0] != "123.123.123.123" || servers[1] != "123.123.123.124") {
			t.Errorf("expected servers ['123.123.123.123', '123.123.123.124'], got %v", servers)
		}

		// Doc: "Output: No output".
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.AddServers(ctx, 4321, []string{"123.123.123.123", "123.123.123.124"})
	if err != nil {
		t.Fatalf("VSwitch.AddServers returned error: %v", err)
	}
}

func TestVSwitchService_RemoveServers(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/4321/server" {
			t.Errorf("expected path '/vswitch/4321/server', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		// Doc example: server[]=123.123.123.123&server[]=123.123.123.124
		if string(body) != "server[]=123.123.123.123&server[]=123.123.123.124" {
			t.Errorf("expected body 'server[]=123.123.123.123&server[]=123.123.123.124', got '%s'", string(body))
		}

		// Doc: "Output: No output".
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.RemoveServers(ctx, 4321, []string{"123.123.123.123", "123.123.123.124"})
	if err != nil {
		t.Fatalf("VSwitch.RemoveServers returned error: %v", err)
	}
}

func TestVSwitchService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.Get(ctx, 12345)
				return err
			},
		},
		{
			name:       "Create unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.Create(ctx, "test", 4000)
				return err
			},
		},
		{
			name:       "Update error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.VSwitch.Update(ctx, 12345, "test", 4000)
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.VSwitch.Delete(ctx, 12345, "2024-12-31")
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

func TestVSwitchService_IntegerConversion(t *testing.T) {
	// Test that integer parameters are properly converted to strings for form encoding
	tests := []struct {
		name string
		id   int
		vlan int
	}{
		{
			name: "small numbers",
			id:   123,
			vlan: 4000,
		},
		{
			// 4091 is the maximum valid VLAN ID for a vSwitch.
			name: "large numbers",
			id:   999999,
			vlan: 4091,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := loadSpec(t)
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && r.URL.Path == "/vswitch" {
					if err := r.ParseForm(); err != nil {
						t.Fatalf("failed to parse form: %v", err)
					}

					vlanStr := r.FormValue("vlan")
					if vlanStr != strconv.Itoa(tt.vlan) {
						t.Errorf("expected vlan '%d', got '%s'", tt.vlan, vlanStr)
					}

					// Doc example (POST /vswitch): unwrapped top-level object.
					response := map[string]any{
						"id":            tt.id,
						"name":          "test",
						"vlan":          tt.vlan,
						"cancelled":     false,
						"server":        []map[string]any{},
						"subnet":        []map[string]any{},
						"cloud_network": []map[string]any{},
					}
					_ = json.NewEncoder(w).Encode(response)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			vswitch, err := client.VSwitch.Create(ctx, "test", tt.vlan)
			if err != nil {
				t.Fatalf("VSwitch.Create returned error: %v", err)
			}

			if vswitch.VLAN != tt.vlan {
				t.Errorf("expected VLAN %d, got %d", tt.vlan, vswitch.VLAN)
			}
		})
	}
}

func TestVSwitchService_WaitForVSwitchReady(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "all servers ready",
			status:      "ready",
			shouldError: false,
		},
		{
			name:          "server failed",
			status:        "failed",
			shouldError:   true,
			errorContains: "is in status failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := loadSpec(t)
			callCount := 0
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/vswitch/4321" && r.Method == "GET" {
					callCount++
					// Doc example (GET /vswitch/{vswitch-id}): unwrapped
					// top-level object.
					response := map[string]any{
						"id":        4321,
						"name":      "test-vswitch",
						"vlan":      4000,
						"cancelled": false,
						"server": []map[string]any{
							{
								"server_ip":       "123.123.123.123",
								"server_ipv6_net": "2a01:4f8:111:4221::",
								"server_number":   321,
								"status":          tt.status,
							},
						},
						"subnet":        []map[string]any{},
						"cloud_network": []map[string]any{},
					}
					_ = json.NewEncoder(w).Encode(response)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			err := client.VSwitch.WaitForVSwitchReady(ctx, 4321)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errorContains != "" && err != nil {
					if !contains(err.Error(), tt.errorContains) {
						t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
					}
				}
				// Verify that we fail fast (should only call Get once for failed status)
				if tt.status == "failed" && callCount != 1 {
					t.Errorf("expected 1 Get call for failed status, got %d", callCount)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
