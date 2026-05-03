package rules

import (
	"fmt"

	"github.com/shopspring/decimal"

	"set-and-trend/backend/internal/patterns"
)

func init() {
	SetupBuilders[H4TrendFollowing] = buildH4TrendFollowingSetup
}

// buildH4TrendFollowingSetup fires when the H4 bullish-trend conditions hold
// on the latest bar. Setup geometry:
//
//	Entry  = latest close
//	SL     = most recent swing-low low (lookback 5 bars)
//	TP     = entry + 2 * (entry - SL)
//
// Returns nil if any required conditions fail or no swing low is available.
func buildH4TrendFollowingSetup(ctx EvalContext) (*Setup, error) {
	candle, ind, ok := ctx.Latest()
	if !ok {
		return nil, fmt.Errorf("h4_trend_following: empty eval context")
	}

	// Diagnostic gates: session + volatility + D1 trend.
	if !InKillzone(candle) {
		return nil, nil
	}
	if !AtrAboveFloor(ctx.Candles, MinAtrPrice(ctx.Symbol)) {
		return nil, nil
	}

	res, err := EvaluateRule(H4TrendFollowing, candle, ind)
	if err != nil {
		return nil, fmt.Errorf("h4_trend_following: evaluate: %w", err)
	}
	if res.Result != "PASS" {
		return nil, nil
	}

	// Need at least ~30 bars to find a usable swing low with lookback=5.
	if len(ctx.Candles) < 30 {
		return nil, nil
	}

	patternCandles := toPatternCandles(ctx.Candles)
	swings := patterns.FindSwingLows(patternCandles, 5)
	if len(swings) == 0 {
		return nil, nil
	}

	// Use the most recent swing low strictly below the latest close.
	var sl float64
	for i := len(swings) - 1; i >= 0; i-- {
		if swings[i].Price < candle.Close {
			sl = swings[i].Price
			break
		}
	}
	if sl == 0 {
		return nil, nil
	}

	entry := candle.Close
	risk := entry - sl
	if risk <= 0 {
		return nil, nil
	}
	tp := entry + 2*risk
	rr := computeRR("LONG", entry, sl, tp)

	if !D1TrendAligned(ind, "LONG") {
		return nil, nil
	}

	return &Setup{
		Direction:  "LONG",
		Entry:      decimal.NewFromFloat(entry),
		StopLoss:   decimal.NewFromFloat(sl),
		TakeProfit: decimal.NewFromFloat(tp),
		RR:         decimal.NewFromFloat(rr),
		Confidence: decimal.NewFromFloat(res.Confidence),
		Reason:     "EMA50>EMA200, close above EMA50, EMA50 rising, D1 trend aligned — long on prior swing low",
	}, nil
}

// toPatternCandles adapts the rules.Candle slice to the patterns.Candle slice
// expected by the swing/zone helpers. The patterns package needs Index +
// Timestamp + OHLC; we don't carry volume here.
func toPatternCandles(in []Candle) []patterns.Candle {
	out := make([]patterns.Candle, len(in))
	for i, c := range in {
		out[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUTC,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
		}
	}
	return out
}
