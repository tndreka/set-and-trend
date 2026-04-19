package services

import (
	"context"
//	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
//	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/domain"
	"set-and-trend/backend/internal/repositories"
)

type ExecutionService struct {
	pool          *pgxpool.Pool
	tradeRepo     *repositories.TradeRepository
	executionRepo *repositories.ExecutionRepository
	intentRepo    *repositories.IntentRepository
	accountRepo   *repositories.AccountRepository
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
	accountRepo *repositories.AccountRepository,
	pool *pgxpool.Pool,
) *ExecutionService {
	return &ExecutionService{
		tradeRepo:     tradeRepo,
		executionRepo: executionRepo,
		intentRepo:    intentRepo,
		accountRepo:   accountRepo,
		pool:          pool,
	}
}

// computePositionSizeFromTrade recomputes position size from the trade's stored risk parameters.
func (s *ExecutionService) computePositionSizeFromTrade(ctx context.Context, trade db.Trade) float64 {
	account, err := s.accountRepo.GetAccountByID(ctx, trade.AccountID)
	if err != nil {
		log.Warn().Err(err).Msg("could not load account for sizing, defaulting to 1.0")
		return 1.0
	}
	symCfg, err := constants.GetSymbolConfig(trade.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", trade.Symbol).Msg("unknown symbol for sizing, defaulting to 1.0")
		return 1.0
	}
	balance := account.Balance.InexactFloat64()
	riskPct := trade.RiskPercent.InexactFloat64()
	entry := trade.PlannedEntry.InexactFloat64()
	sl := trade.StopLoss.InexactFloat64()

	riskAmount, err := ComputeRiskAmount(balance, riskPct)
	if err != nil {
		return 1.0
	}
	stopDist, err := ComputeStopDistance(entry, sl)
	if err != nil {
		return 1.0
	}
	stopPips, err := ComputeStopDistancePips(stopDist, symCfg.PipSize)
	if err != nil {
		return 1.0
	}
	size, err := ComputePositionSize(riskAmount, stopPips, symCfg.PipValue)
	if err != nil {
		return 1.0
	}
	return size
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

	// TODO: Re-enable when trade_intents table is created
	var intent *repositories.TradeIntent = nil
	//intent, err := s.intentRepo.GetIntentByTradeIDTx(ctx, tx, tradeID)
	//if err != nil && !errors.Is(err, pgx.ErrNoRows) {
	//	return nil, fmt.Errorf("get intent: %w", err)
	//}

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
		symCfg, err := constants.GetSymbolConfig(trade.Symbol)
		if err != nil {
			return nil, fmt.Errorf("unknown symbol %s for PnL: %w", trade.Symbol, err)
		}
		pnlMoney, pnlPipsVal, err := ComputePnL(
			strings.ToLower(string(trade.Direction)),
			tradeExecs,
			price,
			positionSize,
			symCfg.PipSize,
			symCfg.PipValue,
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
			Quantity: &positionSize,
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

	positionSize := s.computePositionSizeFromTrade(ctx, trade)

	reason := ""
	if input.Reason != nil {
		reason = *input.Reason
	}

	_, err = s.RecordExecution(
		ctx,
		input.TradeID,
		string(domain.EventEntry),
		input.ActualEntry,
		positionSize,
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

	plannedSize := s.computePositionSizeFromTrade(ctx, trade)

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

	//intent, err := s.intentRepo.GetIntentByTradeID(ctx, tradeID)
	//if err != nil && !errors.Is(err, pgx.ErrNoRows) {
	//	return "", err
	//}
	
	var intent *repositories.TradeIntent = nil

	return DeriveTradeState(
		mapToTradeExecutions(executions),
		mapToTradeIntent(intent),
	)
}
