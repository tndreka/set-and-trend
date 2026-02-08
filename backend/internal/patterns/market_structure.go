package patterns

import (
	"fmt"
	"math"
	"time"
)

// ================================================================================
// MARKET STRUCTURE - Break of Structure (BOS) & Change of Character (CHoCH)
// ================================================================================

type StructureType string

const (
	StructureBullish StructureType = "BULLISH"
	StructureBearish StructureType = "BEARISH"
	StructureRanging StructureType = "RANGING"
)

type BreakType string

const (
	BreakBOS   BreakType = "BOS"
	BreakCHoCH BreakType = "CHOCH"
)

type StructureBreak struct {
	Type           BreakType
	Direction      string
	BreakLevel     float64
	BreakPrice     float64
	BreakIndex     int
	BreakTime      time.Time
	PriorStructure StructureType
	NewStructure   StructureType
	Strength       float64
	IsRetested     bool
	RetestIndex    int
}

// MarketStructure holds the current structure state
type MarketStructure struct {
	CurrentStructure StructureType
	LastSwingHigh    float64
	LastSwingHighIdx int
	LastSwingLow     float64
	LastSwingLowIdx  int
	ProtectedHigh    float64 // ADDED: Last swing high that hasn't been broken
	ProtectedHighIdx int     // ADDED
	ProtectedLow     float64 // ADDED: Last swing low that hasn't been broken
	ProtectedLowIdx  int     // ADDED
	PreviousHigher   bool
	PreviousLower    bool
	RecentBreaks     []StructureBreak
	Strength         float64
}

type StructureConfig struct {
	SwingLookback   int
	MinBreakPips    float64
	RetestTolerance float64
	MaxBreakAge     int
	PipSize         float64
}

func DefaultStructureConfig() StructureConfig {
	return StructureConfig{
		SwingLookback:   3,
		MinBreakPips:    5,
		RetestTolerance: 0.002,
		MaxBreakAge:     20,
		PipSize:         0.0001,
	}
}

// AnalyzeMarketStructure determines the current market structure and any breaks
func AnalyzeMarketStructure(candles []Candle, config StructureConfig) *MarketStructure {
	if len(candles) < config.SwingLookback*4 {
		return nil
	}

	// Find swing points
	highs := FindSwingHighs(candles, config.SwingLookback)
	lows := FindSwingLows(candles, config.SwingLookback)

	if len(highs) < 2 || len(lows) < 2 {
		return &MarketStructure{
			CurrentStructure: StructureRanging,
			Strength:         0.3,
		}
	}

	// Get the two most recent swing highs and lows
	lastHigh := highs[len(highs)-1]
	prevHigh := highs[len(highs)-2]
	lastLow := lows[len(lows)-1]
	prevLow := lows[len(lows)-2]

	// Determine structure
	currentStructure := StructureRanging
	strength := 0.3

	if lastHigh.Price > prevHigh.Price && lastLow.Price > prevLow.Price {
		currentStructure = StructureBullish
		strength = calculateStructureStrength(highs, lows, true)
	} else if lastHigh.Price < prevHigh.Price && lastLow.Price < prevLow.Price {
		currentStructure = StructureBearish
		strength = calculateStructureStrength(highs, lows, false)
	}

	ms := &MarketStructure{
		CurrentStructure: currentStructure,
		LastSwingHigh:    lastHigh.Price,
		LastSwingHighIdx: lastHigh.Index,
		LastSwingLow:     lastLow.Price,
		LastSwingLowIdx:  lastLow.Index,
		ProtectedHigh:    lastHigh.Price, // ADDED: Initialize protected levels
		ProtectedHighIdx: lastHigh.Index, // ADDED
		ProtectedLow:     lastLow.Price,  // ADDED
		ProtectedLowIdx:  lastLow.Index,  // ADDED
		PreviousHigher:   lastHigh.Price > prevHigh.Price,
		PreviousLower:    lastLow.Price < prevLow.Price,
		Strength:         strength,
	}

	// Detect structure breaks
	ms.RecentBreaks = detectStructureBreaks(candles, highs, lows, config, currentStructure)

	return ms
}

