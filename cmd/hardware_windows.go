package cmd

func detectAvailableMemory() uint64 {
	// Fallback for Windows - use conservative default
	return 512 * 1024 * 1024
}

func detectMaxFileDescriptors() uint64 {
	// Fallback for Windows - use conservative default
	return 1024
}

func detectNetworkBufferLimit() int {
	return 2048
}
