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
| rdns          | unverified |                                                                        |
| failover      | unverified |                                                                        |
| wol           | unverified |                                                                        |
| traffic       | unverified |                                                                        |
| key           | unverified |                                                                        |
| storagebox    | unverified | Gist lacks snapshot/subaccount sub-paths per the issue plan.           |

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
