package hrobot

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestTrafficSizeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantBytes uint64
		wantUnlim bool
		wantRaw   string
		wantErr   bool
	}{
		{
			name:      "unlimited string",
			input:     `"unlimited"`,
			wantUnlim: true,
			wantBytes: 0,
			wantRaw:   "unlimited",
			wantErr:   false,
		},
		{
			name:      "numeric string",
			input:     `"5497558138880"`,
			wantUnlim: false,
			wantBytes: 5497558138880,
			wantRaw:   "5497558138880",
			wantErr:   false,
		},
		{
			name:      "zero",
			input:     `"0"`,
			wantUnlim: false,
			wantBytes: 0,
			wantRaw:   "0",
			wantErr:   false,
		},
		{
			name:      "numeric value",
			input:     `1099511627776`,
			wantUnlim: false,
			wantBytes: 1099511627776,
			wantRaw:   "",
			wantErr:   false,
		},
		{
			name:      "5 TB",
			input:     `"5 TB"`,
			wantUnlim: false,
			wantBytes: 5497558138880, // 5 * 1024^4
			wantRaw:   "5 TB",
			wantErr:   false,
		},
		{
			name:      "2 TB",
			input:     `"2 TB"`,
			wantUnlim: false,
			wantBytes: 2199023255552, // 2 * 1024^4
			wantRaw:   "2 TB",
			wantErr:   false,
		},
		{
			name:      "20 TB",
			input:     `"20 TB"`,
			wantUnlim: false,
			wantBytes: 21990232555520, // 20 * 1024^4
			wantRaw:   "20 TB",
			wantErr:   false,
		},
		{
			name:      "oversized human-readable string",
			input:     `"16777216 TB"`,
			wantUnlim: false,
			wantBytes: ^uint64(0),
			wantRaw:   "16777216 TB",
			wantErr:   false,
		},
		{
			name:      "null",
			input:     `null`,
			wantUnlim: false,
			wantBytes: 0,
			wantRaw:   "",
			wantErr:   false,
		},
		{
			name:      "garbage string",
			input:     `"garbage"`,
			wantUnlim: false,
			wantBytes: 0,
			wantRaw:   "garbage",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts TrafficSize
			err := json.Unmarshal([]byte(tt.input), &ts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ts.Unlimited != tt.wantUnlim {
				t.Errorf("Unlimited = %v, want %v", ts.Unlimited, tt.wantUnlim)
			}
			if ts.Bytes != tt.wantBytes {
				t.Errorf("Bytes = %d, want %d", ts.Bytes, tt.wantBytes)
			}
			if ts.Raw != tt.wantRaw {
				t.Errorf("Raw = %q, want %q", ts.Raw, tt.wantRaw)
			}
		})
	}
}

func TestUnmarshalJSONResetsReusedReceiver(t *testing.T) {
	var ts TrafficSize
	if err := json.Unmarshal([]byte(`"5 TB"`), &ts); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if err := json.Unmarshal([]byte(`null`), &ts); err != nil {
		t.Fatalf("null decode: %v", err)
	}
	if ts.Unlimited || ts.Bytes != 0 || ts.Raw != "" {
		t.Errorf("TrafficSize not reset on null: %+v", ts)
	}

	var bt BerlinTime
	if err := json.Unmarshal([]byte(`"2024-01-01 12:00:00"`), &bt); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if err := json.Unmarshal([]byte(`null`), &bt); err != nil {
		t.Fatalf("null decode: %v", err)
	}
	if !bt.IsZero() {
		t.Errorf("BerlinTime not reset on null: %v", bt)
	}

	var sf StringFloat
	if err := json.Unmarshal([]byte(`"1.5"`), &sf); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if err := json.Unmarshal([]byte(`null`), &sf); err != nil {
		t.Fatalf("null decode: %v", err)
	}
	if sf != 0 {
		t.Errorf("StringFloat not reset on null: %v", sf)
	}

	var id FlexibleID
	if err := json.Unmarshal([]byte(`283693`), &id); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if err := json.Unmarshal([]byte(`null`), &id); err != nil {
		t.Fatalf("null decode: %v", err)
	}
	if id != "" {
		t.Errorf("FlexibleID not reset on null: %q", id)
	}
}

