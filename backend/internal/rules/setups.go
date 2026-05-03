package rules

import "github.com/shopspring/decimal"

// Setup is the trade plan a strategy produces when its rule triggers on a
// fresh candle. The signal_evaluator persists this into setup_signals so
// the OpenClaw setup-watcher can push it to Telegram.
//
// Per CLAUDE.md: prices/pips/money are decimal.Decimal end-to-end. Builders
// take in float64 candle data (the ingest path is still float64) and convert
// at this boundary; downstream consumers (signal_repo, telegram_alerter) work
// in decimal.
type Setup struct {
	Direction      string          // "LONG" or "SHORT"
	Entry          decimal.Decimal // intended entry price
	StopLoss       decimal.Decimal
	TakeProfit     decimal.Decimal
	RR             decimal.Decimal // |TP-Entry| / |Entry-SL|
	Confidence     decimal.Decimal // 0..1
	Reason         string          // human-readable explanation, stored in setup_signals.details
	ChecklistScore int             // raw score out of 14 (checklist strategies only)
	ChecklistItems map[string]bool // per-item pass/fail (checklist strategies only)
}

// EvalContext is the rich evaluation context passed to a SetupBuilder. Unlike
// the simple per-bar (Candle, Indicators) signature used by EvaluateRule, this
// gives the builder access to the full recent history — required for
// pattern-based and session-based rules.
//
// Candles is chronological with the latest bar at index len-1. Indicators is
// a parallel slice (Indicators[i] corresponds to Candles[i]).
type EvalContext struct {
	Symbol     string
	Candles    []Candle
	Indicators []Indicators

	// Higher-timeframe candles for multi-TF strategies (optional).
	// Non-MTF builders ignore these.
	D1Candles    []Candle
	D1Indicators []Indicators
	W1Candles    []Candle
	W1Indicators []Indicators
}

// Latest returns the most recent candle/indicator pair for the builder.
func (e EvalContext) Latest() (Candle, Indicators, bool) {
	n := len(e.Candles)
	if n == 0 || len(e.Indicators) != n {
		return Candle{}, Indicators{}, false
	}
	return e.Candles[n-1], e.Indicators[n-1], true
}

// LatestD1 returns the most recent D1 candle, or ok=false if unavailable.
func (e EvalContext) LatestD1() (Candle, bool) {
	if len(e.D1Candles) == 0 {
		return Candle{}, false
	}
	return e.D1Candles[len(e.D1Candles)-1], true
}

// LatestW1 returns the most recent W1 candle, or ok=false if unavailable.
func (e EvalContext) LatestW1() (Candle, bool) {
	if len(e.W1Candles) == 0 {
		return Candle{}, false
	}
	return e.W1Candles[len(e.W1Candles)-1], true
}

// SetupBuilder is the unified evaluator signature for strategy rules. It
// returns a *Setup if the strategy fires on the latest bar, or nil otherwise.
// An error is returned only for unrecoverable problems (missing data,
// computation failures); a "no setup right now" outcome is (nil, nil).
type SetupBuilder func(ctx EvalContext) (*Setup, error)

// SetupBuilders is the registry of setup builders keyed by RuleCode. Builders
// are registered via init() functions in their respective rule files
// (h4_trend_following.go, h4_supply_demand.go, h4_london_breakout.go).
var SetupBuilders = map[RuleCode]SetupBuilder{}

// ---------------------------------------------------------------------------
// Reusable gate functions — each returns true to ALLOW, false to REJECT.
// ---------------------------------------------------------------------------

// InKillzone returns true if the latest H4 bar opened during London (07–10 UTC),
// NY (12–15 UTC), or extended NY (16–19 UTC). Asian (00, 04) and off-hours (20)
// bars are rejected. On H4 the open-hour is sufficient since each bar covers a
// 4-hour block.
func InKillzone(c Candle) bool {
	h := c.TimestampUTC.Hour()
	return h == 8 || h == 12 || h == 16
}

// AtrAboveFloor computes the ATR-14 of the last 14 bars and checks whether it
// exceeds minPips (in price terms, not pip units — caller converts). Rejects
// dead-market signals. Returns true if ATR is above the floor or if there are
// not enough bars (benefit of the doubt).
func AtrAboveFloor(candles []Candle, minPrice float64) bool {
	period := 14
	n := len(candles)
	if n < period+1 {
		return true
	}
	atr := 0.0
	for i := n - period; i < n; i++ {
		prev := candles[i-1]
		c := candles[i]
		tr := c.High - c.Low
		if d := abs(c.High - prev.Close); d > tr {
			tr = d
		}
		if d := abs(c.Low - prev.Close); d > tr {
			tr = d
		}
		atr += tr
	}
	atr /= float64(period)
	return atr >= minPrice
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// MinAtrPrice returns the minimum ATR threshold in price terms for a symbol.
// 15 pips for standard pairs, 15 pips (= 0.15) for JPY crosses.
func MinAtrPrice(symbol string) float64 {
	if len(symbol) >= 6 && symbol[3:6] == "JPY" {
		return 0.15
	}
	return 0.0015
}

// D1TrendAligned returns true when the D1 EMA50 trend agrees with the proposed
// direction. Skipped when D1 EMAs are not populated (returns true).
func D1TrendAligned(ind Indicators, direction string) bool {
	if ind.D1EMA50 == 0 || ind.D1EMA200 == 0 {
		return true // not available, don't block
	}
	if direction == "LONG" {
		return ind.D1EMA50 > ind.D1EMA200
	}
	return ind.D1EMA50 < ind.D1EMA200
}

// computeRR returns the risk:reward ratio for a setup. Returns 0 if the
// stop distance is zero (caller should reject the setup).
func computeRR(direction string, entry, sl, tp float64) float64 {
	risk := entry - sl
	reward := tp - entry
	if direction == "SHORT" {
		risk = sl - entry
		reward = entry - tp
	}
	if risk <= 0 {
		return 0
	}
	return reward / risk
}
