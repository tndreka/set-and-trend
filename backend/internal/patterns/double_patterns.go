package patterns

import (
	"math"
)

// DetectDoubleTop finds double top patterns (bearish reversal)
// Two peaks at similar price levels with a trough between them
func DetectDoubleTop(candles []Candle, lookback int) *DetectedStructure {
	if len(candles) < lookback {
		return nil
	}

	// Find swing highs in recent candles
	highs := findSwingHighsInRange(candles, lookback, 5)
	if len(highs) < 2 {
		return nil
	}

	// Get two most recent significant highs
	top1 := highs[0]
	top2 := highs[1]

	// Ensure top1 is more recent than top2
	if top1.Index < top2.Index {
		top1, top2 = top2, top1
	}

	// Check tops are within 0.5% price tolerance (similar height)
	priceTolerance := 0.005
	if math.Abs(top1.Price-top2.Price)/top1.Price > priceTolerance {
		return nil
	}

	// Minimum separation between tops (at least 10 candles)
	if math.Abs(float64(top1.Index-top2.Index)) < 10 {
		return nil
	}

	// Find the trough (neckline) between the two tops
	troughIdx, troughPrice := findLowestBetween(candles, top2.Index, top1.Index)
	if troughIdx == -1 {
		return nil
	}

	// Validate pattern: trough should be at least 1% below tops
	avgTopPrice := (top1.Price + top2.Price) / 2
	minDepth := 0.01 // 1% minimum depth
	if (avgTopPrice-troughPrice)/avgTopPrice < minDepth {
		return nil
	}

	// Calculate neckline (horizontal at trough level)
	neckline := troughPrice

	// Pattern is valid
	return &DetectedStructure{
		PatternType:         "DOUBLE_TOP",
		LeftShoulderPrice:   top2.Price,
		HeadPrice:           avgTopPrice, // Use average of both tops
		RightShoulderPrice:  top1.Price,
		NecklinePrice:       neckline,
		LeftShoulderIdx:     top2.Index,
		HeadIdx:             (top1.Index + top2.Index) / 2,
		RightShoulderIdx:    top1.Index,
		NecklineIdx:         troughIdx,
		ShoulderSymmetry:    1.0,
		HeadProminence:      0.03,
		OverallConfidence:   0.85,
	}
}

// DetectDoubleBottom finds double bottom patterns (bullish reversal)
// Two troughs at similar price levels with a peak between them
func DetectDoubleBottom(candles []Candle, lookback int) *DetectedStructure {
	if len(candles) < lookback {
		return nil
	}

	// Find swing lows in recent candles
	lows := findSwingLowsInRange(candles, lookback, 5)
	if len(lows) < 2 {
		return nil
	}

	// Get two most recent significant lows
	bottom1 := lows[0]
	bottom2 := lows[1]

	// Ensure bottom1 is more recent than bottom2
	if bottom1.Index < bottom2.Index {
		bottom1, bottom2 = bottom2, bottom1
	}

	// Check bottoms are within 0.5% price tolerance (similar depth)
	priceTolerance := 0.005
	if math.Abs(bottom1.Price-bottom2.Price)/bottom1.Price > priceTolerance {
		return nil
	}

	// Minimum separation between bottoms (at least 10 candles)
	if math.Abs(float64(bottom1.Index-bottom2.Index)) < 10 {
		return nil
	}

	// Find the peak (neckline) between the two bottoms
	peakIdx, peakPrice := findHighestBetween(candles, bottom2.Index, bottom1.Index)
	if peakIdx == -1 {
		return nil
	}

	// Validate pattern: peak should be at least 1% above bottoms
	avgBottomPrice := (bottom1.Price + bottom2.Price) / 2
	minHeight := 0.01 // 1% minimum height
	if (peakPrice-avgBottomPrice)/avgBottomPrice < minHeight {
		return nil
	}

	// Calculate neckline (horizontal at peak level)
	neckline := peakPrice

	// Pattern is valid
	return &DetectedStructure{
		PatternType:         "DOUBLE_BOTTOM",
		LeftShoulderPrice:   bottom2.Price,
		HeadPrice:           avgBottomPrice, // Use average of both bottoms
		RightShoulderPrice:  bottom1.Price,
		NecklinePrice:       neckline,
		LeftShoulderIdx:     bottom2.Index,
		HeadIdx:             (bottom1.Index + bottom2.Index) / 2,
		RightShoulderIdx:    bottom1.Index,
		NecklineIdx:         peakIdx,
		ShoulderSymmetry:    1.0,
		HeadProminence:      0.03,
		OverallConfidence:   0.85,
	}
}

