# Changelog

## Unreleased

BREAKING CHANGES:

- **server**: `Server.LinkedStorageBox` is now `*int` (was `string`) to match the API, which returns
  `linked_storagebox` as a nullable integer
- **client**: Remove the deprecated `Client.GetWrappedList` method. It became a pure alias for `Get` once wrapper-key
  auto-detection landed; callers should pass a plain slice to `Get`, which auto-detects the `[{"<key>": ...}, ...]`
  envelope

BUG FIXES:

- **server**: Fix `Server.LinkedStorageBox` decoding — the field was typed `string` while the API returns a nullable
  integer, so any server-detail response with a linked storage box failed to unmarshal. It is now `*int` (`nil` when
  the server has no linked storage box)

IMPROVEMENTS:

- **client**: Bound the automatic-retry sleep by clamping a server's `Retry-After` to a configurable maximum, so a
  hostile or misbehaving endpoint cannot pin a caller's goroutine on an arbitrarily long wait (the per-request HTTP
  timeout does not cover this sleep). Both the delta-seconds and HTTP-date forms are clamped, and an out-of-range
  numeric value is treated as above the cap rather than ignored
- **client**: Add `WithMaxRetryAfter(d)` to override the cap, defaulting to `DefaultMaxRetryAfter`

## 1.2.0 (2026-07-12)

IMPROVEMENTS:

- **client**: Add `WithApplication(name, version)` to identify the program built on hrobot-go in the User-Agent,
  composed as `<name>/<version> hrobot-go/<Version>`, matching hcloud-go's option
- **client**: Add an exported `Version` constant and derive the default `UserAgent` from it (previously a hardcoded
  string literal that had gone stale)
- **client**: Add `WithEndpoint` as an alias for `WithBaseURL`, matching hcloud-go's option name
- **client**: Validate credentials before the first request — reject an empty username/password or a username
  containing a colon (RFC 7617), returning an `UNAUTHORIZED` validation error. `IsUnauthorizedError` now also matches
  this local rejection

## 1.1.0 (2026-07-12)

IMPROVEMENTS:

- **firewall**: Validate the inbound rule limit client-side — `Firewall.Update`, `CreateTemplate`, and
  `UpdateTemplate` now reject configurations with more than `MaxFirewallInputRules` (10) input rules before contacting
  the API, returning a `FIREWALL_RULE_LIMIT_EXCEEDED` validation error. The new exported `Firewall.ValidateRules` lets
  callers run the same check up front, and `IsFirewallRuleLimitExceededError` now matches both locally rejected and
  API-returned errors
- **firewall**: Add `WithMaxFirewallInputRules(n)` to override the client-side inbound rule ceiling, so a stale
  constant does not block valid configurations if Hetzner raises the documented limit before a library release catches
  up (the API stays authoritative)
- **errors**: Add `ErrKindValidation` and `NewValidationError` for requests this client rejects locally before
  sending, carrying the HTTP status the API would have returned

## 1.0.0 (2026-07-12)

BREAKING CHANGES:

- **restructure**: Project restructured from Terraform provider (`terraform-provider-hrobot`) to pure Go client
  library (`hrobot-go`), following the `hcloud-go` pattern
- **restructure**: Module path changed to `github.com/kaltenecker-kg/hrobot-go`
- **restructure**: CLI tool and Terraform provider removed; this package is now a library only
- **restructure**: API client code moved from `pkg/hrobot/` to repository root
- **scope**: Remove `AuctionService` and `OrderingService` (and the `Client.Auction`/`Client.Ordering` fields);
  server auction, product ordering, and addon purchase endpoints are out of scope for this client
- **client**: `WithDebug(bool)` replaced by `WithLogger(*slog.Logger)`; debug output is now structured `slog`
  events at DEBUG level instead of `fmt.Printf` traces
- **errors**: API error code is now a typed `Code` field on `*Error` (no longer embedded in the message string).
  Callers using `IsAPIError` are unaffected; callers comparing the message text must read `err.Code` instead
- **firewall**: Drop exported `FirewallTemplateWrapper` (use `*FirewallTemplate` directly)
- **wol**: Drop exported `WOLWrapper` (use `*WOLResponse` directly)
- **rdns**: Drop exported `RDNSListItem` (`List` returns `[]RDNS` directly)

FEATURES:

- **subnet**: Add stub SubnetService for subnet management API
- **storagebox**: Add stub StorageBoxService for storage box API
- **boot**: Add GetLastLinux, GetWindows, ActivateWindows, DeactivateWindows methods
- **client**: Track rate limits — parses `RateLimit-Limit/Remaining/Reset` response headers and exposes
  `Client.LastRateLimit()`
- **client**: Retry on `429 Too Many Requests` honoring the `Retry-After` header
- **errors**: `*Error` carries the HTTP status from the response in its `Status` field

IMPROVEMENTS:

- **client**: Cap `401 Unauthorized` retries at one extra attempt (was three), so invalid credentials no longer loop
- **client**: Replace the 17-key wrapper-key registry in `unwrapResponse` with a heuristic auto-unwrap; new
  endpoints no longer require a registry edit
- **boot**: `RescueConfig`, `LinuxConfig`, `VNCConfig`, and `WindowsConfig` gain typed `Active*` / `Available*`
  accessors so callers can extract the active scalar or the option list without type-asserting on `any`
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
