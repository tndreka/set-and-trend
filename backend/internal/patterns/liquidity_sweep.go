package patterns

import (
	"fmt"
	"math"
	"time"
)

// ================================================================================
// LIQUIDITY SWEEPS - Stop Hunt Detection
// ================================================================================
// WHY: 80% of fakeouts = stop hunts above/below zones.
// HOW: Detect price wicks beyond swing highs/lows that quickly reverse.
// H&S shoulders = perfect liquidity targets for institutions.
// ================================================================================

// SweepType indicates bullish or bearish sweep
type SweepType string

const (
	SweepBullish SweepType = "BULLISH" // Swept lows, reversed up
	SweepBearish SweepType = "BEARISH" // Swept highs, reversed down
)

// LiquiditySweep represents a detected stop hunt
type LiquiditySweep struct {
	Type             SweepType
	SweepPrice       float64   // How far the sweep went
	RecoveryPrice    float64   // Where price recovered to
	LiquidityLevel   float64   // The level that was swept (prior swing)
	SweepDepthPips   float64   // How many pips beyond the level
	RecoveryPercent  float64   // How much of the sweep was recovered
	SweepIndex       int       // Candle that made the sweep
	SweepTime        time.Time
	Strength         float64   // 0.0-1.0 sweep quality
	IsConfirmed      bool      // True if follow-through occurred
}

// LiquiditySweepConfig holds detection parameters
type LiquiditySweepConfig struct {
	LookbackBars      int     // How far to look for liquidity levels
	MinSweepPips      float64 // Minimum sweep beyond level (e.g., 5 pips)
	MaxSweepPips      float64 // Maximum sweep (too far = breakout, not sweep)
	MinRecoveryPct    float64 // Minimum recovery to confirm sweep (e.g., 0.5 = 50%)
	ConfirmationBars  int     // Bars to wait for confirmation
	PipSize           float64 // Pip size for the symbol (0.0001 for EUR, 0.01 for JPY)
}

// DefaultLiquiditySweepConfig returns sensible defaults
func DefaultLiquiditySweepConfig() LiquiditySweepConfig {
	return LiquiditySweepConfig{
		LookbackBars:     20,
		MinSweepPips:     3,
		MaxSweepPips:     30,
		MinRecoveryPct:   0.5,
		ConfirmationBars: 3,
		PipSize:          0.0001, // EURUSD default
	}
}

// DetectLiquiditySweeps finds all liquidity sweeps in candle history
func DetectLiquiditySweeps(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	if len(candles) < config.LookbackBars+5 {
		return nil
	}

	var sweeps []LiquiditySweep

	// Find bullish sweeps (sweep lows, reverse up)
	bullishSweeps := detectBullishSweeps(candles, config)
	sweeps = append(sweeps, bullishSweeps...)

	// Find bearish sweeps (sweep highs, reverse down)
	bearishSweeps := detectBearishSweeps(candles, config)
	sweeps = append(sweeps, bearishSweeps...)

	return sweeps
}

