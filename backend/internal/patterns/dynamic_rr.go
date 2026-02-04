package patterns

import (
	"math"
)

// ================================================================================
// DYNAMIC RISK:REWARD - Confluence-Based Position Optimization
// ================================================================================
// WHY: 65%+ confluence = 2.5:1 RR vs fixed 1:1 = +25% profit improvement.
// HOW: Scale R:R targets based on confluence score and pattern quality.
// ================================================================================

// RiskRewardTier represents different R:R targets
type RiskRewardTier struct {
	Name          string
	MinConfluence float64
	TargetRR      float64
	PartialExitRR float64 // First partial exit point
	StopMode      string  // "FIXED", "ATR", "STRUCTURE"
}

// Standard R:R tiers
var RRTiers = []RiskRewardTier{
	{
		Name:          "LOW_CONFLUENCE",
		MinConfluence: 0.0,
		TargetRR:      1.0,
		PartialExitRR: 0.5,
		StopMode:      "FIXED",
	},
	{
		Name:          "MEDIUM_CONFLUENCE",
		MinConfluence: 0.52, // New 52% threshold
		TargetRR:      1.5,
		PartialExitRR: 1.0,
		StopMode:      "FIXED",
	},
	{
		Name:          "HIGH_CONFLUENCE",
		MinConfluence: 0.65,
		TargetRR:      2.0,
		PartialExitRR: 1.0,
		StopMode:      "ATR",
	},
	{
		Name:          "PREMIUM_CONFLUENCE",
		MinConfluence: 0.75,
		TargetRR:      2.5,
		PartialExitRR: 1.5,
		StopMode:      "STRUCTURE",
	},
	{
		Name:          "A_PLUS_SETUP",
		MinConfluence: 0.85,
		TargetRR:      3.0,
		PartialExitRR: 2.0,
		StopMode:      "STRUCTURE",
	},
}

// DynamicRRConfig holds risk management parameters
type DynamicRRConfig struct {
	MinRR               float64 // Never take trades below this R:R
	MaxRR               float64 // Cap R:R for realistic targets
	UseATRStops         bool    // Use ATR for stop placement
	ATRMultiplier       float64 // e.g., 1.5 ATR for stops
	UseStructureStops   bool    // Use swing highs/lows for stops
	StructureBuffer     float64 // Pips beyond structure
	EnablePartialExits  bool    // Take partial profits
	PartialExitPercent  float64 // % of position to exit at first target
}

// DefaultDynamicRRConfig returns sensible defaults
func DefaultDynamicRRConfig() DynamicRRConfig {
	return DynamicRRConfig{
		MinRR:              1.0,
		MaxRR:              4.0,
		UseATRStops:        true,
		ATRMultiplier:      1.5,
		UseStructureStops:  true,
		StructureBuffer:    10,
		EnablePartialExits: true,
		PartialExitPercent: 0.5,
	}
}

// DynamicRRResult holds the calculated R:R parameters
type DynamicRRResult struct {
	Tier             RiskRewardTier
	TargetRR         float64
	AdjustedEntry    float64
	AdjustedStop     float64
	PrimaryTarget    float64
	PartialTarget    float64
	TrailingStopLevel float64
	RiskPips         float64
	RewardPips       float64
	PositionScore    float64 // Overall position quality
}

// CalculateDynamicRR determines optimal R:R based on confluence
func CalculateDynamicRR(signal *TradeSignal, confluenceScore float64, config DynamicRRConfig) *DynamicRRResult {
	if signal == nil {
		return nil
	}

	// Get the appropriate tier
	tier := getTierForConfluence(confluenceScore)

	result := &DynamicRRResult{
		Tier:          tier,
		TargetRR:      tier.TargetRR,
		AdjustedEntry: signal.EntryPrice,
		AdjustedStop:  signal.StopLoss,
	}

	// Calculate risk in pips
	pipSize := 0.0001 // Default, should use constants
	if signal.Direction == DirectionLong {
		result.RiskPips = (signal.EntryPrice - signal.StopLoss) / pipSize
	} else {
		result.RiskPips = (signal.StopLoss - signal.EntryPrice) / pipSize
	}

	// Calculate targets based on tier R:R
	result.RewardPips = result.RiskPips * tier.TargetRR

	if signal.Direction == DirectionLong {
		result.PrimaryTarget = signal.EntryPrice + (result.RewardPips * pipSize)
		result.PartialTarget = signal.EntryPrice + (result.RiskPips * tier.PartialExitRR * pipSize)
		result.TrailingStopLevel = signal.EntryPrice + (result.RiskPips * pipSize) // Breakeven after 1R
	} else {
		result.PrimaryTarget = signal.EntryPrice - (result.RewardPips * pipSize)
		result.PartialTarget = signal.EntryPrice - (result.RiskPips * tier.PartialExitRR * pipSize)
		result.TrailingStopLevel = signal.EntryPrice - (result.RiskPips * pipSize)
	}

	// Calculate position quality score
	result.PositionScore = calculatePositionScore(signal, tier, confluenceScore)

	return result
}

