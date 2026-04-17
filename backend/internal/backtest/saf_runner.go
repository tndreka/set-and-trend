package backtest

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/rules"
	"set-and-trend/backend/internal/synthesis"
)

// SaF (Set-and-Forget) backtest runner: walks the H4 history per symbol and
// executes the 3 live strategies (`SetupBuilders`) instead of the legacy
// detector-consensus engine used by Run(). Outcomes are resolved by walking
// forward candle-by-candle against each setup's SL/TP. PnL is expressed in R
// multiples, not currency — currency conversion is handled by the live
// trading layer, not here.

const (
	safWarmupBars     = 200 // matches EMA200 warmup in signal_evaluator.go
	safMaxHoldBars    = 60  // ~10 trading days on H4; timeout beyond this
	safMinCandles     = safWarmupBars + 50
	safMaxHistoryBars = 0 // 0 = no cap; set for debugging
)

// SaFStrategyStats captures per-strategy outcome counts and risk-adjusted metrics.
type SaFStrategyStats struct {
	Strategy     string  `json:"strategy"`
	Trades       int     `json:"trades"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	Timeouts     int     `json:"timeouts"`
	WinRate      float64 `json:"win_rate"`
	ExpectancyR  float64 `json:"expectancy_r"`  // average R per trade
	TotalPnlR    float64 `json:"total_pnl_r"`
	MaxDrawdownR float64 `json:"max_drawdown_r"` // peak-to-trough in R
}

// SaFResults is the JSON shape persisted to disk for iteration comparison.
type SaFResults struct {
	Iter      int                          `json:"iter"`
	IterName  string                       `json:"iter_name"`
	Timestamp time.Time                    `json:"timestamp"`
	Symbols   []string                     `json:"symbols"`
	Per       map[string]*SaFStrategyStats `json:"per_strategy"`
	Aggregate SaFStrategyStats             `json:"aggregate"`
}

// safTrade is the in-memory record for each resolved setup.
type safTrade struct {
	strategy       string
	symbol         string
	opened         time.Time
	closed         time.Time
	result         string // "win" | "loss" | "timeout"
	rMult          float64
	checklistScore int
	checklistItems map[string]bool
	direction      string
	entry, sl, tp  float64
	rr             float64
}

// RunSaF executes the SaF backtest and writes results to outPath.
// If allSymbols is true, every enabled strategy's builder is evaluated against
// every pair in candles_h4 (research mode). Otherwise each builder runs only
// against the symbol it is configured for in the strategies table (live mode).
func RunSaF(ctx context.Context, pool *pgxpool.Pool, iter int, iterName, outPath string, allSymbols bool) error {
	strategies, err := loadEnabledStrategies(ctx, pool)
	if err != nil {
		return fmt.Errorf("load strategies: %w", err)
	}
	if len(strategies) == 0 {
		return fmt.Errorf("no enabled strategies found")
	}

	// Build symbol list.
	var symbols []string
	if allSymbols {
		symbols, err = loadAllCandleSymbols(ctx, pool)
		if err != nil {
			return fmt.Errorf("load candle symbols: %w", err)
		}
	} else {
		symbolSet := map[string]struct{}{}
		for _, s := range strategies {
			symbolSet[s.Symbol] = struct{}{}
		}
		for sym := range symbolSet {
			symbols = append(symbols, sym)
		}
	}
	sort.Strings(symbols)

	log.Printf("📊 SaF backtest: %d strategies across %d symbol(s)", len(strategies), len(symbols))

	allTrades := make([]safTrade, 0, 1024)
	for _, sym := range symbols {
		candles, err := loadAllH4Candles(ctx, pool, sym)
		if err != nil {
			return fmt.Errorf("load candles for %s: %w", sym, err)
		}
		if len(candles) < safMinCandles {
			log.Printf("  skip %s: only %d bars (need %d)", sym, len(candles), safMinCandles)
			continue
		}
		log.Printf("  %s: %d bars from %s to %s",
			sym, len(candles),
			candles[0].TimestampUTC.Format("2006-01-02"),
			candles[len(candles)-1].TimestampUTC.Format("2006-01-02"))

		// Compute the EMA series and synthesize D1/W1 candles once per symbol.
		mtf := buildFullIndicatorSeries(candles)

		// Resolve which strategies target this symbol.
		symStrategies := make([]strategyRow, 0)
		if allSymbols {
			// Research mode: every strategy's builder runs on every pair.
			// Dedupe by code so we don't run the same rule twice.
			seen := map[string]struct{}{}
			for _, s := range strategies {
				if _, dup := seen[s.Code]; dup {
					continue
				}
				seen[s.Code] = struct{}{}
				symStrategies = append(symStrategies, strategyRow{
					ID: s.ID, Code: s.Code, Symbol: sym, Timeframe: s.Timeframe,
				})
			}
		} else {
			for _, s := range strategies {
				if s.Symbol == sym {
					symStrategies = append(symStrategies, s)
				}
			}
		}

		trades := walkForward(sym, candles, mtf, symStrategies)
		log.Printf("    %d trades resolved", len(trades))
		allTrades = append(allTrades, trades...)
	}

	results := tabulate(allTrades, iter, iterName, symbols)
	results.Timestamp = time.Now().UTC()

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	log.Printf("✅ Wrote %s", outPath)
	printReport(results)
	printScoreSweep(allTrades)
	csvPath := dirOf(outPath) + "/trade_analysis.csv"
	writeTradeCSV(allTrades, csvPath)
	printItemAnalysis(allTrades)
	return nil
}

type strategyRow struct {
	ID       string
	Code     string
	Symbol   string
	Timeframe string
}

func loadAllCandleSymbols(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT symbol FROM candles_h4 ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 8)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func loadEnabledStrategies(ctx context.Context, pool *pgxpool.Pool) ([]strategyRow, error) {
	rows, err := pool.Query(ctx, `SELECT id::text, code, symbol, timeframe FROM strategies WHERE enabled = TRUE ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]strategyRow, 0)
	for rows.Next() {
		var s strategyRow
		if err := rows.Scan(&s.ID, &s.Code, &s.Symbol, &s.Timeframe); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func loadAllH4Candles(ctx context.Context, pool *pgxpool.Pool, symbol string) ([]rules.Candle, error) {
	q := `SELECT timestamp_utc, open, high, low, close FROM candles_h4 WHERE symbol = $1 ORDER BY timestamp_utc ASC`
	args := []any{symbol}
	if safMaxHistoryBars > 0 {
		q += ` LIMIT $2`
		args = append(args, safMaxHistoryBars)
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]rules.Candle, 0, 8192)
	for rows.Next() {
		var c rules.Candle
		if err := rows.Scan(&c.TimestampUTC, &c.Open, &c.High, &c.Low, &c.Close); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// multiTFData is a local alias for the shared synthesis package type.
type multiTFData = synthesis.MultiTFData

// buildFullIndicatorSeries delegates to the shared synthesis package.
func buildFullIndicatorSeries(candles []rules.Candle) multiTFData {
	return synthesis.BuildFullIndicatorSeries(candles)
}


// walkForward iterates bars i=safWarmupBars..N-1, invokes each strategy's
// builder on the slice [0..i], and for every produced setup scans bars i+1..
// for SL/TP resolution.
func walkForward(symbol string, candles []rules.Candle, mtf multiTFData, strats []strategyRow) []safTrade {
	trades := make([]safTrade, 0, 64)

	// Precompute D1/W1 indicator slices (parallel to their candle arrays).
	d1Indicators := make([]rules.Indicators, len(mtf.D1Candles))
	for i := range mtf.D1Candles {
		d1Indicators[i] = rules.Indicators{
			EMA50:  mtf.D1EMA50[i],
			EMA200: mtf.D1EMA200[i],
		}
	}
	w1Indicators := make([]rules.Indicators, len(mtf.W1Candles))
	for i := range mtf.W1Candles {
		w1Indicators[i] = rules.Indicators{
			EMA50:  mtf.W1EMA50[i],
			EMA200: mtf.W1EMA200[i],
		}
	}

	for i := safWarmupBars; i < len(candles); i++ {
		// Truncate D1 candles to the current H4 walk position.
		d1End := 0
		if idx, ok := mtf.H4ToD1[i]; ok && idx < len(mtf.D1Candles) {
			d1End = idx + 1
		}

		// Truncate W1 candles via D1→W1 map.
		w1End := 0
		if d1End > 0 {
			if wIdx, ok := mtf.D1ToW1[d1End-1]; ok && wIdx < len(mtf.W1Candles) {
				w1End = wIdx + 1
			}
		}

		evalCtx := rules.EvalContext{
			Symbol:       symbol,
			Candles:      candles[:i+1],
			Indicators:   mtf.H4Indicators[:i+1],
			D1Candles:    mtf.D1Candles[:d1End],
			D1Indicators: d1Indicators[:d1End],
			W1Candles:    mtf.W1Candles[:w1End],
			W1Indicators: w1Indicators[:w1End],
		}

		for _, s := range strats {
			builder, ok := rules.SetupBuilders[rules.RuleCode(s.Code)]
			if !ok {
				continue
			}
			setup, err := builder(evalCtx)
			if err != nil || setup == nil {
				continue
			}
			t := resolveTrade(s.Code, symbol, i, setup, candles)
			t.checklistScore = setup.ChecklistScore
			t.checklistItems = setup.ChecklistItems
			t.direction = setup.Direction
			t.entry = setup.Entry
			t.sl = setup.StopLoss
			t.tp = setup.TakeProfit
			t.rr = setup.RR
			trades = append(trades, t)
		}
	}
	return trades
}

// resolveTrade walks forward from openIdx+1 up to safMaxHoldBars bars until
// the setup's SL or TP is tagged. Pessimistic on gap bars: if a single bar
// hits both SL and TP, counts as a loss.
func resolveTrade(strategyCode, symbol string, openIdx int, setup *rules.Setup, candles []rules.Candle) safTrade {
	t := safTrade{
		strategy: strategyCode,
		symbol:   symbol,
		opened:   candles[openIdx].TimestampUTC,
	}
	end := openIdx + 1 + safMaxHoldBars
	if end > len(candles) {
		end = len(candles)
	}
	for j := openIdx + 1; j < end; j++ {
		c := candles[j]
		hitSL := false
		hitTP := false
		if setup.Direction == "LONG" {
			if c.Low <= setup.StopLoss {
				hitSL = true
			}
			if c.High >= setup.TakeProfit {
				hitTP = true
			}
		} else { // SHORT
			if c.High >= setup.StopLoss {
				hitSL = true
			}
			if c.Low <= setup.TakeProfit {
				hitTP = true
			}
		}
		if hitSL && hitTP {
			// Pessimistic: treat same-bar as SL hit first.
			t.result = "loss"
			t.rMult = -1
			t.closed = c.TimestampUTC
			return t
		}
		if hitSL {
			t.result = "loss"
			t.rMult = -1
			t.closed = c.TimestampUTC
			return t
		}
		if hitTP {
			t.result = "win"
			t.rMult = setup.RR
			t.closed = c.TimestampUTC
			return t
		}
	}
	// Timeout: close at last bar's close; compute unrealized R.
	last := candles[end-1]
	t.result = "timeout"
	t.closed = last.TimestampUTC
	if setup.Direction == "LONG" {
		risk := setup.Entry - setup.StopLoss
		if risk > 0 {
			t.rMult = (last.Close - setup.Entry) / risk
		}
	} else {
		risk := setup.StopLoss - setup.Entry
		if risk > 0 {
			t.rMult = (setup.Entry - last.Close) / risk
		}
	}
	return t
}

// tabulate collapses trades into per-strategy + aggregate stats.
func tabulate(trades []safTrade, iter int, iterName string, symbols []string) SaFResults {
	res := SaFResults{
		Iter:     iter,
		IterName: iterName,
		Symbols:  symbols,
		Per:      map[string]*SaFStrategyStats{},
	}
	byStrat := map[string][]safTrade{}
	for _, t := range trades {
		byStrat[t.strategy] = append(byStrat[t.strategy], t)
	}
	agg := &SaFStrategyStats{Strategy: "AGGREGATE"}
	for strategy, ts := range byStrat {
		s := &SaFStrategyStats{Strategy: strategy}
		fillStats(s, ts)
		res.Per[strategy] = s
		agg.Trades += s.Trades
		agg.Wins += s.Wins
		agg.Losses += s.Losses
		agg.Timeouts += s.Timeouts
	}
	// Aggregate equity curve: chronological across ALL strategies.
	sortedAll := append([]safTrade(nil), trades...)
	sort.Slice(sortedAll, func(i, j int) bool { return sortedAll[i].opened.Before(sortedAll[j].opened) })
	agg.TotalPnlR, agg.MaxDrawdownR = equityMetrics(sortedAll)
	if agg.Trades > 0 {
		agg.WinRate = float64(agg.Wins) / float64(agg.Trades)
		agg.ExpectancyR = agg.TotalPnlR / float64(agg.Trades)
	}
	res.Aggregate = *agg
	return res
}

func fillStats(s *SaFStrategyStats, trades []safTrade) {
	sorted := append([]safTrade(nil), trades...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].opened.Before(sorted[j].opened) })
	for _, t := range sorted {
		s.Trades++
		switch t.result {
		case "win":
			s.Wins++
		case "loss":
			s.Losses++
		case "timeout":
			s.Timeouts++
		}
	}
	s.TotalPnlR, s.MaxDrawdownR = equityMetrics(sorted)
	if s.Trades > 0 {
		s.WinRate = float64(s.Wins) / float64(s.Trades)
		s.ExpectancyR = s.TotalPnlR / float64(s.Trades)
	}
}

