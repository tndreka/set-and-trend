package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
)

type AccountRepository struct {
	q *db.Queries
}

func NewAccountRepository(q *db.Queries) *AccountRepository {
	return &AccountRepository{q: q}
}

type Account struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	Type               string    `json:"type"`
	BrokerName         string    `json:"broker_name"`
	Currency           string    `json:"currency"`
	Balance            string    `json:"balance"`
	Leverage           int       `json:"leverage"`
	MaxRiskPerTradePct float64   `json:"max_risk_per_trade_pct"`
	MaxDailyRiskPct    float64   `json:"max_daily_risk_pct"`
	Timezone           string    `json:"timezone"`
	PreferredSession   string    `json:"preferred_session"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AccountCreateParams struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Type               string
	BrokerName         string
	Currency           string
	Balance            string // comes as string from handler
	Leverage           int32
	MaxRiskPerTradePct float64
	MaxDailyRiskPct    float64
	Timezone           string
	PreferredSession   string
}

func (r *AccountRepository) CreateAccount(ctx context.Context, params AccountCreateParams) (*Account, error) {
	// Convert string balance to decimal.Decimal
	balanceDec, err := decimal.NewFromString(params.Balance)
	if err != nil {
		return nil, err
	}

	// Convert float64 to decimal.Decimal
	riskTradeDec := decimal.NewFromFloat(params.MaxRiskPerTradePct)
	riskDailyDec := decimal.NewFromFloat(params.MaxDailyRiskPct)

	// Call SQLC-generated CreateAccount with correct field names
	account, err := r.q.CreateAccount(ctx, db.CreateAccountParams{
		ID:                 params.ID,
		UserID:             params.UserID,
		Type:               params.Type, // ✅ Fixed from Column3
		BrokerName:         params.BrokerName,
		Currency:           params.Currency,
		Balance:            balanceDec, // ✅ Direct decimal.Decimal
		Leverage:           params.Leverage,
		MaxRiskPerTradePct: riskTradeDec, // ✅ Direct decimal.Decimal
		MaxDailyRiskPct:    riskDailyDec, // ✅ Direct decimal.Decimal
		Timezone:           params.Timezone,
		PreferredSession:   params.PreferredSession, // ✅ Fixed from Column11
	})
	if err != nil {
		return nil, err
	}

	// Convert back to response format
	return &Account{
		ID:                 account.ID,
		UserID:             account.UserID,
		Type:               account.Type,
		BrokerName:         account.BrokerName,
		Currency:           account.Currency,
		Balance:            account.Balance.String(),
		Leverage:           int(account.Leverage),
		MaxRiskPerTradePct: account.MaxRiskPerTradePct.InexactFloat64(),
		MaxDailyRiskPct:    account.MaxDailyRiskPct.InexactFloat64(),
		Timezone:           account.Timezone,
		PreferredSession:   account.PreferredSession,
		UpdatedAt:          account.UpdatedAt.Time,
	}, nil
}

// GetAccountByID retrieves an account by ID
func (r *AccountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	acc, err := r.q.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Account{
		ID:                 acc.ID,
		UserID:             acc.UserID,
		Type:               acc.Type,
		BrokerName:         acc.BrokerName,
		Currency:           acc.Currency,
		Balance:            acc.Balance.String(),
		Leverage:           int(acc.Leverage),
		MaxRiskPerTradePct: acc.MaxRiskPerTradePct.InexactFloat64(),
		MaxDailyRiskPct:    acc.MaxDailyRiskPct.InexactFloat64(),
		Timezone:           acc.Timezone,
		PreferredSession:   acc.PreferredSession,
		UpdatedAt:          acc.UpdatedAt.Time,
	}, nil
}
