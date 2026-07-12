package hrobot

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// IPService handles IP address related API operations.
//
// For reverse DNS operations, use Client.RDNS (RDNSService) instead.
// For traffic statistics, use Client.Traffic (TrafficService) instead.
type IPService struct {
	client *Client
}

// NewIPService creates a new IP service.
func NewIPService(client *Client) *IPService {
	return &IPService{client: client}
}

// List returns all IP addresses.
func (i *IPService) List(ctx context.Context) ([]IPAddress, error) {
	var ips []IPAddress
	err := i.client.Get(ctx, "/ip", &ips)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// Get returns details for a specific IP address.
func (i *IPService) Get(ctx context.Context, ip net.IP) (*IPAddress, error) {
	if ip == nil {
		return nil, NewParseError("invalid ip address", nil)
	}
	var ipAddr IPAddress
	path := fmt.Sprintf("/ip/%s", ip.String())
	err := i.client.Get(ctx, path, &ipAddr)
	if err != nil {
		return nil, err
	}
	return &ipAddr, nil
}

// SetTrafficWarnings enables or disables traffic warnings.
//
// POST /ip/{ip} returns the updated IP address resource (per the doc's
// Output table), so this returns it rather than discarding the response.
func (i *IPService) SetTrafficWarnings(ctx context.Context, ip net.IP, enabled bool) (*IPAddress, error) {
	if ip == nil {
		return nil, NewParseError("invalid ip address", nil)
	}
	path := fmt.Sprintf("/ip/%s", ip.String())

	data := url.Values{}
	if enabled {
		data.Set("traffic_warnings", "true")
	} else {
		data.Set("traffic_warnings", "false")
	}

	var ipAddr IPAddress
	if err := i.client.Post(ctx, path, data, &ipAddr); err != nil {
		return nil, err
	}
	return &ipAddr, nil
}

// CancelIP cancels an additional IP address.
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Cancel IPs via the Hetzner Robot UI.
func (i *IPService) CancelIP(context.Context, net.IP, string) error {
	return NewPolicyError("IPService.CancelIP")
}

// WithdrawIPCancellation withdraws an IP cancellation.
func (i *IPService) WithdrawIPCancellation(ctx context.Context, ip net.IP) error {
	if ip == nil {
		return NewParseError("invalid ip address", nil)
	}
	path := fmt.Sprintf("/ip/%s/cancellation", ip.String())
	return i.client.Delete(ctx, path)
}

// IPMAC represents the separate MAC address of an IP.
type IPMAC struct {
	IP  net.IP `json:"ip"`
	MAC string `json:"mac"`
}

// GetMAC retrieves the separate MAC address for an IP, if one is set.
//
// GET /ip/{ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-ip-ip-mac
func (i *IPService) GetMAC(ctx context.Context, ip net.IP) (*IPMAC, error) {
	if ip == nil {
		return nil, NewParseError("invalid ip address", nil)
	}
	path := fmt.Sprintf("/ip/%s/mac", ip.String())
	var mac IPMAC
	if err := i.client.Get(ctx, path, &mac); err != nil {
		return nil, err
	}
	return &mac, nil
}

// SetMAC generates a separate MAC address for an IP.
//
// PUT /ip/{ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#put-ip-ip-mac
func (i *IPService) SetMAC(ctx context.Context, ip net.IP) (*IPMAC, error) {
	if ip == nil {
		return nil, NewParseError("invalid ip address", nil)
	}
	path := fmt.Sprintf("/ip/%s/mac", ip.String())
	var mac IPMAC
	if err := i.client.Put(ctx, path, nil, &mac); err != nil {
		return nil, err
	}
	return &mac, nil
}

// DeleteMAC removes the separate MAC address from an IP.
//
// DELETE /ip/{ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-ip-ip-mac
func (i *IPService) DeleteMAC(ctx context.Context, ip net.IP) error {
	if ip == nil {
		return NewParseError("invalid ip address", nil)
	}
	path := fmt.Sprintf("/ip/%s/mac", ip.String())
	return i.client.Delete(ctx, path)
}
