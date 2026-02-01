package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/confluence"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

var MAJOR_PAIRS = []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD"}

const CONFLUENCE_THRESHOLD = 57.0

type Signal struct {
	Timestamp  time.Time
	Symbol     string
	Direction  string
	Price      float64
	Confluence float64
	Score      *confluence.ChecklistScore
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("       PAPER TRADE SIMULATION - HISTORICAL REPLAY")
	fmt.Println("================================================================================")
	fmt.Printf("Threshold: %.0f%% | Target: ~1.75 signals/week\n", CONFLUENCE_THRESHOLD)
	fmt.Println("================================================================================")

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	ctx := context.Background()

	// Load all data
	fmt.Println("Loading candle data...")
	allWeekly := make(map[string][]patterns.Candle)
	allDaily := make(map[string][]patterns.Candle)
	allH4 := make(map[string][]patterns.Candle)

	for _, symbol := range MAJOR_PAIRS {
		w, _ := loadCandlesForSymbol(ctx, queries, symbol, "W1")
		d, _ := loadCandlesForSymbol(ctx, queries, symbol, "D1")
		h, _ := loadCandlesForSymbol(ctx, queries, symbol, "H4")
		allWeekly[symbol] = w
		allDaily[symbol] = d
		allH4[symbol] = h
		fmt.Printf("  %s: W1=%d D1=%d H4=%d\n", symbol, len(w), len(d), len(h))
	}

	// Confluence scorer
	config := confluence.ConfluenceConfig{MinConfluencePercent: CONFLUENCE_THRESHOLD}
	scorer := confluence.NewConfluenceScorer(config)

	// Simulate from 2020 to present (5 years)
	startDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	fmt.Printf("\nSimulating %s to %s...\n\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var signals []Signal
	signalsByYear := make(map[int]int)
	signalsByPair := make(map[string]int)

	// Walk through H4 candles (every 6th = once per day roughly)
	for _, symbol := range MAJOR_PAIRS {
		h4 := allH4[symbol]
		weekly := allWeekly[symbol]
		daily := allDaily[symbol]

		for i := 120; i < len(h4); i += 6 { // Check once per day
			candle := h4[i]
			if candle.Timestamp.Before(startDate) || candle.Timestamp.After(endDate) {
				continue
			}

			// Get data up to this point
			wWindow := getCandlesBefore(weekly, candle.Timestamp, 60)
			dWindow := getCandlesBefore(daily, candle.Timestamp, 200)
			h4Window := h4[:i+1]
			if len(h4Window) > 300 {
				h4Window = h4Window[len(h4Window)-300:]
			}

			if len(wWindow) < 30 || len(dWindow) < 60 {
				continue
			}

			score, sig := scorer.CalculateConfluence(wWindow, dWindow, h4Window, nil, nil, nil, symbol)

			if score.TotalPercent >= CONFLUENCE_THRESHOLD {
				direction := "LONG"
				if sig != nil {
					direction = sig.Direction
				}

				signals = append(signals, Signal{
					Timestamp:  candle.Timestamp,
					Symbol:     symbol,
					Direction:  direction,
					Price:      candle.Close,
					Confluence: score.TotalPercent,
					Score:      score,
				})

				signalsByYear[candle.Timestamp.Year()]++
				signalsByPair[symbol]++
			}
		}
	}

	// Print results
	fmt.Println("================================================================================")
	fmt.Println("       SIMULATION RESULTS")
	fmt.Println("================================================================================")
	fmt.Printf("Total Signals: %d\n\n", len(signals))

	fmt.Println("📅 BY YEAR:")
	for year := 2020; year <= 2025; year++ {
		count := signalsByYear[year]
		perWeek := float64(count) / 52
		fmt.Printf("  %d: %3d signals (%.2f/week)\n", year, count, perWeek)
	}

	fmt.Println("\n💱 BY PAIR:")
	for _, symbol := range MAJOR_PAIRS {
		count := signalsByPair[symbol]
		fmt.Printf("  %s: %3d signals\n", symbol, count)
	}

	// Calculate frequency
	totalYears := 5.0
	totalWeeks := totalYears * 52
	signalsPerWeek := float64(len(signals)) / totalWeeks

	fmt.Println("\n📊 FREQUENCY:")
	fmt.Printf("  Signals/Year: %.1f\n", float64(len(signals))/totalYears)
	fmt.Printf("  Signals/Week: %.2f\n", signalsPerWeek)
	fmt.Printf("  Target:       1.75-2.00/week\n")

	// Sample signals
	fmt.Println("\n📋 SAMPLE SIGNALS (last 10):")
	start := len(signals) - 10
	if start < 0 {
		start = 0
	}
	for _, s := range signals[start:] {
		fmt.Printf("  %s | %s %-5s @ %.5f | %.1f%%\n",
			s.Timestamp.Format("2006-01-02"), s.Symbol, s.Direction, s.Price, s.Confluence)
	}

	// Log all to file
	f, _ := os.Create("paper_trade_signals.log")
	defer f.Close()
	for _, s := range signals {
		f.WriteString(fmt.Sprintf("%s | %s %s @ %.5f | %.1f%%\n",
			s.Timestamp.Format("2006-01-02 15:04"), s.Symbol, s.Direction, s.Price, s.Confluence))
	}
	fmt.Println("\n✅ All signals written to: paper_trade_signals.log")
}

func loadCandlesForSymbol(ctx context.Context, queries *db.Queries, symbol, tf string) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	var candles []patterns.Candle

	switch tf {
	case "W1":
		dbCandles, err := queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
			Symbol: symbol, TimestampUtc: startTime, TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]patterns.Candle, len(dbCandles))
		for i, c := range dbCandles {
			candles[i] = patterns.Candle{
				Index: i, Timestamp: c.TimestampUtc.Time,
				Open: c.Open.InexactFloat64(), High: c.High.InexactFloat64(),
				Low: c.Low.InexactFloat64(), Close: c.Close.InexactFloat64(),
			}
		}
	case "D1":
		dbCandles, err := queries.GetCandlesD1BySymbolAndRange(ctx, db.GetCandlesD1BySymbolAndRangeParams{
			Symbol: symbol, TimestampUtc: startTime, TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]patterns.Candle, len(dbCandles))
		for i, c := range dbCandles {
			candles[i] = patterns.Candle{
				Index: i, Timestamp: c.TimestampUtc.Time,
				Open: c.Open.InexactFloat64(), High: c.High.InexactFloat64(),
				Low: c.Low.InexactFloat64(), Close: c.Close.InexactFloat64(),
			}
		}
	case "H4":
		dbCandles, err := queries.GetCandlesH4BySymbolAndRange(ctx, db.GetCandlesH4BySymbolAndRangeParams{
			Symbol: symbol, TimestampUtc: startTime, TimestampUtc_2: endTime,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]patterns.Candle, len(dbCandles))
		for i, c := range dbCandles {
			candles[i] = patterns.Candle{
				Index: i, Timestamp: c.TimestampUtc.Time,
				Open: c.Open.InexactFloat64(), High: c.High.InexactFloat64(),
				Low: c.Low.InexactFloat64(), Close: c.Close.InexactFloat64(),
			}
		}
	}
	return candles, nil
}

func getCandlesBefore(candles []patterns.Candle, before time.Time, limit int) []patterns.Candle {
	var result []patterns.Candle
	for _, c := range candles {
		if c.Timestamp.Before(before) {
			result = append(result, c)
		}
	}
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}