// getTierForConfluence returns the appropriate R:R tier
func getTierForConfluence(confluence float64) RiskRewardTier {
	var selected RiskRewardTier
	for _, tier := range RRTiers {
		if confluence >= tier.MinConfluence {
			selected = tier
		}
	}
	return selected
}

// calculatePositionScore determines overall position quality
func calculatePositionScore(signal *TradeSignal, tier RiskRewardTier, confluence float64) float64 {
	score := 0.0

	// Base score from confluence
	score += confluence * 0.4

	// R:R quality
	if signal.RiskReward >= 2.0 {
		score += 0.3
	} else if signal.RiskReward >= 1.5 {
		score += 0.2
	} else if signal.RiskReward >= 1.0 {
		score += 0.1
	}

	// Pattern confidence
	score += signal.Confidence * 0.3

	return math.Min(score, 1.0)
}

// CalculateATRStop calculates stop loss using ATR
func CalculateATRStop(candles []Candle, entry float64, direction string, atrMultiplier float64, pipSize float64) float64 {
	if len(candles) < 14 {
		return 0
	}

	atr := CalculateATR(candles, 14)
	atrPips := atr / pipSize

	if direction == DirectionLong {
		return entry - (atrPips * atrMultiplier * pipSize)
	} else {
		return entry + (atrPips * atrMultiplier * pipSize)
	}
}

// CalculateATR calculates Average True Range
func CalculateATR(candles []Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	var trSum float64
	for i := len(candles) - period; i < len(candles); i++ {
		tr := calculateTrueRange(candles[i], candles[i-1])
		trSum += tr
	}

	return trSum / float64(period)
}

func calculateTrueRange(current, previous Candle) float64 {
	highLow := current.High - current.Low
	highClose := math.Abs(current.High - previous.Close)
	lowClose := math.Abs(current.Low - previous.Close)

	return math.Max(highLow, math.Max(highClose, lowClose))
}

// CalculateStructureStop calculates stop based on market structure
func CalculateStructureStop(candles []Candle, entry float64, direction string, bufferPips, pipSize float64) float64 {
	if len(candles) < 10 {
		return 0
	}

	if direction == DirectionLong {
		// Find recent swing low for stop
		lows := FindSwingLows(candles, 3)
		if len(lows) > 0 {
			recentLow := lows[len(lows)-1].Price
			return recentLow - (bufferPips * pipSize)
		}
	} else {
		// Find recent swing high for stop
		highs := FindSwingHighs(candles, 3)
		if len(highs) > 0 {
			recentHigh := highs[len(highs)-1].Price
			return recentHigh + (bufferPips * pipSize)
		}
	}

	return 0
}

