-- name: CreateCandle :one
INSERT INTO candles_weekly (
    id,
    timestamp_utc,
    open,
    high,
    low,
    close,
    volume
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetCandleByTimestamp :one
SELECT * FROM candles_weekly 
WHERE timestamp_utc = $1;

-- name: GetCandlesInRange :many
SELECT * FROM candles_weekly 
WHERE timestamp_utc BETWEEN $1 AND $2
ORDER BY timestamp_utc ASC;

-- name: GetLatestCandles :many
SELECT * FROM candles_weekly 
ORDER BY timestamp_utc DESC
LIMIT $1;

-- name: GetCandleByID :one
SELECT * FROM candles_weekly 
WHERE id = $1;

-- name: GetAllCandlesOrdered :many
SELECT * FROM candles_weekly 
ORDER BY timestamp_utc ASC;

-- name: UpdateIndicatorEMAs :exec
UPDATE indicators_weekly 
SET 
    ema20 = $2,
    ema50 = $3,
    ema200 = $4
WHERE id = $1;
