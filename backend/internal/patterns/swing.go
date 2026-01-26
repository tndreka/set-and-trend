package patterns

import "math"

// FindSwingHighs detects local price maxima (pivot highs)
// lookback = bars to examine left and right
// A swing high is confirmed when:
// - current.High > all bars in [i-lookback, i-1]
// - current.High >= all bars in [i+1, i+lookback]
func FindSwingHighs(candles []Candle, lookback int) []SwingPoint {
	if len(candles) < lookback*2+1 {
		return nil
	}

	var swings []SwingPoint

	for i := lookback; i < len(candles)-lookback; i++ {
		current := candles[i]
		isSwingHigh := true
		maxLeftHigh := 0.0
		maxRightHigh := 0.0

		// Check bars to the left (must be strictly higher)
		for j := i - lookback; j < i; j++ {
			if candles[j].High >= current.High {
				isSwingHigh = false
				break
			}
			if candles[j].High > maxLeftHigh {
				maxLeftHigh = candles[j].High
			}
		}

		if !isSwingHigh {
			continue
		}

		// Check bars to the right (can be equal or lower)
		for j := i + 1; j <= i+lookback; j++ {
			if candles[j].High > current.High {
				isSwingHigh = false
				break
			}
			if candles[j].High > maxRightHigh {
				maxRightHigh = candles[j].High
			}
		}

		if isSwingHigh {
			// Calculate strength (how much higher than neighbors?)
			avgNeighbor := (maxLeftHigh + maxRightHigh) / 2.0
			strength := 0.0
			if avgNeighbor > 0 {
				strength = (current.High - avgNeighbor) / avgNeighbor
				strength = math.Min(strength, 1.0)
			}

			swings = append(swings, SwingPoint{
				Index:     i,
				Price:     current.High,
				IsPeak:    true,
				Strength:  strength,
				Timestamp: current.Timestamp,
			})
		}
	}

	return swings
}

// FindSwingLows detects local price minima (pivot lows)
// Inverse logic of FindSwingHighs
func FindSwingLows(candles []Candle, lookback int) []SwingPoint {
	if len(candles) < lookback*2+1 {
		return nil
	}

	var swings []SwingPoint

	for i := lookback; i < len(candles)-lookback; i++ {
		current := candles[i]
		isSwingLow := true
		minLeftLow := math.MaxFloat64
		minRightLow := math.MaxFloat64

		// Check bars to the left (must be strictly lower)
		for j := i - lookback; j < i; j++ {
			if candles[j].Low <= current.Low {
				isSwingLow = false
				break
			}
			if candles[j].Low < minLeftLow {
				minLeftLow = candles[j].Low
			}
		}

		if !isSwingLow {
			continue
		}

		// Check bars to the right (can be equal or higher)
		for j := i + 1; j <= i+lookback; j++ {
			if candles[j].Low < current.Low {
				isSwingLow = false
				break
			}
			if candles[j].Low < minRightLow {
				minRightLow = candles[j].Low
			}
		}

		if isSwingLow {
			// Calculate strength (how much lower than neighbors?)
			avgNeighbor := (minLeftLow + minRightLow) / 2.0
			strength := 0.0
			if avgNeighbor > 0 {
				strength = (avgNeighbor - current.Low) / avgNeighbor
				strength = math.Min(strength, 1.0)
			}

			swings = append(swings, SwingPoint{
				Index:     i,
				Price:     current.Low,
				IsPeak:    false,
				Strength:  strength,
				Timestamp: current.Timestamp,
			})
		}
	}

	return swings
}

// FindMinBetweenPeaks finds the lowest low between two peak indices
func FindMinBetweenPeaks(candles []Candle, peakA, peakB int) *SwingPoint {
	if peakA > peakB {
		peakA, peakB = peakB, peakA
	}

	if peakA < 0 || peakB >= len(candles) || peakA >= peakB {
		return nil
	}

	minPrice := candles[peakA+1].Low
	minIdx := peakA + 1

	for i := peakA + 1; i < peakB; i++ {
		if candles[i].Low < minPrice {
			minPrice = candles[i].Low
			minIdx = i
		}
	}

	return &SwingPoint{
		Index:     minIdx,
		Price:     minPrice,
		IsPeak:    false,
		Timestamp: candles[minIdx].Timestamp,
	}
}

// FindMaxBetweenTroughs finds the highest high between two trough indices
// Used for inverse H&S detection
func FindMaxBetweenTroughs(candles []Candle, troughA, troughB int) *SwingPoint {
	if troughA > troughB {
		troughA, troughB = troughB, troughA
	}

	if troughA < 0 || troughB >= len(candles) || troughA >= troughB {
		return nil
	}

	maxPrice := candles[troughA+1].High
	maxIdx := troughA + 1

	for i := troughA + 1; i < troughB; i++ {
		if candles[i].High > maxPrice {
			maxPrice = candles[i].High
			maxIdx = i
		}
	}

	return &SwingPoint{
		Index:     maxIdx,
		Price:     maxPrice,
		IsPeak:    true,
		Timestamp: candles[maxIdx].Timestamp,
	}
}
