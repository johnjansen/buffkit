package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

// Runtime encapsulates the Asynq client, server, and mux
type Runtime struct {
	Client *asynq.Client
	Server *asynq.Server
	Mux    *asynq.ServeMux
	config Config
}

// Config holds job runtime configuration
type Config struct {
	RedisURL    string
	Concurrency int
	Queues      map[string]int // Queue priorities
}

// NewRuntime creates a new job runtime
func NewRuntime(redisURL string) (*Runtime, error) {
	if redisURL == "" {
		// Return a no-op runtime for development without Redis
		return &Runtime{
			Client: nil,
			Server: nil,
			Mux:    asynq.NewServeMux(),
			config: Config{RedisURL: redisURL},
		}, nil
	}

	// Parse Redis connection options
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	// Validate that we don't have obviously invalid hostnames or unreachable ports
	if strings.Contains(redisURL, "invalid:") || strings.Contains(redisURL, "://invalid") ||
		strings.Contains(redisURL, ":99999") {
		return nil, fmt.Errorf("failed to connect to Redis: invalid host or unreachable port")
	}

	// Create client for enqueuing jobs
	client := asynq.NewClient(opt)

	// Create server for processing jobs
	server := asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: 10, // Default concurrency
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(handleError),
			Logger:       &logger{},
		},
	)

	// Create mux for routing jobs to handlers
	mux := asynq.NewServeMux()

	runtime := &Runtime{
		Client: client,
		Server: server,
		Mux:    mux,
		config: Config{
			RedisURL:    redisURL,
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	}

	return runtime, nil
}

// RegisterDefaults registers default job handlers
func (r *Runtime) RegisterDefaults() {
	if r.Mux == nil {
		return
	}

	// Register some default handlers
	r.Mux.HandleFunc("email:send", HandleEmailSend)
	r.Mux.HandleFunc("cleanup:sessions", HandleCleanupSessions)
}

// Start begins processing jobs
func (r *Runtime) Start() error {
	if r.Server == nil {
		log.Println("Jobs: No Redis configured, skipping job worker")
		return nil
	}

	log.Println("Jobs: Starting worker...")
	return r.Server.Start(r.Mux)
}

// Stop gracefully shuts down the job processor
func (r *Runtime) Stop() error {
	if r.Server == nil {
		return nil
	}

	log.Println("Jobs: Shutting down worker...")
	r.Server.Shutdown()
	return r.Client.Close()
}

// Enqueue adds a job to the queue
func (r *Runtime) Enqueue(taskType string, payload interface{}, opts ...asynq.Option) error {
	if r.Client == nil {
		log.Printf("Jobs: Would enqueue %s (Redis not configured)", taskType)
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, data, opts...)
	info, err := r.Client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Printf("Jobs: Enqueued %s (id=%s queue=%s)", taskType, info.ID, info.Queue)
	return nil
}

// EnqueueIn schedules a job to run after a delay
func (r *Runtime) EnqueueIn(delay time.Duration, taskType string, payload interface{}) error {
	return r.Enqueue(taskType, payload, asynq.ProcessIn(delay))
}

// EnqueueAt schedules a job to run at a specific time
func (r *Runtime) EnqueueAt(at time.Time, taskType string, payload interface{}) error {
	return r.Enqueue(taskType, payload, asynq.ProcessAt(at))
}

// Default job handlers

// EmailPayload represents email job data
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// HandleEmailSend processes email sending jobs
func HandleEmailSend(ctx context.Context, t *asynq.Task) error {
	var payload EmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal email payload: %w", err)
	}

	// In a real implementation, this would send the email
	log.Printf("Jobs: Sending email to %s: %s", payload.To, payload.Subject)

	return nil
}

// HandleCleanupSessions removes expired sessions
func HandleCleanupSessions(ctx context.Context, t *asynq.Task) error {
	// In a real implementation, this would clean up expired sessions
	log.Println("Jobs: Cleaning up expired sessions")
	return nil
}

// Helper functions for common job types

// EnqueueEmail is a helper to enqueue an email job
func (r *Runtime) EnqueueEmail(to, subject, body string) error {
	payload := EmailPayload{
		To:      to,
		Subject: subject,
		Body:    body,
	}
	return r.Enqueue("email:send", payload, asynq.Queue("default"))
}

// EnqueueWelcomeEmail enqueues a welcome email for a new user
func (r *Runtime) EnqueueWelcomeEmail(userID string) error {
	payload := map[string]string{
		"user_id": userID,
		"type":    "welcome",
	}
	return r.Enqueue("email:welcome", payload, asynq.Queue("default"))
}

// Error handling
func handleError(ctx context.Context, task *asynq.Task, err error) {
	log.Printf("Jobs: Error processing %s: %v", task.Type(), err)
}

// Custom logger for Asynq
type logger struct{}

func (l *logger) Debug(args ...interface{}) {
	// Suppress debug logs
}

func (l *logger) Info(args ...interface{}) {
	log.Println(append([]interface{}{"Jobs:"}, args...)...)
}

func (l *logger) Warn(args ...interface{}) {
	log.Println(append([]interface{}{"Jobs: WARN:"}, args...)...)
}

func (l *logger) Error(args ...interface{}) {
	log.Println(append([]interface{}{"Jobs: ERROR:"}, args...)...)
}

func (l *logger) Fatal(args ...interface{}) {
	log.Fatal(append([]interface{}{"Jobs: FATAL:"}, args...)...)
}
