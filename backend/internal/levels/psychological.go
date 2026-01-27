package levels

import (
	"math"
	"strings"
)

// PsychLevel represents a psychological price level
type PsychLevel struct {
	Level    float64
	Type     string  // "major" (1.0000), "mid" (1.0500), "minor" (1.0250)
	Strength float64 // 1.0 for major, 0.75 for mid, 0.5 for minor
}

// GetPsychLevels returns psychological levels around a price
func GetPsychLevels(price float64, symbol string, count int) []PsychLevel {
	increment := getPsychIncrement(symbol)
	majorIncrement := getMajorPsychIncrement(symbol)

	if count <= 0 {
		count = 5
	}

	var levels []PsychLevel

	// Find the nearest base level
	baseLevel := math.Floor(price/increment) * increment

	// Generate levels above and below
	for i := -count; i <= count; i++ {
		level := baseLevel + float64(i)*increment
		if level <= 0 {
			continue
		}

		levelType := "minor"
		strength := 0.5

		// Check if it's a major level
		remainder := math.Mod(level, majorIncrement)
		if math.Abs(remainder) < 0.0001 || math.Abs(remainder-majorIncrement) < 0.0001 {
			levelType = "major"
			strength = 1.0
		} else if math.Mod(level, increment*2) < 0.0001 {
			levelType = "mid"
			strength = 0.75
		}

		levels = append(levels, PsychLevel{
			Level:    level,
			Type:     levelType,
			Strength: strength,
		})
	}

	return levels
}

// GetNearestPsychLevel finds the closest psychological level to a price
func GetNearestPsychLevel(price float64, symbol string) float64 {
	increment := getPsychIncrement(symbol)
	return math.Round(price/increment) * increment
}

// IsNearPsychLevel checks if price is within tolerance of a psychological level
func IsNearPsychLevel(price float64, symbol string, tolerancePercent float64) bool {
	if tolerancePercent <= 0 {
		tolerancePercent = 0.002 // 0.2% default
	}

	nearestLevel := GetNearestPsychLevel(price, symbol)
	distance := math.Abs(price-nearestLevel) / price

	return distance <= tolerancePercent
}

// GetNearPsychLevelInfo returns info about the nearest psychological level
func GetNearPsychLevelInfo(price float64, symbol string, tolerancePercent float64) *PsychLevel {
	if tolerancePercent <= 0 {
		tolerancePercent = 0.002
	}

	increment := getPsychIncrement(symbol)
	majorIncrement := getMajorPsychIncrement(symbol)

	nearestLevel := GetNearestPsychLevel(price, symbol)
	distance := math.Abs(price-nearestLevel) / price

	if distance > tolerancePercent {
		return nil
	}

	levelType := "minor"
	strength := 0.5

	remainder := math.Mod(nearestLevel, majorIncrement)
	if math.Abs(remainder) < 0.0001 || math.Abs(remainder-majorIncrement) < 0.0001 {
		levelType = "major"
		strength = 1.0
	} else if math.Mod(nearestLevel, increment*2) < 0.0001 {
		levelType = "mid"
		strength = 0.75
	}

	return &PsychLevel{
		Level:    nearestLevel,
		Type:     levelType,
		Strength: strength,
	}
}

// GetPsychLevelScore returns a score based on proximity to psychological levels
// Returns 0-1 where 1 is exactly at a major level
func GetPsychLevelScore(price float64, symbol string) float64 {
	increment := getPsychIncrement(symbol)
	majorIncrement := getMajorPsychIncrement(symbol)

	nearestLevel := GetNearestPsychLevel(price, symbol)
	distance := math.Abs(price-nearestLevel) / increment

	// Distance score (closer = higher)
	distanceScore := 1.0 - math.Min(distance, 1.0)

	// Level type bonus
	levelBonus := 0.0
	remainder := math.Mod(nearestLevel, majorIncrement)
	if math.Abs(remainder) < 0.0001 || math.Abs(remainder-majorIncrement) < 0.0001 {
		levelBonus = 0.2 // Major level bonus
	} else if math.Mod(nearestLevel, increment*2) < 0.0001 {
		levelBonus = 0.1 // Mid level bonus
	}

	score := distanceScore + levelBonus
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// IsPriceAtMajorLevel checks if price is at a major psychological level
func IsPriceAtMajorLevel(price float64, symbol string, tolerancePercent float64) bool {
	if tolerancePercent <= 0 {
		tolerancePercent = 0.001
	}

	majorIncrement := getMajorPsychIncrement(symbol)
	nearestMajor := math.Round(price/majorIncrement) * majorIncrement
	distance := math.Abs(price-nearestMajor) / price

	return distance <= tolerancePercent
}

// getPsychIncrement returns the minor psychological level increment for a symbol type
func getPsychIncrement(symbol string) float64 {
	symbol = strings.ToUpper(symbol)

	// JPY pairs have different pip values
	if strings.Contains(symbol, "JPY") {
		return 0.50 // 50 pips for JPY pairs
	}

	// Gold
	if strings.HasPrefix(symbol, "XAU") {
		return 10.0 //  levels for gold
	}

	// Silver
	if strings.HasPrefix(symbol, "XAG") {
		return 0.50 // /bin/zsh.50 levels for silver
	}

	// Indices
	if strings.Contains(symbol, "IDX") || strings.Contains(symbol, "USA") {
		if strings.Contains(symbol, "30") {
			return 100.0 // 100 points for Dow
		}
		if strings.Contains(symbol, "500") || strings.Contains(symbol, "TECH") {
			return 25.0 // 25 points for S&P and Nasdaq
		}
		return 50.0 // Default for indices
	}

	// Oil
	if strings.Contains(symbol, "BRENT") || strings.Contains(symbol, "LIGHT") || strings.Contains(symbol, "OIL") {
		return 0.50 // /bin/zsh.50 levels for oil
	}

	// Standard forex pairs
	return 0.0050 // 50 pips for standard pairs
}

// getMajorPsychIncrement returns the major psychological level increment
func getMajorPsychIncrement(symbol string) float64 {
	symbol = strings.ToUpper(symbol)

	// JPY pairs
	if strings.Contains(symbol, "JPY") {
		return 1.00 // Full yen levels (110.00, 111.00, etc.)
	}

	// Gold
	if strings.HasPrefix(symbol, "XAU") {
		return 50.0 //  levels (1900, 1950, 2000)
	}

	// Silver
	if strings.HasPrefix(symbol, "XAG") {
		return 1.00 //  levels
	}

	// Indices
	if strings.Contains(symbol, "IDX") || strings.Contains(symbol, "USA") {
		if strings.Contains(symbol, "30") {
			return 1000.0 // 1000 points for Dow (30000, 31000)
		}
		return 100.0 // 100 points for S&P/Nasdaq
	}

	// Oil
	if strings.Contains(symbol, "BRENT") || strings.Contains(symbol, "LIGHT") || strings.Contains(symbol, "OIL") {
		return 5.00 //  levels (70, 75, 80)
	}

	// Standard forex - full cent/figure levels
	return 0.0100 // 100 pips (1.1000, 1.1100, etc.)
}
