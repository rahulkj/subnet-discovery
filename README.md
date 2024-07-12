Subnet Discovery
---

## How to build

Run `go build`, which should generate the binary in the root folder

## How to use

It's simple. Just fire the following command

```
./subnet-discovery -h
DESCRIPTION:
--> Utility to scan for IP's used and unused on the network
--> If ICMP isn't available, then the utility will not work
--> The results may vary based on your network latecy, so use retry flag to ensure you get a reliable response

Usage of ./subnet-discovery:
  -c int
    	Number of pings to send (default 3)
  -i string
    	Provide the subnet to query, ex: 172.0.0.0/16
  -n int
    	Provide the number of IP's you would like to process in batches, ex: 4,6,8,16,32. Default is 32 (default 32)
  -r int
    	Provide the retry count to check if the IP is up. Default is 3 (default 3)
```

Supply the subnet information as the argument, which should then provide you with an output as below

```
./subnet-discovery -i 10.100.1.0/28
2024/07/12 16:29:58 Subnet length: 16
IP Ping Status >>> 100% |██████████████████████████████████████████████████████████████████████████████████████████████████████| (16/16, 5 it/s)

------------------Unavailable IPs------------------------
IP ADDRESS		 STATUS
----------		 ----------
10.100.1.10		 Unavailable

------------------Available IPs------------------------
IP ADDRESS		 STATUS
----------		 ----------
10.100.1.0		 Available
10.100.1.1		 Available
10.100.1.11		 Available
10.100.1.12		 Available
10.100.1.13		 Available
10.100.1.14		 Available
10.100.1.15		 Available
10.100.1.2		 Available
10.100.1.3		 Available
10.100.1.4		 Available
10.100.1.5		 Available
10.100.1.6		 Available
10.100.1.7		 Available
10.100.1.8		 Available
10.100.1.9		 Available

Summary of the scan
------------------------------------------
USED IPS: 1
UNUSED IPS: 15
```

Where `Unavailable` means that the IP is in use, and `Available` means the IP is available for you to use