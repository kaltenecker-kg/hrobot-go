package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

func TestBootService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321" {
			t.Errorf("expected path '/boot/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Verbatim from the doc's example response body for
		// "Boot configuration GET /boot/{server-number}". The doc's plain-text
		// dump renders the deprecated arch field's key as "@deprecated arch",
		// but that is the doc tool's deprecation annotation leaking into the
		// example text, not a real JSON key — the real API (and boot.go's
		// `json:"arch"` tag) uses the plain key "arch".
		response := map[string]any{
			"boot": map[string]any{
				"rescue": map[string]any{
					"server_ip":       "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   321,
					"os":              []string{"linux", "vkvm"},
					"arch":            []int{64, 32},
					"active":          false,
					"password":        nil,
					"authorized_key":  []map[string]any{},
					"host_key":        []map[string]any{},
				},
				"linux": map[string]any{
					"server_ip":       "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   321,
					"dist":            []string{"CentOS 5.5 minimal", "Debian 7.8 minimal"},
					"arch":            []int{64, 32},
					"lang":            []string{"en"},
					"active":          false,
					"password":        nil,
					"authorized_key":  []map[string]any{},
					"host_key":        []map[string]any{},
				},
				"vnc": map[string]any{
					"server_ip":       "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   321,
					"dist":            []string{"centOS-5.0", "Fedora-6", "openSUSE-10.2"},
					"arch":            []int{64, 32},
					"lang":            []string{"de_DE", "en_US"},
					"active":          false,
					"password":        nil,
				},
				"windows": map[string]any{
					"server_ip":       "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number":   321,
					"os": []string{
						"Windows Server 2022 Standard Edition",
						"Windows Server 2019 Standard Edition",
						"Windows Server 2016 Standard Edition",
					},
					"dist":     nil,
					"lang":     nil,
					"active":   false,
					"password": nil,
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

	if config.VNC == nil {
		t.Fatal("expected VNC config, got nil")
	}

	if config.VNC.Active {
		t.Error("expected VNC to be inactive")
	}

	if config.VNC.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.VNC.ServerNumber)
	}

	if config.Windows == nil {
		t.Fatal("expected Windows config, got nil")
	}

	if config.Windows.Active {
		t.Error("expected windows to be inactive")
	}

	if config.Windows.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.Windows.ServerNumber)
	}
}

