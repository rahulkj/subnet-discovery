package cmd

import (
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func detectAvailableMemory() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		log.Printf("Warning: could not read /proc/meminfo: %v, using conservative default\n", err)
		return 512 * 1024 * 1024
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return kb * 1024
				}
			}
		}
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemFree:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return kb * 1024
				}
			}
		}
	}

	return 512 * 1024 * 1024
}

func detectMaxFileDescriptors() uint64 {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Printf("Warning: could not get RLIMIT_NOFILE: %v, using conservative default\n", err)
		return 1024
	}
	return rLimit.Cur
}

func detectNetworkBufferLimit() int {
	wmemMax := readSysctlInt("/proc/sys/net/core/wmem_max")
	rmemMax := readSysctlInt("/proc/sys/net/core/rmem_max")

	if wmemMax <= 0 || rmemMax <= 0 {
		return 2048
	}

	// Each ICMP socket needs kernel memory for send and receive buffers
	perSocketBuf := wmemMax + rmemMax

	// Conservative estimate: kernel can support ~128MB of total socket buffers
	totalKernelBuf := 128 * 1024 * 1024
	limit := totalKernelBuf / perSocketBuf

	if limit < 1 {
		return 2048
	}
	return limit
}

func readSysctlInt(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return val
}
