package smc

import (
	"time"

	"set-and-trend/backend/internal/database"
)

// FVG represents a Fair Value Gap (Market Imbalance Area)
// Bullish FVG: gap between candle[i-1].high and candle[i+1].low (price gap up)
// Bearish FVG: gap between candle[i-1].low and candle[i+1].high (price gap down)
type FVG struct {
	Direction int       // 1=bullish (gap up), -1=bearish (gap down)
	Top       float64   // Upper boundary of the gap
	Bottom    float64   // Lower boundary of the gap
	MidPoint  float64   // Center of the gap
	Size      float64   // Gap size as percentage
	Mitigated bool      // Whether price has returned to fill the gap
	Index     int       // Bar index where FVG was created (middle candle)
	Timestamp time.Time // Timestamp of the FVG
}

// DetectFVGs finds all Fair Value Gaps in the candle data
// An FVG occurs when there's a gap between:
// - Bullish: candle[i-1].High < candle[i+1].Low (price jumped up)
// - Bearish: candle[i-1].Low > candle[i+1].High (price jumped down)
func DetectFVGs(candles []database.Candle, minGapPercent float64) []FVG {
	if len(candles) < 3 {
		return nil
	}

	if minGapPercent <= 0 {
		minGapPercent = 0.001 // 0.1% minimum gap
	}

	var fvgs []FVG

	for i := 1; i < len(candles)-1; i++ {
		prevCandle := candles[i-1]
		currentCandle := candles[i]
		nextCandle := candles[i+1]

		// Check for Bullish FVG (gap up)
		// Gap exists when previous high is below next candle's low
		if prevCandle.High < nextCandle.Low {
			gapSize := (nextCandle.Low - prevCandle.High) / currentCandle.Close
			if gapSize >= minGapPercent {
				fvgs = append(fvgs, FVG{
					Direction: 1,
					Top:       nextCandle.Low,
					Bottom:    prevCandle.High,
					MidPoint:  (nextCandle.Low + prevCandle.High) / 2,
					Size:      gapSize,
					Mitigated: false,
					Index:     i,
					Timestamp: currentCandle.Timestamp,
				})
			}
		}

		// Check for Bearish FVG (gap down)
		// Gap exists when previous low is above next candle's high
		if prevCandle.Low > nextCandle.High {
			gapSize := (prevCandle.Low - nextCandle.High) / currentCandle.Close
			if gapSize >= minGapPercent {
				fvgs = append(fvgs, FVG{
					Direction: -1,
					Top:       prevCandle.Low,
					Bottom:    nextCandle.High,
					MidPoint:  (prevCandle.Low + nextCandle.High) / 2,
					Size:      gapSize,
					Mitigated: false,
					Index:     i,
					Timestamp: currentCandle.Timestamp,
				})
			}
		}
	}

	return fvgs
}

// UpdateFVGMitigation checks if FVGs have been mitigated by price action
// A FVG is mitigated when price returns to trade through the gap
func UpdateFVGMitigation(fvgs []FVG, candles []database.Candle, fromIndex int) []FVG {
	for i := range fvgs {
		if fvgs[i].Mitigated {
			continue
		}

		// Check candles after the FVG was created
		startIdx := fvgs[i].Index + 2
		if startIdx < fromIndex {
			startIdx = fromIndex
		}

		for j := startIdx; j < len(candles); j++ {
			// Bullish FVG mitigated when price drops into it
			if fvgs[i].Direction == 1 && candles[j].Low <= fvgs[i].MidPoint {
				fvgs[i].Mitigated = true
				break
			}
			// Bearish FVG mitigated when price rises into it
			if fvgs[i].Direction == -1 && candles[j].High >= fvgs[i].MidPoint {
				fvgs[i].Mitigated = true
				break
			}
		}
	}
	return fvgs
}

// GetActiveFVGs returns FVGs that haven't been mitigated
func GetActiveFVGs(fvgs []FVG) []FVG {
	var active []FVG
	for _, fvg := range fvgs {
		if !fvg.Mitigated {
			active = append(active, fvg)
		}
	}
	return active
}

// IsPriceInFVG checks if a price is within any FVG zone
func IsPriceInFVG(price float64, fvgs []FVG) *FVG {
	for i := range fvgs {
		if price >= fvgs[i].Bottom && price <= fvgs[i].Top {
			return &fvgs[i]
		}
	}
	return nil
}

// IsFVGInFavor checks if the FVG direction supports the trade direction
// direction: 1=bullish/long, -1=bearish/short
func IsFVGInFavor(fvg *FVG, direction int) bool {
	if fvg == nil {
		return false
	}
	// Bullish FVG supports long trades
	// Bearish FVG supports short trades
	return fvg.Direction == direction
}

// GetNearestFVG finds the closest FVG to current price
func GetNearestFVG(price float64, fvgs []FVG, direction int) *FVG {
	var nearest *FVG
	minDistance := float64(1e10)

	for i := range fvgs {
		if fvgs[i].Mitigated {
			continue
		}
		
		// If direction specified, only consider FVGs in that direction
		if direction != 0 && fvgs[i].Direction != direction {
			continue
		}

		distance := 0.0
		if price > fvgs[i].Top {
			distance = price - fvgs[i].Top
		} else if price < fvgs[i].Bottom {
			distance = fvgs[i].Bottom - price
		}

		if distance < minDistance {
			minDistance = distance
			nearest = &fvgs[i]
		}
	}

	return nearest
}
