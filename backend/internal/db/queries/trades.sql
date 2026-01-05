-- name: CreateTrade :one
INSERT INTO trades (
    id,
    user_id,
    account_id,
    candle_id,
    symbol,
    timeframe,
    setup_timestamp_utc,
    account_balance_at_setup,
    leverage_at_setup,
    max_risk_per_trade_pct_at_setup,
    timezone_at_setup,
    bias,
    planned_entry,
    planned_sl,
    planned_tp,
    planned_rr,
    planned_risk_pct,
    planned_risk_amount,
    planned_position_size,
    reason_for_trade,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW()
)
RETURNING *;

-- name: GetTradeByID :one
SELECT * FROM trades WHERE id = $1;

-- name: GetTradesByUserID :many
SELECT * FROM trades 
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: UpdateTradeExecution :exec
UPDATE trades
SET
    actual_entry = $2,
    actual_sl = $3,
    actual_tp = $4,
    actual_risk_pct = $5,
    actual_risk_amount = $6,
    actual_position_size = $7,
    execution_timestamp_utc = $8
WHERE id = $1;

-- name: UpdateTradeClosure :exec
UPDATE trades
SET
    close_timestamp_utc = $2,
    close_price = $3,
    result = $4,
    pips_gained = $5,
    money_gained = $6,
    rr_realized = $7,
    duration_seconds = $8,
    session = $9
WHERE id = $1;