// OptimizeSignalRR takes a signal and optimizes its R:R based on confluence
func OptimizeSignalRR(signal *TradeSignal, candles []Candle, confluenceScore float64, pipSize float64) *TradeSignal {
	if signal == nil {
		return nil
	}

	config := DefaultDynamicRRConfig()
	dynamicRR := CalculateDynamicRR(signal, confluenceScore, config)

	if dynamicRR == nil {
		return signal
	}

	// Create optimized signal
	optimized := &TradeSignal{
		Symbol:      signal.Symbol,
		Direction:   signal.Direction,
		PatternType: signal.PatternType,
		Confidence:  signal.Confidence,
	}

	// Use structure-based stop if high confluence
	if dynamicRR.Tier.StopMode == "STRUCTURE" && len(candles) > 10 {
		structureStop := CalculateStructureStop(candles, signal.EntryPrice, signal.Direction, config.StructureBuffer, pipSize)
		if structureStop > 0 {
			optimized.StopLoss = structureStop
		} else {
			optimized.StopLoss = signal.StopLoss
		}
	} else if dynamicRR.Tier.StopMode == "ATR" && len(candles) > 14 {
		atrStop := CalculateATRStop(candles, signal.EntryPrice, signal.Direction, config.ATRMultiplier, pipSize)
		if atrStop > 0 {
			optimized.StopLoss = atrStop
		} else {
			optimized.StopLoss = signal.StopLoss
		}
	} else {
		optimized.StopLoss = signal.StopLoss
	}

	optimized.EntryPrice = signal.EntryPrice

	// Calculate target based on dynamic R:R
	risk := math.Abs(optimized.EntryPrice - optimized.StopLoss)
	reward := risk * dynamicRR.TargetRR

	if signal.Direction == DirectionLong {
		optimized.TakeProfit = optimized.EntryPrice + reward
	} else {
		optimized.TakeProfit = optimized.EntryPrice - reward
	}

	optimized.RiskReward = dynamicRR.TargetRR

	return optimized
}

// PartialExitPlan creates a plan for partial exits
type PartialExitPlan struct {
	Exit1Percent float64
	Exit1Price   float64
	Exit1RR      float64
	Exit2Percent float64
	Exit2Price   float64
	Exit2RR      float64
	FinalPercent float64
	FinalPrice   float64
	FinalRR      float64
}

// CreatePartialExitPlan creates a scaling exit strategy
func CreatePartialExitPlan(signal *TradeSignal, tier RiskRewardTier, pipSize float64) *PartialExitPlan {
	if signal == nil {
		return nil
	}

	risk := math.Abs(signal.EntryPrice - signal.StopLoss)

	plan := &PartialExitPlan{
		Exit1Percent: 0.33, // 33% at first target
		Exit1RR:      tier.PartialExitRR,
		Exit2Percent: 0.33, // 33% at second target
		Exit2RR:      tier.TargetRR * 0.75,
		FinalPercent: 0.34, // 34% at final target
		FinalRR:      tier.TargetRR,
	}

	if signal.Direction == DirectionLong {
		plan.Exit1Price = signal.EntryPrice + (risk * plan.Exit1RR)
		plan.Exit2Price = signal.EntryPrice + (risk * plan.Exit2RR)
		plan.FinalPrice = signal.EntryPrice + (risk * plan.FinalRR)
	} else {
		plan.Exit1Price = signal.EntryPrice - (risk * plan.Exit1RR)
		plan.Exit2Price = signal.EntryPrice - (risk * plan.Exit2RR)
		plan.FinalPrice = signal.EntryPrice - (risk * plan.FinalRR)
	}

	return plan
}

// ShouldTakeTrade returns whether to take a trade based on R:R and confluence
func ShouldTakeTrade(signal *TradeSignal, confluenceScore float64, minConfluence float64) bool {
	if signal == nil {
		return false
	}

	// Must meet minimum confluence threshold
	if confluenceScore < minConfluence {
		return false
	}

	// Get tier for confluence
	tier := getTierForConfluence(confluenceScore)

	// R:R must be at least the tier minimum
	if signal.RiskReward < tier.TargetRR*0.8 { // Allow 20% buffer
		return false
	}

	return true
}

// GetRecommendedPositionSize calculates position size based on R:R and confluence
func GetRecommendedPositionSize(baseRiskPercent, confluenceScore float64) float64 {
	// Higher confluence = slightly higher risk allowance (within limits)
	if confluenceScore >= 0.85 {
		return baseRiskPercent * 1.25 // 25% increase for A+ setups
	} else if confluenceScore >= 0.75 {
		return baseRiskPercent * 1.15
	} else if confluenceScore >= 0.65 {
		return baseRiskPercent * 1.0
	} else if confluenceScore >= 0.52 {
		return baseRiskPercent * 0.75 // Reduce risk for lower confluence
	}
	return baseRiskPercent * 0.5 // Minimum for low confluence
}