// equityMetrics returns (totalR, maxDrawdownR) from a time-sorted trade list.
func equityMetrics(trades []safTrade) (float64, float64) {
	equity := 0.0
	peak := 0.0
	maxDD := 0.0
	for _, t := range trades {
		equity += t.rMult
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		if dd > maxDD {
			maxDD = dd
		}
	}
	return equity, maxDD
}

func printReport(r SaFResults) {
	fmt.Println("\n============================================================")
	fmt.Printf("📊 SaF Backtest — iter %d (%s)\n", r.Iter, r.IterName)
	fmt.Println("============================================================")
	keys := make([]string, 0, len(r.Per))
	for k := range r.Per {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s := r.Per[k]
		fmt.Printf("🔸 %-25s trades=%-5d  WR=%5.1f%%  Exp=%+5.2fR  PnL=%+6.1fR  DD=%5.1fR\n",
			s.Strategy, s.Trades, s.WinRate*100, s.ExpectancyR, s.TotalPnlR, s.MaxDrawdownR)
	}
	a := r.Aggregate
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("🏆 %-25s trades=%-5d  WR=%5.1f%%  Exp=%+5.2fR  PnL=%+6.1fR  DD=%5.1fR\n",
		a.Strategy, a.Trades, a.WinRate*100, a.ExpectancyR, a.TotalPnlR, a.MaxDrawdownR)
	fmt.Println("============================================================")
}

