-- name: CreateUser :one
INSERT INTO users (
    id, 
    username, 
    email, 
    password_hash, 
    name, 
    surname,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
RETURNING id, username, email, name, surname, is_email_verified, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT 
    id, 
    username, 
    email, 
    password_hash, 
    name, 
    surname, 
    is_email_verified,
    last_login,
    created_at,
    updated_at
FROM users
WHERE username = $1;

-- name: GetUserByEmail :one
SELECT 
    id, 
    username, 
    email, 
    password_hash, 
    name, 
    surname, 
    is_email_verified,
    last_login,
    created_at,
    updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT 
    id, 
    username, 
    email, 
    name, 
    surname, 
    is_email_verified,
    last_login,
    created_at,
    updated_at
FROM users
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login = NOW()
WHERE id = $1;

-- name: SetEmailVerificationToken :exec
UPDATE users
SET email_verification_token = $2
WHERE id = $1;

-- name: VerifyEmail :exec
UPDATE users
SET 
    is_email_verified = true,
    email_verification_token = NULL,
    updated_at = NOW()
WHERE email_verification_token = $1;

-- name: SetPasswordResetToken :exec
UPDATE users
SET 
    password_reset_token = $2,
    password_reset_expires = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ResetPassword :exec
UPDATE users
SET 
    password_hash = $2,
    password_reset_token = NULL,
    password_reset_expires = NULL,
    updated_at = NOW()
WHERE password_reset_token = $1 
AND password_reset_expires > NOW();

-- name: UpdateUserProfile :exec
UPDATE users
SET 
    name = COALESCE($2, name),
    surname = COALESCE($3, surname),
    updated_at = NOW()
WHERE id = $1;
