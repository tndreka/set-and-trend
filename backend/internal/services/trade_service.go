package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/db"
)

type TradeService struct {
	tradeRepo   TradeRepo
	accountRepo AccountRepo
	candleRepo  CandleRepo
}

func NewTradeService(
	tradeRepo TradeRepo,
	accountRepo AccountRepo,
	candleRepo CandleRepo,
) *TradeService {
	return &TradeService{
		tradeRepo:   tradeRepo,
		accountRepo: accountRepo,
		candleRepo:  candleRepo,
	}
}

// CreateTrade orchestrates trade creation with full validation
func (s *TradeService) CreateTrade(ctx context.Context, input CreateTradeInput) (*Trade, error) {
	// Normalize direction to uppercase for DB enum
	direction := strings.ToUpper(input.Direction)
	
	// Validate symbol is supported
	symbolConfig, err := constants.GetSymbolConfig(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("symbol %s not supported: %w", input.Symbol, err)
	}
	
	// 1. Load account
	account, err := s.accountRepo.GetAccountByID(ctx, input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	
	// 2. Verify candle exists
	if _, err := s.candleRepo.GetCandleByID(ctx, input.CandleID); err != nil {
		return nil, fmt.Errorf("candle not found: %w", err)
	}

	// 3. Validate trade geometry
	err = ValidateTradeGeometry(input.PlannedEntry, input.PlannedSL, input.PlannedTP, strings.ToLower(direction))
	if err != nil {
		return nil, fmt.Errorf("invalid geometry: %w", err)
	}

	// 4. Validate risk percentage
	accountMaxRisk := account.MaxRiskPerTradePct.InexactFloat64()
	if input.PlannedRiskPct > accountMaxRisk {
		return nil, fmt.Errorf("planned risk %.2f%% exceeds account max %.2f%%", input.PlannedRiskPct, accountMaxRisk)
	}
	if input.PlannedRiskPct <= 0 {
		return nil, fmt.Errorf("planned risk must be positive")
	}

	// 5. Compute risk math
	balance := account.Balance.InexactFloat64()
	
	riskAmount, err := ComputeRiskAmount(balance, input.PlannedRiskPct)
	if err != nil {
		return nil, fmt.Errorf("risk calculation: %w", err)
	}

	stopDistance, err := ComputeStopDistance(input.PlannedEntry, input.PlannedSL)
	if err != nil {
		return nil, fmt.Errorf("stop distance: %w", err)
	}

	// Use symbol-specific pip size (0.0001 for EURUSD, 0.01 for JPY pairs)
	stopDistancePips, err := ComputeStopDistancePips(stopDistance, symbolConfig.PipSize)
	if err != nil {
		return nil, fmt.Errorf("pip conversion: %w", err)
	}

	// Position sizing using symbol-specific contract size
	positionSize, err := ComputePositionSize(riskAmount, stopDistancePips, symbolConfig.PipValue)
	if err != nil {
		return nil, fmt.Errorf("position sizing: %w", err)
	}

	rr, err := ComputeRR(input.PlannedEntry, input.PlannedSL, input.PlannedTP, strings.ToLower(direction))
	if err != nil {
		return nil, fmt.Errorf("RR calculation: %w", err)
	}

	// 5.5. Check for duplicate trade
	existingTrades, err := s.tradeRepo.GetTradesByAccountAndCandle(ctx, input.AccountID, input.CandleID)
	if err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	}

	for _, existing := range existingTrades {
		if string(existing.Direction) == direction {
			return nil, fmt.Errorf("duplicate trade: account %s already has %s trade on candle %s",
				input.AccountID, direction, input.CandleID)
		}
	}

	// 5.6. Enforce minimum RR
	if rr < constants.MinimumRR {
		return nil, fmt.Errorf("trade rejected: RR %.2f is below minimum %.2f", rr, constants.MinimumRR)
	}

	// 5.7. Validate position size against leverage (using symbol-specific contract size)
	maxPositionSize, err := ComputeMaxPositionSize(balance, int(account.Leverage), symbolConfig.ContractSize)
	if err != nil {
		return nil, fmt.Errorf("leverage check: %w", err)
	}
	if positionSize > maxPositionSize {
		return nil, fmt.Errorf("position size %.2f lots exceeds max %.2f lots (leverage: %dx)",
			positionSize, maxPositionSize, account.Leverage)
	}

	// 6. Create trade - map to DB params
	dbTrade, err := s.tradeRepo.CreateTrade(ctx, db.CreateTradeParams{
		UserID:       account.UserID,
		AccountID:    input.AccountID,
		CandleID:     input.CandleID,
		Symbol:       input.Symbol,  // <-- USE INPUT SYMBOL, not hardcoded EURUSD
		Timeframe:    constants.TimeframeW1,
		Direction:    db.TradeDirection(direction),
		PlannedEntry: decimal.NewFromFloat(input.PlannedEntry),
		StopLoss:     decimal.NewFromFloat(input.PlannedSL),
		TakeProfit:   decimal.NewFromFloat(input.PlannedTP),
		RiskPercent:  decimal.NewFromFloat(input.PlannedRiskPct),
	})
	if err != nil {
		return nil, fmt.Errorf("persist trade: %w", err)
	}

	// 7. Convert db.Trade to service Trade
	return tradeFromDB(dbTrade), nil
}

