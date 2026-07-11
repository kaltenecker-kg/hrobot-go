package hrobot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFirewallService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"name":       "allow ssh",
							"ip_version": "ipv4",
							"action":     "accept",
							"protocol":   "tcp",
							"dst_port":   "22",
						},
					},
					"output": []map[string]any{},
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

	config, err := client.Firewall.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Get returned error: %v", err)
	}

	if config.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.ServerNumber)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	if !config.WhitelistHOS {
		t.Error("expected whitelist_hos to be true")
	}

	if len(config.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(config.Rules.Input))
	}

	if config.Rules.Input[0].Name != "allow ssh" {
		t.Errorf("expected rule name 'allow ssh', got '%s'", config.Rules.Input[0].Name)
	}
}

// firewallDocRules returns two doc-shaped firewall input rules used to
// verify that Activate/Disable/Update re-post the full existing ruleset.
func firewallDocRules() []map[string]any {
	return []map[string]any{
		{
			"name":       "allow ssh",
			"ip_version": "ipv4",
			"action":     "accept",
			"protocol":   "tcp",
			"dst_port":   "22",
		},
		{
			"name":       "allow http",
			"ip_version": "ipv4",
			"action":     "accept",
			"protocol":   "tcp",
			"dst_port":   "80",
		},
	}
}

// assertInputRuleForm asserts that the posted form contains the
// rules[input][idx][*] keys/values matching the given doc-shaped rule.
func assertInputRuleForm(t *testing.T, r *http.Request, idx int, rule map[string]any) {
	t.Helper()
	for key, value := range rule {
		formKey := fmt.Sprintf("rules[input][%d][%s]", idx, key)
		got := r.FormValue(formKey)
		want := fmt.Sprintf("%v", value)
		if got != want {
			t.Errorf("expected form key %q to be %q, got %q", formKey, want, got)
		}
	}
}

func TestFirewallService_Activate(t *testing.T) {
	getCalled := false
	postCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}

		switch r.Method {
		case http.MethodGet:
			getCalled = true
			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "disabled",
					"whitelist_hos": true,
					"filter_ipv6":   false,
					"port":          "main",
					"rules": map[string]any{
						"input":  firewallDocRules(),
						"output": []map[string]any{},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case http.MethodPost:
			postCalled = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse form: %v", err)
			}

			if r.FormValue("status") != "active" {
				t.Errorf("expected status 'active', got '%s'", r.FormValue("status"))
			}
			if r.FormValue("whitelist_hos") != "true" {
				t.Errorf("expected whitelist_hos 'true', got '%s'", r.FormValue("whitelist_hos"))
			}

			docRules := firewallDocRules()
			for i, rule := range docRules {
				assertInputRuleForm(t, r, i, rule)
			}

			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": true,
					"port":          "main",
					"rules": map[string]any{
						"input":  firewallDocRules(),
						"output": []map[string]any{},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		default:
			t.Errorf("expected GET or POST request, got '%s'", r.Method)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	config, err := client.Firewall.Activate(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Activate returned error: %v", err)
	}

	if !getCalled {
		t.Error("expected Activate to call GET to fetch the current configuration")
	}
	if !postCalled {
		t.Error("expected Activate to call POST to apply the updated configuration")
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	if len(config.Rules.Input) != 2 {
		t.Errorf("expected 2 input rules, got %d", len(config.Rules.Input))
	}
}

func TestFirewallService_Disable(t *testing.T) {
	getCalled := false
	postCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}

		switch r.Method {
		case http.MethodGet:
			getCalled = true
			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": true,
					"filter_ipv6":   false,
					"port":          "main",
					"rules": map[string]any{
						"input":  firewallDocRules(),
						"output": []map[string]any{},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case http.MethodPost:
			postCalled = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse form: %v", err)
			}

			if r.FormValue("status") != "disabled" {
				t.Errorf("expected status 'disabled', got '%s'", r.FormValue("status"))
			}
			if r.FormValue("whitelist_hos") != "true" {
				t.Errorf("expected whitelist_hos 'true', got '%s'", r.FormValue("whitelist_hos"))
			}

			docRules := firewallDocRules()
			for i, rule := range docRules {
				assertInputRuleForm(t, r, i, rule)
			}

			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "disabled",
					"whitelist_hos": true,
					"port":          "main",
					"rules": map[string]any{
						"input":  firewallDocRules(),
						"output": []map[string]any{},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		default:
			t.Errorf("expected GET or POST request, got '%s'", r.Method)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	config, err := client.Firewall.Disable(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Disable returned error: %v", err)
	}

	if !getCalled {
		t.Error("expected Disable to call GET to fetch the current configuration")
	}
	if !postCalled {
		t.Error("expected Disable to call POST to apply the updated configuration")
	}

	if config.Status != FirewallStatusDisabled {
		t.Errorf("expected status 'disabled', got '%s'", config.Status)
	}

	if len(config.Rules.Input) != 2 {
		t.Errorf("expected 2 input rules, got %d", len(config.Rules.Input))
	}
}

func TestFirewallService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Firewall.Delete(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Delete returned error: %v", err)
	}
}

func TestFirewallService_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("status") != "active" {
			t.Errorf("expected status 'active', got '%s'", r.FormValue("status"))
		}
		if r.FormValue("whitelist_hos") != "true" {
			t.Errorf("expected whitelist_hos 'true', got '%s'", r.FormValue("whitelist_hos"))
		}

		wantRule := map[string]any{
			"name":       "allow http",
			"ip_version": "ipv4",
			"action":     "accept",
			"protocol":   "tcp",
			"dst_port":   "80",
		}
		assertInputRuleForm(t, r, 0, wantRule)

		// Confirm the literal-bracket keys are present in the parsed form
		// (not percent-encoded, as the Robot API requires).
		for _, key := range []string{
			"rules[input][0][name]",
			"rules[input][0][ip_version]",
			"rules[input][0][action]",
			"rules[input][0][protocol]",
			"rules[input][0][dst_port]",
		} {
			if _, ok := r.Form[key]; !ok {
				t.Errorf("expected literal-bracket form key %q to be present", key)
			}
		}

		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"name":       "allow http",
							"ip_version": "ipv4",
							"action":     "accept",
							"protocol":   "tcp",
							"dst_port":   "80",
						},
					},
					"output": []map[string]any{},
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

	status := FirewallStatusActive
	whitelist := true
	updateConfig := UpdateConfig{
		Status:       &status,
		WhitelistHOS: &whitelist,
		Rules: FirewallRules{
			Input: []FirewallRule{
				{
					Name:      "allow http",
					IPVersion: IPv4,
					Action:    ActionAccept,
					Protocol:  ProtocolTCP,
					DestPort:  "80",
				},
			},
			Output: []FirewallRule{},
		},
	}

	config, err := client.Firewall.Update(ctx, ServerID(321), updateConfig)
	if err != nil {
		t.Fatalf("Firewall.Update returned error: %v", err)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	if len(config.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(config.Rules.Input))
	}
}

