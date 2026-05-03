package rules

import "math"

// IsCandleRejection returns true if the candle shows rejection in the given
// direction. A rejection candle has a wick on the opposite side that is ≥ 2×
// the body size, plus a body that closes in the direction of the trade.
//
//   LONG  rejection: lower wick ≥ 2× body, close > open (bullish)
//   SHORT rejection: upper wick ≥ 2× body, close < open (bearish)
func IsCandleRejection(c Candle, direction string) bool {
	body := math.Abs(c.Close - c.Open)
	if body <= 0 {
		return false
	}
	if direction == "LONG" {
		lowerWick := math.Min(c.Open, c.Close) - c.Low
		return c.Close > c.Open && lowerWick >= 2*body
	}
	upperWick := c.High - math.Max(c.Open, c.Close)
	return c.Close < c.Open && upperWick >= 2*body
}

// NearEMA returns true if the close is within `pct` fraction of the EMA value.
// A pct of 0.005 means 0.5% proximity.
func NearEMA(close, ema, pct float64) bool {
	if ema <= 0 {
		return false
	}
	return math.Abs(close-ema)/ema < pct
}
