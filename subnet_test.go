package hrobot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/internal/spectest"
)

func TestSubnetService_Cancel_DisallowedByPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("Cancel must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))

	_, err := client.Subnet.Cancel(context.Background(), "2001:db8::", "2024-12-31")
	if !IsPolicyError(err) {
		t.Fatalf("expected policy error, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 451 {
		t.Fatalf("expected status 451, got %v", err)
	}
}

func subnetFixture() string {
	return `{
		"subnet": {
			"ip": "123.123.123.123",
			"mask": 29,
			"gateway": "123.123.123.123",
			"server_ip": "88.198.123.123",
			"server_number": 321,
			"failover": false,
			"locked": false,
			"traffic_warnings": false,
			"traffic_hourly": 100,
			"traffic_daily": 500,
			"traffic_monthly": 2
		}
	}`
}

func TestSubnetService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subnet" {
			t.Errorf("expected '/subnet', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		body := `[
			{
				"subnet": {
					"ip": "123.123.123.123",
					"mask": 29,
					"gateway": "123.123.123.123",
					"server_ip": "88.198.123.123",
					"server_number": 321,
					"failover": false,
					"locked": false,
					"traffic_warnings": false,
					"traffic_hourly": 100,
					"traffic_daily": 500,
					"traffic_monthly": 2
				}
			},
			{
				"subnet": {
					"ip": "178.63.123.123",
					"mask": 25,
					"gateway": "178.63.123.124",
					"server_ip": null,
					"server_number": 421,
					"failover": false,
					"locked": false,
					"traffic_warnings": false,
					"traffic_hourly": 100,
					"traffic_daily": 500,
					"traffic_monthly": 2
				}
			}
		]`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	subnets, err := client.Subnet.List(context.Background())
	if err != nil {
		t.Fatalf("Subnet.List returned error: %v", err)
	}
	if len(subnets) != 2 {
		t.Fatalf("expected 2 subnets, got %d", len(subnets))
	}
	if subnets[0].IP != "123.123.123.123" || subnets[0].ServerNumber != 321 || subnets[0].Mask != 29 {
		t.Errorf("unexpected result for subnets[0]: %+v", subnets[0])
	}
	if subnets[1].IP != "178.63.123.123" || subnets[1].ServerNumber != 421 || subnets[1].ServerIP != "" {
		t.Errorf("unexpected result for subnets[1]: %+v", subnets[1])
	}
}

func TestSubnetService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subnet/123.123.123.123" {
			t.Errorf("expected subnet path, got '%s'", r.URL.Path)
		}
		if _, err := w.Write([]byte(subnetFixture())); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	subnet, err := client.Subnet.Get(context.Background(), "123.123.123.123")
	if err != nil {
		t.Fatalf("Subnet.Get returned error: %v", err)
	}
	if subnet.Mask != 29 {
		t.Errorf("expected mask 29, got %d", subnet.Mask)
	}
	if subnet.Gateway != "123.123.123.123" {
		t.Errorf("expected gateway '123.123.123.123', got '%s'", subnet.Gateway)
	}
	if subnet.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", subnet.ServerNumber)
	}
}

func TestSubnetService_Update(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got '%s'", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("traffic_warnings") != "true" {
			t.Errorf("traffic_warnings: got '%s'", r.FormValue("traffic_warnings"))
		}
		if r.FormValue("traffic_hourly") != "1" {
			t.Errorf("traffic_hourly: got '%s'", r.FormValue("traffic_hourly"))
		}
		if r.FormValue("traffic_daily") != "5" {
			t.Errorf("traffic_daily: got '%s'", r.FormValue("traffic_daily"))
		}
		if r.FormValue("traffic_monthly") != "9" {
			t.Errorf("traffic_monthly: got '%s'", r.FormValue("traffic_monthly"))
		}
		body := `{
			"subnet": {
				"ip": "123.123.123.123",
				"mask": 29,
				"gateway": "123.123.123.123",
				"server_ip": "88.198.123.123",
				"server_number": 321,
				"failover": false,
				"locked": false,
				"traffic_warnings": true,
				"traffic_hourly": 100,
				"traffic_daily": 500,
				"traffic_monthly": 2
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	subnet, err := client.Subnet.Update(context.Background(), "123.123.123.123", true, 1, 5, 9)
	if err != nil {
		t.Fatalf("Subnet.Update returned error: %v", err)
	}
	if !subnet.TrafficWarnings {
		t.Error("expected traffic warnings to be enabled in response")
	}
}

func subnetMACFixture(mac string) string {
	return `{
		"mac": {
			"ip": "2a01:4f8:111:4221::",
			"mask": "64",
			"mac": "` + mac + `",
			"possible_mac": {
				"123.123.123.123": "00:21:85:62:3e:9c",
				"123.123.123.124": "00:21:85:62:3e:9d"
			}
		}
	}`
}

func TestSubnetService_GetMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		if _, err := w.Write([]byte(subnetMACFixture("00:21:85:62:3e:9c"))); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	mac, err := client.Subnet.GetMAC(context.Background(), "2a01:4f8:111:4221::")
	if err != nil {
		t.Fatalf("Subnet.GetMAC returned error: %v", err)
	}
	if mac.MAC != "00:21:85:62:3e:9c" {
		t.Errorf("expected mac, got '%s'", mac.MAC)
	}
	if mac.Mask != "64" {
		t.Errorf("expected mask '64', got '%s'", mac.Mask)
	}
	if len(mac.PossibleMAC) != 2 {
		t.Errorf("expected 2 possible_mac entries, got %d", len(mac.PossibleMAC))
	}
}

func TestSubnetService_SetMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got '%s'", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("mac") != "00:21:85:62:3e:9d" {
			t.Errorf("expected mac form value, got '%s'", r.FormValue("mac"))
		}
		if _, err := w.Write([]byte(subnetMACFixture("00:21:85:62:3e:9d"))); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	mac, err := client.Subnet.SetMAC(context.Background(), "2a01:4f8:111:4221::", "00:21:85:62:3e:9d")
	if err != nil {
		t.Fatalf("Subnet.SetMAC returned error: %v", err)
	}
	if mac.MAC != "00:21:85:62:3e:9d" {
		t.Errorf("expected mac '00:21:85:62:3e:9d', got '%s'", mac.MAC)
	}
}

