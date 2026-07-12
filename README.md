# hrobot-go

Go client library for the [Hetzner Robot API](https://robot.hetzner.com/doc/webservice/en.html).

Follows the same pattern as [hcloud-go](https://github.com/hetznercloud/hcloud-go).

## Installation

```bash
go get github.com/kaltenecker-kg/hrobot-go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/kaltenecker-kg/hrobot-go"
)

func main() {
    client := hrobot.NewClient(
        os.Getenv("HROBOT_USERNAME"),
        os.Getenv("HROBOT_PASSWORD"),
    )

    servers, err := client.Server.List(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, s := range servers {
        fmt.Printf("Server %d: %s (%s)\n", s.ServerNumber, s.ServerName, s.ServerIP)
    }
}
```

## Observability

### Structured logging

Pass a `*slog.Logger` via `WithLogger` to receive structured DEBUG-level
events for every request, response, and retry. Authorization headers are
never logged.

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

client := hrobot.NewClient(user, pass, hrobot.WithLogger(logger))
```

### Rate limits

The client parses `RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset`
response headers, and retries on `429 Too Many Requests` honoring `Retry-After`.
Inspect the most recent rate-limit state via `Client.LastRateLimit()`:

```go
rl := client.LastRateLimit()
fmt.Printf("%d/%d remaining, resets at %s\n", rl.Remaining, rl.Limit, rl.Reset)
```

## API Coverage

| Service     | Status      | Description                                            |
| ----------- | ----------- | ------------------------------------------------------ |
| Server      | Implemented | List, get, rename; cancellation withdraw allowed       |
| Firewall    | Implemented | Full firewall rule management                          |
| Reset       | Implemented | Server reset operations                                |
| Boot        | Implemented | Rescue, Linux, VNC, Windows boot config                |
| IP          | Implemented | IP management and traffic warnings                     |
| SSH Key     | Implemented | SSH key CRUD operations                                |
| RDNS        | Implemented | Reverse DNS management                                 |
| vSwitch     | Implemented | Virtual switch management                              |
| Failover    | Implemented | Failover IP management                                 |
| Traffic     | Implemented | Traffic query                                          |
| WOL         | Implemented | Wake-on-LAN                                            |
| Subnet      | Implemented | Subnet list/get/update, MAC, cancellation status       |
| Storage Box | Implemented | Box, snapshots, snapshot plan, sub-accounts            |

### Disallowed-by-policy operations

To prevent accidents, this client refuses to invoke endpoints that
destructively cancel Hetzner resources. The methods are still part of the
public API surface but short-circuit with an `*Error` of `Kind: Policy` and
`Status: 451` before any HTTP request:

- `ServerService.RequestCancellation`
- `IPService.CancelIP`
- `SubnetService.Cancel`

Reads (lists, cancellation status) and recovery operations
(`WithdrawCancellation`) remain fully callable. Use the Hetzner Robot UI to
cancel resources.

### Unsupported endpoints

Unlike the disallowed-by-policy methods above, these are not part of the client
surface at all. Server auction, product ordering, and addon purchase endpoints
are out of scope and not implemented. Use the Hetzner Robot UI to browse the
server market or place orders.

## Authentication

Get your API credentials from <https://robot.hetzner.com/preferences/index>.

```bash
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='YYYYYY'
```

## API Documentation

Full Hetzner Robot API documentation: <https://robot.hetzner.com/doc/webservice/en.html>

## Acknowledgements

- Original Go authorship of the hrobot client work by [Onni Hakala](https://github.com/onnimonni).
- The repository was initially bootstrapped from
  [hashicorp/terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework);
  none of that scaffolding code remains in the current source tree.
- API surface modelled after [hetznercloud/hcloud-go](https://github.com/hetznercloud/hcloud-go).

## License

Mozilla Public License 2.0 — see [LICENSE](./LICENSE) for details.
