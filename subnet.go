package hrobot

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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
//
// The Hetzner doc returns mask as a string and possible_mac as an
// {ip: mac} map, so we mirror those types here.
type SubnetMAC struct {
	IP          string            `json:"ip"`
	Mask        string            `json:"mask"`
	MAC         string            `json:"mac"`
	PossibleMAC map[string]string `json:"possible_mac"`
}

// SubnetCancellation represents the cancellation status of a subnet.
type SubnetCancellation struct {
	IP                       string  `json:"ip"`
	Mask                     string  `json:"mask"`
	ServerNumber             int     `json:"server_number"`
	EarliestCancellationDate string  `json:"earliest_cancellation_date"`
	Cancelled                bool    `json:"cancelled"`
	CancellationDate         *string `json:"cancellation_date"`
}

// List returns all subnets.
//
// GET /subnet
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-subnet
func (s *SubnetService) List(ctx context.Context) ([]SubnetResource, error) {
	var subnets []SubnetResource
	if err := s.client.Get(ctx, "/subnet", &subnets); err != nil {
		return nil, err
	}
	return subnets, nil
}

// Get returns details for a specific subnet.
//
// GET /subnet/{net-ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-subnet-net-ip
func (s *SubnetService) Get(ctx context.Context, netIP string) (*SubnetResource, error) {
	path := fmt.Sprintf("/subnet/%s", url.PathEscape(netIP))
	var subnet SubnetResource
	if err := s.client.Get(ctx, path, &subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

// Update updates traffic warning options for a subnet. All four fields are
// always sent; pass current values for the ones you want to leave alone.
//
// POST /subnet/{net-ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-subnet-net-ip
func (s *SubnetService) Update(ctx context.Context, netIP string, trafficWarnings bool, trafficHourly, trafficDaily, trafficMonthly int) (*SubnetResource, error) {
	path := fmt.Sprintf("/subnet/%s", url.PathEscape(netIP))
	data := url.Values{}
	data.Set("traffic_warnings", strconv.FormatBool(trafficWarnings))
	data.Set("traffic_hourly", strconv.Itoa(trafficHourly))
	data.Set("traffic_daily", strconv.Itoa(trafficDaily))
	data.Set("traffic_monthly", strconv.Itoa(trafficMonthly))

	var subnet SubnetResource
	if err := s.client.Post(ctx, path, data, &subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

// GetMAC retrieves the MAC address configuration for a subnet.
//
// GET /subnet/{net-ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-subnet-net-ip-mac
func (s *SubnetService) GetMAC(ctx context.Context, netIP string) (*SubnetMAC, error) {
	path := fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	var mac SubnetMAC
	if err := s.client.Get(ctx, path, &mac); err != nil {
		return nil, err
	}
	return &mac, nil
}

// SetMAC sets the MAC address for a subnet to the supplied value, which must
// be one of the entries from PossibleMAC returned by GetMAC.
//
// PUT /subnet/{net-ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#put-subnet-net-ip-mac
func (s *SubnetService) SetMAC(ctx context.Context, netIP string, mac string) (*SubnetMAC, error) {
	path := fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	data := url.Values{}
	data.Set("mac", mac)

	var result SubnetMAC
	if err := s.client.Put(ctx, path, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteMAC removes the custom MAC address assignment for a subnet, reverting
// to the default.
//
// DELETE /subnet/{net-ip}/mac
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-subnet-net-ip-mac
func (s *SubnetService) DeleteMAC(ctx context.Context, netIP string) error {
	path := fmt.Sprintf("/subnet/%s/mac", url.PathEscape(netIP))
	return s.client.Delete(ctx, path)
}

// GetCancellation retrieves the cancellation status for a subnet.
//
// GET /subnet/{net-ip}/cancellation
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-subnet-net-ip-cancellation
func (s *SubnetService) GetCancellation(ctx context.Context, netIP string) (*SubnetCancellation, error) {
	path := fmt.Sprintf("/subnet/%s/cancellation", url.PathEscape(netIP))
	var c SubnetCancellation
	if err := s.client.Get(ctx, path, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Cancel initiates cancellation of a subnet.
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Cancel subnets via the Hetzner Robot UI.
func (s *SubnetService) Cancel(_ context.Context, _ string, _ string) (*SubnetCancellation, error) {
	return nil, NewPolicyError("SubnetService.Cancel")
}

// WithdrawCancellation revokes a pending subnet cancellation.
//
// DELETE /subnet/{net-ip}/cancellation
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-subnet-net-ip-cancellation
func (s *SubnetService) WithdrawCancellation(ctx context.Context, netIP string) error {
	path := fmt.Sprintf("/subnet/%s/cancellation", url.PathEscape(netIP))
	return s.client.Delete(ctx, path)
}
