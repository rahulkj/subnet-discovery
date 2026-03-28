package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sort"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"github.com/schollz/progressbar/v3"
)

type IPStatus struct {
	IP       string `json:"ip"`
	Pingable bool   `json:"pingable"`
}

func ProcessRequest(userInput UserInput) {
	if checkIfInputIsIP(userInput.IPAddr) {
		ip := userInput.IPAddr
		pingable := checkIfIPIsAvailable(ip, userInput)
		printTableHeader()
		printTable(ip, pingable)
	} else if checkIfInputIsCIDR(userInput.IPAddr) {
		expandCIDRAndLogIPStatus(userInput)
	}
}

func expandCIDRAndLogIPStatus(userInput UserInput) {
	ipAddr, ipNet, err := net.ParseCIDR(userInput.IPAddr)
	if err != nil {
		log.Fatal(err)
	}

	var ips []string
	for ip := ipAddr.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	totalIPs := len(ips)
	log.Printf("Subnet length: %d\n", totalIPs)

	bar := progressbar.Default(int64(totalIPs), "IP Ping Status >>>")

	usedIPs, unusedIPs := processAllIPs(ips, bar, userInput)

	bar.Finish()
	fmt.Println()

	var usedIPArray []string
	var unUsedIPArray []string

	for _, ipStatus := range usedIPs {
		usedIPArray = append(usedIPArray, ipStatus.IP)
	}
	for _, ipStatus := range unusedIPs {
		unUsedIPArray = append(unUsedIPArray, ipStatus.IP)
	}

	sort.Slice(usedIPArray, func(i, j int) bool {
		a, _ := netip.ParseAddr(usedIPArray[i])
		b, _ := netip.ParseAddr(usedIPArray[j])
		return a.Less(b)
	})
	sort.Slice(unUsedIPArray, func(i, j int) bool {
		a, _ := netip.ParseAddr(unUsedIPArray[i])
		b, _ := netip.ParseAddr(unUsedIPArray[j])
		return a.Less(b)
	})

	switch userInput.OutputFormat {
	case "table":
		printTableFormat(usedIPArray, unUsedIPArray, totalIPs)
	case "json":
		sort.Slice(usedIPs, func(i, j int) bool {
			a, _ := netip.ParseAddr(usedIPs[i].IP)
			b, _ := netip.ParseAddr(usedIPs[j].IP)
			return a.Less(b)
		})
		sort.Slice(unusedIPs, func(i, j int) bool {
			a, _ := netip.ParseAddr(unusedIPs[i].IP)
			b, _ := netip.ParseAddr(unusedIPs[j].IP)
			return a.Less(b)
		})
		printJSONFormat(usedIPs, unusedIPs, totalIPs)
	}
}

type IPResultsSummary struct {
	TotalIPs       int        `json:"total_ips"`
	AvailableIPs   int        `json:"available_ips"`
	UnavailableIPs int        `json:"unavailable_ips"`
	UsedIPs        []IPStatus `json:"used_ips"`
	UnusedIPs      []IPStatus `json:"unused_ips"`
}

func printJSONFormat(usedIPs []IPStatus, unusedIPs []IPStatus, totalIPs int) {
	summary := IPResultsSummary{
		TotalIPs:       totalIPs,
		AvailableIPs:   len(unusedIPs),
		UnavailableIPs: len(usedIPs),
		UsedIPs:        usedIPs,
		UnusedIPs:      unusedIPs,
	}

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonData))
}

func printTableFormat(usedIPArray []string, unUsedIPArray []string, totalIPs int) {
	printSeparator("Unavailable IPs")
	printTableHeader()

	for _, ip := range usedIPArray {
		printTable(ip, true)
	}

	printSeparator("Available IPs")
	printTableHeader()

	for _, ip := range unUsedIPArray {
		printTable(ip, false)
	}

	printSeparator("Summary of the subnet scan")
	fmt.Printf("TOTAL IPS: \t%d\n", totalIPs)
	fmt.Printf("AVAILABLE IPS: \t%d\n", len(unUsedIPArray))
	fmt.Printf("UNAVAILABLE IPS: %d\n", len(usedIPArray))
}

func processAllIPs(ips []string, bar *progressbar.ProgressBar, userInput UserInput) ([]IPStatus, []IPStatus) {
	concurrency := userInput.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 32
	}
	if concurrency > len(ips) {
		concurrency = len(ips)
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var usedIPs []IPStatus
	var unusedIPs []IPStatus
	var wg sync.WaitGroup

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			pingable := checkIfIPIsAvailable(ip, userInput)
			mu.Lock()
			if pingable {
				usedIPs = append(usedIPs, IPStatus{Pingable: true, IP: ip})
			} else {
				unusedIPs = append(unusedIPs, IPStatus{Pingable: false, IP: ip})
			}
			mu.Unlock()
			bar.Add(1)
		}(ip)
	}

	wg.Wait()

	return usedIPs, unusedIPs
}

func inc(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func checkIfIPIsAvailable(ip string, userInput UserInput) bool {
	for i := 0; i < userInput.RetryCount; i++ {
		pinger, err := probing.NewPinger(ip)
		if err != nil {
			log.Printf("Error creating pinger for %s: %v\n", ip, err)
			return false
		}

		pinger.Timeout = time.Second
		pinger.Count = userInput.PingCount

		err = pinger.Run()
		if err != nil {
			log.Printf("Error running pinger for %s: %v\n", ip, err)
			return false
		}

		stats := pinger.Statistics()
		if stats.PacketsRecv != 0 && stats.PacketsRecv <= stats.PacketsSent {
			return true
		}
	}

	return false
}

func checkIfInputIsIP(input string) bool {
	_, err := netip.ParseAddr(input)
	return err == nil
}

func checkIfInputIsCIDR(input string) bool {
	_, _, err := net.ParseCIDR(input)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func printSeparator(header string) {
	fmt.Println()
	fmt.Printf("****** %s ******\n", header)
}

func printTableHeader() {
	fmt.Printf("%s\t\t %s\n", "IP ADDRESS", "STATUS")
	fmt.Printf("%s\t\t %s\n", "----------", "----------")
}

func printTable(ip string, pingable bool) {
	if pingable {
		fmt.Printf("%s\t\t %s\n", ip, "Unavailable")
	} else {
		fmt.Printf("%s\t\t %s\n", ip, "Available")
	}
}
