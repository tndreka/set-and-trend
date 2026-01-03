-- name: CreateRuleResult :exec
INSERT INTO rule_results (
    id,
    rule_id,
    candle_id,
    result,
    confidence_score,
    evaluated_at
)
SELECT 
    gen_random_uuid(),
    r.id,
    @candle_id,
    @result::rule_result_type,
    @confidence,
    NOW()
FROM rules r
WHERE r.code = @rule_code
ON CONFLICT (rule_id, candle_id) DO NOTHING;

-- name: GetRuleResultsByCandleID :many
SELECT 
    rr.*,
    r.code as rule_code,
    r.name as rule_name
FROM rule_results rr
JOIN rules r ON rr.rule_id = r.id
WHERE rr.candle_id = $1
ORDER BY r.code;

-- name: TruncateRuleResults :exec
TRUNCATE TABLE rule_results;
