package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"subnet-discovery/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	log.Printf("subnet-discovery %s (commit: %s, built: %s)\n", version, commit, date)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION:\n")
		fmt.Fprintf(os.Stderr, "  Utility to scan for IPs used and unused on the network\n")
		fmt.Fprintf(os.Stderr, "  If ICMP isn't available, then the utility will not work\n")
		fmt.Fprintf(os.Stderr, "  Results may vary based on network latency; use -r flag for reliability\n\n")
		fmt.Fprintf(os.Stderr, "  Use -s flag to find available subnets of a given size within a parent network\n")
		fmt.Fprintf(os.Stderr, "  Example: -i 172.16.0.0/23 -s 26  (finds available /26 subnets)\n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	ipAddr := flag.String("i", "", "Subnet to query, ex: 172.0.0.0/16")
	count := flag.Int("c", 3, "Number of pings to send")
	concurrency := flag.Int("p", 64, "Max concurrent ping workers (higher = faster)")
	retryCount := flag.Int("r", 3, "Retry count for IP availability check (min 3)")
	outputFormat := flag.String("o", "table", "Output format: 'table' or 'json'")
	subnetPrefix := flag.Int("s", 0, "Find available subnets of this prefix length (e.g., 26 for /26)")
	flag.Parse()

	if len(*ipAddr) == 0 || *retryCount < 3 || *count < 1 || *concurrency < 1 || (*outputFormat != "table" && *outputFormat != "json") {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *subnetPrefix != 0 && (*subnetPrefix < 1 || *subnetPrefix > 30) {
		fmt.Fprintf(os.Stderr, "Error: subnet prefix must be between 1 and 30\n")
		os.Exit(1)
	}

	userInput := cmd.UserInput{
		PingCount:      *count,
		IPAddr:         *ipAddr,
		MaxConcurrency: *concurrency,
		RetryCount:     *retryCount,
		OutputFormat:   *outputFormat,
		SubnetPrefix:   *subnetPrefix,
	}

	cmd.ProcessRequest(userInput)
}
