package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/domain"
	"set-and-trend/backend/internal/repositories"
)

type ExecutionService struct {
	pool          *pgxpool.Pool
	tradeRepo     *repositories.TradeRepository
	executionRepo *repositories.ExecutionRepository
	intentRepo    *repositories. IntentRepository
}

func NewExecutionService(
	pool *pgxpool.Pool,
	tradeRepo *repositories.TradeRepository,
	executionRepo *repositories.ExecutionRepository,
	intentRepo *repositories.IntentRepository,
) *ExecutionService {
	return &ExecutionService{
		pool:          pool,
		tradeRepo:     tradeRepo,
		executionRepo: executionRepo,
		intentRepo:    intentRepo,
	}
}

// RecordExecution records a market execution with SERIALIZABLE isolation
func (s *ExecutionService) RecordExecution(
	ctx context.Context,
	tradeID uuid.UUID,
	eventType string,
	price float64,
	positionSize float64,
	reason string,
) (*repositories.TradeExecution, error) {
	// Validate event type
	if ! domain.IsValidExecutionEvent(eventType) {
		return nil, fmt.Errorf("invalid execution event type: %s", eventType)
	}
	
	// CRITICAL: Use SERIALIZABLE transaction to prevent race conditions
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction:  %w", err)
	}
	defer tx.Rollback(ctx)
	
	// Set isolation level to SERIALIZABLE
	if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return nil, fmt.Errorf("set isolation level: %w", err)
	}
	
	// 1. Load trade
	trade, err := s.tradeRepo.GetTradeByID(ctx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get trade: %w", err)
	}
	
	// 2. Load existing executions
	executions, err := s.executionRepo.GetExecutionsByTradeIDTx(ctx, tx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get executions: %w", err)
	}
	
	// 3. Load intent (if any)
	intent, err := s.intentRepo.GetIntentByTradeIDTx(ctx, tx, tradeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt. Errorf("get intent: %w", err)
	}
	
	// 4. Validate state allows execution
	tradeExecs := mapToTradeExecutions(executions)
	tradeIntent := mapToTradeIntent(intent)
	
	if err := ValidateTradeExecutable(tradeExecs, tradeIntent); err != nil {
		return nil, err
	}
	
	// 5. Validate execution size
	plannedSize, err := parseDecimal(trade.PlannedPositionSize)
	if err != nil {
		return nil, fmt.Errorf("parse planned size: %w", err)
	}
	
	if err := ValidateExecutionSize(
		eventType,
		positionSize,
		plannedSize,
		tradeExecs,
	); err != nil {
		return nil, err
	}
	
	// 6. Compute PnL if closing
	var pnl, pnlPips *float64
	if domain.IsClosingEvent(domain.ExecutionEventType(eventType)) {
		pnlMoney, pnlPipsVal, err := ComputePnL(
			trade.Bias,
			tradeExecs,
			price,
			positionSize,
			0.0001, // EURUSD pip value
		)
		if err != nil {
			return nil, fmt. Errorf("compute pnl:  %w", err)
		}
		pnl = &pnlMoney
		pnlPips = &pnlPipsVal
	}
	
	// 7. Insert execution
	reasonPtr := &reason
	if reason == "" {
		reasonPtr = nil
	}
	
	execution, err := s.executionRepo. CreateExecutionTx(ctx, tx, repositories.CreateExecutionParams{
		TradeID:      tradeID,
		EventType:    eventType,
		Price:        &price,
		PositionSize: &positionSize,
		ExecutedAt:   time.Now(),
		Reason:       reasonPtr,
		PnL:          pnl,
		PnLPips:       pnlPips,
	})
	if err != nil {
		return nil, fmt. Errorf("create execution: %w", err)
	}
	
	// 8. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	
	log.Info().
		Str("trade_id", tradeID. String()).
		Str("event_type", eventType).
		Float64("price", price).
		Msg("execution recorded")
	
	return execution, nil
}

// RecordIntent records a user decision (cancel/invalidate)
func (s *ExecutionService) RecordIntent(
	ctx context.Context,
	tradeID uuid.UUID,
	intentType string,
	reason string,
) (*repositories.TradeIntent, error) {
	// Validate intent type
	if !domain.IsValidIntent(intentType) {
		return nil, fmt.Errorf("invalid intent type: %s", intentType)
	}
	
	// Use SERIALIZABLE transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx. Rollback(ctx)
	
	// Set isolation level
	if _, err := tx. Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return nil, fmt.Errorf("set isolation level: %w", err)
	}
	
	// 1. Check if trade has executions
	executions, err := s.executionRepo.GetExecutionsByTradeIDTx(ctx, tx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get executions: %w", err)
	}
	
	if len(executions) > 0 {
		return nil, fmt.Errorf("cannot %s:  trade has been executed", intentType)
	}
	
	// 2. Insert intent
	intent, err := s.intentRepo.CreateIntentTx(ctx, tx, repositories. CreateIntentParams{
		TradeID:    tradeID,
		IntentType: intentType,
		Reason:     reason,
	})
	if err != nil {
		return nil, fmt. Errorf("create intent: %w", err)
	}
	
	// 3. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	
	log.Info().
		Str("trade_id", tradeID.String()).
		Str("intent_type", intentType).
		Msg("intent recorded")
	
	return intent, nil
}

// GetTradeState derives the current state of a trade
func (s *ExecutionService) GetTradeState(ctx context.Context, tradeID uuid.UUID) (TradeState, error) {
	// Load executions
	executions, err := s. executionRepo.GetExecutionsByTradeID(ctx, tradeID)
	if err != nil {
		return "", fmt.Errorf("get executions: %w", err)
	}
	
	// Load intent
	intent, err := s.intentRepo.GetIntentByTradeID(ctx, tradeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt. Errorf("get intent: %w", err)
	}
	
	// Derive state
	return DeriveTradeState(
		mapToTradeExecutions(executions),
		mapToTradeIntent(intent),
	)
}

// Helper functions
func mapToTradeExecutions(execs []repositories.TradeExecution) []TradeExecution {
	result := make([]TradeExecution, len(execs))
	for i, e := range execs {
		result[i] = TradeExecution{
			EventType:    e.EventType,
			Price:        parseFloatPtr(e.Price),
			PositionSize: parseFloatPtr(e.PositionSize),
			ExecutedAt:   e.ExecutedAt,
			PnL:          parseFloatPtr(e.PnL),
			PnLPips:      parseFloatPtr(e.PnLPips),
		}
	}
	return result
}

func mapToTradeIntent(i *repositories.TradeIntent) *TradeIntent {
	if i == nil {
		return nil
	}
	return &TradeIntent{
		IntentType: i.IntentType,
		Reason:     i.Reason,
		CreatedAt:  i.CreatedAt,
	}
}

func parseFloatPtr(s *string) float64 {
	if s == nil {
		return 0
	}
	var f float64
	fmt.Sscanf(*s, "%f", &f)
	return f
}

func parseDecimal(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
