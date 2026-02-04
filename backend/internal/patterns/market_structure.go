package patterns

import (
	"fmt"
	"math"
	"time"
)

// ================================================================================
// MARKET STRUCTURE - Break of Structure (BOS) & Change of Character (CHoCH)
// ================================================================================
// WHY: H&S in wrong structure = 35% loss rate. Only trade when H4 BOS aligns with Weekly trend.
// HOW: Track swing high/low sequences to detect structure breaks.
// BOS = continuation break, CHoCH = reversal break
// ================================================================================

// StructureType represents the current market structure
type StructureType string

const (
	StructureBullish  StructureType = "BULLISH"  // Higher highs, higher lows
	StructureBearish  StructureType = "BEARISH"  // Lower highs, lower lows
	StructureRanging  StructureType = "RANGING"  // No clear direction
)

// BreakType represents the type of structure break
type BreakType string

const (
	BreakBOS   BreakType = "BOS"   // Break of Structure (continuation)
	BreakCHoCH BreakType = "CHOCH" // Change of Character (reversal)
)

// StructureBreak represents a detected structure break
type StructureBreak struct {
	Type              BreakType
	Direction         string    // "BULLISH" or "BEARISH"
	BreakLevel        float64   // The level that was broken
	BreakPrice        float64   // Where price broke through
	BreakIndex        int
	BreakTime         time.Time
	PriorStructure    StructureType
	NewStructure      StructureType
	Strength          float64   // 0.0-1.0
	IsRetested        bool      // Did price return to retest the break?
	RetestIndex       int
}

// MarketStructure holds the current structure state
type MarketStructure struct {
	CurrentStructure  StructureType
	LastSwingHigh     float64
	LastSwingHighIdx  int
	LastSwingLow      float64
	LastSwingLowIdx   int
	PreviousHigher    bool // Was last swing high higher than before?
	PreviousLower     bool // Was last swing low lower than before?
	RecentBreaks      []StructureBreak
	Strength          float64 // How strong is the current structure
}

// StructureConfig holds detection parameters
type StructureConfig struct {
	SwingLookback      int     // Bars to confirm swing
	MinBreakPips       float64 // Minimum break distance
	RetestTolerance    float64 // How close price must be for retest
	MaxBreakAge        int     // How recent must break be to count
	PipSize            float64
}

// DefaultStructureConfig returns sensible defaults
func DefaultStructureConfig() StructureConfig {
	return StructureConfig{
		SwingLookback:   3,
		MinBreakPips:    5,
		RetestTolerance: 0.002, // 0.2%
		MaxBreakAge:     20,
		PipSize:         0.0001,
	}
}

// AnalyzeMarketStructure determines the current market structure and any breaks
func AnalyzeMarketStructure(candles []Candle, config StructureConfig) *MarketStructure {
	if len(candles) < config.SwingLookback*4 {
		return nil
	}

	ms := &MarketStructure{
		CurrentStructure: StructureRanging,
	}

	// Find swing points
	highs := FindSwingHighs(candles, config.SwingLookback)
	lows := FindSwingLows(candles, config.SwingLookback)

	if len(highs) < 2 || len(lows) < 2 {
		return ms
	}

	// Get the two most recent swing highs and lows
	lastHigh := highs[len(highs)-1]
	prevHigh := highs[len(highs)-2]
	lastLow := lows[len(lows)-1]
	prevLow := lows[len(lows)-2]

	ms.LastSwingHigh = lastHigh.Price
	ms.LastSwingHighIdx = lastHigh.Index
	ms.LastSwingLow = lastLow.Price
	ms.LastSwingLowIdx = lastLow.Index
	ms.PreviousHigher = lastHigh.Price > prevHigh.Price
	ms.PreviousLower = lastLow.Price < prevLow.Price

	// Determine structure
	if lastHigh.Price > prevHigh.Price && lastLow.Price > prevLow.Price {
		ms.CurrentStructure = StructureBullish
		ms.Strength = calculateStructureStrength(highs, lows, true)
	} else if lastHigh.Price < prevHigh.Price && lastLow.Price < prevLow.Price {
		ms.CurrentStructure = StructureBearish
		ms.Strength = calculateStructureStrength(highs, lows, false)
	} else {
		ms.CurrentStructure = StructureRanging
		ms.Strength = 0.3 // Weak ranging structure
	}

	// Detect structure breaks
	ms.RecentBreaks = detectStructureBreaks(candles, highs, lows, config)

	return ms
}

