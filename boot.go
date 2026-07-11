package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// BootService handles boot configuration API operations.
type BootService struct {
	client *Client
}

// NewBootService creates a new boot service.
func NewBootService(client *Client) *BootService {
	return &BootService{client: client}
}

// BootConfig represents boot configuration response.
type BootConfig struct {
	Rescue  *RescueConfig  `json:"rescue,omitempty"`
	Linux   *LinuxConfig   `json:"linux,omitempty"`
	VNC     *VNCConfig     `json:"vnc,omitempty"`
	Windows *WindowsConfig `json:"windows,omitempty"`
	Plesk   *PleskConfig   `json:"plesk,omitempty"`
	CPanel  *CPanelConfig  `json:"cpanel,omitempty"`
}

// BootKey describes one SSH key attached to a rescue/linux activation.
type BootKey struct {
	Key BootKeyDetail `json:"key"`
}

// BootKeyDetail holds the metadata the API returns for an SSH key attached
// to a rescue/linux activation.
type BootKeyDetail struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
}

// RescueConfig represents rescue system configuration.
type RescueConfig struct {
	ServerIP       string    `json:"server_ip"`
	ServerIPv6Net  string    `json:"server_ipv6_net"`
	ServerNumber   int       `json:"server_number"`
	Active         bool      `json:"active"`
	OS             any       `json:"os,omitempty"`   // string when active, []string when not
	Arch           any       `json:"arch,omitempty"` // int when active, []int when not
	AuthorizedKeys []BootKey `json:"authorized_key,omitempty"`
	HostKeys       []BootKey `json:"host_key,omitempty"`
	Password       *string   `json:"password,omitempty"`
}

// LinuxConfig represents Linux installation configuration.
type LinuxConfig struct {
	ServerIP       string    `json:"server_ip"`
	ServerIPv6Net  string    `json:"server_ipv6_net"`
	ServerNumber   int       `json:"server_number"`
	Dist           any       `json:"dist"` // string when active, []string when not
	Arch           any       `json:"arch"` // int when active, []int when not
	Lang           any       `json:"lang"` // string when active, []string when not
	Active         bool      `json:"active"`
	Hostname       string    `json:"hostname,omitempty"`
	Password       *string   `json:"password,omitempty"`
	AuthorizedKeys []BootKey `json:"authorized_key,omitempty"`
	HostKeys       []BootKey `json:"host_key,omitempty"`
}

// VNCConfig represents VNC configuration.
type VNCConfig struct {
	ServerIP      string  `json:"server_ip"`
	ServerIPv6Net string  `json:"server_ipv6_net"`
	ServerNumber  int     `json:"server_number"`
	Active        bool    `json:"active"`
	Dist          any     `json:"dist,omitempty"` // string when active, []string when not
	Arch          any     `json:"arch,omitempty"` // int when active, []int when not
	Lang          any     `json:"lang,omitempty"` // string when active, []string when not
	Password      *string `json:"password,omitempty"`
}

// WindowsConfig represents Windows installation configuration.
type WindowsConfig struct {
	ServerIP      string  `json:"server_ip"`
	ServerIPv6Net string  `json:"server_ipv6_net"`
	ServerNumber  int     `json:"server_number"`
	Active        bool    `json:"active"`
	OS            any     `json:"os,omitempty"`   // string when active, []string when not
	Lang          any     `json:"lang,omitempty"` // string when active, []string when not
	Password      *string `json:"password,omitempty"`
}

// ActiveOS returns the OS chosen for this rescue session, or "" if rescue
// is not currently active. When inactive, see AvailableOS for the choices
// the API offers.
func (c *RescueConfig) ActiveOS() string { return scalarString(c.OS) }

// AvailableOS lists the OS choices the API offers when rescue is not
// active. Returns nil when rescue is active (use ActiveOS instead).
func (c *RescueConfig) AvailableOS() []string { return optionStrings(c.OS) }

// ActiveArch returns the architecture chosen for this rescue session, or
// 0 if rescue is not currently active.
func (c *RescueConfig) ActiveArch() int { return scalarInt(c.Arch) }

// AvailableArchs lists the architecture choices the API offers when
// rescue is not active.
func (c *RescueConfig) AvailableArchs() []int { return optionInts(c.Arch) }

