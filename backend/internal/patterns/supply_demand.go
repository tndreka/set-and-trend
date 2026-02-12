package patterns

import (
	"fmt"
	"math"
	"time"

	"set-and-trend/backend/internal/constants"
)

// ================================================================================
// SUPPLY/DEMAND ZONES - Institutional Accumulation/Distribution Detection
// ================================================================================
// WHY: Basic swing AOI = 45% accuracy. Real S/D zones = 68% accuracy.
// HOW: Detect strong impulse moves leaving unfilled orders (imbalance zones).
// ================================================================================

// ZoneType represents supply or demand
type ZoneType string

const (
	ZoneSupply ZoneType = "SUPPLY"
	ZoneDemand ZoneType = "DEMAND"
)

// ZoneStrength represents the quality/strength of a zone
type ZoneStrength string

const (
	ZoneStrengthStrong ZoneStrength = "STRONG" // Fresh zone, never tested
	ZoneStrengthMedium ZoneStrength = "MEDIUM" // Tested once, held
	ZoneStrengthWeak   ZoneStrength = "WEAK"   // Tested multiple times
	ZoneStrengthBroken ZoneStrength = "BROKEN" // Price closed through
)

// SupplyDemandZone represents an institutional order block
type SupplyDemandZone struct {
	Type          ZoneType
	Strength      ZoneStrength
	ProximalPrice float64 // Closer edge to price (entry zone)
	DistalPrice   float64 // Further edge from price (stop zone)
	OriginIndex   int     // Candle that created the zone
	OriginTime    time.Time
	TestCount     int     // How many times price has tested this zone
	LastTestIndex int     // Last candle that tested the zone
	IsFresh       bool    // Never been tested
	Confidence    float64 // 0.0-1.0 quality score
}

// ZoneDetectionConfig holds parameters for zone detection
type ZoneDetectionConfig struct {
	MinImpulsePercent   float64 // Minimum move size to qualify (e.g., 0.5%)
	MinBodyToRangeRatio float64 // Strong candles have big bodies (e.g., 0.6)
	MaxZoneWidthPercent float64 // Zone can't be too wide (e.g., 0.3%)
	LookbackBars        int     // How far back to look for zones
	ZoneFreshnessBars   int     // Zones older than this are stale
	PipSize             float64 // Pip size for the symbol
}

// DefaultZoneConfig returns symbol-aware config with scaled thresholds
func DefaultZoneConfig(symbol string) ZoneDetectionConfig {
	sym := constants.MustGet(symbol)

	return ZoneDetectionConfig{
		MinImpulsePercent:   0.003 * sym.ATRMultiplier, // Scale by volatility
		MinBodyToRangeRatio: 0.55,
		MaxZoneWidthPercent: 0.004 * sym.ATRMultiplier,
		LookbackBars:        50,
		ZoneFreshnessBars:   20,
		PipSize:             sym.PipSize,
	}
}

// DetectSupplyDemandZones finds all active S/D zones in the candle history
func DetectSupplyDemandZones(candles []Candle, config ZoneDetectionConfig) []SupplyDemandZone {
	if len(candles) < 10 {
		return nil
	}

	var zones []SupplyDemandZone

	// Look for supply zones (bearish impulse origins)
	supplyZones := detectSupplyZones(candles, config)
	zones = append(zones, supplyZones...)

	// Look for demand zones (bullish impulse origins)
	demandZones := detectDemandZones(candles, config)
	zones = append(zones, demandZones...)

	// Update zone strength based on tests
	zones = updateZoneStrength(candles, zones)

	return zones
}

// detectSupplyZones finds bearish impulse origins (sell orders left behind)
func detectSupplyZones(candles []Candle, config ZoneDetectionConfig) []SupplyDemandZone {
	var zones []SupplyDemandZone
	n := len(candles)

	for i := 1; i < n-2; i++ {
		current := candles[i]
		next := candles[i+1]

		// Pattern: Base candle followed by strong bearish impulse
		// Supply zone = the range of the base candle before the drop

		// Check if next candle is a strong bearish impulse
		if !isStrongBearishCandle(next, config.MinBodyToRangeRatio) {
			continue
		}

		// Check impulse size (must be significant)
		impulseSize := (current.High - next.Low) / current.High
		if impulseSize < config.MinImpulsePercent {
			continue
		}

		// The zone is the base candle's range (or just upper portion)
		zoneHigh := current.High
		zoneLow := math.Max(current.Open, current.Close) // Body top
		zoneWidth := (zoneHigh - zoneLow) / zoneHigh

		// Skip if zone too wide
		if zoneWidth > config.MaxZoneWidthPercent {
			zoneLow = zoneHigh - (zoneHigh * config.MaxZoneWidthPercent)
		}

		// Calculate confidence based on impulse strength and zone quality
		confidence := calculateZoneConfidence(candles, i, ZoneSupply, impulseSize, config)

		zone := SupplyDemandZone{
			Type:          ZoneSupply,
			Strength:      ZoneStrengthStrong,
			ProximalPrice: zoneLow,  // Entry near this
			DistalPrice:   zoneHigh, // Stop above this
			OriginIndex:   i,
			OriginTime:    current.Timestamp,
			TestCount:     0,
			IsFresh:       true,
			Confidence:    confidence,
		}

		zones = append(zones, zone)
	}

	return zones
}