func printScoreSweep(trades []safTrade) {
	fmt.Println("\n============================================================")
	fmt.Println("📊 Score Sweep — trades at each minimum confluence level")
	fmt.Println("============================================================")
	fmt.Printf("%-8s  %-7s  %-7s  %-5s  %-8s  %-8s  %-7s\n",
		"Min", "Trades", "Trades", "WR", "Expect", "PnL", "MaxDD")
	fmt.Printf("%-8s  %-7s  %-7s  %-5s  %-8s  %-8s  %-7s\n",
		"Score", "Total", "/Year", "%", "R", "R", "R")
	fmt.Println("------------------------------------------------------------")

	sorted := append([]safTrade(nil), trades...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].opened.Before(sorted[j].opened) })

	years := 22.0
	if len(sorted) > 0 {
		first := sorted[0].opened
		last := sorted[len(sorted)-1].opened
		y := last.Sub(first).Hours() / (365.25 * 24)
		if y > 1 {
			years = y
		}
	}

	for minScore := 1; minScore <= 16; minScore++ {
		var filtered []safTrade
		for _, t := range sorted {
			if t.checklistScore >= minScore {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			fmt.Printf("≥%-7d  %-7d  %7s  %5s  %8s  %8s  %7s\n",
				minScore, 0, "-", "-", "-", "-", "-")
			continue
		}
		wins := 0
		for _, t := range filtered {
			if t.result == "win" {
				wins++
			}
		}
		pnl, dd := equityMetrics(filtered)
		wr := float64(wins) / float64(len(filtered))
		exp := pnl / float64(len(filtered))
		perYear := float64(len(filtered)) / years

		fmt.Printf("≥%-7d  %-7d  %6.1f   %4.1f%%  %+6.2fR  %+7.1fR  %6.1fR\n",
			minScore, len(filtered), perYear, wr*100, exp, pnl, dd)
	}
	fmt.Println("============================================================")
}


