package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Trade represents a trade in the system
// This type is used for backward compatibility with service tests
type Trade struct {
	ID              uuid.UUID       `json:"id"`
	AccountID       uuid.UUID       `json:"account_id"`
	CandleID        uuid.UUID       `json:"candle_id"`
	Bias            string          `json:"bias"`
	Status          string          `json:"status"`
	PlannedEntry    decimal.Decimal `json:"planned_entry"`
	PlannedSL       decimal.Decimal `json:"planned_sl"`
	PlannedTP       decimal.Decimal `json:"planned_tp"`
	PlannedRiskPct  decimal.Decimal `json:"planned_risk_pct"`
	ActualEntry     decimal.Decimal `json:"actual_entry,omitempty"`
	ActualSL        decimal.Decimal `json:"actual_sl,omitempty"`
	ActualTP        decimal.Decimal `json:"actual_tp,omitempty"`
	ExitPrice       decimal.Decimal `json:"exit_price,omitempty"`
	ReasonForTrade  string          `json:"reason_for_trade"`
	LotSize         decimal.Decimal `json:"lot_size,omitempty"`
	ResultPips      decimal.Decimal `json:"result_pips,omitempty"`
	ResultR         decimal.Decimal `json:"result_r,omitempty"`
	ResultMoney     decimal.Decimal `json:"result_money,omitempty"`
	ScreenshotBefore string         `json:"screenshot_before,omitempty"`
	ScreenshotAfter  string         `json:"screenshot_after,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TradeCreateParams are the parameters needed to create a trade
type TradeCreateParams struct {
	ID             uuid.UUID
	AccountID      uuid.UUID
	CandleID       uuid.UUID
	Bias           string
	Status         string
	PlannedEntry   decimal.Decimal
	PlannedSL      decimal.Decimal
	PlannedTP      decimal.Decimal
	PlannedRiskPct decimal.Decimal
	ReasonForTrade string
}
