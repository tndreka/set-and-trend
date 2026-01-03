package repositories

import (
	"context"
	
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/rules"
)

type RuleResultRepository struct {
	q *db.Queries
}

func NewRuleResultRepository(q *db.Queries) *RuleResultRepository {
	return &RuleResultRepository{q: q}
}

type RuleResultCreateParams struct {
	RuleCode    rules.RuleCode
	CandleID    uuid.UUID
	Result      string  // "PASS" or "FAIL"
	Confidence  float64
}

// CreateRuleResult persists a rule evaluation result
// Uses ON CONFLICT DO NOTHING for idempotency
func (r *RuleResultRepository) CreateRuleResult(
	ctx context.Context,
	params RuleResultCreateParams,
) error {
	// Convert types to match SQLC expectations
	resultType := db.RuleResultType(params.Result)
	confidenceDec := decimal.NewFromFloat(params.Confidence)
	
	return r.q.CreateRuleResult(ctx, db.CreateRuleResultParams{
		CandleID:   params.CandleID,
		Result:     resultType,
		Confidence: confidenceDec,
		RuleCode:   string(params.RuleCode),
	})
}

// GetRuleResultsByCandleID retrieves all rule results for a candle
func (r *RuleResultRepository) GetRuleResultsByCandleID(
	ctx context.Context,
	candleID uuid.UUID,
) ([]db.GetRuleResultsByCandleIDRow, error) {
	return r.q.GetRuleResultsByCandleID(ctx, candleID)
}

// TruncateRuleResults deletes all rule results (for backfill)
func (r *RuleResultRepository) TruncateRuleResults(ctx context.Context) error {
	return r.q.TruncateRuleResults(ctx)
}
