package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	intentRepo    *repositories.IntentRepository
}

type ExecuteTradeInput struct {
	TradeID     uuid.UUID
	ActualEntry float64
	ExecutedAt  time.Time
	Reason      *string
}

type CloseTradeInput struct {
	TradeID    uuid.UUID
	ClosePrice float64
	ExecutedAt time.Time
	Reason     *string
}

type CancelTradeInput struct {
	TradeID    uuid.UUID
	ExecutedAt time.Time
	Reason     string
}

func NewExecutionService(
	tradeRepo *repositories.TradeRepository,
	executionRepo *repositories.ExecutionRepository,
	intentRepo *repositories.IntentRepository,
	pool *pgxpool.Pool,
) *ExecutionService {
	return &ExecutionService{
		tradeRepo:     tradeRepo,
		executionRepo: executionRepo,
		intentRepo:    intentRepo,
		pool:          pool,
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

	if !domain.IsValidExecutionEvent(eventType) {
		return nil, fmt.Errorf("invalid execution event type: %s", eventType)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return nil, fmt.Errorf("set isolation level: %w", err)
	}

	trade, err := s.tradeRepo.GetTradeByID(ctx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get trade: %w", err)
	}

	executions, err := s.executionRepo.GetExecutionsByTradeIDTx(ctx, tx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get executions: %w", err)
	}

	intent, err := s.intentRepo.GetIntentByTradeIDTx(ctx, tx, tradeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get intent: %w", err)
	}

	tradeExecs := mapToTradeExecutions(executions)
	tradeIntent := mapToTradeIntent(intent)

	if err := ValidateTradeExecutable(tradeExecs, tradeIntent); err != nil {
		return nil, err
	}

	// TODO: plannedSize will be computed from risk_percent & stop_loss
	// once calculation is locked in trade creation
	plannedSize := positionSize

	if err := ValidateExecutionSize(
		eventType,
		positionSize,
		plannedSize,
		tradeExecs,
	); err != nil {
		return nil, err
	}

	var pnl, pnlPips *float64
	if domain.IsClosingEvent(domain.ExecutionEventType(eventType)) {
		pnlMoney, pnlPipsVal, err := ComputePnL(
			strings.ToLower(string(trade.Direction)), // LONG -> long
			tradeExecs,
			price,
			positionSize,
			0.0001, // TODO: derive pip value from symbol
		)
		if err != nil {
			return nil, fmt.Errorf("compute pnl: %w", err)
		}
		pnl = &pnlMoney
		pnlPips = &pnlPipsVal
	}

	reasonPtr := &reason
	if reason == "" {
		reasonPtr = nil
	}

	execution, err := s.executionRepo.CreateExecutionTx(
		ctx,
		tx,
		repositories.CreateExecutionParams{
			TradeID:      tradeID,
			ExecutionType:    eventType,
			Price:        &price,
			PositionSize: &positionSize,
			ExecutedAt:   time.Now(),
			Reason:       reasonPtr,
			PnL:          pnl,
			PnLPips:      pnlPips,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	log.Info().
		Str("trade_id", tradeID.String()).
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

	if !domain.IsValidIntent(intentType) {
		return nil, fmt.Errorf("invalid intent type: %s", intentType)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return nil, fmt.Errorf("set isolation level: %w", err)
	}

	executions, err := s.executionRepo.GetExecutionsByTradeIDTx(ctx, tx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get executions: %w", err)
	}
	if len(executions) > 0 {
		return nil, fmt.Errorf("cannot %s: trade already executed", intentType)
	}

	intent, err := s.intentRepo.CreateIntentTx(
		ctx,
		tx,
		repositories.CreateIntentParams{
			TradeID:    tradeID,
			IntentType: intentType,
			Reason:     reason,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create intent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	log.Info().
		Str("trade_id", tradeID.String()).
		Str("intent_type", intentType).
		Msg("intent recorded")

	return intent, nil
}

// ExecuteTrade executes a trade entry
func (s *ExecutionService) ExecuteTrade(ctx context.Context, input ExecuteTradeInput) error {
	trade, err := s.tradeRepo.GetTradeByID(ctx, input.TradeID)
	if err != nil {
		return fmt.Errorf("get trade: %w", err)
	}

	_ = trade // intentional – needed later when sizing is implemented

	plannedSize := 1.0 // TODO: calculate from account + risk_percent + stop_loss

	reason := ""
	if input.Reason != nil {
		reason = *input.Reason
	}

	_, err = s.RecordExecution(
		ctx,
		input.TradeID,
		string(domain.EventEntry),
		input.ActualEntry,
		plannedSize,
		reason,
	)
	return err
}

// CloseTrade closes a trade position
func (s *ExecutionService) CloseTrade(ctx context.Context, input CloseTradeInput) error {
	executions, err := s.executionRepo.GetExecutionsByTradeID(ctx, input.TradeID)
	if err != nil {
		return fmt.Errorf("get executions: %w", err)
	}

	trade, err := s.tradeRepo.GetTradeByID(ctx, input.TradeID)
	if err != nil {
		return fmt.Errorf("get trade: %w", err)
	}

	_ = trade // will be used once sizing is stored

	plannedSize := 1.0 // TODO: calculate from stored calculated_position_size

	tradeExecs := mapToTradeExecutions(executions)
	remainingSize, err := ComputeRemainingPosition(plannedSize, tradeExecs)
	if err != nil {
		return fmt.Errorf("compute remaining: %w", err)
	}

	reason := ""
	if input.Reason != nil {
		reason = *input.Reason
	}

	_, err = s.RecordExecution(
		ctx,
		input.TradeID,
		string(domain.EventManualClose),
		input.ClosePrice,
		remainingSize,
		reason,
	)
	return err
}

// CancelTrade cancels a planned trade
func (s *ExecutionService) CancelTrade(ctx context.Context, input CancelTradeInput) error {
	_, err := s.RecordIntent(
		ctx,
		input.TradeID,
		string(domain.IntentCancel),
		input.Reason,
	)
	return err
}

func (s *ExecutionService) GetTradeState(
	ctx context.Context,
	tradeID uuid.UUID,
) (TradeState, error) {

	executions, err := s.executionRepo.GetExecutionsByTradeID(ctx, tradeID)
	if err != nil {
		return "", err
	}

	intent, err := s.intentRepo.GetIntentByTradeID(ctx, tradeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	return DeriveTradeState(
		mapToTradeExecutions(executions),
		mapToTradeIntent(intent),
	)
}
