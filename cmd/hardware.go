package cmd

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const bytesPerGoroutine = 256 * 1024 // ~256KB per ping goroutine

type HardwareLimits struct {
	CPUCores        int
	AvailableMemory uint64
	MaxFDs          uint64
	SafeConcurrency int
}

func DetectHardwareLimits() HardwareLimits {
	cores := runtime.NumCPU()

	mem := detectAvailableMemory()

	maxFDs := detectMaxFileDescriptors()

	netBufLimit := detectNetworkBufferLimit()

	cpuLimit := cores * 128

	memLimit := int(mem / bytesPerGoroutine)

	fdLimit := int(maxFDs * 80 / 100)

	safe := cpuLimit
	if memLimit < safe {
		safe = memLimit
	}
	if fdLimit > 0 && fdLimit < safe {
		safe = fdLimit
	}
	if netBufLimit > 0 && netBufLimit < safe {
		safe = netBufLimit
	}
	if safe < 1 {
		safe = 1
	}

	return HardwareLimits{
		CPUCores:        cores,
		AvailableMemory: mem,
		MaxFDs:          maxFDs,
		SafeConcurrency: safe,
	}
}

func ClampConcurrency(requested int) (int, bool) {
	limits := DetectHardwareLimits()

	if requested <= limits.SafeConcurrency {
		return requested, false
	}

	log.Printf("Requested concurrency %d exceeds hardware limit (%d CPU cores, %.1f GB available memory, %d max FDs)",
		requested, limits.CPUCores, float64(limits.AvailableMemory)/(1024*1024*1024), limits.MaxFDs)
	log.Printf("Reducing concurrency from %d to %d to match system capacity", requested, limits.SafeConcurrency)

	return limits.SafeConcurrency, true
}

func detectAvailableMemory() uint64 {
	switch runtime.GOOS {
	case "linux":
		return detectLinuxAvailableMemory()
	case "darwin":
		return detectDarwinMemory()
	case "windows":
		return detectWindowsMemory()
	default:
		return 512 * 1024 * 1024
	}
}

func detectLinuxAvailableMemory() uint64 {
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

func detectWindowsMemory() uint64 {
	// Fallback for Windows - use conservative default
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
	switch runtime.GOOS {
	case "linux":
		return detectLinuxNetworkBufferLimit()
	case "darwin":
		return 2048
	default:
		return 2048
	}
}

func detectLinuxNetworkBufferLimit() int {
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