func TestFirewallService_WaitForFirewallReady(t *testing.T) {
	tests := []struct {
		name       string
		responses  []FirewallStatus // Sequence of statuses to return
		wantError  bool
		numRetries int // Expected number of retries
	}{
		{
			name:       "already ready",
			responses:  []FirewallStatus{FirewallStatusActive},
			wantError:  false,
			numRetries: 1,
		},
		{
			name:       "becomes ready after one retry",
			responses:  []FirewallStatus{"in process", FirewallStatusActive},
			wantError:  false,
			numRetries: 2,
		},
		{
			name:       "becomes ready after multiple retries",
			responses:  []FirewallStatus{"in process", "in process", "in process", FirewallStatusActive},
			wantError:  false,
			numRetries: 4,
		},
		{
			name:       "disabled is also ready",
			responses:  []FirewallStatus{"in process", FirewallStatusDisabled},
			wantError:  false,
			numRetries: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/firewall/321" {
					t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
				}
				if r.Method != "GET" {
					t.Errorf("expected GET request, got '%s'", r.Method)
				}

				// Return different status based on call count
				status := FirewallStatusActive
				if callCount < len(tt.responses) {
					status = tt.responses[callCount]
				}
				callCount++

				response := map[string]any{
					"firewall": map[string]any{
						"server_ip":     "123.123.123.123",
						"server_number": 321,
						"status":        status,
						"whitelist_hos": true,
						"port":          "main",
						"rules": map[string]any{
							"input":  []map[string]any{},
							"output": []map[string]any{},
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

			err := client.Firewall.WaitForFirewallReady(ctx, ServerID(321))
			if (err != nil) != tt.wantError {
				t.Errorf("WaitForFirewallReady() error = %v, wantError %v", err, tt.wantError)
			}

			if callCount != tt.numRetries {
				t.Errorf("expected %d retries, got %d", tt.numRetries, callCount)
			}
		})
	}
}

func TestFirewallService_WaitForFirewallReady_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Always return "in process" status
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "in process",
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 1) // Very short timeout
	defer cancel()

	err := client.Firewall.WaitForFirewallReady(ctx, ServerID(321))
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestFirewallService_ListTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template" {
			t.Errorf("expected path '/firewall/template', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]any{
			{
				"id":            1,
				"name":          "default",
				"filter_ipv6":   false,
				"whitelist_hos": true,
				"is_default":    true,
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
				},
			},
			{
				"id":            2,
				"name":          "strict",
				"filter_ipv6":   true,
				"whitelist_hos": false,
				"is_default":    false,
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
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

	templates, err := client.Firewall.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("Firewall.ListTemplates returned error: %v", err)
	}

	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	if templates[0].Name != "default" {
		t.Errorf("expected name 'default', got '%s'", templates[0].Name)
	}
	if !templates[0].IsDefault {
		t.Error("expected first template to be default")
	}
}

