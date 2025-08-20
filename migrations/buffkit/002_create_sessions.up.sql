-- 002_create_sessions.up.sql
-- Creates the sessions table for Buffkit authentication
-- Supports PostgreSQL, MySQL, and SQLite

CREATE TABLE IF NOT EXISTS buffkit_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES buffkit_users(id) ON DELETE CASCADE
);

-- Create index for user_id lookups
CREATE INDEX IF NOT EXISTS idx_buffkit_sessions_user_id ON buffkit_sessions(user_id);

-- Create index for token lookups
CREATE INDEX IF NOT EXISTS idx_buffkit_sessions_token ON buffkit_sessions(token);

-- Create index for cleanup of expired sessions
CREATE INDEX IF NOT EXISTS idx_buffkit_sessions_expires_at ON buffkit_sessions(expires_at);
