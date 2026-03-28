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
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Subnet in CIDR notation or single IP address | *(required)* |
| `-c` | Number of pings to send per IP | `3` |
| `-p` | Max concurrent ping workers. Increase for faster scans | `64` |
| `-r` | Retry count if ping fails (minimum 3) | `3` |
| `-o` | Output format: `table` or `json` | `table` |

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

## Output

- **Unavailable** — the IP responded to ping (in use)
- **Available** — the IP did not respond (available for use)

Results are sorted numerically by IP address in both table and JSON output.

## Notes

- Requires ICMP to be available on the network
- Results may vary based on network latency; use `-r` to increase reliability
- The `-p` flag controls how many ping workers run simultaneously. Higher values speed up scans but use more system resources
