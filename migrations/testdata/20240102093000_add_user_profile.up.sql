-- Add profile fields to users table
ALTER TABLE users ADD COLUMN first_name VARCHAR(100);
ALTER TABLE users ADD COLUMN last_name VARCHAR(100);
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);
ALTER TABLE users ADD COLUMN date_of_birth DATE;
ALTER TABLE users ADD COLUMN phone VARCHAR(20);
ALTER TABLE users ADD COLUMN country VARCHAR(2);
ALTER TABLE users ADD COLUMN timezone VARCHAR(50) DEFAULT 'UTC';
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN is_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN verified_at TIMESTAMP;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMP;

-- Create index on active users for faster filtering
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Create index on country for geographical queries
CREATE INDEX idx_users_country ON users(country);
