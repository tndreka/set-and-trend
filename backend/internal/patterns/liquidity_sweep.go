package patterns

import (
	"math"
	"time"

	"set-and-trend/backend/internal/constants"
)

// ================================================================================
// LIQUIDITY SWEEPS - Stop Hunt Detection (MATHEMATICALLY CORRECTED)
// ================================================================================

type SweepType string

const (
	SweepBullish SweepType = "BULLISH"
	SweepBearish SweepType = "BEARISH"
)

type LiquiditySweep struct {
	Type            SweepType
	SweepPrice      float64
	RecoveryPrice   float64
	LiquidityLevel  float64
	SweepDepthPips  float64
	RecoveryPercent float64
	SweepIndex      int
	SweepTime       time.Time
	Strength        float64
	IsConfirmed     bool
}

type LiquiditySweepConfig struct {
	LookbackBars     int
	MinSweepPips     float64
	MaxSweepPips     float64
	MinRecoveryPct   float64
	ConfirmationBars int
	PipSize          float64
}

// DefaultLiquiditySweepConfig returns symbol-aware config with scaled thresholds
func DefaultLiquiditySweepConfig(symbol string) LiquiditySweepConfig {
	sym := constants.MustGet(symbol)

	// Scale sweep thresholds by volatility
	baseMinSweep := 3.0
	baseMaxSweep := 30.0

	return LiquiditySweepConfig{
		LookbackBars:     20,
		MinSweepPips:     baseMinSweep * sym.ATRMultiplier, // 3 | 4.5 | 6
		MaxSweepPips:     baseMaxSweep * sym.ATRMultiplier, // 30 | 45 | 60
		MinRecoveryPct:   0.5,
		ConfirmationBars: 3,
		PipSize:          sym.PipSize,
	}
}

// DetectLiquiditySweeps detects liquidity sweeps with symbol-aware configuration
func DetectLiquiditySweeps(candles []Candle, symbol string, config LiquiditySweepConfig) []LiquiditySweep {
	// Validate symbol matches config (log warning if mismatch)
	if sym, ok := constants.Get(symbol); ok {
		if config.PipSize != sym.PipSize {
			// Config pip size doesn't match symbol - using config value
		}
	}
	return detectLiquiditySweepsWithConfig(candles, config)
}

// detectLiquiditySweepsWithConfig is the internal implementation
func detectLiquiditySweepsWithConfig(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	if len(candles) < config.LookbackBars+5 {
		return nil
	}

	var sweeps []LiquiditySweep
	sweeps = append(sweeps, detectBullishSweeps(candles, config)...)
	sweeps = append(sweeps, detectBearishSweeps(candles, config)...)
	return sweeps
}

// ===================== BULLISH SWEEPS =====================

func detectBullishSweeps(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	var sweeps []LiquiditySweep
	n := len(candles)

	for i := config.LookbackBars; i < n-1; i++ {
		current := candles[i]

		liquidityLevel := math.MaxFloat64
		for j := i - config.LookbackBars; j < i; j++ {
			liquidityLevel = math.Min(liquidityLevel, candles[j].Low)
		}

		rawDepth := (liquidityLevel - current.Low) / config.PipSize
		sweepDepth := math.Max(0, rawDepth)
		if sweepDepth < config.MinSweepPips || sweepDepth > config.MaxSweepPips {
			continue
		}

		if current.Close <= liquidityLevel {
			continue
		}

		denom := liquidityLevel - current.Low
		if denom <= 0 {
			continue
		}

		recoveryPct := (current.Close - current.Low) / denom
		if recoveryPct < config.MinRecoveryPct {
			continue
		}

		strength := calculateSweepStrength(current, liquidityLevel, sweepDepth, config, SweepBullish)

		sweep := LiquiditySweep{
			Type:            SweepBullish,
			SweepPrice:      current.Low,
			RecoveryPrice:   current.Close,
			LiquidityLevel:  liquidityLevel,
			SweepDepthPips:  sweepDepth,
			RecoveryPercent: recoveryPct,
			SweepIndex:      i,
			SweepTime:       current.Timestamp,
			Strength:        strength,
			IsConfirmed:     false,
		}

		if i+config.ConfirmationBars < n {
			confirmed := true
			for j := i + 1; j <= i+config.ConfirmationBars; j++ {
				if candles[j].Low < current.Low-config.PipSize {
					confirmed = false
					break
				}
			}
			sweep.IsConfirmed = confirmed
		}

		sweeps = append(sweeps, sweep)
	}

	return sweeps
}