func TestFirewallService_GetTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template/1" {
			t.Errorf("expected path '/firewall/template/1', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            1,
				"name":          "default",
				"filter_ipv6":   false,
				"whitelist_hos": true,
				"is_default":    true,
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
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

	tmpl, err := client.Firewall.GetTemplate(ctx, "1")
	if err != nil {
		t.Fatalf("Firewall.GetTemplate returned error: %v", err)
	}

	if tmpl.ID != 1 {
		t.Errorf("expected id 1, got %d", tmpl.ID)
	}
	if tmpl.Name != "default" {
		t.Errorf("expected name 'default', got '%s'", tmpl.Name)
	}
}

func TestFirewallService_CreateTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template" {
			t.Errorf("expected path '/firewall/template', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "my-template" {
			t.Errorf("expected name 'my-template', got '%s'", r.FormValue("name"))
		}
		if r.FormValue("filter_ipv6") != "true" {
			t.Errorf("expected filter_ipv6 'true', got '%s'", r.FormValue("filter_ipv6"))
		}
		if r.FormValue("whitelist_hos") != "true" {
			t.Errorf("expected whitelist_hos 'true', got '%s'", r.FormValue("whitelist_hos"))
		}
		if r.FormValue("is_default") != "false" {
			t.Errorf("expected is_default 'false', got '%s'", r.FormValue("is_default"))
		}

		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            7,
				"name":          "my-template",
				"filter_ipv6":   true,
				"whitelist_hos": true,
				"is_default":    false,
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
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

	tmpl, err := client.Firewall.CreateTemplate(ctx, TemplateConfig{
		Name:         "my-template",
		FilterIPv6:   true,
		WhitelistHOS: true,
		IsDefault:    false,
		Rules: FirewallRules{
			Input:  []FirewallRule{},
			Output: []FirewallRule{},
		},
	})
	if err != nil {
		t.Fatalf("Firewall.CreateTemplate returned error: %v", err)
	}

	if tmpl.ID != 7 {
		t.Errorf("expected id 7, got %d", tmpl.ID)
	}
	if tmpl.Name != "my-template" {
		t.Errorf("expected name 'my-template', got '%s'", tmpl.Name)
	}
}

func TestFirewallService_UpdateTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template/7" {
			t.Errorf("expected path '/firewall/template/7', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "renamed" {
			t.Errorf("expected name 'renamed', got '%s'", r.FormValue("name"))
		}
		if r.FormValue("filter_ipv6") != "false" {
			t.Errorf("expected filter_ipv6 'false', got '%s'", r.FormValue("filter_ipv6"))
		}
		if r.FormValue("whitelist_hos") != "false" {
			t.Errorf("expected whitelist_hos 'false', got '%s'", r.FormValue("whitelist_hos"))
		}
		if r.FormValue("is_default") != "true" {
			t.Errorf("expected is_default 'true', got '%s'", r.FormValue("is_default"))
		}

		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            7,
				"name":          "renamed",
				"filter_ipv6":   false,
				"whitelist_hos": false,
				"is_default":    true,
				"rules": map[string]any{
					"input":  []map[string]any{},
					"output": []map[string]any{},
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

	tmpl, err := client.Firewall.UpdateTemplate(ctx, "7", TemplateConfig{
		Name:      "renamed",
		IsDefault: true,
		Rules: FirewallRules{
			Input:  []FirewallRule{},
			Output: []FirewallRule{},
		},
	})
	if err != nil {
		t.Fatalf("Firewall.UpdateTemplate returned error: %v", err)
	}

	if tmpl.Name != "renamed" {
		t.Errorf("expected name 'renamed', got '%s'", tmpl.Name)
	}
	if !tmpl.IsDefault {
		t.Error("expected template to be default")
	}
}

func TestFirewallService_DeleteTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template/7" {
			t.Errorf("expected path '/firewall/template/7', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Firewall.DeleteTemplate(ctx, "7"); err != nil {
		t.Fatalf("Firewall.DeleteTemplate returned error: %v", err)
	}
}

func TestFirewallService_ApplyTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("template_id") != "7" {
			t.Errorf("expected template_id '7', got '%s'", r.FormValue("template_id"))
		}

		response := map[string]any{
			"server_ip":     "123.123.123.123",
			"server_number": 321,
			"status":        "active",
			"whitelist_hos": true,
			"port":          "main",
			"rules": map[string]any{
				"input":  []map[string]any{},
				"output": []map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	config, err := client.Firewall.ApplyTemplate(ctx, ServerID(321), "7")
	if err != nil {
		t.Fatalf("Firewall.ApplyTemplate returned error: %v", err)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}
	if config.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.ServerNumber)
	}
}

func TestFirewallService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Firewall.Get(ctx, ServerID(321))
				return err
			},
		},
		{
			name:       "Activate unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Firewall.Activate(ctx, ServerID(321))
				return err
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.Firewall.Delete(ctx, ServerID(321))
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
