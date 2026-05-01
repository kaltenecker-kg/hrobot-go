package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBootService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321" {
			t.Errorf("expected path '/boot/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"boot": map[string]interface{}{
				"rescue": map[string]interface{}{
					"server_ip":      "123.123.123.123",
					"server_number":  321,
					"active":         false,
					"os":             []string{"linux", "linuxold", "vkvm"},
					"arch":           []int{64},
					"authorized_key": []string{},
					"host_key":       []string{},
				},
				"linux": map[string]interface{}{
					"server_ip":      "123.123.123.123",
					"server_number":  321,
					"dist":           []string{"Ubuntu 22.04", "Debian 12"},
					"arch":           []int{64},
					"lang":           []string{"en"},
					"active":         false,
					"authorized_key": []string{},
					"host_key":       []string{},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	config, err := client.Boot.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.Get returned error: %v", err)
	}

	if config.Rescue == nil {
		t.Fatal("expected Rescue config, got nil")
	}

	if config.Rescue.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.Rescue.ServerNumber)
	}

	if config.Rescue.Active {
		t.Error("expected rescue to be inactive")
	}

	if config.Linux == nil {
		t.Fatal("expected Linux config, got nil")
	}

	if config.Linux.Active {
		t.Error("expected linux to be inactive")
	}
}

func TestBootService_ActivateRescue(t *testing.T) {
	tests := []struct {
		name           string
		os             string
		arch           int
		authorizedKeys []string
	}{
		{
			name:           "linux rescue with keys",
			os:             "linux",
			arch:           64,
			authorizedKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EA..."},
		},
		{
			name:           "linux rescue without keys",
			os:             "linux",
			arch:           64,
			authorizedKeys: []string{},
		},
		{
			name:           "vkvm rescue",
			os:             "vkvm",
			arch:           64,
			authorizedKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/boot/321/rescue" {
					t.Errorf("expected path '/boot/321/rescue', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("os") != tt.os {
					t.Errorf("expected os '%s', got '%s'", tt.os, r.FormValue("os"))
				}

				if r.FormValue("arch") != "64" {
					t.Errorf("expected arch '64', got '%s'", r.FormValue("arch"))
				}

				// Check authorized keys
				formKeys := r.Form["authorized_key[]"]
				if len(formKeys) != len(tt.authorizedKeys) {
					t.Errorf("expected %d authorized keys, got %d", len(tt.authorizedKeys), len(formKeys))
				}

				password := "test-password-123"
				response := map[string]interface{}{
					"rescue": map[string]interface{}{
						"server_ip":      "123.123.123.123",
						"server_number":  321,
						"active":         true,
						"os":             tt.os,
						"arch":           tt.arch,
						"authorized_key": tt.authorizedKeys,
						"host_key":       []string{"AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."},
						"password":       password,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			rescue, err := client.Boot.ActivateRescue(ctx, ServerID(321), tt.os, tt.arch, tt.authorizedKeys)
			if err != nil {
				t.Fatalf("Boot.ActivateRescue returned error: %v", err)
			}

			if !rescue.Active {
				t.Error("expected rescue to be active")
			}

			if rescue.ServerNumber != 321 {
				t.Errorf("expected server number 321, got %d", rescue.ServerNumber)
			}

			if rescue.Password == nil {
				t.Error("expected password to be set")
			}
		})
	}
}

func TestBootService_DeactivateRescue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/rescue" {
			t.Errorf("expected path '/boot/321/rescue', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Boot.DeactivateRescue(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.DeactivateRescue returned error: %v", err)
	}
}

func TestBootService_GetLastRescue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/rescue/last" {
			t.Errorf("expected path '/boot/321/rescue/last', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		password := "previous-password-456"
		response := map[string]interface{}{
			"rescue": map[string]interface{}{
				"server_ip":      "123.123.123.123",
				"server_number":  321,
				"active":         false,
				"os":             "linux",
				"arch":           64,
				"authorized_key": []string{"ssh-rsa AAAAB3NzaC1yc2EA..."},
				"host_key":       []string{},
				"password":       password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	rescue, err := client.Boot.GetLastRescue(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.GetLastRescue returned error: %v", err)
	}

	if rescue.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", rescue.ServerNumber)
	}

	if rescue.Password == nil {
		t.Error("expected password to be set")
	} else if *rescue.Password != "previous-password-456" {
		t.Errorf("expected password 'previous-password-456', got '%s'", *rescue.Password)
	}
}

func TestBootService_ActivateLinux(t *testing.T) {
	tests := []struct {
		name string
		dist string
		arch int
		lang string
	}{
		{
			name: "Ubuntu 22.04",
			dist: "Ubuntu 22.04",
			arch: 64,
			lang: "en",
		},
		{
			name: "Debian 12",
			dist: "Debian 12",
			arch: 64,
			lang: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/boot/321/linux" {
					t.Errorf("expected path '/boot/321/linux', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("dist") != tt.dist {
					t.Errorf("expected dist '%s', got '%s'", tt.dist, r.FormValue("dist"))
				}

				if r.FormValue("arch") != "64" {
					t.Errorf("expected arch '64', got '%s'", r.FormValue("arch"))
				}

				if r.FormValue("lang") != tt.lang {
					t.Errorf("expected lang '%s', got '%s'", tt.lang, r.FormValue("lang"))
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			_, err := client.Boot.ActivateLinux(ctx, ServerID(321), tt.dist, tt.arch, tt.lang, []string{})
			if err != nil {
				t.Fatalf("Boot.ActivateLinux returned error: %v", err)
			}
		})
	}
}

func TestBootService_DeactivateLinux(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/linux" {
			t.Errorf("expected path '/boot/321/linux', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Boot.DeactivateLinux(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.DeactivateLinux returned error: %v", err)
	}
}

