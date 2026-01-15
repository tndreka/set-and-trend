-- name: CreateTrade :one
INSERT INTO trades (
    user_id,
    account_id,
    candle_id,
    symbol,
    timeframe,
    direction,
    planned_entry,
    stop_loss,
    take_profit,
    risk_percent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetTradeByID :one
SELECT * FROM trades
WHERE id = $1;

-- name: GetTradesByUserID :many
SELECT * FROM trades
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetTradesByAccountAndCandle :many
SELECT * FROM trades
WHERE account_id = $1
  AND candle_id = $2
ORDER BY created_at DESC;

-- name: CreateTradeExecution :one
INSERT INTO trade_executions (
    trade_id,
    execution_type,
    price,
    quantity
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetTradeExecutions :many
SELECT * FROM trade_executions
WHERE trade_id = $1
ORDER BY executed_at ASC;
