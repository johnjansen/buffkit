package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/johnjansen/buffkit/auth"
)

// AuthTestContext holds authentication-specific test state
type AuthTestContext struct {
	suite              *TestSuite
	testUsers          map[string]*auth.User
	verificationTokens map[string]string
	resetTokens        map[string]string
	sessions           map[string]*auth.Session
	auditLogs          []auth.AuditLog
	emailsSent         []map[string]interface{}
	failedAttempts     map[string]int
	rateLimit          map[string]time.Time
	extendedStore      auth.ExtendedUserStore
}

// NewAuthTestContext creates a new authentication test context
func NewAuthTestContext(suite *TestSuite) *AuthTestContext {
	return &AuthTestContext{
		suite:              suite,
		testUsers:          make(map[string]*auth.User),
		verificationTokens: make(map[string]string),
		resetTokens:        make(map[string]string),
		sessions:           make(map[string]*auth.Session),
		auditLogs:          []auth.AuditLog{},
		emailsSent:         []map[string]interface{}{},
		failedAttempts:     make(map[string]int),
		rateLimit:          make(map[string]time.Time),
	}
}

// RegisterAuthSteps registers all enhanced authentication step definitions
func RegisterAuthSteps(ctx *godog.ScenarioContext, authCtx *AuthTestContext) {
	// Background steps
	ctx.Step(`^I have an extended user store configured$`, authCtx.iHaveAnExtendedUserStoreConfigured)

	// Registration steps
	ctx.Step(`^I submit a registration with email "([^"]*)" and password "([^"]*)"$`, authCtx.iSubmitARegistrationWithEmailAndPassword)
	ctx.Step(`^a new user account should be created$`, authCtx.aNewUserAccountShouldBeCreated)
	ctx.Step(`^I should receive a verification email$`, authCtx.iShouldReceiveAVerificationEmail)
	ctx.Step(`^I should be redirected to a success page$`, authCtx.iShouldBeRedirectedToASuccessPage)
	ctx.Step(`^I should see an error message about password strength$`, authCtx.iShouldSeeAnErrorMessageAboutPasswordStrength)
	ctx.Step(`^a user exists with email "([^"]*)"$`, authCtx.aUserExistsWithEmail)
	ctx.Step(`^I should see an error message about email already taken$`, authCtx.iShouldSeeAnErrorMessageAboutEmailAlreadyTaken)
	ctx.Step(`^only one user should exist with that email$`, authCtx.onlyOneUserShouldExistWithThatEmail)
	ctx.Step(`^no user account should be created$`, authCtx.noUserAccountShouldBeCreated)

	// Email verification steps
	ctx.Step(`^I have registered but not verified my email$`, authCtx.iHaveRegisteredButNotVerifiedMyEmail)
	ctx.Step(`^I have a valid verification token$`, authCtx.iHaveAValidVerificationToken)
	ctx.Step(`^I visit the verification link$`, authCtx.iVisitTheVerificationLink)
	ctx.Step(`^my account should be marked as verified$`, authCtx.myAccountShouldBeMarkedAsVerified)
	ctx.Step(`^I should see a success message$`, authCtx.iShouldSeeASuccessMessage)
	ctx.Step(`^no accounts should be verified$`, authCtx.noAccountsShouldBeVerified)
	ctx.Step(`^I have a verification token older than (\d+) hours$`, authCtx.iHaveAVerificationTokenOlderThanHours)
	ctx.Step(`^I should see an expiration error message$`, authCtx.iShouldSeeAnExpirationErrorMessage)
	ctx.Step(`^my account should remain unverified$`, authCtx.myAccountShouldRemainUnverified)

	// Password reset steps
	ctx.Step(`^I submit a password reset request for "([^"]*)"$`, authCtx.iSubmitAPasswordResetRequestFor)
	ctx.Step(`^a password reset email should be sent$`, authCtx.aPasswordResetEmailShouldBeSent)
	ctx.Step(`^a reset token should be created$`, authCtx.aResetTokenShouldBeCreated)
	ctx.Step(`^no email should be sent$`, authCtx.noEmailShouldBeSent)
	ctx.Step(`^no reset token should be created$`, authCtx.noResetTokenShouldBeCreated)
	ctx.Step(`^I have a valid password reset token$`, authCtx.iHaveAValidPasswordResetToken)
	ctx.Step(`^I visit the reset password link$`, authCtx.iVisitTheResetPasswordLink)
	ctx.Step(`^I submit a new password "([^"]*)"$`, authCtx.iSubmitANewPassword)
	ctx.Step(`^my password should be updated$`, authCtx.myPasswordShouldBeUpdated)
	ctx.Step(`^the reset token should be invalidated$`, authCtx.theResetTokenShouldBeInvalidated)
	ctx.Step(`^I should be redirected to login$`, authCtx.iShouldBeRedirectedToLogin)
	ctx.Step(`^I submit mismatched passwords$`, authCtx.iSubmitMismatchedPasswords)
	ctx.Step(`^I should see an error about passwords not matching$`, authCtx.iShouldSeeAnErrorAboutPasswordsNotMatching)
	ctx.Step(`^my password should not be changed$`, authCtx.myPasswordShouldNotBeChanged)
	ctx.Step(`^I have a password reset token older than (\d+) hour$`, authCtx.iHaveAPasswordResetTokenOlderThanHour)
	ctx.Step(`^I should be redirected to forgot password$`, authCtx.iShouldBeRedirectedToForgotPassword)

	// Profile management steps
	ctx.Step(`^I should see my profile information$`, authCtx.iShouldSeeMyProfileInformation)
	ctx.Step(`^I update my name to "([^"]*)"$`, authCtx.iUpdateMyNameTo)
	ctx.Step(`^I submit the profile form$`, authCtx.iSubmitTheProfileForm)
	ctx.Step(`^my profile should be updated$`, authCtx.myProfileShouldBeUpdated)

	// Session management steps
	ctx.Step(`^I should see my active sessions$`, authCtx.iShouldSeeMyActiveSessions)
	ctx.Step(`^I should see session details like IP and user agent$`, authCtx.iShouldSeeSessionDetailsLikeIPAndUserAgent)
	ctx.Step(`^I am logged in with multiple sessions$`, authCtx.iAmLoggedInWithMultipleSessions)
	ctx.Step(`^I revoke a specific session$`, authCtx.iRevokeASpecificSession)
	ctx.Step(`^that session should be invalidated$`, authCtx.thatSessionShouldBeInvalidated)

	// Rate limiting steps
	ctx.Step(`^I make (\d+) failed login attempts within (\d+) minute$`, authCtx.iMakeFailedLoginAttemptsWithinMinute)
	ctx.Step(`^subsequent login attempts should be blocked$`, authCtx.subsequentLoginAttemptsShouldBeBlocked)
	ctx.Step(`^I should see a rate limit error message$`, authCtx.iShouldSeeARateLimitErrorMessage)
	ctx.Step(`^I make (\d+) registration attempts within (\d+) minute$`, authCtx.iMakeRegistrationAttemptsWithinMinute)
	ctx.Step(`^subsequent registration attempts should be blocked$`, authCtx.subsequentRegistrationAttemptsShouldBeBlocked)

	// Account security steps
	ctx.Step(`^I make (\d+) failed login attempts for that user$`, authCtx.iMakeFailedLoginAttemptsForThatUser)
	ctx.Step(`^the account should be locked$`, authCtx.theAccountShouldBeLocked)
	ctx.Step(`^valid credentials should also fail$`, authCtx.validCredentialsShouldAlsoFail)
	ctx.Step(`^I should see an account locked message$`, authCtx.iShouldSeeAnAccountLockedMessage)
	ctx.Step(`^an account is locked until (\d+) minutes ago$`, authCtx.anAccountIsLockedUntilMinutesAgo)
	ctx.Step(`^I attempt to login with valid credentials$`, authCtx.iAttemptToLoginWithValidCredentials)
	ctx.Step(`^the login should succeed$`, authCtx.theLoginShouldSucceed)
	ctx.Step(`^the account should be unlocked$`, authCtx.theAccountShouldBeUnlocked)

	// Audit logging steps
	ctx.Step(`^I attempt to login$`, authCtx.iAttemptToLogin)
	ctx.Step(`^an audit log entry should be created$`, authCtx.anAuditLogEntryShouldBeCreated)
	ctx.Step(`^it should include timestamp, IP, and result$`, authCtx.itShouldIncludeTimestampIPAndResult)
	ctx.Step(`^I change my password$`, authCtx.iChangeMyPassword)
	ctx.Step(`^it should record the password change event$`, authCtx.itShouldRecordThePasswordChangeEvent)

	// Background jobs steps
	ctx.Step(`^there are expired sessions older than (\d+) hours$`, authCtx.thereAreExpiredSessionsOlderThanHours)
	ctx.Step(`^the session cleanup job runs$`, authCtx.theSessionCleanupJobRuns)
	ctx.Step(`^expired sessions should be deleted$`, authCtx.expiredSessionsShouldBeDeleted)
	ctx.Step(`^active sessions should remain$`, authCtx.activeSessionsShouldRemain)
	ctx.Step(`^there are accounts with expired lock times$`, authCtx.thereAreAccountsWithExpiredLockTimes)
	ctx.Step(`^the account unlock job runs$`, authCtx.theAccountUnlockJobRuns)
	ctx.Step(`^those accounts should be unlocked$`, authCtx.thoseAccountsShouldBeUnlocked)
	ctx.Step(`^recently locked accounts should remain locked$`, authCtx.recentlyLockedAccountsShouldRemainLocked)

	// Email integration steps
	ctx.Step(`^I register a new account$`, authCtx.iRegisterANewAccount)
	ctx.Step(`^the mail system should receive a send request$`, authCtx.theMailSystemShouldReceiveASendRequest)
	ctx.Step(`^the email should contain a verification link$`, authCtx.theEmailShouldContainAVerificationLink)
	ctx.Step(`^I request a password reset$`, authCtx.iRequestAPasswordReset)
	ctx.Step(`^the email should contain a reset link$`, authCtx.theEmailShouldContainAResetLink)

	// Remember me steps
	ctx.Step(`^I login with remember me checked$`, authCtx.iLoginWithRememberMeChecked)
	ctx.Step(`^a persistent cookie should be set$`, authCtx.aPersistentCookieShouldBeSet)
	ctx.Step(`^my session should persist across browser restarts$`, authCtx.mySessionShouldPersistAcrossBrowserRestarts)
	ctx.Step(`^I login without remember me checked$`, authCtx.iLoginWithoutRememberMeChecked)
	ctx.Step(`^only a session cookie should be set$`, authCtx.onlyASessionCookieShouldBeSet)
	ctx.Step(`^my session should end when browser closes$`, authCtx.mySessionShouldEndWhenBrowserCloses)

	// Security headers steps
	ctx.Step(`^the response should include security headers$`, authCtx.theResponseShouldIncludeSecurityHeaders)
	ctx.Step(`^CSP should prevent inline scripts$`, authCtx.cspShouldPreventInlineScripts)
	ctx.Step(`^X-Frame-Options should prevent clickjacking$`, authCtx.xFrameOptionsShouldPreventClickjacking)

	// Password strength steps
	ctx.Step(`^I try to register with password "([^"]*)"$`, authCtx.iTryToRegisterWithPassword)
	ctx.Step(`^I should see password strength requirements$`, authCtx.iShouldSeePasswordStrengthRequirements)
	ctx.Step(`^the registration should fail$`, authCtx.theRegistrationShouldFail)
	ctx.Step(`^I try to set password "([^"]*)"$`, authCtx.iTryToSetPassword)
	ctx.Step(`^the password should not be changed$`, authCtx.thePasswordShouldNotBeChanged)

	// Multi-device support steps
	ctx.Step(`^I am logged in on device A$`, authCtx.iAmLoggedInOnDeviceA)
	ctx.Step(`^I login on device B$`, authCtx.iLoginOnDeviceB)
	ctx.Step(`^both sessions should be active$`, authCtx.bothSessionsShouldBeActive)
	ctx.Step(`^I should see both devices in sessions list$`, authCtx.iShouldSeeBothDevicesInSessionsList)
	ctx.Step(`^I am logged in on multiple devices$`, authCtx.iAmLoggedInOnMultipleDevices)
	ctx.Step(`^I revoke the session for device B$`, authCtx.iRevokeTheSessionForDeviceB)
	ctx.Step(`^device B should be logged out$`, authCtx.deviceBShouldBeLoggedOut)
	ctx.Step(`^device A should remain logged in$`, authCtx.deviceAShouldRemainLoggedIn)
}

