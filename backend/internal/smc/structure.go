package smc

import (
	"time"

	"set-and-trend/backend/internal/database"
)

// StructureShift represents a Break of Structure (BOS) or Change of Character (CHoCH)
// BOS: Price breaks a swing in the direction of the trend (continuation)
// CHoCH: Price breaks a swing against the trend (potential reversal)
type StructureShift struct {
	Type      string    // "BOS" or "CHoCH"
	Direction int       // 1=bullish (broke above), -1=bearish (broke below)
	Level     float64   // The swing level that was broken
	Index     int       // Bar index where the break occurred
	Timestamp time.Time // Time of the break
	Strength  float64   // How decisively the level was broken (close vs wick)
}

// MarketStructure tracks the overall structure state
type MarketStructure struct {
	Trend              int              // Current trend: 1=bullish, -1=bearish, 0=ranging
	LastHigherHigh     float64          // Most recent higher high
	LastHigherLow      float64          // Most recent higher low
	LastLowerHigh      float64          // Most recent lower high
	LastLowerLow       float64          // Most recent lower low
	RecentShifts       []StructureShift // Recent structure shifts
	SwingHighs         []SwingPoint     // Tracked swing highs
	SwingLows          []SwingPoint     // Tracked swing lows
}

// DetectStructureShifts analyzes candles and swings to find BOS/CHoCH
func DetectStructureShifts(candles []database.Candle, swingHighs, swingLows []SwingPoint) []StructureShift {
	if len(candles) < 10 || len(swingHighs) < 2 || len(swingLows) < 2 {
		return nil
	}

	var shifts []StructureShift
	
	// Track the current trend by comparing swing points
	trend := determineTrend(swingHighs, swingLows)

	// Look for breaks of recent swing points
	for i := 0; i < len(candles); i++ {
		candle := candles[i]

		// Check for bullish breaks (breaking above swing highs)
		for _, sh := range swingHighs {
			if sh.Index >= i { // Only consider swings before current candle
				continue
			}
			
			// Check if this candle breaks above the swing high
			if candle.Close > sh.Price {
				// Determine if BOS or CHoCH
				shiftType := "BOS"
				if trend == -1 { // Breaking high in downtrend = CHoCH (reversal signal)
					shiftType = "CHoCH"
				}
				
				// Calculate strength (how much of the candle closed above)
				strength := (candle.Close - sh.Price) / (candle.High - candle.Low + 0.00001)
				
				shifts = append(shifts, StructureShift{
					Type:      shiftType,
					Direction: 1,
					Level:     sh.Price,
					Index:     i,
					Timestamp: candle.Timestamp,
					Strength:  strength,
				})
				break // Only count first break per candle
			}
		}

		// Check for bearish breaks (breaking below swing lows)
		for _, sl := range swingLows {
			if sl.Index >= i {
				continue
			}
			
			if candle.Close < sl.Price {
				shiftType := "BOS"
				if trend == 1 { // Breaking low in uptrend = CHoCH
					shiftType = "CHoCH"
				}
				
				strength := (sl.Price - candle.Close) / (candle.High - candle.Low + 0.00001)
				
				shifts = append(shifts, StructureShift{
					Type:      shiftType,
					Direction: -1,
					Level:     sl.Price,
					Index:     i,
					Timestamp: candle.Timestamp,
					Strength:  strength,
				})
				break
			}
		}
	}

	// Remove duplicates and keep only significant shifts
	return filterSignificantShifts(shifts)
}

// AnalyzeMarketStructure builds a comprehensive market structure view
func AnalyzeMarketStructure(candles []database.Candle, lookback int) *MarketStructure {
	if len(candles) < lookback*2+5 {
		return nil
	}

	swingHighs := findSwingHighs(candles, lookback)
	swingLows := findSwingLows(candles, lookback)

	if len(swingHighs) < 2 || len(swingLows) < 2 {
		return &MarketStructure{Trend: 0}
	}

	ms := &MarketStructure{
		SwingHighs: swingHighs,
		SwingLows:  swingLows,
	}

	// Analyze higher highs/lows and lower highs/lows
	ms.Trend = determineTrend(swingHighs, swingLows)
	
	// Set last significant levels
	if len(swingHighs) >= 2 {
		lastTwo := swingHighs[len(swingHighs)-2:]
		if lastTwo[1].Price > lastTwo[0].Price {
			ms.LastHigherHigh = lastTwo[1].Price
		} else {
			ms.LastLowerHigh = lastTwo[1].Price
		}
	}
	
	if len(swingLows) >= 2 {
		lastTwo := swingLows[len(swingLows)-2:]
		if lastTwo[1].Price > lastTwo[0].Price {
			ms.LastHigherLow = lastTwo[1].Price
		} else {
			ms.LastLowerLow = lastTwo[1].Price
		}
	}

	ms.RecentShifts = DetectStructureShifts(candles, swingHighs, swingLows)

	return ms
}