// ===================== BEARISH SWEEPS =====================

func detectBearishSweeps(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	var sweeps []LiquiditySweep
	n := len(candles)

	for i := config.LookbackBars; i < n-1; i++ {
		current := candles[i]

		liquidityLevel := 0.0
		for j := i - config.LookbackBars; j < i; j++ {
			liquidityLevel = math.Max(liquidityLevel, candles[j].High)
		}

		rawDepth := (current.High - liquidityLevel) / config.PipSize
		sweepDepth := math.Max(0, rawDepth)
		if sweepDepth < config.MinSweepPips || sweepDepth > config.MaxSweepPips {
			continue
		}

		if current.Close >= liquidityLevel {
			continue
		}

		denom := current.High - liquidityLevel
		if denom <= 0 {
			continue
		}

		recoveryPct := (current.High - current.Close) / denom
		if recoveryPct < config.MinRecoveryPct {
			continue
		}

		strength := calculateSweepStrength(current, liquidityLevel, sweepDepth, config, SweepBearish)

		sweep := LiquiditySweep{
			Type:            SweepBearish,
			SweepPrice:      current.High,
			RecoveryPrice:   current.Close,
			LiquidityLevel:  liquidityLevel,
			SweepDepthPips:  sweepDepth,
			RecoveryPercent: recoveryPct,
			SweepIndex:      i,
			SweepTime:       current.Timestamp,
			Strength:        strength,
			IsConfirmed:     false,
		}

		if i+config.ConfirmationBars < n {
			confirmed := true
			for j := i + 1; j <= i+config.ConfirmationBars; j++ {
				if candles[j].High > current.High+config.PipSize {
					confirmed = false
					break
				}
			}
			sweep.IsConfirmed = confirmed
		}

		sweeps = append(sweeps, sweep)
	}

	return sweeps
}

// ===================== STRENGTH =====================

func calculateSweepStrength(
	candle Candle,
	liquidityLevel float64,
	sweepDepthPips float64,
	config LiquiditySweepConfig,
	sweepType SweepType,
) float64 {

	strength := 0.3

	depthScore := sweepDepthPips / config.MaxSweepPips
	strength += depthScore * 0.25

	rangeSize := candle.High - candle.Low
	if rangeSize <= 0 {
		return strength
	}

	bodySize := math.Abs(candle.Open - candle.Close)
	wickRatio := 1.0 - (bodySize / rangeSize)
	strength += wickRatio * 0.25

	if sweepType == SweepBullish {
		closeStrength := (candle.Close - candle.Low) / rangeSize
		strength += closeStrength * 0.2
	} else {
		closeStrength := (candle.High - candle.Close) / rangeSize
		strength += closeStrength * 0.2
	}

	return math.Min(strength, 1.0)
}

// ===================== SCORING =====================

func CalculateLiquiditySweepScore(candles []Candle, direction string, symbol string) float64 {
	return CalculateLiquiditySweepScoreDebug(candles, direction, symbol, false)
}

func CalculateLiquiditySweepScoreDebug(candles []Candle, direction string, symbol string, debug bool) float64 {
	if len(candles) < 25 {
		return 0
	}

	config := DefaultLiquiditySweepConfig(symbol)
	sweeps := DetectLiquiditySweeps(candles, symbol, config)

	if len(sweeps) == 0 {
		return 0
	}

	currentBar := len(candles) - 1
	score := 0.0

	for _, sweep := range sweeps {
		barsAgo := currentBar - sweep.SweepIndex
		if barsAgo > 30 {
			continue
		}

		recencyBonus := 1.0 - (float64(barsAgo) / 30.0)
		sweepScore := sweep.Strength * (1.0 + recencyBonus*0.3)

		if sweep.IsConfirmed {
			sweepScore *= 1.2
		}

		if direction == DirectionLong && sweep.Type == SweepBullish {
			score = math.Max(score, sweepScore)
		}

		if direction == DirectionShort && sweep.Type == SweepBearish {
			score = math.Max(score, sweepScore)
		}
	}

	return math.Min(score, 1.0)
}