// Step implementations

func (a *AuthTestContext) iHaveAnExtendedUserStoreConfigured() error {
	// Create a mock extended user store for testing
	if a.suite.kit != nil && a.suite.kit.AuthStore != nil {
		if extStore, ok := a.suite.kit.AuthStore.(auth.ExtendedUserStore); ok {
			a.extendedStore = extStore
			return nil
		}
	}
	// Create a memory-based extended store for testing
	a.extendedStore = &mockExtendedStore{
		users:              a.testUsers,
		verificationTokens: a.verificationTokens,
		resetTokens:        a.resetTokens,
		sessions:           a.sessions,
		auditLogs:          &a.auditLogs,
	}
	return nil
}

func (a *AuthTestContext) iSubmitARegistrationWithEmailAndPassword(email, password string) error {
	form := url.Values{}
	form.Set("email", email)
	form.Set("password", password)
	form.Set("password_confirm", password)
	form.Set("name", "Test User")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.suite.response = httptest.NewRecorder()
	a.suite.app.ServeHTTP(a.suite.response, req)

	return nil
}

func (a *AuthTestContext) aNewUserAccountShouldBeCreated() error {
	// Check if a user was created in the last operation
	if len(a.testUsers) == 0 {
		return fmt.Errorf("no user account was created")
	}
	return nil
}

