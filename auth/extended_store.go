package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Ensure SQLStore implements ExtendedUserStore
var _ ExtendedUserStore = (*SQLStore)(nil)

// Update updates user fields
func (s *SQLStore) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic UPDATE query
	query := "UPDATE users SET "
	args := []interface{}{}
	i := 1

	for field, value := range updates {
		if i > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", field, i)
		args = append(args, value)
		i++
	}

	query += fmt.Sprintf(" WHERE id = $%d", i)
	args = append(args, id)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete removes a user
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM users WHERE id = $1"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List returns paginated users
func (s *SQLStore) List(ctx context.Context, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, password_digest,
		       first_name, last_name, display_name,
		       is_active, is_verified, is_admin,
		       created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		u := &User{}
		err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordDigest,
			&u.FirstName, &u.LastName, &u.DisplayName,
			&u.IsActive, &u.IsVerified, &u.IsAdmin,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

// Count returns total user count
func (s *SQLStore) Count(ctx context.Context) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM users"
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// SetEmailVerificationToken sets the email verification token
func (s *SQLStore) SetEmailVerificationToken(ctx context.Context, id, token string) error {
	query := `
		UPDATE users
		SET email_verification_token = $2,
		    email_verification_sent_at = $3,
		    updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, id, token, now, now)
	if err != nil {
		return fmt.Errorf("failed to set verification token: %w", err)
	}

	return nil
}

// VerifyEmail verifies a user's email
func (s *SQLStore) VerifyEmail(ctx context.Context, token string) (*User, error) {
	// First, find the user with this token
	var userID string
	var sentAt *time.Time

	query := `
		SELECT id, email_verification_sent_at
		FROM users
		WHERE email_verification_token = $1
	`

	err := s.db.QueryRowContext(ctx, query, token).Scan(&userID, &sentAt)
	if err == sql.ErrNoRows {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find verification token: %w", err)
	}

	// Check if token has expired (24 hours)
	if sentAt != nil && time.Since(*sentAt) > 24*time.Hour {
		return nil, ErrTokenExpired
	}

	// Update the user as verified
	now := time.Now()
	updateQuery := `
		UPDATE users
		SET is_verified = true,
		    email_verified_at = $2,
		    email_verification_token = NULL,
		    email_verification_sent_at = NULL,
		    updated_at = $3
		WHERE id = $1
	`

	_, err = s.db.ExecContext(ctx, updateQuery, userID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	// Return the updated user
	return s.ByID(ctx, userID)
}

// ResendVerificationEmail generates a new verification token
func (s *SQLStore) ResendVerificationEmail(ctx context.Context, email string) error {
	// Check if user exists and is not verified
	var userID string
	var isVerified bool

	query := "SELECT id, is_verified FROM users WHERE email = $1"
	err := s.db.QueryRowContext(ctx, query, email).Scan(&userID, &isVerified)
	if err == sql.ErrNoRows {
		// Don't reveal if email exists
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}

	if isVerified {
		// Already verified, no need to resend
		return nil
	}

	// Generate new token
	token := generateToken()
	return s.SetEmailVerificationToken(ctx, userID, token)
}

// SetPasswordResetToken sets the password reset token
func (s *SQLStore) SetPasswordResetToken(ctx context.Context, email, token string) error {
	query := `
		UPDATE users
		SET password_reset_token = $2,
		    password_reset_sent_at = $3,
		    updated_at = $4
		WHERE email = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, email, token, now, now)
	if err != nil {
		return fmt.Errorf("failed to set reset token: %w", err)
	}

	return nil
}

