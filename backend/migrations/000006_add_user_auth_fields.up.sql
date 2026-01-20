-- Add authentication fields to users table
ALTER TABLE users 
ADD COLUMN username VARCHAR(50) UNIQUE NOT NULL DEFAULT '',
ADD COLUMN email VARCHAR(255) UNIQUE NOT NULL DEFAULT '',
ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT '',
ADD COLUMN name VARCHAR(100),
ADD COLUMN surname VARCHAR(100),
ADD COLUMN is_email_verified BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN email_verification_token VARCHAR(255),
ADD COLUMN password_reset_token VARCHAR(255),
ADD COLUMN password_reset_expires TIMESTAMPTZ,
ADD COLUMN last_login TIMESTAMPTZ,
ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Remove defaults after adding columns (defaults are just for migration)
ALTER TABLE users 
ALTER COLUMN username DROP DEFAULT,
ALTER COLUMN email DROP DEFAULT,
ALTER COLUMN password_hash DROP DEFAULT;

-- Create indexes for faster lookups
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_email_verification_token ON users(email_verification_token);
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token);