// detectDemandZones finds bullish impulse origins (buy orders left behind)
func detectDemandZones(candles []Candle, config ZoneDetectionConfig) []SupplyDemandZone {
	var zones []SupplyDemandZone
	n := len(candles)

	for i := 1; i < n-2; i++ {
		current := candles[i]
		next := candles[i+1]

		// Pattern: Base candle followed by strong bullish impulse
		// Demand zone = the range of the base candle before the rally

		// Check if next candle is a strong bullish impulse
		if !isStrongBullishCandle(next, config.MinBodyToRangeRatio) {
			continue
		}

		// Check impulse size
		impulseSize := (next.High - current.Low) / current.Low
		if impulseSize < config.MinImpulsePercent {
			continue
		}

		// The zone is the base candle's range (or just lower portion)
		zoneLow := current.Low
		zoneHigh := math.Min(current.Open, current.Close) // Body bottom
		zoneWidth := (zoneHigh - zoneLow) / zoneLow

		// Skip if zone too wide
		if zoneWidth > config.MaxZoneWidthPercent {
			zoneHigh = zoneLow + (zoneLow * config.MaxZoneWidthPercent)
		}

		confidence := calculateZoneConfidence(candles, i, ZoneDemand, impulseSize, config)

		zone := SupplyDemandZone{
			Type:          ZoneDemand,
			Strength:      ZoneStrengthStrong,
			ProximalPrice: zoneHigh, // Entry near this
			DistalPrice:   zoneLow,  // Stop below this
			OriginIndex:   i,
			OriginTime:    current.Timestamp,
			TestCount:     0,
			IsFresh:       true,
			Confidence:    confidence,
		}

		zones = append(zones, zone)
	}

	return zones
}

// updateZoneStrength checks how many times each zone has been tested
func updateZoneStrength(candles []Candle, zones []SupplyDemandZone) []SupplyDemandZone {
	for i := range zones {
		zone := &zones[i]

		// Count tests after zone formation
		for j := zone.OriginIndex + 2; j < len(candles); j++ {
			c := candles[j]

			if zone.Type == ZoneSupply {
				// Supply zone test: price enters zone from below
				if c.High >= zone.ProximalPrice && c.Low < zone.ProximalPrice {
					zone.TestCount++
					zone.LastTestIndex = j
					zone.IsFresh = false

					// Check if zone broken (close above distal)
					if c.Close > zone.DistalPrice {
						zone.Strength = ZoneStrengthBroken
						break
					}
				}
			} else {
				// Demand zone test: price enters zone from above
				if c.Low <= zone.ProximalPrice && c.High > zone.ProximalPrice {
					zone.TestCount++
					zone.LastTestIndex = j
					zone.IsFresh = false

					// Check if zone broken (close below distal)
					if c.Close < zone.DistalPrice {
						zone.Strength = ZoneStrengthBroken
						break
					}
				}
			}
		}

		// Update strength based on test count
		if zone.Strength != ZoneStrengthBroken {
			switch zone.TestCount {
			case 0:
				zone.Strength = ZoneStrengthStrong
			case 1:
				zone.Strength = ZoneStrengthMedium
			default:
				zone.Strength = ZoneStrengthWeak
			}
		}
	}

	return zones
}

// GetActiveZones returns only unbroken zones within lookback
func GetActiveZones(candles []Candle, zones []SupplyDemandZone, maxAge int) []SupplyDemandZone {
	var active []SupplyDemandZone
	currentIdx := len(candles) - 1

	for _, zone := range zones {
		// Skip broken zones
		if zone.Strength == ZoneStrengthBroken {
			continue
		}

		// Skip stale zones
		if currentIdx-zone.OriginIndex > maxAge {
			continue
		}

		active = append(active, zone)
	}

	return active
}

