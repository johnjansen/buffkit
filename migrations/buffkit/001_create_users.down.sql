-- 001_create_users.down.sql
-- Drops the users table and related indexes

-- Drop indexes first
DROP INDEX IF EXISTS idx_buffkit_users_is_active;
DROP INDEX IF EXISTS idx_buffkit_users_email;

-- Drop the users table
DROP TABLE IF EXISTS buffkit_users;
