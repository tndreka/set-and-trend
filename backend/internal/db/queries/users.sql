-- name: CreateUser :one
INSERT INTO users (id) 
VALUES ($1)
RETURNING id, created_at;

-- name: GetUser :one
SELECT id, created_at FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, created_at FROM users ORDER BY created_at DESC;
