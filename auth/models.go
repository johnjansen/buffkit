package auth

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"time"
)

// User represents an authenticated user in the system.
// This is the core user model with all authentication-related fields.
type User struct {
	// Core identification
	ID             string `json:"id" db:"id"`
	Email          string `json:"email" db:"email"`
	PasswordDigest string `json:"-" db:"password_digest"` // Never expose in JSON

	// Profile information
	FirstName   string  `json:"first_name,omitempty" db:"first_name"`
	LastName    string  `json:"last_name,omitempty" db:"last_name"`
	DisplayName string  `json:"display_name,omitempty" db:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty" db:"avatar_url"`

	// Account status
	IsActive   bool `json:"is_active" db:"is_active"`
	IsVerified bool `json:"is_verified" db:"is_verified"`
	IsAdmin    bool `json:"is_admin" db:"is_admin"`

	// Email verification
	EmailVerifiedAt         *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	EmailVerificationToken  *string    `json:"-" db:"email_verification_token"`
	EmailVerificationSentAt *time.Time `json:"-" db:"email_verification_sent_at"`

	// Password reset
	PasswordResetToken  *string    `json:"-" db:"password_reset_token"`
	PasswordResetSentAt *time.Time `json:"-" db:"password_reset_sent_at"`

	// Security tracking
	FailedLoginAttempts int        `json:"-" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"-" db:"locked_until"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP         *string    `json:"-" db:"last_login_ip"`

	// Two-factor authentication (preparation for future)
	TOTPSecret    *string `json:"-" db:"totp_secret"`
	TOTPEnabled   bool    `json:"totp_enabled" db:"totp_enabled"`
	RecoveryCodes *string `json:"-" db:"recovery_codes"`

	// Extensible metadata
	Extra UserExtra `json:"extra,omitempty" db:"extra"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserExtra holds additional user metadata that can be extended
// without changing the database schema.
type UserExtra map[string]interface{}

// Value implements driver.Valuer for database storage
func (e UserExtra) Value() (driver.Value, error) {
	if e == nil {
		return "{}", nil
	}
	return json.Marshal(e)
}

// Scan implements sql.Scanner for database retrieval
func (e *UserExtra) Scan(value interface{}) error {
	if value == nil {
		*e = make(UserExtra)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, e)
	case string:
		return json.Unmarshal([]byte(v), e)
	default:
		*e = make(UserExtra)
		return nil
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.DisplayName
	}
	if u.FirstName == "" {
		return u.LastName
	}
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

// Name returns the best available name for display
func (u *User) Name() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.FullName()
}

// IsLocked returns true if the account is currently locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return u.LockedUntil.After(time.Now())
}

// CanLogin returns true if the user can currently log in
func (u *User) CanLogin() bool {
	return u.IsActive && !u.IsLocked()
}

// NeedsEmailVerification returns true if email verification is required
func (u *User) NeedsEmailVerification() bool {
	return !u.IsVerified && u.EmailVerifiedAt == nil
}

// Session represents an active user session
type Session struct {
	ID     string `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`
	Token  string `json:"token" db:"token"`

	// Session context
	IPAddress string `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string `json:"user_agent,omitempty" db:"user_agent"`

	// Expiry management
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	LastActivityAt time.Time `json:"last_activity_at" db:"last_activity_at"`

	// Additional session data
	Data SessionData `json:"data,omitempty" db:"data"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Associated user (for joins)
	User *User `json:"user,omitempty" db:"-"`
}

// SessionData holds additional session metadata
type SessionData map[string]interface{}

// Value implements driver.Valuer for database storage
func (d SessionData) Value() (driver.Value, error) {
	if d == nil {
		return "{}", nil
	}
	return json.Marshal(d)
}

// Scan implements sql.Scanner for database retrieval
func (d *SessionData) Scan(value interface{}) error {
	if value == nil {
		*d = make(SessionData)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		*d = make(SessionData)
		return nil
	}
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now())
}

// IsStale returns true if the session hasn't been active recently
func (s *Session) IsStale(maxInactivity time.Duration) bool {
	return time.Since(s.LastActivityAt) > maxInactivity
}

