package services

import (
	"math"
)

type WeeklyIndicators struct {
	EMA20  float64
	EMA50  float64
	EMA200 float64

	RangeSize float64
	BodySize  float64
	UpperWick float64
	LowerWick float64
	MidPrice  float64

	LastSwingHighPrice *float64
	LastSwingLowPrice  *float64
}

// Candle represents a single OHLCV candle for computation
type Candle struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume *int64
}

// ComputeBasicIndicators computes deterministic weekly indicators
// NO EMA yet - that requires historical data
func ComputeBasicIndicators(candle Candle) WeeklyIndicators {
	open := candle.Open
	close := candle.Close
	high := candle.High
	low := candle.Low

	rangeSize := high - low
	bodySize := math.Abs(close - open)

	upperWick := high - math.Max(open, close)
	lowerWick := math.Min(open, close) - low

	midPrice := (high + low) / 2.0

	return WeeklyIndicators{
		RangeSize: rangeSize,
		BodySize:  bodySize,
		UpperWick: upperWick,
		LowerWick: lowerWick,
		MidPrice:  midPrice,

		// EMA values placeholder (need historical candles)
		EMA20:  0,
		EMA50:  0,
		EMA200: 0,

		// Swing points placeholder (need previous candles)
		LastSwingHighPrice: nil,
		LastSwingLowPrice:  nil,
	}
}
