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

var MAJOR_PAIRS = []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD"}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("           H&S PATTERN BACKTEST - 5 MAJOR PAIRS")
	fmt.Println("================================================================================")
	fmt.Println()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	ctx := context.Background()

	// Aggregate results across all pairs
	totalResults := AggResults{}

	// Run backtest for each pair and timeframe
	for _, symbol := range MAJOR_PAIRS {
		fmt.Printf("\n\n========== %s ==========\n", symbol)

		for _, tf := range []patterns.Timeframe{patterns.TF_W1, patterns.TF_D1, patterns.TF_H4} {
			runBacktestForSymbolAndTF(ctx, queries, symbol, tf, &totalResults)
		}
	}

	// Print summary
	fmt.Println("\n\n================================================================================")
	fmt.Println("           AGGREGATE RESULTS - ALL PAIRS")
	fmt.Println("================================================================================")
	fmt.Printf("Total Trades: %d\n", totalResults.Trades)
	fmt.Printf("Wins: %d (%.1f%%)\n", totalResults.Wins, float64(totalResults.Wins)/float64(maxInt(totalResults.Trades, 1))*100)
	fmt.Printf("Losses: %d\n", totalResults.Losses)
	fmt.Printf("Total P&L: %.2fR (%.1f pips)\n", totalResults.TotalPnLR, totalResults.TotalPips)
	fmt.Printf("Avg per trade: %.2fR\n", totalResults.TotalPnLR/float64(maxInt(totalResults.Trades, 1)))

	// Trades per year estimate
	yearsOfData := 10.0
	tradesPerYear := float64(totalResults.Trades) / yearsOfData
	tradesPerWeek := tradesPerYear / 52
	fmt.Printf("\nTrades/Year: %.1f\n", tradesPerYear)
	fmt.Printf("Trades/Week: %.2f\n", tradesPerWeek)
}

type AggResults struct {
	Trades    int
	Wins      int
	Losses    int
	TotalPnLR float64
	TotalPips float64
}

func runBacktestForSymbolAndTF(ctx context.Context, queries *db.Queries, symbol string, tf patterns.Timeframe, results *AggResults) {
	candles, err := loadCandlesBySymbol(ctx, queries, symbol, tf)
	if err != nil {
		fmt.Printf("  %s %s: Error loading candles: %v\n", symbol, tf, err)
		return
	}

	if len(candles) < 60 {
		fmt.Printf("  %s %s: Insufficient data (%d candles)\n", symbol, tf, len(candles))
		return
	}

	config := patterns.BacktestConfig{
		WindowSize:          30,
		ConfidenceThreshold: 0.35, // Lowered from 0.55 to get more signals
		MinRR:               1.5,
		MaxBarsToExit:       20,
		Lookback:            patterns.LookbackByTimeframe(tf),
		CooldownBars:        10, // Reduced cooldown
	}

	// Timeframe-specific adjustments
	switch tf {
	case patterns.TF_W1:
		config.WindowSize = 20
		config.MaxBarsToExit = 10
		config.CooldownBars = 6
	case patterns.TF_H4:
		config.WindowSize = 40
		config.Lookback = 5
		config.CooldownBars = 12
	case patterns.TF_D1:
		config.CooldownBars = 8
	}

	metrics, trades := patterns.RunBacktest(candles, config)
	if metrics == nil || len(trades) == 0 {
		fmt.Printf("  %s %s: No trades (%d candles)\n", symbol, tf, len(candles))
		return
	}

	// Count H&S vs IHS trades
	hsCount, ihsCount := 0, 0
	for _, t := range trades {
		if t.PatternType == "HS" || t.Direction == "SHORT" {
			hsCount++
		} else {
			ihsCount++
		}
	}

	fmt.Printf("  %s %s: %d trades (H&S:%d IHS:%d) | Win: %.0f%% | P&L: %.2fR\n",
		symbol, tf, len(trades), hsCount, ihsCount, metrics.WinRate, metrics.TotalPnLR)

	// Aggregate
	results.Trades += len(trades)
	results.Wins += int(metrics.WinningTrades)
	results.Losses += int(metrics.LosingTrades)
	results.TotalPnLR += metrics.TotalPnLR
	results.TotalPips += metrics.TotalPnLPips
}

func loadCandlesBySymbol(ctx context.Context, queries *db.Queries, symbol string, tf patterns.Timeframe) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	switch tf {
	case patterns.TF_W1:
		dbCandles, err := queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
			Symbol:         symbol,
			TimestampUtc:   startTime,
			TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		return convertWeeklyCandles(dbCandles), nil

	case patterns.TF_D1:
		dbCandles, err := queries.GetCandlesD1BySymbolAndRange(ctx, db.GetCandlesD1BySymbolAndRangeParams{
			Symbol:         symbol,
			TimestampUtc:   startTime,
			TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		return convertD1Candles(dbCandles), nil

	case patterns.TF_H4:
		dbCandles, err := queries.GetCandlesH4BySymbolAndRange(ctx, db.GetCandlesH4BySymbolAndRangeParams{
			Symbol:         symbol,
			TimestampUtc:   startTime,
			TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		return convertH4Candles(dbCandles), nil

	default:
		return nil, fmt.Errorf("unsupported timeframe: %s", tf)
	}
}

func convertWeeklyCandles(dbCandles []db.CandlesWeekly) []patterns.Candle {
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
	return candles
}

func convertD1Candles(dbCandles []db.CandlesD1) []patterns.Candle {
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
	return candles
}

func convertH4Candles(dbCandles []db.CandlesH4) []patterns.Candle {
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
	return candles
}

func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
