-- Drop tables in reverse order of creation to respect foreign key constraints

DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS auth_audit_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

-- Drop indexes if they exist separately (some databases)
DROP INDEX IF EXISTS idx_user_devices_device_fingerprint;
DROP INDEX IF EXISTS idx_user_devices_user_id;
DROP INDEX IF EXISTS idx_login_attempts_created_at;
DROP INDEX IF EXISTS idx_login_attempts_ip_address;
DROP INDEX IF EXISTS idx_login_attempts_email;
DROP INDEX IF EXISTS idx_auth_audit_logs_created_at;
DROP INDEX IF EXISTS idx_auth_audit_logs_event_type;
DROP INDEX IF EXISTS idx_auth_audit_logs_user_id;
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_token;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_email_verification_token;
DROP INDEX IF EXISTS idx_users_email;
