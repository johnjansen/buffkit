-- 001_create_test_table.up.sql
-- Test migration for BDD testing

CREATE TABLE IF NOT EXISTS test_users (
    id INTEGER PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_test_users_email ON test_users(email);
