package main

import (
	"os"

	"github.com/nuthan-ms/codecontext/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}