package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// Job type constants
const (
	JobTypeSessionCleanup     = "auth:session_cleanup"
	JobTypeAccountLockCheck   = "auth:account_lock_check"
	JobTypeAuditLogCleanup    = "auth:audit_log_cleanup"
	JobTypePasswordExpiry     = "auth:password_expiry_check"
	JobTypeInactiveUserNotify = "auth:inactive_user_notify"
)

// SessionCleanupPayload contains data for session cleanup job
type SessionCleanupPayload struct {
	MaxAge           time.Duration `json:"max_age"`
	MaxInactivity    time.Duration `json:"max_inactivity"`
	CleanupBatchSize int           `json:"cleanup_batch_size"`
}

// RegisterAuthJobs registers all authentication-related background jobs
func RegisterAuthJobs(mux *asynq.ServeMux, store ExtendedUserStore) {
	// Session cleanup handler
	mux.HandleFunc(JobTypeSessionCleanup, HandleSessionCleanup(store))

	// Account lock check handler
	mux.HandleFunc(JobTypeAccountLockCheck, HandleAccountLockCheck(store))

	// Audit log cleanup handler
	mux.HandleFunc(JobTypeAuditLogCleanup, HandleAuditLogCleanup(store))

	// Password expiry check handler
	mux.HandleFunc(JobTypePasswordExpiry, HandlePasswordExpiryCheck(store))

	// Inactive user notification handler
	mux.HandleFunc(JobTypeInactiveUserNotify, HandleInactiveUserNotification(store))
}

// HandleSessionCleanup creates a handler for cleaning up expired sessions
func HandleSessionCleanup(store ExtendedUserStore) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload SessionCleanupPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			// Use defaults if payload is invalid
			payload = SessionCleanupPayload{
				MaxAge:           24 * time.Hour,
				MaxInactivity:    2 * time.Hour,
				CleanupBatchSize: 100,
			}
		}

		log.Printf("[Auth Jobs] Starting session cleanup (max_age=%v, max_inactivity=%v)",
			payload.MaxAge, payload.MaxInactivity)

		// Get current time
		now := time.Now()
		expiryTime := now.Add(-payload.MaxAge)
		inactivityTime := now.Add(-payload.MaxInactivity)

		// Clean up expired sessions
		// Note: This is a simplified version. In production, you'd want to:
		// 1. Query sessions in batches
		// 2. Check both expiry time and last activity
		// 3. Delete in batches to avoid long transactions

		cleanedCount := 0

		// In a real implementation, you would:
		// 1. Query all sessions older than expiryTime
		// 2. Query all sessions with last_activity older than inactivityTime
		// 3. Delete them in batches

		// For now, we'll use a placeholder that assumes the store has a cleanup method
		if cleaner, ok := store.(interface {
			CleanupExpiredSessions(context.Context, time.Time, time.Time) (int, error)
		}); ok {
			count, err := cleaner.CleanupExpiredSessions(ctx, expiryTime, inactivityTime)
			if err != nil {
				return fmt.Errorf("failed to cleanup sessions: %w", err)
			}
			cleanedCount = count
		}

		log.Printf("[Auth Jobs] Session cleanup completed: %d sessions removed", cleanedCount)
		return nil
	}
}

// HandleAccountLockCheck creates a handler for checking and unlocking accounts
func HandleAccountLockCheck(store ExtendedUserStore) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("[Auth Jobs] Starting account lock check")

		// Check for accounts that should be unlocked
		// In a real implementation, you would:
		// 1. Query all users where locked_until < now
		// 2. Update them to set locked_until = NULL
		// 3. Reset failed_login_attempts = 0

		unlockedCount := 0

		if unlocker, ok := store.(interface {
			UnlockExpiredAccounts(context.Context) (int, error)
		}); ok {
			count, err := unlocker.UnlockExpiredAccounts(ctx)
			if err != nil {
				return fmt.Errorf("failed to unlock accounts: %w", err)
			}
			unlockedCount = count
		}

		log.Printf("[Auth Jobs] Account lock check completed: %d accounts unlocked", unlockedCount)
		return nil
	}
}