func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

// Canonical order for the 20 checklist items.
var checklistItemOrder = []string{
	"w1_in_favor", "w1_touching_ema", "w1_candle_rejection", "w1_pattern",
	"d1_in_favor", "d1_aoi", "d1_touching_ema", "d1_candle_rejection", "d1_structure_rejection", "d1_pattern",
	"h4_in_favor", "h4_ema_touch", "h4_candle_rejection", "h4_pattern",
	"psych_level", "engulfing",
}

func boolTo01(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func writeTradeCSV(trades []safTrade, path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Printf("⚠️  CSV write failed: %v", err)
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"symbol", "date", "direction", "score", "result", "rMult", "entry", "sl", "tp", "rr"}
	header = append(header, checklistItemOrder...)
	w.Write(header)

	sorted := append([]safTrade(nil), trades...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].opened.Before(sorted[j].opened) })

	for _, t := range sorted {
		row := []string{
			t.symbol,
			t.opened.Format("2006-01-02"),
			t.direction,
			fmt.Sprintf("%d", t.checklistScore),
			t.result,
			fmt.Sprintf("%.2f", t.rMult),
			fmt.Sprintf("%.5f", t.entry),
			fmt.Sprintf("%.5f", t.sl),
			fmt.Sprintf("%.5f", t.tp),
			fmt.Sprintf("%.2f", t.rr),
		}
		for _, item := range checklistItemOrder {
			row = append(row, boolTo01(t.checklistItems[item]))
		}
		w.Write(row)
	}
	log.Printf("📄 Wrote %d trades to %s", len(trades), path)
}