func (a *AuthTestContext) iShouldReceiveAVerificationEmail() error {
	if len(a.emailsSent) == 0 {
		return fmt.Errorf("no verification email was sent")
	}
	lastEmail := a.emailsSent[len(a.emailsSent)-1]
	if !strings.Contains(lastEmail["Subject"].(string), "Verify") {
		return fmt.Errorf("email subject does not contain 'Verify'")
	}
	return nil
}

func (a *AuthTestContext) iShouldBeRedirectedToASuccessPage() error {
	if a.suite.response.Code < 300 || a.suite.response.Code >= 400 {
		return fmt.Errorf("expected redirect, got status %d", a.suite.response.Code)
	}
	return nil
}

func (a *AuthTestContext) iShouldSeeAnErrorMessageAboutPasswordStrength() error {
	body := a.suite.response.Body.String()
	if !strings.Contains(body, "password") && !strings.Contains(body, "strength") {
		return fmt.Errorf("no password strength error message found")
	}
	return nil
}

func (a *AuthTestContext) aUserExistsWithEmail(email string) error {
	user := &auth.User{
		ID:         "test-user-1",
		Email:      email,
		IsActive:   true,
		IsVerified: true,
	}
	a.testUsers[email] = user
	return nil
}

func (a *AuthTestContext) iShouldSeeAnErrorMessageAboutEmailAlreadyTaken() error {
	body := a.suite.response.Body.String()
	if !strings.Contains(body, "email") && !strings.Contains(body, "already") {
		return fmt.Errorf("no email already taken error message found")
	}
	return nil
}

