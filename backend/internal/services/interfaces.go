package services

import (
	"context"
	"github.com/google/uuid"
	"set-and-trend/backend/internal/repositories"
)

// AccountRepo defines the interface for account operations
type AccountRepo interface {
	GetAccountByID(ctx context.Context, id uuid.UUID) (*repositories.Account, error)
	CreateAccount(ctx context.Context, params repositories.AccountCreateParams) (*repositories.Account, error)
}

// CandleRepo defines the interface for candle operations
type CandleRepo interface {
	GetCandleByID(ctx context.Context, id uuid.UUID) (*repositories.Candle, error)
	CreateCandle(ctx context.Context, params repositories.CandleCreateParams) (*repositories.Candle, error)
	GetLatestCandles(ctx context.Context, limit int) ([]repositories.Candle, error)
}

// TradeRepo defines the interface for trade operations
type TradeRepo interface {
	CreateTrade(ctx context.Context, params repositories.TradeCreateParams) (*repositories.Trade, error)
	GetTradeByID(ctx context.Context, id uuid.UUID) (*repositories.Trade, error)
	GetTradesByAccountAndCandle(ctx context.Context, accountID, candleID uuid.UUID) ([]*repositories.Trade, error)
}
