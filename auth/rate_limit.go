package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gobuffalo/buffalo"
)

// RateLimiter provides rate limiting for authentication endpoints
type RateLimiter struct {
	mu sync.RWMutex

	// Per-email rate limiting
	emailAttempts map[string]*attemptRecord

	// Per-IP rate limiting
	ipAttempts map[string]*attemptRecord

	// Configuration
	maxEmailAttempts int           // Max attempts per email
	maxIPAttempts    int           // Max attempts per IP
	windowDuration   time.Duration // Time window for counting attempts
	lockoutDuration  time.Duration // How long to lock out after max attempts

	// Cleanup
	lastCleanup time.Time
}

// attemptRecord tracks attempts for rate limiting
type attemptRecord struct {
	attempts    []time.Time
	lockedUntil time.Time
}

// NewRateLimiter creates a new rate limiter with default settings
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		emailAttempts:    make(map[string]*attemptRecord),
		ipAttempts:       make(map[string]*attemptRecord),
		maxEmailAttempts: 5,  // 5 attempts per email
		maxIPAttempts:    20, // 20 attempts per IP
		windowDuration:   15 * time.Minute,
		lockoutDuration:  30 * time.Minute,
		lastCleanup:      time.Now(),
	}
}

// NewRateLimiterWithConfig creates a rate limiter with custom configuration
func NewRateLimiterWithConfig(maxEmail, maxIP int, window, lockout time.Duration) *RateLimiter {
	return &RateLimiter{
		emailAttempts:    make(map[string]*attemptRecord),
		ipAttempts:       make(map[string]*attemptRecord),
		maxEmailAttempts: maxEmail,
		maxIPAttempts:    maxIP,
		windowDuration:   window,
		lockoutDuration:  lockout,
		lastCleanup:      time.Now(),
	}
}

// RateLimitMiddleware returns Buffalo middleware for rate limiting auth endpoints
func RateLimitMiddleware(limiter *RateLimiter) buffalo.MiddlewareFunc {
	if limiter == nil {
		limiter = NewRateLimiter()
	}

	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Only rate limit specific auth endpoints
			path := c.Request().URL.Path
			method := c.Request().Method

			// Check if this is a rate-limited endpoint
			if shouldRateLimit(path, method) {
				// Get client IP
				ip := getClientIP(c.Request())

				// For login/password reset, also check email if provided
				var email string
				if method == http.MethodPost {
					email = c.Param("email")
				}

				// Check rate limits
				allowed, retryAfter, reason := limiter.CheckRateLimit(ip, email)

				if !allowed {
					// Set retry-after header
					c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

					// Return rate limit error
					return c.Error(http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded: %s", reason))
				}

				// Record the attempt after the request completes
				defer func() {
					// Only record failed attempts for login/reset endpoints
					// Buffalo doesn't expose status directly, so we track it differently
					// This will be called after the handler completes
					_ = limiter.RecordAttempt(ip, email)
				}()
			}

			return next(c)
		}
	}
}

// CheckRateLimit checks if a request should be allowed
func (rl *RateLimiter) CheckRateLimit(ip, email string) (allowed bool, retryAfter time.Duration, reason string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodically clean up old records
	rl.cleanupIfNeeded()

	now := time.Now()

	// Check IP-based rate limit
	if ip != "" {
		if record, exists := rl.ipAttempts[ip]; exists {
			// Check if locked out
			if record.lockedUntil.After(now) {
				return false, record.lockedUntil.Sub(now), "IP address temporarily blocked"
			}

			// Count recent attempts
			recentCount := rl.countRecentAttempts(record, now)
			if recentCount >= rl.maxIPAttempts {
				// Lock out the IP
				record.lockedUntil = now.Add(rl.lockoutDuration)
				return false, rl.lockoutDuration, "too many attempts from this IP"
			}
		}
	}

	// Check email-based rate limit
	if email != "" {
		if record, exists := rl.emailAttempts[email]; exists {
			// Check if locked out
			if record.lockedUntil.After(now) {
				return false, record.lockedUntil.Sub(now), "account temporarily locked"
			}

			// Count recent attempts
			recentCount := rl.countRecentAttempts(record, now)
			if recentCount >= rl.maxEmailAttempts {
				// Lock out the email
				record.lockedUntil = now.Add(rl.lockoutDuration)
				return false, rl.lockoutDuration, "too many attempts for this account"
			}
		}
	}

	return true, 0, ""
}

// RecordAttempt records a failed attempt
func (rl *RateLimiter) RecordAttempt(ip, email string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Record IP attempt
	if ip != "" {
		if _, exists := rl.ipAttempts[ip]; !exists {
			rl.ipAttempts[ip] = &attemptRecord{
				attempts: []time.Time{},
			}
		}
		rl.ipAttempts[ip].attempts = append(rl.ipAttempts[ip].attempts, now)
	}

	// Record email attempt
	if email != "" {
		if _, exists := rl.emailAttempts[email]; !exists {
			rl.emailAttempts[email] = &attemptRecord{
				attempts: []time.Time{},
			}
		}
		rl.emailAttempts[email].attempts = append(rl.emailAttempts[email].attempts, now)
	}
}

// ResetAttempts clears attempts for a specific email (used after successful login)
func (rl *RateLimiter) ResetAttempts(email string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if record, exists := rl.emailAttempts[email]; exists {
		record.attempts = []time.Time{}
		record.lockedUntil = time.Time{}
	}
}

