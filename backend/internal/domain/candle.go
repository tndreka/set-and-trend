package domain

import "time"
import "github.com/google/uuid"

// Candle represents a single OHLCV market candle.
// Pure domain object. No indicators. No DB tags.
type Candle struct {
	Timestamp time.Time

	Open  float64
	High  float64
	Low   float64
	Close float64

	Volume *int64
}

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
