-- 002_create_sessions.down.sql
-- Drops the sessions table and related indexes

-- Drop indexes first
DROP INDEX IF EXISTS idx_buffkit_sessions_expires_at;
DROP INDEX IF EXISTS idx_buffkit_sessions_token;
DROP INDEX IF EXISTS idx_buffkit_sessions_user_id;

-- Drop the sessions table
DROP TABLE IF EXISTS buffkit_sessions;
