-- Create users table for authentication
-- Supports multiple database dialects (PostgreSQL, MySQL, SQLite)

-- PostgreSQL version
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    -- Primary key - UUID for PostgreSQL, string for others
    id VARCHAR(36) PRIMARY KEY,

    -- Core fields
    email VARCHAR(255) NOT NULL UNIQUE,
    password_digest VARCHAR(255) NOT NULL,

    -- Profile fields
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(100),
    avatar_url VARCHAR(500),

    -- Status fields
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    is_admin BOOLEAN DEFAULT false,

    -- Email verification
    email_verified_at TIMESTAMP NULL,
    email_verification_token VARCHAR(255),
    email_verification_sent_at TIMESTAMP NULL,

    -- Password reset
    password_reset_token VARCHAR(255),
    password_reset_sent_at TIMESTAMP NULL,

    -- Security fields
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP NULL,
    last_login_at TIMESTAMP NULL,
    last_login_ip VARCHAR(45),

    -- Two-factor auth preparation
    totp_secret VARCHAR(255),
    totp_enabled BOOLEAN DEFAULT false,
    recovery_codes TEXT,

    -- Metadata
    extra JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token);
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Sessions table for managing user sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,

    -- Session data
    ip_address VARCHAR(45),
    user_agent TEXT,

    -- Expiry
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP NOT NULL,

    -- Metadata
    data JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Audit log table for security tracking
CREATE TABLE IF NOT EXISTS auth_audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),

    -- Event details
    event_type VARCHAR(50) NOT NULL, -- login, logout, register, password_reset, etc.
    event_status VARCHAR(20) NOT NULL, -- success, failure

    -- Context
    ip_address VARCHAR(45),
    user_agent TEXT,

    -- Additional data
    metadata JSON,
    error_message TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_user_id ON auth_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_event_type ON auth_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_created_at ON auth_audit_logs(created_at);

-- Login attempts table for rate limiting
CREATE TABLE IF NOT EXISTS login_attempts (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255),
    ip_address VARCHAR(45),

    -- Attempt details
    success BOOLEAN DEFAULT false,

    -- Metadata
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts(email);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_address ON login_attempts(ip_address);
CREATE INDEX IF NOT EXISTS idx_login_attempts_created_at ON login_attempts(created_at);

-- Device tracking table for security
CREATE TABLE IF NOT EXISTS user_devices (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- Device identification
    device_name VARCHAR(255),
    device_fingerprint VARCHAR(255) UNIQUE,

    -- Device details
    platform VARCHAR(50),
    browser VARCHAR(50),
    ip_address VARCHAR(45),

    -- Trust status
    is_trusted BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP,

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_user_devices_device_fingerprint ON user_devices(device_fingerprint);
