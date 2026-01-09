package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"set-and-trend/backend/internal/repositories"
)

// Mock repositories for testing
type mockAccountRepo struct {
	account *repositories.Account
	err     error
}

func (m *mockAccountRepo) GetAccountByID(ctx context.Context, id uuid.UUID) (*repositories.Account, error) {
	return m.account, m.err
}

func (m *mockAccountRepo) CreateAccount(ctx context.Context, params repositories.AccountCreateParams) (*repositories.Account, error) {
	return nil, nil
}

type mockCandleRepo struct {
	candle *repositories.Candle
	err    error
}

func (m *mockCandleRepo) GetCandleByID(ctx context.Context, id uuid.UUID) (*repositories.Candle, error) {
	return m.candle, m.err
}

func (m *mockCandleRepo) CreateCandle(ctx context.Context, params repositories.CandleCreateParams) (*repositories.Candle, error) {
	return nil, nil
}

func (m *mockCandleRepo) GetLatestCandles(ctx context.Context, limit int) ([]repositories.Candle, error) {
	return nil, nil
}

type mockTradeRepo struct {
	trade  *repositories.Trade
	trades []*repositories.Trade
	err    error
}

func (m *mockTradeRepo) CreateTrade(ctx context.Context, params repositories.TradeCreateParams) (*repositories.Trade, error) {
	return m.trade, m.err
}

func (m *mockTradeRepo) GetTradeByID(ctx context.Context, id uuid.UUID) (*repositories.Trade, error) {
	return m.trade, m.err
}

func (m *mockTradeRepo) GetTradesByAccountAndCandle(ctx context.Context, accountID, candleID uuid.UUID) ([]*repositories.Trade, error) {
	return m.trades, m.err
}

func TestCreateTrade_ValidLongTrade(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: &repositories.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            "10000.00",
			Leverage:           100,
			MaxRiskPerTradePct: 2.0,
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{
			ID: uuid.New(),
		},
	}

	tradeRepo := &mockTradeRepo{
		trades: []*repositories.Trade{}, // No duplicates
		trade: &repositories.Trade{
			ID: uuid.New(),
		},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Bias:           "long",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200, // 1:3 RR
		PlannedRiskPct: 1.0,
		ReasonForTrade: "Weekly bullish trend confirmed",
	}

	trade, err := service.CreateTrade(ctx, input)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if trade == nil {
		t.Fatal("Expected trade to be created")
	}
}

func TestCreateTrade_RejectLowRR(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: &repositories.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            "10000.00",
			Leverage:           100,
			MaxRiskPerTradePct: 2.0,
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []*repositories.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Bias:           "long",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1070, // Only 1:0.4 RR - should fail
		PlannedRiskPct: 1.0,
		ReasonForTrade: "Bad risk/reward",
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for low RR, got success")
	}
}

func TestCreateTrade_RejectDuplicate(t *testing.T) {
	ctx := context.Background()

	accountID := uuid.New()
	candleID := uuid.New()

	accountRepo := &mockAccountRepo{
		account: &repositories.Account{
			ID:                 accountID,
			UserID:             uuid.New(),
			Balance:            "10000.00",
			Leverage:           100,
			MaxRiskPerTradePct: 2.0,
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: candleID},
	}

	// Simulate existing trade
	tradeRepo := &mockTradeRepo{
		trades: []*repositories.Trade{
			{
				ID:        uuid.New(),
				AccountID: accountID,
				CandleID:  candleID,
				Bias:      "long",
			},
		},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountID,
		CandleID:       candleID,
		Bias:           "long", // Same bias - should be rejected
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200,
		PlannedRiskPct: 1.0,
		ReasonForTrade: "Duplicate attempt",
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for duplicate trade, got success")
	}
}

func TestCreateTrade_RejectExcessiveRisk(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: &repositories.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            "10000.00",
			Leverage:           100,
			MaxRiskPerTradePct: 2.0, // Max 2%
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []*repositories.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Bias:           "long",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200,
		PlannedRiskPct: 3.0, // Exceeds account max of 2%
		ReasonForTrade: "Excessive risk",
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for excessive risk, got success")
	}
}

func TestCreateTrade_RejectInvalidGeometry(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: &repositories.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            "10000.00",
			Leverage:           100,
			MaxRiskPerTradePct: 2.0,
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []*repositories.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Bias:           "long",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1100, // SL above entry for long - invalid!
		PlannedTP:      1.1200,
		PlannedRiskPct: 1.0,
		ReasonForTrade: "Invalid geometry",
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for invalid geometry, got success")
	}
}
