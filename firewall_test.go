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
	assertRuleForm(t, r, "input", idx, rule)
}

// assertRuleForm asserts that the posted form contains the
// rules[direction][idx][*] keys/values matching the given doc-shaped rule.
func assertRuleForm(t *testing.T, r *http.Request, direction string, idx int, rule map[string]any) {
	t.Helper()
	for key, value := range rule {
		formKey := fmt.Sprintf("rules[%s][%d][%s]", direction, idx, key)
		got := r.FormValue(formKey)
		want := fmt.Sprintf("%v", value)
		if got != want {
			t.Errorf("expected form key %q to be %q, got %q", formKey, want, got)
		}
	}
}

// assertNoRuleForm asserts that the posted form carries no rule at
// rules[direction][idx], so a ruleset is not silently padded.
func assertNoRuleForm(t *testing.T, r *http.Request, direction string, idx int) {
	t.Helper()
	formKey := fmt.Sprintf("rules[%s][%d][action]", direction, idx)
	if got := r.FormValue(formKey); got != "" {
		t.Errorf("expected no rule at %s[%d], got form key %q = %q", direction, idx, formKey, got)
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

func TestFirewallRules_Equivalent(t *testing.T) {
	sshIn := FirewallRule{
		Name:      "ssh",
		IPVersion: IPv4,
		Action:    ActionAccept,
		Protocol:  ProtocolTCP,
		DestPort:  "22",
	}
	mail25 := FirewallRule{
		Name:     "block mail",
		Action:   ActionDiscard,
		Protocol: ProtocolTCP,
		DestPort: "25",
	}
	mail465 := FirewallRule{
		Name:     "block mail",
		Action:   ActionDiscard,
		Protocol: ProtocolTCP,
		DestPort: "465",
	}
	allowAll := FirewallRule{
		Name:   "Allow all",
		Action: ActionAccept,
	}

	withVersion := func(r FirewallRule, v IPVersion) FirewallRule {
		r.IPVersion = v
		return r
	}

	tests := []struct {
		name string
		a    FirewallRules
		b    FirewallRules
		want bool
	}{
		{
			name: "identical rulesets",
			a: FirewallRules{
				Input:  []FirewallRule{sshIn},
				Output: []FirewallRule{mail25, allowAll},
			},
			b: FirewallRules{
				Input:  []FirewallRule{sshIn},
				Output: []FirewallRule{mail25, allowAll},
			},
			want: true,
		},
		{
			name: "version-less rule equals adjacent ipv4+ipv6 pair",
			a:    FirewallRules{Output: []FirewallRule{mail25, allowAll}},
			b: FirewallRules{Output: []FirewallRule{
				withVersion(mail25, IPv4),
				withVersion(mail25, IPv6),
				allowAll,
			}},
			want: true,
		},
		{
			name: "version-less rules equal version-grouped expansion",
			a:    FirewallRules{Output: []FirewallRule{mail25, mail465}},
			b: FirewallRules{Output: []FirewallRule{
				withVersion(mail25, IPv4),
				withVersion(mail465, IPv4),
				withVersion(mail25, IPv6),
				withVersion(mail465, IPv6),
			}},
			want: true,
		},
		{
			name: "reordering within a version is not equivalent",
			a:    FirewallRules{Output: []FirewallRule{mail25, allowAll}},
			b:    FirewallRules{Output: []FirewallRule{allowAll, mail25}},
			want: false,
		},
		{
			name: "differing field is not equivalent",
			a:    FirewallRules{Output: []FirewallRule{mail25}},
			b:    FirewallRules{Output: []FirewallRule{mail465}},
			want: false,
		},
		{
			name: "expansion for one version only is not equivalent",
			a:    FirewallRules{Output: []FirewallRule{mail25}},
			b:    FirewallRules{Output: []FirewallRule{withVersion(mail25, IPv4)}},
			want: false,
		},
		{
			name: "missing rule is not equivalent",
			a:    FirewallRules{Output: []FirewallRule{mail25, allowAll}},
			b:    FirewallRules{Output: []FirewallRule{mail25}},
			want: false,
		},
		{
			name: "empty and nil rule lists are equivalent",
			a:    FirewallRules{Input: []FirewallRule{}, Output: nil},
			b:    FirewallRules{Input: nil, Output: []FirewallRule{}},
			want: true,
		},
		{
			name: "input rules are compared independently of output rules",
			a:    FirewallRules{Input: []FirewallRule{sshIn}},
			b:    FirewallRules{Output: []FirewallRule{sshIn}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equivalent(tt.b); got != tt.want {
				t.Errorf("a.Equivalent(b) = %v, want %v", got, tt.want)
			}
			if got := tt.b.Equivalent(tt.a); got != tt.want {
				t.Errorf("b.Equivalent(a) = %v, want %v", got, tt.want)
			}
		})
	}
}

// hetznerMailBlock returns the outgoing rule Hetzner enforces for accounts
// whose mail ports are not unblocked, in the given IP version. The API
// prepends this pair to every accepted configuration but marks it in no way,
// which is what ServerInjectedRules exists to work around.
func hetznerMailBlock(v IPVersion) FirewallRule {
	return FirewallRule{
		Name:      "Block mail ports",
		IPVersion: v,
		Action:    ActionDiscard,
		Protocol:  ProtocolTCP,
		DestPort:  "25,465",
	}
}

// assertRuleSlice compares two rule slices element by element.
func assertRuleSlice(t *testing.T, direction string, got, want []FirewallRule) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %d rules %+v, want %d", direction, len(got), got, len(want))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %+v, want %+v", direction, i, got[i], want[i])
		}
	}
}