// GetLastCHoCH returns the most recent Change of Character
func GetLastCHoCH(shifts []StructureShift) *StructureShift {
	for i := len(shifts) - 1; i >= 0; i-- {
		if shifts[i].Type == "CHoCH" {
			return &shifts[i]
		}
	}
	return nil
}

// GetLastBOS returns the most recent Break of Structure
func GetLastBOS(shifts []StructureShift) *StructureShift {
	for i := len(shifts) - 1; i >= 0; i-- {
		if shifts[i].Type == "BOS" {
			return &shifts[i]
		}
	}
	return nil
}

// HasRecentStructureShift checks if there's been a structure shift in the last N bars
func HasRecentStructureShift(shifts []StructureShift, direction int, maxBarsAgo int, currentBar int) bool {
	for _, s := range shifts {
		if direction != 0 && s.Direction != direction {
			continue
		}
		if currentBar-s.Index <= maxBarsAgo {
			return true
		}
	}
	return false
}

// IsPriceRejectedFromStructure checks if price is bouncing off a key structure level
func IsPriceRejectedFromStructure(candle database.Candle, ms *MarketStructure, tolerance float64) bool {
	if ms == nil {
		return false
	}

	// Check rejection from swing highs
	for _, sh := range ms.SwingHighs {
		upperBound := sh.Price * (1 + tolerance)
		lowerBound := sh.Price * (1 - tolerance)
		
		// Candle wicked into the zone but closed away
		if candle.High >= lowerBound && candle.High <= upperBound {
			if candle.Close < lowerBound {
				return true // Rejected from above
			}
		}
	}

	// Check rejection from swing lows
	for _, sl := range ms.SwingLows {
		upperBound := sl.Price * (1 + tolerance)
		lowerBound := sl.Price * (1 - tolerance)
		
		if candle.Low >= lowerBound && candle.Low <= upperBound {
			if candle.Close > upperBound {
				return true // Rejected from below
			}
		}
	}

	return false
}

// Helper: find swing highs
func findSwingHighs(candles []database.Candle, lookback int) []SwingPoint {
	var swings []SwingPoint
	for i := lookback; i < len(candles)-lookback; i++ {
		isHigh := true
		for j := 1; j <= lookback; j++ {
			if candles[i-j].High >= candles[i].High || candles[i+j].High >= candles[i].High {
				isHigh = false
				break
			}
		}
		if isHigh {
			swings = append(swings, SwingPoint{
				Index:  i,
				Price:  candles[i].High,
				IsHigh: true,
			})
		}
	}
	return swings
}

// Helper: find swing lows
func findSwingLows(candles []database.Candle, lookback int) []SwingPoint {
	var swings []SwingPoint
	for i := lookback; i < len(candles)-lookback; i++ {
		isLow := true
		for j := 1; j <= lookback; j++ {
			if candles[i-j].Low <= candles[i].Low || candles[i+j].Low <= candles[i].Low {
				isLow = false
				break
			}
		}
		if isLow {
			swings = append(swings, SwingPoint{
				Index:  i,
				Price:  candles[i].Low,
				IsHigh: false,
			})
		}
	}
	return swings
}

// Helper: determine trend from swing points
func determineTrend(highs, lows []SwingPoint) int {
	if len(highs) < 2 || len(lows) < 2 {
		return 0
	}

	// Check last two highs and lows
	lastHighs := highs[len(highs)-2:]
	lastLows := lows[len(lows)-2:]

	higherHighs := lastHighs[1].Price > lastHighs[0].Price
	higherLows := lastLows[1].Price > lastLows[0].Price
	lowerHighs := lastHighs[1].Price < lastHighs[0].Price
	lowerLows := lastLows[1].Price < lastLows[0].Price

	if higherHighs && higherLows {
		return 1 // Bullish trend
	}
	if lowerHighs && lowerLows {
		return -1 // Bearish trend
	}
	return 0 // Ranging
}

// Helper: filter to keep only significant shifts
func filterSignificantShifts(shifts []StructureShift) []StructureShift {
	if len(shifts) == 0 {
		return nil
	}

	// Keep only shifts with minimum strength
	var filtered []StructureShift
	lastIndex := make(map[int]int) // direction -> last index

	for _, s := range shifts {
		// Skip if too close to previous shift in same direction
		if lastIdx, ok := lastIndex[s.Direction]; ok && s.Index-lastIdx < 5 {
			continue
		}
		
		if s.Strength >= 0.3 { // Minimum 30% strength
			filtered = append(filtered, s)
			lastIndex[s.Direction] = s.Index
		}
	}

	return filtered
}
