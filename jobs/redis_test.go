package jobs_test

import (
	"testing"

	"github.com/hibiken/asynq"
	"github.com/johnjansen/buffkit/jobs"
)

// TestRedisConnection verifies Redis is accessible for testing
func TestRedisConnection(t *testing.T) {
	// Start Redis container (or use GHA service)
	container, err := jobs.StartRedisContainer()
	if err != nil {
		t.Fatalf("Failed to start Redis: %v", err)
	}
	defer func() {
		_ = container.Stop()
	}()

	// Try to connect to Redis
	opt, err := asynq.ParseRedisURI(container.URL())
	if err != nil {
		t.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := asynq.NewClient(opt)
	defer func() {
		_ = client.Close()
	}()

	// Try to enqueue a test task
	task := asynq.NewTask("test:ping", []byte("{}"))
	info, err := client.Enqueue(task)
	if err != nil {
		t.Fatalf("Failed to enqueue test task: %v", err)
	}

	if info.ID == "" {
		t.Fatal("Expected task ID, got empty string")
	}

	t.Logf("Successfully connected to Redis and enqueued task %s", info.ID)
}
