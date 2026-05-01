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
| Auction     | Implemented | Auction server listing                                 |
| Ordering    | Implemented | Read-only; order placement disallowed by client policy |
| WOL         | Implemented | Wake-on-LAN                                            |
| Subnet      | Stub        | Subnet management (not yet implemented)                |
| Storage Box | Stub        | Storage box management (not yet implemented)           |

### Disallowed-by-policy operations

To prevent accidents, this client refuses to invoke endpoints that purchase or
destructively cancel Hetzner resources. The methods are still part of the
public API surface but short-circuit with an `*Error` of `Kind: Policy` and
`Status: 451` before any HTTP request:

- `OrderingService.PlaceMarketOrder`, `PlaceProductOrder`, `PlaceAddonOrder`
- `ServerService.RequestCancellation`
- `SubnetService.Cancel`

Reads (lists, transactions, cancellation status) and recovery operations
(`WithdrawCancellation`) remain fully callable. Use the Hetzner Robot UI to
purchase or cancel.

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
