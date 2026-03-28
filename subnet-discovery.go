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
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	ipAddr := flag.String("i", "", "Subnet to query, ex: 172.0.0.0/16")
	count := flag.Int("c", 3, "Number of pings to send")
	concurrency := flag.Int("p", 64, "Max concurrent ping workers (higher = faster)")
	retryCount := flag.Int("r", 3, "Retry count for IP availability check (min 3)")
	outputFormat := flag.String("o", "table", "Output format: 'table' or 'json'")
	flag.Parse()

	if len(*ipAddr) == 0 || *retryCount < 3 || *count < 1 || *concurrency < 1 || (*outputFormat != "table" && *outputFormat != "json") {
		flag.PrintDefaults()
		os.Exit(1)
	}

	userInput := cmd.UserInput{
		PingCount:      *count,
		IPAddr:         *ipAddr,
		MaxConcurrency: *concurrency,
		RetryCount:     *retryCount,
		OutputFormat:   *outputFormat,
	}

	cmd.ProcessRequest(userInput)
}
