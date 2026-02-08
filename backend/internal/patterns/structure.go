package patterns

import "math"

// DetectHeadAndShoulders detects bearish H&S patterns
// Input: candles slice (typically 20-30 bars)
// Output: DetectedStructure if found, nil if not
func DetectHeadAndShoulders(candles []Candle, lookback int) *DetectedStructure {
	if len(candles) < 20 {
		return nil
	}

	// Step 1: Find swing highs (potential shoulders and head)
	highs := FindSwingHighs(candles, lookback)
	if len(highs) < 3 {
		return nil
	}

	// Step 2: Try to find valid 3-peak H&S sequence
	// Iterate through possible combinations
	for i := 0; i < len(highs)-2; i++ {
		ls := highs[i]   // Left Shoulder candidate
		head := highs[i+1] // Head candidate
		rs := highs[i+2]   // Right Shoulder candidate

		structure := validateHSStructure(candles, ls, head, rs)
		if structure != nil {
			structure.PatternType = PatternHS
			return structure
		}
	}

	return nil
}

// DetectInverseHeadAndShoulders detects bullish IHS patterns
func DetectInverseHeadAndShoulders(candles []Candle, lookback int) *DetectedStructure {
	if len(candles) < 20 {
		return nil
	}

	// Step 1: Find swing lows (potential shoulders and head)
	lows := FindSwingLows(candles, lookback)
	if len(lows) < 3 {
		return nil
	}

	// Step 2: Try to find valid 3-trough IHS sequence
	for i := 0; i < len(lows)-2; i++ {
		ls := lows[i]    // Left Shoulder candidate
		head := lows[i+1] // Head candidate (lowest)
		rs := lows[i+2]   // Right Shoulder candidate

		structure := validateIHSStructure(candles, ls, head, rs)
		if structure != nil {
			structure.PatternType = PatternIHS
			return structure
		}
	}

	return nil
}

// validateHSStructure validates a bearish H&S pattern
func validateHSStructure(candles []Candle, ls, head, rs SwingPoint) *DetectedStructure {
	// Rule 1: Head must be the highest
	if head.Price <= ls.Price || head.Price <= rs.Price {
		return nil
	}

	// Rule 2: Shoulder symmetry (within 20%)
	shoulderDiff := math.Abs(ls.Price-rs.Price) / math.Max(ls.Price, rs.Price)
	symmetry := 1.0 - shoulderDiff
	if symmetry < 0.80 {
		return nil
	}

	// Step 3: Find troughs (neckline points)
	trough1 := FindMinBetweenPeaks(candles, ls.Index, head.Index)
	trough2 := FindMinBetweenPeaks(candles, head.Index, rs.Index)

	if trough1 == nil || trough2 == nil {
		return nil
	}

	// Calculate neckline (average of troughs)
	necklinePrice := (trough1.Price + trough2.Price) / 2.0

	// Rule 3: Pattern must have meaningful height
	headProminence := (head.Price - necklinePrice) / head.Price
	if headProminence < 0.02 {
		return nil // Pattern too small (< 2%)
	}

	// Step 4: Calculate time symmetry (FIXED - division by zero protection)
	distLS := head.Index - ls.Index
	distRS := rs.Index - head.Index
	maxDist := max(distLS, distRS)
	var timeSymmetry float64
	if maxDist == 0 {
		timeSymmetry = 0.5 // Neutral if no distance
	} else {
		timeSymmetry = 1.0 - (math.Abs(float64(distLS-distRS)) / float64(maxDist))
	}

	// Step 5: Volume analysis
	volumeScore := calculateVolumeScore(candles, ls.Index, head.Index, rs.Index)

	// Step 6: Neckline quality (how horizontal?)
	troughDiff := math.Abs(trough1.Price-trough2.Price) / necklinePrice
	necklineQuality := 1.0 - troughDiff

	// Build structure
	structure := &DetectedStructure{
		LeftShoulderPrice:  ls.Price,
		LeftShoulderIdx:    ls.Index,
		HeadPrice:          head.Price,
		HeadIdx:            head.Index,
		RightShoulderPrice: rs.Price,
		RightShoulderIdx:   rs.Index,
		NecklinePrice:      necklinePrice,
		NecklineIdx:        (trough1.Index + trough2.Index) / 2,
		Trough1Price:       trough1.Price,
		Trough1Idx:         trough1.Index,
		Trough2Price:       trough2.Price,
		Trough2Idx:         trough2.Index,
		ShoulderSymmetry:   symmetry,
		HeadProminence:     headProminence,
		TimeSymmetry:       timeSymmetry,
		VolumeProfile:      volumeScore,
		NecklineQuality:    necklineQuality,
	}

	// Calculate overall confidence
	structure.OverallConfidence = CalculateStructureConfidence(structure)

	return structure
}


