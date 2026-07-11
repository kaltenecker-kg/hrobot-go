# Hetzner Robot API — vendored OpenAPI spec

`robot.yaml` is a vendored copy of a community-maintained OpenAPI 3.0.3
description of the Hetzner Robot API.

- **Source:** [gist by xavierleune](https://gist.github.com/xavierleune/660084e31f291eef2442c39f4c7f97e8)
- **Vendored:** 2026-07-11

Hetzner ships no official machine-readable spec for the Robot API — only the
HTML doc at <https://robot.hetzner.com/doc/webservice/en.html> and a PHP
client. This spec is a third-party reconstruction of that doc.

## Why vendor a spec instead of generating the client from it

See [issue #16](https://github.com/kaltenecker-kg/hrobot-go/issues/16) for
the full decision record. In short: this client is hand-written (not
generated) because OpenAPI's form-urlencoded serialization rules cannot
express the firewall endpoint's nested bracket-key form bodies
(`rules[input][0][name]`), and generated Go would be less ergonomic than the
hand-written surface regardless (same precedent as hcloud-go). Instead, the
spec is enforced as a **test contract**: `internal/spectest` validates every
mocked request and response in the test suite against it, so a fixture that
drifts from the documented shape fails the test that uses it.

## Verification status

Each tag is verified section-by-section against the canonical HTML doc
(`.claude/CLAUDE.md` names this doc, not the web, as the reference to use).
Fixing a tag means: correcting wrong or missing fields, adding response
envelopes/nullability the doc requires, and fixing schema bugs found along
the way.

| Tag           | Status     | Notes                                                                 |
| ------------- | ---------- | ---------------------------------------------------------------------- |
| server        | verified   | See "server tag fixes" below.                                          |
| ip            | unverified |                                                                        |
| subnet        | unverified |                                                                        |
| reset         | unverified |                                                                        |
| boot          | unverified | Gist lacks linux/vnc/windows boot paths per the issue plan.            |
| firewall      | partially verified | `/firewall/{server-id}` GET/POST/DELETE and `/firewall/template` GET/POST checked against the doc and wrapped with `spectest.Handler`; see "firewall tag fixes" below. Gist still lacks `firewall/template/{id}`, so those tests remain unwrapped; rule body validated out-of-band (see `internal/spectest`, known exception). |
| vswitch       | verified   | See "vswitch tag fixes" below.                                        |
| rdns          | verified   | See "rdns tag fixes" below.                                            |
| failover      | verified   | Clean — see "failover tag fixes" below.                                |
| wol           | unverified |                                                                        |
| traffic       | verified   | Spec had no request/response modeling at all; rewritten. See "traffic tag fixes" below. |
| key           | verified   | See "key tag fixes" below.                                             |
| storagebox    | verified   | Authored the missing password/snapshot/snapshotplan/subaccount paths. See "storagebox tag fixes" below. |

Untracked/out of scope: `/order/*` (server ordering/auction/cancellation) is
policy-stubbed in this client (see project scope notes) and is not part of
this verification effort.

### server tag fixes

Verified against the doc's `Server` section (`GET /server`,
`GET /server/{server-number}`, `POST /server/{server-number}`; the
cancellation endpoints were left unverified per the issue's scope for this
commit). Fixes made to the spec:

- `components.schemas.Error` was corrupted — it modeled server fields
  (`status` as a `ready`/`in process` enum, `cancelled`, `paid_until`, `ip`,
  `subnet`, with `server_ip`/`server_number`/... marked `required`) instead
  of the documented `{"error": {"status", "code", "message"}}` shape. This
  blocked every error response in the spec, not just `server`'s. Fixed to
  match the doc's "Errors" section.
- `components.schemas.ServerCancellation.properties.cancellation.properties.cancellation_reason`
  used `oneOf: [..., {type: "null"}]`, which is invalid in OpenAPI 3.0.x
  (`type: "null"` is a 3.1-ism). Replaced with `nullable: true` alongside the
  existing `oneOf: [array, string]`, matching the doc's
  `cancellation_reason (Array|String)`, nullable per the POST example.

Confirmed already correct (no change needed): list/single response envelopes
(`[{"server": {...}}]` / `{"server": {...}}`), `traffic` as a human-readable
string (`"5 TB"`, `"unlimited"`), `server_ipv6_net` presence, and `subnet`
nullability.

### vswitch tag fixes

Verified against the doc's `vSwitch` section (`GET /vswitch`,
`GET|POST /vswitch/{vswitch-id}`, `DELETE /vswitch/{vswitch-id}`,
`POST|DELETE /vswitch/{vswitch-id}/server`). No `spec/robot.yaml` schema
changes were needed — `VSwitchDetailed`/`VSwitchBasic` already modeled the
doc's top-level unwrapped response shape correctly; the bug was in the test
fixtures (`vswitch_test.go` previously wrapped responses in a spurious
`{"vswitch": {...}}` envelope the API never sends). `internal/client.go`'s
generic single-key envelope unwrapping masked the fixture bug, since it
leaves top-level objects with an `id` key untouched either way.

