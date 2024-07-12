package cmd

import (
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
	IP       string
	Pingable bool
}

func ValidateIFInputIsReachableOrNot(input string, ipsToProcessInABatch int, retry int) {
	if checkIfInputIsIP(input) {
		ip := net.ParseIP(input).String()
		pingable := checkIfIPIsAvailable(ip, retry)
		printTableHeader()
		printTable(ip, pingable)
	} else if checkIfInputIsCIDR(input) {
		expandCIDRAndLogIPStatus(input, ipsToProcessInABatch, retry)
	}
}

func expandCIDRAndLogIPStatus(subnet string, ipsToProcessInABatch int, retry int) {
	ipAddr, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatal(err)
	}

	var ips []string

	for ip := ipAddr.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	log.Printf("Subnet length: %d\n", len(ips))

	// placeholders for used and unused IP's
	var usedIPs []IPStatus
	var unusedIPs []IPStatus

	// Get the current length of the array
	currentLen := len(ips)

	bar := progressbar.Default(int64(len(ips)), "IP Ping Status >>>")

	var numberOfLoops int
	if currentLen > ipsToProcessInABatch {
		numberOfLoops = currentLen / ipsToProcessInABatch
	}

	completedLoops := 0
	startIndex, endIndex := getNextIndex(numberOfLoops, completedLoops, 0, 0, currentLen, ipsToProcessInABatch)

	if numberOfLoops == 0 {
		newArray := ips[startIndex:endIndex]
		usedIPs, unusedIPs = processRequest(newArray, usedIPs, unusedIPs, bar, retry)
	} else {
		for i := 0; i < numberOfLoops; i++ {
			newArray := ips[startIndex:endIndex]

			usedIPs, unusedIPs = processRequest(newArray, usedIPs, unusedIPs, bar, retry)

			completedLoops = i + 1
			startIndex, endIndex = getNextIndex(numberOfLoops, completedLoops, startIndex, endIndex, currentLen, ipsToProcessInABatch)
		}
	}

	var usedIPArray []string
	var unUsedIPArray []string

	for _, usedIPResponse := range usedIPs {
		usedIPArray = append(usedIPArray, usedIPResponse.IP)
	}

	for _, unUsedIPResponse := range unusedIPs {
		unUsedIPArray = append(unUsedIPArray, unUsedIPResponse.IP)
	}

	sort.Strings(usedIPArray)
	sort.Strings(unUsedIPArray)

	printSeparater("UnAvailable IPs")
	printTableHeader()

	for _, usedIP := range usedIPArray {
		printTable(usedIP, true)
	}

	printSeparater("Available IPs")
	printTableHeader()

	for _, unUsedIP := range unUsedIPArray {
		printTable(unUsedIP, false)
	}

	fmt.Println("\nSummary of the scan")
	printSeparater("")
	fmt.Println("USED IPS:", len(usedIPArray))
	fmt.Println("UNUSED IPS:", len(unUsedIPArray))
}

func processRequest(newArray []string, usedIPs []IPStatus, unusedIPs []IPStatus, bar *progressbar.ProgressBar, retry int) ([]IPStatus, []IPStatus) {
	wg := new(sync.WaitGroup)
	results := make(chan IPStatus)

	for _, ip := range newArray {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			checkIfIPIsPingable(ip, results, retry)
		}(ip)
	}

	for i := 0; i < len(newArray); i++ {
		res := <-results
		if res.Pingable {
			usedIPs = append(usedIPs, res)
		} else {
			unusedIPs = append(unusedIPs, res)
		}
		bar.Add(1)
	}

	wg.Wait()

	return usedIPs, unusedIPs
}

func getNextIndex(numberOfLoops int, completedLoops int, startIndex int, endIndex int, currentLen int, ipsToProcessInABatch int) (int, int) {
	if currentLen < ipsToProcessInABatch || (numberOfLoops-completedLoops) == 0 {
		startIndex = 0
		endIndex = currentLen
	} else if numberOfLoops == completedLoops {
		startIndex = endIndex
		endIndex = currentLen
	} else if (numberOfLoops - completedLoops) >= 1 {
		startIndex = endIndex
		endIndex = endIndex + ipsToProcessInABatch
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

func checkIfIPIsPingable(ip string, results chan IPStatus, retry int) {
	pingable := checkIfIPIsAvailable(ip, retry)
	ipstatus := IPStatus{Pingable: pingable, IP: ip}
	results <- ipstatus
}

func checkIfIPIsAvailable(ip string, retry int) bool {
	var pingable bool

	for i := 0; i < retry; i++ {
		pinger, err := probing.NewPinger(ip)
		if err != nil {
			panic(err)
		}

		pinger.Timeout = time.Second
		pinger.Count = 10

		err = pinger.Run()
		if err != nil {
			panic(err)
		}

		stats := pinger.Statistics()

		if stats.PacketsRecv != 0 && stats.PacketsRecv <= stats.PacketsSent {
			pingable = true
		}

		if pingable {
			break
		}
	}

	return pingable
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

func printSeparater(header string) {
	fmt.Printf("------------------%s------------------------\n", header)
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
