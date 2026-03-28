package cmd

import (
	"log"
	"runtime"
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
