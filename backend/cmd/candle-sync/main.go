// candle-sync fetches the most recent H4 bars from Dukascopy (via the
// `dukascopy-node` CLI), upserts them into candles_h4, and runs the signal
// evaluator. Designed to be invoked periodically (every 4 hours, a minute or
// two past the close) by launchd.
//
// Usage:
//
//	candle-sync --once             # one fetch + evaluate cycle
//	candle-sync                    # loop forever, sleeping until next H4 close + 60s
//	candle-sync --symbols EURUSD   # override symbol set (default: all enabled strategies)
//
// Environment:
//
//	DUKASCOPY_NODE_BIN (optional; defaults to `npx --yes dukascopy-node`)
//	DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE (same as cmd/api)
package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

const defaultLimit = 10 // bars per fetch — enough to backfill a missed cycle or two

func main() {
	var (
		once        = flag.Bool("once", false, "run one cycle and exit")
		symbolsFlag = flag.String("symbols", "", "comma-separated symbols (default: all enabled strategies)")
		fetchSize   = flag.Int("size", defaultLimit, "candles to fetch per cycle")
		evalOnly    = flag.Bool("eval-only", false, "skip fetch and only run the signal evaluator (used for testing)")
	)
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config.Load")
	}

	ctx := context.Background()
	queries, pool, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("database")
	}
	_ = queries
	defer pool.Close()

	strategyRepo := repositories.NewStrategyRepository(pool)
	signalRepo := repositories.NewSignalRepository(pool)
	evaluator := services.NewSignalEvaluator(pool, strategyRepo, signalRepo)

	syncer := &Syncer{pool: pool, fetchSize: *fetchSize}

	runCycle := func() {
		if !*evalOnly {
			symbols, err := resolveSymbols(ctx, strategyRepo, *symbolsFlag)
			if err != nil {
				log.Error().Err(err).Msg("resolve symbols")
				return
			}
			for _, sym := range symbols {
				n, err := syncer.SyncSymbol(ctx, sym)
				if err != nil {
					log.Error().Err(err).Str("symbol", sym).Msg("sync failed")
					continue
				}
				log.Info().Str("symbol", sym).Int("upserted", n).Msg("candles synced")
			}
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
		log.Info().Time("next_run", next).Msg("sleeping until next H4 close")
		time.Sleep(time.Until(next))
	}
}

// nextH4Tick returns the next H4 boundary plus 60s of grace so the broker has
// actually published the closing print.
func nextH4Tick(now time.Time) time.Time {
	hour := now.Hour()
	nextHour := ((hour / 4) + 1) * 4
	day := now
	if nextHour >= 24 {
		nextHour -= 24
		day = day.Add(24 * time.Hour)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), nextHour, 1, 0, 0, time.UTC)
}

func resolveSymbols(ctx context.Context, sr *repositories.StrategyRepository, override string) ([]string, error) {
	if override != "" {
		parts := strings.Split(override, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
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

// Syncer runs the Dukascopy fetch + DB upsert per symbol.
type Syncer struct {
	pool      *pgxpool.Pool
	fetchSize int
}

// SyncSymbol fetches the latest H4 bars from Dukascopy and upserts them.
func (s *Syncer) SyncSymbol(ctx context.Context, symbol string) (int, error) {
	bars, err := fetchDukascopy(ctx, symbol, s.fetchSize)
	if err != nil {
		return 0, err
	}
	if len(bars) == 0 {
		return 0, nil
	}
	upserted := 0
	for _, b := range bars {
		ts := time.UnixMilli(b.Timestamp).UTC()
		if err := s.upsertBar(ctx, symbol, ts, b.Open, b.High, b.Low, b.Close); err != nil {
			return upserted, err
		}
		upserted++
	}
	return upserted, nil
}

func (s *Syncer) upsertBar(ctx context.Context, symbol string, ts time.Time, open, high, low, close float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO candles_h4 (symbol, timestamp_utc, open, high, low, close, volume)
		VALUES ($1, $2, $3::numeric, $4::numeric, $5::numeric, $6::numeric, NULL)
		ON CONFLICT (symbol, timestamp_utc) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close`,
		symbol, ts, floatStr(open), floatStr(high), floatStr(low), floatStr(close),
	)
	return err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
