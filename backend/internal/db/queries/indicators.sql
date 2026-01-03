-- name: CreateIndicator :one
INSERT INTO indicators_weekly (
    id,
    candle_id,
    ema20,
    ema50,
    ema200,
    range_size,
    body_size,
    upper_wick,
    lower_wick,
    mid_price,
    last_swing_high_price,
    last_swing_low_price
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetIndicatorByCandleID :one
SELECT * FROM indicators_weekly 
WHERE candle_id = $1;

-- name: GetLatestIndicators :many
SELECT 
    i.*,
    c.timestamp_utc,
    c.open,
    c.high,
    c.low,
    c.close
FROM indicators_weekly i
JOIN candles_weekly c ON i.candle_id = c.id
ORDER BY c.timestamp_utc DESC
LIMIT $1;

-- name: GetPreviousIndicatorByTimestamp :one
SELECT * FROM indicators_weekly i
JOIN candles_weekly c ON i.candle_id = c.id
WHERE c.timestamp_utc < $1
ORDER BY c.timestamp_utc DESC
LIMIT 1;
