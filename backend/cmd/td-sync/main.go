// td-sync fetches H4 OHLCV from TwelveData (https://twelvedata.com) for every
// enabled SAF symbol, upserts to candles_h4, then runs the live signal
// evaluator. Replaces the Dukascopy path in cmd/candle-sync and the
// TradingView MCP path (which proved race-prone).
//
// Usage:
//
//	td-sync --once             # one cycle and exit (good for cron)
//	td-sync                    # loop, sleeping until next H4 close + 90s
//	td-sync --symbols EURUSD   # restrict to one symbol (default: all enabled)
//	td-sync --size 60          # bars per fetch (default 10; 60 for backfill)
//	td-sync --no-eval          # skip the signal evaluator (just ingest)
//
// Environment:
//
//	TWELVEDATA_API_KEY  — required. Free signup at https://twelvedata.com
//	DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE  — DB pool
//
// Free tier (800 credits/day, 8/min) handles 19 symbols × 6 H4 cycles/day
// (~114 calls) with comfortable headroom.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

const (
	defaultSize  = 10
	tdInterval   = "4h"
	tdEndpoint   = "https://api.twelvedata.com/time_series"
	httpTimeout  = 15 * time.Second
	betweenCalls = 8 * time.Second // TwelveData free tier caps at 8 req/min → 7.5s minimum; 8s gives headroom
)

// dbToTD maps internal DB symbols to TwelveData ticker format. Most are just
// "AAA/BBB". XAUUSD on TwelveData is "XAU/USD".
func dbToTD(sym string) string {
	if len(sym) != 6 {
		return sym
	}
	return sym[:3] + "/" + sym[3:]
}

// sanityRange is a rough plausibility band for each symbol. A close outside
// the band marks the bar as suspect — we log loudly and skip it. This is the
// guard that prevented the original cross-symbol corruption from recurring.
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

type tdValue struct {
	Datetime string `json:"datetime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume,omitempty"`
}

type tdResponse struct {
	Meta   map[string]any `json:"meta,omitempty"`
	Values []tdValue      `json:"values,omitempty"`
	Status string         `json:"status,omitempty"`
	Code   int            `json:"code,omitempty"`
	Msg    string         `json:"message,omitempty"`
}

func main() {
	var (
		once        = flag.Bool("once", false, "run one cycle and exit")
		symbolsFlag = flag.String("symbols", "", "comma-separated symbols (default: all enabled)")
		size        = flag.Int("size", defaultSize, "bars to fetch per symbol")
		noEval      = flag.Bool("no-eval", false, "skip the signal evaluator step")
	)
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	apiKey := os.Getenv("TWELVEDATA_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("TWELVEDATA_API_KEY not set — sign up at https://twelvedata.com (free)")
	}

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
	evaluator := services.NewSignalEvaluator(pool, strategyRepo, signalRepo)

	runCycle := func() {
		symbols, err := resolveSymbols(ctx, strategyRepo, *symbolsFlag)
		if err != nil {
			log.Error().Err(err).Msg("resolve symbols")
			return
		}
		log.Info().Int("count", len(symbols)).Msg("syncing symbols")

		client := &http.Client{Timeout: httpTimeout}
		totalBars := 0
		ok, fail := 0, 0
		for _, sym := range symbols {
			n, err := fetchAndUpsert(ctx, client, pool, apiKey, sym, *size)
			if err != nil {
				log.Error().Err(err).Str("symbol", sym).Msg("sync failed")
				fail++
			} else {
				totalBars += n
				ok++
				log.Info().Str("symbol", sym).Int("bars", n).Msg("synced")
			}
			time.Sleep(betweenCalls)
		}
		log.Info().Int("ok", ok).Int("fail", fail).Int("total_bars", totalBars).Msg("ingest done")

		if *noEval {
			return
		}
		created, err := evaluator.EvaluateAll(ctx)
		if err != nil {
			log.Error().Err(err).Msg("evaluator")
			return
		}
		log.Info().Int("new_signals", created).Msg("evaluator done")
	}

	if *once {
		runCycle()
		return
	}
	for {
		runCycle()
		next := nextH4Tick(time.Now().UTC())
		log.Info().Time("next_run", next).Msg("sleeping until next H4 close + 90s")
		time.Sleep(time.Until(next))
	}
}

