package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/v2/internal/spectest"
)

func TestFirewallService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Fixture matches the doc's GET /firewall/{server-id} example
		// response verbatim, including explicit nulls for unset rule fields.
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"filter_ipv6":   false,
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"ip_version": "ipv4",
							"name":       "rule 1",
							"dst_ip":     nil,
							"src_ip":     "1.1.1.1",
							"dst_port":   "80",
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
					},
					"output": []map[string]any{
						{
							"ip_version": nil,
							"name":       "Allow all",
							"dst_ip":     nil,
							"src_ip":     nil,
							"dst_port":   nil,
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
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

	if config.Rules.Input[0].Name != "rule 1" {
		t.Errorf("expected rule name 'rule 1', got '%s'", config.Rules.Input[0].Name)
	}

	if config.Rules.Input[0].SourceIP != "1.1.1.1" {
		t.Errorf("expected rule src_ip '1.1.1.1', got '%s'", config.Rules.Input[0].SourceIP)
	}

	if len(config.Rules.Output) != 1 {
		t.Errorf("expected 1 output rule, got %d", len(config.Rules.Output))
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

	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if r.FormValue("filter_ipv6") != "false" {
				t.Errorf("expected filter_ipv6 'false', got '%s'", r.FormValue("filter_ipv6"))
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
		default:
			t.Errorf("expected GET or POST request, got '%s'", r.Method)
		}
	})))
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

	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if r.FormValue("filter_ipv6") != "false" {
				t.Errorf("expected filter_ipv6 'false', got '%s'", r.FormValue("filter_ipv6"))
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
		default:
			t.Errorf("expected GET or POST request, got '%s'", r.Method)
		}
	})))
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
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// Fixture matches the doc's DELETE /firewall/{server-id} example
		// response verbatim: status flips to "in process" and rules is an
		// empty object (not {"input":[],"output":[]}).
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "in process",
				"filter_ipv6":   false,
				"whitelist_hos": true,
				"port":          "main",
				"rules":         map[string]any{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Firewall.Delete(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Delete returned error: %v", err)
	}
}