// ActiveDist returns the distribution chosen for this Linux install, or
// "" if not currently active.
func (c *LinuxConfig) ActiveDist() string { return scalarString(c.Dist) }

// AvailableDists lists the distributions the API offers when not active.
func (c *LinuxConfig) AvailableDists() []string { return optionStrings(c.Dist) }

// ActiveArch returns the architecture chosen for this Linux install, or
// 0 if not currently active.
func (c *LinuxConfig) ActiveArch() int { return scalarInt(c.Arch) }

// AvailableArchs lists the architecture choices the API offers when not
// active.
func (c *LinuxConfig) AvailableArchs() []int { return optionInts(c.Arch) }

// ActiveLang returns the language chosen for this Linux install, or "" if
// not currently active.
func (c *LinuxConfig) ActiveLang() string { return scalarString(c.Lang) }

// AvailableLangs lists the language choices the API offers when not
// active.
func (c *LinuxConfig) AvailableLangs() []string { return optionStrings(c.Lang) }

// ActiveDist returns the distribution chosen for this VNC session, or ""
// if not currently active.
func (c *VNCConfig) ActiveDist() string { return scalarString(c.Dist) }

// AvailableDists lists the distributions the API offers when not active.
func (c *VNCConfig) AvailableDists() []string { return optionStrings(c.Dist) }

// ActiveArch returns the architecture chosen for this VNC session, or 0
// if not currently active.
func (c *VNCConfig) ActiveArch() int { return scalarInt(c.Arch) }

// AvailableArchs lists the architecture choices the API offers when not
// active.
func (c *VNCConfig) AvailableArchs() []int { return optionInts(c.Arch) }

// ActiveLang returns the language chosen for this VNC session, or "" if
// not currently active.
func (c *VNCConfig) ActiveLang() string { return scalarString(c.Lang) }

// AvailableLangs lists the language choices the API offers when not
// active.
func (c *VNCConfig) AvailableLangs() []string { return optionStrings(c.Lang) }

// ActiveOS returns the OS chosen for this Windows install, or "" if not
// currently active.
func (c *WindowsConfig) ActiveOS() string { return scalarString(c.OS) }

// AvailableOS lists the OS choices the API offers when not active.
func (c *WindowsConfig) AvailableOS() []string { return optionStrings(c.OS) }

// ActiveLang returns the language chosen for this Windows install, or ""
// if not currently active.
func (c *WindowsConfig) ActiveLang() string { return scalarString(c.Lang) }

// AvailableLangs lists the language choices the API offers when not
// active.
func (c *WindowsConfig) AvailableLangs() []string { return optionStrings(c.Lang) }

// scalarString returns v as a string when it holds the active value, or
// "" otherwise (including when it holds the list of choices).
func scalarString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// optionStrings returns v as a slice of strings when it holds the list of
// choices offered by the API, or nil otherwise.
func optionStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// scalarInt returns v as an int when it holds the active value. Numbers
// unmarshalled from JSON arrive as float64; the conversion truncates.
func scalarInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// optionInts returns v as a slice of ints when it holds the list of
// choices offered by the API, or nil otherwise.
func optionInts(v any) []int {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(arr))
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			out = append(out, int(n))
		case int:
			out = append(out, n)
		}
	}
	return out
}

// PleskConfig represents Plesk installation configuration.
type PleskConfig struct {
	Active   bool   `json:"active"`
	Hostname string `json:"hostname,omitempty"`
}

// CPanelConfig represents cPanel installation configuration.
type CPanelConfig struct {
	Active   bool   `json:"active"`
	Hostname string `json:"hostname,omitempty"`
}