func TestBootService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Boot.Get(ctx, ServerID(321))
				return err
			},
		},
		{
			name:       "ActivateRescue unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "activaterescue",
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Boot.ActivateRescue(ctx, ServerID(321), "linux", 64, []string{})
				return err
			},
		},
		{
			name:       "DeactivateRescue error",
			statusCode: http.StatusInternalServerError,
			method:     "deactivaterescue",
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.Boot.DeactivateRescue(ctx, ServerID(321))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
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

func TestBootService_ActivateVNC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/vnc" {
			t.Errorf("expected path '/boot/321/vnc', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("dist") != "Debian 12" {
			t.Errorf("expected dist 'Debian 12', got '%s'", r.FormValue("dist"))
		}
		if r.FormValue("arch") != "64" {
			t.Errorf("expected arch '64', got '%s'", r.FormValue("arch"))
		}
		if r.FormValue("lang") != "en" {
			t.Errorf("expected lang 'en', got '%s'", r.FormValue("lang"))
		}

		password := "vnc-password"
		// "vnc" is not a registered response wrapper key, so the fixture
		// is returned unwrapped and decoded directly into VNCConfig.
		response := map[string]interface{}{
			"server_ip":     "123.123.123.123",
			"server_number": 321,
			"active":        true,
			"dist":          "Debian 12",
			"arch":          64,
			"lang":          "en",
			"password":      password,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vnc, err := client.Boot.ActivateVNC(ctx, ServerID(321), "Debian 12", 64, "en")
	if err != nil {
		t.Fatalf("Boot.ActivateVNC returned error: %v", err)
	}

	if !vnc.Active {
		t.Error("expected VNC to be active")
	}
	if vnc.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", vnc.ServerNumber)
	}
	if vnc.Password == nil || *vnc.Password != "vnc-password" {
		t.Errorf("expected password 'vnc-password', got %v", vnc.Password)
	}
}

func TestBootService_DeactivateVNC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/vnc" {
			t.Errorf("expected path '/boot/321/vnc', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Boot.DeactivateVNC(ctx, ServerID(321)); err != nil {
		t.Fatalf("Boot.DeactivateVNC returned error: %v", err)
	}
}

func TestBootService_GetLastLinux(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/linux/last" {
			t.Errorf("expected path '/boot/321/linux/last', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		password := "last-linux-pw"
		// "linux" is not a registered response wrapper key.
		response := map[string]interface{}{
			"server_ip":     "123.123.123.123",
			"server_number": 321,
			"dist":          "Ubuntu 22.04",
			"arch":          64,
			"lang":          "en",
			"active":        false,
			"password":      password,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	linux, err := client.Boot.GetLastLinux(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.GetLastLinux returned error: %v", err)
	}

	if linux.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", linux.ServerNumber)
	}
	if linux.Password == nil || *linux.Password != "last-linux-pw" {
		t.Errorf("expected password 'last-linux-pw', got %v", linux.Password)
	}
}

func TestBootService_GetWindows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/windows" {
			t.Errorf("expected path '/boot/321/windows', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// "windows" is not a registered response wrapper key.
		response := map[string]interface{}{
			"server_ip":     "123.123.123.123",
			"server_number": 321,
			"active":        false,
			"lang":          []string{"en", "de"},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	windows, err := client.Boot.GetWindows(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.GetWindows returned error: %v", err)
	}

	if windows.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", windows.ServerNumber)
	}
	if windows.Active {
		t.Error("expected windows to be inactive")
	}
}

func TestBootService_ActivateWindows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/windows" {
			t.Errorf("expected path '/boot/321/windows', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("lang") != "en" {
			t.Errorf("expected lang 'en', got '%s'", r.FormValue("lang"))
		}

		password := "windows-pw"
		// "windows" is not a registered response wrapper key.
		response := map[string]interface{}{
			"server_ip":     "123.123.123.123",
			"server_number": 321,
			"active":        true,
			"lang":          "en",
			"password":      password,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	windows, err := client.Boot.ActivateWindows(ctx, ServerID(321), "en")
	if err != nil {
		t.Fatalf("Boot.ActivateWindows returned error: %v", err)
	}

	if !windows.Active {
		t.Error("expected windows to be active")
	}
	if windows.Password == nil || *windows.Password != "windows-pw" {
		t.Errorf("expected password 'windows-pw', got %v", windows.Password)
	}
}

func TestBootService_DeactivateWindows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/windows" {
			t.Errorf("expected path '/boot/321/windows', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Boot.DeactivateWindows(ctx, ServerID(321)); err != nil {
		t.Fatalf("Boot.DeactivateWindows returned error: %v", err)
	}
}

func TestBootService_ActivateRescue_EmptyKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Verify that authorized_key[] is not present when empty
		if _, exists := r.Form["authorized_key[]"]; exists {
			formKeys := r.Form["authorized_key[]"]
			if len(formKeys) > 0 && strings.TrimSpace(formKeys[0]) != "" {
				t.Error("expected no authorized keys or empty values")
			}
		}

		password := "test-password"
		response := map[string]interface{}{
			"rescue": map[string]interface{}{
				"server_ip":      "123.123.123.123",
				"server_number":  321,
				"active":         true,
				"os":             "linux",
				"arch":           64,
				"authorized_key": []string{},
				"host_key":       []string{},
				"password":       password,
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	rescue, err := client.Boot.ActivateRescue(ctx, ServerID(321), "linux", 64, []string{})
	if err != nil {
		t.Fatalf("Boot.ActivateRescue returned error: %v", err)
	}

	if !rescue.Active {
		t.Error("expected rescue to be active")
	}
}