func TestServerInjectedRules(t *testing.T) {
	mailV4, mailV6 := hetznerMailBlock(IPv4), hetznerMailBlock(IPv6)
	dns := FirewallRule{
		Name:      "dns",
		IPVersion: IPv4,
		Action:    ActionAccept,
		Protocol:  ProtocolUDP,
		DestPort:  "53",
	}
	allowAll := FirewallRule{Name: "Allow all", Action: ActionAccept}
	ssh := FirewallRule{
		Name:      "ssh",
		IPVersion: IPv4,
		Action:    ActionAccept,
		Protocol:  ProtocolTCP,
		DestPort:  "22",
	}

	withVersion := func(r FirewallRule, v IPVersion) FirewallRule {
		r.IPVersion = v
		return r
	}

	tests := []struct {
		name     string
		posted   FirewallRules
		returned FirewallRules
		want     FirewallRules
	}{
		{
			name:     "enforced mail block is reported as injected",
			posted:   FirewallRules{Output: []FirewallRule{dns, allowAll}},
			returned: FirewallRules{Output: []FirewallRule{mailV4, mailV6, dns, allowAll}},
			want:     FirewallRules{Output: []FirewallRule{mailV4, mailV6}},
		},
		{
			// Hetzner unblocks the mail ports on request, in which case
			// nothing is prepended and nothing may be reported.
			name:     "lifted mail block reports nothing",
			posted:   FirewallRules{Output: []FirewallRule{dns, allowAll}},
			returned: FirewallRules{Output: []FirewallRule{dns, allowAll}},
			want:     FirewallRules{},
		},
		{
			// The caller's own block-mail rule is indistinguishable from the
			// enforced one by shape, so only the diff keeps it.
			name:     "caller's own mail block is not reported",
			posted:   FirewallRules{Output: []FirewallRule{mailV4, allowAll}},
			returned: FirewallRules{Output: []FirewallRule{mailV4, allowAll}},
			want:     FirewallRules{},
		},
		{
			// What Activate does: the ruleset read back from Get is posted
			// again, so the API prepends a second copy of its own rules.
			name:     "re-posted internal rules are reported as duplicates",
			posted:   FirewallRules{Output: []FirewallRule{mailV4, mailV6, dns}},
			returned: FirewallRules{Output: []FirewallRule{mailV4, mailV6, mailV4, mailV6, dns}},
			want:     FirewallRules{Output: []FirewallRule{mailV4, mailV6}},
		},
		{
			name:   "version-less posted rule accounts for its expansion",
			posted: FirewallRules{Output: []FirewallRule{allowAll}},
			returned: FirewallRules{Output: []FirewallRule{
				withVersion(allowAll, IPv4),
				withVersion(allowAll, IPv6),
			}},
			want: FirewallRules{},
		},
		{
			// Conservative by design: a returned rule that is accounted for
			// in one of the versions it applies to is not claimed as
			// injected, so stripping it can never drop a caller's rule.
			name:     "rule accounted for in one version only is not reported",
			posted:   FirewallRules{Output: []FirewallRule{withVersion(allowAll, IPv4)}},
			returned: FirewallRules{Output: []FirewallRule{allowAll}},
			want:     FirewallRules{},
		},
		{
			name:     "injected input rule is reported",
			posted:   FirewallRules{Input: []FirewallRule{ssh}},
			returned: FirewallRules{Input: []FirewallRule{ssh, dns}},
			want:     FirewallRules{Input: []FirewallRule{dns}},
		},
		{
			name:     "input and output are diffed independently",
			posted:   FirewallRules{Input: []FirewallRule{ssh}},
			returned: FirewallRules{Input: []FirewallRule{ssh}, Output: []FirewallRule{mailV4}},
			want:     FirewallRules{Output: []FirewallRule{mailV4}},
		},
		{
			name:     "nothing posted reports the whole returned ruleset",
			posted:   FirewallRules{},
			returned: FirewallRules{Output: []FirewallRule{mailV4, mailV6}},
			want:     FirewallRules{Output: []FirewallRule{mailV4, mailV6}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ServerInjectedRules(tt.posted, tt.returned)
			assertRuleSlice(t, "input", got.Input, tt.want.Input)
			assertRuleSlice(t, "output", got.Output, tt.want.Output)
		})
	}
}