// detectStructureBreaks finds BOS and CHoCH events
// FIXED: Correct BOS/CHoCH logic + protected level tracking + RANGING handling
func detectStructureBreaks(candles []Candle, highs, lows []SwingPoint, config StructureConfig, initialStructure StructureType) []StructureBreak {
	var breaks []StructureBreak
	n := len(candles)

	if len(highs) < 2 || len(lows) < 2 {
		return breaks
	}

	// Track structure state as we iterate
	priorStructure := initialStructure

	// Track protected levels (last swing that hasn't been broken)
	protectedHigh := highs[len(highs)-1]
	protectedLow := lows[len(lows)-1]

	// Track which direction was last broken to prevent conflicting signals
	lastBreakDirection := ""

	// Look for breaks of protected swing lows
	for j := protectedLow.Index + 1; j < n && j <= protectedLow.Index+config.MaxBreakAge; j++ {
		c := candles[j]

		breakPips := (protectedLow.Price - c.Low) / config.PipSize
		if breakPips < config.MinBreakPips || c.Close >= protectedLow.Price {
			continue
		}

		// VALID BREAK OF LOW DETECTED
		// Determine break type based on prior structure (FIXED LOGIC)
		var breakType BreakType
		var newStructure StructureType

		if priorStructure == StructureBullish {
			// Breaking low in bullish structure = REVERSAL (CHoCH)
			breakType = BreakCHoCH
			newStructure = StructureBearish
		} else if priorStructure == StructureBearish {
			// Breaking low in bearish structure = CONTINUATION (BOS)
			breakType = BreakBOS
			newStructure = StructureBearish
		} else { // RANGING
			// First break from ranging = CHoCH (not BOS)
			breakType = BreakCHoCH
			newStructure = StructureBearish
		}

		brk := StructureBreak{
			Type:           breakType,
			Direction:      "BEARISH",
			BreakLevel:     protectedLow.Price,
			BreakPrice:     c.Low,
			BreakIndex:     j,
			BreakTime:      c.Timestamp,
			PriorStructure: priorStructure,
			NewStructure:   newStructure,
			Strength:       calculateBreakStrength(c, protectedLow.Price, breakPips, config),
		}

		brk.IsRetested, brk.RetestIndex = checkBreakRetest(candles, j, protectedLow.Price, "BEARISH", config)

		breaks = append(breaks, brk)
		priorStructure = newStructure
		lastBreakDirection = "BEARISH"

		// Update protected low to next swing low
		for i := len(lows) - 2; i >= 0; i-- {
			if lows[i].Index < protectedLow.Index {
				protectedLow = lows[i]
				break
			}
		}

		break // Only count first break of this level
	}

	// Look for breaks of protected swing highs
	// Skip if we already had a bearish break (prevent conflicting signals)
	if lastBreakDirection != "BEARISH" {
		for j := protectedHigh.Index + 1; j < n && j <= protectedHigh.Index+config.MaxBreakAge; j++ {
			c := candles[j]

			breakPips := (c.High - protectedHigh.Price) / config.PipSize
			if breakPips < config.MinBreakPips || c.Close <= protectedHigh.Price {
				continue
			}

			// VALID BREAK OF HIGH DETECTED
			// Determine break type based on prior structure (FIXED LOGIC)
			var breakType BreakType
			var newStructure StructureType

			if priorStructure == StructureBearish {
				// Breaking high in bearish structure = REVERSAL (CHoCH)
				breakType = BreakCHoCH
				newStructure = StructureBullish
			} else if priorStructure == StructureBullish {
				// Breaking high in bullish structure = CONTINUATION (BOS)
				breakType = BreakBOS
				newStructure = StructureBullish
			} else { // RANGING
				// First break from ranging = CHoCH (not BOS)
				breakType = BreakCHoCH
				newStructure = StructureBullish
			}

			brk := StructureBreak{
				Type:           breakType,
				Direction:      "BULLISH",
				BreakLevel:     protectedHigh.Price,
				BreakPrice:     c.High,
				BreakIndex:     j,
				BreakTime:      c.Timestamp,
				PriorStructure: priorStructure,
				NewStructure:   newStructure,
				Strength:       calculateBreakStrength(c, protectedHigh.Price, breakPips, config),
			}

			brk.IsRetested, brk.RetestIndex = checkBreakRetest(candles, j, protectedHigh.Price, "BULLISH", config)

			breaks = append(breaks, brk)
			priorStructure = newStructure

			// Update protected high to next swing high
			for i := len(highs) - 2; i >= 0; i-- {
				if highs[i].Index < protectedHigh.Index {
					protectedHigh = highs[i]
					break
				}
			}

			break // Only count first break of this level
		}
	}

	return breaks
}

// calculateStructureStrength determines how clean the structure is
func calculateStructureStrength(highs, lows []SwingPoint, isBullish bool) float64 {
	if len(highs) < 3 || len(lows) < 3 {
		return 0.5
	}

	strength := 0.5
	consecutiveValid := 0

	// Count consecutive structure-confirming swings
	for i := 1; i < len(highs) && i < len(lows); i++ {
		if isBullish {
			if highs[i].Price > highs[i-1].Price && lows[i].Price > lows[i-1].Price {
				consecutiveValid++
			}
		} else {
			if highs[i].Price < highs[i-1].Price && lows[i].Price < lows[i-1].Price {
				consecutiveValid++
			}
		}
	}

	strength += float64(consecutiveValid) * 0.1
	return math.Min(strength, 1.0)
}

