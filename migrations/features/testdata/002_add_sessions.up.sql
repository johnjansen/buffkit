-- 002_add_sessions.up.sql
-- Test migration for BDD testing - adds sessions table

CREATE TABLE IF NOT EXISTS test_sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES test_users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_test_sessions_user_id ON test_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_test_sessions_token ON test_sessions(token);
CREATE INDEX IF NOT EXISTS idx_test_sessions_expires_at ON test_sessions(expires_at);
