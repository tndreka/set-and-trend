-- name: CreateAccount :one
INSERT INTO accounts (
    id,
    user_id,
    type,
    broker_name,
    currency,
    balance,
    leverage,
    max_risk_per_trade_pct,
    max_daily_risk_pct,
    timezone,
    preferred_session,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: GetAccountsByUserID :many
SELECT * FROM accounts WHERE user_id = $1;
