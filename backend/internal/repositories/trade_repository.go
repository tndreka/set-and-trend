package repositories

import (
	"context"

	"github.com/google/uuid"
	"set-and-trend/backend/internal/db"
)

type TradeRepository struct {
	q *db.Queries
}

func NewTradeRepository(q *db.Queries) *TradeRepository {
	return &TradeRepository{q: q}
}

//
// CREATE
//

func (r *TradeRepository) CreateTrade(
	ctx context.Context,
	params db.CreateTradeParams,
) (db.Trade, error) {
	return r.q.CreateTrade(ctx, params)
}

//
// READ
//

func (r *TradeRepository) GetTradeByID(
	ctx context.Context,
	tradeID uuid.UUID,
) (db.Trade, error) {
	return r.q.GetTradeByID(ctx, tradeID)
}

func (r *TradeRepository) GetTradesByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int32,
) ([]db.Trade, error) {
	return r.q.GetTradesByUserID(ctx, db.GetTradesByUserIDParams{
		UserID: userID,
		Limit:  limit,
	})
}

func (r *TradeRepository) GetTradesByAccountAndCandle(
	ctx context.Context,
	accountID uuid.UUID,
	candleID uuid.UUID,
) ([]db.Trade, error) {
	return r.q.GetTradesByAccountAndCandle(ctx, db.GetTradesByAccountAndCandleParams{
		AccountID: accountID,
		CandleID:  candleID,
	})
}

//
// EXECUTION EVENTS
//

func (r *TradeRepository) CreateTradeExecution(
	ctx context.Context,
	params db.CreateTradeExecutionParams,
) (db.TradeExecution, error) {
	return r.q.CreateTradeExecution(ctx, params)
}

func (r *TradeRepository) GetTradeExecutions(
	ctx context.Context,
	tradeID uuid.UUID,
) ([]db.TradeExecution, error) {
	return r.q.GetTradeExecutions(ctx, tradeID)
}
