-- name: InsertIndicatorD1 :one
INSERT INTO indicators_d1 (
    candle_id, ema20, ema50, ema200,
    range_size, body_size, upper_wick, lower_wick,
    mid_price, last_swing_high_price, last_swing_low_price
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (candle_id) DO UPDATE SET
    ema20 = EXCLUDED.ema20,
    ema50 = EXCLUDED.ema50,
    ema200 = EXCLUDED.ema200,
    range_size = EXCLUDED.range_size,
    body_size = EXCLUDED.body_size,
    upper_wick = EXCLUDED.upper_wick,
    lower_wick = EXCLUDED.lower_wick,
    mid_price = EXCLUDED.mid_price,
    last_swing_high_price = EXCLUDED.last_swing_high_price,
    last_swing_low_price = EXCLUDED.last_swing_low_price
RETURNING *;

-- name: GetIndicatorD1ByCandleID :one
SELECT * FROM indicators_d1 WHERE candle_id = $1;

-- name: InsertIndicatorH4 :one
INSERT INTO indicators_h4 (
    candle_id, ema20, ema50, ema200,
    range_size, body_size, upper_wick, lower_wick,
    mid_price, last_swing_high_price, last_swing_low_price
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (candle_id) DO UPDATE SET
    ema20 = EXCLUDED.ema20,
    ema50 = EXCLUDED.ema50,
    ema200 = EXCLUDED.ema200,
    range_size = EXCLUDED.range_size,
    body_size = EXCLUDED.body_size,
    upper_wick = EXCLUDED.upper_wick,
    lower_wick = EXCLUDED.lower_wick,
    mid_price = EXCLUDED.mid_price,
    last_swing_high_price = EXCLUDED.last_swing_high_price,
    last_swing_low_price = EXCLUDED.last_swing_low_price
RETURNING *;

-- name: GetIndicatorH4ByCandleID :one
SELECT * FROM indicators_h4 WHERE candle_id = $1;

-- name: InsertIndicatorH1 :one
INSERT INTO indicators_h1 (
    candle_id, ema20, ema50, ema200,
    range_size, body_size, upper_wick, lower_wick,
    mid_price, last_swing_high_price, last_swing_low_price
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (candle_id) DO UPDATE SET
    ema20 = EXCLUDED.ema20,
    ema50 = EXCLUDED.ema50,
    ema200 = EXCLUDED.ema200,
    range_size = EXCLUDED.range_size,
    body_size = EXCLUDED.body_size,
    upper_wick = EXCLUDED.upper_wick,
    lower_wick = EXCLUDED.lower_wick,
    mid_price = EXCLUDED.mid_price,
    last_swing_high_price = EXCLUDED.last_swing_high_price,
    last_swing_low_price = EXCLUDED.last_swing_low_price
RETURNING *;

-- name: GetIndicatorH1ByCandleID :one
SELECT * FROM indicators_h1 WHERE candle_id = $1;
