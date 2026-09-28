# Security Policy

## Supported versions

Only the latest release on the `v2` major line receives security fixes. Older
tags stay available through the Go module proxy but are not patched.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub's
[private vulnerability reporting](https://github.com/kaltenecker-kg/hrobot-go/security/advisories/new)
for this repository. You should receive an acknowledgement within a few days;
fixes are published as a normal patch release, and the advisory is made public
once the release is out.

## Scope

This module is an HTTP client for the Hetzner Robot API. In scope:

- Credential handling: the client sends HTTP Basic auth and never logs the
  `Authorization` header or the request body.
- Request construction: every caller-supplied path parameter is escaped, and
  form bodies are encoded by the client.
- Response handling: bodies are read with a size cap, `Retry-After` sleeps are
  clamped, and non-2xx responses always surface as errors.
- Policy stubs: the cancellation endpoints are intentionally never invoked.

Vulnerabilities in the Hetzner Robot API itself should be reported to Hetzner,
not here.

## Supply chain

- Dependencies are verified against the Go checksum database and scanned with
  the current `govulncheck` release in CI and in `make vulncheck`.
- GitHub Actions are pinned by commit SHA and the repository requires it.
  Dependabot keeps action pins, Go modules, and npm dev tooling current, with a
  release cooldown so brand-new versions are not pulled in on the day they
  appear.
- Releases are Git tags created from `main` (`task release`); the module is
  served through `proxy.golang.org` with `sum.golang.org` verification.
