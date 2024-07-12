package main

import (
	"flag"
	"subnet-discovery/cmd"
)

func main() {
	input := flag.String("s", "", "Provide the subnet to query, ex: 172.0.0.0/16")
	ipsToProcessInABatch := flag.Int("n", 32, "Provide the number of IP's you would like to process in batches, ex: 4,6,8,16,32. Default is 32")
	retry := flag.Int("r", 3, "Provide the retry count to check if the IP is up. Default is 3")
	flag.Parse()

	if len(*input) == 0 || *ipsToProcessInABatch < 32 || *retry < 3 {
		flag.PrintDefaults()
		return
	}

	cmd.ValidateIFInputIsReachableOrNot(*input, *ipsToProcessInABatch, *retry)
}