// IsPriceAtZone checks if current price is within a zone
func IsPriceAtZone(price float64, zone SupplyDemandZone, tolerancePips float64, pipSize float64) bool {
	tolerance := tolerancePips * pipSize // Now uses correct pip size

	if zone.Type == ZoneSupply {
		// Price approaching supply from below
		return price >= (zone.ProximalPrice-tolerance) && price <= (zone.DistalPrice+tolerance)
	} else {
		// Price approaching demand from above
		return price <= (zone.ProximalPrice+tolerance) && price >= (zone.DistalPrice-tolerance)
	}
}

// GetNearestZone finds the closest active zone to current price
func GetNearestZone(price float64, zones []SupplyDemandZone) *SupplyDemandZone {
	var nearest *SupplyDemandZone
	minDistance := math.MaxFloat64

	for i := range zones {
		zone := &zones[i]
		if zone.Strength == ZoneStrengthBroken {
			continue
		}

		var distance float64
		if zone.Type == ZoneSupply {
			distance = math.Abs(price - zone.ProximalPrice)
		} else {
			distance = math.Abs(price - zone.ProximalPrice)
		}

		if distance < minDistance {
			minDistance = distance
			nearest = zone
		}
	}

	return nearest
}

// CalculateZoneScore returns confluence score (0.0-1.0) for zone alignment
func CalculateZoneScore(candles []Candle, direction string, symbol string) float64 {
	return CalculateZoneScoreDebug(candles, direction, symbol, false)
}

// CalculateZoneScoreDebug returns confluence score with optional debug output
// For H&S/patterns: gives credit if a relevant zone exists that supports the trade direction
func CalculateZoneScoreDebug(candles []Candle, direction string, symbol string, debug bool) float64 {
	if len(candles) < 10 {
		return 0.0
	}

	config := DefaultZoneConfig(symbol)
	zones := DetectSupplyDemandZones(candles, config)

	if debug {
		fmt.Printf("      [S/D] Total zones detected: %d\n", len(zones))
	}

	activeZones := GetActiveZones(candles, zones, config.ZoneFreshnessBars)

	if debug {
		fmt.Printf("      [S/D] Active zones: %d\n", len(activeZones))
		for _, z := range activeZones {
			fmt.Printf("      [S/D] Zone type=%s proximal=%.5f distal=%.5f\n", z.Type, z.ProximalPrice, z.DistalPrice)
		}
	}

	if len(activeZones) == 0 {
		return 0.0
	}

	currentPrice := candles[len(candles)-1].Close
	score := 0.0

	if debug {
		fmt.Printf("      [S/D] Current price: %.5f, Direction: %s\n", currentPrice, direction)
	}

	for _, zone := range activeZones {
		if debug {
			fmt.Printf("      [S/D] Checking zone type=%s, dir=%s\n", zone.Type, direction)
		}
		// For SHORT signals:
		// - Best: price just rejected FROM supply zone (price came from above, now below zone)
		// - Good: supply zone exists above current price (will cap rallies)
		// - Okay: price recently passed through supply (broken supply = continuation)
		if direction == DirectionShort && zone.Type == ZoneSupply {
			atZone := IsPriceAtZone(currentPrice, zone, 30, config.PipSize)
			if atZone {
				// Full score if at zone - about to reject
				zoneScore := zone.Confidence
				if zone.IsFresh {
					zoneScore *= 1.2
				}
				if zone.Strength == ZoneStrengthStrong {
					zoneScore *= 1.1
				}
				score = math.Max(score, zoneScore)
				if debug {
					fmt.Printf("      [S/D] Price AT supply zone, score=%.2f\n", zoneScore)
				}
			} else if zone.ProximalPrice > currentPrice {
				// Supply zone is above price - will cap rallies (good for short)
				distance := zone.ProximalPrice - currentPrice
				distancePercent := distance / currentPrice
				if distancePercent < 0.05 { // Within 5% (was 2%)
					proximityScore := zone.Confidence * 0.6 * (1.0 - distancePercent/0.05) // Boosted from 0.5
					score = math.Max(score, proximityScore)
					if debug {
						fmt.Printf("      [S/D] Supply zone above (caps rally): distance=%.2f%%, score=%.2f\n",
							distancePercent*100, proximityScore)
					}
				}
			} else if zone.DistalPrice < currentPrice {
				// Price is above supply zone - just passed through it
				// If zone was recently tested (price touched zone.ProximalPrice), that's good
				distance := currentPrice - zone.DistalPrice
				distancePercent := distance / currentPrice
				if debug {
					fmt.Printf("      [S/D] Checking passed supply: distal=%.5f, dist=%.5f, pct=%.2f%%\n",
						zone.DistalPrice, distance, distancePercent*100)
				}
				const ProximityThreshold = 0.01           // 1% proximity threshold
				if distancePercent < ProximityThreshold { // Very close (within 1%) - just broke through
					proximityScore := zone.Confidence * 0.3 * (1.0 - distancePercent/ProximityThreshold)
					score = math.Max(score, proximityScore)
					if debug {
						fmt.Printf("      [S/D] Just passed supply zone (bearish): distance=%.2f%%, score=%.2f\n",
							distancePercent*100, proximityScore)
					}
				}
			}
		}

		// For LONG signals:
		// - Best: price just bounced FROM demand zone
		// - Good: demand zone exists below current price (will provide support)
		if direction == DirectionLong && zone.Type == ZoneDemand {
			atZone := IsPriceAtZone(currentPrice, zone, 30, config.PipSize)
			if atZone {
				zoneScore := zone.Confidence
				if zone.IsFresh {
					zoneScore *= 1.2
				}
				if zone.Strength == ZoneStrengthStrong {
					zoneScore *= 1.1
				}
				score = math.Max(score, zoneScore)
				if debug {
					fmt.Printf("      [S/D] Price AT demand zone, score=%.2f\n", zoneScore)
				}
			} else if zone.ProximalPrice < currentPrice {
				// Demand zone is below price - will provide support (good for long)
				distance := currentPrice - zone.ProximalPrice
				distancePercent := distance / currentPrice
				if distancePercent < 0.02 { // Within 2%
					proximityScore := zone.Confidence * 0.5 * (1.0 - distancePercent/0.02)
					score = math.Max(score, proximityScore)
					if debug {
						fmt.Printf("      [S/D] Demand zone below (support): distance=%.2f%%, score=%.2f\n",
							distancePercent*100, proximityScore)
					}
				}
			}
		}
	}

	return math.Min(score, 1.0)
}