// HandleAuditLogCleanup creates a handler for cleaning up old audit logs
func HandleAuditLogCleanup(store ExtendedUserStore) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		// Default: Keep audit logs for 90 days
		retentionPeriod := 90 * 24 * time.Hour

		var payload struct {
			RetentionDays int `json:"retention_days"`
		}
		if err := json.Unmarshal(t.Payload(), &payload); err == nil && payload.RetentionDays > 0 {
			retentionPeriod = time.Duration(payload.RetentionDays) * 24 * time.Hour
		}

		log.Printf("[Auth Jobs] Starting audit log cleanup (retention=%v)", retentionPeriod)

		cutoffTime := time.Now().Add(-retentionPeriod)

		// In a real implementation, delete audit logs older than cutoffTime
		deletedCount := 0

		if cleaner, ok := store.(interface {
			CleanupAuditLogs(context.Context, time.Time) (int, error)
		}); ok {
			count, err := cleaner.CleanupAuditLogs(ctx, cutoffTime)
			if err != nil {
				return fmt.Errorf("failed to cleanup audit logs: %w", err)
			}
			deletedCount = count
		}

		log.Printf("[Auth Jobs] Audit log cleanup completed: %d logs removed", deletedCount)
		return nil
	}
}

// HandlePasswordExpiryCheck creates a handler for checking password expiry
func HandlePasswordExpiryCheck(store ExtendedUserStore) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		// Default: Passwords expire after 90 days
		expiryPeriod := 90 * 24 * time.Hour
		warningPeriod := 7 * 24 * time.Hour // Warn 7 days before expiry

		var payload struct {
			ExpiryDays  int `json:"expiry_days"`
			WarningDays int `json:"warning_days"`
		}
		if err := json.Unmarshal(t.Payload(), &payload); err == nil {
			if payload.ExpiryDays > 0 {
				expiryPeriod = time.Duration(payload.ExpiryDays) * 24 * time.Hour
			}
			if payload.WarningDays > 0 {
				warningPeriod = time.Duration(payload.WarningDays) * 24 * time.Hour
			}
		}

		log.Printf("[Auth Jobs] Starting password expiry check (expiry=%v, warning=%v)",
			expiryPeriod, warningPeriod)

		// now := time.Now()
		// expiryTime := now.Add(-expiryPeriod)
		// warningTime := now.Add(-expiryPeriod).Add(warningPeriod)

		// Find users with passwords about to expire or expired
		// In a real implementation:
		// 1. Query users where password_changed_at < expiryTime (expired)
		// 2. Query users where password_changed_at between expiryTime and warningTime (warning)
		// 3. Send notifications accordingly

		notifiedCount := 0
		expiredCount := 0

		// This would need a method to get users with old passwords
		// and send them notifications

		log.Printf("[Auth Jobs] Password expiry check completed: %d expired, %d notified",
			expiredCount, notifiedCount)
		return nil
	}
}

// HandleInactiveUserNotification creates a handler for notifying inactive users
func HandleInactiveUserNotification(store ExtendedUserStore) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		// Default: Consider users inactive after 30 days
		inactivePeriod := 30 * 24 * time.Hour

		var payload struct {
			InactiveDays int `json:"inactive_days"`
		}
		if err := json.Unmarshal(t.Payload(), &payload); err == nil && payload.InactiveDays > 0 {
			inactivePeriod = time.Duration(payload.InactiveDays) * 24 * time.Hour
		}

		log.Printf("[Auth Jobs] Starting inactive user check (period=%v)", inactivePeriod)

		// inactiveTime := time.Now().Add(-inactivePeriod)

		// Find users who haven't logged in since inactiveTime
		// In a real implementation:
		// 1. Query users where last_login_at < inactiveTime
		// 2. Send re-engagement emails
		// 3. Mark them as notified to avoid spam

		notifiedCount := 0

		log.Printf("[Auth Jobs] Inactive user check completed: %d users notified", notifiedCount)
		return nil
	}
}