func tradeFromDB(t db.Trade) *Trade {
	return &Trade{
		ID:           t.ID,
		UserID:       t.UserID,
		AccountID:    t.AccountID,
		CandleID:     t.CandleID,
		Symbol:       t.Symbol,
		Timeframe:    t.Timeframe,
		Direction:    string(t.Direction),
		PlannedEntry: t.PlannedEntry.String(),
		StopLoss:     t.StopLoss.String(),
		TakeProfit:   t.TakeProfit.String(),
		RiskPercent:  t.RiskPercent.String(),
		CreatedAt:    t.CreatedAt.Time.Format(time.RFC3339),
	}
}

// CreateSimpleTrade is the multi-strategy lab path. The handler has already
// resolved an H4 candle id, so we skip the weekly-only candle check and
// honour the caller-supplied timeframe. All other validation (geometry, risk,
// duplicates, leverage, RR) is identical to CreateTrade.
func (s *TradeService) CreateSimpleTrade(ctx context.Context, input CreateSimpleTradeInput) (*Trade, error) {
	direction := strings.ToUpper(input.Direction)

	if !constants.ValidateSymbol(input.Symbol) {
		return nil, fmt.Errorf("symbol %s not supported", input.Symbol)
	}
	if input.Timeframe == "" {
		input.Timeframe = constants.TimeframeH4
	}
	symbolConfig, err := constants.GetSymbolConfig(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("symbol %s not supported: %w", input.Symbol, err)
	}

	account, err := s.accountRepo.GetAccountByID(ctx, input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	if err := ValidateTradeGeometry(input.PlannedEntry, input.PlannedSL, input.PlannedTP, strings.ToLower(direction)); err != nil {
		return nil, fmt.Errorf("invalid geometry: %w", err)
	}

	accountMaxRisk := account.MaxRiskPerTradePct.InexactFloat64()
	if input.PlannedRiskPct > accountMaxRisk {
		return nil, fmt.Errorf("planned risk %.2f%% exceeds account max %.2f%%", input.PlannedRiskPct, accountMaxRisk)
	}
	if input.PlannedRiskPct <= 0 {
		return nil, fmt.Errorf("planned risk must be positive")
	}

	balance := account.Balance.InexactFloat64()
	riskAmount, err := ComputeRiskAmount(balance, input.PlannedRiskPct)
	if err != nil {
		return nil, fmt.Errorf("risk calculation: %w", err)
	}
	stopDistance, err := ComputeStopDistance(input.PlannedEntry, input.PlannedSL)
	if err != nil {
		return nil, fmt.Errorf("stop distance: %w", err)
	}
	stopDistancePips, err := ComputeStopDistancePips(stopDistance, symbolConfig.PipSize)
	if err != nil {
		return nil, fmt.Errorf("pip conversion: %w", err)
	}
	positionSize, err := ComputePositionSize(riskAmount, stopDistancePips, symbolConfig.PipValue)
	if err != nil {
		return nil, fmt.Errorf("position sizing: %w", err)
	}
	rr, err := ComputeRR(input.PlannedEntry, input.PlannedSL, input.PlannedTP, strings.ToLower(direction))
	if err != nil {
		return nil, fmt.Errorf("RR calculation: %w", err)
	}

	existingTrades, err := s.tradeRepo.GetTradesByAccountAndCandle(ctx, input.AccountID, input.CandleID)
	if err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	}
	for _, existing := range existingTrades {
		if string(existing.Direction) == direction {
			return nil, fmt.Errorf("duplicate trade: account %s already has %s trade on candle %s",
				input.AccountID, direction, input.CandleID)
		}
	}

	if rr < constants.MinimumRR {
		return nil, fmt.Errorf("trade rejected: RR %.2f below minimum %.2f", rr, constants.MinimumRR)
	}

	maxPositionSize, err := ComputeMaxPositionSize(balance, int(account.Leverage), symbolConfig.ContractSize)
	if err != nil {
		return nil, fmt.Errorf("leverage check: %w", err)
	}
	if positionSize > maxPositionSize {
		return nil, fmt.Errorf("position size %.2f lots exceeds max %.2f lots", positionSize, maxPositionSize)
	}

	dbTrade, err := s.tradeRepo.CreateTrade(ctx, db.CreateTradeParams{
		UserID:       account.UserID,
		AccountID:    input.AccountID,
		CandleID:     input.CandleID,
		Symbol:       input.Symbol,
		Timeframe:    input.Timeframe,
		Direction:    db.TradeDirection(direction),
		PlannedEntry: decimal.NewFromFloat(input.PlannedEntry),
		StopLoss:     decimal.NewFromFloat(input.PlannedSL),
		TakeProfit:   decimal.NewFromFloat(input.PlannedTP),
		RiskPercent:  decimal.NewFromFloat(input.PlannedRiskPct),
	})
	if err != nil {
		return nil, fmt.Errorf("persist trade: %w", err)
	}
	return tradeFromDB(dbTrade), nil
}