// detectStructureBreaks finds BOS and CHoCH events
func detectStructureBreaks(candles []Candle, highs, lows []SwingPoint, config StructureConfig) []StructureBreak {
	var breaks []StructureBreak
	n := len(candles)

	if len(highs) < 2 || len(lows) < 2 {
		return breaks
	}

	// Track structure state as we iterate
	priorStructure := StructureRanging
	
	// Determine initial structure from first swings
	if len(highs) >= 2 && len(lows) >= 2 {
		if highs[1].Price > highs[0].Price && lows[1].Price > lows[0].Price {
			priorStructure = StructureBullish
		} else if highs[1].Price < highs[0].Price && lows[1].Price < lows[0].Price {
			priorStructure = StructureBearish
		}
	}

	// Look for breaks of swing lows in bullish structure (bearish BOS/CHoCH)
	for i := 2; i < len(lows); i++ {
		swingLow := lows[i-1]
		
		// Find the candle that broke this low
		for j := swingLow.Index + 1; j < n && j <= swingLow.Index+config.MaxBreakAge; j++ {
			c := candles[j]
			
			breakPips := (swingLow.Price - c.Low) / config.PipSize
			if breakPips >= config.MinBreakPips && c.Close < swingLow.Price {
				// This is a break of the swing low
				breakType := BreakBOS
				newStructure := StructureBearish

				if priorStructure == StructureBullish {
					breakType = BreakCHoCH // Reversal from bullish to bearish
				}

				brk := StructureBreak{
					Type:           breakType,
					Direction:      "BEARISH",
					BreakLevel:     swingLow.Price,
					BreakPrice:     c.Low,
					BreakIndex:     j,
					BreakTime:      c.Timestamp,
					PriorStructure: priorStructure,
					NewStructure:   newStructure,
					Strength:       calculateBreakStrength(c, swingLow.Price, breakPips, config),
				}

				// Check for retest
				brk.IsRetested, brk.RetestIndex = checkBreakRetest(candles, j, swingLow.Price, "BEARISH", config)

				breaks = append(breaks, brk)
				priorStructure = newStructure
				break // Only count first break of this level
			}
		}
	}

	// Look for breaks of swing highs in bearish structure (bullish BOS/CHoCH)
	for i := 2; i < len(highs); i++ {
		swingHigh := highs[i-1]
		
		for j := swingHigh.Index + 1; j < n && j <= swingHigh.Index+config.MaxBreakAge; j++ {
			c := candles[j]
			
			breakPips := (c.High - swingHigh.Price) / config.PipSize
			if breakPips >= config.MinBreakPips && c.Close > swingHigh.Price {
				breakType := BreakBOS
				newStructure := StructureBullish

				if priorStructure == StructureBearish {
					breakType = BreakCHoCH // Reversal from bearish to bullish
				}

				brk := StructureBreak{
					Type:           breakType,
					Direction:      "BULLISH",
					BreakLevel:     swingHigh.Price,
					BreakPrice:     c.High,
					BreakIndex:     j,
					BreakTime:      c.Timestamp,
					PriorStructure: priorStructure,
					NewStructure:   newStructure,
					Strength:       calculateBreakStrength(c, swingHigh.Price, breakPips, config),
				}

				brk.IsRetested, brk.RetestIndex = checkBreakRetest(candles, j, swingHigh.Price, "BULLISH", config)

				breaks = append(breaks, brk)
				priorStructure = newStructure
				break
			}
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

	// More consecutive = stronger structure
	strength += float64(consecutiveValid) * 0.1

	return math.Min(strength, 1.0)
}

// calculateBreakStrength determines break quality
func calculateBreakStrength(candle Candle, breakLevel, breakPips float64, config StructureConfig) float64 {
	strength := 0.5

	// Larger break = stronger
	breakScore := breakPips / (config.MinBreakPips * 3)
	strength += math.Min(breakScore*0.2, 0.2)

	// Close strongly beyond break level
	closeDistance := math.Abs(candle.Close - breakLevel)
	if closeDistance > breakLevel*0.001 { // 0.1% beyond
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
func checkBreakRetest(candles []Candle, breakIdx int, breakLevel float64, direction string, config StructureConfig) (bool, int) {
	tolerance := breakLevel * config.RetestTolerance

	for i := breakIdx + 1; i < len(candles) && i <= breakIdx+10; i++ {
		c := candles[i]

		if direction == "BULLISH" {
			// Retest = price pulls back to the broken high (now support)
			if c.Low <= breakLevel+tolerance && c.Low >= breakLevel-tolerance {
				// Retest confirmed if it holds above
				if c.Close > breakLevel {
					return true, i
				}
			}
		} else {
			// Retest = price pulls back to the broken low (now resistance)
			if c.High >= breakLevel-tolerance && c.High <= breakLevel+tolerance {
				if c.Close < breakLevel {
					return true, i
				}
			}
		}
	}

	return false, 0
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
		score += 0.5 // Boosted from 0.4
		if debug {
			fmt.Printf("      [BOS] +0.5 for bullish structure alignment\n")
		}
	} else if direction == DirectionLong && ms.CurrentStructure == StructureRanging {
		score += 0.35 // Boosted from 0.2 - pattern might be forming breakout
		if debug {
			fmt.Printf("      [BOS] +0.35 for ranging structure (potential breakout)\n")
		}
	}
	if direction == DirectionShort && ms.CurrentStructure == StructureBearish {
		score += 0.5 // Boosted from 0.4
		if debug {
			fmt.Printf("      [BOS] +0.5 for bearish structure alignment\n")
		}
	} else if direction == DirectionShort && ms.CurrentStructure == StructureRanging {
		score += 0.35 // Boosted from 0.2
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
			fmt.Printf("      [BOS] Break: direction=%s, barsAgo=%d, strength=%.2f\n", 
				brk.Direction, barsAgo, brk.Strength)
		}

		if direction == DirectionLong && brk.Direction == "BULLISH" {
			bosScore := brk.Strength * 0.4
			if brk.IsRetested {
				bosScore *= 1.2 // Retested breaks are stronger
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

	// Perfect alignment
	if direction == DirectionLong {
		if htfStructure == StructureBullish && ltfStructure == StructureBullish {
			score = 1.0
		} else if htfStructure == StructureBullish && ltfStructure == StructureRanging {
			score = 0.6 // HTF bullish, LTF ranging = okay
		} else if htfStructure == StructureRanging && ltfStructure == StructureBullish {
			score = 0.4 // Less confident
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