// validateIHSStructure validates a bullish Inverse H&S pattern
func validateIHSStructure(candles []Candle, ls, head, rs SwingPoint) *DetectedStructure {
	// Rule 1: Head must be the lowest
	if head.Price >= ls.Price || head.Price >= rs.Price {
		return nil
	}

	// Rule 2: Shoulder symmetry (within 20%)
	shoulderDiff := math.Abs(ls.Price-rs.Price) / math.Max(ls.Price, rs.Price)
	symmetry := 1.0 - shoulderDiff
	if symmetry < 0.80 {
		return nil
	}

	// Step 3: Find peaks (neckline points for IHS)
	peak1 := FindMaxBetweenTroughs(candles, ls.Index, head.Index)
	peak2 := FindMaxBetweenTroughs(candles, head.Index, rs.Index)

	if peak1 == nil || peak2 == nil {
		return nil
	}

	// Calculate neckline (average of peaks)
	necklinePrice := (peak1.Price + peak2.Price) / 2.0

	// Rule 3: Pattern must have meaningful height
	headProminence := (necklinePrice - head.Price) / necklinePrice
	if headProminence < 0.02 {
		return nil
	}

	// Step 4: Calculate time symmetry (FIXED - division by zero protection)
	distLS := head.Index - ls.Index
	distRS := rs.Index - head.Index
	maxDist := max(distLS, distRS)
	var timeSymmetry float64
	if maxDist == 0 {
		timeSymmetry = 0.5
	} else {
		timeSymmetry = 1.0 - (math.Abs(float64(distLS-distRS)) / float64(maxDist))
	}

	// Step 5: Volume analysis (inverse: volume should increase on RS)
	volumeScore := calculateVolumeScoreIHS(candles, ls.Index, head.Index, rs.Index)

	// Step 6: Neckline quality
	peakDiff := math.Abs(peak1.Price-peak2.Price) / necklinePrice
	necklineQuality := 1.0 - peakDiff

	structure := &DetectedStructure{
		LeftShoulderPrice:  ls.Price,
		LeftShoulderIdx:    ls.Index,
		HeadPrice:          head.Price,
		HeadIdx:            head.Index,
		RightShoulderPrice: rs.Price,
		RightShoulderIdx:   rs.Index,
		NecklinePrice:      necklinePrice,
		NecklineIdx:        (peak1.Index + peak2.Index) / 2,
		Trough1Price:       peak1.Price,
		Trough1Idx:         peak1.Index,
		Trough2Price:       peak2.Price,
		Trough2Idx:         peak2.Index,
		ShoulderSymmetry:   symmetry,
		HeadProminence:     headProminence,
		TimeSymmetry:       timeSymmetry,
		VolumeProfile:      volumeScore,
		NecklineQuality:    necklineQuality,
	}

	structure.OverallConfidence = CalculateStructureConfidence(structure)

	return structure
}

// calculateVolumeScore for bearish H&S
// Ideal: declining volume (LS > Head > RS)
func calculateVolumeScore(candles []Candle, lsIdx, headIdx, rsIdx int) float64 {
	if lsIdx >= len(candles) || headIdx >= len(candles) || rsIdx >= len(candles) {
		return 0.0
	}

	lsVol := candles[lsIdx].Volume
	headVol := candles[headIdx].Volume
	rsVol := candles[rsIdx].Volume

	// All zero volume = no signal
	if lsVol == 0 && headVol == 0 && rsVol == 0 {
		return 0.5 // Neutral
	}

	if rsVol < headVol && headVol < lsVol {
		return 0.8 // Ideal: declining volume
	} else if rsVol <= headVol {
		return 0.4 // Partial
	} else {
		return -0.2 // Negative: volume increasing (reversal weak)
	}
}

// calculateVolumeScoreIHS for bullish IHS
// Ideal: increasing volume on RS (accumulation)
func calculateVolumeScoreIHS(candles []Candle, lsIdx, headIdx, rsIdx int) float64 {
	if lsIdx >= len(candles) || headIdx >= len(candles) || rsIdx >= len(candles) {
		return 0.0
	}

	lsVol := candles[lsIdx].Volume
	headVol := candles[headIdx].Volume
	rsVol := candles[rsIdx].Volume

	if lsVol == 0 && headVol == 0 && rsVol == 0 {
		return 0.5
	}

	if rsVol > headVol {
		return 0.8 // Ideal: increasing volume on RS
	} else if rsVol >= lsVol {
		return 0.4
	} else {
		return -0.2
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
