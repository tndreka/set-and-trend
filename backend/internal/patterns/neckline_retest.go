package patterns

import (
	"math"
	"time"
)

// ================================================================================
// NECKLINE RETEST - High Probability Entry Confirmation
// ================================================================================
// WHY: H&S breakout → wait for pullback = 65% win rate vs 52% on first break.
// HOW: After neckline break, wait for price to retest and reject the level.
// ================================================================================

// RetestType indicates the retest status
type RetestType string

const (
	RetestPending   RetestType = "PENDING"   // Waiting for retest
	RetestInProgress RetestType = "IN_PROGRESS" // Price at neckline
	RetestConfirmed RetestType = "CONFIRMED" // Retest complete, rejected
	RetestFailed    RetestType = "FAILED"    // Price broke back through
	RetestMissed    RetestType = "MISSED"    // Price continued without retest
)

// NecklineRetest represents a neckline retest event
type NecklineRetest struct {
	PatternType     string     // Original pattern (H&S, IHS, etc.)
	Direction       string     // Trade direction
	NecklineLevel   float64    // The neckline price
	BreakPrice      float64    // Where the break occurred
	BreakIndex      int        // Candle that broke the neckline
	BreakTime       time.Time
	RetestPrice     float64    // Where price retested
	RetestIndex     int        // Candle that retested
	RetestTime      time.Time
	Status          RetestType
	RejectionStrength float64  // How strong was the rejection (0.0-1.0)
	Confidence      float64    // Overall retest quality
}

// RetestConfig holds detection parameters
type RetestConfig struct {
	MaxBarsToRetest    int     // How long to wait for retest
	RetestTolerance    float64 // How close price must get (as % of neckline)
	MinRejectionPips   float64 // Minimum rejection distance
	MaxRetestBreakPips float64 // Max penetration allowed (else it's a fail)
	PipSize            float64
}

// DefaultRetestConfig returns sensible defaults
func DefaultRetestConfig() RetestConfig {
	return RetestConfig{
		MaxBarsToRetest:    10,
		RetestTolerance:    0.002, // 0.2%
		MinRejectionPips:   10,
		MaxRetestBreakPips: 15, // Can penetrate 15 pips max
		PipSize:            0.0001,
	}
}

// DetectNecklineRetest checks if a pattern has a valid neckline retest
func DetectNecklineRetest(structure *DetectedStructure, candles []Candle, breakIdx int, config RetestConfig) *NecklineRetest {
	if structure == nil || len(candles) <= breakIdx {
		return nil
	}

	neckline := structure.NecklinePrice
	direction := DirectionShort
	if structure.PatternType == PatternIHS || structure.PatternType == "TRIPLE_BOTTOM" || structure.PatternType == "DOUBLE_BOTTOM" {
		direction = DirectionLong
	}

	retest := &NecklineRetest{
		PatternType:   structure.PatternType,
		Direction:     direction,
		NecklineLevel: neckline,
		BreakPrice:    candles[breakIdx].Close,
		BreakIndex:    breakIdx,
		BreakTime:     candles[breakIdx].Timestamp,
		Status:        RetestPending,
	}

	// Look for retest in subsequent candles
	for i := breakIdx + 1; i < len(candles) && i <= breakIdx+config.MaxBarsToRetest; i++ {
		c := candles[i]

		if direction == DirectionShort {
			// For bearish patterns: retest = price rallies back UP to neckline
			retestCheck := detectBearishRetest(c, neckline, config)
			if retestCheck.Status != RetestPending {
				retest.RetestPrice = retestCheck.RetestPrice
				retest.RetestIndex = i
				retest.RetestTime = c.Timestamp
				retest.Status = retestCheck.Status
				retest.RejectionStrength = retestCheck.RejectionStrength
				break
			}
		} else {
			// For bullish patterns: retest = price pulls back DOWN to neckline
			retestCheck := detectBullishRetest(c, neckline, config)
			if retestCheck.Status != RetestPending {
				retest.RetestPrice = retestCheck.RetestPrice
				retest.RetestIndex = i
				retest.RetestTime = c.Timestamp
				retest.Status = retestCheck.Status
				retest.RejectionStrength = retestCheck.RejectionStrength
				break
			}
		}
	}

	// Check for missed retest (price just continued without coming back)
	if retest.Status == RetestPending && len(candles) > breakIdx+config.MaxBarsToRetest {
		// Check if price moved away significantly without retest
		lastCandle := candles[len(candles)-1]
		
		if direction == DirectionShort {
			// Price kept going down without coming back
			if lastCandle.Close < neckline-(50*config.PipSize) {
				retest.Status = RetestMissed
			}
		} else {
			if lastCandle.Close > neckline+(50*config.PipSize) {
				retest.Status = RetestMissed
			}
		}
	}

	// Calculate overall confidence
	retest.Confidence = calculateRetestConfidence(retest, config)

	return retest
}