func TestTrafficSizeString(t *testing.T) {
	tests := []struct {
		name string
		ts   TrafficSize
		want string
	}{
		{
			name: "unlimited",
			ts:   TrafficSize{Unlimited: true},
			want: "unlimited",
		},
		{
			name: "5 TB",
			ts:   TrafficSize{Bytes: 5497558138880},
			want: "5.0 TB",
		},
		{
			name: "1 TB",
			ts:   TrafficSize{Bytes: 1099511627776},
			want: "1.0 TB",
		},
		{
			name: "500 GB",
			ts:   TrafficSize{Bytes: 536870912000},
			want: "500.0 GB",
		},
		{
			name: "zero",
			ts:   TrafficSize{Bytes: 0},
			want: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTrafficSizeMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		ts   TrafficSize
		want string
	}{
		{"raw preserved", TrafficSize{Raw: "5 TB", Bytes: 5497558138880}, `"5 TB"`},
		{"unlimited without raw", TrafficSize{Unlimited: true}, `"unlimited"`},
		{"bytes without raw", TrafficSize{Bytes: 1099511627776}, `1099511627776`},
		{"zero", TrafficSize{}, `0`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.ts)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestTrafficSizeRoundTrip verifies that decoding an API value and re-encoding
// it yields the original wire representation.
func TestTrafficSizeRoundTrip(t *testing.T) {
	for _, wire := range []string{`"unlimited"`, `"5 TB"`, `"5497558138880"`, `1099511627776`} {
		t.Run(wire, func(t *testing.T) {
			var ts TrafficSize
			if err := json.Unmarshal([]byte(wire), &ts); err != nil {
				t.Fatalf("decode: %v", err)
			}
			got, err := json.Marshal(ts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(got) != wire {
				t.Errorf("round-trip = %s, want %s", got, wire)
			}
		})
	}
}

func TestStringFloatMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		sf   StringFloat
		want string
	}{
		{"typical", StringFloat(123.4567), `"123.4567"`},
		{"zero", StringFloat(0), `"0.0000"`},
		{"rounds to four places", StringFloat(1.23456789), `"1.2346"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.sf)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestStringFloatRoundTrip verifies a string-encoded float survives a
// decode/encode cycle at the API's fixed precision.
func TestStringFloatRoundTrip(t *testing.T) {
	var sf StringFloat
	if err := json.Unmarshal([]byte(`"123.4567"`), &sf); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(got) != `"123.4567"` {
		t.Errorf("round-trip = %s, want %q", got, `"123.4567"`)
	}
}

func TestBerlinTimeMarshalJSON(t *testing.T) {
	// A Berlin-local timestamp encodes back to the API's wire format.
	bt := BerlinTime{Time: time.Date(2025, 10, 24, 14, 30, 0, 0, berlinLocation)}
	got, err := json.Marshal(bt)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(got) != `"2025-10-24 14:30:00"` {
		t.Errorf("MarshalJSON() = %s, want %q", got, `"2025-10-24 14:30:00"`)
	}

	// A UTC instant is converted to Berlin local time before formatting.
	// This asserts the +2 (CEST) October offset, so it only holds when the
	// real IANA zone is available; if the tz database is missing,
	// berlinLocation falls back to a fixed CET (+1) zone (see types.go init)
	// and the offset would differ. Skip that leg rather than fail spuriously.
	if _, err := time.LoadLocation("Europe/Berlin"); err != nil {
		t.Skipf("Europe/Berlin tz data unavailable, skipping UTC-conversion assertion: %v", err)
	}
	utc := BerlinTime{Time: time.Date(2025, 10, 24, 12, 30, 0, 0, time.UTC)}
	got, err = json.Marshal(utc)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(got) != `"2025-10-24 14:30:00"` { // UTC+2 in October (CEST)
		t.Errorf("MarshalJSON() = %s, want %q", got, `"2025-10-24 14:30:00"`)
	}
}

// TestBerlinTimeRoundTrip verifies a wire timestamp decodes and re-encodes to
// the same string.
func TestBerlinTimeRoundTrip(t *testing.T) {
	const wire = `"2025-10-24 14:30:00"`
	var bt BerlinTime
	if err := json.Unmarshal([]byte(wire), &bt); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := json.Marshal(bt)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(got) != wire {
		t.Errorf("round-trip = %s, want %s", got, wire)
	}
}

func TestBerlinTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // Expected time in Berlin
		wantErr bool
	}{
		{
			name:    "valid datetime",
			input:   `"2025-10-24 14:30:00"`,
			want:    "2025-10-24 14:30:00 +0200 CEST",
			wantErr: false,
		},
		{
			name:    "date only",
			input:   `"2025-10-24"`,
			want:    "2025-10-24 00:00:00 +0200 CEST",
			wantErr: false,
		},
		{
			name:    "null",
			input:   `null`,
			want:    "0001-01-01 00:00:00 +0000 UTC",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bt BerlinTime
			err := json.Unmarshal([]byte(tt.input), &bt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if bt.Format("2006-01-02 15:04:05 -0700 MST") != tt.want {
				t.Errorf("Time = %s, want %s", bt.Format("2006-01-02 15:04:05 -0700 MST"), tt.want)
			}
		})
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []PortRange
		wantErr bool
	}{
		{
			name:    "single port",
			input:   "22",
			want:    []PortRange{{Start: 22, End: 22}},
			wantErr: false,
		},
		{
			name:    "port range",
			input:   "80-443",
			want:    []PortRange{{Start: 80, End: 443}},
			wantErr: false,
		},
		{
			name:    "multiple ports",
			input:   "80,443",
			want:    []PortRange{{Start: 80, End: 80}, {Start: 443, End: 443}},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid port",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid range",
			input:   "80-abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "inverted range",
			input:   "443-80",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "out of range port",
			input:   "65536",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "out of range in range",
			input:   "65535-65536",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePortRange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPortRangeString(t *testing.T) {
	tests := []struct {
		name string
		pr   PortRange
		want string
	}{
		{
			name: "single port",
			pr:   PortRange{Start: 22, End: 22},
			want: "22",
		},
		{
			name: "port range",
			pr:   PortRange{Start: 80, End: 443},
			want: "80-443",
		},
		{
			name: "zero",
			pr:   PortRange{Start: 0, End: 0},
			want: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestServerIDString(t *testing.T) {
	tests := []struct {
		name string
		id   ServerID
		want string
	}{
		{
			name: "regular id",
			id:   ServerID(123456),
			want: "123456",
		},
		{
			name: "zero",
			id:   ServerID(0),
			want: "0",
		},
		{
			name: "large id",
			id:   ServerID(9999999),
			want: "9999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestServerUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"server_ip": "1.2.3.4",
		"server_number": 123456,
		"server_name": "test-server",
		"product": "AX41",
		"dc": "FSN1-DC14",
		"traffic": "unlimited",
		"status": "ready",
		"cancelled": false,
		"paid_until": "2025-12-31",
		"ip": ["1.2.3.4", "5.6.7.8"],
		"subnet": [
			{"ip": "2a01:4f8:1::", "mask": "64"}
		]
	}`

	var server Server
	err := json.Unmarshal([]byte(jsonData), &server)
	if err != nil {
		t.Fatalf("failed to unmarshal server: %v", err)
	}

	if server.ServerNumber != 123456 {
		t.Errorf("ServerNumber = %d, want 123456", server.ServerNumber)
	}
	if server.ServerName != "test-server" {
		t.Errorf("ServerName = %s, want test-server", server.ServerName)
	}
	if server.Product != "AX41" {
		t.Errorf("Product = %s, want AX41", server.Product)
	}
	if server.DC != "FSN1-DC14" {
		t.Errorf("DC = %s, want FSN1-DC14", server.DC)
	}
	if !server.Traffic.Unlimited {
		t.Error("Traffic should be unlimited")
	}
	if server.Status != ServerStatusReady {
		t.Errorf("Status = %s, want ready", server.Status)
	}
	if server.Cancelled {
		t.Error("Cancelled should be false")
	}
	if server.PaidUntil != "2025-12-31" {
		t.Errorf("PaidUntil = %s, want 2025-12-31", server.PaidUntil)
	}
	if len(server.IP) != 2 {
		t.Errorf("len(IP) = %d, want 2", len(server.IP))
	}
	if server.ServerIP.String() != "1.2.3.4" {
		t.Errorf("ServerIP = %s, want 1.2.3.4", server.ServerIP.String())
	}
	if len(server.Subnet) != 1 {
		t.Errorf("len(Subnet) = %d, want 1", len(server.Subnet))
	}
	expectedSubnet := net.ParseIP("2a01:4f8:1::")
	if !server.Subnet[0].IP.Equal(expectedSubnet) {
		t.Errorf("Subnet[0].IP = %s, want 2a01:4f8:1::", server.Subnet[0].IP)
	}
	if server.Subnet[0].Mask != "64" {
		t.Errorf("Subnet[0].Mask = %s, want 64", server.Subnet[0].Mask)
	}
}

func TestIPAddressUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"ip": "1.2.3.4",
		"server_ip": "5.6.7.8",
		"server_number": 123456,
		"locked": false,
		"separate_mac": "00:11:22:33:44:55",
		"traffic_warnings": true,
		"traffic_hourly": 1000,
		"traffic_daily": 50000,
		"traffic_monthly": 1500000
	}`

	var ipAddr IPAddress
	err := json.Unmarshal([]byte(jsonData), &ipAddr)
	if err != nil {
		t.Fatalf("failed to unmarshal IP address: %v", err)
	}

	if ipAddr.IP.String() != "1.2.3.4" {
		t.Errorf("IP = %s, want 1.2.3.4", ipAddr.IP.String())
	}
	if ipAddr.ServerIP.String() != "5.6.7.8" {
		t.Errorf("ServerIP = %s, want 5.6.7.8", ipAddr.ServerIP.String())
	}
	if ipAddr.ServerNumber != 123456 {
		t.Errorf("ServerNumber = %d, want 123456", ipAddr.ServerNumber)
	}
	if ipAddr.Locked {
		t.Error("Locked should be false")
	}
	if ipAddr.SeparateMac != "00:11:22:33:44:55" {
		t.Errorf("SeparateMac = %s, want 00:11:22:33:44:55", ipAddr.SeparateMac)
	}
	if !ipAddr.TrafficWarnings {
		t.Error("TrafficWarnings should be true")
	}
	if ipAddr.TrafficHourly != 1000 {
		t.Errorf("TrafficHourly = %d, want 1000", ipAddr.TrafficHourly)
	}
	if ipAddr.TrafficDaily != 50000 {
		t.Errorf("TrafficDaily = %d, want 50000", ipAddr.TrafficDaily)
	}
	if ipAddr.TrafficMonthly != 1500000 {
		t.Errorf("TrafficMonthly = %d, want 1500000", ipAddr.TrafficMonthly)
	}
}

func TestBerlinTimeLocation(t *testing.T) {
	// Test that BerlinTime uses Europe/Berlin location
	bt := BerlinTime{Time: time.Date(2025, 10, 24, 12, 0, 0, 0, time.UTC)}

	// Convert to Berlin time
	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load Berlin location: %v", err)
	}

	berlinTime := bt.In(berlinLoc)

	// In October, Berlin is CEST (UTC+2)
	_, offset := berlinTime.Zone()
	expectedOffset := 2 * 3600 // 2 hours in seconds
	if offset != expectedOffset {
		t.Errorf("Berlin offset = %d, want %d (UTC+2)", offset, expectedOffset)
	}
}

func TestStringFloatUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "string float",
			input:   `"123.4567"`,
			want:    123.4567,
			wantErr: false,
		},
		{
			name:    "numeric float",
			input:   `123.4567`,
			want:    123.4567,
			wantErr: false,
		},
		{
			name:    "string zero",
			input:   `"0"`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "numeric zero",
			input:   `0`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "null",
			input:   `null`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   `""`,
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf StringFloat
			err := json.Unmarshal([]byte(tt.input), &sf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if float64(sf) != tt.want {
				t.Errorf("StringFloat = %v, want %v", float64(sf), tt.want)
			}
		})
	}
}
