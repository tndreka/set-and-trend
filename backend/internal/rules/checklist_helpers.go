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

// IsEngulfing returns true if curr's body fully engulfs prev's body in the
// given direction.
//
//   LONG  engulfing: curr closes above prev's open AND curr opens below prev's close
//   SHORT engulfing: curr closes below prev's open AND curr opens above prev's close
func IsEngulfing(prev, curr Candle, direction string) bool {
	if direction == "LONG" {
		return curr.Close > curr.Open &&
			curr.Open <= math.Min(prev.Open, prev.Close) &&
			curr.Close >= math.Max(prev.Open, prev.Close)
	}
	return curr.Close < curr.Open &&
		curr.Open >= math.Max(prev.Open, prev.Close) &&
		curr.Close <= math.Min(prev.Open, prev.Close)
}

// NearPsychLevel returns true if the price is within `withinPips` pips of a
// round .X000 or .X500 level (or X.00 / X.50 for JPY).
func NearPsychLevel(price float64, isJPY bool) bool {
	var pipSize, interval float64
	if isJPY {
		pipSize = 0.01
		interval = 0.50
	} else {
		pipSize = 0.0001
		interval = 0.0500
	}
	threshold := 20 * pipSize
	rem := math.Mod(price, interval)
	if rem < 0 {
		rem += interval
	}
	return rem < threshold || (interval-rem) < threshold
}

// NearEMA returns true if the close is within `pct` fraction of the EMA value.
// A pct of 0.005 means 0.5% proximity.
func NearEMA(close, ema, pct float64) bool {
	if ema <= 0 {
		return false
	}
	return math.Abs(close-ema)/ema < pct
}

// IsJPYPair returns true for JPY-denominated pairs (pip = 0.01 not 0.0001).
func IsJPYPair(symbol string) bool {
	return len(symbol) >= 6 && symbol[3:6] == "JPY"
}
