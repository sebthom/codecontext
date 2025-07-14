package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	// Simulate the exact test conditions
	cwd, _ := os.Getwd()
	fmt.Println("Current directory:", cwd)
	
	// The exact command from the test
	buildCmd := exec.Command("go", "build", "-o", "../codecontext", "../cmd/codecontext")
	buildCmd.Dir = ".."
	
	fmt.Println("Command:", buildCmd.String())
	fmt.Println("Working directory:", buildCmd.Dir)
	
	// Capture both stdout and stderr
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		fmt.Println("Build succeeded")
		fmt.Printf("Output: %s\n", string(output))
	}
}