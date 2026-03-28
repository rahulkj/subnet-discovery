package cmd

import (
	"log"
	"syscall"
)

func detectAvailableMemory() uint64 {
	val, err := syscall.Sysctl("hw.memsize")
	if err != nil {
		log.Printf("Warning: could not get hw.memsize: %v, using conservative default\n", err)
		return 512 * 1024 * 1024
	}

	// sysctl returns little-endian bytes; right-pad to 8 bytes
	b := make([]byte, 8)
	copy(b, []byte(val))
	physMem := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56

	return physMem * 60 / 100
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
	return 2048
}
