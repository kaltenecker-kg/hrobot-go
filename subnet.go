package hrobot

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// SubnetService handles subnet-related API operations.
type SubnetService struct {
	client *Client
}

// NewSubnetService creates a new subnet service.
func NewSubnetService(client *Client) *SubnetService {
	return &SubnetService{client: client}
}

// SubnetResource represents a subnet resource from the Subnet API.
type SubnetResource struct {
	IP              string `json:"ip"`
	Mask            int    `json:"mask"`
	Gateway         string `json:"gateway"`
	ServerIP        string `json:"server_ip"`
	ServerNumber    int    `json:"server_number"`
	Failover        bool   `json:"failover"`
	Locked          bool   `json:"locked"`
	TrafficWarnings bool   `json:"traffic_warnings"`
	TrafficHourly   int    `json:"traffic_hourly"`
	TrafficDaily    int    `json:"traffic_daily"`
	TrafficMonthly  int    `json:"traffic_monthly"`
}

// SubnetMAC represents the MAC address configuration for a subnet.
type SubnetMAC struct {
	IP          string   `json:"ip"`
	Mask        int      `json:"mask"`
	MAC         string   `json:"mac"`
	PossibleMAC []string `json:"possible_mac"`
}

// SubnetCancellation represents the cancellation status of a subnet.
type SubnetCancellation struct {
	IP                       string `json:"ip"`
	Mask                     int    `json:"mask"`
	ServerNumber             int    `json:"server_number"`
	EarliestCancellationDate string `json:"earliest_cancellation_date"`
	Cancelled                bool   `json:"cancelled"`
	CancellationDate         string `json:"cancellation_date"`
}

var errNotImplemented = errors.New("not implemented")

// List returns all subnets.
func (s *SubnetService) List(ctx context.Context) ([]SubnetResource, error) {
	return nil, errNotImplemented
}

// Get returns details for a specific subnet.
func (s *SubnetService) Get(ctx context.Context, netIP string) (*SubnetResource, error) {
	_ = fmt.Sprintf("/subnet/%s", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// Update updates traffic warning options for a subnet.
func (s *SubnetService) Update(ctx context.Context, netIP string, trafficWarnings bool, trafficHourly, trafficDaily, trafficMonthly int) (*SubnetResource, error) {
	_ = fmt.Sprintf("/subnet/%s", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// GetMAC retrieves the MAC address configuration for a subnet.
func (s *SubnetService) GetMAC(ctx context.Context, netIP string) (*SubnetMAC, error) {
	_ = fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// SetMAC generates a separate MAC address for a subnet.
func (s *SubnetService) SetMAC(ctx context.Context, netIP string, mac string) (*SubnetMAC, error) {
	_ = fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// DeleteMAC removes the custom MAC address assignment for a subnet.
func (s *SubnetService) DeleteMAC(ctx context.Context, netIP string) error {
	_ = fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	return errNotImplemented
}

// GetCancellation retrieves the cancellation status for a subnet.
func (s *SubnetService) GetCancellation(ctx context.Context, netIP string) (*SubnetCancellation, error) {
	_ = fmt.Sprintf("/subnet/%s/cancellation", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// Cancel initiates cancellation of a subnet.
func (s *SubnetService) Cancel(ctx context.Context, netIP string, cancellationDate string) (*SubnetCancellation, error) {
	_ = fmt.Sprintf("/subnet/%s/cancellation", url.PathEscape(netIP))
	return nil, errNotImplemented
}

// WithdrawCancellation revokes a pending subnet cancellation.
func (s *SubnetService) WithdrawCancellation(ctx context.Context, netIP string) error {
	_ = fmt.Sprintf("/subnet/%s/cancellation", url.PathEscape(netIP))
	return errNotImplemented
}
