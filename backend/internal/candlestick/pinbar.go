package candlestick

import (
	"time"

	"set-and-trend/backend/internal/database"
)

// PinBar represents a rejection candlestick pattern
// Long wick indicates price rejection at a level
type PinBar struct {
	Direction   int       // 1=bullish (hammer), -1=bearish (shooting star)
	WickRatio   float64   // Wick length / total range (should be > 0.60)
	BodyRatio   float64   // Body / total range (should be < 0.25)
	WickLength  float64   // Absolute wick length
	Index       int       // Bar index
	Timestamp   time.Time // Time of the pin bar
	Strength    float64   // Overall quality score 0-1
}

// DetectPinBars finds pin bar patterns in the candle data
// minWickRatio: minimum wick as percentage of total range (default 0.60)
// maxBodyRatio: maximum body as percentage of total range (default 0.30)
func DetectPinBars(candles []database.Candle, minWickRatio, maxBodyRatio float64) []PinBar {
	if len(candles) < 1 {
		return nil
	}

	if minWickRatio <= 0 {
		minWickRatio = 0.60
	}
	if maxBodyRatio <= 0 {
		maxBodyRatio = 0.30
	}

	var pinBars []PinBar

	for i := 0; i < len(candles); i++ {
		c := candles[i]
		
		totalRange := c.High - c.Low
		if totalRange <= 0 {
			continue
		}

		body := abs(c.Close - c.Open)
		bodyRatio := body / totalRange

		// Skip if body is too large
		if bodyRatio > maxBodyRatio {
			continue
		}

		// Calculate wick lengths
		var upperWick, lowerWick float64
		if c.Close >= c.Open { // Bullish candle
			upperWick = c.High - c.Close
			lowerWick = c.Open - c.Low
		} else { // Bearish candle
			upperWick = c.High - c.Open
			lowerWick = c.Close - c.Low
		}

		upperWickRatio := upperWick / totalRange
		lowerWickRatio := lowerWick / totalRange

		// Check for Bullish Pin Bar (Hammer)
		// Long lower wick, small body near the top
		if lowerWickRatio >= minWickRatio && upperWickRatio < 0.15 {
			strength := calculatePinBarStrength(lowerWickRatio, bodyRatio, true)
			pinBars = append(pinBars, PinBar{
				Direction:   1,
				WickRatio:   lowerWickRatio,
				BodyRatio:   bodyRatio,
				WickLength:  lowerWick,
				Index:       i,
				Timestamp:   c.Timestamp,
				Strength:    strength,
			})
		}

		// Check for Bearish Pin Bar (Shooting Star)
		// Long upper wick, small body near the bottom
		if upperWickRatio >= minWickRatio && lowerWickRatio < 0.15 {
			strength := calculatePinBarStrength(upperWickRatio, bodyRatio, false)
			pinBars = append(pinBars, PinBar{
				Direction:   -1,
				WickRatio:   upperWickRatio,
				BodyRatio:   bodyRatio,
				WickLength:  upperWick,
				Index:       i,
				Timestamp:   c.Timestamp,
				Strength:    strength,
			})
		}
	}

	return pinBars
}

// DetectPinBarAtIndex checks if a specific candle is a pin bar
func DetectPinBarAtIndex(candles []database.Candle, index int, minWickRatio, maxBodyRatio float64) *PinBar {
	if index < 0 || index >= len(candles) {
		return nil
	}

	if minWickRatio <= 0 {
		minWickRatio = 0.60
	}
	if maxBodyRatio <= 0 {
		maxBodyRatio = 0.30
	}

	c := candles[index]
	
	totalRange := c.High - c.Low
	if totalRange <= 0 {
		return nil
	}

	body := abs(c.Close - c.Open)
	bodyRatio := body / totalRange

	if bodyRatio > maxBodyRatio {
		return nil
	}

	var upperWick, lowerWick float64
	if c.Close >= c.Open {
		upperWick = c.High - c.Close
		lowerWick = c.Open - c.Low
	} else {
		upperWick = c.High - c.Open
		lowerWick = c.Close - c.Low
	}

	upperWickRatio := upperWick / totalRange
	lowerWickRatio := lowerWick / totalRange

	// Bullish Pin Bar
	if lowerWickRatio >= minWickRatio && upperWickRatio < 0.15 {
		return &PinBar{
			Direction:   1,
			WickRatio:   lowerWickRatio,
			BodyRatio:   bodyRatio,
			WickLength:  lowerWick,
			Index:       index,
			Timestamp:   c.Timestamp,
			Strength:    calculatePinBarStrength(lowerWickRatio, bodyRatio, true),
		}
	}

	// Bearish Pin Bar
	if upperWickRatio >= minWickRatio && lowerWickRatio < 0.15 {
		return &PinBar{
			Direction:   -1,
			WickRatio:   upperWickRatio,
			BodyRatio:   bodyRatio,
			WickLength:  upperWick,
			Index:       index,
			Timestamp:   c.Timestamp,
			Strength:    calculatePinBarStrength(upperWickRatio, bodyRatio, false),
		}
	}

	return nil
}

// IsPinBarAtLevel checks if a pin bar formed at a specific price level
func IsPinBarAtLevel(pinBar *PinBar, candles []database.Candle, level float64, tolerance float64) bool {
	if pinBar == nil || pinBar.Index >= len(candles) {
		return false
	}

	c := candles[pinBar.Index]
	
	// For bullish pin bars, check if the low touched the level
	if pinBar.Direction == 1 {
		return c.Low >= level*(1-tolerance) && c.Low <= level*(1+tolerance)
	}
	
	// For bearish pin bars, check if the high touched the level
	return c.High >= level*(1-tolerance) && c.High <= level*(1+tolerance)
}

// GetRecentPinBar finds the most recent pin bar within N bars
func GetRecentPinBar(pinBars []PinBar, currentBar int, maxBarsAgo int, direction int) *PinBar {
	for i := len(pinBars) - 1; i >= 0; i-- {
		pb := pinBars[i]
		if direction != 0 && pb.Direction != direction {
			continue
		}
		if currentBar-pb.Index <= maxBarsAgo {
			return &pb
		}
	}
	return nil
}

// HasPinBarConfirmation checks if there's a recent pin bar supporting a trade direction
func HasPinBarConfirmation(candles []database.Candle, currentBar int, direction int, lookback int) bool {
	if currentBar < 1 || currentBar >= len(candles) {
		return false
	}

	start := currentBar - lookback
	if start < 0 {
		start = 0
	}

	for i := currentBar - 1; i >= start; i-- {
		pb := DetectPinBarAtIndex(candles, i, 0.60, 0.30)
		if pb != nil && pb.Direction == direction {
			return true
		}
	}

	return false
}

// calculatePinBarStrength calculates a quality score for the pin bar
func calculatePinBarStrength(wickRatio, bodyRatio float64, isBullish bool) float64 {
	// Higher wick ratio = stronger
	wickScore := wickRatio // Already 0.6-1.0 range
	
	// Lower body ratio = stronger
	bodyScore := 1 - (bodyRatio / 0.30) // Normalized to 0-1
	if bodyScore < 0 {
		bodyScore = 0
	}
	
	// Weighted average
	strength := wickScore*0.7 + bodyScore*0.3
	
	if strength > 1 {
		strength = 1
	}
	
	return strength
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
