-- name: CreateTradeFeedback :one
INSERT INTO trade_feedback (
    id,
    trade_id,
    followed_plan,
    emotion_before,
    emotion_during,
    emotion_after,
    biggest_mistake,
    screenshot_url
) VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
) RETURNING *;

-- name: GetTradeFeedbackByTradeID :one
SELECT * FROM trade_feedback WHERE trade_id = $1;

-- name: UpdateTradeFeedback :one
UPDATE trade_feedback
SET
    followed_plan = $2,
    emotion_before = $3,
    emotion_during = $4,
    emotion_after = $5,
    biggest_mistake = $6,
    screenshot_url = $7
WHERE trade_id = $1
RETURNING *;

-- name: DeleteTradeFeedback :exec
DELETE FROM trade_feedback WHERE trade_id = $1;