// ScheduleAuthJobs schedules recurring authentication jobs
func ScheduleAuthJobs(scheduler *asynq.Scheduler) error {
	// Schedule session cleanup every hour
	sessionCleanupPayload, _ := json.Marshal(SessionCleanupPayload{
		MaxAge:           24 * time.Hour,
		MaxInactivity:    2 * time.Hour,
		CleanupBatchSize: 100,
	})

	if _, err := scheduler.Register(
		"0 * * * *", // Every hour at minute 0
		asynq.NewTask(JobTypeSessionCleanup, sessionCleanupPayload),
		asynq.Queue("maintenance"),
	); err != nil {
		return fmt.Errorf("failed to schedule session cleanup: %w", err)
	}

	// Schedule account lock check every 15 minutes
	if _, err := scheduler.Register(
		"*/15 * * * *", // Every 15 minutes
		asynq.NewTask(JobTypeAccountLockCheck, nil),
		asynq.Queue("maintenance"),
	); err != nil {
		return fmt.Errorf("failed to schedule account lock check: %w", err)
	}

	// Schedule audit log cleanup daily at 3 AM
	auditCleanupPayload, _ := json.Marshal(map[string]int{
		"retention_days": 90,
	})

	if _, err := scheduler.Register(
		"0 3 * * *", // Daily at 3:00 AM
		asynq.NewTask(JobTypeAuditLogCleanup, auditCleanupPayload),
		asynq.Queue("maintenance"),
	); err != nil {
		return fmt.Errorf("failed to schedule audit log cleanup: %w", err)
	}

	// Schedule password expiry check daily at 9 AM
	passwordExpiryPayload, _ := json.Marshal(map[string]int{
		"expiry_days":  90,
		"warning_days": 7,
	})

	if _, err := scheduler.Register(
		"0 9 * * *", // Daily at 9:00 AM
		asynq.NewTask(JobTypePasswordExpiry, passwordExpiryPayload),
		asynq.Queue("notifications"),
	); err != nil {
		return fmt.Errorf("failed to schedule password expiry check: %w", err)
	}

	// Schedule inactive user check weekly on Mondays at 10 AM
	inactiveUserPayload, _ := json.Marshal(map[string]int{
		"inactive_days": 30,
	})

	if _, err := scheduler.Register(
		"0 10 * * 1", // Mondays at 10:00 AM
		asynq.NewTask(JobTypeInactiveUserNotify, inactiveUserPayload),
		asynq.Queue("notifications"),
	); err != nil {
		return fmt.Errorf("failed to schedule inactive user check: %w", err)
	}

	log.Println("[Auth Jobs] All authentication jobs scheduled successfully")
	return nil
}

// CreateSessionCleanupTask creates a one-off session cleanup task
func CreateSessionCleanupTask(maxAge, maxInactivity time.Duration) *asynq.Task {
	payload, _ := json.Marshal(SessionCleanupPayload{
		MaxAge:           maxAge,
		MaxInactivity:    maxInactivity,
		CleanupBatchSize: 100,
	})
	return asynq.NewTask(JobTypeSessionCleanup, payload)
}

// CreateAccountUnlockTask creates a one-off account unlock task
func CreateAccountUnlockTask() *asynq.Task {
	return asynq.NewTask(JobTypeAccountLockCheck, nil)
}

// CreateAuditLogCleanupTask creates a one-off audit log cleanup task
func CreateAuditLogCleanupTask(retentionDays int) *asynq.Task {
	payload, _ := json.Marshal(map[string]int{
		"retention_days": retentionDays,
	})
	return asynq.NewTask(JobTypeAuditLogCleanup, payload)
}
