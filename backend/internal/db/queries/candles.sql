-- name: CreateCandle :one
INSERT INTO candles_weekly (
  id, timestamp_utc, open, high, low, close, volume
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetCandles :many
SELECT * FROM candles_weekly 
WHERE timestamp_utc >= $1 AND timestamp_utc <= $2 
ORDER BY timestamp_utc DESC
LIMIT $3;