// detectBearishRetest checks for bearish pattern retest (price comes up to neckline, rejects)
func detectBearishRetest(c Candle, neckline float64, config RetestConfig) *NecklineRetest {
	result := &NecklineRetest{Status: RetestPending}
	
	tolerance := neckline * config.RetestTolerance
	maxPenetration := config.MaxRetestBreakPips * config.PipSize

	// Check if high reached the neckline area
	if c.High >= neckline-tolerance && c.High <= neckline+maxPenetration {
		result.RetestPrice = c.High

		// Check for rejection (close well below high)
		rejectionPips := (c.High - c.Close) / config.PipSize
		
		if c.Close > neckline {
			// Closed above neckline = failed retest
			result.Status = RetestFailed
			result.RejectionStrength = 0.0
		} else if rejectionPips >= config.MinRejectionPips {
			// Strong rejection below neckline = confirmed
			result.Status = RetestConfirmed
			result.RejectionStrength = math.Min(rejectionPips/30, 1.0) // Normalize
		} else if c.High >= neckline-tolerance {
			// Touched but weak rejection = in progress
			result.Status = RetestInProgress
			result.RejectionStrength = rejectionPips / config.MinRejectionPips
		}
	}

	return result
}

// detectBullishRetest checks for bullish pattern retest (price comes down to neckline, rejects)
func detectBullishRetest(c Candle, neckline float64, config RetestConfig) *NecklineRetest {
	result := &NecklineRetest{Status: RetestPending}
	
	tolerance := neckline * config.RetestTolerance
	maxPenetration := config.MaxRetestBreakPips * config.PipSize

	// Check if low reached the neckline area
	if c.Low <= neckline+tolerance && c.Low >= neckline-maxPenetration {
		result.RetestPrice = c.Low

		// Check for rejection (close well above low)
		rejectionPips := (c.Close - c.Low) / config.PipSize
		
		if c.Close < neckline {
			// Closed below neckline = failed retest
			result.Status = RetestFailed
			result.RejectionStrength = 0.0
		} else if rejectionPips >= config.MinRejectionPips {
			// Strong rejection above neckline = confirmed
			result.Status = RetestConfirmed
			result.RejectionStrength = math.Min(rejectionPips/30, 1.0)
		} else if c.Low <= neckline+tolerance {
			// Touched but weak rejection = in progress
			result.Status = RetestInProgress
			result.RejectionStrength = rejectionPips / config.MinRejectionPips
		}
	}

	return result
}

// calculateRetestConfidence determines overall retest quality
func calculateRetestConfidence(retest *NecklineRetest, config RetestConfig) float64 {
	if retest.Status == RetestFailed || retest.Status == RetestMissed {
		return 0.0
	}

	if retest.Status == RetestPending {
		return 0.3 // Waiting is okay but not ideal
	}

	confidence := 0.5

	// Confirmed retest = base bonus
	if retest.Status == RetestConfirmed {
		confidence += 0.2
	}

	// Strong rejection = bonus
	confidence += retest.RejectionStrength * 0.2

	// Quick retest = bonus (didn't take too long)
	barsToRetest := retest.RetestIndex - retest.BreakIndex
	if barsToRetest <= 3 {
		confidence += 0.1
	} else if barsToRetest <= 5 {
		confidence += 0.05
	}

	return math.Min(confidence, 1.0)
}

// CalculateRetestScore returns confluence score (0.0-1.0) for retest alignment
func CalculateRetestScore(structure *DetectedStructure, candles []Candle) float64 {
	if structure == nil || len(candles) < 10 {
		return 0.0
	}

	// Find approximate break index (where neckline was crossed)
	breakIdx := findNecklineBreakIndex(structure, candles)
	if breakIdx < 0 {
		return 0.0
	}

	config := DefaultRetestConfig()
	retest := DetectNecklineRetest(structure, candles, breakIdx, config)

	if retest == nil {
		return 0.0
	}

	return retest.Confidence
}

