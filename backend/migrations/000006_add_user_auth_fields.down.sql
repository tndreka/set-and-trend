-- Rollback: remove authentication fields
DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_email_verification_token;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

ALTER TABLE users 
DROP COLUMN IF EXISTS updated_at,
DROP COLUMN IF EXISTS last_login,
DROP COLUMN IF EXISTS password_reset_expires,
DROP COLUMN IF EXISTS password_reset_token,
DROP COLUMN IF EXISTS email_verification_token,
DROP COLUMN IF EXISTS is_email_verified,
DROP COLUMN IF EXISTS surname,
DROP COLUMN IF EXISTS name,
DROP COLUMN IF EXISTS password_hash,
DROP COLUMN IF EXISTS email,
DROP COLUMN IF EXISTS username;