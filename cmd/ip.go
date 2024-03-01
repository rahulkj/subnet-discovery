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

func ValidateIFInputIsReachableOrNot(input string) {
	if checkIfInputIsIP(input) {
		ip := net.ParseIP(input).String()
		pingable := checkIfIPIsAvailable(ip)
		printTableHeader()
		printTable(ip, pingable)
	} else if checkIfInputIsCIDR(input) {
		expandCIDRAndLogIPStatus(input)
	}
}

func expandCIDRAndLogIPStatus(subnet string) {
	ipAddr, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatal(err)
	}

	var ips []string

	for ip := ipAddr.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	log.Printf("Subnet length: %d\n", len(ips))

	wg := new(sync.WaitGroup)
	wg.Add(len(ips))

	results := make(chan IPStatus)

	printTableHeader()
	for _, ip := range ips {
		go func(ip string) {
			defer wg.Done()
			checkIfIPIsPingable(ip, results)
		}(ip)
	}

	var unusedIPs []string
	var usedIPs []string

	for i := 0; i < len(ips); i++ {
		res := <-results
		if res.Pingable {
			usedIPs = append(usedIPs, res.IP)
		} else {
			unusedIPs = append(unusedIPs, res.IP)
		}
		printTable(res.IP, res.Pingable)
	}

	sort.Strings(usedIPs)
	sort.Strings(unusedIPs)

	fmt.Println("USED IPS:", len(usedIPs))
	fmt.Println("UNUSED IPS:", len(unusedIPs))

	wg.Wait()
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
