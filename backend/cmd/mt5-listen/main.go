// mt5-listen accepts H4 OHLCV pushes from a companion MQL5 Expert Advisor
// running in MetaTrader 5, validates them with per-symbol sanity bands,
// upserts into candles_h4, and (optionally) fires the SignalEvaluator.
//
// This is the broker-direct ingest path: no third-party data API, no rate
// limits, no race conditions like the TradingView MCP. Works in Albania and
// any other geo where MT5 itself runs.
//
// Usage:
//
//	mt5-listen                 # listen on :9991 forever
//	mt5-listen --addr :9991
//	mt5-listen --no-eval       # ingest only, skip evaluator (useful for testing)
//
// Environment: same DB_* vars as cmd/api.
//
// HTTP API (called by the EA — see scripts/SafExport.mq5):
//
//	POST /ingest
//	Content-Type: application/json
//	Body: {"symbol":"EURUSD","bars":[
//	         {"time":1777496400,"open":1.16762,"high":1.16772,"low":1.16707,"close":1.16758,"volume":0}
//	       ]}
//	Response: {"upserted":N} on success; non-2xx with {"error":...} on failure.
//
//	GET /health → {"ok":true}
//
// `time` is Unix epoch seconds (UTC). Symbols are bare (no broker suffix —
// the EA strips suffixes like "EURUSD.m" or "EURUSD-Pro" before posting).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

// Same per-symbol sanity bands as cmd/td-sync. The corruption guard.
var sanityRange = map[string][2]float64{
	"AUDCHF": {0.45, 1.10}, "AUDUSD": {0.40, 1.20},
	"CADJPY": {60, 130}, "CHFJPY": {70, 200},
	"EURAUD": {1.10, 2.10}, "EURCAD": {1.20, 1.80},
	"EURCHF": {0.90, 1.70}, "EURGBP": {0.65, 0.99},
	"EURJPY": {90, 200}, "EURUSD": {0.80, 1.70},
	"GBPCHF": {1.00, 2.50}, "GBPJPY": {115, 260},
	"GBPUSD": {1.00, 2.20}, "NZDJPY": {40, 100},
	"NZDUSD": {0.35, 1.00}, "USDCAD": {0.90, 1.65},
	"USDCHF": {0.65, 1.50}, "USDJPY": {70, 170},
	"XAUUSD": {200, 6000},
}

type bar struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume,omitempty"`
}

type ingestRequest struct {
	Symbol string `json:"symbol"`
	Bars   []bar  `json:"bars"`
}

type server struct {
	pool      *pgxpool.Pool
	evaluator *services.SignalEvaluator
	noEval    bool

	// Debounce evaluator: only run once per N seconds even if many symbols
	// post within a tight window. The MQL5 EA sends one POST per symbol per
	// cycle; without debouncing we'd run EvaluateAll 19 times.
	lastEvalUnix int64
}

const evalDebounceSec = 30

func main() {
	var (
		addr   = flag.String("addr", ":9991", "HTTP listen address")
		noEval = flag.Bool("no-eval", false, "skip the signal evaluator after upsert")
	)
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config.Load")
	}
	ctx := context.Background()
	_, pool, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("database")
	}
	defer pool.Close()

	strategyRepo := repositories.NewStrategyRepository(pool)
	signalRepo := repositories.NewSignalRepository(pool)

	srv := &server{
		pool:      pool,
		evaluator: services.NewSignalEvaluator(pool, strategyRepo, signalRepo),
		noEval:    *noEval,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ingest", srv.handleIngest)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info().Str("addr", *addr).Bool("no_eval", *noEval).Msg("mt5-listen ready")
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("listen")
	}
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
		return
	}
	defer r.Body.Close()

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json: " + err.Error()})
		return
	}
	if req.Symbol == "" || len(req.Bars) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "symbol and non-empty bars required"})
		return
	}

	band, hasBand := sanityRange[req.Symbol]
	if !hasBand {
		log.Warn().Str("symbol", req.Symbol).Msg("no sanity band configured — accepting anyway")
	}

	upserted := 0
	for _, b := range req.Bars {
		if hasBand && (b.Close < band[0] || b.Close > band[1]) {
			log.Error().Str("symbol", req.Symbol).Float64("close", b.Close).
				Floats64("band", []float64{band[0], band[1]}).
				Msg("close outside sanity band — refusing whole batch")
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("sanity check failed: close=%v outside [%v,%v]", b.Close, band[0], band[1]),
			})
			return
		}
		if err := upsertBar(r.Context(), s.pool, req.Symbol, b); err != nil {
			log.Error().Err(err).Str("symbol", req.Symbol).Msg("upsert failed")
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		upserted++
	}

	log.Info().Str("symbol", req.Symbol).Int("bars", upserted).Msg("ingested")

	if !s.noEval {
		s.maybeEvaluate(r.Context())
	}
	writeJSON(w, http.StatusOK, map[string]any{"upserted": upserted})
}

// maybeEvaluate runs EvaluateAll at most once per evalDebounceSec to avoid
// 19 redundant runs when the EA fans out 19 symbol POSTs in a tight window.
// Uses an atomic compare-and-swap on a unix-second timestamp.
func (s *server) maybeEvaluate(ctx context.Context) {
	now := time.Now().Unix()
	last := atomic.LoadInt64(&s.lastEvalUnix)
	if now-last < evalDebounceSec {
		return
	}
	if !atomic.CompareAndSwapInt64(&s.lastEvalUnix, last, now) {
		return // another goroutine claimed this window
	}
	// Run async so the HTTP response isn't blocked by evaluator latency.
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		created, err := s.evaluator.EvaluateAll(bg)
		if err != nil {
			log.Error().Err(err).Msg("evaluator")
			return
		}
		log.Info().Int("new_signals", created).Msg("evaluator done")
	}()
}

func upsertBar(ctx context.Context, pool *pgxpool.Pool, symbol string, b bar) error {
	const q = `
		INSERT INTO candles_h4 (symbol, timestamp_utc, open, high, low, close, volume)
		VALUES ($1, $2, $3::numeric, $4::numeric, $5::numeric, $6::numeric, $7)
		ON CONFLICT (symbol, timestamp_utc) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close`
	ts := time.Unix(b.Time, 0).UTC()
	var vol any
	if b.Volume > 0 {
		vol = b.Volume
	}
	_, err := pool.Exec(ctx, q,
		symbol, ts,
		fStr(b.Open), fStr(b.High), fStr(b.Low), fStr(b.Close),
		vol,
	)
	return err
}

func fStr(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