// nextH4Tick: next 4-hour boundary in UTC, plus 90s to let TwelveData
// finalize the close.
func nextH4Tick(now time.Time) time.Time {
	hour := now.Hour()
	nextHour := ((hour / 4) + 1) * 4
	day := now
	if nextHour >= 24 {
		nextHour -= 24
		day = day.Add(24 * time.Hour)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), nextHour, 1, 30, 0, time.UTC)
}

func resolveSymbols(ctx context.Context, sr *repositories.StrategyRepository, override string) ([]string, error) {
	if override != "" {
		parts := strings.Split(override, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, strings.ToUpper(s))
			}
		}
		return out, nil
	}
	strategies, err := sr.ListStrategies(ctx, true)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, s := range strategies {
		if _, ok := seen[s.Symbol]; ok {
			continue
		}
		seen[s.Symbol] = struct{}{}
		out = append(out, s.Symbol)
	}
	return out, nil
}

func fetchAndUpsert(ctx context.Context, client *http.Client, pool *pgxpool.Pool, apiKey, symbol string, size int) (int, error) {
	tdSym := dbToTD(symbol)
	q := url.Values{}
	q.Set("symbol", tdSym)
	q.Set("interval", tdInterval)
	q.Set("apikey", apiKey)
	q.Set("outputsize", strconv.Itoa(size))
	q.Set("format", "JSON")
	q.Set("timezone", "UTC")

	req, err := http.NewRequestWithContext(ctx, "GET", tdEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return 0, fmt.Errorf("td http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var td tdResponse
	if err := json.NewDecoder(resp.Body).Decode(&td); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}
	if td.Status != "ok" || len(td.Values) == 0 {
		return 0, fmt.Errorf("td error: code=%d msg=%q status=%q", td.Code, td.Msg, td.Status)
	}

	band, hasBand := sanityRange[symbol]
	count := 0
	for _, v := range td.Values {
		ts, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			log.Warn().Err(err).Str("symbol", symbol).Str("dt", v.Datetime).Msg("bad datetime; skipping")
			continue
		}
		ts = ts.UTC()
		open, errO := strconv.ParseFloat(v.Open, 64)
		high, errH := strconv.ParseFloat(v.High, 64)
		low, errL := strconv.ParseFloat(v.Low, 64)
		close_, errC := strconv.ParseFloat(v.Close, 64)
		if errO != nil || errH != nil || errL != nil || errC != nil {
			return count, fmt.Errorf("td unparseable OHLC for %s @ %s: o=%q h=%q l=%q c=%q",
				symbol, ts, v.Open, v.High, v.Low, v.Close)
		}
		var vol float64
		if v.Volume != "" {
			parsed, errV := strconv.ParseFloat(v.Volume, 64)
			if errV != nil {
				log.Warn().Err(errV).Str("symbol", symbol).Str("vol", v.Volume).Msg("bad volume; treating as 0")
			} else {
				vol = parsed
			}
		}

		if hasBand {
			for _, price := range [4]float64{open, high, low, close_} {
				if price < band[0] || price > band[1] {
					log.Error().Str("symbol", symbol).Float64("price", price).
						Floats64("band", []float64{band[0], band[1]}).
						Msg("price outside sanity band — refusing to ingest (corruption guard)")
					return 0, fmt.Errorf("sanity check failed for %s @ %s: price=%v outside [%v,%v]", symbol, ts, price, band[0], band[1])
				}
			}
		}
		if low > high || open < low || open > high || close_ < low || close_ > high {
			log.Error().Str("symbol", symbol).
				Float64("open", open).Float64("high", high).
				Float64("low", low).Float64("close", close_).
				Msg("invalid OHLC ordering — refusing to ingest")
			return 0, fmt.Errorf("invalid OHLC for %s @ %s: o=%v h=%v l=%v c=%v", symbol, ts, open, high, low, close_)
		}

		if err := upsertBar(ctx, pool, symbol, ts, open, high, low, close_, vol); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func upsertBar(ctx context.Context, pool *pgxpool.Pool, symbol string, ts time.Time, open, high, low, close_, vol float64) error {
	const q = `
		INSERT INTO candles_h4 (symbol, timestamp_utc, open, high, low, close, volume)
		VALUES ($1, $2, $3::numeric, $4::numeric, $5::numeric, $6::numeric, $7)
		ON CONFLICT (symbol, timestamp_utc) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close`
	var volArg any
	if vol > 0 {
		volArg = vol
	}
	_, err := pool.Exec(ctx, q,
		symbol, ts, fStr(open), fStr(high), fStr(low), fStr(close_), volArg,
	)
	return err
}

func fStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
