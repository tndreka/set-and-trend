package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/patterns"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/rules"
	"set-and-trend/backend/internal/synthesis"
)

// SignalEvaluator runs every enabled strategy against the latest H4 bar for
// the strategy's symbol and persists triggered setups to setup_signals. The
// (strategy_id, candle_id) UNIQUE constraint on setup_signals deduplicates
// repeated runs against the same bar.
type SignalEvaluator struct {
	pool         *pgxpool.Pool
	strategyRepo *repositories.StrategyRepository
	signalRepo   *repositories.SignalRepository
	alerter      *TelegramAlerter // nil-safe; Send is a no-op when nil
}

func NewSignalEvaluator(
	pool *pgxpool.Pool,
	strategyRepo *repositories.StrategyRepository,
	signalRepo *repositories.SignalRepository,
) *SignalEvaluator {
	return &SignalEvaluator{
		pool:         pool,
		strategyRepo: strategyRepo,
		signalRepo:   signalRepo,
		alerter:      NewTelegramAlerterFromEnv(),
	}
}

// historyBars is the number of recent H4 candles loaded per evaluation. 7200
// gives ~1200 D1 bars and ~240 W1 bars — enough for EMA200 warmup on all
// timeframes including W1, plus lookback for patterns and zones.
const historyBars = 7200

// symbolData caches per-symbol candle data and synthesis results so multiple
// strategies sharing a symbol don't repeat the expensive DB load + synthesis.
type symbolData struct {
	candles  []rules.Candle
	latestID uuid.UUID
	evalCtx  rules.EvalContext
}

// EvaluateAll loads the most recent `historyBars` H4 candles for each unique
// symbol, computes multi-TF synthesis once per symbol, then evaluates every
// strategy against the cached context. Returns the number of new signals.
func (e *SignalEvaluator) EvaluateAll(ctx context.Context) (int, error) {
	strategies, err := e.strategyRepo.ListStrategies(ctx, true)
	if err != nil {
		return 0, fmt.Errorf("list enabled strategies: %w", err)
	}

	cache := make(map[string]*symbolData)

	created := 0
	for _, s := range strategies {
		sd, ok := cache[s.Symbol]
		if !ok {
			sd, err = e.buildSymbolData(ctx, s.Symbol)
			if err != nil {
				log.Error().Err(err).Str("symbol", s.Symbol).Msg("build symbol data failed")
				continue
			}
			cache[s.Symbol] = sd
		}
		if sd == nil {
			continue
		}

		n, err := e.evaluateStrategy(ctx, s, sd)
		if err != nil {
			log.Error().Err(err).Str("strategy", s.Code).Msg("evaluate strategy failed")
			continue
		}
		created += n
	}
	return created, nil
}

// buildSymbolData loads candles and runs multi-TF synthesis once per symbol.
// Returns nil (not an error) if there is not enough history yet.
func (e *SignalEvaluator) buildSymbolData(ctx context.Context, symbol string) (*symbolData, error) {
	candles, latestID, err := e.loadH4Candles(ctx, symbol, historyBars)
	if err != nil {
		return nil, fmt.Errorf("load candles for %s: %w", symbol, err)
	}
	if len(candles) < 30 {
		return nil, nil
	}

	mtf := synthesis.BuildFullIndicatorSeries(candles)

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

	// No truncation needed: all bars are historical at evaluation time, so
	// the full D1/W1 slices are safe to pass (no future leakage).
	return &symbolData{
		candles:  candles,
		latestID: latestID,
		evalCtx: rules.EvalContext{
			Symbol:       symbol,
			Candles:      candles,
			Indicators:   mtf.H4Indicators,
			D1Candles:    mtf.D1Candles,
			D1Indicators: d1Indicators,
			W1Candles:    mtf.W1Candles,
			W1Indicators: w1Indicators,
		},
	}, nil
}