// countRecentAttempts counts attempts within the time window
func (rl *RateLimiter) countRecentAttempts(record *attemptRecord, now time.Time) int {
	cutoff := now.Add(-rl.windowDuration)
	count := 0

	// Count attempts within the window
	var recentAttempts []time.Time
	for _, attempt := range record.attempts {
		if attempt.After(cutoff) {
			count++
			recentAttempts = append(recentAttempts, attempt)
		}
	}

	// Clean up old attempts
	record.attempts = recentAttempts

	return count
}

// cleanupIfNeeded removes old records to prevent memory growth
func (rl *RateLimiter) cleanupIfNeeded() {
	// Clean up every hour
	if time.Since(rl.lastCleanup) < time.Hour {
		return
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour) // Keep records for 24 hours

	// Clean up IP attempts
	for ip, record := range rl.ipAttempts {
		if len(record.attempts) == 0 && record.lockedUntil.Before(now) {
			delete(rl.ipAttempts, ip)
			continue
		}

		// Remove very old attempts
		var kept []time.Time
		for _, attempt := range record.attempts {
			if attempt.After(cutoff) {
				kept = append(kept, attempt)
			}
		}
		record.attempts = kept

		// Remove record if empty and not locked
		if len(record.attempts) == 0 && record.lockedUntil.Before(now) {
			delete(rl.ipAttempts, ip)
		}
	}

	// Clean up email attempts
	for email, record := range rl.emailAttempts {
		if len(record.attempts) == 0 && record.lockedUntil.Before(now) {
			delete(rl.emailAttempts, email)
			continue
		}

		// Remove very old attempts
		var kept []time.Time
		for _, attempt := range record.attempts {
			if attempt.After(cutoff) {
				kept = append(kept, attempt)
			}
		}
		record.attempts = kept

		// Remove record if empty and not locked
		if len(record.attempts) == 0 && record.lockedUntil.Before(now) {
			delete(rl.emailAttempts, email)
		}
	}

	rl.lastCleanup = now
}

// shouldRateLimit determines if an endpoint should be rate limited
func shouldRateLimit(path, method string) bool {
	// Rate limit these POST endpoints
	if method == http.MethodPost {
		switch path {
		case "/login", "/register", "/forgot-password":
			return true
		}
	}
	return false
}

// Database-backed rate limiter for distributed systems

// DBRateLimiter uses the database for rate limiting (works across multiple servers)
type DBRateLimiter struct {
	store            ExtendedUserStore
	maxEmailAttempts int
	maxIPAttempts    int
	windowDuration   time.Duration
	lockoutDuration  time.Duration
}

// NewDBRateLimiter creates a database-backed rate limiter
func NewDBRateLimiter(store ExtendedUserStore) *DBRateLimiter {
	return &DBRateLimiter{
		store:            store,
		maxEmailAttempts: 5,
		maxIPAttempts:    20,
		windowDuration:   15 * time.Minute,
		lockoutDuration:  30 * time.Minute,
	}
}

// CheckRateLimit checks rate limits using the database
func (drl *DBRateLimiter) CheckRateLimit(ctx context.Context, ip, email string) (allowed bool, retryAfter time.Duration, reason string) {
	now := time.Now()
	since := now.Add(-drl.windowDuration)

	// Check IP-based rate limit
	if ip != "" {
		count, err := drl.store.CountRecentIPAttempts(ctx, ip, since)
		if err == nil && count >= drl.maxIPAttempts {
			return false, drl.lockoutDuration, "too many attempts from this IP"
		}
	}

	// Check email-based rate limit
	if email != "" {
		count, err := drl.store.CountRecentLoginAttempts(ctx, email, since)
		if err == nil && count >= drl.maxEmailAttempts {
			// Also check if account is locked
			if user, err := drl.store.ByEmail(ctx, email); err == nil && user.IsLocked() {
				if user.LockedUntil != nil {
					return false, user.LockedUntil.Sub(now), "account temporarily locked"
				}
			}
			return false, drl.lockoutDuration, "too many attempts for this account"
		}
	}

	return true, 0, ""
}

// RecordAttempt records a failed attempt in the database
func (drl *DBRateLimiter) RecordAttempt(ctx context.Context, ip, email string, success bool) error {
	attempt := &LoginAttempt{
		ID:        generateUUID(),
		Email:     email,
		IPAddress: ip,
		Success:   success,
		CreatedAt: time.Now(),
	}

	return drl.store.RecordLoginAttempt(ctx, attempt)
}

// DBRateLimitMiddleware returns middleware using database-backed rate limiting
func DBRateLimitMiddleware(store ExtendedUserStore) buffalo.MiddlewareFunc {
	limiter := NewDBRateLimiter(store)

	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Only rate limit specific auth endpoints
			path := c.Request().URL.Path
			method := c.Request().Method

			if shouldRateLimit(path, method) {
				ctx := c.Request().Context()
				ip := getClientIP(c.Request())

				var email string
				if method == http.MethodPost {
					email = c.Param("email")
				}

				// Check rate limits
				allowed, retryAfter, reason := limiter.CheckRateLimit(ctx, ip, email)

				if !allowed {
					c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
					c.Flash().Add("error", reason)

					// For API requests, return JSON error
					if c.Request().Header.Get("Accept") == "application/json" {
						return c.Render(http.StatusTooManyRequests, r.JSON(map[string]interface{}{
							"error":       reason,
							"retry_after": int(retryAfter.Seconds()),
						}))
					}

					// For regular requests, redirect back with error
					return c.Redirect(http.StatusSeeOther, c.Request().Referer())
				}

				// Record the attempt after the request completes
				defer func() {
					// Buffalo doesn't expose status directly
					// For now, always record the attempt
					// In production, you'd track success via context or wrapper
					success := false
					_ = limiter.RecordAttempt(ctx, ip, email, success)
				}()
			}

			return next(c)
		}
	}
}
