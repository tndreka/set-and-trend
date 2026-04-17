package rules

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"set-and-trend/backend/internal/patterns"
)

// MinChecklistScore is the default minimum score (out of 20) to emit a setup.
// Tunable per-backtest via the SAF_MIN_SCORE environment variable.
var MinChecklistScore = 9

func init() {
	SetupBuilders[SAFChecklist] = buildSAFChecklistSetup
	SetupBuilders[SAFW1EmaRejection] = buildW1EmaRejectionSetup
	SetupBuilders[SAFD1RejectionStructure] = buildD1RejectionStructureSetup
	SetupBuilders[SAFW1RejectionD1AOI] = buildW1RejectionD1AOISetup
	if v := os.Getenv("SAF_MIN_SCORE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 16 {
			MinChecklistScore = n
		}
	}
}

// ChecklistResult holds the breakdown of which items passed.
type ChecklistResult struct {
	Score     int
	Max       int
	Direction string
	Items     map[string]bool
}

func buildSAFChecklistSetup(ctx EvalContext) (*Setup, error) {
	if len(ctx.Candles) < 250 {
		return nil, nil
	}
	candle, ind, ok := ctx.Latest()
	if !ok {
		return nil, fmt.Errorf("saf_checklist: empty eval context")
	}

	// Session gate: only trade during London/NY.
	if !InKillzone(candle) {
		return nil, nil
	}

	h4Candles := toPatternCandles(ctx.Candles)
	symbol := ctx.Symbol
	isJPY := IsJPYPair(symbol)

	// Determine direction from majority of "in favor" votes across timeframes.
	direction := deriveDirection(ind)
	if direction == "" {
		return nil, nil
	}

	items := make(map[string]bool, 16)
	score := 0

	check := func(name string, pass bool) {
		items[name] = pass
		if pass {
			score++
		}
	}

	// ---- Weekly (W1) — real W1 candles ----
	// Removed w1_aoi (-3.8% lift, anti-signal) and w1_structure_rejection (-3.0% lift)
	w1Candles := toPatternCandles(ctx.W1Candles)
	check("w1_in_favor", checkInFavor(ind.W1EMA50, ind.W1EMA200, direction))
	check("w1_touching_ema", ind.W1EMA50 > 0 && NearEMA(candle.Close, ind.W1EMA50, 0.005))
	if w1Latest, ok := ctx.LatestW1(); ok {
		check("w1_candle_rejection", IsCandleRejection(w1Latest, direction))
	} else {
		check("w1_candle_rejection", false)
	}
	if len(w1Candles) >= 30 {
		check("w1_pattern", checkPattern(w1Candles, symbol, direction, 30))
	} else {
		check("w1_pattern", false)
	}

	// ---- Daily (D1) — real D1 candles ----
	d1Candles := toPatternCandles(ctx.D1Candles)
	check("d1_in_favor", checkInFavor(ind.D1EMA50, ind.D1EMA200, direction))
	if len(d1Candles) >= 30 {
		check("d1_aoi", checkAOI(d1Candles, symbol, direction))
	} else {
		check("d1_aoi", false)
	}
	check("d1_touching_ema", ind.D1EMA50 > 0 && NearEMA(candle.Close, ind.D1EMA50, 0.005))
	if d1Latest, ok := ctx.LatestD1(); ok {
		check("d1_candle_rejection", IsCandleRejection(d1Latest, direction))
		check("d1_structure_rejection", checkStructureRejection(d1Candles, d1Latest, direction, 15))
	} else {
		check("d1_candle_rejection", false)
		check("d1_structure_rejection", false)
	}
	if len(d1Candles) >= 40 {
		check("d1_pattern", checkPattern(d1Candles, symbol, direction, 40))
	} else {
		check("d1_pattern", false)
	}

	// ---- 4H ----
	// Removed h4_structure_rejection (0 trades, never fires)
	check("h4_in_favor", checkInFavor(ind.EMA50, ind.EMA200, direction))
	check("h4_ema_touch", ind.EMA50 > 0 && NearEMA(candle.Close, ind.EMA50, 0.008))
	check("h4_candle_rejection", IsCandleRejection(candle, direction))
	check("h4_pattern", checkPattern(h4Candles, symbol, direction, 30))

	// ---- Bonus ----
	// Removed shift_of_structure (-9.9% lift, anti-signal)
	check("psych_level", NearPsychLevel(candle.Close, isJPY))
	if len(ctx.Candles) >= 2 {
		prev := ctx.Candles[len(ctx.Candles)-2]
		check("engulfing", IsEngulfing(prev, candle, direction))
	} else {
		check("engulfing", false)
	}

	if score < MinChecklistScore {
		return nil, nil
	}

	// Build the setup geometry.
	entry := candle.Close
	sl := findSL(h4Candles, candle, direction, symbol)
	if sl == 0 {
		return nil, nil
	}
	risk := math.Abs(entry - sl)
	if risk <= 0 {
		return nil, nil
	}

	tp := findTP(h4Candles, entry, direction, symbol, risk)
	rr := computeRR(direction, entry, sl, tp)
	minRR := 2.0
	if v := os.Getenv("SAF_MIN_RR"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 1.0 && f <= 5.0 {
			minRR = f
		}
	}
	if rr < minRR {
		return nil, nil
	}

	itemNames := make([]string, 0, score)
	for k, v := range items {
		if v {
			itemNames = append(itemNames, k)
		}
	}

	return &Setup{
		Direction:      direction,
		Entry:          entry,
		StopLoss:       sl,
		TakeProfit:     tp,
		RR:             rr,
		Confidence:     float64(score) / 16.0,
		Reason:         fmt.Sprintf("SAF checklist %d/16: %s", score, strings.Join(itemNames, ",")),
		ChecklistScore: score,
		ChecklistItems: items,
	}, nil
}

