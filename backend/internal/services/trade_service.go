package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	//"github.com/google/uuid"
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

	stopDistancePips, err := ComputeStopDistancePips(stopDistance, constants.PipValueEURUSD)
	if err != nil {
		return nil, fmt.Errorf("pip conversion: %w", err)
	}

	// Position sizing (assuming $10 per pip for standard lot)
	const pipValuePerLot = 10.0
	positionSize, err := ComputePositionSize(riskAmount, stopDistancePips, pipValuePerLot)
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

	// 5.7. Validate position size against leverage
	maxPositionSize, err := ComputeMaxPositionSize(balance, int(account.Leverage), constants.ContractSizeEURUSD)
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
		Symbol:       constants.SymbolEURUSD,
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
	return &Trade{
		ID:           dbTrade.ID,
		UserID:       dbTrade.UserID,
		AccountID:    dbTrade.AccountID,
		CandleID:     dbTrade.CandleID,
		Symbol:       dbTrade.Symbol,
		Timeframe:    dbTrade.Timeframe,
		Direction:    string(dbTrade.Direction),
		PlannedEntry: dbTrade.PlannedEntry.String(),
		StopLoss:     dbTrade.StopLoss.String(),
		TakeProfit:   dbTrade.TakeProfit.String(),
		RiskPercent:  dbTrade.RiskPercent.String(),
		CreatedAt:    dbTrade.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}
