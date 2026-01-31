package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

func main() {
	fmt.Println("H&S Pattern Detection Backtester")
	fmt.Println("================================")
	fmt.Println()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://stt_user:lantidhe42H@@localhost:5432/set_the_trend?sslmode=disable"
	}
	
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	runBacktestForTimeframe(queries, "D1", patterns.TF_D1)
	runBacktestForTimeframe(queries, "H4", patterns.TF_H4)
	runBacktestForTimeframe(queries, "W1", patterns.TF_W1)
}

func runBacktestForTimeframe(queries *db.Queries, name string, tf patterns.Timeframe) {
	fmt.Printf("\n\n========== %s TIMEFRAME BACKTEST ==========\n", name)
	
	ctx := context.Background()
	
	var candles []patterns.Candle
	var err error

	switch tf {
	case patterns.TF_D1:
		candles, err = loadD1Candles(ctx, queries)
	case patterns.TF_H4:
		candles, err = loadH4Candles(ctx, queries)
	case patterns.TF_W1:
		candles, err = loadWeeklyCandles(ctx, queries)
	default:
		fmt.Printf("Unsupported timeframe: %s\n", tf)
		return
	}

	if err != nil {
		fmt.Printf("Error loading candles: %v\n", err)
		return
	}

	fmt.Printf("Loaded %d candles\n", len(candles))
	if len(candles) > 0 {
		fmt.Printf("Date range: %s to %s\n", 
			candles[0].Timestamp.Format("2006-01-02"),
			candles[len(candles)-1].Timestamp.Format("2006-01-02"))
	}

	// Configure backtest based on timeframe
	config := patterns.BacktestConfig{
		WindowSize:          30,
		ConfidenceThreshold: 0.55,
		MinRR:               1.5,
		MaxBarsToExit:       20,
		Lookback:            patterns.LookbackByTimeframe(tf),
		CooldownBars:        15, // Prevent duplicate trades
	}

	// Timeframe-specific adjustments
	switch tf {
	case patterns.TF_W1:
		config.WindowSize = 20
		config.MaxBarsToExit = 10
		config.CooldownBars = 8
	case patterns.TF_H4:
		config.WindowSize = 40
		config.Lookback = 5 // Lower lookback for H4
		config.CooldownBars = 20
	case patterns.TF_D1:
		config.CooldownBars = 15
	}

	fmt.Printf("\nBacktest Config:\n")
	fmt.Printf("  Window Size: %d\n", config.WindowSize)
	fmt.Printf("  Confidence Threshold: %.2f\n", config.ConfidenceThreshold)
	fmt.Printf("  Min R:R: %.1f\n", config.MinRR)
	fmt.Printf("  Max Bars to Exit: %d\n", config.MaxBarsToExit)
	fmt.Printf("  Lookback: %d\n", config.Lookback)
	fmt.Printf("  Cooldown Bars: %d\n", config.CooldownBars)

	startTime := time.Now()
	metrics, trades := patterns.RunBacktest(candles, config)
	elapsed := time.Since(startTime)

	if metrics == nil {
		fmt.Println("Backtest returned no results (insufficient data)")
		return
	}

	report := patterns.PrintBacktestReport(metrics, trades, "EURUSD")
	fmt.Println(report)
	fmt.Printf("Backtest completed in %v\n", elapsed)

	if len(trades) > 0 {
		fmt.Println("\nSAMPLE TRADES (first 15):")
		fmt.Println("-------------------------")
		maxTrades := 15
		if len(trades) < maxTrades {
			maxTrades = len(trades)
		}
		for i := 0; i < maxTrades; i++ {
			t := trades[i]
			fmt.Printf("%d. %s %s %s | Conf: %.2f | Entry: %.5f | SL: %.5f | TP: %.5f | Exit: %.5f | PnL: %.1f pips (%.2fR) | %s\n",
				i+1, t.EntryTime.Format("2006-01-02"), t.PatternType, t.Direction,
				t.Confidence, t.EntryPrice, t.StopLoss, t.TakeProfit, t.ExitPrice, 
				t.PnLPips, t.PnLR, t.Result)
		}
	}
}

func loadD1Candles(ctx context.Context, queries *db.Queries) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	dbCandles, err := queries.GetCandlesD1ByRange(ctx, db.GetCandlesD1ByRangeParams{
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func loadH4Candles(ctx context.Context, queries *db.Queries) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	dbCandles, err := queries.GetCandlesH4ByRange(ctx, db.GetCandlesH4ByRangeParams{
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func loadWeeklyCandles(ctx context.Context, queries *db.Queries) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	dbCandles, err := queries.GetCandlesWeeklyByRange(ctx, db.GetCandlesWeeklyByRangeParams{
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}
