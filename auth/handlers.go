package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/validate"
)

// RegistrationFormHandler renders the registration form
func RegistrationFormHandler(c buffalo.Context) error {
	// Create empty registration form
	reg := &Registration{}
	c.Set("registration", reg)
	c.Set("errors", validate.NewErrors())

	return c.Render(http.StatusOK, r.HTML("auth/register.plush.html"))
}

// RegistrationHandler processes user registration
func RegistrationHandler(c buffalo.Context) error {
	reg := &Registration{}
	if err := c.Bind(reg); err != nil {
		return c.Error(http.StatusBadRequest, err)
	}

	// Validate registration data
	errors := validate.NewErrors()

	// Check email format
	if !isValidEmail(reg.Email) {
		errors.Add("email", "Invalid email address")
	}

	// Check password strength
	if len(reg.Password) < 8 {
		errors.Add("password", "Password must be at least 8 characters")
	}

	// Check password confirmation
	if reg.Password != reg.PasswordConfirmation {
		errors.Add("password_confirmation", "Passwords do not match")
	}

	// Check terms acceptance
	if !reg.AcceptTerms {
		errors.Add("accept_terms", "You must accept the terms and conditions")
	}

	if errors.HasAny() {
		c.Set("registration", reg)
		c.Set("errors", errors)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("auth/register.plush.html"))
	}

	// Get the store
	store := GetStore()
	if store == nil {
		return c.Error(http.StatusInternalServerError, fmt.Errorf("auth store not configured"))
	}

	// Check if email already exists
	exists, err := store.ExistsEmail(c.Request().Context(), reg.Email)
	if err != nil {
		return c.Error(http.StatusInternalServerError, err)
	}
	if exists {
		errors.Add("email", "Email address is already registered")
		c.Set("registration", reg)
		c.Set("errors", errors)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("auth/register.plush.html"))
	}

	// Hash the password
	passwordDigest, err := HashPassword(reg.Password)
	if err != nil {
		return c.Error(http.StatusInternalServerError, err)
	}

	// Create the user
	user := &User{
		ID:             generateUUID(),
		Email:          strings.ToLower(strings.TrimSpace(reg.Email)),
		PasswordDigest: passwordDigest,
		FirstName:      reg.FirstName,
		LastName:       reg.LastName,
		IsActive:       true,
		IsVerified:     false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Generate email verification token
	verificationToken := generateToken()
	user.EmailVerificationToken = &verificationToken
	now := time.Now()
	user.EmailVerificationSentAt = &now

	// Create the user in the store
	if err := store.Create(c.Request().Context(), user); err != nil {
		if err == ErrUserExists {
			errors.Add("email", "Email address is already registered")
			c.Set("registration", reg)
			c.Set("errors", errors)
			return c.Render(http.StatusUnprocessableEntity, r.HTML("auth/register.plush.html"))
		}
		return c.Error(http.StatusInternalServerError, err)
	}

	// Log the registration event
	if extStore, ok := store.(ExtendedUserStore); ok {
		audit := &AuditLog{
			ID:          generateUUID(),
			UserID:      &user.ID,
			EventType:   string(EventRegister),
			EventStatus: string(StatusSuccess),
			IPAddress:   getClientIP(c.Request()),
			UserAgent:   c.Request().UserAgent(),
			CreatedAt:   time.Now(),
		}
		_ = extStore.LogAuthEvent(c.Request().Context(), audit)
	}

	// Send verification email (if mail sender is configured)
	sendVerificationEmail(c, user, verificationToken)

	// Set flash message
	c.Flash().Add("success", "Registration successful! Please check your email to verify your account.")

	// Redirect to login
	return c.Redirect(http.StatusSeeOther, "/login")
}