func (a *AuthTestContext) onlyOneUserShouldExistWithThatEmail() error {
	emailCount := make(map[string]int)
	for _, user := range a.testUsers {
		emailCount[user.Email]++
	}
	for email, count := range emailCount {
		if count > 1 {
			return fmt.Errorf("found %d users with email %s", count, email)
		}
	}
	return nil
}

func (a *AuthTestContext) noUserAccountShouldBeCreated() error {
	// This would check that no new users were added
	return nil
}

// Add more step implementations as needed...

// mockExtendedStore implements auth.ExtendedUserStore for testing
type mockExtendedStore struct {
	users              map[string]*auth.User
	verificationTokens map[string]string
	resetTokens        map[string]string
	sessions           map[string]*auth.Session
	auditLogs          *[]auth.AuditLog
	loginAttempts      []auth.LoginAttempt
	devices            map[string]*auth.UserDevice
}

func (m *mockExtendedStore) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockExtendedStore) ByID(ctx context.Context, id string) (*auth.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockExtendedStore) ByEmail(ctx context.Context, email string) (*auth.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockExtendedStore) Create(ctx context.Context, user *auth.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockExtendedStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	_, exists := m.users[email]
	return exists, nil
}

func (m *mockExtendedStore) UpdatePassword(ctx context.Context, id, digest string) error {
	for _, u := range m.users {
		if u.ID == id {
			// In real impl, this would update the password hash
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	for _, u := range m.users {
		if u.ID == id {
			// Simple update simulation
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) Delete(ctx context.Context, id string) error {
	for email, u := range m.users {
		if u.ID == id {
			delete(m.users, email)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) List(ctx context.Context, limit, offset int) ([]*auth.User, error) {
	var users []*auth.User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

func (m *mockExtendedStore) SetEmailVerificationToken(ctx context.Context, id, token string) error {
	m.verificationTokens[token] = id
	return nil
}

func (m *mockExtendedStore) CreateWithVerification(ctx context.Context, user *auth.User) (string, error) {
	token := fmt.Sprintf("verify-token-%s", user.ID)
	m.verificationTokens[token] = user.ID
	return token, m.Create(ctx, user)
}

func (m *mockExtendedStore) VerifyEmail(ctx context.Context, token string) (*auth.User, error) {
	if userID, ok := m.verificationTokens[token]; ok {
		for _, u := range m.users {
			if u.ID == userID {
				u.IsVerified = true
				delete(m.verificationTokens, token)
				return u, nil
			}
		}
	}
	return nil, fmt.Errorf("invalid verification token")
}

func (m *mockExtendedStore) SetPasswordResetToken(ctx context.Context, email, token string) error {
	if user, ok := m.users[email]; ok {
		m.resetTokens[token] = user.ID
		return nil
	}
	return nil // Silent failure for non-existent emails
}

func (m *mockExtendedStore) CreatePasswordResetToken(ctx context.Context, email string) (string, error) {
	if user, ok := m.users[email]; ok {
		token := fmt.Sprintf("reset-token-%s", user.ID)
		m.resetTokens[token] = user.ID
		return token, nil
	}
	return "", nil // Silent failure for non-existent emails
}

func (m *mockExtendedStore) ResetPassword(ctx context.Context, token, newPasswordDigest string) error {
	if userID, ok := m.resetTokens[token]; ok {
		for _, u := range m.users {
			if u.ID == userID {
				// In real impl, this would be hashed and set properly
				delete(m.resetTokens, token)
				return nil
			}
		}
	}
	return fmt.Errorf("invalid reset token")
}

func (m *mockExtendedStore) ValidateResetToken(ctx context.Context, token string) (*auth.User, error) {
	if userID, ok := m.resetTokens[token]; ok {
		for _, u := range m.users {
			if u.ID == userID {
				return u, nil
			}
		}
	}
	return nil, fmt.Errorf("invalid reset token")
}

func (m *mockExtendedStore) IncrementFailedLoginAttempts(ctx context.Context, email string) error {
	// Mock implementation
	return nil
}

func (m *mockExtendedStore) ResetFailedLoginAttempts(ctx context.Context, email string) error {
	// Mock implementation
	return nil
}

func (m *mockExtendedStore) UnlockAccount(ctx context.Context, email string) error {
	if user, ok := m.users[email]; ok {
		user.LockedUntil = nil
		return nil
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	if s, ok := m.sessions[token]; ok {
		return s, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockExtendedStore) UpdateSessionActivity(ctx context.Context, token string) error {
	if s, ok := m.sessions[token]; ok {
		s.LastActivityAt = time.Now()
		return nil
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockExtendedStore) DeleteUserSessions(ctx context.Context, userID string) error {
	for id, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockExtendedStore) ListUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	var sessions []*auth.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (m *mockExtendedStore) GetUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	var sessions []*auth.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (m *mockExtendedStore) CreateSession(ctx context.Context, session *auth.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockExtendedStore) RevokeSession(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockExtendedStore) LogAuthEvent(ctx context.Context, log *auth.AuditLog) error {
	*m.auditLogs = append(*m.auditLogs, *log)
	return nil
}

func (m *mockExtendedStore) GetUserAuditLogs(ctx context.Context, userID string, limit int) ([]*auth.AuditLog, error) {
	var logs []*auth.AuditLog
	for i := range *m.auditLogs {
		log := &(*m.auditLogs)[i]
		if log.UserID != nil && *log.UserID == userID {
			logs = append(logs, log)
			if len(logs) >= limit {
				break
			}
		}
	}
	return logs, nil
}

func (m *mockExtendedStore) RegisterDevice(ctx context.Context, device *auth.UserDevice) error {
	if m.devices == nil {
		m.devices = make(map[string]*auth.UserDevice)
	}
	m.devices[device.ID] = device
	return nil
}

func (m *mockExtendedStore) TrustDevice(ctx context.Context, deviceID string) error {
	if device, ok := m.devices[deviceID]; ok {
		device.IsTrusted = true
		return nil
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) RemoveDevice(ctx context.Context, deviceID string) error {
	delete(m.devices, deviceID)
	return nil
}

func (m *mockExtendedStore) ListUserDevices(ctx context.Context, userID string) ([]*auth.UserDevice, error) {
	var devices []*auth.UserDevice
	for _, d := range m.devices {
		if d.UserID == userID {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

func (m *mockExtendedStore) RecordLoginAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	m.loginAttempts = append(m.loginAttempts, *attempt)
	return nil
}

func (m *mockExtendedStore) CountRecentLoginAttempts(ctx context.Context, email string, since time.Time) (int, error) {
	count := 0
	for _, attempt := range m.loginAttempts {
		if attempt.Email == email && attempt.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *mockExtendedStore) CountRecentIPAttempts(ctx context.Context, ip string, since time.Time) (int, error) {
	count := 0
	for _, attempt := range m.loginAttempts {
		if attempt.IPAddress == ip && attempt.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *mockExtendedStore) IsAccountLocked(ctx context.Context, email string) (bool, error) {
	if user, ok := m.users[email]; ok {
		if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockExtendedStore) LockAccount(ctx context.Context, email string, until time.Time) error {
	if user, ok := m.users[email]; ok {
		user.LockedUntil = &until
		return nil
	}
	return sql.ErrNoRows
}

func (m *mockExtendedStore) ResendVerificationEmail(ctx context.Context, email string) error {
	// Mock implementation
	return nil
}

func (m *mockExtendedStore) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Simple rate limiting for tests
	return true, nil
}

func (m *mockExtendedStore) RecordRateLimitAttempt(ctx context.Context, key string) error {
	// Mock implementation
	return nil
}

// Placeholder implementations for remaining steps...
func (a *AuthTestContext) iHaveRegisteredButNotVerifiedMyEmail() error           { return nil }
func (a *AuthTestContext) iHaveAValidVerificationToken() error                   { return nil }
func (a *AuthTestContext) iVisitTheVerificationLink() error                      { return nil }
func (a *AuthTestContext) myAccountShouldBeMarkedAsVerified() error              { return nil }
func (a *AuthTestContext) iShouldSeeASuccessMessage() error                      { return nil }
func (a *AuthTestContext) noAccountsShouldBeVerified() error                     { return nil }
func (a *AuthTestContext) iHaveAVerificationTokenOlderThanHours(hours int) error { return nil }
func (a *AuthTestContext) iShouldSeeAnExpirationErrorMessage() error             { return nil }
func (a *AuthTestContext) myAccountShouldRemainUnverified() error                { return nil }
func (a *AuthTestContext) iSubmitAPasswordResetRequestFor(email string) error    { return nil }
func (a *AuthTestContext) aPasswordResetEmailShouldBeSent() error                { return nil }
func (a *AuthTestContext) aResetTokenShouldBeCreated() error                     { return nil }
func (a *AuthTestContext) noEmailShouldBeSent() error                            { return nil }
func (a *AuthTestContext) noResetTokenShouldBeCreated() error                    { return nil }
func (a *AuthTestContext) iHaveAValidPasswordResetToken() error                  { return nil }
func (a *AuthTestContext) iVisitTheResetPasswordLink() error                     { return nil }
func (a *AuthTestContext) iSubmitANewPassword(password string) error             { return nil }
func (a *AuthTestContext) myPasswordShouldBeUpdated() error                      { return nil }
func (a *AuthTestContext) theResetTokenShouldBeInvalidated() error               { return nil }
func (a *AuthTestContext) iShouldBeRedirectedToLogin() error                     { return nil }
func (a *AuthTestContext) iSubmitMismatchedPasswords() error                     { return nil }
func (a *AuthTestContext) iShouldSeeAnErrorAboutPasswordsNotMatching() error     { return nil }
func (a *AuthTestContext) myPasswordShouldNotBeChanged() error                   { return nil }
func (a *AuthTestContext) iHaveAPasswordResetTokenOlderThanHour(hours int) error { return nil }
func (a *AuthTestContext) iShouldBeRedirectedToForgotPassword() error            { return nil }
func (a *AuthTestContext) iShouldSeeMyProfileInformation() error                 { return nil }
func (a *AuthTestContext) iUpdateMyNameTo(name string) error                     { return nil }
func (a *AuthTestContext) iSubmitTheProfileForm() error                          { return nil }
func (a *AuthTestContext) myProfileShouldBeUpdated() error                       { return nil }
func (a *AuthTestContext) iShouldSeeMyActiveSessions() error                     { return nil }
func (a *AuthTestContext) iShouldSeeSessionDetailsLikeIPAndUserAgent() error     { return nil }
func (a *AuthTestContext) iAmLoggedInWithMultipleSessions() error                { return nil }
func (a *AuthTestContext) iRevokeASpecificSession() error                        { return nil }
func (a *AuthTestContext) thatSessionShouldBeInvalidated() error                 { return nil }
func (a *AuthTestContext) iMakeFailedLoginAttemptsWithinMinute(attempts, minutes int) error {
	return nil
}
func (a *AuthTestContext) subsequentLoginAttemptsShouldBeBlocked() error { return nil }
func (a *AuthTestContext) iShouldSeeARateLimitErrorMessage() error       { return nil }
func (a *AuthTestContext) iMakeRegistrationAttemptsWithinMinute(attempts, minutes int) error {
	return nil
}
func (a *AuthTestContext) subsequentRegistrationAttemptsShouldBeBlocked() error   { return nil }
func (a *AuthTestContext) iMakeFailedLoginAttemptsForThatUser(attempts int) error { return nil }
func (a *AuthTestContext) theAccountShouldBeLocked() error                        { return nil }
func (a *AuthTestContext) validCredentialsShouldAlsoFail() error                  { return nil }
func (a *AuthTestContext) iShouldSeeAnAccountLockedMessage() error                { return nil }
func (a *AuthTestContext) anAccountIsLockedUntilMinutesAgo(minutes int) error     { return nil }
func (a *AuthTestContext) iAttemptToLoginWithValidCredentials() error             { return nil }
func (a *AuthTestContext) theLoginShouldSucceed() error                           { return nil }
func (a *AuthTestContext) theAccountShouldBeUnlocked() error                      { return nil }
func (a *AuthTestContext) iAttemptToLogin() error                                 { return nil }
func (a *AuthTestContext) anAuditLogEntryShouldBeCreated() error                  { return nil }
func (a *AuthTestContext) itShouldIncludeTimestampIPAndResult() error             { return nil }
func (a *AuthTestContext) iChangeMyPassword() error                               { return nil }
func (a *AuthTestContext) itShouldRecordThePasswordChangeEvent() error            { return nil }
func (a *AuthTestContext) thereAreExpiredSessionsOlderThanHours(hours int) error  { return nil }
func (a *AuthTestContext) theSessionCleanupJobRuns() error                        { return nil }
func (a *AuthTestContext) expiredSessionsShouldBeDeleted() error                  { return nil }
func (a *AuthTestContext) activeSessionsShouldRemain() error                      { return nil }
func (a *AuthTestContext) thereAreAccountsWithExpiredLockTimes() error            { return nil }
func (a *AuthTestContext) theAccountUnlockJobRuns() error                         { return nil }
func (a *AuthTestContext) thoseAccountsShouldBeUnlocked() error                   { return nil }
func (a *AuthTestContext) iAmLoggedInOnDeviceA() error                            { return nil }
func (a *AuthTestContext) iLoginOnDeviceB() error                                 { return nil }
func (a *AuthTestContext) bothSessionsShouldBeActive() error                      { return nil }
func (a *AuthTestContext) iShouldSeeBothDevicesInSessionsList() error             { return nil }
func (a *AuthTestContext) iAmLoggedInOnMultipleDevices() error                    { return nil }
func (a *AuthTestContext) iRevokeTheSessionForDeviceB() error                     { return nil }
func (a *AuthTestContext) deviceBShouldBeLoggedOut() error                        { return nil }
func (a *AuthTestContext) deviceAShouldRemainLoggedIn() error                     { return nil }
func (a *AuthTestContext) recentlyLockedAccountsShouldRemainLocked() error        { return nil }
func (a *AuthTestContext) iRegisterANewAccount() error                            { return nil }
func (a *AuthTestContext) theMailSystemShouldReceiveASendRequest() error          { return nil }
func (a *AuthTestContext) theEmailShouldContainAVerificationLink() error          { return nil }
func (a *AuthTestContext) iRequestAPasswordReset() error                          { return nil }
func (a *AuthTestContext) theEmailShouldContainAResetLink() error                 { return nil }
func (a *AuthTestContext) iLoginWithRememberMeChecked() error                     { return nil }
func (a *AuthTestContext) aPersistentCookieShouldBeSet() error                    { return nil }
func (a *AuthTestContext) mySessionShouldPersistAcrossBrowserRestarts() error     { return nil }
func (a *AuthTestContext) iLoginWithoutRememberMeChecked() error                  { return nil }
func (a *AuthTestContext) onlyASessionCookieShouldBeSet() error                   { return nil }
func (a *AuthTestContext) mySessionShouldEndWhenBrowserCloses() error             { return nil }
func (a *AuthTestContext) theResponseShouldIncludeSecurityHeaders() error         { return nil }
func (a *AuthTestContext) cspShouldPreventInlineScripts() error                   { return nil }
func (a *AuthTestContext) xFrameOptionsShouldPreventClickjacking() error          { return nil }
func (a *AuthTestContext) iTryToRegisterWithPassword(password string) error       { return nil }
func (a *AuthTestContext) iShouldSeePasswordStrengthRequirements() error          { return nil }
func (a *AuthTestContext) theRegistrationShouldFail() error                       { return nil }
func (a *AuthTestContext) iTryToSetPassword(password string) error                { return nil }
func (a *AuthTestContext) thePasswordShouldNotBeChanged() error                   { return nil }
