-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, created_at)
VALUES ($1, NOW())
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
