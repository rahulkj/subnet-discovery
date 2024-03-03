package main

import (
	"flag"
	"subnet-discovery/cmd"
)

func main() {
	input := flag.String("s", "", "Provide the subnet to query, ex: 172.0.0.0/16")
	ipsToProcessInABatch := flag.Int("n", 32, "Provide the number of IP's you would like to process in batches, ex: 4,6,8,16,32. Default is 32")
	flag.Parse()

	if len(*input) == 0 {
		flag.PrintDefaults()
		return
	}

	cmd.ValidateIFInputIsReachableOrNot(*input, *ipsToProcessInABatch)
}
