package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IntentRepository struct {
	pool *pgxpool.Pool
}

func NewIntentRepository(pool *pgxpool.Pool) *IntentRepository {
	return &IntentRepository{pool:  pool}
}

// TradeIntent represents a user/system intent
type TradeIntent struct {
	ID         uuid.UUID `json:"id"`
	TradeID    uuid.UUID `json:"trade_id"`
	IntentType string    `json:"intent_type"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateIntentParams contains parameters for creating an intent
type CreateIntentParams struct {
	TradeID    uuid.UUID
	IntentType string
	Reason     string
}

// CreateIntent records a cancel or invalidate intent
func (r *IntentRepository) CreateIntent(ctx context.Context, params CreateIntentParams) (*TradeIntent, error) {
	var intent TradeIntent
	
	err := r. pool.QueryRow(ctx, `
		INSERT INTO trade_intents (id, trade_id, intent_type, reason, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, trade_id, intent_type, reason, created_at
	`, uuid.New(), params.TradeID, params.IntentType, params. Reason).Scan(
		&intent.ID,
		&intent.TradeID,
		&intent.IntentType,
		&intent.Reason,
		&intent.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("create intent: %w", err)
	}
	
	return &intent, nil
}

// GetIntentByTradeID retrieves intent for a trade (if exists)
func (r *IntentRepository) GetIntentByTradeID(ctx context.Context, tradeID uuid.UUID) (*TradeIntent, error) {
	var intent TradeIntent
	
	err := r.pool.QueryRow(ctx, `
		SELECT id, trade_id, intent_type, reason, created_at
		FROM trade_intents
		WHERE trade_id = $1
	`, tradeID).Scan(
		&intent.ID,
		&intent.TradeID,
		&intent.IntentType,
		&intent.Reason,
		&intent.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil // No intent is valid (trade not cancelled)
	}
	
	if err != nil {
		return nil, fmt.Errorf("query intent: %w", err)
	}
	
	return &intent, nil
}

// CreateIntentTx creates intent within a transaction
func (r *IntentRepository) CreateIntentTx(ctx context.Context, tx pgx.Tx, params CreateIntentParams) (*TradeIntent, error) {
	var intent TradeIntent
	
	err := tx.QueryRow(ctx, `
		INSERT INTO trade_intents (id, trade_id, intent_type, reason, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, trade_id, intent_type, reason, created_at
	`, uuid.New(), params.TradeID, params.IntentType, params.Reason).Scan(
		&intent.ID,
		&intent.TradeID,
		&intent.IntentType,
		&intent. Reason,
		&intent.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("create intent (tx): %w", err)
	}
	
	return &intent, nil
}

// GetIntentByTradeIDTx retrieves intent within a transaction
func (r *IntentRepository) GetIntentByTradeIDTx(ctx context. Context, tx pgx.Tx, tradeID uuid.UUID) (*TradeIntent, error) {
	var intent TradeIntent
	
	err := tx.QueryRow(ctx, `
		SELECT id, trade_id, intent_type, reason, created_at
		FROM trade_intents
		WHERE trade_id = $1
	`, tradeID).Scan(
		&intent.ID,
		&intent.TradeID,
		&intent.IntentType,
		&intent. Reason,
		&intent.CreatedAt,
	)
	
	if err == pgx. ErrNoRows {
		return nil, nil // No intent is valid
	}
	
	if err != nil {
		return nil, fmt.Errorf("query intent (tx): %w", err)
	}
	
	return &intent, nil
}
