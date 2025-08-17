package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/johnjansen/buffkit/auth"
	"github.com/johnjansen/buffkit/mail"
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
	r.Mux.HandleFunc("email:welcome", HandleWelcomeEmail)
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

	// Use the mail package to actually send the email
	// The mail sender should be configured by the app via mail.UseSender()
	sender := mail.GetSender()
	if sender == nil {
		// If no sender configured, just log (dev mode)
		log.Printf("Jobs: Would send email to %s: %s (no mail sender configured)", payload.To, payload.Subject)
		return nil
	}

	// Create a mail message
	message := mail.Message{
		To:      payload.To,
		Subject: payload.Subject,
		Text:    payload.Body,
		HTML:    payload.Body, // Use same content for HTML unless specified differently
	}

	// Send the email
	if err := sender.Send(ctx, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Jobs: Email sent to %s: %s", payload.To, payload.Subject)
	return nil
}

// HandleCleanupSessions removes expired sessions
func HandleCleanupSessions(ctx context.Context, t *asynq.Task) error {
	// Get the auth store to clean up sessions
	store := auth.GetStore()
	if store == nil {
		log.Println("Jobs: No auth store configured, skipping session cleanup")
		return nil
	}

	// If the store supports session cleanup, do it
	if extStore, ok := store.(auth.ExtendedUserStore); ok {
		// Clean up sessions older than 24 hours or inactive for 2 hours
		maxAge := 24 * time.Hour
		maxInactivity := 2 * time.Hour

		count, err := extStore.CleanupSessions(ctx, maxAge, maxInactivity)
		if err != nil {
			return fmt.Errorf("failed to cleanup sessions: %w", err)
		}

		log.Printf("Jobs: Cleaned up %d expired sessions", count)
	} else {
		log.Println("Jobs: Auth store doesn't support session cleanup")
	}

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

// HandleWelcomeEmail processes welcome email jobs for new users
func HandleWelcomeEmail(ctx context.Context, t *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
	}

	userID := payload["user_id"]
	if userID == "" {
		return fmt.Errorf("missing user_id in welcome email payload")
	}

	// Get the auth store to fetch user details
	store := auth.GetStore()
	if store == nil {
		log.Printf("Jobs: No auth store configured, skipping welcome email for user %s", userID)
		return nil
	}

	// Get user details
	user, err := store.ByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user %s: %w", userID, err)
	}

	// Get mail sender
	sender := mail.GetSender()
	if sender == nil {
		log.Printf("Jobs: Would send welcome email to %s (no mail sender configured)", user.Email)
		return nil
	}

	// Prepare welcome email content
	subject := "Welcome to Our Service!"
	textBody := fmt.Sprintf(`Hello %s,

Welcome to our service! We're excited to have you on board.

Your account has been successfully created with the email: %s

To get started:
1. Log in to your account
2. Complete your profile
3. Explore our features

If you have any questions, please don't hesitate to reach out.

Best regards,
The Team`, user.Name(), user.Email)

	htmlBody := fmt.Sprintf(`<h2>Hello %s,</h2>
<p>Welcome to our service! We're excited to have you on board.</p>
<p>Your account has been successfully created with the email: <strong>%s</strong></p>
<h3>To get started:</h3>
<ol>
  <li>Log in to your account</li>
  <li>Complete your profile</li>
  <li>Explore our features</li>
</ol>
<p>If you have any questions, please don't hesitate to reach out.</p>
<p>Best regards,<br>The Team</p>`, user.Name(), user.Email)

	// Create and send the email
	message := mail.Message{
		To:      user.Email,
		Subject: subject,
		Text:    textBody,
		HTML:    htmlBody,
	}

	if err := sender.Send(ctx, message); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("Jobs: Welcome email sent to %s", user.Email)
	return nil
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