// deriveDirection picks LONG or SHORT based on majority of W1/D1/H4 "in favor".
// Returns "" if no majority (2+ must agree).
func deriveDirection(ind Indicators) string {
	bullish, bearish := 0, 0
	if ind.W1EMA50 > 0 && ind.W1EMA200 > 0 {
		if ind.W1EMA50 > ind.W1EMA200 {
			bullish++
		} else {
			bearish++
		}
	}
	if ind.D1EMA50 > 0 && ind.D1EMA200 > 0 {
		if ind.D1EMA50 > ind.D1EMA200 {
			bullish++
		} else {
			bearish++
		}
	}
	if ind.EMA50 > 0 && ind.EMA200 > 0 {
		if ind.EMA50 > ind.EMA200 {
			bullish++
		} else {
			bearish++
		}
	}
	if bullish >= 2 {
		return "LONG"
	}
	if bearish >= 2 {
		return "SHORT"
	}
	return ""
}

func checkInFavor(ema50, ema200 float64, direction string) bool {
	if ema50 == 0 || ema200 == 0 {
		return false
	}
	if direction == "LONG" {
		return ema50 > ema200
	}
	return ema50 < ema200
}

func checkAOI(candles []patterns.Candle, symbol, direction string) bool {
	if len(candles) < 50 {
		return false
	}
	cfg := patterns.DefaultZoneConfig(symbol)
	zones := patterns.DetectSupplyDemandZones(candles, cfg)
	active := patterns.GetActiveZones(candles, zones, cfg.ZoneFreshnessBars)
	latest := candles[len(candles)-1]
	for _, z := range active {
		if direction == "LONG" && z.Type == patterns.ZoneDemand {
			if latest.Low <= z.ProximalPrice && latest.Close > z.DistalPrice {
				return true
			}
		}
		if direction == "SHORT" && z.Type == patterns.ZoneSupply {
			if latest.High >= z.ProximalPrice && latest.Close < z.DistalPrice {
				return true
			}
		}
	}
	return false
}

