package main

import (
	"fmt"
	"log"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/exec"
)

func main() {
	// Example 1: SSH execution with key authentication
	fmt.Println("=== Example 1: SSH remote execution ===")
	config := exec.SSHConfig{
		User:    "admin",
		Host:    "10.0.0.1",
		KeyPath: "/home/user/.ssh/id_rsa",
	}

	// Note: This example assumes SSH connectivity
	// out, errOut, err := exec.RunSSHCommand(config, "docker ps -a", 30)
	// if err != nil {
	//     log.Printf("SSH execution failed: %v (code=%d)", err, errors.GetCode(err))
	// } else {
	//     fmt.Printf("Exit code: %d\nOutput: %s\n", errors.GetCode(err), out)
	// }

	fmt.Println("SSH example would run here with proper credentials")

	// Example 2: Batch server operations
	fmt.Println("\n=== Example 2: Batch server operations ===")
	servers := []string{"server1", "server2", "server3"}
	for _, host := range servers {
		config.Host = host
		fmt.Printf("Would execute maintenance on %s...\n", host)
		// exec.RunSSHCommand(config, "apt update && apt upgrade -y", 300)
	}

	// Example 3: CI/CD pipeline health check
	fmt.Println("\n=== Example 3: CI/CD pipeline health check ===")
	out, _, err := exec.RunReturnAll("curl -sSf http://localhost:8080/health", 10)
	if errors.GetCode(err) != 0 {
		log.Fatal("Service health check failed")
	}
	fmt.Printf("Health check passed: %s\n", out[:min(50, len(out))])

	// Example 4: Container management
	fmt.Println("\n=== Example 4: Container management ===")
	containerCmd := "singularity exec /images/debian.sif apt-get update"
	fmt.Printf("Would execute container command: %s\n", containerCmd)
	// exec.RunSSHCommand(config, containerCmd, 60)

	// Example 5: Background process
	fmt.Println("\n=== Example 5: Background process execution ===")
	fmt.Println("Would start background command and get PID")
	// Example: Background SSH command (commented out)
	// bgConfig := exec.SSHConfig{
	// 	Host:       "10.0.0.1",
	// 	User:       "admin",
	// 	Background: true,
	// 	KeyPath:    "/home/user/.ssh/id_rsa",
	// }
	// pid, _, _ := exec.RunSSHCommand(bgConfig, "long-running-command", 0)

	// Example 6: Command with retries
	fmt.Println("\n=== Example 6: Command with retries ===")
	retryCode, retryErr := exec.RunWithRetries("curl -sSf http://localhost:8080/ready", 3, 5)
	if retryErr != nil {
		fmt.Printf("Command with retries error: %v (exit code: %d)\n", retryErr, retryCode)
	} else {
		fmt.Printf("Command with retries final exit code: %d\n", retryCode)
	}

	// Example 7: Timeout handling
	fmt.Println("\n=== Example 7: Timeout handling ===")
	_, _, err = exec.RunReturnAll("sleep 5", 1)
	if err != nil {
		fmt.Printf("Command timed out as expected: %v (exit code: %d)\n", err, errors.GetCode(err))
	}

	// Example 8: Security considerations
	fmt.Println("\n=== Example 8: Security considerations ===")
	userInput := "../sensitive/path"
	safeCmd := fmt.Sprintf("ls %s", userInput)
	fmt.Printf("Safe command would be: %s\n", safeCmd)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