func printItemAnalysis(trades []safTrade) {
	if len(trades) == 0 {
		return
	}

	fmt.Println("\n============================================================")
	fmt.Println("📊 Per-Item Analysis — hit rate & win correlation")
	fmt.Println("============================================================")
	fmt.Printf("%-25s  %7s  %7s  %7s  %7s  %7s\n",
		"Item", "HitRate", "WR|Hit", "WR|Miss", "Lift", "Trades")
	fmt.Println("------------------------------------------------------------")

	for _, item := range checklistItemOrder {
		hitTrue, hitFalse := 0, 0
		winTrue, winFalse := 0, 0
		for _, t := range trades {
			if t.checklistItems[item] {
				hitTrue++
				if t.result == "win" {
					winTrue++
				}
			} else {
				hitFalse++
				if t.result == "win" {
					winFalse++
				}
			}
		}
		hitRate := float64(hitTrue) / float64(len(trades)) * 100
		wrTrue := 0.0
		if hitTrue > 0 {
			wrTrue = float64(winTrue) / float64(hitTrue) * 100
		}
		wrFalse := 0.0
		if hitFalse > 0 {
			wrFalse = float64(winFalse) / float64(hitFalse) * 100
		}
		lift := wrTrue - wrFalse

		fmt.Printf("%-25s  %6.1f%%  %6.1f%%  %6.1f%%  %+6.1f%%  %5d\n",
			item, hitRate, wrTrue, wrFalse, lift, hitTrue)
	}

	// Win vs Loss item profile
	fmt.Println("\n============================================================")
	fmt.Println("📊 Win vs Loss Profile — avg items true per outcome")
	fmt.Println("============================================================")

	type bucket struct {
		label string
		minS  int
		maxS  int
	}
	buckets := []bucket{
		{"Score 1-5", 1, 5},
		{"Score 6-8", 6, 8},
		{"Score 9-10", 9, 10},
		{"Score 11-12", 11, 12},
		{"Score 13+", 13, 20},
	}

	fmt.Printf("%-14s  %6s  %6s  %6s  %6s  %6s  %6s\n",
		"Bucket", "Trades", "Wins", "Losses", "T/O", "WR%", "ExpR")
	fmt.Println("------------------------------------------------------------")

	for _, b := range buckets {
		wins, losses, timeouts := 0, 0, 0
		totalR := 0.0
		for _, t := range trades {
			if t.checklistScore >= b.minS && t.checklistScore <= b.maxS {
				totalR += t.rMult
				switch t.result {
				case "win":
					wins++
				case "loss":
					losses++
				case "timeout":
					timeouts++
				}
			}
		}
		total := wins + losses + timeouts
		if total == 0 {
			continue
		}
		wr := float64(wins) / float64(total) * 100
		exp := totalR / float64(total)
		fmt.Printf("%-14s  %6d  %6d  %6d  %6d  %5.1f%%  %+.2fR\n",
			b.label, total, wins, losses, timeouts, wr, exp)
	}

	// Per-item breakdown for wins vs losses at score ≥9
	fmt.Println("\n============================================================")
	fmt.Println("📊 Item Breakdown — Wins vs Losses (score ≥ 9)")
	fmt.Println("============================================================")
	fmt.Printf("%-25s  %8s  %8s  %8s\n", "Item", "Win%Hit", "Loss%Hit", "Delta")
	fmt.Println("------------------------------------------------------------")

	var winTrades, lossTrades []safTrade
	for _, t := range trades {
		if t.checklistScore < 9 {
			continue
		}
		if t.result == "win" {
			winTrades = append(winTrades, t)
		} else if t.result == "loss" {
			lossTrades = append(lossTrades, t)
		}
	}

	for _, item := range checklistItemOrder {
		winHit := 0
		for _, t := range winTrades {
			if t.checklistItems[item] {
				winHit++
			}
		}
		lossHit := 0
		for _, t := range lossTrades {
			if t.checklistItems[item] {
				lossHit++
			}
		}
		winPct := 0.0
		if len(winTrades) > 0 {
			winPct = float64(winHit) / float64(len(winTrades)) * 100
		}
		lossPct := 0.0
		if len(lossTrades) > 0 {
			lossPct = float64(lossHit) / float64(len(lossTrades)) * 100
		}
		delta := winPct - lossPct
		marker := "  "
		if delta > 10 {
			marker = "🟢"
		} else if delta < -10 {
			marker = "🔴"
		}
		fmt.Printf("%-25s  %7.1f%%  %7.1f%%  %+6.1f%% %s\n",
			item, winPct, lossPct, delta, marker)
	}
	fmt.Println("============================================================")
}
