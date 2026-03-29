package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"github.com/schollz/progressbar/v3"
)

type IPStatus struct {
	IP       string `json:"ip"`
	Pingable bool   `json:"pingable"`
}

func ProcessRequest(userInput UserInput) {
	if userInput.SubnetPrefix > 0 {
		if !checkIfInputIsCIDR(userInput.IPAddr) {
			log.Fatal("Subnet recommendation requires a CIDR input (e.g., 172.16.0.0/23)")
		}
		FindAvailableSubnets(userInput)
		return
	}

	if checkIfInputIsIP(userInput.IPAddr) {
		ip := userInput.IPAddr
		pingable, _ := checkIfIPIsAvailable(ip, userInput)
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

type AvailableSubnet struct {
	Subnet    string `json:"subnet"`
	IPRange   string `json:"ip_range"`
	TotalIPs  int    `json:"total_ips"`
	UsableIPs int    `json:"usable_ips"`
}

type SubnetRecommendationResult struct {
	ParentNetwork    string            `json:"parent_network"`
	RequestedPrefix  int               `json:"requested_prefix"`
	ParentTotalIPs   int               `json:"parent_total_ips"`
	ParentUsedIPs    int               `json:"parent_used_ips"`
	ParentUnusedIPs  int               `json:"parent_unused_ips"`
	TotalCandidates  int               `json:"total_candidates"`
	AvailableSubnets []AvailableSubnet `json:"available_subnets"`
	UsedSubnets      []string          `json:"used_subnets"`
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
	maxConcurrency := userInput.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 32
	}
	maxConcurrency, _ = ClampConcurrency(maxConcurrency)
	if maxConcurrency > len(ips) {
		maxConcurrency = len(ips)
	}

	var usedIPs []IPStatus
	var unusedIPs []IPStatus
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, maxConcurrency)
	var currentMax atomic.Int64
	currentMax.Store(int64(maxConcurrency))

	ipChan := make(chan string, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	for range maxConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				sem <- struct{}{}
				pingable, err := checkIfIPIsAvailable(ip, userInput)
				if err != nil && strings.Contains(err.Error(), "no buffer space available") {
					cur := currentMax.Load()
					if cur > 1 {
						newMax := cur / 2
						if newMax < 1 {
							newMax = 1
						}
						if currentMax.CompareAndSwap(cur, newMax) {
							log.Printf("No buffer space available, reducing concurrency from %d to %d", cur, newMax)
							// Drain excess semaphore slots
							for range cur - newMax {
								select {
								case <-sem:
								default:
								}
							}
						}
					}
					<-sem
					ipChan <- ip
					time.Sleep(100 * time.Millisecond)
					continue
				}
				mu.Lock()
				if pingable {
					usedIPs = append(usedIPs, IPStatus{Pingable: true, IP: ip})
				} else {
					unusedIPs = append(unusedIPs, IPStatus{Pingable: false, IP: ip})
				}
				mu.Unlock()
				bar.Add(1)
				<-sem
			}
		}()
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

