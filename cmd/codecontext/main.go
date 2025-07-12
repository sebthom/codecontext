package main

import (
	"os"

	"github.com/nuthan-ms/codecontext/internal/cli"
)

// Version information - set during build time
var (
	version   = "dev"
	buildDate = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Set version information for CLI
	cli.SetVersion(version, buildDate, gitCommit)

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
