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
| ip            | verified   | See "ip tag fixes" below.                                              |
| subnet        | verified   | Gist lacked `/subnet/{net-ip}/mac` and `/subnet/{net-ip}/cancellation`; added. See "subnet tag fixes" below. |
| reset         | verified   | No spec changes needed; see "reset tag fixes" below.                   |
| boot          | verified   | Gist lacked linux/vnc/windows boot paths; added, plus a schema bug fix. See "boot tag fixes" below. |
| firewall      | verified   | Gist lacked `firewall/template/{id}`; added. See "firewall tag fixes" below. |
| vswitch       | verified   | See "vswitch tag fixes" below.                                        |
| rdns          | unverified |                                                                        |
| failover      | unverified |                                                                        |
| wol           | verified   | No spec changes needed; see "wol tag fixes" below.                     |
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

Verified against the doc's `Firewall` section in full, including the paths
the vendored gist was missing: `GET|POST|DELETE /firewall/{server-id}`,
`GET|POST /firewall/template`, and `GET|POST|DELETE
/firewall/template/{template-id}`. Fixes made to the spec:

- `components.schemas.FirewallRule.properties.ip_version` was a non-nullable
  enum (`ipv4`/`ipv6`), but the doc's own `GET /firewall/{server-id}` example
  response sends `"ip_version": null` for rules that omit it ("Omitted rule
  fields will have the value 'null' and will act like a wildcard" — doc
  text). Added `nullable: true` to match.
- Added the `/firewall/template/{template-id}` path (`GET`/`POST`/`DELETE`),
  a `TemplateId` path parameter, and reused
  `FirewallTemplateDetailed`/`InvalidInput`/`Error` for its responses, per
  the doc's `GET|POST|DELETE /firewall/template/{template-id}` sections.
  `DELETE` documents "No output", so its `200` response has no content
  schema.
- `internal/spectest.isFirewallRulesPath` only recognized 2-segment firewall
  paths (`/firewall/{server-id}`, `/firewall/template`), so POST/DELETE
  bodies to the new 3-segment `/firewall/template/{template-id}` path —
  which can also carry the `rules[direction][index][field]` bracket-key
  grammar — would have fallen through to schema-based form validation that
  cannot express that grammar. Extended it to also recognize
  `/firewall/template/{id}`.

