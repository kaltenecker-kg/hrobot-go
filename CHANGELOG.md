# Changelog

## 1.0.0 (Unreleased)

BREAKING CHANGES:

- **restructure**: Project restructured from Terraform provider (`terraform-provider-hrobot`) to pure Go client
  library (`hrobot-go`), following the `hcloud-go` pattern
- **restructure**: Module path changed to `github.com/kaltenecker-kg/hrobot-go`
- **restructure**: CLI tool and Terraform provider removed; this package is now a library only
- **restructure**: API client code moved from `pkg/hrobot/` to repository root
- **client**: `WithDebug(bool)` replaced by `WithLogger(*slog.Logger)`; debug output is now structured `slog`
  events at DEBUG level instead of `fmt.Printf` traces
- **errors**: API error code is now a typed `Code` field on `*Error` (no longer embedded in the message string).
  Callers using `IsAPIError` are unaffected; callers comparing the message text must read `err.Code` instead

FEATURES:

- **subnet**: Add stub SubnetService for subnet management API
- **storagebox**: Add stub StorageBoxService for storage box API
- **boot**: Add GetLastLinux, GetWindows, ActivateWindows, DeactivateWindows methods
- **ordering**: Add ListMarketProducts, GetMarketProduct, ListTransactions, GetTransaction, ListAddonProducts methods
- **client**: Track rate limits — parses `RateLimit-Limit/Remaining/Reset` response headers and exposes
  `Client.LastRateLimit()`
- **client**: Retry on `429 Too Many Requests` honoring the `Retry-After` header
- **errors**: `*Error` carries the HTTP status from the response in its `Status` field

IMPROVEMENTS:

- **client**: Cap `401 Unauthorized` retries at one extra attempt (was three), so invalid credentials no longer loop
- **errors**: `IsAPIError` and friends use `errors.As` and unwrap through wrapped errors
- **firewall**: Deduplicate rule-encoding logic across `Update`, `CreateTemplate`, and `UpdateTemplate`;
  `Update` now sends literal brackets in `rules[...]` keys (the previous path round-tripped through `url.Values`
  and would have URL-encoded them)
- **deps**: Bump minimum Go to 1.26

## 0.2.0 (2024-11-07)

BREAKING CHANGES:

- **firewall**: Renamed `source_ip` to `source_ips` (now requires list syntax)
- **firewall**: Renamed `destination_ip` to `destination_ips` (now requires list syntax)
- **firewall**: Changed `filter_ipv6` default from `false` to `true` for better security

FEATURES:

- **firewall**: Add automatic IP array expansion - rules with multiple source/destination IPs are now automatically
  expanded into individual firewall rules
- **firewall**: Add validation to ensure expanded rules don't exceed Hetzner's 10-rule limit per direction
- **firewall**: Automatically append `/32` CIDR notation to IPv4 addresses without it
- **server**: Fix import to properly populate `public_net` block, resolving output evaluation errors

IMPROVEMENTS:

- **docs**: Update all examples to demonstrate array syntax for firewall IPs
- **docs**: Clarify that IP attributes are lists, not single values

## 0.1.2

Rename `hrobot_failover` to `hrobot_failover_ip`.

## 0.1.1

Remove the endpoint configuration from the provider to make it more simple.

## 0.1.0

Initial release as Terraform provider with CLI tool.