// EmailVerificationHandler verifies a user's email address
func EmailVerificationHandler(c buffalo.Context) error {
	token := c.Param("token")
	if token == "" {
		c.Flash().Add("error", "Invalid verification link")
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	store := GetStore()
	if store == nil {
		return c.Error(http.StatusInternalServerError, fmt.Errorf("auth store not configured"))
	}

	// Verify the email using extended store if available
	if extStore, ok := store.(ExtendedUserStore); ok {
		user, err := extStore.VerifyEmail(c.Request().Context(), token)
		if err != nil {
			c.Flash().Add("error", "Invalid or expired verification link")
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		// Log the verification event
		audit := &AuditLog{
			ID:          generateUUID(),
			UserID:      &user.ID,
			EventType:   string(EventEmailVerification),
			EventStatus: string(StatusSuccess),
			IPAddress:   getClientIP(c.Request()),
			UserAgent:   c.Request().UserAgent(),
			CreatedAt:   time.Now(),
		}
		_ = extStore.LogAuthEvent(c.Request().Context(), audit)

		c.Flash().Add("success", "Email verified successfully! You can now log in.")
	} else {
		c.Flash().Add("error", "Email verification not supported")
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

// ForgotPasswordFormHandler renders the forgot password form
func ForgotPasswordFormHandler(c buffalo.Context) error {
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("auth/forgot_password.plush.html"))
}

// ForgotPasswordHandler processes forgot password requests
func ForgotPasswordHandler(c buffalo.Context) error {
	email := c.Param("email")
	errors := validate.NewErrors()

	if !isValidEmail(email) {
		errors.Add("email", "Invalid email address")
		c.Set("errors", errors)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("auth/forgot_password.plush.html"))
	}

	store := GetStore()
	if extStore, ok := store.(ExtendedUserStore); ok {
		// Generate reset token
		resetToken := generateToken()

		// Set the reset token (this will fail silently if email doesn't exist)
		_ = extStore.SetPasswordResetToken(c.Request().Context(), email, resetToken)

		// Send reset email (if user exists and mail is configured)
		if user, err := store.ByEmail(c.Request().Context(), email); err == nil {
			sendPasswordResetEmail(c, user, resetToken)

			// Log the event
			audit := &AuditLog{
				ID:          generateUUID(),
				UserID:      &user.ID,
				EventType:   string(EventPasswordReset),
				EventStatus: string(StatusSuccess),
				IPAddress:   getClientIP(c.Request()),
				UserAgent:   c.Request().UserAgent(),
				CreatedAt:   time.Now(),
			}
			_ = extStore.LogAuthEvent(c.Request().Context(), audit)
		}
	}

	// Always show success message (don't reveal if email exists)
	c.Flash().Add("success", "If your email is registered, you will receive password reset instructions.")
	return c.Redirect(http.StatusSeeOther, "/login")
}

// ResetPasswordFormHandler renders the password reset form
func ResetPasswordFormHandler(c buffalo.Context) error {
	token := c.Param("token")
	if token == "" {
		c.Flash().Add("error", "Invalid reset link")
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	c.Set("token", token)
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("auth/reset_password.plush.html"))
}

// ResetPasswordHandler processes password reset
func ResetPasswordHandler(c buffalo.Context) error {
	update := &PasswordUpdate{}
	if err := c.Bind(update); err != nil {
		return c.Error(http.StatusBadRequest, err)
	}

	errors := validate.NewErrors()

	// Validate password
	if len(update.Password) < 8 {
		errors.Add("password", "Password must be at least 8 characters")
	}

	if update.Password != update.PasswordConfirmation {
		errors.Add("password_confirmation", "Passwords do not match")
	}

	if errors.HasAny() {
		c.Set("token", update.Token)
		c.Set("errors", errors)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("auth/reset_password.plush.html"))
	}

	store := GetStore()
	if extStore, ok := store.(ExtendedUserStore); ok {
		// Validate token first
		user, err := extStore.ValidateResetToken(c.Request().Context(), update.Token)
		if err != nil {
			c.Flash().Add("error", "Invalid or expired reset token")
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		// Hash new password
		passwordDigest, err := HashPassword(update.Password)
		if err != nil {
			return c.Error(http.StatusInternalServerError, err)
		}

		// Reset the password
		if err := extStore.ResetPassword(c.Request().Context(), update.Token, passwordDigest); err != nil {
			c.Flash().Add("error", "Failed to reset password")
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		// Log the event
		audit := &AuditLog{
			ID:          generateUUID(),
			UserID:      &user.ID,
			EventType:   string(EventPasswordUpdate),
			EventStatus: string(StatusSuccess),
			IPAddress:   getClientIP(c.Request()),
			UserAgent:   c.Request().UserAgent(),
			CreatedAt:   time.Now(),
		}
		_ = extStore.LogAuthEvent(c.Request().Context(), audit)

		c.Flash().Add("success", "Password reset successfully! You can now log in.")
	} else {
		c.Flash().Add("error", "Password reset not supported")
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

// ProfileHandler shows the user profile
func ProfileHandler(c buffalo.Context) error {
	user := CurrentUser(c)
	if user == nil {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	c.Set("user", user)
	return c.Render(http.StatusOK, r.HTML("auth/profile.plush.html"))
}

// ProfileUpdateHandler updates user profile
func ProfileUpdateHandler(c buffalo.Context) error {
	user := CurrentUser(c)
	if user == nil {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	update := &ProfileUpdate{}
	if err := c.Bind(update); err != nil {
		return c.Error(http.StatusBadRequest, err)
	}

	store := GetStore()
	if extStore, ok := store.(ExtendedUserStore); ok {
		updates := map[string]interface{}{
			"first_name":   update.FirstName,
			"last_name":    update.LastName,
			"display_name": update.DisplayName,
			"updated_at":   time.Now(),
		}

		if update.AvatarURL != nil {
			updates["avatar_url"] = *update.AvatarURL
		}

		if err := extStore.Update(c.Request().Context(), user.ID, updates); err != nil {
			c.Flash().Add("error", "Failed to update profile")
			return c.Redirect(http.StatusSeeOther, "/profile")
		}

		// Log the event
		audit := &AuditLog{
			ID:          generateUUID(),
			UserID:      &user.ID,
			EventType:   string(EventProfileUpdate),
			EventStatus: string(StatusSuccess),
			IPAddress:   getClientIP(c.Request()),
			UserAgent:   c.Request().UserAgent(),
			CreatedAt:   time.Now(),
		}
		_ = extStore.LogAuthEvent(c.Request().Context(), audit)

		c.Flash().Add("success", "Profile updated successfully!")
	}

	return c.Redirect(http.StatusSeeOther, "/profile")
}

// SessionsHandler shows active sessions
func SessionsHandler(c buffalo.Context) error {
	user := CurrentUser(c)
	if user == nil {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	store := GetStore()
	if extStore, ok := store.(ExtendedUserStore); ok {
		sessions, err := extStore.ListUserSessions(c.Request().Context(), user.ID)
		if err != nil {
			c.Set("sessions", []*Session{})
		} else {
			c.Set("sessions", sessions)
		}
	} else {
		c.Set("sessions", []*Session{})
	}

	c.Set("user", user)
	return c.Render(http.StatusOK, r.HTML("auth/sessions.plush.html"))
}

// RevokeSessionHandler revokes a specific session
func RevokeSessionHandler(c buffalo.Context) error {
	user := CurrentUser(c)
	if user == nil {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	sessionID := c.Param("session_id")
	store := GetStore()
	if extStore, ok := store.(ExtendedUserStore); ok {
		_ = extStore.DeleteSession(c.Request().Context(), sessionID)
		c.Flash().Add("success", "Session revoked")
	}

	return c.Redirect(http.StatusSeeOther, "/sessions")
}

// Helper functions

func isValidEmail(email string) bool {
	// Simple email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// getClientIP is now defined in auth.go

func sendVerificationEmail(c buffalo.Context, user *User, token string) {
	// Build verification URL
	host := c.Value("host")
	if host == nil {
		scheme := "https"
		if c.Request().TLS == nil {
			scheme = "http"
		}
		host = fmt.Sprintf("%s://%s", scheme, c.Request().Host)
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", host, token)

	// Prepare template data
	data := render.Data{
		"user":       user,
		"verifyURL":  verifyURL,
		"appName":    c.Value("app_name"),
		"appAddress": c.Value("app_address"),
	}

	// Set defaults if not configured
	if data["appName"] == nil {
		data["appName"] = "Buffkit App"
	}
	if data["appAddress"] == nil {
		data["appAddress"] = ""
	}

	// Render email templates using plush
	htmlBody := bytes.Buffer{}
	textBody := bytes.Buffer{}

	// Try to use the actual templates we created
	htmlRenderer := r.HTML("mail/auth/verification.plush.html")
	if err := htmlRenderer.Render(&htmlBody, data); err != nil {
		// Fallback to inline template if plush template not found
		c.Logger().Errorf("Failed to render HTML email template: %v", err)
		htmlTemplate := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
	<h2>Verify Your Email</h2>
	<p>Hi %s,</p>
	<p>Thank you for registering! Please verify your email address by clicking the link below:</p>
	<p style="margin: 30px 0;">
		<a href="%s" style="background: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">
			Verify Email
		</a>
	</p>
	<p>Or copy and paste this link: %s</p>
	<p style="color: #666;">If you didn't create an account, you can safely ignore this email.</p>
</body>
</html>`
		htmlBody.WriteString(fmt.Sprintf(htmlTemplate, user.Name(), verifyURL, verifyURL))
	}

	textRenderer := r.Plain("mail/auth/verification.plush.txt")
	if err := textRenderer.Render(&textBody, data); err != nil {
		// Fallback to inline template if plush template not found
		c.Logger().Errorf("Failed to render text email template: %v", err)
		textTemplate := `Hi %s,

Thank you for registering! Please verify your email address by visiting:

%s

If you didn't create an account, you can safely ignore this email.`
		textBody.WriteString(fmt.Sprintf(textTemplate, user.Name(), verifyURL))
	}

	// Get the mail sender from context
	sender := c.Value("mail_sender")
	if sender == nil {
		// Try to get global mail sender from buffkit
		// Try to get from context value set by Wire
		if s := c.Value("mail_sender"); s != nil {
			sender = s
		}
	}

	// Send the email
	if sender != nil {
		// Use mail package if available
		if mailSender, ok := sender.(interface {
			Send(context.Context, map[string]interface{}) error
		}); ok {
			message := map[string]interface{}{
				"to":      user.Email,
				"subject": "Verify Your Email",
				"html":    htmlBody.String(),
				"text":    textBody.String(),
			}
			if err := mailSender.Send(c.Request().Context(), message); err != nil {
				c.Logger().Errorf("Failed to send verification email: %v", err)
			} else {
				c.Logger().Infof("Verification email sent to %s", user.Email)
			}
		}
	} else {
		// Log for development
		c.Logger().Infof("Verification email would be sent to %s with link: %s", user.Email, verifyURL)
	}
}

func sendPasswordResetEmail(c buffalo.Context, user *User, token string) {
	// Build reset URL
	host := c.Value("host")
	if host == nil {
		scheme := "https"
		if c.Request().TLS == nil {
			scheme = "http"
		}
		host = fmt.Sprintf("%s://%s", scheme, c.Request().Host)
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", host, token)

	// Get request IP for security notification
	requestIP := getClientIP(c.Request())

	// Prepare template data
	data := render.Data{
		"user":       user,
		"resetURL":   resetURL,
		"requestIP":  requestIP,
		"appName":    c.Value("app_name"),
		"appAddress": c.Value("app_address"),
	}

	// Set defaults if not configured
	if data["appName"] == nil {
		data["appName"] = "Buffkit App"
	}
	if data["appAddress"] == nil {
		data["appAddress"] = ""
	}

	// Render email templates using plush
	htmlBody := bytes.Buffer{}
	textBody := bytes.Buffer{}

	// Try to use the actual templates we created
	htmlRenderer := r.HTML("mail/auth/password_reset.plush.html")
	if err := htmlRenderer.Render(&htmlBody, data); err != nil {
		// Fallback to inline template if plush template not found
		c.Logger().Errorf("Failed to render HTML email template: %v", err)
		htmlTemplate := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
	<h2>Reset Your Password</h2>
	<p>Hi %s,</p>
	<p>We received a request to reset your password. Click the link below to create a new password:</p>
	<p style="margin: 30px 0;">
		<a href="%s" style="background: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">
			Reset Password
		</a>
	</p>
	<p>Or copy and paste this link: %s</p>
	<p style="color: #ff9800;">⚠️ This link expires in 1 hour.</p>
	<p style="color: #666;">Request from IP: %s</p>
	<p style="color: #666;">If you didn't request this, please ignore this email.</p>
</body>
</html>`
		htmlBody.WriteString(fmt.Sprintf(htmlTemplate, user.Name(), resetURL, resetURL, requestIP))
	}

	textRenderer := r.Plain("mail/auth/password_reset.plush.txt")
	if err := textRenderer.Render(&textBody, data); err != nil {
		// Fallback to inline template if plush template not found
		c.Logger().Errorf("Failed to render text email template: %v", err)
		textTemplate := `Hi %s,

We received a request to reset your password. Visit this link to create a new password:

%s

⚠️ This link expires in 1 hour.

Request from IP: %s

If you didn't request this, please ignore this email.`
		textBody.WriteString(fmt.Sprintf(textTemplate, user.Name(), resetURL, requestIP))
	}

	// Get the mail sender from context
	sender := c.Value("mail_sender")
	if sender == nil {
		// Try to get global mail sender from buffkit
		// Try to get from context value set by Wire
		if s := c.Value("mail_sender"); s != nil {
			sender = s
		}
	}

	// Send the email
	if sender != nil {
		// Use mail package if available
		if mailSender, ok := sender.(interface {
			Send(context.Context, map[string]interface{}) error
		}); ok {
			message := map[string]interface{}{
				"to":      user.Email,
				"subject": "Reset Your Password",
				"html":    htmlBody.String(),
				"text":    textBody.String(),
			}
			if err := mailSender.Send(c.Request().Context(), message); err != nil {
				c.Logger().Errorf("Failed to send password reset email: %v", err)
			} else {
				c.Logger().Infof("Password reset email sent to %s", user.Email)
			}
		}
	} else {
		// Log for development
		c.Logger().Infof("Password reset email would be sent to %s with link: %s", user.Email, resetURL)
	}
}

// r (renderer) is defined in auth.go