func TestFirewallService_Update(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"filter_ipv6":   false,
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
	})))
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

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						"filter_ipv6":   false,
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
			})))
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
	// Not wrapped with spectest.Handler: the context timeout is 1ns, so the
	// request may be cancelled mid-flight; wrapping would add flakiness
	// without exercising anything spec-fidelity related.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Always return "in process" status
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "in process",
				"filter_ipv6":   false,
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
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template" {
			t.Errorf("expected path '/firewall/template', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Fixture matches the doc's GET /firewall/template example response
		// verbatim: an array of {"firewall_template": {...}} wrappers. The
		// doc's list example omits "rules" (only the detailed GET/POST
		// responses include it), so it is abridged here too.
		response := []map[string]any{
			{
				"firewall_template": map[string]any{
					"id":            1,
					"name":          "My template",
					"filter_ipv6":   false,
					"whitelist_hos": true,
					"is_default":    true,
				},
			},
			{
				"firewall_template": map[string]any{
					"id":            2,
					"name":          "My second template",
					"filter_ipv6":   false,
					"whitelist_hos": true,
					"is_default":    false,
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

	templates, err := client.Firewall.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("Firewall.ListTemplates returned error: %v", err)
	}

	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	if templates[0].Name != "My template" {
		t.Errorf("expected name 'My template', got '%s'", templates[0].Name)
	}
	if !templates[0].IsDefault {
		t.Error("expected first template to be default")
	}
	if templates[1].Name != "My second template" {
		t.Errorf("expected name 'My second template', got '%s'", templates[1].Name)
	}
	if templates[1].IsDefault {
		t.Error("expected second template not to be default")
	}
}

func TestFirewallService_GetTemplate(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template/123" {
			t.Errorf("expected path '/firewall/template/123', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		// Fixture matches the doc's GET /firewall/template/{template-id}
		// example response verbatim.
		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            123,
				"filter_ipv6":   false,
				"whitelist_hos": true,
				"is_default":    false,
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"ip_version": "ipv4",
							"name":       "rule 1",
							"dst_ip":     nil,
							"src_ip":     "1.1.1.1",
							"dst_port":   "80",
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
						{
							"ip_version": "ipv4",
							"name":       "Allow MySQL",
							"dst_ip":     nil,
							"src_ip":     nil,
							"dst_port":   "3306",
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
					},
					"output": []map[string]any{
						{
							"ip_version": nil,
							"name":       "Allow all",
							"dst_ip":     nil,
							"src_ip":     nil,
							"dst_port":   nil,
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
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

	tmpl, err := client.Firewall.GetTemplate(ctx, "123")
	if err != nil {
		t.Fatalf("Firewall.GetTemplate returned error: %v", err)
	}

	if tmpl.ID != 123 {
		t.Errorf("expected id 123, got %d", tmpl.ID)
	}
	if len(tmpl.Rules.Input) != 2 {
		t.Errorf("expected 2 input rules, got %d", len(tmpl.Rules.Input))
	}
	if tmpl.Rules.Input[0].SourceIP != "1.1.1.1" {
		t.Errorf("expected rule src_ip '1.1.1.1', got '%s'", tmpl.Rules.Input[0].SourceIP)
	}
	if len(tmpl.Rules.Output) != 1 {
		t.Errorf("expected 1 output rule, got %d", len(tmpl.Rules.Output))
	}
}

func TestFirewallService_CreateTemplate(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		wantRule := map[string]any{
			"name":       "rule 1",
			"ip_version": "ipv4",
			"action":     "accept",
			"src_ip":     "1.1.1.1",
			"dst_port":   "80",
		}
		assertInputRuleForm(t, r, 0, wantRule)

		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            7,
				"name":          "my-template",
				"filter_ipv6":   true,
				"whitelist_hos": true,
				"is_default":    false,
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"ip_version": "ipv4",
							"name":       "rule 1",
							"dst_ip":     nil,
							"src_ip":     "1.1.1.1",
							"dst_port":   "80",
							"src_port":   nil,
							"protocol":   nil,
							"tcp_flags":  nil,
							"action":     "accept",
						},
					},
					"output": []map[string]any{},
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

	tmpl, err := client.Firewall.CreateTemplate(ctx, TemplateConfig{
		Name:         "my-template",
		FilterIPv6:   true,
		WhitelistHOS: true,
		IsDefault:    false,
		Rules: FirewallRules{
			Input: []FirewallRule{
				{
					Name:      "rule 1",
					IPVersion: IPv4,
					Action:    ActionAccept,
					SourceIP:  "1.1.1.1",
					DestPort:  "80",
				},
			},
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
	if len(tmpl.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(tmpl.Rules.Input))
	}
}

func TestFirewallService_UpdateTemplate(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		wantRule := map[string]any{
			"name":       "Allow HTTPS",
			"ip_version": "ipv4",
			"action":     "accept",
			"protocol":   "tcp",
			"dst_port":   "443",
		}
		assertInputRuleForm(t, r, 0, wantRule)

		response := map[string]any{
			"firewall_template": map[string]any{
				"id":            7,
				"name":          "renamed",
				"filter_ipv6":   false,
				"whitelist_hos": false,
				"is_default":    true,
				"rules": map[string]any{
					"input": []map[string]any{
						{
							"ip_version": "ipv4",
							"name":       "Allow HTTPS",
							"dst_ip":     nil,
							"src_ip":     nil,
							"dst_port":   "443",
							"src_port":   nil,
							"protocol":   "tcp",
							"tcp_flags":  nil,
							"action":     "accept",
						},
					},
					"output": []map[string]any{},
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

	tmpl, err := client.Firewall.UpdateTemplate(ctx, "7", TemplateConfig{
		Name:      "renamed",
		IsDefault: true,
		Rules: FirewallRules{
			Input: []FirewallRule{
				{
					Name:      "Allow HTTPS",
					IPVersion: IPv4,
					Action:    ActionAccept,
					Protocol:  ProtocolTCP,
					DestPort:  "443",
				},
			},
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
	if len(tmpl.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(tmpl.Rules.Input))
	}
}

func TestFirewallService_DeleteTemplate(t *testing.T) {
	// The doc documents "No output" for this endpoint, so an empty 200
	// body is correct as-is (spec/robot.yaml's response for this operation
	// has no content schema, matching).
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/template/7" {
			t.Errorf("expected path '/firewall/template/7', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	if err := client.Firewall.DeleteTemplate(ctx, "7"); err != nil {
		t.Fatalf("Firewall.DeleteTemplate returned error: %v", err)
	}
}

func TestFirewallService_ApplyTemplate(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// ApplyTemplate posts to POST /firewall/{server-id}, the same
		// operation as Activate/Disable/Update, so the response uses the
		// same {"firewall": {...}} envelope as the doc's POST example.
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"filter_ipv6":   false,
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
	})))
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

// makeInputRules returns n distinct accept input rules for exercising the
// inbound rule-limit validation.
func makeInputRules(n int) []FirewallRule {
	rules := make([]FirewallRule, n)
	for i := range rules {
		rules[i] = FirewallRule{
			Name:      fmt.Sprintf("rule %d", i),
			IPVersion: IPv4,
			Action:    ActionAccept,
			Protocol:  ProtocolTCP,
			DestPort:  strconv.Itoa(1000 + i),
		}
	}
	return rules
}

func TestFirewallService_ValidateRules(t *testing.T) {
	client := NewClient("test-user", "test-pass")

	tests := []struct {
		name       string
		inputRules int
		wantErr    bool
	}{
		{name: "empty", inputRules: 0, wantErr: false},
		{name: "at limit", inputRules: MaxFirewallInputRules, wantErr: false},
		{name: "over limit", inputRules: MaxFirewallInputRules + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Output rules are unbounded, so a large output set must not trip
			// the input-only limit.
			rules := FirewallRules{
				Input:  makeInputRules(tt.inputRules),
				Output: makeInputRules(MaxFirewallInputRules + 5),
			}

			err := client.Firewall.ValidateRules(rules)
			if tt.wantErr != (err != nil) {
				t.Fatalf("ValidateRules() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}

			if !IsFirewallRuleLimitExceededError(err) {
				t.Errorf("expected IsFirewallRuleLimitExceededError to be true for %v", err)
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("expected *Error, got %T", err)
			}
			if e.Kind != ErrKindValidation {
				t.Errorf("expected kind %q, got %q", ErrKindValidation, e.Kind)
			}
			if e.Status != http.StatusConflict {
				t.Errorf("expected status %d, got %d", http.StatusConflict, e.Status)
			}
		})
	}
}

func TestFirewallService_Update_InputRuleLimit(t *testing.T) {
	// The over-limit config must be rejected locally, so the server handler
	// must never be reached.
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("expected Update to reject over-limit rules before contacting the API")
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	status := FirewallStatusActive
	_, err := client.Firewall.Update(ctx, ServerID(321), UpdateConfig{
		Status: &status,
		Rules:  FirewallRules{Input: makeInputRules(MaxFirewallInputRules + 1)},
	})
	if err == nil {
		t.Fatal("expected Update to return an error for over-limit rules")
	}
	if !IsFirewallRuleLimitExceededError(err) {
		t.Errorf("expected IsFirewallRuleLimitExceededError to be true for %v", err)
	}
}

func TestWithMaxFirewallInputRules(t *testing.T) {
	// A raised ceiling must let a config that exceeds the default limit reach
	// the API instead of being rejected locally.
	posted := false
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posted = true
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}
		response := map[string]any{
			"firewall": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"filter_ipv6":   false,
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
	})))
	defer server.Close()

	overDefault := MaxFirewallInputRules + 1
	client := NewClient("test-user", "test-pass",
		WithBaseURL(server.URL),
		WithMaxFirewallInputRules(overDefault),
	)
	ctx := context.Background()

	status := FirewallStatusActive
	_, err := client.Firewall.Update(ctx, ServerID(321), UpdateConfig{
		Status: &status,
		Rules:  FirewallRules{Input: makeInputRules(overDefault)},
	})
	if err != nil {
		t.Fatalf("Update returned error with raised ceiling: %v", err)
	}
	if !posted {
		t.Error("expected Update to reach the API once the ceiling was raised")
	}

	// Non-positive overrides are ignored, so the default still applies.
	def := NewClient("test-user", "test-pass", WithMaxFirewallInputRules(0))
	if got := def.maxFirewallInputRules; got != MaxFirewallInputRules {
		t.Errorf("expected non-positive override to be ignored (%d), got %d", MaxFirewallInputRules, got)
	}
}

func TestFirewallService_Template_InputRuleLimit(t *testing.T) {
	// Templates with more than MaxFirewallInputRules input rules can never be
	// applied to a server, so Create/UpdateTemplate must reject them locally
	// without reaching the API.
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("expected template methods to reject over-limit rules before contacting the API")
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	config := TemplateConfig{
		Name:  "too-many-rules",
		Rules: FirewallRules{Input: makeInputRules(MaxFirewallInputRules + 1)},
	}

	ops := map[string]func() error{
		"CreateTemplate": func() error {
			_, err := client.Firewall.CreateTemplate(ctx, config)
			return err
		},
		"UpdateTemplate": func() error {
			_, err := client.Firewall.UpdateTemplate(ctx, "7", config)
			return err
		},
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			err := op()
			if err == nil {
				t.Fatalf("expected %s to return an error for over-limit rules", name)
			}
			if !IsFirewallRuleLimitExceededError(err) {
				t.Errorf("expected IsFirewallRuleLimitExceededError to be true for %v", err)
			}
		})
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