func checkStructureRejection(candles []patterns.Candle, c Candle, direction string, lookback int) bool {
	if len(candles) < lookback {
		return false
	}
	window := candles[len(candles)-lookback:]
	if direction == "LONG" {
		swings := patterns.FindSwingLows(window, 5)
		for _, s := range swings {
			dist := math.Abs(c.Close - s.Price)
			rng := c.High - c.Low
			if rng > 0 && dist < rng*3 {
				return true
			}
		}
	} else {
		swings := patterns.FindSwingHighs(window, 5)
		for _, s := range swings {
			dist := math.Abs(c.Close - s.Price)
			rng := c.High - c.Low
			if rng > 0 && dist < rng*3 {
				return true
			}
		}
	}
	return false
}

func checkPattern(candles []patterns.Candle, symbol, direction string, lookback int) bool {
	if len(candles) < lookback {
		return false
	}
	window := candles[len(candles)-lookback:]
	if direction == "LONG" {
		if d := patterns.DetectDoubleBottom(window, lookback, symbol); d != nil {
			return true
		}
		if d := patterns.DetectTripleBottom(window, lookback, symbol); d != nil {
			return true
		}
		if d := patterns.DetectInverseHeadAndShoulders(window, lookback); d != nil {
			return true
		}
	} else {
		if d := patterns.DetectDoubleTop(window, lookback, symbol); d != nil {
			return true
		}
		if d := patterns.DetectTripleTop(window, lookback, symbol); d != nil {
			return true
		}
		if d := patterns.DetectHeadAndShoulders(window, lookback); d != nil {
			return true
		}
	}
	return false
}

func checkSOS(candles []patterns.Candle, symbol, direction string) bool {
	lookback := 20
	if direction == "LONG" {
		return patterns.HasRecentBullishBOS(candles, lookback, symbol)
	}
	return patterns.HasRecentBearishBOS(candles, lookback, symbol)
}

func atr14(candles []patterns.Candle) float64 {
	n := len(candles)
	if n < 15 {
		return 0
	}
	sum := 0.0
	for i := n - 14; i < n; i++ {
		c := candles[i]
		prev := candles[i-1]
		tr := c.High - c.Low
		if d := math.Abs(c.High - prev.Close); d > tr {
			tr = d
		}
		if d := math.Abs(c.Low - prev.Close); d > tr {
			tr = d
		}
		sum += tr
	}
	return sum / 14.0
}

func findSL(candles []patterns.Candle, c Candle, direction, symbol string) float64 {
	minDist := atr14(candles) * 0.5
	if direction == "LONG" {
		swings := patterns.FindSwingLows(candles, 5)
		for i := len(swings) - 1; i >= 0; i-- {
			dist := c.Close - swings[i].Price
			if dist >= minDist {
				return swings[i].Price
			}
		}
	} else {
		swings := patterns.FindSwingHighs(candles, 5)
		for i := len(swings) - 1; i >= 0; i-- {
			dist := swings[i].Price - c.Close
			if dist >= minDist {
				return swings[i].Price
			}
		}
	}
	return 0
}

// buildW1EmaRejectionSetup fires when w1_touching_ema + w1_candle_rejection
// are both true. Backtested: 54.4% WR, +1.36R, stable across all eras.
func buildW1EmaRejectionSetup(ctx EvalContext) (*Setup, error) {
	candle, ind, ok := ctx.Latest()
	if !ok || len(ctx.Candles) < 250 {
		return nil, nil
	}
	if !InKillzone(candle) {
		return nil, nil
	}
	direction := deriveDirection(ind)
	if direction == "" {
		return nil, nil
	}
	if ind.W1EMA50 <= 0 || !NearEMA(candle.Close, ind.W1EMA50, 0.005) {
		return nil, nil
	}
	w1Latest, ok := ctx.LatestW1()
	if !ok || !IsCandleRejection(w1Latest, direction) {
		return nil, nil
	}
	return buildConfluenceSetup(ctx, candle, direction, "w1_touching_ema + w1_candle_rejection")
}

