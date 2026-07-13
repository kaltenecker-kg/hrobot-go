package hrobot

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IPVersion represents the IP protocol version.
type IPVersion string

// IP protocol versions.
const (
	IPv4 IPVersion = "ipv4"
	IPv6 IPVersion = "ipv6"
)

// Action represents a firewall rule action.
type Action string

// Firewall rule actions.
const (
	ActionAccept  Action = "accept"
	ActionDiscard Action = "discard"
)

// Protocol represents network protocol.
type Protocol string

// Network protocols accepted in firewall rules.
const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolUDP  Protocol = "udp"
	ProtocolICMP Protocol = "icmp"
	ProtocolESP  Protocol = "esp"
	ProtocolGRE  Protocol = "gre"
	ProtocolIPIP Protocol = "ipip"
	ProtocolAH   Protocol = "ah"
)

// ServerID represents a server identifier.
type ServerID int

func (s ServerID) String() string {
	return strconv.Itoa(int(s))
}

// IPAddress represents an IP address with additional metadata.
//
// GET /ip (list) omits Gateway/Mask/Broadcast; GET /ip/{ip} and
// POST /ip/{ip} (single-address responses) include them. All three are
// therefore optional here so the same type can decode either shape.
type IPAddress struct {
	IP              net.IP `json:"ip"`
	Gateway         net.IP `json:"gateway,omitempty"`
	Mask            int    `json:"mask,omitempty"`
	Broadcast       net.IP `json:"broadcast,omitempty"`
	ServerIP        net.IP `json:"server_ip"`
	ServerNumber    int    `json:"server_number"`
	Locked          bool   `json:"locked"`
	SeparateMac     string `json:"separate_mac,omitempty"`
	TrafficWarnings bool   `json:"traffic_warnings"`
	TrafficHourly   int    `json:"traffic_hourly"`
	TrafficDaily    int    `json:"traffic_daily"`
	TrafficMonthly  int    `json:"traffic_monthly"`
}

// ServerStatus represents the status of a server.
type ServerStatus string

// Server lifecycle states.
const (
	ServerStatusReady     ServerStatus = "ready"
	ServerStatusInProcess ServerStatus = "in process"
	ServerStatusCancelled ServerStatus = "cancelled"
)

// ResetType represents different reset types.
type ResetType string

// Reset operation types.
const (
	ResetTypeSoftware  ResetType = "sw"
	ResetTypeHardware  ResetType = "hw"
	ResetTypePower     ResetType = "power"
	ResetTypePowerLong ResetType = "power_long"
	ResetTypeManual    ResetType = "man"
)

// TrafficSize represents traffic with support for "unlimited".
type TrafficSize struct {
	Unlimited bool
	Bytes     uint64
	Raw       string
}

// UnmarshalJSON handles "unlimited" string, human-readable strings like "5 TB", and numeric values.
func (t *TrafficSize) UnmarshalJSON(data []byte) error {
	// Reset so reused receivers do not retain values from a prior decode.
	*t = TrafficSize{}
	// Handle null
	if string(data) == "null" {
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		t.Raw = str // Always preserve the original wire value

		// Handle "unlimited"
		if str == "unlimited" {
			t.Unlimited = true
			return nil
		}

		// Try parsing as human-readable format: number + unit (case-insensitive)
		// Pattern: ^\s*([0-9]+(?:\.[0-9]+)?)\s*(B|KB|MB|GB|TB)\s*$
		pattern := regexp.MustCompile(`(?i)^\s*([0-9]+(?:\.[0-9]+)?)\s*(B|KB|MB|GB|TB)\s*$`)
		matches := pattern.FindStringSubmatch(str)
		if len(matches) == 3 {
			numStr := matches[1]
			unit := strings.ToUpper(matches[2])

			// Parse the numeric part
			num, err := strconv.ParseFloat(numStr, 64)
			if err == nil {
				// Calculate multiplier based on unit
				multiplier := uint64(1)
				switch unit {
				case "B":
					multiplier = 1
				case "KB":
					multiplier = 1024
				case "MB":
					multiplier = 1024 * 1024
				case "GB":
					multiplier = 1024 * 1024 * 1024
				case "TB":
					multiplier = 1024 * 1024 * 1024 * 1024
				}

				bytes := num * float64(multiplier)
				if bytes >= float64(math.MaxUint64) {
					t.Bytes = math.MaxUint64
				} else {
					t.Bytes = uint64(bytes)
				}
				return nil
			}
		}

		// Try parsing as pure-digit string (legacy behavior)
		bytes, err := strconv.ParseUint(str, 10, 64)
		if err == nil {
			t.Bytes = bytes
			return nil
		}

		// Unknown string - don't fail, just preserve Raw and leave Bytes = 0
		return nil
	}

	// Try as number
	var num uint64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	t.Bytes = num
	return nil
}