Also fixed in test fixtures (not the spec): `GET /firewall/template` was
served as a bare JSON array instead of the doc's `[{"firewall_template":
{...}}, ...]` envelope, and the apply-template response
(`POST /firewall/{server-id}?template_id=...`) was served unenveloped instead
of `{"firewall": {...}}`.

`TestFirewallService_GetTemplate`, `UpdateTemplate`, and `DeleteTemplate` are
now wrapped with `spectest.Handler` (previously unwrapped for lack of a spec
path).

### ip tag fixes

Verified against the doc's `IP` section (`GET /ip`, `GET|POST /ip/{ip}`,
`GET|PUT|DELETE /ip/{ip}/mac`, `GET|POST|DELETE /ip/{ip}/cancellation`; the
cancellation POST endpoint is policy-stubbed and out of scope, matching
`SubnetService.Cancel`/`IPService.CancelIP`'s existing precedent). No
`spec/robot.yaml` changes were needed for this tag — `IPAddress`,
`IPAddressDetailed`, `MACAddress`, and `IPCancellation` already matched the
doc.

A real code bug was found and fixed: `IPService.Get` and
`IPService.SetTrafficWarnings` both decode into the same `IPAddress` struct
used by `IPService.List`, but the doc's `GET /ip/{ip}` and `POST /ip/{ip}`
responses include `gateway`, `mask`, and `broadcast` fields that `GET /ip`
(list) does not — `types.IPAddress` was missing all three, so they were
silently dropped on every single-address response. Added them as optional
fields (present on `Get`/`SetTrafficWarnings` responses, absent on `List`).
Additionally, `IPService.SetTrafficWarnings` posted to `POST /ip/{ip}` but
discarded the response body entirely (`return i.client.Post(ctx, path, data,
nil)`), even though the doc's Output table for that endpoint documents a
full updated IP address resource. Changed its signature from `error` to
`(*IPAddress, error)` to return it, updating the (only) callers in
`ip_test.go` accordingly.

All `ip_test.go` suites are now wrapped with `spectest.Handler`.
`TestIPService_WithdrawIPCancellation` previously asserted on a bare `200`
with no body; DELETE /ip/{ip}/cancellation's doc example returns a full
`{"cancellation": {...}}` envelope, so the fixture was corrected to match.

### subnet tag fixes

Verified against the doc's `Subnet` section in full, including the paths the
vendored gist was missing: `GET|PUT|DELETE /subnet/{net-ip}/mac` and
`GET|POST|DELETE /subnet/{net-ip}/cancellation` (previously only
`GET /subnet` and `GET|POST /subnet/{net-ip}` were present). Fixes made to
the spec:

- Added the `/subnet/{net-ip}/mac` and `/subnet/{net-ip}/cancellation`
  paths, reusing the existing `MACAddress` schema (already generic enough to
  cover the subnet shape's `mask`/`possible_mac` fields) and adding a new
  `SubnetCancellation` schema (`IPCancellation` doesn't have subnet's `mask`
  field).
- `components.schemas.Subnet.properties.subnet.properties.server_ip` was a
  non-nullable `ipv4` string, but the doc's own `GET /subnet` example
  response includes an unassigned subnet with `"server_ip": null`
  (`internal/spectest` caught this immediately once `GET /subnet` was
  wrapped — the fixture is doc-verbatim, so the mismatch was in the spec).
  Added `nullable: true` to match.

Also fixed in test fixtures (not the spec):
`TestSubnetService_DeleteMAC`/`WithdrawCancellation` previously asserted on
a bare `200` with no body; the doc's examples for both `DELETE` endpoints
return the full `MACAddress`/`SubnetCancellation` envelope (mac reverts to
default; cancellation is revoked), so the fixtures were corrected to match.
All `subnet_test.go` suites are now wrapped with `spectest.Handler`.

### reset tag fixes

Verified against the doc's `Reset` section (`GET /reset`,
`GET|POST /reset/{server-number}`; the deprecated `{server-ip}` aliases are
out of scope). No `spec/robot.yaml` changes were needed —
`ResetOptions`/`ResetOptionsDetailed`/`ResetResult` already matched the doc,
including `POST /reset/{server-number}`'s response omitting `server_number`
(the doc's own POST example omits it; `ResetResult.server_number` isn't
`required` in the spec, so this doesn't fail validation). All
`reset_test.go` suites are now wrapped with `spectest.Handler`.

### wol tag fixes

Verified against the doc's `Wake on LAN` section
(`GET|POST /wol/{server-number}`; the deprecated `{server-ip}` alias is out
of scope). No `spec/robot.yaml` changes were needed — `WakeOnLAN` already
matched the doc, including `POST /wol/{server-number}` taking no request
body. Both `wol_test.go` suites are now wrapped with `spectest.Handler`.

### boot tag fixes

Verified against the doc's `Boot configuration` section in full, including
the paths the vendored gist was missing:
`GET|POST|DELETE /boot/{server-number}/linux`,
`GET /boot/{server-number}/linux/last`,
`GET|POST|DELETE /boot/{server-number}/vnc`, and
`GET|POST|DELETE /boot/{server-number}/windows` (previously only
`GET /boot/{server-number}`, `GET|POST|DELETE /boot/{server-number}/rescue`,
and `GET /boot/{server-number}/rescue/last` were present — the last of these
was also missing from the *paths* section despite existing implicitly; it's
now an explicit path too). Fixes made to the spec:

- **Known defect (from the issue plan), fixed:**
  `components.schemas.RescueSystem` and `RescueSystemActivated` declared
  `authorized_key`/`host_key` as arrays of bare strings. Every populated
  example of this shape elsewhere in the doc (the `SSH keys` section's `key`
  objects, and the analogous Storage Box SSH-access arrays) shows each
  array entry wrapped as `{"key": {name, fingerprint, type, size}}`, not a
  bare fingerprint string — the boot doc's own examples only ever show empty
  arrays (`[]`), so the object shape wasn't directly visible there, but nothing
  in the doc supports the bare-string shape either. Added a new
  `components.schemas.BootKey` schema modeling the `{"key": {...}}` wrapper
  and referenced it from both `authorized_key`/`host_key` array items. This
  also applies to the newly-added `LinuxSystem`/`LinuxSystemActivated`
  schemas (rescue and Linux installs share the same authorized_key/host_key
  shape per the doc).
- Added `LinuxSystem`, `LinuxSystemActivated`, `VNCSystem`,
  `VNCSystemActivated`, `WindowsSystem`, and `WindowsSystemActivated`
  schemas (the gist had no equivalent of `RescueSystem`/
  `RescueSystemActivated` for the other three boot targets), and pointed
  `BootConfiguration`'s `linux`/`vnc`/`windows` properties at them (they were
  previously untyped `type: object` placeholders).
- `WindowsSystem`'s deprecated `dist`/`arch`/`lang` fields are nullable per
  the doc's `GET /boot/{server-number}/windows` example
  (`"dist": null, "lang": null`).

`TestBootService_ActivateRescue` is now wrapped with `spectest.Handler` — the
schema mismatch documented in its comment (spec expected bare strings, the
API returns `{"key": {...}}` objects) is resolved by the `BootKey` schema
fix above. All other `boot_test.go` suites are now wrapped too; several
`DELETE` fixtures (`DeactivateRescue`/`Linux`/`VNC`/`Windows`) previously
asserted on a bare `200` with no body, but the doc's `DELETE` examples for
these endpoints return the full deactivated-state resource, so those
fixtures were corrected to match.