// detectBullishSweeps finds sweeps below prior lows that reverse up
func detectBullishSweeps(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	var sweeps []LiquiditySweep
	n := len(candles)

	for i := config.LookbackBars; i < n-1; i++ {
		current := candles[i]

		// Find the lowest low in lookback (this is the liquidity level)
		liquidityLevel := math.MaxFloat64
		for j := i - config.LookbackBars; j < i; j++ {
			if candles[j].Low < liquidityLevel {
				liquidityLevel = candles[j].Low
			}
		}

		// Check if current candle swept below this level
		sweepDepth := (liquidityLevel - current.Low) / config.PipSize
		if sweepDepth < config.MinSweepPips || sweepDepth > config.MaxSweepPips {
			continue
		}

		// Check for recovery - candle must close above the sweep
		// Bearish candle - weaker signal but still valid if big lower wick
		// Calculate wick range for bullish/bearish handling
		_ = current.Low - math.Min(current.Open, current.Close) // wick measurement
		
		totalSweepRange := current.Low - liquidityLevel
		recoveryPct := 0.0
		if totalSweepRange != 0 {
			// How much did we recover from the low back toward the liquidity level
			recoveryPct = (current.Close - current.Low) / (liquidityLevel - current.Low + math.Abs(totalSweepRange))
		}

		// Must recover significantly to be a valid sweep
		if current.Close <= liquidityLevel {
			// Price must close back above the swept level
			continue
		}

		// Calculate strength
		strength := calculateSweepStrength(current, liquidityLevel, sweepDepth, config)

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

		// Check for confirmation in following bars
		if i+config.ConfirmationBars < n {
			confirmed := true
			for j := i + 1; j <= i+config.ConfirmationBars && j < n; j++ {
				if candles[j].Low < current.Low {
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

// detectBearishSweeps finds sweeps above prior highs that reverse down
func detectBearishSweeps(candles []Candle, config LiquiditySweepConfig) []LiquiditySweep {
	var sweeps []LiquiditySweep
	n := len(candles)

	for i := config.LookbackBars; i < n-1; i++ {
		current := candles[i]

		// Find the highest high in lookback (this is the liquidity level)
		liquidityLevel := 0.0
		for j := i - config.LookbackBars; j < i; j++ {
			if candles[j].High > liquidityLevel {
				liquidityLevel = candles[j].High
			}
		}

		// Check if current candle swept above this level
		sweepDepth := (current.High - liquidityLevel) / config.PipSize
		if sweepDepth < config.MinSweepPips || sweepDepth > config.MaxSweepPips {
			continue
		}

		// Check for recovery - candle must close below the swept level
		if current.Close >= liquidityLevel {
			continue
		}

		// Calculate recovery percentage
		totalSweepRange := current.High - liquidityLevel
		recoveryPct := 0.0
		if totalSweepRange != 0 {
			recoveryPct = (current.High - current.Close) / (current.High - liquidityLevel + math.Abs(totalSweepRange))
		}

		strength := calculateSweepStrength(current, liquidityLevel, sweepDepth, config)

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

		// Check for confirmation
		if i+config.ConfirmationBars < n {
			confirmed := true
			for j := i + 1; j <= i+config.ConfirmationBars && j < n; j++ {
				if candles[j].High > current.High {
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

// calculateSweepStrength determines sweep quality (0.0-1.0)
func calculateSweepStrength(candle Candle, liquidityLevel, sweepDepthPips float64, config LiquiditySweepConfig) float64 {
	strength := 0.5 // Base

	// Deeper sweeps that still recover = stronger
	depthScore := sweepDepthPips / config.MaxSweepPips
	strength += depthScore * 0.2

	// Strong rejection wick = stronger
	bodySize := math.Abs(candle.Open - candle.Close)
	rangeSize := candle.High - candle.Low
	if rangeSize > 0 {
		wickRatio := 1.0 - (bodySize / rangeSize)
		strength += wickRatio * 0.2
	}

	// Close strongly in rejection direction = stronger
	if candle.Close > candle.Open { // Bullish
		closeStrength := (candle.Close - candle.Low) / rangeSize
		strength += closeStrength * 0.1
	} else { // Bearish
		closeStrength := (candle.High - candle.Close) / rangeSize
		strength += closeStrength * 0.1
	}

	return math.Min(strength, 1.0)
}

// GetRecentSweeps returns sweeps within the last N bars
func GetRecentSweeps(sweeps []LiquiditySweep, currentBar, lookback int) []LiquiditySweep {
	var recent []LiquiditySweep
	for _, sweep := range sweeps {
		if currentBar-sweep.SweepIndex <= lookback {
			recent = append(recent, sweep)
		}
	}
	return recent
}

// HasRecentBullishSweep checks for bullish sweep in recent bars
func HasRecentBullishSweep(candles []Candle, lookback int) bool {
	config := DefaultLiquiditySweepConfig()
	sweeps := DetectLiquiditySweeps(candles, config)
	
	currentBar := len(candles) - 1
	for _, sweep := range sweeps {
		if sweep.Type == SweepBullish && 
		   sweep.IsConfirmed && 
		   currentBar-sweep.SweepIndex <= lookback {
			return true
		}
	}
	return false
}

// HasRecentBearishSweep checks for bearish sweep in recent bars
func HasRecentBearishSweep(candles []Candle, lookback int) bool {
	config := DefaultLiquiditySweepConfig()
	sweeps := DetectLiquiditySweeps(candles, config)
	
	currentBar := len(candles) - 1
	for _, sweep := range sweeps {
		if sweep.Type == SweepBearish && 
		   sweep.IsConfirmed && 
		   currentBar-sweep.SweepIndex <= lookback {
			return true
		}
	}
	return false
}

// CalculateLiquiditySweepScore returns confluence score (0.0-1.0) for sweep alignment
func CalculateLiquiditySweepScore(candles []Candle, direction string) float64 {
	return CalculateLiquiditySweepScoreDebug(candles, direction, false)
}

// CalculateLiquiditySweepScoreDebug with optional debug output
func CalculateLiquiditySweepScoreDebug(candles []Candle, direction string, debug bool) float64 {
	if len(candles) < 25 {
		return 0.0
	}

	config := DefaultLiquiditySweepConfig()
	sweeps := DetectLiquiditySweeps(candles, config)
	
	if debug {
		fmt.Printf("      [SWEEP] Total sweeps detected: %d\n", len(sweeps))
	}
	
	if len(sweeps) == 0 {
		return 0.0
	}

	currentBar := len(candles) - 1
	score := 0.0

	for _, sweep := range sweeps {
		// Consider sweeps within last 30 bars
		barsAgo := currentBar - sweep.SweepIndex
		
		if debug {
			fmt.Printf("      [SWEEP] type=%v confirmed=%v barsAgo=%d strength=%.2f (need dir=%s)\n", 
				sweep.Type, sweep.IsConfirmed, barsAgo, sweep.Strength, direction)
		}
		
		if barsAgo > 30 {
			continue
		}

		// For LONG signals, we want recent bullish sweep (swept lows, reversed up)
		if direction == DirectionLong && sweep.Type == SweepBullish {
			sweepScore := sweep.Strength
			// More recent = higher score
			recencyBonus := 1.0 - (float64(barsAgo) / 30.0)
			sweepScore *= (1.0 + recencyBonus*0.3)
			// Bonus for confirmed sweeps
			if sweep.IsConfirmed {
				sweepScore *= 1.2
			}
			score = math.Max(score, sweepScore)
		}

		// For SHORT signals, we want recent bearish sweep (swept highs, reversed down)
		if direction == DirectionShort && sweep.Type == SweepBearish {
			sweepScore := sweep.Strength
			recencyBonus := 1.0 - (float64(barsAgo) / 30.0)
			sweepScore *= (1.0 + recencyBonus*0.3)
			if sweep.IsConfirmed {
				sweepScore *= 1.2
			}
			score = math.Max(score, sweepScore)
		}
	}

	return math.Min(score, 1.0)
}

// DetectHSSweep specifically checks if H&S shoulders were swept (high probability setup)
func DetectHSSweep(structure *DetectedStructure, candles []Candle) *LiquiditySweep {
	if structure == nil || len(candles) == 0 {
		return nil
	}

	config := DefaultLiquiditySweepConfig()
	config.LookbackBars = 5 // Only look near the shoulder

	// For bearish H&S, check if right shoulder high was swept then rejected
	if structure.PatternType == PatternHS {
		// The liquidity sits above the right shoulder
		rsHigh := structure.RightShoulderPrice
		
		// Check recent candles for sweep above RS
		for i := len(candles) - 5; i < len(candles); i++ {
			if i < 0 {
				continue
			}
			c := candles[i]
			
			sweepDepth := (c.High - rsHigh) / config.PipSize
			if sweepDepth >= config.MinSweepPips && sweepDepth <= config.MaxSweepPips {
				// Check if it recovered below
				if c.Close < rsHigh {
					return &LiquiditySweep{
						Type:            SweepBearish,
						SweepPrice:      c.High,
						RecoveryPrice:   c.Close,
						LiquidityLevel:  rsHigh,
						SweepDepthPips:  sweepDepth,
						SweepIndex:      i,
						SweepTime:       c.Timestamp,
						Strength:        0.8, // High confidence when aligned with H&S
						IsConfirmed:     true,
					}
				}
			}
		}
	}

	// For IHS, check if right shoulder low was swept then rejected
	if structure.PatternType == PatternIHS {
		rsLow := structure.RightShoulderPrice
		
		for i := len(candles) - 5; i < len(candles); i++ {
			if i < 0 {
				continue
			}
			c := candles[i]
			
			sweepDepth := (rsLow - c.Low) / config.PipSize
			if sweepDepth >= config.MinSweepPips && sweepDepth <= config.MaxSweepPips {
				if c.Close > rsLow {
					return &LiquiditySweep{
						Type:            SweepBullish,
						SweepPrice:      c.Low,
						RecoveryPrice:   c.Close,
						LiquidityLevel:  rsLow,
						SweepDepthPips:  sweepDepth,
						SweepIndex:      i,
						SweepTime:       c.Timestamp,
						Strength:        0.8,
						IsConfirmed:     true,
					}
				}
			}
		}
	}

	return nil
}
