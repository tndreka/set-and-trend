-- name: CreateAccount :one
INSERT INTO accounts (
  id, user_id, type, broker_name, currency, balance, leverage,
  max_risk_per_trade_pct, max_daily_risk_pct, timezone, preferred_session
) VALUES (
  $1, $2, $3::account_type, $4, $5, $6, $7, $8, $9, $10, $11::session_type
)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1;

-- name: ListAccountsByUser :many
SELECT * FROM accounts WHERE user_id = $1 ORDER BY updated_at DESC;
