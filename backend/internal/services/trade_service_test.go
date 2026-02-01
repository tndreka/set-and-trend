package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/repositories"
)

// Mock repositories for testing
type mockAccountRepo struct {
	account db.Account
	err     error
}

func (m *mockAccountRepo) GetAccountByID(ctx context.Context, id uuid.UUID) (db.Account, error) {
	return m.account, m.err
}

type mockCandleRepo struct {
	candle *repositories.Candle
	err    error
}

func (m *mockCandleRepo) GetCandleByID(ctx context.Context, id uuid.UUID) (*repositories.Candle, error) {
	return m.candle, m.err
}

type mockTradeRepo struct {
	trade  db.Trade
	trades []db.Trade
	err    error
}

func (m *mockTradeRepo) CreateTrade(ctx context.Context, params db.CreateTradeParams) (db.Trade, error) {
	return m.trade, m.err
}

func (m *mockTradeRepo) GetTradeByID(ctx context.Context, id uuid.UUID) (db.Trade, error) {
	return m.trade, m.err
}

func (m *mockTradeRepo) GetTradesByAccountAndCandle(ctx context.Context, accountID, candleID uuid.UUID) ([]db.Trade, error) {
	return m.trades, m.err
}

func TestCreateTrade_ValidLongTrade(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: db.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            decimal.NewFromFloat(10000.00),
			Leverage:           100,
			MaxRiskPerTradePct: decimal.NewFromFloat(2.0),
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{
			ID: uuid.New(),
		},
	}

	tradeRepo := &mockTradeRepo{
		trades: []db.Trade{}, // No duplicates
		trade: db.Trade{
			ID: uuid.New(),
		},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Symbol:         "EURUSD",
		Direction:      "LONG",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200, // 1:3 RR
		PlannedRiskPct: 1.0,
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
		account: db.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            decimal.NewFromFloat(10000.00),
			Leverage:           100,
			MaxRiskPerTradePct: decimal.NewFromFloat(2.0),
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []db.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Direction:      "LONG",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1070, // Only 1:0.4 RR - should fail
		PlannedRiskPct: 1.0,
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
		account: db.Account{
			ID:                 accountID,
			UserID:             uuid.New(),
			Balance:            decimal.NewFromFloat(10000.00),
			Leverage:           100,
			MaxRiskPerTradePct: decimal.NewFromFloat(2.0),
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: candleID},
	}

	// Simulate existing trade
	tradeRepo := &mockTradeRepo{
		trades: []db.Trade{
			{
				ID:        uuid.New(),
				AccountID: accountID,
				CandleID:  candleID,
				Direction: "LONG",
			},
		},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountID,
		CandleID:       candleID,
		Direction:      "LONG", // Same direction - should be rejected
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200,
		PlannedRiskPct: 1.0,
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for duplicate trade, got success")
	}
}

func TestCreateTrade_RejectExcessiveRisk(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: db.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            decimal.NewFromFloat(10000.00),
			Leverage:           100,
			MaxRiskPerTradePct: decimal.NewFromFloat(2.0), // Max 2%
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []db.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Direction:      "LONG",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1000,
		PlannedTP:      1.1200,
		PlannedRiskPct: 3.0, // Exceeds account max of 2%
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for excessive risk, got success")
	}
}

func TestCreateTrade_RejectInvalidGeometry(t *testing.T) {
	ctx := context.Background()

	accountRepo := &mockAccountRepo{
		account: db.Account{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Balance:            decimal.NewFromFloat(10000.00),
			Leverage:           100,
			MaxRiskPerTradePct: decimal.NewFromFloat(2.0),
			Timezone:           "UTC",
		},
	}

	candleRepo := &mockCandleRepo{
		candle: &repositories.Candle{ID: uuid.New()},
	}

	tradeRepo := &mockTradeRepo{
		trades: []db.Trade{},
	}

	service := NewTradeService(tradeRepo, accountRepo, candleRepo)

	input := CreateTradeInput{
		AccountID:      accountRepo.account.ID,
		CandleID:       candleRepo.candle.ID,
		Direction:      "LONG",
		PlannedEntry:   1.1050,
		PlannedSL:      1.1100, // SL above entry for long - invalid!
		PlannedTP:      1.1200,
		PlannedRiskPct: 1.0,
	}

	_, err := service.CreateTrade(ctx, input)
	if err == nil {
		t.Fatal("Expected error for invalid geometry, got success")
	}
}
