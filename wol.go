package hrobot

import (
	"context"
	"fmt"
)

// WOLService handles Wake-on-LAN related API operations.
type WOLService struct {
	client *Client
}

// NewWOLService creates a new WOL service.
func NewWOLService(client *Client) *WOLService {
	return &WOLService{client: client}
}

// WOLResponse represents the response from a Wake-on-LAN request.
type WOLResponse struct {
	ServerIP      string `json:"server_ip"`
	ServerIPv6Net string `json:"server_ipv6_net"`
	ServerNumber  int    `json:"server_number"`
}

// Send sends a Wake-on-LAN packet to the server.
func (w *WOLService) Send(ctx context.Context, serverID ServerID) (*WOLResponse, error) {
	var resp WOLResponse
	path := fmt.Sprintf("/wol/%s", serverID.String())
	if err := w.client.Post(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get queries Wake-on-LAN data for a server, indicating whether WOL is
// available without sending a packet.
//
// GET /wol/{server-number}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-wol-server-number
func (w *WOLService) Get(ctx context.Context, serverID ServerID) (*WOLResponse, error) {
	var resp WOLResponse
	path := fmt.Sprintf("/wol/%s", serverID.String())
	if err := w.client.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
