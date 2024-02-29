package cmd

import (
	"fmt"
	probing "github.com/prometheus-community/pro-bing"
	"log"
	"net"
	"net/netip"
)

func ValidateIFInputIsReachableOrNot(input string) {
	if checkIfInputIsIP(input) {
		ip := net.ParseIP(input).String()
		pingable := checkIfIPIsPingable(ip)
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

	printTableHeader()
	for _, ip := range ips {
		pingable := checkIfIPIsPingable(ip)
		printTable(ip, pingable)
	}
}

func inc(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func checkIfIPIsPingable(ip string) bool {
	var pingable bool
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		panic(err)
	}

	pinger.Timeout = 500000000

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
