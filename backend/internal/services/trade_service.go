package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/repositories"
)

type TradeService struct {
	tradeRepo   *repositories.TradeRepository
	accountRepo *repositories.AccountRepository
	candleRepo  *repositories.CandleRepository
}

func NewTradeService(
	tradeRepo *repositories.TradeRepository,
	accountRepo *repositories.AccountRepository,
	candleRepo *repositories.CandleRepository,
) *TradeService {
	return &TradeService{
		tradeRepo:   tradeRepo,
		accountRepo: accountRepo,
		candleRepo:  candleRepo,
	}
}

// CreateTradeInput represents user intent
type CreateTradeInput struct {
	AccountID      uuid.UUID
	CandleID       uuid.UUID
	Bias           string
	PlannedEntry   float64
	PlannedSL      float64
	PlannedTP      float64
	PlannedRiskPct float64
	ReasonForTrade string
}

// CreateTrade orchestrates trade creation with full validation
func (s *TradeService) CreateTrade(ctx context.Context, input CreateTradeInput) (*repositories.Trade, error) {
	// 1. Load account
	account, err := s.accountRepo.GetAccountByID(ctx, input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// 2. Load candle
	candle, err := s.candleRepo.GetCandleByID(ctx, input.CandleID)
	if err != nil {
		return nil, fmt.Errorf("candle not found: %w", err)
	}

	// 3. Validate trade geometry
	err = ValidateTradeGeometry(input.PlannedEntry, input.PlannedSL, input.PlannedTP, input.Bias)
	if err != nil {
		return nil, fmt.Errorf("invalid geometry: %w", err)
	}

	// 4. Validate risk percentage
	accountMaxRisk := account.MaxRiskPerTradePct
	if input.PlannedRiskPct > accountMaxRisk {
		return nil, fmt.Errorf("planned risk %.2f%% exceeds account max %.2f%%", input.PlannedRiskPct, accountMaxRisk)
	}
	if input.PlannedRiskPct <= 0 {
		return nil, fmt.Errorf("planned risk must be positive")
	}

	// 5. Compute risk math
	balance, _ := strconv.ParseFloat(account.Balance, 64)
	
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

	rr, err := ComputeRR(input.PlannedEntry, input.PlannedSL, input.PlannedTP, input.Bias)
	if err != nil {
		return nil, fmt.Errorf("RR calculation: %w", err)
	}

	// 6. Create trade with immutable snapshots
	trade, err := s.tradeRepo.CreateTrade(ctx, repositories.TradeCreateParams{
		ID:                        uuid.New(),
		UserID:                    account.UserID,
		AccountID:                 input.AccountID,
		CandleID:                  input.CandleID,
		Symbol:                    constants.SymbolEURUSD,
		Timeframe:                 constants.TimeframeW1,
		SetupTimestampUTC:         time.Now().UTC(),
		AccountBalanceAtSetup:     account.Balance,
		LeverageAtSetup:           int32(account.Leverage),
		MaxRiskPerTradePctAtSetup: fmt.Sprintf("%.2f", account.MaxRiskPerTradePct),
		TimezoneAtSetup:           account.Timezone,
		Bias:                      input.Bias,
		PlannedEntry:              fmt.Sprintf("%.5f", input.PlannedEntry),
		PlannedSL:                 fmt.Sprintf("%.5f", input.PlannedSL),
		PlannedTP:                 fmt.Sprintf("%.5f", input.PlannedTP),
		PlannedRR:                 fmt.Sprintf("%.2f", rr),
		PlannedRiskPct:            fmt.Sprintf("%.2f", input.PlannedRiskPct),
		PlannedRiskAmount:         fmt.Sprintf("%.2f", riskAmount),
		PlannedPositionSize:       fmt.Sprintf("%.2f", positionSize),
		ReasonForTrade:            input.ReasonForTrade,
	})
	if err != nil {
		return nil, fmt.Errorf("persist trade: %w", err)
	}

	return trade, nil
}