// calculateBreakStrength determines break quality
// FIXED: Use pips instead of price percentage for close distance
func calculateBreakStrength(candle Candle, breakLevel, breakPips float64, config StructureConfig) float64 {
	strength := 0.5

	// Larger break = stronger
	breakScore := breakPips / (config.MinBreakPips * 3)
	strength += math.Min(breakScore*0.2, 0.2)

	// Close strongly beyond break level - FIXED: use pips not price %
	closeDistance := math.Abs(candle.Close - breakLevel)
	minCloseDistancePips := config.MinBreakPips * config.PipSize
	if closeDistance > minCloseDistancePips {
		strength += 0.15
	}

	// Strong candle body
	bodySize := math.Abs(candle.Open - candle.Close)
	rangeSize := candle.High - candle.Low
	if rangeSize > 0 && bodySize/rangeSize > 0.6 {
		strength += 0.15
	}

	return math.Min(strength, 1.0)
}

// checkBreakRetest checks if price returned to retest the broken level
// FIXED: Option to use ATR-based tolerance instead of price %
func checkBreakRetest(candles []Candle, breakIdx int, breakLevel float64, direction string, config StructureConfig) (bool, int) {
	// Option 1: Price-based tolerance (original)
	// tolerance := breakLevel * config.RetestTolerance

	// Option 2: ATR-based tolerance (better for multi-asset)
	var tolerance float64
	if len(candles) >= 20 && breakIdx >= 14 {
		atr := calculateATRForRetest(candles[breakIdx-14:breakIdx], 14)
		tolerance = atr * 0.3 // 30% of ATR
	} else {
		tolerance = breakLevel * config.RetestTolerance // Fallback
	}

	for i := breakIdx + 1; i < len(candles) && i <= breakIdx+10; i++ {
		c := candles[i]

		if direction == "BULLISH" {
			if c.Low <= breakLevel+tolerance && c.Low >= breakLevel-tolerance {
				if c.Close > breakLevel {
					return true, i
				}
			}
		} else {
			if c.High >= breakLevel-tolerance && c.High <= breakLevel+tolerance {
				if c.Close < breakLevel {
					return true, i
				}
			}
		}
	}

	return false, 0
}

// Add helper function
func calculateATRForRetest(candles []Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	var trSum float64
	for i := 1; i < len(candles); i++ {
		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		trSum += math.Max(highLow, math.Max(highClose, lowClose))
	}

	return trSum / float64(len(candles)-1)
}

// GetRecentBreaks returns breaks within the last N bars
func GetRecentBreaks(ms *MarketStructure, currentBar, lookback int) []StructureBreak {
	if ms == nil {
		return nil
	}

	var recent []StructureBreak
	for _, brk := range ms.RecentBreaks {
		if currentBar-brk.BreakIndex <= lookback {
			recent = append(recent, brk)
		}
	}
	return recent
}

// HasRecentBullishBOS checks for bullish BOS in recent bars
func HasRecentBullishBOS(candles []Candle, lookback int) bool {
	config := DefaultStructureConfig()
	ms := AnalyzeMarketStructure(candles, config)

	if ms == nil {
		return false
	}

	currentBar := len(candles) - 1
	for _, brk := range ms.RecentBreaks {
		if brk.Direction == "BULLISH" &&
			brk.Type == BreakBOS &&
			currentBar-brk.BreakIndex <= lookback {
			return true
		}
	}
	return false
}

// HasRecentBearishBOS checks for bearish BOS in recent bars
func HasRecentBearishBOS(candles []Candle, lookback int) bool {
	config := DefaultStructureConfig()
	ms := AnalyzeMarketStructure(candles, config)

	if ms == nil {
		return false
	}

	currentBar := len(candles) - 1
	for _, brk := range ms.RecentBreaks {
		if brk.Direction == "BEARISH" &&
			brk.Type == BreakBOS &&
			currentBar-brk.BreakIndex <= lookback {
			return true
		}
	}
	return false
}

// HasRecentCHoCH checks for change of character (reversal)
func HasRecentCHoCH(candles []Candle, lookback int, direction string) bool {
	config := DefaultStructureConfig()
	ms := AnalyzeMarketStructure(candles, config)

	if ms == nil {
		return false
	}

	currentBar := len(candles) - 1
	for _, brk := range ms.RecentBreaks {
		if brk.Type == BreakCHoCH &&
			brk.Direction == direction &&
			currentBar-brk.BreakIndex <= lookback {
			return true
		}
	}
	return false
}