// ResetPassword resets a user's password using a token
func (s *SQLStore) ResetPassword(ctx context.Context, token, newPasswordDigest string) error {
	// Validate token and get user
	user, err := s.ValidateResetToken(ctx, token)
	if err != nil {
		return err
	}

	// Update password and clear reset token
	query := `
		UPDATE users
		SET password_digest = $2,
		    password_reset_token = NULL,
		    password_reset_sent_at = NULL,
		    failed_login_attempts = 0,
		    locked_until = NULL,
		    updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	_, err = s.db.ExecContext(ctx, query, user.ID, newPasswordDigest, now)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// ValidateResetToken validates a password reset token
func (s *SQLStore) ValidateResetToken(ctx context.Context, token string) (*User, error) {
	var userID string
	var sentAt *time.Time

	query := `
		SELECT id, password_reset_sent_at
		FROM users
		WHERE password_reset_token = $1
	`

	err := s.db.QueryRowContext(ctx, query, token).Scan(&userID, &sentAt)
	if err == sql.ErrNoRows {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate reset token: %w", err)
	}

	// Check if token has expired (1 hour)
	if sentAt != nil && time.Since(*sentAt) > time.Hour {
		return nil, ErrTokenExpired
	}

	return s.ByID(ctx, userID)
}

// IncrementFailedLoginAttempts increments the failed login counter
func (s *SQLStore) IncrementFailedLoginAttempts(ctx context.Context, email string) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts >= 4 THEN $2
		        ELSE locked_until
		    END,
		    updated_at = $3
		WHERE email = $1
	`

	lockTime := time.Now().Add(30 * time.Minute)
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, email, lockTime, now)
	if err != nil {
		return fmt.Errorf("failed to increment login attempts: %w", err)
	}

	return nil
}

// ResetFailedLoginAttempts resets the failed login counter
func (s *SQLStore) ResetFailedLoginAttempts(ctx context.Context, email string) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until = NULL,
		    last_login_at = $2,
		    updated_at = $3
		WHERE email = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, email, now, now)
	if err != nil {
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}

	return nil
}

// LockAccount locks a user account until a specific time
func (s *SQLStore) LockAccount(ctx context.Context, email string, until time.Time) error {
	query := `
		UPDATE users
		SET locked_until = $2,
		    updated_at = $3
		WHERE email = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, email, until, now)
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}

	return nil
}

// UnlockAccount unlocks a user account
func (s *SQLStore) UnlockAccount(ctx context.Context, email string) error {
	query := `
		UPDATE users
		SET locked_until = NULL,
		    failed_login_attempts = 0,
		    updated_at = $2
		WHERE email = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, email, now)
	if err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}

	return nil
}

