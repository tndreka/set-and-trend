package patterns

import (
	"math"

	"set-and-trend/backend/internal/constants"
)

// ================================================================================
// DYNAMIC RISK:REWARD - Confluence-Based Position Optimization
// ================================================================================

type RiskRewardTier struct {
	Name          string
	MinConfluence float64
	TargetRR      float64
	PartialExitRR float64
	StopMode      string
}

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
		MinConfluence: 0.52,
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

type DynamicRRConfig struct {
	MinRR              float64
	MaxRR              float64
	UseATRStops        bool
	ATRMultiplier      float64
	UseStructureStops  bool
	StructureBuffer    float64
	EnablePartialExits bool
	PartialExitPercent float64
}

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

type DynamicRRResult struct {
	Tier              RiskRewardTier
	TargetRR          float64
	AdjustedEntry     float64
	AdjustedStop      float64
	PrimaryTarget     float64
	PartialTarget     float64
	TrailingStopLevel float64
	RiskPips          float64
	RewardPips        float64
	PositionScore     float64
}

// CalculateDynamicRR determines optimal R:R based on confluence
func CalculateDynamicRR(signal *TradeSignal, confluenceScore float64, config DynamicRRConfig) *DynamicRRResult {
	if signal == nil {
		return nil
	}

	tier := getTierForConfluence(confluenceScore)

	result := &DynamicRRResult{
		Tier:          tier,
		TargetRR:      tier.TargetRR,
		AdjustedEntry: signal.EntryPrice,
		AdjustedStop:  signal.StopLoss,
	}

	// Calculate risk in pips using symbol-aware pip size
	sym := constants.MustGet(signal.Symbol)
	pipSize := sym.PipSize
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
		result.TrailingStopLevel = signal.EntryPrice + (result.RiskPips * pipSize)
	} else {
		result.PrimaryTarget = signal.EntryPrice - (result.RewardPips * pipSize)
		result.PartialTarget = signal.EntryPrice - (result.RiskPips * tier.PartialExitRR * pipSize)
		result.TrailingStopLevel = signal.EntryPrice - (result.RiskPips * pipSize)
	}

	result.PositionScore = calculatePositionScore(signal, tier, confluenceScore)

	return result
}

func getTierForConfluence(confluence float64) RiskRewardTier {
	var selected RiskRewardTier
	for _, tier := range RRTiers {
		if confluence >= tier.MinConfluence {
			selected = tier
		}
	}
	return selected
}

func calculatePositionScore(signal *TradeSignal, tier RiskRewardTier, confluence float64) float64 {
	score := 0.0
	score += confluence * 0.4

	if signal.RiskReward >= 2.0 {
		score += 0.3
	} else if signal.RiskReward >= 1.5 {
		score += 0.2
	} else if signal.RiskReward >= 1.0 {
		score += 0.1
	}

	score += signal.Confidence * 0.3

	return math.Min(score, 1.0)
}

// CalculateATRStop calculates stop loss using ATR in PRICE units
// Note: ATR is calculated in price units, not pips, so we don't convert
// The result is a price level directly usable for orders
func CalculateATRStop(candles []Candle, entry float64, direction string, atrMultiplier float64, symbol string) float64 {
	if len(candles) < 14 {
		return 0
	}

	// We fetch symbol info for validation only - ATR is price-based
	sym := constants.MustGet(symbol)
	_ = sym // Used for validation; ATR calculation doesn't need pip conversion

	atr := CalculateATR(candles, 14)

	// ATR is in price units, entry is in price units
	// Result is directly a price level
	if direction == DirectionLong {
		return entry - (atr * atrMultiplier)
	}
	return entry + (atr * atrMultiplier)
}

// CalculateATRStopInPips returns the stop distance in pips (for logging/RR calc)
func CalculateATRStopInPips(candles []Candle, entry float64, direction string, atrMultiplier float64, symbol string) float64 {
	stopPrice := CalculateATRStop(candles, entry, direction, atrMultiplier, symbol)
	if stopPrice == 0 {
		return 0
	}

	sym := constants.MustGet(symbol)
	distance := math.Abs(entry - stopPrice)
	return sym.PriceToPips(distance)
}

// CalculateATR calculates Average True Range
func CalculateATR(candles []Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	var trSum float64
	startIdx := len(candles) - period

	for i := startIdx; i < len(candles); i++ {
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
		lows := FindSwingLows(candles, 3)
		if len(lows) > 0 {
			recentLow := lows[len(lows)-1].Price
			return recentLow - (bufferPips * pipSize)
		}
	} else {
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
		atrStop := CalculateATRStop(candles, signal.EntryPrice, signal.Direction, config.ATRMultiplier, signal.Symbol)
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

func CreatePartialExitPlan(signal *TradeSignal, tier RiskRewardTier, pipSize float64) *PartialExitPlan {
	if signal == nil {
		return nil
	}

	risk := math.Abs(signal.EntryPrice - signal.StopLoss)

	plan := &PartialExitPlan{
		Exit1Percent: 0.33,
		Exit1RR:      tier.PartialExitRR,
		Exit2Percent: 0.33,
		Exit2RR:      tier.TargetRR * 0.75,
		FinalPercent: 0.34,
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

func ShouldTakeTrade(signal *TradeSignal, confluenceScore float64, minConfluence float64) bool {
	if signal == nil {
		return false
	}

	if confluenceScore < minConfluence {
		return false
	}

	tier := getTierForConfluence(confluenceScore)

	if signal.RiskReward < tier.TargetRR*0.8 {
		return false
	}

	return true
}

func GetRecommendedPositionSize(baseRiskPercent, confluenceScore float64) float64 {
	if confluenceScore >= 0.85 {
		return baseRiskPercent * 1.25
	} else if confluenceScore >= 0.75 {
		return baseRiskPercent * 1.15
	} else if confluenceScore >= 0.65 {
		return baseRiskPercent * 1.0
	} else if confluenceScore >= 0.52 {
		return baseRiskPercent * 0.75
	}
	return baseRiskPercent * 0.5
}
