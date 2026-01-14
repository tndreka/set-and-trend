package services

import (
	"context"
	"github.com/google/uuid"
	"set-and-trend/backend/internal/db"
)

// ============================================================================
// SERVICE INPUT TYPES (What handlers send to services)
// ============================================================================

type CreateTradeInput struct {
	AccountID   uuid.UUID
	CandleID    uuid.UUID
	Direction   string  // "LONG" or "SHORT" (handler normalizes to uppercase)
	PlannedEntry   float64
	PlannedSL      float64
	PlannedTP      float64
	PlannedRiskPct float64
}

// ============================================================================
// SERVICE OUTPUT TYPES (What services return to handlers)
// ============================================================================

type Trade struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	AccountID    uuid.UUID `json:"account_id"`
	CandleID     uuid.UUID `json:"candle_id"`
	Symbol       string    `json:"symbol"`
	Timeframe    string    `json:"timeframe"`
	Direction    string    `json:"direction"`
	PlannedEntry string    `json:"planned_entry"`
	StopLoss     string    `json:"stop_loss"`
	TakeProfit   string    `json:"take_profit"`
	RiskPercent  string    `json:"risk_percent"`
	CreatedAt    string    `json:"created_at"`
}

// ============================================================================
// REPOSITORY INTERFACES (What services expect from repositories)
// ============================================================================

type TradeRepo interface {
	CreateTrade(ctx context.Context, params db.CreateTradeParams) (db.Trade, error)
	GetTradeByID(ctx context.Context, id uuid.UUID) (db.Trade, error)
	GetTradesByAccountAndCandle(ctx context.Context, accountID, candleID uuid.UUID) ([]db.Trade, error)
}

type AccountRepo interface {
	GetAccountByID(ctx context.Context, id uuid.UUID) (db.Account, error)
}

type CandleRepo interface {
	GetCandleByID(ctx context.Context, id uuid.UUID) (*CandlesWeekly, error)
}

// CandlesWeekly is what candle repo returns (not db.CandlesWeekly)
type CandlesWeekly struct {
	ID           uuid.UUID
	TimestampUTC string
	Open         string
	High         string
	Low          string
	Close        string
	Volume       *int64
	CreatedAt    string
}
