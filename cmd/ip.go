package cmd

import (
	"fmt"
	probing "github.com/prometheus-community/pro-bing"
	"log"
	"net"
	"net/netip"
	"sort"
	"sync"
	"time"
)

type IPStatus struct {
	IP       string
	Pingable bool
}

func ValidateIFInputIsReachableOrNot(input string, ipsToProcessInABatch int) {
	if checkIfInputIsIP(input) {
		ip := net.ParseIP(input).String()
		pingable := checkIfIPIsAvailable(ip)
		printTableHeader()
		printTable(ip, pingable)
	} else if checkIfInputIsCIDR(input) {
		expandCIDRAndLogIPStatus(input, ipsToProcessInABatch)
	}
}

func expandCIDRAndLogIPStatus(subnet string, ipsToProcessInABatch int) {
	ipAddr, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatal(err)
	}

	var ips []string

	for ip := ipAddr.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	log.Printf("Subnet length: %d\n", len(ips))

	printTableHeader()

	// placeholders for used and unused IP's
	var usedIPs []string
	var unusedIPs []string

	// Get the current length of the array
	currentLen := len(ips)

	var numberOfLoops int
	if currentLen > ipsToProcessInABatch {
		numberOfLoops = currentLen / ipsToProcessInABatch
	}

	completedLoops := 0
	startIndex, endIndex := getNextIndex(numberOfLoops, completedLoops, 0, 0, currentLen, ipsToProcessInABatch)

	for i := 0; i <= numberOfLoops; i++ {
		newArray := ips[startIndex:endIndex]

		wg := new(sync.WaitGroup)
		results := make(chan IPStatus)

		for _, ip := range newArray {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				checkIfIPIsPingable(ip, results)
			}(ip)
		}

		for i := 0; i < len(newArray); i++ {
			res := <-results
			if res.Pingable {
				usedIPs = append(usedIPs, res.IP)
			} else {
				unusedIPs = append(unusedIPs, res.IP)
			}
			printTable(res.IP, res.Pingable)
		}

		wg.Wait()

		completedLoops = i + 1
		startIndex, endIndex = getNextIndex(numberOfLoops, completedLoops, startIndex, endIndex, currentLen, ipsToProcessInABatch)
	}

	sort.Strings(usedIPs)
	sort.Strings(unusedIPs)

	fmt.Println("\nSummary of the scan\n---------------------")
	fmt.Println("USED IPS:", len(usedIPs))
	fmt.Println("UNUSED IPS:", len(unusedIPs))

}

func getNextIndex(numberOfLoops int, completedLoops int, startIndex int, endIndex int, currentLen int, ipsToProcessInABatch int) (int, int) {
	if currentLen < ipsToProcessInABatch || (numberOfLoops-completedLoops) == 0 {
		startIndex = 0
		endIndex = currentLen
	} else if (numberOfLoops - completedLoops) >= 1 {
		startIndex = endIndex + 1
		endIndex = endIndex + ipsToProcessInABatch
	} else if numberOfLoops == completedLoops {
		startIndex = endIndex + 1
		endIndex = currentLen
	}

	return startIndex, endIndex
}

func inc(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func checkIfIPIsPingable(ip string, results chan IPStatus) {
	pingable := checkIfIPIsAvailable(ip)
	ipstatus := IPStatus{Pingable: pingable, IP: ip}
	results <- ipstatus
}

func checkIfIPIsAvailable(ip string) bool {
	var pingable bool
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		panic(err)
	}

	pinger.Timeout = time.Second
	pinger.Count = 3

	pinger.OnFinish = func(stats *probing.Statistics) {
		if stats.PacketsRecv > 0 {
			pingable = true
		}
	}

	err = pinger.Run()
	if err != nil {
		panic(err)
	}

	return pingable
}

func checkIfInputIsIP(input string) bool {
	_, err := netip.ParseAddr(input)
	if err != nil {
		return false
	}
	return true
}

func checkIfInputIsCIDR(input string) bool {
	_, _, err := net.ParseCIDR(input)
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func printTableHeader() {
	fmt.Printf("%s\t\t %s\n", "IP ADDRESS", "STATUS")
	fmt.Printf("%s\t\t %s\n", "----------", "----------")
}

func printTable(ip string, pingable bool) {
	if pingable {
		fmt.Printf("%s\t\t %s\n", ip, "Unvailable")
	} else {
		fmt.Printf("%s\t\t %s\n", ip, "Available")
	}
}
