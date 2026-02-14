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

| Service | Status | Description |
|---------|--------|-------------|
| Server | Implemented | List, get, rename, cancel servers |
| Firewall | Implemented | Full firewall rule management |
| Reset | Implemented | Server reset operations |
| Boot | Implemented | Rescue, Linux, VNC, Windows boot config |
| IP | Implemented | IP management and traffic warnings |
| SSH Key | Implemented | SSH key CRUD operations |
| RDNS | Implemented | Reverse DNS management |
| vSwitch | Implemented | Virtual switch management |
| Failover | Implemented | Failover IP management |
| Traffic | Implemented | Traffic query |
| Auction | Implemented | Auction server listing |
| Ordering | Implemented | Product/market ordering and transactions |
| WOL | Implemented | Wake-on-LAN |
| Subnet | Stub | Subnet management (not yet implemented) |
| Storage Box | Stub | Storage box management (not yet implemented) |

## Authentication

Get your API credentials from <https://robot.hetzner.com/preferences/index>.

```bash
export HROBOT_USERNAME='#ws+XXXXXXX'
export HROBOT_PASSWORD='YYYYYY'
```

## API Documentation

Full Hetzner Robot API documentation: <https://robot.hetzner.com/doc/webservice/en.html>

## License

Mozilla Public License Version 2.0 - see [LICENSE](./LICENSE) for details.
