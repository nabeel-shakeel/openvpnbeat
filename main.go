package main

import (
	"os"

	"github.com/nabeel-shakeel/openvpnbeat/cmd"

	// Make sure all your modules and metricsets are linked in this file
	_ "github.com/nabeel-shakeel/openvpnbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
