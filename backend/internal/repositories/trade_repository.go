package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"set-and-trend/backend/internal/db"
)

type TradeRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool  // pgx pool for transactions
}

func NewTradeRepository(q *db.Queries, pool *pgxpool.Pool) *TradeRepository {
	return &TradeRepository{
		q:    q,
		pool: pool,
	}
}

//
// CREATE with Transaction
//

func (r *TradeRepository) CreateTradeWithExecution(
	ctx context.Context,
	tradeParams db.CreateTradeParams,
	execParams db.CreateTradeExecutionParams,
) (db.Trade, db.TradeExecution, error) {
	// Start transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Trade{}, db.TradeExecution{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Create queries with transaction
	txQueries := r.q.WithTx(tx)

	// Create trade
	trade, err := txQueries.CreateTrade(ctx, tradeParams)
	if err != nil {
		return db.Trade{}, db.TradeExecution{}, fmt.Errorf("create trade: %w", err)
	}

	// Update execution params with trade ID
	execParams.TradeID = trade.ID

	// Create execution
	execution, err := txQueries.CreateTradeExecution(ctx, execParams)
	if err != nil {
		return db.Trade{}, db.TradeExecution{}, fmt.Errorf("create execution: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return db.Trade{}, db.TradeExecution{}, fmt.Errorf("commit transaction: %w", err)
	}

	return trade, execution, nil
}

//
// CREATE (original, kept for backward compatibility)
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