package rules

// Setup is the trade plan a strategy produces when its rule triggers on a
// fresh candle. The signal_evaluator persists this into setup_signals so
// the OpenClaw setup-watcher can push it to Telegram.
type Setup struct {
	Direction  string  // "LONG" or "SHORT"
	Entry      float64 // intended entry price
	StopLoss   float64
	TakeProfit float64
	RR         float64 // |TP-Entry| / |Entry-SL|
	Confidence float64 // 0..1
	Reason     string  // human-readable explanation, stored in setup_signals.details
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
}

// Latest returns the most recent candle/indicator pair for the builder.
func (e EvalContext) Latest() (Candle, Indicators, bool) {
	n := len(e.Candles)
	if n == 0 || len(e.Indicators) != n {
		return Candle{}, Indicators{}, false
	}
	return e.Candles[n-1], e.Indicators[n-1], true
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