// findNecklineBreakIndex finds where the neckline was broken
func findNecklineBreakIndex(structure *DetectedStructure, candles []Candle) int {
	if structure == nil {
		return -1
	}

	neckline := structure.NecklinePrice
	rightShoulderIdx := structure.RightShoulderIdx

	// Look after right shoulder for the break
	for i := rightShoulderIdx + 1; i < len(candles); i++ {
		c := candles[i]

		if structure.PatternType == PatternHS || structure.PatternType == "TRIPLE_TOP" || structure.PatternType == "DOUBLE_TOP" {
			// Bearish: break is close below neckline
			if c.Close < neckline {
				return i
			}
		} else {
			// Bullish: break is close above neckline
			if c.Close > neckline {
				return i
			}
		}
	}

	return -1
}

// HasConfirmedRetest checks if pattern has a confirmed retest
func HasConfirmedRetest(structure *DetectedStructure, candles []Candle) bool {
	if structure == nil {
		return false
	}

	breakIdx := findNecklineBreakIndex(structure, candles)
	if breakIdx < 0 {
		return false
	}

	config := DefaultRetestConfig()
	retest := DetectNecklineRetest(structure, candles, breakIdx, config)

	return retest != nil && retest.Status == RetestConfirmed
}

// GetRetestEntry calculates optimal entry after confirmed retest
func GetRetestEntry(structure *DetectedStructure, retest *NecklineRetest, pipSize float64) float64 {
	if structure == nil || retest == nil {
		return 0.0
	}

	if retest.Status != RetestConfirmed {
		return 0.0 // Only enter on confirmed retest
	}

	// Entry is slightly beyond the retest close
	if retest.Direction == DirectionShort {
		// Enter below neckline after rejection
		return structure.NecklinePrice - (5 * pipSize)
	} else {
		// Enter above neckline after rejection
		return structure.NecklinePrice + (5 * pipSize)
	}
}

// RetestTradeSignal generates a signal specifically for retest entry
type RetestTradeSignal struct {
	TradeSignal
	RetestConfidence float64
	BarsAfterBreak   int
	RetestType       RetestType
}

// GenerateRetestSignal creates a trade signal based on confirmed retest
func GenerateRetestSignal(structure *DetectedStructure, retest *NecklineRetest, symbol string, pipSize float64) *RetestTradeSignal {
	if structure == nil || retest == nil || retest.Status != RetestConfirmed {
		return nil
	}

	signal := &RetestTradeSignal{
		RetestConfidence: retest.Confidence,
		BarsAfterBreak:   retest.RetestIndex - retest.BreakIndex,
		RetestType:       retest.Status,
	}

	// Calculate entry/SL/TP
	if retest.Direction == DirectionShort {
		signal.Direction = DirectionShort
		signal.EntryPrice = structure.NecklinePrice - (5 * pipSize)
		signal.StopLoss = retest.RetestPrice + (15 * pipSize)
		
		patternHeight := structure.HeadPrice - structure.NecklinePrice
		signal.TakeProfit = structure.NecklinePrice - patternHeight
	} else {
		signal.Direction = DirectionLong
		signal.EntryPrice = structure.NecklinePrice + (5 * pipSize)
		signal.StopLoss = retest.RetestPrice - (15 * pipSize)
		
		patternHeight := structure.NecklinePrice - structure.HeadPrice
		signal.TakeProfit = structure.NecklinePrice + patternHeight
	}

	signal.Symbol = symbol
	signal.PatternType = structure.PatternType + "_RETEST"

	// Calculate R:R
	risk := math.Abs(signal.EntryPrice - signal.StopLoss)
	reward := math.Abs(signal.TakeProfit - signal.EntryPrice)
	if risk > 0 {
		signal.RiskReward = reward / risk
	}

	// Higher confidence for retest entries
	signal.Confidence = structure.FinalConfidence * 1.15 // 15% bonus for retest
	if signal.Confidence > 1.0 {
		signal.Confidence = 1.0
	}

	return signal
}
