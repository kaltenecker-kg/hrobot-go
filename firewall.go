package hrobot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kaltenecker-kg/hrobot-go/v2/internal/urlencode"
)

// MaxFirewallInputRules is the default maximum number of inbound (input)
// firewall rules Hetzner accepts per server. Submitting more makes the API
// respond with HTTP 409 and code FIREWALL_RULE_LIMIT_EXCEEDED. The ceiling
// enforced by a client can be overridden with WithMaxFirewallInputRules.
const MaxFirewallInputRules = 10

// FirewallService handles firewall-related API operations.
type FirewallService struct {
	client *Client
}

// NewFirewallService creates a new firewall service.
func NewFirewallService(client *Client) *FirewallService {
	return &FirewallService{client: client}
}

// ValidateRules checks a ruleset against documented Hetzner constraints
// without contacting the API. It currently enforces the inbound rule limit,
// returning a validation error with code FIREWALL_RULE_LIMIT_EXCEEDED when the
// number of input rules exceeds the client's configured ceiling (see
// WithMaxFirewallInputRules; MaxFirewallInputRules by default). Update,
// CreateTemplate, and UpdateTemplate run this check before posting, so callers
// only need to invoke it directly to fail fast (for example when validating
// user input up front).
func (f *FirewallService) ValidateRules(rules FirewallRules) error {
	return validateInputRuleCount(rules, f.client.maxFirewallInputRules)
}

// FirewallStatus represents the firewall status.
type FirewallStatus string

// Firewall status values reported by the API.
const (
	FirewallStatusActive    FirewallStatus = "active"
	FirewallStatusDisabled  FirewallStatus = "disabled"
	FirewallStatusInProcess FirewallStatus = "in process"
)

// FirewallRule represents a single firewall rule.
type FirewallRule struct {
	Name       string    `json:"name,omitempty"`
	IPVersion  IPVersion `json:"ip_version,omitempty"`
	Action     Action    `json:"action"`
	Protocol   Protocol  `json:"protocol,omitempty"`
	SourceIP   string    `json:"src_ip,omitempty"`
	DestIP     string    `json:"dst_ip,omitempty"`
	SourcePort string    `json:"src_port,omitempty"`
	DestPort   string    `json:"dst_port,omitempty"`
	TCPFlags   string    `json:"tcp_flags,omitempty"`
}

// FirewallConfig represents the complete firewall configuration.
type FirewallConfig struct {
	ServerIP     string         `json:"server_ip"`
	ServerNumber int            `json:"server_number"`
	Status       FirewallStatus `json:"status"`
	FilterIPv6   bool           `json:"filter_ipv6"`
	WhitelistHOS bool           `json:"whitelist_hos"`
	Port         string         `json:"port"`
	Rules        FirewallRules  `json:"rules"`
}

// FirewallRules contains input and output rules.
type FirewallRules struct {
	Input  []FirewallRule `json:"input"`
	Output []FirewallRule `json:"output"`
}

// appliesTo reports whether the rule matches traffic of the given IP version.
// A rule without an IP version applies to both versions.
func (r FirewallRule) appliesTo(v IPVersion) bool {
	return r.IPVersion == "" || r.IPVersion == v
}

// projectRules returns the ordered rules applicable to the given IP version,
// with IPVersion cleared so a version-less rule compares equal to its
// per-version expansion.
func projectRules(rules []FirewallRule, v IPVersion) []FirewallRule {
	var out []FirewallRule
	for _, r := range rules {
		if r.appliesTo(v) {
			r.IPVersion = ""
			out = append(out, r)
		}
	}
	return out
}

// rulesEquivalent reports whether two ordered rule lists filter traffic
// identically, by comparing their IPv4 and IPv6 projections.
func rulesEquivalent(a, b []FirewallRule) bool {
	for _, v := range []IPVersion{IPv4, IPv6} {
		pa, pb := projectRules(a, v), projectRules(b, v)
		if len(pa) != len(pb) {
			return false
		}
		for i := range pa {
			if pa[i] != pb[i] {
				return false
			}
		}
	}
	return true
}

// Equivalent reports whether both rulesets filter traffic identically.
//
// The API may return a normalized form of a posted ruleset: rules submitted
// without an IP version have been observed to come back expanded into
// separate ipv4 and ipv6 entries (the API doc does not specify a canonical
// returned form). Equivalent treats such expansions as equal by comparing,
// per IP version, the ordered sequence of rules that apply to that version.
// Rule order within a version is significant; rules are compared field by
// field otherwise.
func (r FirewallRules) Equivalent(other FirewallRules) bool {
	return rulesEquivalent(r.Input, other.Input) && rulesEquivalent(r.Output, other.Output)
}

