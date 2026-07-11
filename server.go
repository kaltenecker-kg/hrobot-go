package hrobot

import (
	"context"
	"fmt"
)

// ServerService handles server-related API operations.
type ServerService struct {
	client *Client
}

// NewServerService creates a new server service.
func NewServerService(client *Client) *ServerService {
	return &ServerService{client: client}
}

// List returns all servers.
func (s *ServerService) List(ctx context.Context) ([]Server, error) {
	var servers []Server
	err := s.client.Get(ctx, "/server", &servers)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// Get returns details for a specific server.
func (s *ServerService) Get(ctx context.Context, serverID ServerID) (*Server, error) {
	var server Server
	path := fmt.Sprintf("/server/%s", serverID.String())
	err := s.client.Get(ctx, path, &server)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// SetName sets the name for a server.
func (s *ServerService) SetName(ctx context.Context, serverID ServerID, name string) (*Server, error) {
	var server Server
	path := fmt.Sprintf("/server/%s", serverID.String())

	data := make(map[string]string)
	data["server_name"] = name

	err := s.client.Post(ctx, path, encodeForm(data), &server)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// Cancellation represents a server cancellation request.
type Cancellation struct {
	ServerID           ServerID
	CancellationDate   string
	CancellationReason string
}

// RequestCancellation requests cancellation of a server.
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Cancel servers via the Hetzner Robot UI.
func (s *ServerService) RequestCancellation(context.Context, Cancellation) error {
	return NewPolicyError("ServerService.RequestCancellation")
}

// WithdrawCancellation withdraws a server cancellation request.
func (s *ServerService) WithdrawCancellation(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/server/%s/cancellation", serverID.String())
	return s.client.Delete(ctx, path)
}