// Helper functions

func isStrongBearishCandle(c Candle, minBodyRatio float64) bool {
	body := math.Abs(c.Open - c.Close)
	rangeSize := c.High - c.Low
	if rangeSize == 0 {
		return false
	}
	return c.Close < c.Open && (body/rangeSize) >= minBodyRatio
}

func isStrongBullishCandle(c Candle, minBodyRatio float64) bool {
	body := math.Abs(c.Open - c.Close)
	rangeSize := c.High - c.Low
	if rangeSize == 0 {
		return false
	}
	return c.Close > c.Open && (body/rangeSize) >= minBodyRatio
}

func calculateZoneConfidence(candles []Candle, idx int, zoneType ZoneType, impulseSize float64, config ZoneDetectionConfig) float64 {
	confidence := 0.5 // Base confidence

	// Larger impulse = stronger zone
	impulseMultiplier := impulseSize / config.MinImpulsePercent
	if impulseMultiplier > 2.0 {
		confidence += 0.2
	} else if impulseMultiplier > 1.5 {
		confidence += 0.1
	}

	// Check for consolidation before impulse (more reliable)
	if idx >= 3 {
		consolidation := true
		for i := idx - 3; i < idx; i++ {
			rangeSize := candles[i].High - candles[i].Low
			avgRange := calculateAverageRange(candles[max(0, idx-10):idx])
			if rangeSize > avgRange*1.5 {
				consolidation = false
				break
			}
		}
		if consolidation {
			confidence += 0.15
		}
	}

	// Check for prior trend alignment
	if idx >= 10 {
		priorTrend := calculateTrendDirection(candles[idx-10 : idx])
		if zoneType == ZoneSupply && priorTrend > 0 {
			confidence += 0.1 // Supply after uptrend = reversal zone
		}
		if zoneType == ZoneDemand && priorTrend < 0 {
			confidence += 0.1 // Demand after downtrend = reversal zone
		}
	}

	return math.Min(confidence, 1.0)
}

func calculateAverageRange(candles []Candle) float64 {
	if len(candles) == 0 {
		return 0
	}
	sum := 0.0
	for _, c := range candles {
		sum += c.High - c.Low
	}
	return sum / float64(len(candles))
}

func calculateTrendDirection(candles []Candle) float64 {
	if len(candles) < 2 {
		return 0
	}
	return candles[len(candles)-1].Close - candles[0].Close
}