// Get retrieves the boot configuration for a server.
func (b *BootService) Get(ctx context.Context, serverID ServerID) (*BootConfig, error) {
	var config BootConfig
	path := fmt.Sprintf("/boot/%s", serverID.String())
	err := b.client.Get(ctx, path, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ActivateRescue activates the rescue system. fingerprints are the
// fingerprints of one or more SSH keys already stored in Robot; the API
// does not accept full public keys here.
func (b *BootService) ActivateRescue(ctx context.Context, serverID ServerID, os string, arch int, fingerprints []string) (*RescueConfig, error) {
	path := fmt.Sprintf("/boot/%s/rescue", serverID.String())

	data := url.Values{}
	data.Set("os", os)
	data.Set("arch", fmt.Sprintf("%d", arch))

	for _, fingerprint := range fingerprints {
		data.Add("authorized_key[]", fingerprint)
	}

	var rescue RescueConfig
	err := b.client.Post(ctx, path, data, &rescue)
	if err != nil {
		return nil, err
	}

	return &rescue, nil
}

// DeactivateRescue deactivates the rescue system.
func (b *BootService) DeactivateRescue(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/rescue", serverID.String())
	return b.client.Delete(ctx, path)
}

// GetLastRescue retrieves the last activated rescue system information.
func (b *BootService) GetLastRescue(ctx context.Context, serverID ServerID) (*RescueConfig, error) {
	var rescue RescueConfig
	path := fmt.Sprintf("/boot/%s/rescue/last", serverID.String())
	err := b.client.Get(ctx, path, &rescue)
	if err != nil {
		return nil, err
	}
	return &rescue, nil
}

// ActivateLinux activates Linux installation.
func (b *BootService) ActivateLinux(ctx context.Context, serverID ServerID, dist string, arch int, lang string, authorizedKeys []string) (*LinuxConfig, error) {
	path := fmt.Sprintf("/boot/%s/linux", serverID.String())

	data := url.Values{}
	data.Set("dist", dist)
	data.Set("arch", fmt.Sprintf("%d", arch))
	data.Set("lang", lang)

	for _, key := range authorizedKeys {
		data.Add("authorized_key[]", key)
	}

	var linux LinuxConfig
	err := b.client.Post(ctx, path, data, &linux)
	if err != nil {
		return nil, err
	}

	return &linux, nil
}

// DeactivateLinux deactivates Linux installation.
func (b *BootService) DeactivateLinux(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/linux", serverID.String())
	return b.client.Delete(ctx, path)
}

// ActivateVNC activates VNC installation.
func (b *BootService) ActivateVNC(ctx context.Context, serverID ServerID, dist string, arch int, lang string) (*VNCConfig, error) {
	path := fmt.Sprintf("/boot/%s/vnc", serverID.String())

	data := url.Values{}
	data.Set("dist", dist)
	data.Set("arch", fmt.Sprintf("%d", arch))
	data.Set("lang", lang)

	var vnc VNCConfig
	err := b.client.Post(ctx, path, data, &vnc)
	if err != nil {
		return nil, err
	}

	return &vnc, nil
}

// DeactivateVNC deactivates VNC installation.
func (b *BootService) DeactivateVNC(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/vnc", serverID.String())
	return b.client.Delete(ctx, path)
}

// GetLastLinux retrieves the last activated Linux installation information.
func (b *BootService) GetLastLinux(ctx context.Context, serverID ServerID) (*LinuxConfig, error) {
	var linux LinuxConfig
	path := fmt.Sprintf("/boot/%s/linux/last", serverID.String())
	err := b.client.Get(ctx, path, &linux)
	if err != nil {
		return nil, err
	}
	return &linux, nil
}

// GetWindows retrieves the Windows installation configuration.
func (b *BootService) GetWindows(ctx context.Context, serverID ServerID) (*WindowsConfig, error) {
	var windows WindowsConfig
	path := fmt.Sprintf("/boot/%s/windows", serverID.String())
	err := b.client.Get(ctx, path, &windows)
	if err != nil {
		return nil, err
	}
	return &windows, nil
}

// ActivateWindows activates Windows installation.
func (b *BootService) ActivateWindows(ctx context.Context, serverID ServerID, lang string) (*WindowsConfig, error) {
	path := fmt.Sprintf("/boot/%s/windows", serverID.String())

	data := url.Values{}
	data.Set("lang", lang)

	var windows WindowsConfig
	err := b.client.Post(ctx, path, data, &windows)
	if err != nil {
		return nil, err
	}

	return &windows, nil
}

// DeactivateWindows deactivates Windows installation.
func (b *BootService) DeactivateWindows(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/windows", serverID.String())
	return b.client.Delete(ctx, path)
}
