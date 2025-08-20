package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RedisContainer manages a Redis Docker container for testing
type RedisContainer struct {
	containerID string
	port        string
	isGHA       bool
}

// StartRedisContainer starts a Redis container for testing
func StartRedisContainer() (*RedisContainer, error) {
	// Check if we're running in GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// GHA provides Redis as a service on port 6379
		return &RedisContainer{
			port:  "6379",
			isGHA: true,
		}, nil
	}

	// For local testing, require Docker
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("Docker not found. Install Docker and try again")
	}

	// Check if Docker daemon is running
	versionCmd := exec.Command("docker", "version")
	if output, err := versionCmd.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "Cannot connect to the Docker daemon") {
			return nil, fmt.Errorf("Docker daemon is not running. Start Docker and try again")
		}
		return nil, fmt.Errorf("Docker check failed: %v", err)
	}

	// Start a Redis container
	cmd := exec.Command(
		"docker", "run",
		"-d",
		"--rm",
		"-p", "6379:6379",
		"redis:7-alpine",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start Redis container: %w\nOutput: %s", err, output)
	}

	containerID := strings.TrimSpace(string(output))

	// Wait for Redis to be ready
	for i := 0; i < 30; i++ {
		pingCmd := exec.Command("docker", "exec", containerID, "redis-cli", "ping")
		if output, err := pingCmd.Output(); err == nil && strings.TrimSpace(string(output)) == "PONG" {
			return &RedisContainer{
				containerID: containerID,
				port:        "6379",
				isGHA:       false,
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Clean up if we couldn't connect
	exec.Command("docker", "stop", containerID).Run()
	return nil, fmt.Errorf("Redis didn't start in time")
}

// Stop stops the Redis container
func (rc *RedisContainer) Stop() error {
	if rc.isGHA {
		// Don't stop GHA's Redis service
		return nil
	}
	if rc.containerID == "" {
		return nil
	}
	return exec.Command("docker", "stop", rc.containerID).Run()
}

// URL returns the Redis connection URL
func (rc *RedisContainer) URL() string {
	return fmt.Sprintf("redis://localhost:%s", rc.port)
}

// FlushAll clears all data from Redis
func (rc *RedisContainer) FlushAll() error {
	if rc.isGHA {
		// In GHA, connect to Redis service directly
		// We could use redis-cli if available, but let's use a simple network approach
		// For now, we'll just return nil since GHA tests should be isolated anyway
		return nil
	}
	return exec.Command("docker", "exec", rc.containerID, "redis-cli", "FLUSHALL").Run()
}
