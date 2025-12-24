-- name: CreateAccount :one
INSERT INTO accounts (
  id, user_id, broker_name, account_type, currency, balance, leverage, 
  max_risk_per_trade, max_daily_risk
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1;

-- name: ListAccountsByUser :many
SELECT * FROM accounts WHERE user_id = $1 ORDER BY created_at DESC;
