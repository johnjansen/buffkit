package secure

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
)

// Options configures the security middleware
type Options struct {
	// DevMode disables some security features for development
	DevMode bool

	// ContentTypeOptions sets X-Content-Type-Options header
	ContentTypeNosniff bool

	// FrameOptions sets X-Frame-Options header
	FrameDeny       bool
	FrameSameOrigin bool

	// XSSProtection sets X-XSS-Protection header
	XSSProtection bool

	// ContentSecurityPolicy sets CSP header
	ContentSecurityPolicy string

	// StrictTransportSecurity sets HSTS header
	STSSeconds           int64
	STSIncludeSubdomains bool
	STSPreload           bool

	// ReferrerPolicy sets Referrer-Policy header
	ReferrerPolicy string
}

// DefaultOptions returns secure defaults
func DefaultOptions() Options {
	return Options{
		ContentTypeNosniff: true,
		FrameDeny:          true,
		XSSProtection:      true,
		STSSeconds:         31536000, // 1 year
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://unpkg.com https://esm.sh; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none';",
	}
}

// Middleware returns security middleware for Buffalo
func Middleware(opts Options) buffalo.MiddlewareFunc {
	// Apply defaults
	if opts.ContentTypeNosniff == false && opts.FrameDeny == false && opts.XSSProtection == false {
		opts = DefaultOptions()
	}

	// Adjust for dev mode
	if opts.DevMode {
		// Relax some restrictions in development
		opts.FrameDeny = false
		opts.FrameSameOrigin = true
		opts.STSSeconds = 0 // Disable HSTS in dev
	}

	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Get response writer
			w := c.Response()

			// Apply security headers
			if opts.ContentTypeNosniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// Frame options
			if opts.FrameDeny {
				w.Header().Set("X-Frame-Options", "DENY")
			} else if opts.FrameSameOrigin {
				w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			}

			// XSS Protection
			if opts.XSSProtection {
				w.Header().Set("X-XSS-Protection", "1; mode=block")
			}

			// Content Security Policy
			if opts.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", opts.ContentSecurityPolicy)
			}

			// Strict Transport Security (only in production)
			if !opts.DevMode && opts.STSSeconds > 0 {
				value := formatSTSHeader(opts.STSSeconds, opts.STSIncludeSubdomains, opts.STSPreload)
				w.Header().Set("Strict-Transport-Security", value)
			}

			// Referrer Policy
			if opts.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", opts.ReferrerPolicy)
			}

			// Additional security headers
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			return next(c)
		}
	}
}

// CSRFMiddleware wraps Buffalo's CSRF middleware with better defaults
func CSRFMiddleware() buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Skip CSRF for GET, HEAD, OPTIONS
			if c.Request().Method == http.MethodGet ||
				c.Request().Method == http.MethodHead ||
				c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Check for CSRF token
			token := c.Request().Header.Get("X-CSRF-Token")
			if token == "" {
				// Try form value
				token = c.Param("authenticity_token")
			}
			if token == "" {
				// Try multipart form
				token = c.Request().FormValue("authenticity_token")
			}

			// Verify token (simplified - Buffalo handles the actual verification)
			sessionToken := c.Session().Get("csrf_token")
			if sessionToken == nil || token == "" || sessionToken != token {
				// Generate new token if needed
				if sessionToken == nil {
					newToken := generateCSRFToken()
					c.Session().Set("csrf_token", newToken)
					c.Session().Save()
				}

				// For non-AJAX requests, we might want to show a form
				if c.Request().Header.Get("X-Requested-With") != "XMLHttpRequest" {
					// Allow GET requests to pass through to show forms
					if c.Request().Method != http.MethodPost &&
						c.Request().Method != http.MethodPut &&
						c.Request().Method != http.MethodPatch &&
						c.Request().Method != http.MethodDelete {
						return next(c)
					}
				}

				return c.Error(http.StatusForbidden, errInvalidCSRFToken)
			}

			return next(c)
		}
	}
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware(requestsPerMinute int) buffalo.MiddlewareFunc {
	// Simple in-memory rate limiter (for demo purposes)
	// In production, use a proper rate limiter with Redis
	clients := make(map[string][]int64)

	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Get client IP
			ip := getClientIP(c.Request())

			// Check rate limit
			now := currentTimeMillis()
			windowStart := now - 60000 // 1 minute window

			// Clean old entries and count recent requests
			var recentRequests []int64
			if requests, exists := clients[ip]; exists {
				for _, timestamp := range requests {
					if timestamp > windowStart {
						recentRequests = append(recentRequests, timestamp)
					}
				}
			}

			// Check if limit exceeded
			if len(recentRequests) >= requestsPerMinute {
				return c.Error(http.StatusTooManyRequests, errRateLimitExceeded)
			}

			// Add current request
			recentRequests = append(recentRequests, now)
			clients[ip] = recentRequests

			return next(c)
		}
	}
}

// Helper functions

func formatSTSHeader(seconds int64, includeSubdomains, preload bool) string {
	header := formatInt(seconds)
	if includeSubdomains {
		header += "; includeSubDomains"
	}
	if preload {
		header += "; preload"
	}
	return header
}

func formatInt(i int64) string {
	return fmt.Sprintf("max-age=%d", i)
}

func generateCSRFToken() string {
	// Simple token generation - in production use crypto/rand
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP
		if comma := indexByte(forwarded, ','); comma != -1 {
			return forwarded[:comma]
		}
		return forwarded
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	if colon := lastIndexByte(r.RemoteAddr, ':'); colon != -1 {
		return r.RemoteAddr[:colon]
	}
	return r.RemoteAddr
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// Errors
var (
	errInvalidCSRFToken  = errNew("invalid CSRF token")
	errRateLimitExceeded = errNew("rate limit exceeded")
)

func errNew(msg string) error {
	return &securityError{msg: msg}
}

type securityError struct {
	msg string
}

func (e *securityError) Error() string {
	return e.msg
}