func checkIfIPIsAvailable(ip string, userInput UserInput) (bool, error) {
	for i := 0; i < userInput.RetryCount; i++ {
		pinger, err := probing.NewPinger(ip)
		if err != nil {
			log.Printf("Error creating pinger for %s: %v\n", ip, err)
			return false, nil
		}

		pinger.Timeout = time.Second
		pinger.Count = userInput.PingCount

		err = pinger.Run()
		if err != nil {
			if strings.Contains(err.Error(), "no buffer space available") {
				return false, err
			}
			log.Printf("Error running pinger for %s: %v\n", ip, err)
			return false, nil
		}

		stats := pinger.Statistics()
		if stats.PacketsRecv != 0 && stats.PacketsRecv <= stats.PacketsSent {
			return true, nil
		}
	}

	return false, nil
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

func FindAvailableSubnets(userInput UserInput) {
	parentIP, parentNet, err := net.ParseCIDR(userInput.IPAddr)
	if err != nil {
		log.Fatal(err)
	}

	requestedPrefix := userInput.SubnetPrefix
	ones, bits := parentNet.Mask.Size()
	if requestedPrefix <= ones || requestedPrefix > bits-1 {
		log.Fatalf("Requested prefix /%d must be larger than parent prefix /%d and less than /%d\n", requestedPrefix, ones, bits)
	}

	// Expand all IPs in the parent network
	var parentIPs []string
	for ip := parentIP.Mask(parentNet.Mask); parentNet.Contains(ip); inc(ip) {
		parentIPs = append(parentIPs, ip.String())
	}

	totalParentIPs := len(parentIPs)
	log.Printf("Scanning %d IPs in parent network %s\n", totalParentIPs, userInput.IPAddr)

	bar := progressbar.Default(int64(totalParentIPs), "Scanning network >>>")

	usedIPs, unusedIPs := processAllIPs(parentIPs, bar, userInput)

	bar.Finish()

	// Build a set of used IP addresses for fast lookup
	usedIPSet := make(map[string]bool, len(usedIPs))
	for _, ipStatus := range usedIPs {
		usedIPSet[ipStatus.IP] = true
	}

	// Calculate subnet size and iterate candidate subnets
	subnetSize := 1 << uint(bits-requestedPrefix)

	// Get the start IP of the parent network
	parentStartIP := parentIP.Mask(parentNet.Mask)

	// Build candidate subnets
	type candidateSubnet struct {
		network string
		startIP net.IP
	}

	var candidates []candidateSubnet
	ip := make(net.IP, len(parentStartIP))
	copy(ip, parentStartIP)

	for i := 0; i < totalParentIPs; i += subnetSize {
		if !parentNet.Contains(ip) {
			break
		}
		networkAddr := make(net.IP, len(ip))
		copy(networkAddr, ip)
		candidates = append(candidates, candidateSubnet{
			network: fmt.Sprintf("%s/%d", networkAddr.String(), requestedPrefix),
			startIP: networkAddr,
		})
		// Advance by subnetSize
		for j := 0; j < subnetSize; j++ {
			inc(ip)
		}
	}

	var availableSubnets []AvailableSubnet
	var usedSubnets []string

	for _, candidate := range candidates {
		// Check each IP in this candidate subnet
		subnetIP := make(net.IP, len(candidate.startIP))
		copy(subnetIP, candidate.startIP)
		hasConflict := false

		for j := 0; j < subnetSize; j++ {
			if usedIPSet[subnetIP.String()] {
				hasConflict = true
				break
			}
			inc(subnetIP)
		}

		// Calculate IP range for display
		rangeStart := candidate.startIP.String()
		rangeEndIP := make(net.IP, len(candidate.startIP))
		copy(rangeEndIP, candidate.startIP)
		for j := 1; j < subnetSize; j++ {
			inc(rangeEndIP)
		}
		rangeEnd := rangeEndIP.String()

		if hasConflict {
			usedSubnets = append(usedSubnets, candidate.network)
		} else {
			usableIPs := subnetSize - 2
			if subnetSize <= 2 {
				usableIPs = subnetSize
			}
			availableSubnets = append(availableSubnets, AvailableSubnet{
				Subnet:    candidate.network,
				IPRange:   fmt.Sprintf("%s - %s", rangeStart, rangeEnd),
				TotalIPs:  subnetSize,
				UsableIPs: usableIPs,
			})
		}
	}

	switch userInput.OutputFormat {
	case "table":
		printSubnetRecommendationTable(userInput.IPAddr, requestedPrefix, totalParentIPs,
			len(usedIPs), len(unusedIPs), len(candidates), availableSubnets, usedSubnets)
	case "json":
		printSubnetRecommendationJSON(userInput.IPAddr, requestedPrefix, totalParentIPs,
			len(usedIPs), len(unusedIPs), len(candidates), availableSubnets, usedSubnets)
	}
}

func printSubnetRecommendationTable(parent string, prefix int, totalIPs int,
	usedCount int, unusedCount int, totalCandidates int,
	available []AvailableSubnet, used []string) {

	printSeparator(fmt.Sprintf("Subnet Recommendations for /%d in %s", prefix, parent))
	fmt.Printf("  PARENT NETWORK:      %s\n", parent)
	fmt.Printf("  REQUESTED PREFIX:    /%d (%d IPs per subnet)\n", prefix, 1<<(uint(32-prefix)))
	fmt.Printf("  TOTAL IPS SCANNED:   %d\n", totalIPs)
	fmt.Printf("  USED IPS:            %d\n", usedCount)
	fmt.Printf("  AVAILABLE IPS:       %d\n", unusedCount)

	printSeparator("Available Subnets")
	if len(available) == 0 {
		fmt.Println("  No available subnets found.")
	} else {
		fmt.Printf("  %-22s %-35s %-11s %s\n", "SUBNET", "IP RANGE", "TOTAL IPS", "USABLE IPS")
		fmt.Printf("  %-22s %-35s %-11s %s\n", "------", "--------", "---------", "----------")
		for _, s := range available {
			fmt.Printf("  %-22s %-35s %-11d %d\n", s.Subnet, s.IPRange, s.TotalIPs, s.UsableIPs)
		}
	}

	if len(used) > 0 {
		printSeparator("Used Subnets (conflicts detected)")
		for _, s := range used {
			fmt.Printf("  %s\n", s)
		}
	}

	printSeparator("Summary")
	fmt.Printf("  TOTAL CANDIDATE SUBNETS:   %d\n", totalCandidates)
	fmt.Printf("  AVAILABLE SUBNETS:         %d\n", len(available))
	fmt.Printf("  USED SUBNETS:              %d\n", len(used))
}

func printSubnetRecommendationJSON(parent string, prefix int, totalIPs int,
	usedCount int, unusedCount int, totalCandidates int,
	available []AvailableSubnet, used []string) {

	result := SubnetRecommendationResult{
		ParentNetwork:    parent,
		RequestedPrefix:  prefix,
		ParentTotalIPs:   totalIPs,
		ParentUsedIPs:    usedCount,
		ParentUnusedIPs:  unusedCount,
		TotalCandidates:  totalCandidates,
		AvailableSubnets: available,
		UsedSubnets:      used,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonData))
}