func TestFirewallRules_Without(t *testing.T) {
	mailV4, mailV6 := hetznerMailBlock(IPv4), hetznerMailBlock(IPv6)
	dns := FirewallRule{
		Name:      "dns",
		IPVersion: IPv4,
		Action:    ActionAccept,
		Protocol:  ProtocolUDP,
		DestPort:  "53",
	}
	allowAll := FirewallRule{Name: "Allow all", Action: ActionAccept}

	tests := []struct {
		name   string
		rules  FirewallRules
		remove FirewallRules
		want   FirewallRules
	}{
		{
			name:   "injected prefix is stripped",
			rules:  FirewallRules{Output: []FirewallRule{mailV4, mailV6, dns, allowAll}},
			remove: FirewallRules{Output: []FirewallRule{mailV4, mailV6}},
			want:   FirewallRules{Output: []FirewallRule{dns, allowAll}},
		},
		{
			// Multiplicity matters: stripping one injected copy must leave
			// the caller's identical rule in place.
			name:   "only one copy of a duplicated rule is removed",
			rules:  FirewallRules{Output: []FirewallRule{mailV4, mailV4, allowAll}},
			remove: FirewallRules{Output: []FirewallRule{mailV4}},
			want:   FirewallRules{Output: []FirewallRule{mailV4, allowAll}},
		},
		{
			name:   "removing an absent rule is a no-op",
			rules:  FirewallRules{Output: []FirewallRule{dns, allowAll}},
			remove: FirewallRules{Output: []FirewallRule{mailV4}},
			want:   FirewallRules{Output: []FirewallRule{dns, allowAll}},
		},
		{
			name:   "removing nothing keeps the ruleset",
			rules:  FirewallRules{Output: []FirewallRule{dns, allowAll}},
			remove: FirewallRules{},
			want:   FirewallRules{Output: []FirewallRule{dns, allowAll}},
		},
		{
			name:   "directions are stripped independently",
			rules:  FirewallRules{Input: []FirewallRule{dns}, Output: []FirewallRule{dns}},
			remove: FirewallRules{Output: []FirewallRule{dns}},
			want:   FirewallRules{Input: []FirewallRule{dns}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rules.Without(tt.remove)
			assertRuleSlice(t, "input", got.Input, tt.want.Input)
			assertRuleSlice(t, "output", got.Output, tt.want.Output)
		})
	}
}

// TestFirewallRules_ReconcileLoopConverges exercises the intended flow end to
// end: capture what the API injected once, then strip it from later reads so
// a desired ruleset compares equal instead of drifting forever.
func TestFirewallRules_ReconcileLoopConverges(t *testing.T) {
	mailV4, mailV6 := hetznerMailBlock(IPv4), hetznerMailBlock(IPv6)
	desired := FirewallRules{
		Output: []FirewallRule{
			{Name: "dns", IPVersion: IPv4, Action: ActionAccept, Protocol: ProtocolUDP, DestPort: "53"},
			{Name: "Allow all", Action: ActionAccept},
		},
	}

	applied := FirewallRules{Output: append([]FirewallRule{mailV4, mailV6}, desired.Output...)}

	if applied.Equivalent(desired) {
		t.Fatal("applied ruleset must not compare equal to desired while internal rules are present")
	}

	injected := ServerInjectedRules(desired, applied)
	if !injected.Equivalent(FirewallRules{Output: []FirewallRule{mailV4, mailV6}}) {
		t.Fatalf("unexpected injected ruleset: %+v", injected)
	}

	// A later Get returns the same configuration; stripping the injected
	// rules must now converge.
	if !applied.Without(injected).Equivalent(desired) {
		t.Errorf("reconcile did not converge: %+v", applied.Without(injected))
	}
}

