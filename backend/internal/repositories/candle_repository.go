package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"set-and-trend/backend/internal/db"
)

type CandleRepository struct {
	q *db.Queries
}

func NewCandleRepository(q *db.Queries) *CandleRepository {
	return &CandleRepository{q: q}
}

type Candle struct {
	ID           uuid.UUID `json:"id"`
	TimestampUTC time.Time `json:"timestamp_utc"`
	Open         string    `json:"open"`
	High         string    `json:"high"`
	Low          string    `json:"low"`
	Close        string    `json:"close"`
	Volume       *int64    `json:"volume,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type CandleCreateParams struct {
	ID           uuid.UUID
	TimestampUTC time.Time
	Open         string
	High         string
	Low          string
	Close        string
	Volume       *int64
}

func (r *CandleRepository) CreateCandle(ctx context.Context, params CandleCreateParams) (*Candle, error) {
	// Convert strings to decimal.Decimal
	openDec, err := decimal.NewFromString(params.Open)
	if err != nil {
		return nil, err
	}
	highDec, err := decimal.NewFromString(params.High)
	if err != nil {
		return nil, err
	}
	lowDec, err := decimal.NewFromString(params.Low)
	if err != nil {
		return nil, err
	}
	closeDec, err := decimal.NewFromString(params.Close)
	if err != nil {
		return nil, err
	}

	// Convert time.Time to pgtype.Timestamptz
	var timestampPg pgtype.Timestamptz
	timestampPg.Scan(params.TimestampUTC)

	// Convert *int64 to pgtype.Int8
	var volumePg pgtype.Int8
	if params.Volume != nil {
		volumePg.Scan(*params.Volume)
	}

	candle, err := r.q.CreateCandle(ctx, db.CreateCandleParams{
		ID:           params.ID,
		TimestampUtc: timestampPg,
		Open:         openDec,
		High:         highDec,
		Low:          lowDec,
		Close:        closeDec,
		Volume:       volumePg,
	})
	if err != nil {
		return nil, err
	}

	// Convert pgtype.Int8 back to *int64
	var volume *int64
	if candle.Volume.Valid {
		v := candle.Volume.Int64
		volume = &v
	}

	return &Candle{
		ID:           candle.ID,
		TimestampUTC: candle.TimestampUtc.Time,
		Open:         candle.Open.String(),
		High:         candle.High.String(),
		Low:          candle.Low.String(),
		Close:        candle.Close.String(),
		Volume:       volume,
		CreatedAt:    candle.CreatedAt.Time,
	}, nil
}

func (r *CandleRepository) GetLatestCandles(ctx context.Context, limit int) ([]Candle, error) {
	dbCandles, err := r.q.GetLatestCandles(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	candles := make([]Candle, len(dbCandles))
	for i, c := range dbCandles {
		var volume *int64
		if c.Volume.Valid {
			v := c.Volume.Int64
			volume = &v
		}

		candles[i] = Candle{
			ID:           c.ID,
			TimestampUTC: c.TimestampUtc.Time,
			Open:         c.Open.String(),
			High:         c.High.String(),
			Low:          c.Low.String(),
			Close:        c.Close.String(),
			Volume:       volume,
			CreatedAt:    c.CreatedAt.Time,
		}
	}
	return candles, nil
}