New known exception (analogous to the firewall bracket-key one): the Robot
API encodes `POST|DELETE /vswitch/{vswitch-id}/server` request bodies as
repeated `server[]=<ip>` form keys, which OpenAPI 3 form-urlencoded
serialization rules cannot express as a schema. `internal/spectest/vswitch.go`
validates this by hand (`validateVSwitchServerForm`), following the same
pattern as `internal/spectest/firewall.go`.

A real production bug was found and fixed independently of the spec:
`vswitch.go`'s `VSwitchServerStatusProcessing` constant was `"processing"`;
the doc documents the value as `"in process"` (matching the sibling
`FirewallStatusInProcess` constant already in the codebase).

### firewall tag fixes

Verified against the doc's `Firewall` section for the paths the vendored gist
covers: `GET|POST|DELETE /firewall/{server-id}` and
`GET|POST /firewall/template`. Fix made to the spec:

- `components.schemas.FirewallRule.properties.ip_version` was a non-nullable
  enum (`ipv4`/`ipv6`), but the doc's own `GET /firewall/{server-id}` example
  response sends `"ip_version": null` for rules that omit it ("Omitted rule
  fields will have the value 'null' and will act like a wildcard" — doc
  text). Added `nullable: true` to match.

Also fixed in test fixtures (not the spec): `GET /firewall/template` was
served as a bare JSON array instead of the doc's `[{"firewall_template":
{...}}, ...]` envelope, and the apply-template response
(`POST /firewall/{server-id}?template_id=...`) was served unenveloped instead
of `{"firewall": {...}}`.

`GET /firewall/template/{template-id}`, `POST /firewall/template/{template-id}`,
and `DELETE /firewall/template/{template-id}` remain unverified against the
spec (not wrapped with `spectest.Handler`) because the vendored gist has no
path for `firewall/template/{id}`; their fixtures were still corrected
against the doc text directly.

### rdns tag fixes

Verified against the doc's `Reverse DNS` section (`GET /rdns`,
`GET|PUT|POST|DELETE /rdns/{ip}`). Fix made to the spec:

- The shared `components.parameters.IPAddress` path parameter (used by both
  the `ip` and `rdns` tags) restricted `ip` to `format: ipv4`. The `rdns`
  tag's `{ip}` path parameter accepts IPv6 addresses too — the doc's own
  wording and `rdns_test.go`'s existing IPv6 test cases (e.g.
  `2001:db8::1`) confirm this — so a new `components.parameters.RDNSIPAddress`
  parameter (no `format: ipv4` restriction) was added and wired into all
  four `/rdns/{ip}` operations instead of loosening the shared `IPAddress`
  parameter, which the `ip` tag's IPv4-only endpoints still correctly use.

Confirmed already correct (no change needed): the `{"rdns": {...}}` response
envelope on every operation, the `[{"rdns": {...}}]` list envelope on
`GET /rdns`, and the doc's `server_ip` query filter on `GET /rdns` (that one
correctly stays IPv4-only, matching the `ip` tag convention for "server main
IP" fields).

### failover tag fixes

Verified against the doc's `Failover` section (`GET /failover`,
`GET|POST|DELETE /failover/{failover-ip}`). No spec changes were needed —
`components.schemas.FailoverIP` already modeled the doc's `{"failover":
{...}}` envelope, `active_server_ip` nullability, and IPv6-capable
`failover-ip` path parameter correctly, and the existing rate limits and
error codes matched the doc exactly.

### traffic tag fixes

Verified against the doc's `Traffic` section (`POST /traffic`). The spec had
essentially no correct modeling of this endpoint — it documented completely
different parameters (`server_ip`, `type`/`from`/`to` as `format: date`) and
no response schema at all, none of which matched the doc's actual `type`
(day/month/year, not a date format), `from`/`to` (free-form strings whose
format depends on `type`, not RFC 3339 dates), or `ip[]`/`subnet[]`/
`single_values` parameters. Rewritten to match:

- Added the `Traffic`/`TrafficStats` schemas modeling the `{"traffic":
  {type, from, to, data}}` envelope, with `data`'s two documented shapes
  (aggregate-per-IP without `single_values`, per-IP-per-interval with it)
  captured via `oneOf`. `TrafficStats` needed `required: [in, out, sum]` +
  `additionalProperties: false` to make the two `oneOf` branches
  structurally distinguishable — otherwise kin-openapi reported "input
  matches more than one oneOf schemas", since an unconstrained object
  schema also loosely accepts the per-interval shape's nested objects.
- New known exception (same pattern as firewall/vswitch bracket-keys): the
  doc's "multiple IPs" and "subnet" examples encode `ip[]=`/`subnet[]=` as
  repeated bracket-key form fields, which OpenAPI 3 form-urlencoded
  serialization cannot express as schema-validated array properties.
  `internal/spectest/traffic.go` validates the key grammar by hand
  (`validateTrafficForm`), wired into `internal/spectest/spectest.go`
  alongside the existing firewall/vswitch exceptions.

**Code bug fixed**: `TrafficService.Get`/`TrafficGetParams` (`traffic.go`)
only ever supported a single `ip` value and had no way to query by subnet at
all, even though the doc documents `ip[]`/`subnet[]` as the primary
multi-value request shape (a single `ip=` is shown as one example among
several). Added `TrafficGetParams.IPs []string` and `.Subnets []string`
(alongside the existing single-value `IP` field, kept for
backwards-compatible callers), encoded as literal `ip[]=`/`subnet[]=` form
keys the same way `VSwitchService.AddServers` encodes `server[]=`.

### key tag fixes

Verified against the doc's `SSH keys` section (`GET|POST /key`,
`GET|POST|DELETE /key/{fingerprint}`). Fix made to the spec:

- `components.schemas.SSHKey.properties.key.properties.created_at` was
  `format: date-time` (RFC 3339), but the doc's examples use
  `"2021-12-31 23:59:59"` (space-separated, no timezone offset), which
  kin-openapi's built-in `date-time` format validator rejects (confirmed by
  running `key_test.go` against the spec before this fix: every test
  failed response validation). Removed `format: date-time`, since
  `key.go`'s `BerlinTime.UnmarshalJSON` already parses this exact
  space-separated format alongside RFC 3339 as a fallback.

Confirmed already correct (no change needed): the `{"key": {...}}` envelope,
list envelope, and all input/output fields for create/rename/delete.

### storagebox tag fixes

The vendored gist only had `GET /storagebox`, `GET /storagebox/{id}`, and
`POST /storagebox/{id}` — every other path `storagebox.go` calls was
missing from the spec. Authored, matching the doc's Input tables and
response examples exactly:

- `POST /storagebox/{id}/password` and
  `POST /storagebox/{id}/subaccount/{username}/password` — both share a new
  unwrapped `StorageBoxPassword` schema (`{"password": "..."}`, no
  `storagebox`/`subaccount` envelope, per the doc's examples).
- `GET|POST /storagebox/{id}/snapshot` — list uses the existing
  `StorageBoxSnapshot` shape; a new `StorageBoxSnapshotCreated` schema
  models the create response, which the doc's example shows returning only
  `name`/`timestamp`/`size` (omitting `filesystem_size`/`automatic`/
  `comment`, unlike the list/GET shape).
- `DELETE|POST /storagebox/{id}/snapshot/{name}` (delete, revert) and
  `POST /storagebox/{id}/snapshot/{name}/comment` — new `SnapshotName` path
  parameter; both documented as "No output"/plain 200.
- `GET|POST /storagebox/{id}/snapshotplan` — new `StorageBoxSnapshotPlan`
  schema. Both GET and POST return a single-element array
  (`[{"snapshotplan": {...}}]`) per the doc's examples, even though POST
  updates one plan; `storagebox.go`'s `decodeSnapshotPlanResponse` already
  unwraps this correctly. (The doc's Output section prose mislabels the
  wrapper as `storagebox (Object)` for this endpoint — trusted the JSON
  example's actual `snapshotplan` key over the prose typo, per doc
  precedence rules.)
- `GET|POST /storagebox/{id}/subaccount` and
  `PUT|DELETE /storagebox/{id}/subaccount/{username}` — new
  `StorageBoxSubAccount`/`StorageBoxSubAccountCreated` schemas and
  `SubAccountUsername` path parameter. `StorageBoxSubAccountCreated` (the
  POST response) additionally carries `password`, matching the doc's create
  example.

No changes were needed to the previously-verified `storagebox`/
`storagebox/{id}` paths or the `StorageBoxBasic`/`StorageBoxDetailed`
schemas.