// CalculateBOSScore returns confluence score (0.0-1.0) for structure alignment
func CalculateBOSScore(candles []Candle, direction string) float64 {
	return CalculateBOSScoreDebug(candles, direction, false)
}

// CalculateBOSScoreDebug with optional debug output
func CalculateBOSScoreDebug(candles []Candle, direction string, debug bool) float64 {
	if len(candles) < 20 {
		return 0.0
	}

	config := DefaultStructureConfig()
	ms := AnalyzeMarketStructure(candles, config)

	if ms == nil {
		if debug {
			fmt.Printf("      [BOS] AnalyzeMarketStructure returned nil\n")
		}
		return 0.0
	}

	if debug {
		fmt.Printf("      [BOS] Current structure: %s, Recent breaks: %d\n", ms.CurrentStructure, len(ms.RecentBreaks))
	}

	score := 0.0
	currentBar := len(candles) - 1

	// Base score from structure alignment
	if direction == DirectionLong && ms.CurrentStructure == StructureBullish {
		score += 0.5
		if debug {
			fmt.Printf("      [BOS] +0.5 for bullish structure alignment\n")
		}
	} else if direction == DirectionLong && ms.CurrentStructure == StructureRanging {
		score += 0.35
		if debug {
			fmt.Printf("      [BOS] +0.35 for ranging structure (potential breakout)\n")
		}
	}
	if direction == DirectionShort && ms.CurrentStructure == StructureBearish {
		score += 0.5
		if debug {
			fmt.Printf("      [BOS] +0.5 for bearish structure alignment\n")
		}
	} else if direction == DirectionShort && ms.CurrentStructure == StructureRanging {
		score += 0.35
		if debug {
			fmt.Printf("      [BOS] +0.35 for ranging structure (potential breakout)\n")
		}
	}

	// Bonus for recent BOS in our direction
	for _, brk := range ms.RecentBreaks {
		barsAgo := currentBar - brk.BreakIndex
		if barsAgo > 15 {
			continue
		}

		if debug {
			fmt.Printf("      [BOS] Break: direction=%s, type=%s, barsAgo=%d, strength=%.2f\n",
				brk.Direction, brk.Type, barsAgo, brk.Strength)
		}

		if direction == DirectionLong && brk.Direction == "BULLISH" {
			bosScore := brk.Strength * 0.4
			if brk.IsRetested {
				bosScore *= 1.2
			}
			recencyBonus := 1.0 - (float64(barsAgo) / 15.0)
			bosScore *= (1.0 + recencyBonus*0.2)
			score += bosScore
		}

		if direction == DirectionShort && brk.Direction == "BEARISH" {
			bosScore := brk.Strength * 0.4
			if brk.IsRetested {
				bosScore *= 1.2
			}
			recencyBonus := 1.0 - (float64(barsAgo) / 15.0)
			bosScore *= (1.0 + recencyBonus*0.2)
			score += bosScore
		}
	}

	// Penalty for trading against structure
	if direction == DirectionLong && ms.CurrentStructure == StructureBearish {
		score -= 0.3
		if debug {
			fmt.Printf("      [BOS] -0.3 penalty for trading against bearish structure\n")
		}
	}
	if direction == DirectionShort && ms.CurrentStructure == StructureBullish {
		score -= 0.3
		if debug {
			fmt.Printf("      [BOS] -0.3 penalty for trading against bullish structure\n")
		}
	}

	if debug {
		fmt.Printf("      [BOS] Final score: %.2f\n", math.Max(0.0, math.Min(score, 1.0)))
	}

	return math.Max(0.0, math.Min(score, 1.0))
}

// CheckMultiTFStructureAlignment checks if structures align across timeframes
func CheckMultiTFStructureAlignment(htfStructure, ltfStructure StructureType, direction string) float64 {
	score := 0.0

	if direction == DirectionLong {
		if htfStructure == StructureBullish && ltfStructure == StructureBullish {
			score = 1.0
		} else if htfStructure == StructureBullish && ltfStructure == StructureRanging {
			score = 0.6
		} else if htfStructure == StructureRanging && ltfStructure == StructureBullish {
			score = 0.4
		}
	} else if direction == DirectionShort {
		if htfStructure == StructureBearish && ltfStructure == StructureBearish {
			score = 1.0
		} else if htfStructure == StructureBearish && ltfStructure == StructureRanging {
			score = 0.6
		} else if htfStructure == StructureRanging && ltfStructure == StructureBearish {
			score = 0.4
		}
	}

	return score
}
