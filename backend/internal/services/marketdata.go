package services

import (
	"math"

	//"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/domain"
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

	LastSwingHigh *float64
	LastSwingLow  *float64
}

// ComputeIndicators computes deterministic weekly indicators.
// NO DB. NO IO. PURE FUNCTION.
func ComputeIndicators(
	candles []domain.Candle,
	index int,
) WeeklyIndicators {

	c := candles[index]

	open := c.Open
	close := c.Close
	high := c.High
	low := c.Low

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

		// EMA + swings come next
	}
}