// TestBootService_ActivateRescue is now wrapped with spectest.Handler: the
// spec/robot.yaml RescueSystemActivated.authorized_key/host_key schema
// previously declared arrays of bare fingerprint strings; it now models the
// doc's actual {"key": {name, fingerprint, type, size}} object shape (see
// spec/README.md, "boot tag fixes"), matching this fixture.
func TestBootService_ActivateRescue(t *testing.T) {
	tests := []struct {
		name         string
		opts         RescueActivateOpts
		fingerprints []string
	}{
		{
			name: "linux rescue with keys",
			opts: RescueActivateOpts{
				OS:             "linux",
				Arch:           64,
				AuthorizedKeys: []string{"15:28:b0:03:95:f0:77:b3:10:56:15:6b:77:22:a5:bb"},
			},
			fingerprints: []string{"15:28:b0:03:95:f0:77:b3:10:56:15:6b:77:22:a5:bb"},
		},
		{
			name: "linux rescue without keys",
			opts: RescueActivateOpts{
				OS:   "linux",
				Arch: 64,
			},
			fingerprints: []string{},
		},
		{
			name: "vkvm rescue",
			opts: RescueActivateOpts{
				OS:             "vkvm",
				Arch:           64,
				AuthorizedKeys: []string{"c1:e4:47:2d:f5:0a:1b:22:33:44:55:66:77:88:99:00"},
			},
			fingerprints: []string{"c1:e4:47:2d:f5:0a:1b:22:33:44:55:66:77:88:99:00"},
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/boot/321/rescue" {
					t.Errorf("expected path '/boot/321/rescue', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("os") != tt.opts.OS {
					t.Errorf("expected os '%s', got '%s'", tt.opts.OS, r.FormValue("os"))
				}

				if r.FormValue("arch") != "64" {
					t.Errorf("expected arch '64', got '%s'", r.FormValue("arch"))
				}

				// Check authorized key fingerprints
				formKeys := r.Form["authorized_key[]"]
				if len(formKeys) != len(tt.fingerprints) {
					t.Errorf("expected %d authorized keys, got %d", len(tt.fingerprints), len(formKeys))
				}

				authorizedKey := []map[string]any{}
				if len(tt.fingerprints) > 0 {
					authorizedKey = []map[string]any{
						{
							"key": map[string]any{
								"name":        "key1",
								"fingerprint": tt.fingerprints[0],
								"type":        "ED25519",
								"size":        256,
							},
						},
					}
				}

				password := "test-password-123"
				response := map[string]any{
					"rescue": map[string]any{
						"server_ip":       "123.123.123.123",
						"server_ipv6_net": "2a01:4f8:111:4221::",
						"server_number":   321,
						"active":          true,
						"os":              tt.opts.OS,
						"arch":            tt.opts.Arch,
						"authorized_key":  authorizedKey,
						"host_key": []map[string]any{
							{
								"key": map[string]any{
									"fingerprint": "c1:e4:47:2d:f5:0a:1b:22:33:44:55:66:77:88:99:00",
									"type":        "DSA",
									"size":        1024,
								},
							},
						},
						"password": password,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			rescue, err := client.Boot.ActivateRescue(ctx, ServerID(321), tt.opts)
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

			if len(tt.fingerprints) > 0 {
				if len(rescue.AuthorizedKeys) != 1 {
					t.Fatalf("expected 1 authorized key, got %d", len(rescue.AuthorizedKeys))
				}
				if rescue.AuthorizedKeys[0].Key.Fingerprint != tt.fingerprints[0] {
					t.Errorf("expected fingerprint '%s', got '%s'", tt.fingerprints[0], rescue.AuthorizedKeys[0].Key.Fingerprint)
				}
			}

			if len(rescue.HostKeys) != 1 {
				t.Fatalf("expected 1 host key, got %d", len(rescue.HostKeys))
			}
			if rescue.HostKeys[0].Key.Fingerprint != "c1:e4:47:2d:f5:0a:1b:22:33:44:55:66:77:88:99:00" {
				t.Errorf("expected host key fingerprint 'c1:e4:...', got '%s'", rescue.HostKeys[0].Key.Fingerprint)
			}
		})
	}
}

// TestBootService_ActivateRescue_OmitsOptionalFields asserts that arch and
// keyboard are omitted from the POST body entirely when left at their zero
// value, per the doc's Input table for POST /boot/{server-number}/rescue
// (both are optional).
func TestBootService_ActivateRescue_OmitsOptionalFields(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if _, present := r.Form["arch"]; present {
			t.Errorf("expected arch to be omitted, got %q", r.FormValue("arch"))
		}
		if _, present := r.Form["keyboard"]; present {
			t.Errorf("expected keyboard to be omitted, got %q", r.FormValue("keyboard"))
		}

		password := "test-password-123"
		response := map[string]any{
			"rescue": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"active":          true,
				"os":              "linux",
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
				"password":        password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.Boot.ActivateRescue(ctx, ServerID(321), RescueActivateOpts{OS: "linux"}); err != nil {
		t.Fatalf("Boot.ActivateRescue returned error: %v", err)
	}
}

// TestBootService_ActivateRescue_SendsOptionalFields asserts that arch and
// keyboard are sent on the POST body when explicitly set.
func TestBootService_ActivateRescue_SendsOptionalFields(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if r.FormValue("arch") != "64" {
			t.Errorf("expected arch '64', got '%s'", r.FormValue("arch"))
		}
		if r.FormValue("keyboard") != "de" {
			t.Errorf("expected keyboard 'de', got '%s'", r.FormValue("keyboard"))
		}

		password := "test-password-123"
		response := map[string]any{
			"rescue": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"active":          true,
				"os":              "linux",
				"arch":            64,
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
				"password":        password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	opts := RescueActivateOpts{OS: "linux", Arch: 64, Keyboard: "de"}
	if _, err := client.Boot.ActivateRescue(ctx, ServerID(321), opts); err != nil {
		t.Fatalf("Boot.ActivateRescue returned error: %v", err)
	}
}

// TestBootService_ActivateRescue_RequiresOS asserts that a missing OS is
// rejected locally without making a request, per the doc's Input table for
// POST /boot/{server-number}/rescue (os is the only required field).
func TestBootService_ActivateRescue_RequiresOS(t *testing.T) {
	client := NewClient("test-user", "test-pass", WithBaseURL("http://127.0.0.1:0"))
	ctx := context.Background()

	_, err := client.Boot.ActivateRescue(ctx, ServerID(321), RescueActivateOpts{})
	if err == nil {
		t.Fatal("expected error for missing os, got nil")
	}
}

func TestBootService_DeactivateRescue(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/rescue" {
			t.Errorf("expected path '/boot/321/rescue', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Doc-verbatim example response body for
		// DELETE /boot/{server-number}/rescue.
		response := map[string]any{
			"rescue": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"os":              []string{"linux", "vkvm"},
				"arch":            []int{64, 32},
				"active":          false,
				"password":        nil,
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Boot.DeactivateRescue(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Boot.DeactivateRescue returned error: %v", err)
	}
}

func TestBootService_GetLastRescue(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/rescue/last" {
			t.Errorf("expected path '/boot/321/rescue/last', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		password := "previous-password-456"
		response := map[string]any{
			"rescue": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"active":          false,
				"os":              "linux",
				"arch":            64,
				"authorized_key": []map[string]any{
					{
						"key": map[string]any{
							"name":        "key1",
							"fingerprint": "15:28:b0:03:95:f0:77:b3:10:56:15:6b:77:22:a5:bb",
							"type":        "ED25519",
							"size":        256,
						},
					},
				},
				"host_key": []map[string]any{},
				"password": password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
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

	if len(rescue.AuthorizedKeys) != 1 {
		t.Fatalf("expected 1 authorized key, got %d", len(rescue.AuthorizedKeys))
	}
	if rescue.AuthorizedKeys[0].Key.Fingerprint != "15:28:b0:03:95:f0:77:b3:10:56:15:6b:77:22:a5:bb" {
		t.Errorf("expected fingerprint '15:28:b0:03:95:f0:77:b3:10:56:15:6b:77:22:a5:bb', got '%s'", rescue.AuthorizedKeys[0].Key.Fingerprint)
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

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

				// authorized_key[] must not be present since no keys were passed.
				if _, exists := r.Form["authorized_key[]"]; exists {
					t.Error("expected no authorized_key[] form values")
				}

				// Verbatim (dist/arch/lang substituted per test case) from the
				// doc's example response body for
				// "POST /boot/{server-number}/linux".
				password := "jEt0dtUvomlyOwRr"
				response := map[string]any{
					"linux": map[string]any{
						"server_ip":       "123.123.123.123",
						"server_ipv6_net": "2a01:4f8:111:4221::",
						"server_number":   321,
						"dist":            tt.dist,
						"arch":            tt.arch,
						"lang":            tt.lang,
						"active":          true,
						"password":        password,
						"authorized_key":  []map[string]any{},
						"host_key":        []map[string]any{},
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			linux, err := client.Boot.ActivateLinux(ctx, ServerID(321), LinuxActivateOpts{
				Dist: tt.dist,
				Arch: tt.arch,
				Lang: tt.lang,
			})
			if err != nil {
				t.Fatalf("Boot.ActivateLinux returned error: %v", err)
			}

			if !linux.Active {
				t.Error("expected linux to be active")
			}
			if linux.ServerNumber != 321 {
				t.Errorf("expected server number 321, got %d", linux.ServerNumber)
			}
			if linux.Password == nil || *linux.Password != "jEt0dtUvomlyOwRr" {
				t.Errorf("expected password 'jEt0dtUvomlyOwRr', got %v", linux.Password)
			}
			if got := linux.ActiveDist(); got != tt.dist {
				t.Errorf("expected active dist '%s', got '%s'", tt.dist, got)
			}
			if got := linux.ActiveLang(); got != tt.lang {
				t.Errorf("expected active lang '%s', got '%s'", tt.lang, got)
			}
		})
	}
}

func TestBootService_DeactivateLinux(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/linux" {
			t.Errorf("expected path '/boot/321/linux', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Doc-verbatim example response body for
		// DELETE /boot/{server-number}/linux.
		response := map[string]any{
			"linux": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            []string{"CentOS 5.5 minimal", "Debian 7.8 minimal"},
				"arch":            []int{64, 32},
				"lang":            []string{"en"},
				"active":          false,
				"password":        nil,
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
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
				_, err := c.Boot.ActivateRescue(ctx, ServerID(321), RescueActivateOpts{OS: "linux", Arch: 64})
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

func TestBootService_ActivateVNC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		// Verbatim (dist/lang substituted) from the doc's example response
		// body for "POST /boot/{server-number}/vnc".
		response := map[string]any{
			"vnc": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            "Debian 12",
				"arch":            64,
				"lang":            "en",
				"active":          true,
				"password":        password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vnc, err := client.Boot.ActivateVNC(ctx, ServerID(321), VNCActivateOpts{Dist: "Debian 12", Arch: 64, Lang: "en"})
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
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/vnc" {
			t.Errorf("expected path '/boot/321/vnc', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Doc-verbatim example response body for
		// DELETE /boot/{server-number}/vnc.
		response := map[string]any{
			"vnc": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            []string{"centOS-5.0", "Fedora-6", "openSUSE-10.2"},
				"arch":            []int{64, 32},
				"lang":            []string{"de_DE", "en_US"},
				"active":          false,
				"password":        nil,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Boot.DeactivateVNC(ctx, ServerID(321)); err != nil {
		t.Fatalf("Boot.DeactivateVNC returned error: %v", err)
	}
}

func TestBootService_GetLastLinux(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/linux/last" {
			t.Errorf("expected path '/boot/321/linux/last', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		password := "last-linux-pw"
		// Verbatim from the doc's example response body for
		// "GET /boot/{server-number}/linux/last".
		response := map[string]any{
			"linux": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            "Ubuntu 22.04",
				"arch":            64,
				"lang":            "en",
				"active":          false,
				"password":        password,
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
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
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/windows" {
			t.Errorf("expected path '/boot/321/windows', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Verbatim from the doc's example response body for
		// "GET /boot/{server-number}/windows".
		response := map[string]any{
			"windows": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            []string{"standard"},
				"os": []string{
					"Windows Server 2022 Standard Edition",
					"Windows Server 2019 Standard Edition",
					"Windows Server 2016 Standard Edition",
				},
				"lang":     []string{"en", "de"},
				"active":   false,
				"password": nil,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
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
	if got := windows.AvailableLangs(); !equalStringSlice(got, []string{"en", "de"}) {
		t.Errorf("expected available langs ['en', 'de'], got %v", got)
	}
	if got := windows.AvailableOS(); len(got) != 3 || got[0] != "Windows Server 2022 Standard Edition" {
		t.Errorf("expected 3 available OS options starting with 'Windows Server 2022 Standard Edition', got %v", got)
	}
}

func TestBootService_ActivateWindows(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if r.FormValue("os") != "Windows Server 2019 Standard Edition" {
			t.Errorf("expected os 'Windows Server 2019 Standard Edition', got '%s'", r.FormValue("os"))
		}

		password := "jEt0dtUvomlyOwRr"
		// Verbatim from the doc's example response body for
		// "POST /boot/{server-number}/windows".
		response := map[string]any{
			"windows": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"dist":            "standard",
				"os":              "Windows Server 2019 Standard Edition",
				"lang":            "en",
				"active":          true,
				"password":        password,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	windows, err := client.Boot.ActivateWindows(ctx, ServerID(321), "en", "Windows Server 2019 Standard Edition")
	if err != nil {
		t.Fatalf("Boot.ActivateWindows returned error: %v", err)
	}

	if !windows.Active {
		t.Error("expected windows to be active")
	}
	if windows.Password == nil || *windows.Password != "jEt0dtUvomlyOwRr" {
		t.Errorf("expected password 'jEt0dtUvomlyOwRr', got %v", windows.Password)
	}
	if got := windows.ActiveOS(); got != "Windows Server 2019 Standard Edition" {
		t.Errorf("expected active OS 'Windows Server 2019 Standard Edition', got '%s'", got)
	}
	if got := windows.ActiveLang(); got != "en" {
		t.Errorf("expected active lang 'en', got '%s'", got)
	}
}

// TestBootService_ActivateWindows_RequiresLangAndOS asserts that missing lang
// or os is rejected locally without making a request, per the doc's Input
// table for POST /boot/{server-number}/windows (both are required).
func TestBootService_ActivateWindows_RequiresLangAndOS(t *testing.T) {
	// The server fails the test if reached, proving the rejection happens
	// locally rather than the request erroring for some other reason.
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("ActivateWindows must not perform an HTTP call for missing input; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.Boot.ActivateWindows(ctx, ServerID(321), "", "Windows Server 2019 Standard Edition"); err == nil {
		t.Fatal("expected error for missing lang, got nil")
	}
	if _, err := client.Boot.ActivateWindows(ctx, ServerID(321), "en", ""); err == nil {
		t.Fatal("expected error for missing os, got nil")
	}
}

func TestBootService_DeactivateWindows(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boot/321/windows" {
			t.Errorf("expected path '/boot/321/windows', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Doc-verbatim example response body for
		// DELETE /boot/{server-number}/windows.
		response := map[string]any{
			"windows": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"os": []string{
					"Windows Server 2022 Standard Edition",
					"Windows Server 2019 Standard Edition",
					"Windows Server 2016 Standard Edition",
				},
				"dist":     []string{"standard"},
				"lang":     []string{"en", "de"},
				"active":   false,
				"password": nil,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Boot.DeactivateWindows(ctx, ServerID(321)); err != nil {
		t.Fatalf("Boot.DeactivateWindows returned error: %v", err)
	}
}

func TestBootService_ActivateRescue_EmptyKeys(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		response := map[string]any{
			"rescue": map[string]any{
				"server_ip":       "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number":   321,
				"active":          true,
				"os":              "linux",
				"arch":            64,
				"authorized_key":  []map[string]any{},
				"host_key":        []map[string]any{},
				"password":        password,
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	rescue, err := client.Boot.ActivateRescue(ctx, ServerID(321), RescueActivateOpts{OS: "linux", Arch: 64})
	if err != nil {
		t.Fatalf("Boot.ActivateRescue returned error: %v", err)
	}

	if !rescue.Active {
		t.Error("expected rescue to be active")
	}
}

func TestRescueConfig_Accessors(t *testing.T) {
	t.Run("inactive returns options", func(t *testing.T) {
		body := []byte(`{"active":false,"os":["linux","vkvm"],"arch":[64,32]}`)
		var c RescueConfig
		if err := json.Unmarshal(body, &c); err != nil {
			t.Fatal(err)
		}
		if got := c.ActiveOS(); got != "" {
			t.Errorf("ActiveOS = %q, want \"\"", got)
		}
		if got := c.AvailableOS(); !equalStringSlice(got, []string{"linux", "vkvm"}) {
			t.Errorf("AvailableOS = %v", got)
		}
		if got := c.ActiveArch(); got != 0 {
			t.Errorf("ActiveArch = %d, want 0", got)
		}
		if got := c.AvailableArchs(); !equalIntSlice(got, []int{64, 32}) {
			t.Errorf("AvailableArchs = %v", got)
		}
	})

	t.Run("active returns scalar", func(t *testing.T) {
		body := []byte(`{"active":true,"os":"linux","arch":64}`)
		var c RescueConfig
		if err := json.Unmarshal(body, &c); err != nil {
			t.Fatal(err)
		}
		if got := c.ActiveOS(); got != "linux" {
			t.Errorf("ActiveOS = %q, want \"linux\"", got)
		}
		if got := c.AvailableOS(); got != nil {
			t.Errorf("AvailableOS = %v, want nil", got)
		}
		if got := c.ActiveArch(); got != 64 {
			t.Errorf("ActiveArch = %d, want 64", got)
		}
		if got := c.AvailableArchs(); got != nil {
			t.Errorf("AvailableArchs = %v, want nil", got)
		}
	})
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