// validateInputRuleCount enforces the inbound rule limit, returning a
// validation error with code FIREWALL_RULE_LIMIT_EXCEEDED when the number of
// input rules exceeds maxInput.
func validateInputRuleCount(rules FirewallRules, maxInput int) error {
	if n := len(rules.Input); n > maxInput {
		return NewValidationError(
			ErrFirewallRuleLimitExceeded,
			fmt.Sprintf("inbound firewall rule count %d exceeds the maximum of %d", n, maxInput),
			http.StatusConflict,
		)
	}
	return nil
}

// Get retrieves the firewall configuration for a server.
//
// The returned rules may be a normalized form of what was last posted; see
// FirewallRules.Equivalent for comparing rulesets across that normalization.
func (f *FirewallService) Get(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	var config FirewallConfig
	path := fmt.Sprintf("/firewall/%s", serverID.String())
	err := f.client.Get(ctx, path, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateConfig updates the firewall configuration.
// Nil pointers indicate unchanged fields (omitted from the request).
type UpdateConfig struct {
	Status       *FirewallStatus `json:"status,omitempty"`
	WhitelistHOS *bool           `json:"whitelist_hos,omitempty"`
	FilterIPv6   *bool           `json:"filter_ipv6,omitempty"`
	Rules        FirewallRules   `json:"rules,omitempty"`
}

// Update updates the firewall configuration for a server.
// Only non-nil fields in config are sent to the API.
//
// The rules echoed in the response (and by later Get calls) may be a
// normalized form of the posted rules; see FirewallRules.Equivalent for
// comparing rulesets across that normalization.
func (f *FirewallService) Update(ctx context.Context, serverID ServerID, config UpdateConfig) (*FirewallConfig, error) {
	if err := f.ValidateRules(config.Rules); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/firewall/%s", serverID.String())

	extra := make(map[string]string)
	if config.Status != nil {
		extra["status"] = string(*config.Status)
	}
	if config.WhitelistHOS != nil {
		extra["whitelist_hos"] = strconv.FormatBool(*config.WhitelistHOS)
	}
	if config.FilterIPv6 != nil {
		extra["filter_ipv6"] = strconv.FormatBool(*config.FilterIPv6)
	}

	formData := f.encodeRules(config.Rules, extra)

	var result FirewallConfig
	if err := f.client.PostRaw(ctx, path, formData, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// encodeRules encodes a FirewallRules pair plus extra fields into a
// pre-encoded body string. Hetzner requires literal `[`/`]` in the keys, so
// the result must be passed via PostRaw (not via url.Values).
func (f *FirewallService) encodeRules(rules FirewallRules, extra map[string]string) string {
	encoder := urlencode.NewFirewallRuleEncoder()
	for _, rule := range rules.Input {
		encoder.AddInputRule(f.encodeRule(rule))
	}
	for _, rule := range rules.Output {
		encoder.AddOutputRule(f.encodeRule(rule))
	}
	return encoder.EncodeToString(extra)
}

// encodeRule converts a FirewallRule to a map for URL encoding.
func (f *FirewallService) encodeRule(rule FirewallRule) map[string]string {
	data := make(map[string]string)

	if rule.Name != "" {
		data["name"] = rule.Name
	}
	if rule.IPVersion != "" {
		data["ip_version"] = string(rule.IPVersion)
	}
	data["action"] = string(rule.Action)
	if rule.Protocol != "" {
		data["protocol"] = string(rule.Protocol)
	}
	if rule.SourceIP != "" {
		data["src_ip"] = rule.SourceIP
	}
	if rule.DestIP != "" {
		data["dst_ip"] = rule.DestIP
	}
	if rule.SourcePort != "" {
		data["src_port"] = rule.SourcePort
	}
	if rule.DestPort != "" {
		data["dst_port"] = rule.DestPort
	}
	if rule.TCPFlags != "" {
		data["tcp_flags"] = rule.TCPFlags
	}

	return data
}

// Activate activates the firewall for a server.
//
// POST /firewall/{server-id} applies a full firewall configuration, so
// Activate re-posts the currently configured rules and whitelist_hos with
// only the status flipped to "active" (a read-modify-write over Get and
// Update). Posting status alone would replace the configuration with an
// empty ruleset and lock the server out of inbound traffic.
//
// The API may respond with status FIREWALL_IN_PROCESS while the change is
// applied; callers can use WaitForFirewallReady to wait for it to settle.
func (f *FirewallService) Activate(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	current, err := f.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	status := FirewallStatusActive
	updateConfig := UpdateConfig{
		Status:       &status,
		WhitelistHOS: &current.WhitelistHOS,
		FilterIPv6:   &current.FilterIPv6,
		Rules:        current.Rules,
	}

	return f.Update(ctx, serverID, updateConfig)
}

// Disable disables the firewall for a server.
//
// POST /firewall/{server-id} applies a full firewall configuration, so
// Disable re-posts the currently configured rules and whitelist_hos with
// only the status flipped to "disabled" (a read-modify-write over Get and
// Update). Posting status alone would replace the configuration with an
// empty ruleset and lock the server out of inbound traffic.
//
// The API may respond with status FIREWALL_IN_PROCESS while the change is
// applied; callers can use WaitForFirewallReady to wait for it to settle.
func (f *FirewallService) Disable(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	current, err := f.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}

	status := FirewallStatusDisabled
	updateConfig := UpdateConfig{
		Status:       &status,
		WhitelistHOS: &current.WhitelistHOS,
		FilterIPv6:   &current.FilterIPv6,
		Rules:        current.Rules,
	}

	return f.Update(ctx, serverID, updateConfig)
}

// Delete removes all firewall rules (resets to empty configuration).
func (f *FirewallService) Delete(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/firewall/%s", serverID.String())
	return f.client.Delete(ctx, path)
}

// FirewallTemplate represents a firewall template.
type FirewallTemplate struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	FilterIPv6   bool          `json:"filter_ipv6"`
	WhitelistHOS bool          `json:"whitelist_hos"`
	IsDefault    bool          `json:"is_default"`
	Rules        FirewallRules `json:"rules"`
}

// TemplateConfig is used for creating/updating templates.
type TemplateConfig struct {
	Name         string
	FilterIPv6   bool
	WhitelistHOS bool
	IsDefault    bool
	Rules        FirewallRules
}

// ListTemplates retrieves all firewall templates.
func (f *FirewallService) ListTemplates(ctx context.Context) ([]FirewallTemplate, error) {
	var templates []FirewallTemplate
	if err := f.client.Get(ctx, "/firewall/template", &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// GetTemplate retrieves a firewall template.
func (f *FirewallService) GetTemplate(ctx context.Context, templateID string) (*FirewallTemplate, error) {
	var tmpl FirewallTemplate
	path := fmt.Sprintf("/firewall/template/%s", templateID)
	if err := f.client.Get(ctx, path, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// CreateTemplate creates a new firewall template.
func (f *FirewallService) CreateTemplate(ctx context.Context, config TemplateConfig) (*FirewallTemplate, error) {
	if err := f.ValidateRules(config.Rules); err != nil {
		return nil, err
	}

	formData := f.encodeRules(config.Rules, templateExtras(config))

	var tmpl FirewallTemplate
	if err := f.client.PostRaw(ctx, "/firewall/template", formData, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// UpdateTemplate updates an existing firewall template.
func (f *FirewallService) UpdateTemplate(ctx context.Context, templateID string, config TemplateConfig) (*FirewallTemplate, error) {
	if err := f.ValidateRules(config.Rules); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/firewall/template/%s", templateID)

	formData := f.encodeRules(config.Rules, templateExtras(config))

	var tmpl FirewallTemplate
	if err := f.client.PostRaw(ctx, path, formData, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func templateExtras(config TemplateConfig) map[string]string {
	return map[string]string{
		"name":          config.Name,
		"filter_ipv6":   strconv.FormatBool(config.FilterIPv6),
		"whitelist_hos": strconv.FormatBool(config.WhitelistHOS),
		"is_default":    strconv.FormatBool(config.IsDefault),
	}
}

// DeleteTemplate deletes a firewall template.
func (f *FirewallService) DeleteTemplate(ctx context.Context, templateID string) error {
	path := fmt.Sprintf("/firewall/template/%s", templateID)
	return f.client.Delete(ctx, path)
}

// WaitForFirewallReady waits for the firewall to be ready (not in process state).
// It polls the firewall status with exponential backoff until it's ready or the context times out.
func (f *FirewallService) WaitForFirewallReady(ctx context.Context, serverID ServerID) error {
	return waitForCondition(ctx, func() (bool, error) {
		config, err := f.Get(ctx, serverID)
		if err != nil {
			return false, err
		}
		// Check if status is not "in process"
		return config.Status != FirewallStatusInProcess, nil
	})
}

// ApplyTemplate applies a firewall template to a server.
// Note: The whitelist_hos setting comes from the template itself and cannot be overridden.
func (f *FirewallService) ApplyTemplate(ctx context.Context, serverID ServerID, templateID string) (*FirewallConfig, error) {
	path := fmt.Sprintf("/firewall/%s", serverID.String())

	data := url.Values{}
	data.Set("template_id", templateID)
	// Note: whitelist_hos cannot be passed with template_id according to API docs

	var config FirewallConfig
	err := f.client.Post(ctx, path, data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
