package main

import (
	"flag"
	"fmt"
	"os"
	"subnet-discovery/cmd"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION: \n")
		fmt.Fprintf(os.Stderr, "--> Utility to scan for IP's used and unused on the network \n")
		fmt.Fprintf(os.Stderr, "--> If ICMP isn't available, then the utility will not work \n")
		fmt.Fprintf(os.Stderr, "--> The results may vary based on your network latecy, so use retry flag to ensure you get a reliable response \n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	ipAddr := flag.String("i", "", "Provide the subnet to query, ex: 172.0.0.0/16")
	count := flag.Int("c", 3, "Number of pings to send")
	batchSize := flag.Int("n", 32, "Provide the number of IP's you would like to process in batches, ex: 4,6,8,16,32. Default is 32")
	retryCount := flag.Int("r", 3, "Provide the retry count to check if the IP is up. Default is 3")
	flag.Parse()

	if len(*ipAddr) == 0 || *batchSize < 32 || *retryCount < 3 || *count > 3 {
		flag.PrintDefaults()
		return
	}

	userInput := cmd.UserInput{PingCount: *count, IPAddr: *ipAddr, BatchSize: *batchSize, RetryCount: *retryCount}

	cmd.ValidateIFInputIsReachableOrNot(userInput)
}