// TestFirewallService_ActivateWithRules covers the reason the method exists:
// the ruleset returned by Get carries the API's internal rules, and posting
// those back is what makes them accumulate. ActivateWithRules must send the
// caller's ruleset instead.
func TestFirewallService_ActivateWithRules(t *testing.T) {
	// Doc-shaped JSON for the mail-port block the API enforces on the
	// outgoing chain, in both IP versions.
	mailBlockJSON := func(version string) map[string]any {
		return map[string]any{
			"name":       "Block mail ports",
			"ip_version": version,
			"action":     "discard",
			"protocol":   "tcp",
			"dst_port":   "25,465",
		}
	}
	allowAllJSON := map[string]any{"name": "Allow all", "action": "accept"}

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
						// The API reports its internal rules exactly like any
						// other rule, with no marker distinguishing them.
						"input": firewallDocRules(),
						"output": []map[string]any{
							mailBlockJSON("ipv4"),
							mailBlockJSON("ipv6"),
							allowAllJSON,
						},
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
			// whitelist_hos and filter_ipv6 still come from the current
			// configuration, since omitting them would reset them.
			if r.FormValue("whitelist_hos") != "true" {
				t.Errorf("expected whitelist_hos 'true', got '%s'", r.FormValue("whitelist_hos"))
			}
			if r.FormValue("filter_ipv6") != "false" {
				t.Errorf("expected filter_ipv6 'false', got '%s'", r.FormValue("filter_ipv6"))
			}

			assertRuleForm(t, r, "input", 0, map[string]any{
				"name":       "allow ssh",
				"ip_version": "ipv4",
				"action":     "accept",
				"protocol":   "tcp",
				"dst_port":   "22",
			})
			assertNoRuleForm(t, r, "input", 1)

			assertRuleForm(t, r, "output", 0, allowAllJSON)
			// The decisive assertion: the internal rules that GET reported
			// must not be posted back.
			assertNoRuleForm(t, r, "output", 1)

			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": true,
					"filter_ipv6":   false,
					"port":          "main",
					"rules": map[string]any{
						"input": []map[string]any{{
							"name":       "allow ssh",
							"ip_version": "ipv4",
							"action":     "accept",
							"protocol":   "tcp",
							"dst_port":   "22",
						}},
						// The API prepends its internal rules again.
						"output": []map[string]any{
							mailBlockJSON("ipv4"),
							mailBlockJSON("ipv6"),
							allowAllJSON,
						},
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

	desired := FirewallRules{
		Input: []FirewallRule{{
			Name:      "allow ssh",
			IPVersion: IPv4,
			Action:    ActionAccept,
			Protocol:  ProtocolTCP,
			DestPort:  "22",
		}},
		Output: []FirewallRule{{Name: "Allow all", Action: ActionAccept}},
	}

	config, err := client.Firewall.ActivateWithRules(ctx, ServerID(321), desired)
	if err != nil {
		t.Fatalf("Firewall.ActivateWithRules returned error: %v", err)
	}

	if !getCalled {
		t.Error("expected GET request to be made for whitelist_hos/filter_ipv6")
	}
	if !postCalled {
		t.Error("expected POST request to be made")
	}
	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	// The applied configuration differs from the desired one only by what the
	// API injected, which the caller can now identify and strip.
	injected := ServerInjectedRules(desired, config.Rules)
	assertRuleSlice(t, "injected output", injected.Output, []FirewallRule{
		hetznerMailBlock(IPv4),
		hetznerMailBlock(IPv6),
	})
	assertRuleSlice(t, "injected input", injected.Input, nil)
	if !config.Rules.Without(injected).Equivalent(desired) {
		t.Errorf("stripping injected rules did not yield the desired ruleset: %+v", config.Rules.Without(injected))
	}
}

// TestFirewallService_DisableWithRules asserts the disable path posts the
// caller's ruleset rather than the one Get returned.
func TestFirewallService_DisableWithRules(t *testing.T) {
	postCalled := false

	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rules := map[string]any{
			"input":  []map[string]any{},
			"output": []map[string]any{{"name": "Allow all", "action": "accept"}},
		}

		switch r.Method {
		case http.MethodGet:
			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": false,
					"filter_ipv6":   true,
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
			if r.FormValue("whitelist_hos") != "false" {
				t.Errorf("expected whitelist_hos 'false', got '%s'", r.FormValue("whitelist_hos"))
			}
			if r.FormValue("filter_ipv6") != "true" {
				t.Errorf("expected filter_ipv6 'true', got '%s'", r.FormValue("filter_ipv6"))
			}
			// The doc rules returned by GET must not be re-posted.
			assertNoRuleForm(t, r, "input", 0)
			assertRuleForm(t, r, "output", 0, map[string]any{"name": "Allow all", "action": "accept"})

			response := map[string]any{
				"firewall": map[string]any{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "disabled",
					"whitelist_hos": false,
					"filter_ipv6":   true,
					"port":          "main",
					"rules":         rules,
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

	desired := FirewallRules{Output: []FirewallRule{{Name: "Allow all", Action: ActionAccept}}}

	config, err := client.Firewall.DisableWithRules(context.Background(), ServerID(321), desired)
	if err != nil {
		t.Fatalf("Firewall.DisableWithRules returned error: %v", err)
	}
	if !postCalled {
		t.Error("expected POST request to be made")
	}
	if config.Status != FirewallStatusDisabled {
		t.Errorf("expected status 'disabled', got '%s'", config.Status)
	}
}
