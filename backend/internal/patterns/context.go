package patterns

import "math"

// DetermineMarketContext analyzes the market state
func DetermineMarketContext(candles []Candle) MarketContext {
	ctx := MarketContext{
		Trend:            TrendSideways,
		VolatilityRegime: VolatilityNormal,
	}

	if len(candles) < 50 {
		return ctx
	}

	// Calculate EMAs
	ema50 := CalculateEMA(candles, 50)
	ema200 := 0.0
	if len(candles) >= 200 {
		ema200 = CalculateEMA(candles, 200)
	} else {
		ema200 = ema50 // Fallback if not enough data
	}

	current := candles[len(candles)-1]

	// Determine trend
	if current.Close > ema50 && ema50 > ema200 {
		ctx.Trend = TrendStrongUp
	} else if current.Close > ema50 {
		ctx.Trend = TrendWeakUp
	} else if current.Close < ema50 && ema50 < ema200 {
		ctx.Trend = TrendStrongDown
	} else if current.Close < ema50 {
		ctx.Trend = TrendWeakDown
	} else {
		ctx.Trend = TrendSideways
	}

	// Determine volatility regime
	recentVolatility := CalculateVolatility(candles[len(candles)-20:], 20)
	
	historicalStart := len(candles) - 100
	if historicalStart < 0 {
		historicalStart = 0
	}
	historicalVolatility := CalculateVolatility(candles[historicalStart:], 20)

	if historicalVolatility > 0 {
		ratio := recentVolatility / historicalVolatility
		if ratio < 0.7 {
			ctx.VolatilityRegime = VolatilityCompression
		} else if ratio > 1.3 {
			ctx.VolatilityRegime = VolatilityExpansion
		} else {
			ctx.VolatilityRegime = VolatilityNormal
		}
	}

	// Distance from EMA200
	if ema200 > 0 {
		ctx.DistanceFromEMA200 = (current.Close - ema200) / ema200
	}

	// Count recent swings
	recentCandles := candles
	if len(candles) > 20 {
		recentCandles = candles[len(candles)-20:]
	}
	recentHighs := FindSwingHighs(recentCandles, 2)
	recentLows := FindSwingLows(recentCandles, 2)
	ctx.RecentSwings = len(recentHighs) + len(recentLows)

	return ctx
}

// CalculateContextBonus calculates the confidence adjustment based on market context
// Same pattern has VERY different edge depending on market state
func CalculateContextBonus(s *DetectedStructure, ctx MarketContext) float64 {
	bonus := 0.0

	if s.PatternType == PatternHS {
		// Bearish H&S context bonuses
		bonus = calculateHSContextBonus(ctx)
	} else if s.PatternType == PatternIHS {
		// Bullish IHS context bonuses (inverse logic)
		bonus = calculateIHSContextBonus(ctx)
	}

	return bonus
}

// calculateHSContextBonus for bearish H&S patterns
func calculateHSContextBonus(ctx MarketContext) float64 {
	bonus := 0.0

	// Distribution after compression + downtrend = STRONG signal
	if ctx.Trend == TrendStrongDown && ctx.VolatilityRegime == VolatilityExpansion {
		bonus += 0.20
	}

	// H&S in strong uptrend = likely trap, reduce confidence
	if ctx.Trend == TrendStrongUp && ctx.VolatilityRegime == VolatilityNormal {
		bonus -= 0.15
	}

	// H&S at new highs (far above EMA200) = weak reversal
	if ctx.DistanceFromEMA200 > 0.05 {
		bonus -= 0.10
	}

	// H&S after compression break = strong
	if ctx.VolatilityRegime == VolatilityExpansion {
		bonus += 0.05
	}

	// Too many swings = choppy market, weak pattern
	if ctx.RecentSwings > 8 {
		bonus -= 0.05
	}

	// H&S in downtrend = trend continuation, stronger
	if ctx.Trend == TrendWeakDown || ctx.Trend == TrendStrongDown {
		bonus += 0.10
	}

	return bonus
}

// calculateIHSContextBonus for bullish IHS patterns
func calculateIHSContextBonus(ctx MarketContext) float64 {
	bonus := 0.0

	// Accumulation after compression + uptrend = STRONG signal
	if ctx.Trend == TrendStrongUp && ctx.VolatilityRegime == VolatilityExpansion {
		bonus += 0.20
	}

	// IHS in strong downtrend = likely trap
	if ctx.Trend == TrendStrongDown && ctx.VolatilityRegime == VolatilityNormal {
		bonus -= 0.15
	}

	// IHS at new lows (far below EMA200) = weak reversal
	if ctx.DistanceFromEMA200 < -0.05 {
		bonus -= 0.10
	}

	// IHS after compression break
	if ctx.VolatilityRegime == VolatilityExpansion {
		bonus += 0.05
	}

	// Choppy market
	if ctx.RecentSwings > 8 {
		bonus -= 0.05
	}

	// IHS in uptrend = trend continuation
	if ctx.Trend == TrendWeakUp || ctx.Trend == TrendStrongUp {
		bonus += 0.10
	}

	return bonus
}

// CalculateEMA calculates Exponential Moving Average
func CalculateEMA(candles []Candle, period int) float64 {
	if len(candles) < period {
		if len(candles) == 0 {
			return 0.0
		}
		return candles[len(candles)-1].Close
	}

	multiplier := 2.0 / float64(period+1)
	ema := candles[0].Close

	for i := 1; i < len(candles); i++ {
		ema = (candles[i].Close * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// CalculateVolatility calculates standard deviation of log returns
func CalculateVolatility(candles []Candle, period int) float64 {
	if len(candles) < 2 {
		return 0.0
	}

	// Use available data if less than period
	usePeriod := period
	if len(candles) < period {
		usePeriod = len(candles)
	}

	// Calculate log returns
	returns := make([]float64, usePeriod-1)
	startIdx := len(candles) - usePeriod
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx + 1; i < len(candles); i++ {
		if candles[i-1].Close > 0 {
			returns[i-startIdx-1] = math.Log(candles[i].Close / candles[i-1].Close)
		}
	}

	if len(returns) == 0 {
		return 0.0
	}

	// Calculate mean
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate variance
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance)
}
