package smc

import (
	"math"
	"time"

	"set-and-trend/backend/internal/database"
)

// LiquidityZone represents an area where stop losses accumulate
// Equal highs/lows or clusters of swing points create liquidity pools
type LiquidityZone struct {
	Direction int       // 1=buyside (above highs), -1=sellside (below lows)
	Level     float64   // The liquidity level
	Strength  int       // Number of touches/equal levels
	Swept     bool      // Whether liquidity has been taken
	SweptAt   int       // Bar index where sweep occurred
	Index     int       // Bar index where zone was identified
	Timestamp time.Time // Time of the zone creation
}

// LiquiditySweep represents a sweep event
type LiquiditySweep struct {
	Zone      *LiquidityZone
	SweepBar  int
	SweepTime time.Time
	WickOnly  bool    // True if only wicked through, false if closed through
	Magnitude float64 // How far past the level price went
}

// DetectLiquidityZones finds areas where stop losses likely accumulate
// tolerance: how close prices need to be to count as "equal"
func DetectLiquidityZones(candles []database.Candle, swingHighs, swingLows []SwingPoint, tolerance float64) []LiquidityZone {
	if len(candles) < 10 {
		return nil
	}

	if tolerance <= 0 {
		tolerance = 0.001 // 0.1% default tolerance
	}

	var zones []LiquidityZone

	// Find buyside liquidity (equal highs)
	zones = append(zones, findBuysideLiquidity(candles, swingHighs, tolerance)...)

	// Find sellside liquidity (equal lows)
	zones = append(zones, findSellsideLiquidity(candles, swingLows, tolerance)...)

	return zones
}

// findBuysideLiquidity detects equal highs and swing high clusters
func findBuysideLiquidity(candles []database.Candle, swingHighs []SwingPoint, tolerance float64) []LiquidityZone {
	var zones []LiquidityZone

	// Method 1: Equal highs from swing points
	if len(swingHighs) >= 2 {
		for i := 0; i < len(swingHighs)-1; i++ {
			for j := i + 1; j < len(swingHighs); j++ {
				h1 := swingHighs[i]
				h2 := swingHighs[j]
				
				// Check if highs are equal (within tolerance)
				avgPrice := (h1.Price + h2.Price) / 2
				diff := math.Abs(h1.Price-h2.Price) / avgPrice
				
				if diff <= tolerance {
					zones = append(zones, LiquidityZone{
						Direction: 1,
						Level:     math.Max(h1.Price, h2.Price),
						Strength:  2,
						Swept:     false,
						Index:     h2.Index,
						Timestamp: candles[h2.Index].Timestamp,
					})
				}
			}
		}
	}

	// Method 2: Candle high clusters (multiple candles hitting same level)
	for i := 5; i < len(candles); i++ {
		level := candles[i].High
		touches := 1
		
		// Look back for candles with similar highs
		for j := i - 1; j >= 0 && j >= i-20; j-- {
			diff := math.Abs(candles[j].High-level) / level
			if diff <= tolerance {
				touches++
			}
		}
		
		// 3 or more touches creates a liquidity zone
		if touches >= 3 {
			// Check if zone already exists
			exists := false
			for _, z := range zones {
				if math.Abs(z.Level-level)/level <= tolerance*2 {
					exists = true
					if touches > z.Strength {
						z.Strength = touches
					}
					break
				}
			}
			if !exists {
				zones = append(zones, LiquidityZone{
					Direction: 1,
					Level:     level,
					Strength:  touches,
					Swept:     false,
					Index:     i,
					Timestamp: candles[i].Timestamp,
				})
			}
		}
	}

	return deduplicateZones(zones, tolerance)
}

// findSellsideLiquidity detects equal lows and swing low clusters
func findSellsideLiquidity(candles []database.Candle, swingLows []SwingPoint, tolerance float64) []LiquidityZone {
	var zones []LiquidityZone

	// Method 1: Equal lows from swing points
	if len(swingLows) >= 2 {
		for i := 0; i < len(swingLows)-1; i++ {
			for j := i + 1; j < len(swingLows); j++ {
				l1 := swingLows[i]
				l2 := swingLows[j]
				
				avgPrice := (l1.Price + l2.Price) / 2
				diff := math.Abs(l1.Price-l2.Price) / avgPrice
				
				if diff <= tolerance {
					zones = append(zones, LiquidityZone{
						Direction: -1,
						Level:     math.Min(l1.Price, l2.Price),
						Strength:  2,
						Swept:     false,
						Index:     l2.Index,
						Timestamp: candles[l2.Index].Timestamp,
					})
				}
			}
		}
	}

	// Method 2: Candle low clusters
	for i := 5; i < len(candles); i++ {
		level := candles[i].Low
		touches := 1
		
		for j := i - 1; j >= 0 && j >= i-20; j-- {
			diff := math.Abs(candles[j].Low-level) / level
			if diff <= tolerance {
				touches++
			}
		}
		
		if touches >= 3 {
			exists := false
			for _, z := range zones {
				if math.Abs(z.Level-level)/level <= tolerance*2 {
					exists = true
					if touches > z.Strength {
						z.Strength = touches
					}
					break
				}
			}
			if !exists {
				zones = append(zones, LiquidityZone{
					Direction: -1,
					Level:     level,
					Strength:  touches,
					Swept:     false,
					Index:     i,
					Timestamp: candles[i].Timestamp,
				})
			}
		}
	}

	return deduplicateZones(zones, tolerance)
}

