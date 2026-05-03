package rules

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func init() {
	SetupBuilders[H4LondonBreakout] = buildH4LondonBreakoutSetup
}

// buildH4LondonBreakoutSetup fires on the 08:00 UTC H4 bar (London open
// killzone) when price breaks the range built by the prior Asian bars
// (the 00:00 and 04:00 UTC bars on the same day, plus the 20:00 UTC bar
// from the previous day for completeness).
//
// Setup geometry:
//
//	LONG  if current high > Asian range high
//	  Entry = current close
//	  SL    = Asian range low
//	  TP    = entry + 1.5 * range_size
//
//	SHORT if current low < Asian range low
//	  Entry = current close
//	  SL    = Asian range high
//	  TP    = entry - 1.5 * range_size
func buildH4LondonBreakoutSetup(ctx EvalContext) (*Setup, error) {
	if len(ctx.Candles) < 4 {
		return nil, nil
	}
	candle, _, ok := ctx.Latest()
	if !ok {
		return nil, fmt.Errorf("h4_london_breakout: empty eval context")
	}

	// Only fires on the 08:00 UTC bar (London open killzone for H4 grid).
	if candle.TimestampUTC.Hour() != 8 {
		return nil, nil
	}

	// Diagnostic gate: volatility floor.
	if !AtrAboveFloor(ctx.Candles, MinAtrPrice(ctx.Symbol)) {
		return nil, nil
	}

	// Asian range = 20:00 (prev day) + 00:00 + 04:00 bars on the current day.
	// These are the 3 bars immediately before the latest one on the H4 grid.
	n := len(ctx.Candles)
	asianBars := ctx.Candles[n-4 : n-1]
	rangeHigh := asianBars[0].High
	rangeLow := asianBars[0].Low
	for _, b := range asianBars[1:] {
		if b.High > rangeHigh {
			rangeHigh = b.High
		}
		if b.Low < rangeLow {
			rangeLow = b.Low
		}
	}
	rangeSize := rangeHigh - rangeLow
	if rangeSize <= 0 {
		return nil, nil
	}

	_, ind, _ := ctx.Latest()

	// Long breakout: current bar pushes through range high.
	if candle.High > rangeHigh && D1TrendAligned(ind, "LONG") {
		entry := candle.Close
		sl := rangeLow
		tp := entry + 1.5*rangeSize
		risk := entry - sl
		if risk <= 0 {
			return nil, nil
		}
		return &Setup{
			Direction:  "LONG",
			Entry:      decimal.NewFromFloat(entry),
			StopLoss:   decimal.NewFromFloat(sl),
			TakeProfit: decimal.NewFromFloat(tp),
			RR:         decimal.NewFromFloat(computeRR("LONG", entry, sl, tp)),
			Confidence: decimal.NewFromFloat(0.6),
			Reason:     "London open broke Asian range high",
		}, nil
	}

	// Short breakout: current bar breaks below range low.
	if candle.Low < rangeLow && D1TrendAligned(ind, "SHORT") {
		entry := candle.Close
		sl := rangeHigh
		tp := entry - 1.5*rangeSize
		risk := sl - entry
		if risk <= 0 {
			return nil, nil
		}
		return &Setup{
			Direction:  "SHORT",
			Entry:      decimal.NewFromFloat(entry),
			StopLoss:   decimal.NewFromFloat(sl),
			TakeProfit: decimal.NewFromFloat(tp),
			RR:         decimal.NewFromFloat(computeRR("SHORT", entry, sl, tp)),
			Confidence: decimal.NewFromFloat(0.6),
			Reason:     "London open broke Asian range low",
		}, nil
	}

	return nil, nil
}
