Subnet Discovery
---

## How to build

Run `go build`, which should generate the binary in the root folder

## How to use

It's simple. Just fire the following command

```
./subnet-planner -h
  -s string
    	Provide the subnet to query, ex: 172.0.0.0/16
```

Supply the subnet information as the argument, which should then provide you with an output as below

```
./subnet-planner -s 10.0.10.254/27
2024/02/29 10:11:29 Subnet length: 32
IP ADDRESS		 STATUS
----------		 ----------
10.0.10.224		 Available
10.0.10.225		 Unvailable
10.0.10.226		 Unvailable
10.0.10.227		 Unvailable
10.0.10.228		 Unvailable
10.0.10.229		 Unvailable
10.0.10.230		 Unvailable
10.0.10.231		 Unvailable
10.0.10.232		 Unvailable
10.0.10.233		 Unvailable
10.0.10.234		 Unvailable
10.0.10.235		 Unvailable
10.0.10.236		 Unvailable
10.0.10.237		 Unvailable
10.0.10.238		 Unvailable
10.0.10.239		 Unvailable
10.0.10.240		 Unvailable
10.0.10.241		 Unvailable
10.0.10.242		 Unvailable
10.0.10.243		 Available
10.0.10.244		 Available
10.0.10.245		 Available
10.0.10.246		 Available
10.0.10.247		 Available
10.0.10.248		 Available
10.0.10.249		 Available
10.0.10.250		 Available
10.0.10.251		 Available
10.0.10.252		 Unvailable
10.0.10.253		 Unvailable
10.0.10.254		 Unvailable
10.0.10.255		 Available
```

Where `Unvailable` means that the IP is in use, and `Available` means the IP is available for you to use