// MarshalJSON encodes TrafficSize. If Raw is set, emit it as a JSON string;
// otherwise encode as "unlimited" or as a number of bytes.
func (t TrafficSize) MarshalJSON() ([]byte, error) {
	if t.Raw != "" {
		return json.Marshal(t.Raw)
	}
	if t.Unlimited {
		return json.Marshal("unlimited")
	}
	return json.Marshal(t.Bytes)
}

// String returns the Raw value if set, otherwise "unlimited" or a human-readable byte count.
func (t TrafficSize) String() string {
	if t.Raw != "" {
		return t.Raw
	}
	if t.Unlimited {
		return "unlimited"
	}
	return formatBytes(t.Bytes)
}

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// BerlinTime represents a timestamp in Europe/Berlin timezone.
type BerlinTime struct {
	time.Time
}

var berlinLocation *time.Location

func init() {
	var err error
	berlinLocation, err = time.LoadLocation("Europe/Berlin")
	if err != nil {
		// Fallback to UTC+1
		berlinLocation = time.FixedZone("CET", 3600)
	}
}

// UnmarshalJSON parses timestamp and converts to Berlin time.
// Treats JSON null as the zero time.Time value.
func (bt *BerlinTime) UnmarshalJSON(data []byte) error {
	// Reset so reused receivers do not retain values from a prior decode.
	*bt = BerlinTime{}
	// Handle null
	if string(data) == "null" {
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Parse various timestamp formats from Hetzner API
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.ParseInLocation(format, str, berlinLocation)
		if err == nil {
			bt.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse timestamp: %s", str)
}

// MarshalJSON formats the timestamp in Europe/Berlin local time using
// the wire format expected by the Hetzner Robot API.
func (bt BerlinTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(bt.In(berlinLocation).Format("2006-01-02 15:04:05"))
}

// FlexibleID decodes a JSON string or number into a string.
type FlexibleID string

// UnmarshalJSON handles both JSON string and JSON number encodings of an ID.
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Reset so reused receivers do not retain values from a prior decode.
	*f = ""
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = FlexibleID(n.String())
	return nil
}

// StringFloat represents a float that is encoded as a string in JSON.
type StringFloat float64

// UnmarshalJSON handles string-encoded floats and JSON null.
// Treats JSON null (and empty string) as the zero value.
func (sf *StringFloat) UnmarshalJSON(data []byte) error {
	// Reset so reused receivers do not retain values from a prior decode.
	*sf = 0
	// Handle null
	if string(data) == "null" {
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		// Try as number directly
		var f float64
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		*sf = StringFloat(f)
		return nil
	}

	// Handle empty string as zero value
	if str == "" {
		return nil
	}

	// Parse string as float
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("invalid float string: %s", str)
	}
	*sf = StringFloat(f)
	return nil
}

// MarshalJSON encodes the value as a fixed-precision string, matching the
// wire format used by the Hetzner Robot API.
func (sf StringFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%.4f", float64(sf)))
}

// Float64 returns the underlying float64 value.
func (sf StringFloat) Float64() float64 {
	return float64(sf)
}

