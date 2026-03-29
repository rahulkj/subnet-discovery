# Subnet Discovery

Utility to scan for IPs used and unused on a network using ICMP ping.

## How to build

```
go build -o subnet-discovery .
```

## Usage

```
./subnet-discovery -h

DESCRIPTION:
  Utility to scan for IPs used and unused on the network
  If ICMP isn't available, then the utility will not work
  Results may vary based on network latency; use -r flag for reliability

  Use -s flag to find available subnets of a given size within a parent network
  Example: -i 172.16.0.0/23 -s 26  (finds available /26 subnets)

Usage of ./subnet-discovery:
  -c int
        Number of pings to send (default 3)
  -i string
        Subnet to query, ex: 172.0.0.0/16
  -o string
        Output format: 'table' or 'json' (default "table")
  -p int
        Max concurrent ping workers (higher = faster) (default 64)
  -r int
        Retry count for IP availability check (min 3) (default 3)
  -s int
        Find available subnets of this prefix length (e.g., 26 for /26)
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Subnet in CIDR notation or single IP address | *(required)* |
| `-c` | Number of pings to send per IP | `3` |
| `-p` | Max concurrent ping workers. Increase for faster scans | `64` |
| `-r` | Retry count if ping fails (minimum 3) | `3` |
| `-o` | Output format: `table` or `json` | `table` |
| `-s` | Find available subnets of this prefix length (e.g., `26` for `/26`) | *disabled* |

## Examples

### Scan a small subnet

```
./subnet-discovery -i 10.100.1.0/28
```

```
Subnet length: 16
IP Ping Status >>> 100% |██████████████████████████████████████████████████████████████████████████████████████████████████████| (16/16, 5 it/s)

****** Unavailable IPs ******
IP ADDRESS		 STATUS
----------		 ----------
10.100.1.10		 Unavailable

****** Available IPs ******
IP ADDRESS		 STATUS
----------		 ----------
10.100.1.0		 Available
10.100.1.1		 Available
10.100.1.2		 Available
10.100.1.3		 Available
10.100.1.4		 Available
10.100.1.5		 Available
10.100.1.6		 Available
10.100.1.7		 Available
10.100.1.8		 Available
10.100.1.9		 Available
10.100.1.11		 Available
10.100.1.12		 Available
10.100.1.13		 Available
10.100.1.14		 Available
10.100.1.15		 Available

****** Summary of the subnet scan ******
TOTAL IPS: 	16
AVAILABLE IPS: 	15
UNAVAILABLE IPS: 1
```

### Fast scan with high concurrency

For larger subnets, increase `-p` to scan more IPs in parallel:

```
./subnet-discovery -i 10.0.0.0/24 -p 256
```

### JSON output

```
./subnet-discovery -i 10.100.1.0/28 -o json
```

```json
{
  "total_ips": 16,
  "available_ips": 15,
  "unavailable_ips": 1,
  "used_ips": [
    {"ip": "10.100.1.10", "pingable": true}
  ],
  "unused_ips": [
    {"ip": "10.100.1.0", "pingable": false},
    {"ip": "10.100.1.1", "pingable": false}
  ]
}
```

### Single IP check

```
./subnet-discovery -i 10.0.0.1
```

### Find available subnets

Scan a parent network and find available subnets of a given size. The tool pings every IP in the parent block, then evaluates candidate subnets at the requested prefix length, reporting only those with zero used IPs.

```
./subnet-discovery -i 172.16.0.0/24 -s 26
```

```
****** Subnet Recommendations for /26 in 172.16.0.0/24 ******
  PARENT NETWORK:      172.16.0.0/24
  REQUESTED PREFIX:    /26 (64 IPs per subnet)
  TOTAL IPS SCANNED:   256
  USED IPS:            39
  AVAILABLE IPS:       217

****** Available Subnets ******
  SUBNET                 IP RANGE                            TOTAL IPS   USABLE IPS
  ------                 --------                            ---------   ----------
  172.16.0.64/26         172.16.0.64 - 172.16.0.127          64          62
  172.16.0.128/26        172.16.0.128 - 172.16.0.191         64          62
  172.16.0.192/26        172.16.0.192 - 172.16.0.255         64          62

****** Used Subnets (conflicts detected) ******
  172.16.0.0/26

****** Summary ******
  TOTAL CANDIDATE SUBNETS:   4
  AVAILABLE SUBNETS:         3
  USED SUBNETS:              1
```

#### Find available subnets (JSON)

```
./subnet-discovery -i 172.16.0.0/24 -s 26 -o json
```

```json
{
  "parent_network": "172.16.0.0/24",
  "requested_prefix": 26,
  "parent_total_ips": 256,
  "parent_used_ips": 39,
  "parent_unused_ips": 217,
  "total_candidates": 4,
  "available_subnets": [
    {
      "subnet": "172.16.0.64/26",
      "ip_range": "172.16.0.64 - 172.16.0.127",
      "total_ips": 64,
      "usable_ips": 62
    },
    {
      "subnet": "172.16.0.128/26",
      "ip_range": "172.16.0.128 - 172.16.0.191",
      "total_ips": 64,
      "usable_ips": 62
    },
    {
      "subnet": "172.16.0.192/26",
      "ip_range": "172.16.0.192 - 172.16.0.255",
      "total_ips": 64,
      "usable_ips": 62
    }
  ],
  "used_subnets": [
    "172.16.0.0/26"
  ]
}
```

## Output

### IP scan

- **Unavailable** — the IP responded to ping (in use)
- **Available** — the IP did not respond (available for use)

Results are sorted numerically by IP address in both table and JSON output.

### Subnet recommendation

- **Available Subnets** — candidate subnets where no IPs responded to ping; safe to allocate
- **Used Subnets** — candidate subnets containing at least one used IP; not safe to allocate
- **Usable IPs** — total IPs minus network and broadcast addresses

## Notes

- Requires ICMP to be available on the network
- Results may vary based on network latency; use `-r` to increase reliability
- The `-p` flag controls how many ping workers run simultaneously. Higher values speed up scans but use more system resources
- The `-s` flag requires a CIDR input (e.g., `172.16.0.0/24`), not a single IP
- The requested prefix must be larger than the parent prefix (e.g., `/24` parent requires `-s 25` through `-s 30`)