// UpdateLiquiditySweeps checks which zones have been swept
func UpdateLiquiditySweeps(zones []LiquidityZone, candles []database.Candle, fromIndex int) []LiquidityZone {
	for i := range zones {
		if zones[i].Swept {
			continue
		}

		startIdx := zones[i].Index + 1
		if startIdx < fromIndex {
			startIdx = fromIndex
		}

		for j := startIdx; j < len(candles); j++ {
			if zones[i].Direction == 1 {
				// Buyside: swept if price goes above the level
				if candles[j].High > zones[i].Level {
					zones[i].Swept = true
					zones[i].SweptAt = j
					break
				}
			} else {
				// Sellside: swept if price goes below the level
				if candles[j].Low < zones[i].Level {
					zones[i].Swept = true
					zones[i].SweptAt = j
					break
				}
			}
		}
	}

	return zones
}

// DetectLiquiditySweep checks if the current candle swept liquidity
func DetectLiquiditySweep(candle database.Candle, zones []LiquidityZone, barIndex int) *LiquiditySweep {
	for i := range zones {
		if zones[i].Swept {
			continue
		}

		if zones[i].Direction == 1 {
			// Buyside sweep: price went above the level
			if candle.High > zones[i].Level {
				wickOnly := candle.Close < zones[i].Level
				return &LiquiditySweep{
					Zone:      &zones[i],
					SweepBar:  barIndex,
					SweepTime: candle.Timestamp,
					WickOnly:  wickOnly,
					Magnitude: (candle.High - zones[i].Level) / zones[i].Level,
				}
			}
		} else {
			// Sellside sweep: price went below the level
			if candle.Low < zones[i].Level {
				wickOnly := candle.Close > zones[i].Level
				return &LiquiditySweep{
					Zone:      &zones[i],
					SweepBar:  barIndex,
					SweepTime: candle.Timestamp,
					WickOnly:  wickOnly,
					Magnitude: (zones[i].Level - candle.Low) / zones[i].Level,
				}
			}
		}
	}

	return nil
}

// WasLiquiditySweptRecently checks if liquidity was swept within N bars
func WasLiquiditySweptRecently(zones []LiquidityZone, currentBar int, maxBarsAgo int, direction int) *LiquidityZone {
	for i := range zones {
		if !zones[i].Swept {
			continue
		}
		if direction != 0 && zones[i].Direction != direction {
			continue
		}
		if currentBar-zones[i].SweptAt <= maxBarsAgo {
			return &zones[i]
		}
	}
	return nil
}

// GetActiveZones returns zones that haven't been swept
func GetActiveZones(zones []LiquidityZone) []LiquidityZone {
	var active []LiquidityZone
	for _, z := range zones {
		if !z.Swept {
			active = append(active, z)
		}
	}
	return active
}

// GetNearestLiquidityZone finds the closest unswepted zone to current price
func GetNearestLiquidityZone(price float64, zones []LiquidityZone, direction int) *LiquidityZone {
	var nearest *LiquidityZone
	minDistance := float64(1e10)

	for i := range zones {
		if zones[i].Swept {
			continue
		}
		if direction != 0 && zones[i].Direction != direction {
			continue
		}

		distance := math.Abs(price - zones[i].Level)
		if distance < minDistance {
			minDistance = distance
			nearest = &zones[i]
		}
	}

	return nearest
}

// Helper: remove duplicate zones within tolerance
func deduplicateZones(zones []LiquidityZone, tolerance float64) []LiquidityZone {
	if len(zones) == 0 {
		return zones
	}

	var unique []LiquidityZone
	for _, z := range zones {
		isDup := false
		for i := range unique {
			if unique[i].Direction == z.Direction {
				diff := math.Abs(unique[i].Level-z.Level) / unique[i].Level
				if diff <= tolerance*2 {
					isDup = true
					// Keep the stronger one
					if z.Strength > unique[i].Strength {
						unique[i] = z
					}
					break
				}
			}
		}
		if !isDup {
			unique = append(unique, z)
		}
	}

	return unique
}
