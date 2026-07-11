package hrobot

import (
	"context"
	"encoding/json"
	"net/url"
)

// TrafficService provides access to traffic related functions in the Hetzner Robot API.
type TrafficService struct {
	client *Client
}

// NewTrafficService creates a new TrafficService.
func NewTrafficService(client *Client) *TrafficService {
	return &TrafficService{client: client}
}

// TrafficType represents the type of traffic data to retrieve.
type TrafficType string

// Traffic aggregation windows.
const (
	TrafficTypeDay   TrafficType = "day"
	TrafficTypeMonth TrafficType = "month"
	TrafficTypeYear  TrafficType = "year"
)

// ServerTrafficData represents traffic statistics for a server.
type ServerTrafficData struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
	// Data holds the aggregate traffic per IP (GB), populated when the
	// request did not set SingleValues.
	Data map[string]TrafficStats
	// SingleValues holds per-interval traffic per IP (GB), keyed by
	// interval (e.g. date), populated when the request set
	// SingleValues to true.
	SingleValues map[string]map[string]TrafficStats
}

// TrafficStats represents traffic statistics for a specific time period.
type TrafficStats struct {
	In  float64 `json:"in"`  // Incoming traffic in GB
	Out float64 `json:"out"` // Outgoing traffic in GB
	Sum float64 `json:"sum"` // Total traffic in GB
}

// TrafficGetParams represents parameters for retrieving traffic data.
// The From and To date formats depend on Type:
//   - day: YYYY-MM-DDTHH (e.g., "2025-01-15T14")
//   - month: YYYY-MM-DD (e.g., "2025-01-15")
//   - year: YYYY-MM (e.g., "2025-01")
type TrafficGetParams struct {
	Type         TrafficType // Type of data (day, month, year)
	From         string      // Start date (format depends on Type; see comments)
	To           string      // End date (format depends on Type; see comments)
	IP           string      // Server IP address (optional; omit for all IPs)
	SingleValues bool        // Return single values per day/month/year
}

// Get retrieves traffic statistics for a server.
//
// POST /traffic
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-traffic
func (t *TrafficService) Get(ctx context.Context, params TrafficGetParams) (*ServerTrafficData, error) {
	path := "/traffic"

	// Build form data (API uses POST, not GET)
	formData := url.Values{}
	formData.Set("type", string(params.Type))
	formData.Set("from", params.From)
	formData.Set("to", params.To)
	if params.IP != "" {
		formData.Set("ip", params.IP)
	}
	if params.SingleValues {
		formData.Set("single_values", "true")
	}

	// The shape of the "data" field depends on whether single_values was
	// requested: an aggregate per IP by default, or per-interval values
	// per IP with single_values=true. Decode it separately based on which
	// mode was requested.
	var raw struct {
		Type string          `json:"type"`
		From string          `json:"from"`
		To   string          `json:"to"`
		Data json.RawMessage `json:"data"`
	}
	if err := t.client.Post(ctx, path, formData, &raw); err != nil {
		return nil, err
	}

	result := ServerTrafficData{
		Type: raw.Type,
		From: raw.From,
		To:   raw.To,
	}

	if len(raw.Data) > 0 {
		if params.SingleValues {
			if err := json.Unmarshal(raw.Data, &result.SingleValues); err != nil {
				return nil, NewParseError("failed to decode traffic single_values data", err)
			}
		} else {
			if err := json.Unmarshal(raw.Data, &result.Data); err != nil {
				return nil, NewParseError("failed to decode traffic data", err)
			}
		}
	}

	return &result, nil
}
