package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
)

type TradeRepository struct {
	q *db.Queries
}

func NewTradeRepository(q *db.Queries) *TradeRepository {
	return &TradeRepository{q: q}
}

// Trade represents a trade (for API responses)
type Trade struct {
	ID                        uuid.UUID  `json:"id"`
	UserID                    uuid.UUID  `json:"user_id"`
	AccountID                 uuid.UUID  `json:"account_id"`
	CandleID                  uuid.UUID  `json:"candle_id"`
	Symbol                    string     `json:"symbol"`
	Timeframe                 string     `json:"timeframe"`
	SetupTimestampUTC         time.Time  `json:"setup_timestamp_utc"`
	AccountBalanceAtSetup     string     `json:"account_balance_at_setup"`
	LeverageAtSetup           int32      `json:"leverage_at_setup"`
	MaxRiskPerTradePctAtSetup string     `json:"max_risk_per_trade_pct_at_setup"`
	TimezoneAtSetup           string     `json:"timezone_at_setup"`
	Bias                      string     `json:"bias"`
	PlannedEntry              string     `json:"planned_entry"`
	PlannedSL                 string     `json:"planned_sl"`
	PlannedTP                 string     `json:"planned_tp"`
	PlannedRR                 string     `json:"planned_rr"`
	PlannedRiskPct            string     `json:"planned_risk_pct"`
	PlannedRiskAmount         string     `json:"planned_risk_amount"`
	PlannedPositionSize       string     `json:"planned_position_size"`
	ReasonForTrade            string     `json:"reason_for_trade"`
	CreatedAt                 time.Time  `json:"created_at"`
}

// TradeCreateParams contains parameters for creating a trade
type TradeCreateParams struct {
	ID                        uuid.UUID
	UserID                    uuid.UUID
	AccountID                 uuid.UUID
	CandleID                  uuid.UUID
	Symbol                    string
	Timeframe                 string
	SetupTimestampUTC         time.Time
	AccountBalanceAtSetup     string
	LeverageAtSetup           int32
	MaxRiskPerTradePctAtSetup string
	TimezoneAtSetup           string
	Bias                      string
	PlannedEntry              string
	PlannedSL                 string
	PlannedTP                 string
	PlannedRR                 string
	PlannedRiskPct            string
	PlannedRiskAmount         string
	PlannedPositionSize       string
	ReasonForTrade            string
}

// CreateTrade inserts a new planned trade
func (r *TradeRepository) CreateTrade(ctx context.Context, params TradeCreateParams) (*Trade, error) {
	// Convert string values to decimal
	balanceDec, _ := decimal.NewFromString(params.AccountBalanceAtSetup)
	riskPctDec, _ := decimal.NewFromString(params.MaxRiskPerTradePctAtSetup)
	entryDec, _ := decimal.NewFromString(params.PlannedEntry)
	slDec, _ := decimal.NewFromString(params.PlannedSL)
	tpDec, _ := decimal.NewFromString(params.PlannedTP)
	rrDec, _ := decimal.NewFromString(params.PlannedRR)
	plannedRiskPctDec, _ := decimal.NewFromString(params.PlannedRiskPct)
	plannedRiskAmtDec, _ := decimal.NewFromString(params.PlannedRiskAmount)
	plannedPosSizeDec, _ := decimal.NewFromString(params.PlannedPositionSize)

	// Convert timestamp
	var timestampPg pgtype.Timestamptz
	timestampPg.Scan(params.SetupTimestampUTC)

	trade, err := r.q.CreateTrade(ctx, db.CreateTradeParams{
		ID:                        params.ID,
		UserID:                    params.UserID,
		AccountID:                 params.AccountID,
		CandleID:                  params.CandleID,
		Symbol:                    params.Symbol,
		Timeframe:                 params.Timeframe,
		SetupTimestampUtc:         timestampPg,
		AccountBalanceAtSetup:     balanceDec,
		LeverageAtSetup:           params.LeverageAtSetup,
		MaxRiskPerTradePctAtSetup: riskPctDec,
		TimezoneAtSetup:           params.TimezoneAtSetup,
		Bias:                      params.Bias,
		PlannedEntry:              entryDec,
		PlannedSl:                 slDec,
		PlannedTp:                 tpDec,
		PlannedRr:                 rrDec,
		PlannedRiskPct:            plannedRiskPctDec,
		PlannedRiskAmount:         plannedRiskAmtDec,
		PlannedPositionSize:       plannedPosSizeDec,
		ReasonForTrade:            params.ReasonForTrade,
	})
	if err != nil {
		return nil, err
	}

	return &Trade{
		ID:                        trade.ID,
		UserID:                    trade.UserID,
		AccountID:                 trade.AccountID,
		CandleID:                  trade.CandleID,
		Symbol:                    trade.Symbol,
		Timeframe:                 trade.Timeframe,
		SetupTimestampUTC:         trade.SetupTimestampUtc.Time,
		AccountBalanceAtSetup:     trade.AccountBalanceAtSetup.String(),
		LeverageAtSetup:           trade.LeverageAtSetup,
		MaxRiskPerTradePctAtSetup: trade.MaxRiskPerTradePctAtSetup.String(),
		TimezoneAtSetup:           trade.TimezoneAtSetup,
		Bias:                      trade.Bias,
		PlannedEntry:              trade.PlannedEntry.String(),
		PlannedSL:                 trade.PlannedSl.String(),
		PlannedTP:                 trade.PlannedTp.String(),
		PlannedRR:                 trade.PlannedRr.String(),
		PlannedRiskPct:            trade.PlannedRiskPct.String(),
		PlannedRiskAmount:         trade.PlannedRiskAmount.String(),
		PlannedPositionSize:       trade.PlannedPositionSize.String(),
		ReasonForTrade:            trade.ReasonForTrade,
		CreatedAt:                 trade.CreatedAt.Time,
	}, nil
}

// GetTradeByID retrieves a trade by ID
func (r *TradeRepository) GetTradeByID(ctx context.Context, id uuid.UUID) (*Trade, error) {
	trade, err := r.q.GetTradeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Trade{
		ID:                        trade.ID,
		UserID:                    trade.UserID,
		AccountID:                 trade.AccountID,
		CandleID:                  trade.CandleID,
		Symbol:                    trade.Symbol,
		Timeframe:                 trade.Timeframe,
		SetupTimestampUTC:         trade.SetupTimestampUtc.Time,
		AccountBalanceAtSetup:     trade.AccountBalanceAtSetup.String(),
		LeverageAtSetup:           trade.LeverageAtSetup,
		MaxRiskPerTradePctAtSetup: trade.MaxRiskPerTradePctAtSetup.String(),
		TimezoneAtSetup:           trade.TimezoneAtSetup,
		Bias:                      trade.Bias,
		PlannedEntry:              trade.PlannedEntry.String(),
		PlannedSL:                 trade.PlannedSl.String(),
		PlannedTP:                 trade.PlannedTp.String(),
		PlannedRR:                 trade.PlannedRr.String(),
		PlannedRiskPct:            trade.PlannedRiskPct.String(),
		PlannedRiskAmount:         trade.PlannedRiskAmount.String(),
		PlannedPositionSize:       trade.PlannedPositionSize.String(),
		ReasonForTrade:            trade.ReasonForTrade,
		CreatedAt:                 trade.CreatedAt.Time,
	}, nil
}
