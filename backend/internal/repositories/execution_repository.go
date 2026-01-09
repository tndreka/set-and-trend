package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type ExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionRepository(pool *pgxpool.Pool) *ExecutionRepository {
	return &ExecutionRepository{pool:  pool}
}

// TradeExecution represents an execution event
type TradeExecution struct {
	ID           uuid.UUID  `json:"id"`
	TradeID      uuid.UUID  `json:"trade_id"`
	EventType    string     `json:"event_type"`
	Price        *string    `json:"price,omitempty"`
	PositionSize *string    `json:"position_size,omitempty"`
	PnL          *string    `json:"pnl,omitempty"`
	PnLPips      *string    `json:"pnl_pips,omitempty"`
	ExecutedAt   time.Time  `json:"executed_at"`
	Session      *string    `json:"session,omitempty"`
	Reason       *string    `json:"reason,omitempty"`
	SlippagePips *string    `json:"slippage_pips,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateExecutionParams contains parameters for creating an execution event
type CreateExecutionParams struct {
	TradeID      uuid. UUID
	EventType    string
	Price        *float64
	PositionSize *float64
	PnL          *float64
	PnLPips      *float64
	ExecutedAt   time.Time
	Session      *string
	Reason       *string
	SlippagePips *float64
}

// CreateExecution inserts a new execution event
func (r *ExecutionRepository) CreateExecution(ctx context.Context, params CreateExecutionParams) (*TradeExecution, error) {
	// Convert floats to decimals for storage
	var priceStr, sizeStr, pnlStr, pnlPipsStr, slippageStr *string
	
	if params.Price != nil {
		s := decimal.NewFromFloat(*params.Price).String()
		priceStr = &s
	}
	if params.PositionSize != nil {
		s := decimal.NewFromFloat(*params.PositionSize).String()
		sizeStr = &s
	}
	if params.PnL != nil {
		s := decimal. NewFromFloat(*params.PnL).String()
		pnlStr = &s
	}
	if params.PnLPips != nil {
		s := decimal.NewFromFloat(*params.PnLPips).String()
		pnlPipsStr = &s
	}
	if params.SlippagePips != nil {
		s := decimal.NewFromFloat(*params. SlippagePips).String()
		slippageStr = &s
	}
	
	var exec TradeExecution
	
	err := r.pool.QueryRow(ctx, `
		INSERT INTO trade_executions (
			id, trade_id, event_type, price, position_size, 
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, trade_id, event_type, price, position_size, 
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
	`, uuid.New(), params.TradeID, params. EventType, 
		priceStr, sizeStr, params.ExecutedAt, params.Session, params.Reason, 
		slippageStr, pnlStr, pnlPipsStr).Scan(
		&exec.ID,
		&exec.TradeID,
		&exec.EventType,
		&exec.Price,
		&exec.PositionSize,
		&exec.ExecutedAt,
		&exec. Session,
		&exec. Reason,
		&exec.SlippagePips,
		&exec.PnL,
		&exec.PnLPips,
		&exec.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}
	
	return &exec, nil
}

// GetExecutionsByTradeID retrieves all execution events for a trade
func (r *ExecutionRepository) GetExecutionsByTradeID(ctx context.Context, tradeID uuid.UUID) ([]TradeExecution, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, trade_id, event_type, price, position_size,
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
		FROM trade_executions
		WHERE trade_id = $1
		ORDER BY executed_at ASC
	`, tradeID)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows. Close()
	
	var executions []TradeExecution
	for rows.Next() {
		var exec TradeExecution
		err := rows.Scan(
			&exec.ID,
			&exec.TradeID,
			&exec.EventType,
			&exec.Price,
			&exec.PositionSize,
			&exec.ExecutedAt,
			&exec.Session,
			&exec.Reason,
			&exec.SlippagePips,
			&exec. PnL,
			&exec.PnLPips,
			&exec.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan execution: %w", err)
		}
		executions = append(executions, exec)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt. Errorf("rows error: %w", err)
	}
	
	return executions, nil
}
// CreateExecutionTx inserts execution within a transaction
func (r *ExecutionRepository) CreateExecutionTx(ctx context.Context, tx pgx.Tx, params CreateExecutionParams) (*TradeExecution, error) {
	// Convert floats to decimals for storage
	var priceStr, sizeStr, pnlStr, pnlPipsStr, slippageStr *string
	
	if params.Price != nil {
		s := decimal.NewFromFloat(*params.Price).String()
		priceStr = &s
	}
	if params. PositionSize != nil {
		s := decimal.NewFromFloat(*params.PositionSize).String()
		sizeStr = &s
	}
	if params.PnL != nil {
		s := decimal.NewFromFloat(*params.PnL).String()
		pnlStr = &s
	}
	if params.PnLPips != nil {
		s := decimal. NewFromFloat(*params.PnLPips).String()
		pnlPipsStr = &s
	}
	if params.SlippagePips != nil {
		s := decimal.NewFromFloat(*params. SlippagePips).String()
		slippageStr = &s
	}
	
	var exec TradeExecution
	
	err := tx.QueryRow(ctx, `
		INSERT INTO trade_executions (
			id, trade_id, event_type, price, position_size, 
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, trade_id, event_type, price, position_size, 
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
	`, uuid.New(), params.TradeID, params. EventType, 
		priceStr, sizeStr, params. ExecutedAt, params.Session, params.Reason, 
		slippageStr, pnlStr, pnlPipsStr).Scan(
		&exec.ID,
		&exec. TradeID,
		&exec.EventType,
		&exec.Price,
		&exec. PositionSize,
		&exec.ExecutedAt,
		&exec.Session,
		&exec.Reason,
		&exec.SlippagePips,
		&exec.PnL,
		&exec.PnLPips,
		&exec.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt. Errorf("create execution (tx): %w", err)
	}
	
	return &exec, nil
}

// GetExecutionsByTradeIDTx retrieves executions within a transaction
func (r *ExecutionRepository) GetExecutionsByTradeIDTx(ctx context. Context, tx pgx.Tx, tradeID uuid.UUID) ([]TradeExecution, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, trade_id, event_type, price, position_size,
			executed_at, session, reason, slippage_pips, pnl, pnl_pips, created_at
		FROM trade_executions
		WHERE trade_id = $1
		ORDER BY executed_at ASC
	`, tradeID)
	if err != nil {
		return nil, fmt. Errorf("query executions (tx): %w", err)
	}
	defer rows.Close()
	
	var executions []TradeExecution
	for rows.Next() {
		var exec TradeExecution
		err := rows.Scan(
			&exec.ID,
			&exec.TradeID,
			&exec.EventType,
			&exec.Price,
			&exec.PositionSize,
			&exec.ExecutedAt,
			&exec.Session,
			&exec.Reason,
			&exec.SlippagePips,
			&exec.PnL,
			&exec.PnLPips,
			&exec.CreatedAt,
		)
		if err != nil {
			return nil, fmt. Errorf("scan execution (tx): %w", err)
		}
		executions = append(executions, exec)
	}
	
	if err := rows. Err(); err != nil {
		return nil, fmt.Errorf("rows error (tx): %w", err)
	}
	
	return executions, nil
}
