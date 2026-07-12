package hrobot

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Error represents all possible errors from the hrobot library.
type Error struct {
	Kind ErrorKind
	// Code is the Hetzner-side error code, set for API errors. Empty for
	// non-API errors (network, parse, auth, policy).
	Code ErrorCode
	// Status is the HTTP-style status code associated with this error.
	// Zero means none was attached (e.g. local errors before any HTTP call).
	Status  int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	prefix := string(e.Kind)
	if e.Code != "" {
		prefix = fmt.Sprintf("%s[%s]", e.Kind, e.Code)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", prefix, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", prefix, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ErrorKind categorizes the error type.
type ErrorKind string

// ErrorKind values categorize errors returned by this library.
const (
	ErrKindAPI        ErrorKind = "API"
	ErrKindNetwork    ErrorKind = "Network"
	ErrKindParse      ErrorKind = "Parse"
	ErrKindAuth       ErrorKind = "Auth"
	ErrKindPolicy     ErrorKind = "Policy"
	ErrKindValidation ErrorKind = "Validation"
)

// NewAPIError creates a new API error.
func NewAPIError(code ErrorCode, message string) *Error {
	return &Error{
		Kind:    ErrKindAPI,
		Code:    code,
		Message: message,
	}
}

// newAPIErrorWithStatus creates an API error including the HTTP status code.
func newAPIErrorWithStatus(code ErrorCode, message string, status int) *Error {
	return &Error{
		Kind:    ErrKindAPI,
		Code:    code,
		Status:  status,
		Message: message,
	}
}

// NewNetworkError creates a new network error.
func NewNetworkError(message string, cause error) *Error {
	return &Error{
		Kind:    ErrKindNetwork,
		Message: message,
		Cause:   cause,
	}
}

// NewParseError creates a new parse error.
func NewParseError(message string, cause error) *Error {
	return &Error{
		Kind:    ErrKindParse,
		Message: message,
		Cause:   cause,
	}
}

// NewAuthError creates a new authentication error.
func NewAuthError(message string) *Error {
	return &Error{
		Kind:    ErrKindAuth,
		Message: message,
	}
}

// NewPolicyError returns an error indicating that the named operation is
// implemented in this client but intentionally not invoked: purchasing or
// destructively cancelling Hetzner resources is reserved for the Robot UI to
// avoid automation accidents. The returned error carries HTTP status 451 to
// signal that the block is non-technical.
func NewPolicyError(operation string) *Error {
	return &Error{
		Kind:    ErrKindPolicy,
		Code:    ErrDisallowedByClientPolicy,
		Status:  451,
		Message: fmt.Sprintf("%s is disallowed by client policy; perform this action via the Hetzner Robot UI", operation),
	}
}

// NewValidationError returns an error for a request that this client rejected
// locally, before sending it to the API, because it violates a documented
// Hetzner constraint. Status mirrors the status the API would have returned
// for the same condition, so callers can treat local and remote rejections
// uniformly.
func NewValidationError(code ErrorCode, message string, status int) *Error {
	return &Error{
		Kind:    ErrKindValidation,
		Code:    code,
		Status:  status,
		Message: message,
	}
}

// ErrorCode represents specific API error codes from Hetzner.
type ErrorCode string

// ErrorCode values mirror the `code` field returned by the Hetzner Robot
// API on error responses.
const (
	ErrUnauthorized            ErrorCode = "UNAUTHORIZED"
	ErrInvalidInput            ErrorCode = "INVALID_INPUT"
	ErrInvalidInputServerIP    ErrorCode = "INVALID_INPUT_SERVER_IP"
	ErrInvalidInputIPAddress   ErrorCode = "INVALID_INPUT_IP_ADDRESS"
	ErrServerNotFound          ErrorCode = "SERVER_NOT_FOUND"
	ErrIPNotFound              ErrorCode = "IP_NOT_FOUND"
	ErrIPLocked                ErrorCode = "IP_LOCKED"
	ErrInsufficientPermissions ErrorCode = "INSUFFICIENT_PERMISSIONS"
	ErrRateLimitExceeded       ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrMaintenanceMode         ErrorCode = "MAINTENANCE_MODE"

	ErrFirewallInProcess         ErrorCode = "FIREWALL_IN_PROCESS"
	ErrFirewallAlreadyActive     ErrorCode = "FIREWALL_ALREADY_ACTIVE"
	ErrFirewallAlreadyDisabled   ErrorCode = "FIREWALL_ALREADY_DISABLED"
	ErrFirewallConfigInvalid     ErrorCode = "FIREWALL_CONFIG_INVALID"
	ErrFirewallRuleLimitExceeded ErrorCode = "FIREWALL_RULE_LIMIT_EXCEEDED"

	ErrBootConfigNotFound  ErrorCode = "BOOT_CONFIG_NOT_FOUND"
	ErrBootAlreadyActive   ErrorCode = "BOOT_ALREADY_ACTIVE"
	ErrRescueNotActive     ErrorCode = "RESCUE_NOT_ACTIVE"
	ErrRescueAlreadyActive ErrorCode = "RESCUE_ALREADY_ACTIVE"

	ErrResetNotAvailable ErrorCode = "RESET_NOT_AVAILABLE"
	ErrResetManualActive ErrorCode = "RESET_MANUAL_ACTIVE"

	ErrVNCDisabled     ErrorCode = "VNC_DISABLED"
	ErrVNCNotAvailable ErrorCode = "VNC_NOT_AVAILABLE"

	ErrReverseDNSNotFound ErrorCode = "RDNS_NOT_FOUND"
	ErrReverseDNSInvalid  ErrorCode = "RDNS_INVALID"

	// ErrDisallowedByClientPolicy is returned by stub methods that this
	// client refuses to call against the live API; see NewPolicyError.
	ErrDisallowedByClientPolicy ErrorCode = "DISALLOWED_BY_CLIENT_POLICY"

	// ErrUnknown is set when the error response from the API could not be
	// parsed or did not include a recognised code.
	ErrUnknown ErrorCode = "UNKNOWN"
)

// APIErrorResponse represents the error response from Hetzner API.
type APIErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

// APIErrorDetail contains the error details.
type APIErrorDetail struct {
	Status  int       `json:"status"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// UnmarshalJSON handles both known and unknown error codes.
func (d *APIErrorDetail) UnmarshalJSON(data []byte) error {
	type Alias APIErrorDetail
	aux := &struct {
		Code json.RawMessage `json:"code"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var codeStr string
	if err := json.Unmarshal(aux.Code, &codeStr); err == nil {
		d.Code = ErrorCode(codeStr)
		return nil
	}

	d.Code = ErrUnknown
	return nil
}

// IsAPIError reports whether err is, or wraps, an API error with the given code.
func IsAPIError(err error, code ErrorCode) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == ErrKindAPI && e.Code == code
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	return IsAPIError(err, ErrRateLimitExceeded)
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return IsAPIError(err, ErrServerNotFound) || IsAPIError(err, ErrIPNotFound)
}

// IsFirewallInProcessError checks if the error is a firewall in process error.
func IsFirewallInProcessError(err error) bool {
	return IsAPIError(err, ErrFirewallInProcess)
}

// IsUnauthorizedError reports whether err indicates an unauthorized request,
// whether returned by the Hetzner API or rejected locally by this client for
// missing/invalid credentials (see validateCredentials). It matches on the
// error code regardless of kind, so both origins are caught.
func IsUnauthorizedError(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == ErrUnauthorized
}

// IsFirewallRuleLimitExceededError reports whether err indicates the inbound
// firewall rule limit was exceeded, whether the request was rejected locally
// by this client (see Firewall.ValidateRules) or by the Hetzner API. It
// matches on the error code regardless of kind, so both origins are caught.
func IsFirewallRuleLimitExceededError(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == ErrFirewallRuleLimitExceeded
}

// IsInvalidInputError checks if the error is an invalid input error.
func IsInvalidInputError(err error) bool {
	return IsAPIError(err, ErrInvalidInput)
}

// IsPolicyError reports whether err was returned because the operation is
// disallowed by client-side policy (and so never reached the Hetzner API).
func IsPolicyError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrKindPolicy
}
