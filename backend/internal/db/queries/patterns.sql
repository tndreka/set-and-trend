-- name: InsertPatternDetection :one
INSERT INTO pattern_detections (
    symbol, timeframe, detected_candle_idx,
    left_shoulder_price, left_shoulder_idx,
    head_price, head_idx,
    right_shoulder_price, right_shoulder_idx,
    neckline_price, neckline_idx,
    shoulder_symmetry, head_prominence, time_symmetry,
    volume_profile, neckline_quality,
    context_trend, volatility_regime, context_dist_ema200, recent_swings,
    overall_confidence, final_confidence, pattern_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
)
RETURNING *;

-- name: GetPatternDetectionByID :one
SELECT * FROM pattern_detections WHERE id = $1;

-- name: GetPatternDetectionsByTimeframe :many
SELECT * FROM pattern_detections 
WHERE symbol = $1 AND timeframe = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: GetPatternDetectionsByConfidence :many
SELECT * FROM pattern_detections 
WHERE symbol = $1 AND final_confidence >= $2
ORDER BY final_confidence DESC
LIMIT $3;

-- name: CountPatternDetections :one
SELECT COUNT(*) FROM pattern_detections 
WHERE symbol = $1 AND timeframe = $2;

-- name: InsertSignal :one
INSERT INTO signals (
    pattern_detection_id, symbol, timeframe, execution_timeframe,
    direction, detected_price, theoretical_sl, theoretical_tp,
    projected_rr, confidence, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSignalByID :one
SELECT * FROM signals WHERE id = $1;

-- name: GetSignalsByStatus :many
SELECT * FROM signals 
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetSignalsByPattern :many
SELECT * FROM signals 
WHERE pattern_detection_id = $1;

-- name: UpdateSignalStatus :one
UPDATE signals SET status = $2 WHERE id = $1
RETURNING *;

-- name: CountSignalsByStatus :one
SELECT COUNT(*) FROM signals WHERE status = $1;