// doublePatternSwing is a local swing point type for double pattern detection
type doublePatternSwing struct {
	Index int
	Price float64
}

// findSwingHighsInRange finds swing highs within a range
func findSwingHighsInRange(candles []Candle, lookback, windowSize int) []doublePatternSwing {
	var highs []doublePatternSwing
	
	startIdx := len(candles) - lookback
	if startIdx < windowSize {
		startIdx = windowSize
	}

	for i := startIdx; i < len(candles)-windowSize; i++ {
		isHigh := true
		currentHigh := candles[i].High

		// Check if this is a local maximum
		for j := i - windowSize; j <= i+windowSize; j++ {
			if j != i && j >= 0 && j < len(candles) {
				if candles[j].High >= currentHigh {
					isHigh = false
					break
				}
			}
		}

		if isHigh {
			highs = append(highs, doublePatternSwing{Index: i, Price: currentHigh})
		}
	}

	// Sort by recency (most recent first)
	for i := 0; i < len(highs)/2; i++ {
		highs[i], highs[len(highs)-1-i] = highs[len(highs)-1-i], highs[i]
	}

	return highs
}

// findSwingLowsInRange finds swing lows within a range
func findSwingLowsInRange(candles []Candle, lookback, windowSize int) []doublePatternSwing {
	var lows []doublePatternSwing
	
	startIdx := len(candles) - lookback
	if startIdx < windowSize {
		startIdx = windowSize
	}

	for i := startIdx; i < len(candles)-windowSize; i++ {
		isLow := true
		currentLow := candles[i].Low

		// Check if this is a local minimum
		for j := i - windowSize; j <= i+windowSize; j++ {
			if j != i && j >= 0 && j < len(candles) {
				if candles[j].Low <= currentLow {
					isLow = false
					break
				}
			}
		}

		if isLow {
			lows = append(lows, doublePatternSwing{Index: i, Price: currentLow})
		}
	}

	// Sort by recency (most recent first)
	for i := 0; i < len(lows)/2; i++ {
		lows[i], lows[len(lows)-1-i] = lows[len(lows)-1-i], lows[i]
	}

	return lows
}

// findLowestBetween finds the lowest price between two indices
func findLowestBetween(candles []Candle, startIdx, endIdx int) (int, float64) {
	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
	}
	
	if startIdx < 0 || endIdx >= len(candles) {
		return -1, 0
	}

	lowestIdx := startIdx
	lowestPrice := candles[startIdx].Low

	for i := startIdx + 1; i <= endIdx; i++ {
		if candles[i].Low < lowestPrice {
			lowestPrice = candles[i].Low
			lowestIdx = i
		}
	}

	return lowestIdx, lowestPrice
}

// findHighestBetween finds the highest price between two indices
func findHighestBetween(candles []Candle, startIdx, endIdx int) (int, float64) {
	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
	}
	
	if startIdx < 0 || endIdx >= len(candles) {
		return -1, 0
	}

	highestIdx := startIdx
	highestPrice := candles[startIdx].High

	for i := startIdx + 1; i <= endIdx; i++ {
		if candles[i].High > highestPrice {
			highestPrice = candles[i].High
			highestIdx = i
		}
	}

	return highestIdx, highestPrice
}
