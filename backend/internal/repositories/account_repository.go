package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	Type               string     `json:"type"`
	BrokerName         string     `json:"broker_name"`
	Currency           string     `json:"currency"`
	Balance            string     `json:"balance"`
	Leverage           int        `json:"leverage"`
	MaxRiskPerTradePct float64    `json:"max_risk_per_trade_pct"`
	MaxDailyRiskPct    float64    `json:"max_daily_risk_pct"`
	Timezone           string     `json:"timezone"`
	PreferredSession   string     `json:"preferred_session"`
	StrategyID         *uuid.UUID `json:"strategy_id,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
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
	StrategyID         *uuid.UUID // optional — links account to a strategy
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

	var strategyID pgtype.UUID
	if params.StrategyID != nil {
		strategyID = pgtype.UUID{Bytes: *params.StrategyID, Valid: true}
	}

	// Call SQLC-generated CreateAccount with correct field names
	account, err := r.q.CreateAccount(ctx, db.CreateAccountParams{
		ID:                 params.ID,
		UserID:             params.UserID,
		Type:               db.AccountType(params.Type),
		BrokerName:         params.BrokerName,
		Currency:           params.Currency,
		Balance:            balanceDec, // Direct decimal.Decimal
		Leverage:           params.Leverage,
		MaxRiskPerTradePct: riskTradeDec, // Direct decimal.Decimal
		MaxDailyRiskPct:    riskDailyDec, // Direct decimal.Decimal
		Timezone:           params.Timezone,
		PreferredSession:   db.SessionType(params.PreferredSession),
		StrategyID:         strategyID,
	})
	if err != nil {
		return nil, err
	}

	return accountFromDB(account), nil
}

func accountFromDB(a db.Account) *Account {
	out := &Account{
		ID:                 a.ID,
		UserID:             a.UserID,
		Type:               string(a.Type),
		BrokerName:         a.BrokerName,
		Currency:           a.Currency,
		Balance:            a.Balance.String(),
		Leverage:           int(a.Leverage),
		MaxRiskPerTradePct: a.MaxRiskPerTradePct.InexactFloat64(),
		MaxDailyRiskPct:    a.MaxDailyRiskPct.InexactFloat64(),
		Timezone:           a.Timezone,
		PreferredSession:   string(a.PreferredSession),
		UpdatedAt:          a.UpdatedAt.Time,
	}
	if a.StrategyID.Valid {
		id := uuid.UUID(a.StrategyID.Bytes)
		out.StrategyID = &id
	}
	return out
}

func (r *AccountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (db.Account, error) {
	return r.q.GetAccountByID(ctx, id)
}

// GetAccountByIDForHandler retrieves account in handler format
func (r *AccountRepository) GetAccountByIDForHandler(ctx context.Context, id uuid.UUID) (*Account, error) {
	acc, err := r.q.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return accountFromDB(acc), nil
}