// AuditLog represents an authentication audit log entry
type AuditLog struct {
	ID     string  `json:"id" db:"id"`
	UserID *string `json:"user_id,omitempty" db:"user_id"`

	// Event information
	EventType   string `json:"event_type" db:"event_type"`     // login, logout, register, etc.
	EventStatus string `json:"event_status" db:"event_status"` // success, failure

	// Context
	IPAddress string `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string `json:"user_agent,omitempty" db:"user_agent"`

	// Additional information
	Metadata     AuditMetadata `json:"metadata,omitempty" db:"metadata"`
	ErrorMessage *string       `json:"error_message,omitempty" db:"error_message"`

	// Timestamp
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuditMetadata holds additional audit log metadata
type AuditMetadata map[string]interface{}

// Value implements driver.Valuer for database storage
func (m AuditMetadata) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for database retrieval
func (m *AuditMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(AuditMetadata)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		*m = make(AuditMetadata)
		return nil
	}
}

// LoginAttempt tracks login attempts for rate limiting
type LoginAttempt struct {
	ID        string `json:"id" db:"id"`
	Email     string `json:"email" db:"email"`
	IPAddress string `json:"ip_address" db:"ip_address"`

	// Attempt result
	Success bool `json:"success" db:"success"`

	// Context
	UserAgent string `json:"user_agent,omitempty" db:"user_agent"`

	// Timestamp
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserDevice represents a trusted device for a user
type UserDevice struct {
	ID     string `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`

	// Device identification
	DeviceName        string `json:"device_name" db:"device_name"`
	DeviceFingerprint string `json:"device_fingerprint" db:"device_fingerprint"`

	// Device details
	Platform  string `json:"platform,omitempty" db:"platform"`
	Browser   string `json:"browser,omitempty" db:"browser"`
	IPAddress string `json:"ip_address,omitempty" db:"ip_address"`

	// Trust status
	IsTrusted  bool       `json:"is_trusted" db:"is_trusted"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Registration holds data for user registration
type Registration struct {
	Email                string `json:"email" validate:"required,email"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
	FirstName            string `json:"first_name" validate:"max=100"`
	LastName             string `json:"last_name" validate:"max=100"`
	AcceptTerms          bool   `json:"accept_terms" validate:"required"`
}

// PasswordReset holds data for password reset requests
type PasswordReset struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordUpdate holds data for updating a password
type PasswordUpdate struct {
	Token                string `json:"token" validate:"required"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
}

// ProfileUpdate holds data for updating user profile
type ProfileUpdate struct {
	FirstName   string  `json:"first_name" validate:"max=100"`
	LastName    string  `json:"last_name" validate:"max=100"`
	DisplayName string  `json:"display_name" validate:"max=100"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty,url"`
}

// EmailVerification holds data for email verification
type EmailVerification struct {
	Token string `json:"token" validate:"required"`
}

// AuthEvent represents types of authentication events for audit logging
type AuthEvent string

const (
	EventLogin             AuthEvent = "login"
	EventLogout            AuthEvent = "logout"
	EventRegister          AuthEvent = "register"
	EventPasswordReset     AuthEvent = "password_reset"
	EventPasswordUpdate    AuthEvent = "password_update"
	EventEmailVerification AuthEvent = "email_verification"
	EventProfileUpdate     AuthEvent = "profile_update"
	EventAccountLocked     AuthEvent = "account_locked"
	EventAccountUnlocked   AuthEvent = "account_unlocked"
	EventSessionCreated    AuthEvent = "session_created"
	EventSessionDestroyed  AuthEvent = "session_destroyed"
	EventDeviceTrusted     AuthEvent = "device_trusted"
	EventDeviceRemoved     AuthEvent = "device_removed"
	EventTwoFactorEnabled  AuthEvent = "two_factor_enabled"
	EventTwoFactorDisabled AuthEvent = "two_factor_disabled"
)

// AuthStatus represents the status of an authentication event
type AuthStatus string

const (
	StatusSuccess AuthStatus = "success"
	StatusFailure AuthStatus = "failure"
	StatusPending AuthStatus = "pending"
)

// ExtendedUserStore interface with all enhanced authentication features
type ExtendedUserStore interface {
	UserStore // Embed the basic interface

	// User management
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Count(ctx context.Context) (int64, error)

	// Email verification
	SetEmailVerificationToken(ctx context.Context, id, token string) error
	VerifyEmail(ctx context.Context, token string) (*User, error)
	ResendVerificationEmail(ctx context.Context, email string) error

	// Password reset
	SetPasswordResetToken(ctx context.Context, email, token string) error
	ResetPassword(ctx context.Context, token, newPasswordDigest string) error
	ValidateResetToken(ctx context.Context, token string) (*User, error)

	// Security
	IncrementFailedLoginAttempts(ctx context.Context, email string) error
	ResetFailedLoginAttempts(ctx context.Context, email string) error
	LockAccount(ctx context.Context, email string, until time.Time) error
	UnlockAccount(ctx context.Context, email string) error

	// Sessions
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	UpdateSessionActivity(ctx context.Context, token string) error
	DeleteSession(ctx context.Context, token string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	ListUserSessions(ctx context.Context, userID string) ([]*Session, error)

	// Audit logging
	LogAuthEvent(ctx context.Context, log *AuditLog) error
	GetUserAuditLogs(ctx context.Context, userID string, limit int) ([]*AuditLog, error)

	// Device management
	RegisterDevice(ctx context.Context, device *UserDevice) error
	TrustDevice(ctx context.Context, deviceID string) error
	RemoveDevice(ctx context.Context, deviceID string) error
	ListUserDevices(ctx context.Context, userID string) ([]*UserDevice, error)

	// Login attempts (for rate limiting)
	RecordLoginAttempt(ctx context.Context, attempt *LoginAttempt) error
	CountRecentLoginAttempts(ctx context.Context, email string, since time.Time) (int, error)
	CountRecentIPAttempts(ctx context.Context, ip string, since time.Time) (int, error)
}
