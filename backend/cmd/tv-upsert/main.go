// tv-upsert reads OHLCV bars from stdin (as produced by the TradingView MCP
// data_get_ohlcv tool) and upserts them into candles_h4. It is the live
// replacement for the Dukascopy fetch path in cmd/candle-sync.
//
// Stdin format: a JSON array of per-symbol payloads:
//
//	[
//	  {"symbol": "EURUSD", "bars": [
//	    {"time": 1777453200, "open": 1.085, "high": 1.087, "low": 1.084, "close": 1.086, "volume": 12345}
//	  ]},
//	  ...
//	]
//
// `time` is Unix epoch seconds (UTC) — the format the TradingView MCP returns.
// Symbols are bare (e.g. "EURUSD", "XAUUSD"), no exchange prefix.
//
// The upsert is idempotent: ON CONFLICT (symbol, timestamp_utc) DO UPDATE.
// Re-running on the same bars is safe; late revisions overwrite cleanly.
//
// Typical flow (driven by a Claude/MCP session every H4 close + grace):
//
//	1. for each symbol in `strategies` (enabled):
//	     mcp.chart_set_symbol("FX:" + symbol)
//	     mcp.data_get_ohlcv(count=10)
//	2. compose a single JSON array of all symbol payloads
//	3. pipe to: tv-upsert
//	4. shell: candle-sync --eval-only
//
// Environment: same DB_* vars as cmd/api / cmd/candle-sync.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/config"
)

type bar struct {
	Time   int64   `json:"time"` // Unix epoch SECONDS (TradingView convention)
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume,omitempty"`
}

type symbolPayload struct {
	Symbol string `json:"symbol"`
	Bars   []bar  `json:"bars"`
}

// sanityRange — same per-symbol bands as cmd/mt5-listen and cmd/td-sync.
// Any bar whose close is outside its band is rejected before the upsert.
// This is the same guard that prevents cross-symbol contamination from any
// ingest path; tv-upsert previously lacked it (which let a USDCAD-priced
// AUDUSD bar through on 2026-05-01).
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

func main() {
	symbolFlag := flag.String("symbol", "", "if set, stdin is parsed as the MCP response shape ({bars:[...]}) and ingested under this symbol")
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal().Err(err).Msg("read stdin")
	}
	if len(raw) == 0 {
		log.Fatal().Msg("empty stdin — expected JSON")
	}

	var payloads []symbolPayload
	if *symbolFlag != "" {
		// Single-symbol mode: stdin is the raw MCP response shape.
		var mcp struct {
			Bars []bar `json:"bars"`
		}
		if err := json.Unmarshal(raw, &mcp); err != nil {
			log.Fatal().Err(err).Msg("parse mcp json")
		}
		payloads = []symbolPayload{{Symbol: *symbolFlag, Bars: mcp.Bars}}
	} else if err := json.Unmarshal(raw, &payloads); err != nil {
		log.Fatal().Err(err).Msg("parse json")
	}
	if len(payloads) == 0 {
		log.Warn().Msg("no symbols in payload — exiting")
		return
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

	totalBars := 0
	totalSymbols := 0
	for _, p := range payloads {
		if p.Symbol == "" || len(p.Bars) == 0 {
			log.Warn().Str("symbol", p.Symbol).Int("bars", len(p.Bars)).Msg("skipping empty payload")
			continue
		}
		n, err := upsertBars(ctx, pool, p)
		if err != nil {
			log.Error().Err(err).Str("symbol", p.Symbol).Msg("upsert failed")
			continue
		}
		totalBars += n
		totalSymbols++
		log.Info().Str("symbol", p.Symbol).Int("bars", n).Msg("upserted")
	}
	log.Info().Int("symbols", totalSymbols).Int("bars", totalBars).Msg("tv-upsert done")
}

// floatStr renders a float without scientific notation, matching the cast
// expected by candles_h4's numeric columns.
func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func upsertBars(ctx context.Context, pool *pgxpool.Pool, p symbolPayload) (int, error) {
	const q = `
		INSERT INTO candles_h4 (symbol, timestamp_utc, open, high, low, close, volume)
		VALUES ($1, $2, $3::numeric, $4::numeric, $5::numeric, $6::numeric, $7)
		ON CONFLICT (symbol, timestamp_utc) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close`

	band, hasBand := sanityRange[p.Symbol]
	count := 0
	for _, b := range p.Bars {
		if hasBand {
			for _, price := range [4]float64{b.Open, b.High, b.Low, b.Close} {
				if price < band[0] || price > band[1] {
					log.Error().Str("symbol", p.Symbol).Float64("price", price).
						Floats64("band", []float64{band[0], band[1]}).
						Int64("time", b.Time).
						Msg("price outside sanity band — refusing to ingest (corruption guard)")
					return count, fmt.Errorf("sanity check failed: %s price=%v outside [%v,%v]", p.Symbol, price, band[0], band[1])
				}
			}
		}
		if b.Low > b.High || b.Open < b.Low || b.Open > b.High || b.Close < b.Low || b.Close > b.High {
			log.Error().Str("symbol", p.Symbol).
				Float64("open", b.Open).Float64("high", b.High).
				Float64("low", b.Low).Float64("close", b.Close).
				Int64("time", b.Time).
				Msg("invalid OHLC ordering — refusing to ingest")
			return count, fmt.Errorf("invalid OHLC for %s: o=%v h=%v l=%v c=%v", p.Symbol, b.Open, b.High, b.Low, b.Close)
		}
		ts := time.Unix(b.Time, 0).UTC()
		var vol any
		if b.Volume > 0 {
			vol = b.Volume
		}
		if _, err := pool.Exec(ctx, q,
			p.Symbol, ts,
			floatStr(b.Open), floatStr(b.High), floatStr(b.Low), floatStr(b.Close),
			vol,
		); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
