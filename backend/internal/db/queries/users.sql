-- name: GetUser :one
SELECT id, email, username, timezone, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (
  id, email, username, password, timezone
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING id, email, username, timezone, created_at, updated_at;

-- name: ListUsers :many
SELECT id, email, username, timezone, created_at, updated_at
FROM users
ORDER BY created_at DESC;
