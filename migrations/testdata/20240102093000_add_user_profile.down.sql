-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
-- This is the standard approach for removing columns in SQLite

-- Step 1: Create a temporary table with only the original columns
CREATE TABLE users_temp (
    id INTEGER PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy data from the existing table to the temporary table
INSERT INTO users_temp (id, email, username, password_hash, created_at, updated_at)
SELECT id, email, username, password_hash, created_at, updated_at
FROM users;

-- Step 3: Drop the original table
DROP TABLE users;

-- Step 4: Rename the temporary table to the original name
ALTER TABLE users_temp RENAME TO users;

-- Step 5: Recreate the original indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

-- Note: The additional indexes (idx_users_active, idx_users_country)
-- are already gone since they were on the dropped columns