func (e *SignalEvaluator) evaluateStrategy(ctx context.Context, s *repositories.Strategy, sd *symbolData) (int, error) {
	builder, ok := rules.SetupBuilders[rules.RuleCode(s.Code)]
	if !ok {
		log.Warn().Str("strategy", s.Code).Msg("no setup builder registered")
		return 0, nil
	}

	setup, err := builder(sd.evalCtx)
	if err != nil {
		return 0, fmt.Errorf("builder %s: %w", s.Code, err)
	}
	if setup == nil {
		return 0, nil
	}

	details, _ := json.Marshal(e.buildRichDetails(setup, sd, s.Code))

	_, err = e.signalRepo.CreateSignal(ctx, repositories.CreateSignalParams{
		StrategyID: s.ID,
		Symbol:     s.Symbol,
		Timeframe:  s.Timeframe,
		CandleID:   sd.latestID,
		Direction:  setup.Direction,
		Entry:      setup.Entry,
		StopLoss:   setup.StopLoss,
		TakeProfit: setup.TakeProfit,
		RR:         setup.RR,
		Confidence: setup.Confidence,
		Details:    details,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("persist signal: %w", err)
	}

	log.Info().
		Str("strategy", s.Code).
		Str("direction", setup.Direction).
		Float64("entry", setup.Entry).
		Float64("rr", setup.RR).
		Msg("signal created")

	// Push to Telegram. Nil-safe; failures don't block.
	if a := e.alerter; a != nil {
		text := a.FormatSetup(s.Symbol, setup)
		go a.Send(context.Background(), text)
	}
	return 1, nil
}

// buildRichDetails creates a detailed JSON payload for the signal, including
// the full checklist breakdown, EMA levels, key zones, and swing structure.
// This data drives chart drawings and the Telegram analysis text.
func (e *SignalEvaluator) buildRichDetails(setup *rules.Setup, sd *symbolData, strategyCode string) map[string]any {
	ec := sd.evalCtx
	_, ind, _ := ec.Latest()
	latestBar := sd.candles[len(sd.candles)-1]

	details := map[string]any{
		"reason":        setup.Reason,
		"strategy_code": strategyCode,
		"latest_bar":    latestBar.TimestampUTC.Format(time.RFC3339),
	}

	// EMA levels for chart drawing
	emas := map[string]any{
		"h4_ema50":  round5(ind.EMA50),
		"h4_ema200": round5(ind.EMA200),
	}
	if ind.D1EMA50 > 0 {
		emas["d1_ema50"] = round5(ind.D1EMA50)
		emas["d1_ema200"] = round5(ind.D1EMA200)
	}
	if ind.W1EMA50 > 0 {
		emas["w1_ema50"] = round5(ind.W1EMA50)
		emas["w1_ema200"] = round5(ind.W1EMA200)
	}
	details["emas"] = emas

	// Checklist items (only for SAF_CHECKLIST strategy)
	if setup.ChecklistItems != nil {
		details["checklist_score"] = setup.ChecklistScore
		details["checklist_max"] = 14
		details["checklist_items"] = setup.ChecklistItems
	}

	// Confluence factors — which key factors triggered
	details["confluence"] = buildConfluenceFactors(ec, setup, ind)

	// Supply/demand zones near entry
	h4Candles := ruleCandlesToPatternCandles(ec.Candles)
	cfg := patterns.DefaultZoneConfig(ec.Symbol)
	zones := patterns.DetectSupplyDemandZones(h4Candles, cfg)
	active := patterns.GetActiveZones(h4Candles, zones, cfg.ZoneFreshnessBars)
	zoneData := make([]map[string]any, 0)
	for _, z := range active {
		dist := math.Abs(latestBar.Close - z.ProximalPrice)
		atr := computeATR14(h4Candles)
		if atr > 0 && dist < atr*5 {
			zoneData = append(zoneData, map[string]any{
				"type":     string(z.Type),
				"proximal": round5(z.ProximalPrice),
				"distal":   round5(z.DistalPrice),
			})
		}
	}
	if len(zoneData) > 0 {
		details["zones"] = zoneData
	}

	// Recent swing highs/lows for structure
	swingHighs := patterns.FindSwingHighs(h4Candles, 5)
	swingLows := patterns.FindSwingLows(h4Candles, 5)
	if n := len(swingHighs); n > 0 {
		last3 := swingHighs
		if n > 3 {
			last3 = swingHighs[n-3:]
		}
		sh := make([]float64, len(last3))
		for i, s := range last3 {
			sh[i] = round5(s.Price)
		}
		details["swing_highs"] = sh
	}
	if n := len(swingLows); n > 0 {
		last3 := swingLows
		if n > 3 {
			last3 = swingLows[n-3:]
		}
		sl := make([]float64, len(last3))
		for i, s := range last3 {
			sl[i] = round5(s.Price)
		}
		details["swing_lows"] = sl
	}

	return details
}

func buildConfluenceFactors(ec rules.EvalContext, setup *rules.Setup, ind rules.Indicators) []string {
	factors := make([]string, 0, 8)
	dir := setup.Direction

	// W1 trend
	if ind.W1EMA50 > 0 && ind.W1EMA200 > 0 {
		if (dir == "LONG" && ind.W1EMA50 > ind.W1EMA200) || (dir == "SHORT" && ind.W1EMA50 < ind.W1EMA200) {
			factors = append(factors, "W1 trend aligned (EMA50 vs EMA200)")
		}
	}
	// W1 EMA touch
	if ind.W1EMA50 > 0 {
		c, _, _ := ec.Latest()
		if math.Abs(c.Close-ind.W1EMA50)/c.Close < 0.005 {
			factors = append(factors, "W1 EMA50 touch/rejection")
		}
	}
	// W1 candle rejection
	if w1, ok := ec.LatestW1(); ok && rules.IsCandleRejection(w1, dir) {
		factors = append(factors, "W1 candle rejection wick")
	}
	// D1 trend
	if ind.D1EMA50 > 0 && ind.D1EMA200 > 0 {
		if (dir == "LONG" && ind.D1EMA50 > ind.D1EMA200) || (dir == "SHORT" && ind.D1EMA50 < ind.D1EMA200) {
			factors = append(factors, "D1 trend aligned")
		}
	}
	// D1 candle rejection
	if d1, ok := ec.LatestD1(); ok && rules.IsCandleRejection(d1, dir) {
		factors = append(factors, "D1 candle rejection wick")
	}
	// H4 trend
	if ind.EMA50 > 0 && ind.EMA200 > 0 {
		if (dir == "LONG" && ind.EMA50 > ind.EMA200) || (dir == "SHORT" && ind.EMA50 < ind.EMA200) {
			factors = append(factors, "H4 trend aligned")
		}
	}
	// H4 candle rejection
	c, _, _ := ec.Latest()
	if rules.IsCandleRejection(c, dir) {
		factors = append(factors, "H4 candle rejection wick")
	}

	return factors
}

func round5(f float64) float64 {
	return math.Round(f*100000) / 100000
}

func computeATR14(candles []patterns.Candle) float64 {
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

func ruleCandlesToPatternCandles(rc []rules.Candle) []patterns.Candle {
	out := make([]patterns.Candle, len(rc))
	for i, c := range rc {
		out[i] = patterns.Candle{Open: c.Open, High: c.High, Low: c.Low, Close: c.Close}
	}
	return out
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx surfaces unique-violation as code 23505. Avoid pulling in the full
	// pgconn import just for the type assertion — substring check is fine.
	return contains(err.Error(), "23505")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// loadH4Candles pulls the last `limit` H4 bars for a symbol in chronological
// order (oldest first), and returns the latest bar's UUID for the signal
// linkage.
func (e *SignalEvaluator) loadH4Candles(ctx context.Context, symbol string, limit int) ([]rules.Candle, uuid.UUID, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT id, timestamp_utc, open, high, low, close
		FROM candles_h4
		WHERE symbol = $1
		ORDER BY timestamp_utc DESC
		LIMIT $2`, symbol, limit)
	if err != nil {
		return nil, uuid.Nil, err
	}
	defer rows.Close()

	type loaded struct {
		ID uuid.UUID
		C  rules.Candle
	}
	out := make([]loaded, 0, limit)
	for rows.Next() {
		var (
			id                     uuid.UUID
			ts                     time.Time
			open, high, low, close float64
		)
		if err := rows.Scan(&id, &ts, &open, &high, &low, &close); err != nil {
			return nil, uuid.Nil, err
		}
		out = append(out, loaded{
			ID: id,
			C: rules.Candle{
				Open:         open,
				High:         high,
				Low:          low,
				Close:        close,
				TimestampUTC: ts,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, uuid.Nil, err
	}
	if len(out) == 0 {
		return nil, uuid.Nil, nil
	}

	// reverse to chronological (oldest first)
	candles := make([]rules.Candle, len(out))
	for i, l := range out {
		candles[len(out)-1-i] = l.C
	}
	latestID := out[0].ID
	return candles, latestID, nil
}

