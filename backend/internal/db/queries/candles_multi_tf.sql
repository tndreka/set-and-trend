-- name: GetCandlesD1ByRange :many
SELECT * FROM candles_d1 
WHERE timestamp_utc >= $1 AND timestamp_utc <= $2
ORDER BY timestamp_utc ASC;

-- name: GetCandlesD1Last :many
SELECT * FROM candles_d1 
ORDER BY timestamp_utc DESC
LIMIT $1;

-- name: GetCandleD1ByTimestamp :one
SELECT * FROM candles_d1 
WHERE timestamp_utc = $1;

-- name: InsertCandleD1 :one
INSERT INTO candles_d1 (timestamp_utc, open, high, low, close, volume)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (timestamp_utc) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume
RETURNING *;

-- name: GetCandlesH4ByRange :many
SELECT * FROM candles_h4 
WHERE timestamp_utc >= $1 AND timestamp_utc <= $2
ORDER BY timestamp_utc ASC;

-- name: GetCandlesH4Last :many
SELECT * FROM candles_h4 
ORDER BY timestamp_utc DESC
LIMIT $1;

-- name: GetCandleH4ByTimestamp :one
SELECT * FROM candles_h4 
WHERE timestamp_utc = $1;

-- name: InsertCandleH4 :one
INSERT INTO candles_h4 (timestamp_utc, open, high, low, close, volume)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (timestamp_utc) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume
RETURNING *;

-- name: GetCandlesH1ByRange :many
SELECT * FROM candles_h1 
WHERE timestamp_utc >= $1 AND timestamp_utc <= $2
ORDER BY timestamp_utc ASC;

-- name: GetCandlesH1Last :many
SELECT * FROM candles_h1 
ORDER BY timestamp_utc DESC
LIMIT $1;

-- name: GetCandleH1ByTimestamp :one
SELECT * FROM candles_h1 
WHERE timestamp_utc = $1;

-- name: InsertCandleH1 :one
INSERT INTO candles_h1 (timestamp_utc, open, high, low, close, volume)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (timestamp_utc) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume
RETURNING *;

-- name: CountCandlesD1 :one
SELECT COUNT(*) FROM candles_d1;

-- name: CountCandlesH4 :one
SELECT COUNT(*) FROM candles_h4;

-- name: CountCandlesH1 :one
SELECT COUNT(*) FROM candles_h1;

-- name: GetCandlesWeeklyByRange :many
SELECT * FROM candles_weekly 
WHERE timestamp_utc >= $1 AND timestamp_utc <= $2
ORDER BY timestamp_utc ASC;

-- name: GetCandlesWeeklyBySymbolAndRange :many
SELECT * FROM candles_weekly 
WHERE symbol = $1 AND timestamp_utc >= $2 AND timestamp_utc <= $3
ORDER BY timestamp_utc ASC;

-- name: GetCandlesD1BySymbolAndRange :many
SELECT * FROM candles_d1 
WHERE symbol = $1 AND timestamp_utc >= $2 AND timestamp_utc <= $3
ORDER BY timestamp_utc ASC;

-- name: GetCandlesH4BySymbolAndRange :many
SELECT * FROM candles_h4 
WHERE symbol = $1 AND timestamp_utc >= $2 AND timestamp_utc <= $3
ORDER BY timestamp_utc ASC;

-- name: GetCandlesH1BySymbolAndRange :many
SELECT * FROM candles_h1 
WHERE symbol = $1 AND timestamp_utc >= $2 AND timestamp_utc <= $3
ORDER BY timestamp_utc ASC;