// PriceDetail holds one net/gross price pair; monthly and (where offered) hourly.
type PriceDetail struct {
	Net         StringFloat `json:"net"`
	Gross       StringFloat `json:"gross"`
	HourlyNet   StringFloat `json:"hourly_net"`
	HourlyGross StringFloat `json:"hourly_gross"`
}

// AddonPrice is one location's pricing for an orderable addon.
type AddonPrice struct {
	Location   string      `json:"location"`
	Price      PriceDetail `json:"price"`
	PriceSetup PriceDetail `json:"price_setup"`
}

// PortRange represents a port or range of ports.
type PortRange struct {
	Start uint16
	End   uint16
}

// ParsePortRange parses port specifications like "80", "80-443", "80,443".
func ParsePortRange(s string) ([]PortRange, error) {
	if s == "" {
		return nil, nil
	}

	// Handle comma-separated ports
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		ranges := make([]PortRange, 0, len(parts))
		for _, part := range parts {
			rs, err := ParsePortRange(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			ranges = append(ranges, rs...)
		}
		return ranges, nil
	}

	// Handle range
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		start, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port start: %s", parts[0])
		}
		end, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port end: %s", parts[1])
		}
		if start > end {
			return nil, fmt.Errorf("inverted port range: %s-%s (start > end)", parts[0], parts[1])
		}
		return []PortRange{{Start: uint16(start), End: uint16(end)}}, nil
	}

	// Single port
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", s)
	}
	return []PortRange{{Start: uint16(port), End: uint16(port)}}, nil
}

func (p PortRange) String() string {
	if p.Start == p.End {
		return strconv.Itoa(int(p.Start))
	}
	return fmt.Sprintf("%d-%d", p.Start, p.End)
}

// Server represents a Hetzner dedicated server.
type Server struct {
	ServerIP         net.IP       `json:"server_ip"`
	ServerNumber     int          `json:"server_number"`
	ServerName       string       `json:"server_name"`
	Product          string       `json:"product"`
	DC               string       `json:"dc"`
	Traffic          TrafficSize  `json:"traffic"`
	Status           ServerStatus `json:"status"`
	Cancelled        bool         `json:"cancelled"`
	PaidUntil        string       `json:"paid_until"`
	IP               []net.IP     `json:"ip"`
	IPv6Net          string       `json:"server_ipv6_net,omitempty"`
	Subnet           []Subnet     `json:"subnet"`
	Reset            bool         `json:"reset,omitempty"`
	Rescue           bool         `json:"rescue,omitempty"`
	VNC              bool         `json:"vnc,omitempty"`
	Windows          bool         `json:"windows,omitempty"`
	WOL              bool         `json:"wol,omitempty"`
	HotSwap          bool         `json:"hot_swap,omitempty"`
	LinkedStorageBox *int         `json:"linked_storagebox,omitempty"`
}

// Subnet represents a network subnet.
type Subnet struct {
	IP   net.IP `json:"ip"`
	Mask string `json:"mask"`
}

// Reset represents a server reset configuration.
type Reset struct {
	ServerIP        net.IP      `json:"server_ip"`
	ServerIPv6Net   string      `json:"server_ipv6_net,omitempty"`
	ServerNumber    int         `json:"server_number"`
	Type            []ResetType `json:"type"`
	OperatingStatus string      `json:"operating_status,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for Reset to handle both
// string and array formats for the Type field.
// GET /reset/{id} returns an array: {"type": ["hw", "sw", "power"]}
// POST /reset/{id} returns a string: {"type": "hw"}.
func (r *Reset) UnmarshalJSON(data []byte) error {
	type Alias Reset
	aux := &struct {
		Type json.RawMessage `json:"type"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try to unmarshal as array first
	var typeArray []ResetType
	if err := json.Unmarshal(aux.Type, &typeArray); err == nil {
		r.Type = typeArray
		return nil
	}

	// If that fails, try to unmarshal as string
	var typeString ResetType
	if err := json.Unmarshal(aux.Type, &typeString); err == nil {
		r.Type = []ResetType{typeString}
		return nil
	}

	return fmt.Errorf("type field must be either string or array")
}
