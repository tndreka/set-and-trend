package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type IndicatorRepository struct {
	q *db.Queries
}

func NewIndicatorRepository(q *db.Queries) *IndicatorRepository {
	return &IndicatorRepository{q: q}
}

type Indicator struct {
	ID                 uuid.UUID `json:"id"`
	CandleID           uuid.UUID `json:"candle_id"`
	EMA20              string    `json:"ema20"`
	EMA50              string    `json:"ema50"`
	EMA200             string    `json:"ema200"`
	RangeSize          string    `json:"range_size"`
	BodySize           string    `json:"body_size"`
	UpperWick          string    `json:"upper_wick"`
	LowerWick          string    `json:"lower_wick"`
	MidPrice           string    `json:"mid_price"`
	LastSwingHighPrice *string   `json:"last_swing_high_price,omitempty"`
	LastSwingLowPrice  *string   `json:"last_swing_low_price,omitempty"`
	ComputedAt         time.Time `json:"computed_at"`
}

type IndicatorCreateParams struct {
	ID                 uuid.UUID
	CandleID           uuid.UUID
	EMA20              float64
	EMA50              float64
	EMA200             float64
	RangeSize          float64
	BodySize           float64
	UpperWick          float64
	LowerWick          float64
	MidPrice           float64
	LastSwingHighPrice *float64
	LastSwingLowPrice  *float64
}

func (r *IndicatorRepository) CreateIndicator(ctx context.Context, params IndicatorCreateParams) (*Indicator, error) {
	// Convert float64 to decimal.Decimal
	ema20Dec := decimal.NewFromFloat(params.EMA20)
	ema50Dec := decimal.NewFromFloat(params.EMA50)
	ema200Dec := decimal.NewFromFloat(params.EMA200)
	rangeDec := decimal.NewFromFloat(params.RangeSize)
	bodyDec := decimal.NewFromFloat(params.BodySize)
	upperDec := decimal.NewFromFloat(params.UpperWick)
	lowerDec := decimal.NewFromFloat(params.LowerWick)
	midDec := decimal.NewFromFloat(params.MidPrice)

	var swingHighDec, swingLowDec decimal.Decimal
	if params.LastSwingHighPrice != nil {
		swingHighDec = decimal.NewFromFloat(*params.LastSwingHighPrice)
	}
	if params.LastSwingLowPrice != nil {
		swingLowDec = decimal.NewFromFloat(*params.LastSwingLowPrice)
	}

	indicator, err := r.q.CreateIndicator(ctx, db.CreateIndicatorParams{
		ID:                 params.ID,
		CandleID:           params.CandleID,
		Ema20:              ema20Dec,
		Ema50:              ema50Dec,
		Ema200:             ema200Dec,
		RangeSize:          rangeDec,
		BodySize:           bodyDec,
		UpperWick:          upperDec,
		LowerWick:          lowerDec,
		MidPrice:           midDec,
		LastSwingHighPrice: swingHighDec,
		LastSwingLowPrice:  swingLowDec,
	})
	if err != nil {
		return nil, err
	}

	// Convert back to strings
	var swingHighStr, swingLowStr *string
	if params.LastSwingHighPrice != nil {
		s := indicator.LastSwingHighPrice.String()
		swingHighStr = &s
	}
	if params.LastSwingLowPrice != nil {
		s := indicator.LastSwingLowPrice.String()
		swingLowStr = &s
	}

	return &Indicator{
		ID:                 indicator.ID,
		CandleID:           indicator.CandleID,
		EMA20:              indicator.Ema20.String(),
		EMA50:              indicator.Ema50.String(),
		EMA200:             indicator.Ema200.String(),
		RangeSize:          indicator.RangeSize.String(),
		BodySize:           indicator.BodySize.String(),
		UpperWick:          indicator.UpperWick.String(),
		LowerWick:          indicator.LowerWick.String(),
		MidPrice:           indicator.MidPrice.String(),
		LastSwingHighPrice: swingHighStr,
		LastSwingLowPrice:  swingLowStr,
		ComputedAt:         indicator.ComputedAt.Time,
	}, nil
}

func (r *IndicatorRepository) GetIndicatorByCandleID(ctx context.Context, candleID uuid.UUID) (*Indicator, error) {
	indicator, err := r.q.GetIndicatorByCandleID(ctx, candleID)
	if err != nil {
		return nil, err
	}

	var swingHighStr, swingLowStr *string
	if indicator.LastSwingHighPrice.String() != "0" {
		s := indicator.LastSwingHighPrice.String()
		swingHighStr = &s
	}
	if indicator.LastSwingLowPrice.String() != "0" {
		s := indicator.LastSwingLowPrice.String()
		swingLowStr = &s
	}

	return &Indicator{
		ID:                 indicator.ID,
		CandleID:           indicator.CandleID,
		EMA20:              indicator.Ema20.String(),
		EMA50:              indicator.Ema50.String(),
		EMA200:             indicator.Ema200.String(),
		RangeSize:          indicator.RangeSize.String(),
		BodySize:           indicator.BodySize.String(),
		UpperWick:          indicator.UpperWick.String(),
		LowerWick:          indicator.LowerWick.String(),
		MidPrice:           indicator.MidPrice.String(),
		LastSwingHighPrice: swingHighStr,
		LastSwingLowPrice:  swingLowStr,
		ComputedAt:         indicator.ComputedAt.Time,
	}, nil
}

func (r *IndicatorRepository) GetPreviousIndicatorByTimestamp(
	ctx context.Context,
	timestamp time.Time,
) (*Indicator, error) {
	// Convert timestamp to pgtype.Timestamptz
	var timestampPg pgtype.Timestamptz
	timestampPg.Scan(timestamp)

	indicator, err := r.q.GetPreviousIndicatorByTimestamp(ctx, timestampPg)
	if err != nil {
		return nil, err // No previous indicator (first candle)
	}

	var swingHighStr, swingLowStr *string
	if indicator.LastSwingHighPrice.String() != "0" {
		s := indicator.LastSwingHighPrice.String()
		swingHighStr = &s
	}
	if indicator.LastSwingLowPrice.String() != "0" {
		s := indicator.LastSwingLowPrice.String()
		swingLowStr = &s
	}

	return &Indicator{
		ID:                 indicator.ID,
		CandleID:           indicator.CandleID,
		EMA20:              indicator.Ema20.String(),
		EMA50:              indicator.Ema50.String(),
		EMA200:             indicator.Ema200.String(),
		RangeSize:          indicator.RangeSize.String(),
		BodySize:           indicator.BodySize.String(),
		UpperWick:          indicator.UpperWick.String(),
		LowerWick:          indicator.LowerWick.String(),
		MidPrice:           indicator.MidPrice.String(),
		LastSwingHighPrice: swingHighStr,
		LastSwingLowPrice:  swingLowStr,
		ComputedAt:         indicator.ComputedAt.Time,
	}, nil
}
