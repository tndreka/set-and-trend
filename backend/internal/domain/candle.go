package domain

import "time"

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