// buildD1RejectionStructureSetup fires when d1_candle_rejection +
// d1_structure_rejection are both true. Backtested: 44.1% WR, +1.07R.
func buildD1RejectionStructureSetup(ctx EvalContext) (*Setup, error) {
	candle, ind, ok := ctx.Latest()
	if !ok || len(ctx.Candles) < 250 {
		return nil, nil
	}
	if !InKillzone(candle) {
		return nil, nil
	}
	direction := deriveDirection(ind)
	if direction == "" {
		return nil, nil
	}
	d1Latest, ok := ctx.LatestD1()
	if !ok || !IsCandleRejection(d1Latest, direction) {
		return nil, nil
	}
	d1Candles := toPatternCandles(ctx.D1Candles)
	if !checkStructureRejection(d1Candles, d1Latest, direction, 15) {
		return nil, nil
	}
	return buildConfluenceSetup(ctx, candle, direction, "d1_candle_rejection + d1_structure_rejection")
}

// buildW1RejectionD1AOISetup fires when w1_candle_rejection + d1_aoi are
// both true. Backtested: 36.6% WR, +0.68R.
func buildW1RejectionD1AOISetup(ctx EvalContext) (*Setup, error) {
	candle, ind, ok := ctx.Latest()
	if !ok || len(ctx.Candles) < 250 {
		return nil, nil
	}
	if !InKillzone(candle) {
		return nil, nil
	}
	direction := deriveDirection(ind)
	if direction == "" {
		return nil, nil
	}
	w1Latest, ok := ctx.LatestW1()
	if !ok || !IsCandleRejection(w1Latest, direction) {
		return nil, nil
	}
	d1Candles := toPatternCandles(ctx.D1Candles)
	if len(d1Candles) < 50 || !checkAOI(d1Candles, ctx.Symbol, direction) {
		return nil, nil
	}
	return buildConfluenceSetup(ctx, candle, direction, "w1_candle_rejection + d1_aoi")
}

// buildConfluenceSetup is the shared geometry builder for the top-3 strategies.
// ATR SL + RR >= 2.5 gate.
func buildConfluenceSetup(ctx EvalContext, candle Candle, direction, confluence string) (*Setup, error) {
	h4Candles := toPatternCandles(ctx.Candles)
	entry := candle.Close
	sl := findSL(h4Candles, candle, direction, ctx.Symbol)
	if sl == 0 {
		return nil, nil
	}
	risk := math.Abs(entry - sl)
	if risk <= 0 {
		return nil, nil
	}
	tp := findTP(h4Candles, entry, direction, ctx.Symbol, risk)
	rr := computeRR(direction, entry, sl, tp)
	if rr < 2.5 {
		return nil, nil
	}
	return &Setup{
		Direction:  direction,
		Entry:      entry,
		StopLoss:   sl,
		TakeProfit: tp,
		RR:         rr,
		Confidence: 0.8,
		Reason:     fmt.Sprintf("Top-3 confluence: %s", confluence),
	}, nil
}

func findTP(candles []patterns.Candle, entry float64, direction, symbol string, risk float64) float64 {
	cfg := patterns.DefaultZoneConfig(symbol)
	zones := patterns.DetectSupplyDemandZones(candles, cfg)
	active := patterns.GetActiveZones(candles, zones, cfg.ZoneFreshnessBars)

	if direction == "LONG" {
		best := math.MaxFloat64
		for _, z := range active {
			if z.Type == patterns.ZoneSupply && z.ProximalPrice > entry {
				if z.ProximalPrice < best {
					best = z.ProximalPrice
				}
			}
		}
		if best != math.MaxFloat64 && (best-entry) >= 2*risk {
			return best
		}
		return entry + 2*risk
	}
	best := 0.0
	for _, z := range active {
		if z.Type == patterns.ZoneDemand && z.ProximalPrice < entry {
			if z.ProximalPrice > best {
				best = z.ProximalPrice
			}
		}
	}
	if best > 0 && (entry-best) >= 2*risk {
		return best
	}
	return entry - 2*risk
}
