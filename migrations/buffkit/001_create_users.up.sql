-- 001_create_users.up.sql
-- Creates the users table for Buffkit authentication
-- Supports PostgreSQL, MySQL, and SQLite

-- PostgreSQL and SQLite syntax (mostly compatible)
CREATE TABLE IF NOT EXISTS buffkit_users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    password_digest VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for email lookups
CREATE INDEX IF NOT EXISTS idx_buffkit_users_email ON buffkit_users(email);

-- Create index for active users
CREATE INDEX IF NOT EXISTS idx_buffkit_users_is_active ON buffkit_users(is_active);
