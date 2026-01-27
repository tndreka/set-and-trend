package candlestick

import (
	"time"

	"set-and-trend/backend/internal/database"
)

// EngulfingPattern represents a candlestick engulfing pattern
// Bullish: Current bullish candle body completely engulfs previous bearish body
// Bearish: Current bearish candle body completely engulfs previous bullish body
type EngulfingPattern struct {
	Direction   int       // 1=bullish, -1=bearish
	Index       int       // Bar index (of the engulfing candle)
	Timestamp   time.Time // Time of the pattern
	EngulfRatio float64   // How much larger the engulfing candle is
	Strength    float64   // Overall quality score 0-1
}

// DetectEngulfing finds engulfing patterns in the candle data
func DetectEngulfing(candles []database.Candle) []EngulfingPattern {
	if len(candles) < 2 {
		return nil
	}

	var patterns []EngulfingPattern

	for i := 1; i < len(candles); i++ {
		prev := candles[i-1]
		curr := candles[i]

		pattern := checkEngulfing(prev, curr, i)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns
}

// DetectEngulfingAtIndex checks if a specific candle creates an engulfing pattern
func DetectEngulfingAtIndex(candles []database.Candle, index int) *EngulfingPattern {
	if index < 1 || index >= len(candles) {
		return nil
	}

	prev := candles[index-1]
	curr := candles[index]

	return checkEngulfing(prev, curr, index)
}

// checkEngulfing determines if two candles form an engulfing pattern
func checkEngulfing(prev, curr database.Candle, index int) *EngulfingPattern {
	prevBody := abs(prev.Close - prev.Open)
	currBody := abs(curr.Close - curr.Open)

	// Skip if bodies are too small (doji-like candles)
	if prevBody < (prev.High-prev.Low)*0.1 || currBody < (curr.High-curr.Low)*0.1 {
		return nil
	}

	// Get body boundaries
	prevBodyHigh := max(prev.Open, prev.Close)
	prevBodyLow := min(prev.Open, prev.Close)
	currBodyHigh := max(curr.Open, curr.Close)
	currBodyLow := min(curr.Open, curr.Close)

	// Check for Bullish Engulfing
	// Previous candle bearish (close < open)
	// Current candle bullish (close > open)
	// Current body completely engulfs previous body
	if prev.Close < prev.Open && curr.Close > curr.Open {
		if currBodyLow <= prevBodyLow && currBodyHigh >= prevBodyHigh {
			engulfRatio := currBody / prevBody
			strength := calculateEngulfingStrength(prev, curr, engulfRatio, true)
			
			return &EngulfingPattern{
				Direction:   1,
				Index:       index,
				Timestamp:   curr.Timestamp,
				EngulfRatio: engulfRatio,
				Strength:    strength,
			}
		}
	}

	// Check for Bearish Engulfing
	// Previous candle bullish (close > open)
	// Current candle bearish (close < open)
	// Current body completely engulfs previous body
	if prev.Close > prev.Open && curr.Close < curr.Open {
		if currBodyLow <= prevBodyLow && currBodyHigh >= prevBodyHigh {
			engulfRatio := currBody / prevBody
			strength := calculateEngulfingStrength(prev, curr, engulfRatio, false)
			
			return &EngulfingPattern{
				Direction:   -1,
				Index:       index,
				Timestamp:   curr.Timestamp,
				EngulfRatio: engulfRatio,
				Strength:    strength,
			}
		}
	}

	return nil
}

// GetRecentEngulfing finds the most recent engulfing pattern within N bars
func GetRecentEngulfing(patterns []EngulfingPattern, currentBar int, maxBarsAgo int, direction int) *EngulfingPattern {
	for i := len(patterns) - 1; i >= 0; i-- {
		p := patterns[i]
		if direction != 0 && p.Direction != direction {
			continue
		}
		if currentBar-p.Index <= maxBarsAgo {
			return &p
		}
	}
	return nil
}

// HasEngulfingConfirmation checks if there's a recent engulfing pattern supporting the trade
func HasEngulfingConfirmation(candles []database.Candle, currentBar int, direction int, lookback int) bool {
	if currentBar < 2 || currentBar >= len(candles) {
		return false
	}

	start := currentBar - lookback
	if start < 1 {
		start = 1
	}

	for i := currentBar - 1; i >= start; i-- {
		pattern := DetectEngulfingAtIndex(candles, i)
		if pattern != nil && pattern.Direction == direction {
			return true
		}
	}

	return false
}

// calculateEngulfingStrength calculates a quality score
func calculateEngulfingStrength(prev, curr database.Candle, engulfRatio float64, isBullish bool) float64 {
	// Factor 1: Engulf ratio (larger is better, but cap at 3x)
	ratioScore := engulfRatio / 3.0
	if ratioScore > 1 {
		ratioScore = 1
	}

	// Factor 2: Relative volume (if current candle has higher volume)
	volumeScore := 0.5
	if curr.Volume > 0 && prev.Volume > 0 {
		volRatio := curr.Volume / prev.Volume
		if volRatio > 1.5 {
			volumeScore = 1.0
		} else if volRatio > 1.0 {
			volumeScore = 0.7
		}
	}

	// Factor 3: Close position (should close near extreme)
	currRange := curr.High - curr.Low
	closeScore := 0.5
	if currRange > 0 {
		if isBullish {
			// Bullish engulfing should close near the high
			closePosition := (curr.Close - curr.Low) / currRange
			closeScore = closePosition
		} else {
			// Bearish engulfing should close near the low
			closePosition := (curr.High - curr.Close) / currRange
			closeScore = closePosition
		}
	}

	// Weighted average
	strength := ratioScore*0.4 + volumeScore*0.3 + closeScore*0.3
	
	if strength > 1 {
		strength = 1
	}
	
	return strength
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