func TestSubnetService_DeleteMAC(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		// Doc-verbatim example response body for
		// DELETE /subnet/{net-ip}/mac: reverts to the default MAC (the
		// server's main IP MAC), still shaped as a MACAddress envelope.
		if _, err := w.Write([]byte(subnetMACFixture("00:21:85:62:3e:9c"))); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if err := client.Subnet.DeleteMAC(context.Background(), "2a01:4f8:111:4221::"); err != nil {
		t.Fatalf("Subnet.DeleteMAC returned error: %v", err)
	}
}

func TestSubnetService_GetCancellation(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		// Doc-verbatim example from GET /subnet/{net-ip}/cancellation. Note
		// the doc's example body uses the key "cancellation-date" (hyphen)
		// while the field description table below it documents
		// "cancellation_date" (underscore) as the field name; the hyphen
		// form appears to be a documentation typo since it is inconsistent
		// with every other field in the same object (and with the POST/
		// DELETE examples for this same resource).
		body := `{
			"cancellation": {
				"ip": "123.123.123.123",
				"mask": "29",
				"server_number": 321,
				"earliest_cancellation_date": "2022-02-11",
				"cancelled": false,
				"cancellation_date": null
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	c, err := client.Subnet.GetCancellation(context.Background(), "123.123.123.123")
	if err != nil {
		t.Fatalf("Subnet.GetCancellation returned error: %v", err)
	}
	if c.IP != "123.123.123.123" {
		t.Errorf("expected ip '123.123.123.123', got '%s'", c.IP)
	}
	if c.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", c.ServerNumber)
	}
	if c.EarliestCancellationDate != "2022-02-11" {
		t.Errorf("expected earliest_cancellation_date '2022-02-11', got '%s'", c.EarliestCancellationDate)
	}
	if c.Cancelled {
		t.Error("expected cancelled false")
	}
	if c.CancellationDate != nil {
		t.Errorf("expected nil cancellation_date, got %v", *c.CancellationDate)
	}
}

func TestSubnetService_WithdrawCancellation(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got '%s'", r.Method)
		}
		// Doc-verbatim example response body for
		// DELETE /subnet/{net-ip}/cancellation.
		body := `{
			"cancellation": {
				"ip": "123.123.123.123",
				"mask": "29",
				"server_number": 321,
				"earliest_cancellation_date": "2022-02-11",
				"cancelled": false,
				"cancellation_date": null
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	if err := client.Subnet.WithdrawCancellation(context.Background(), "2a01:4f8:111:4221::"); err != nil {
		t.Fatalf("Subnet.WithdrawCancellation returned error: %v", err)
	}
}

// TestSubnetService_List_Empty verifies an empty array response decodes to an empty slice, not an error.
func TestSubnetService_List_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subnet" {
			t.Errorf("expected path '/subnet', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.Subnet.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}