// CreateSession creates a new session
func (s *SQLStore) CreateSession(ctx context.Context, session *Session) error {
	if session.ID == "" {
		session.ID = generateUUID()
	}
	if session.Token == "" {
		session.Token = generateToken()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO sessions (
			id, user_id, token,
			ip_address, user_agent,
			expires_at, last_activity_at,
			data, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	dataJSON, _ := json.Marshal(session.Data)

	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.Token,
		session.IPAddress, session.UserAgent,
		session.ExpiresAt, session.LastActivityAt,
		dataJSON, session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (s *SQLStore) GetSession(ctx context.Context, token string) (*Session, error) {
	session := &Session{}
	var dataJSON []byte

	query := `
		SELECT id, user_id, token,
		       ip_address, user_agent,
		       expires_at, last_activity_at,
		       data, created_at
		FROM sessions
		WHERE token = $1
	`

	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID, &session.UserID, &session.Token,
		&session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActivityAt,
		&dataJSON, &session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if len(dataJSON) > 0 {
		json.Unmarshal(dataJSON, &session.Data)
	}

	return session, nil
}

// UpdateSessionActivity updates the last activity timestamp
func (s *SQLStore) UpdateSessionActivity(ctx context.Context, token string) error {
	query := `
		UPDATE sessions
		SET last_activity_at = $2
		WHERE token = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, token, now)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	return nil
}

// DeleteSession deletes a session
func (s *SQLStore) DeleteSession(ctx context.Context, token string) error {
	query := "DELETE FROM sessions WHERE token = $1"
	_, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (s *SQLStore) DeleteUserSessions(ctx context.Context, userID string) error {
	query := "DELETE FROM sessions WHERE user_id = $1"
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// ListUserSessions lists all sessions for a user
func (s *SQLStore) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	query := `
		SELECT id, user_id, token,
		       ip_address, user_agent,
		       expires_at, last_activity_at,
		       data, created_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY last_activity_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []*Session{}
	for rows.Next() {
		session := &Session{}
		var dataJSON []byte

		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token,
			&session.IPAddress, &session.UserAgent,
			&session.ExpiresAt, &session.LastActivityAt,
			&dataJSON, &session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &session.Data)
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// LogAuthEvent logs an authentication event
func (s *SQLStore) LogAuthEvent(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = generateUUID()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO auth_audit_logs (
			id, user_id,
			event_type, event_status,
			ip_address, user_agent,
			metadata, error_message,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	metadataJSON, _ := json.Marshal(log.Metadata)

	_, err := s.db.ExecContext(ctx, query,
		log.ID, log.UserID,
		log.EventType, log.EventStatus,
		log.IPAddress, log.UserAgent,
		metadataJSON, log.ErrorMessage,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to log auth event: %w", err)
	}

	return nil
}

// GetUserAuditLogs retrieves audit logs for a user
func (s *SQLStore) GetUserAuditLogs(ctx context.Context, userID string, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, user_id,
		       event_type, event_status,
		       ip_address, user_agent,
		       metadata, error_message,
		       created_at
		FROM auth_audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*AuditLog{}
	for rows.Next() {
		log := &AuditLog{}
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID, &log.UserID,
			&log.EventType, &log.EventStatus,
			&log.IPAddress, &log.UserAgent,
			&metadataJSON, &log.ErrorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &log.Metadata)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// RegisterDevice registers a new device for a user
func (s *SQLStore) RegisterDevice(ctx context.Context, device *UserDevice) error {
	if device.ID == "" {
		device.ID = generateUUID()
	}
	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	device.LastSeenAt = &now

	query := `
		INSERT INTO user_devices (
			id, user_id,
			device_name, device_fingerprint,
			platform, browser, ip_address,
			is_trusted, last_seen_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (device_fingerprint) DO UPDATE
		SET last_seen_at = $9,
		    ip_address = $7,
		    updated_at = $11
	`

	_, err := s.db.ExecContext(ctx, query,
		device.ID, device.UserID,
		device.DeviceName, device.DeviceFingerprint,
		device.Platform, device.Browser, device.IPAddress,
		device.IsTrusted, device.LastSeenAt,
		device.CreatedAt, device.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}

	return nil
}

// TrustDevice marks a device as trusted
func (s *SQLStore) TrustDevice(ctx context.Context, deviceID string) error {
	query := `
		UPDATE user_devices
		SET is_trusted = true,
		    updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, deviceID, now)
	if err != nil {
		return fmt.Errorf("failed to trust device: %w", err)
	}

	return nil
}

// RemoveDevice removes a device
func (s *SQLStore) RemoveDevice(ctx context.Context, deviceID string) error {
	query := "DELETE FROM user_devices WHERE id = $1"
	_, err := s.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to remove device: %w", err)
	}
	return nil
}

// ListUserDevices lists all devices for a user
func (s *SQLStore) ListUserDevices(ctx context.Context, userID string) ([]*UserDevice, error) {
	query := `
		SELECT id, user_id,
		       device_name, device_fingerprint,
		       platform, browser, ip_address,
		       is_trusted, last_seen_at,
		       created_at, updated_at
		FROM user_devices
		WHERE user_id = $1
		ORDER BY last_seen_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := []*UserDevice{}
	for rows.Next() {
		device := &UserDevice{}
		err := rows.Scan(
			&device.ID, &device.UserID,
			&device.DeviceName, &device.DeviceFingerprint,
			&device.Platform, &device.Browser, &device.IPAddress,
			&device.IsTrusted, &device.LastSeenAt,
			&device.CreatedAt, &device.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// RecordLoginAttempt records a login attempt
func (s *SQLStore) RecordLoginAttempt(ctx context.Context, attempt *LoginAttempt) error {
	if attempt.ID == "" {
		attempt.ID = generateUUID()
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO login_attempts (
			id, email, ip_address,
			success, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query,
		attempt.ID, attempt.Email, attempt.IPAddress,
		attempt.Success, attempt.UserAgent, attempt.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record login attempt: %w", err)
	}

	return nil
}

// CountRecentLoginAttempts counts recent login attempts for an email
func (s *SQLStore) CountRecentLoginAttempts(ctx context.Context, email string, since time.Time) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE email = $1 AND created_at > $2 AND success = false
	`

	err := s.db.QueryRowContext(ctx, query, email, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count login attempts: %w", err)
	}

	return count, nil
}

// CountRecentIPAttempts counts recent login attempts from an IP
func (s *SQLStore) CountRecentIPAttempts(ctx context.Context, ip string, since time.Time) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE ip_address = $1 AND created_at > $2 AND success = false
	`

	err := s.db.QueryRowContext(ctx, query, ip, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count IP attempts: %w", err)
	}

	return count, nil
}
