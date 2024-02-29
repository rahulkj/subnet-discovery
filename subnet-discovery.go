package main

import (
	"flag"
	"subnet-planner/cmd"
)

func main() {
	input := flag.String("s", "", "Provide the subnet to query, ex: 172.0.0.0/16")
	flag.Parse()

	if len(*input) == 0 {
		flag.PrintDefaults()
		return
	}

	cmd.ValidateIFInputIsReachableOrNot(*input)
}
