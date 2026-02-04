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
	fmt.Println("================================================================================")
	fmt.Println("    ENHANCED BACKTEST - With S/D Zones, Liquidity Sweeps, BOS, Kill Zones")
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

	// Run enhanced backtest for H4 (main trading timeframe)
	fmt.Println("Loading H4 candles for EURUSD...")
	
	candles, err := loadH4Candles(ctx, queries)
	if err != nil {
		log.Fatalf("Failed to load candles: %v", err)
	}
	fmt.Printf("Loaded %d H4 candles\n\n", len(candles))

	// Run with enhanced config
	config := patterns.DefaultEnhancedConfig()
	// Use balanced threshold for good trade count + quality
	config.MinConfluence = 0.55 // 55% minimum = upper MEDIUM tier
	config.ConfidenceThreshold = 0.50 // Pattern confidence minimum
	config.Lookback = 3
	config.WindowSize = 50 // Make sure window is set
	
	fmt.Printf("\n--- DEBUG CONFIG ---\n")
	fmt.Printf("  WindowSize: %d\n", config.WindowSize)
	fmt.Printf("  MaxBarsToExit: %d\n", config.MaxBarsToExit)
	fmt.Printf("  ConfidenceThreshold: %.2f\n", config.ConfidenceThreshold)
	fmt.Printf("  MinConfluence: %.2f\n", config.MinConfluence)
	fmt.Printf("  Lookback: %d\n", config.Lookback)
	fmt.Printf("  CooldownBars: %d\n", config.CooldownBars)
	fmt.Printf("  Total candles: %d\n", len(candles))
	fmt.Printf("  Check len < WindowSize+MaxBars+50: %d < %d = %v\n", 
		len(candles), config.WindowSize+config.MaxBarsToExit+50,
		len(candles) < config.WindowSize+config.MaxBarsToExit+50)
	fmt.Printf("  Candles available for backtest: %d\n\n", len(candles)-config.WindowSize-config.MaxBarsToExit)
	
	// Debug: First check if patterns are detected at all
	fmt.Println("--- DEBUG: Scanning for patterns ---")
	patternCount := 0
	for i := 50; i < len(candles)-20 && i < 500; i++ {
		window := candles[i-50:i]
		if hs := patterns.DetectHeadAndShoulders(window, 3); hs != nil {
			patternCount++
			if patternCount <= 3 {
				fmt.Printf("  Found H&S at bar %d, confidence: %.2f\n", i, hs.OverallConfidence)
			}
		}
		if ihs := patterns.DetectInverseHeadAndShoulders(window, 3); ihs != nil {
			patternCount++
			if patternCount <= 3 {
				fmt.Printf("  Found IHS at bar %d, confidence: %.2f\n", i, ihs.OverallConfidence)
			}
		}
	}
	fmt.Printf("  Total patterns in first 500 bars: %d\n\n", patternCount)
	
	fmt.Println("Running ENHANCED backtest with:")
	fmt.Println("  ✓ Supply/Demand Zones (15%)")
	fmt.Println("  ✓ Liquidity Sweeps (10%)")
	fmt.Println("  ✓ Market Structure BOS (12%)")
	fmt.Println("  ✓ Kill Zone Sessions (8%)")
	fmt.Println("  ✓ Neckline Retest (5.6%)")
	fmt.Println("  ✓ Dynamic R:R (1.5-3.0 based on confluence)")
	fmt.Printf("  → Minimum Confluence: %.0f%%\n\n", config.MinConfluence*100)

	result, trades := patterns.RunEnhancedBacktest(candles, config)
	
	if result == nil {
		fmt.Println("No results - check data")
		return
	}

	// Print detailed report
	patterns.PrintEnhancedReport(result, trades)

	// Compare with old backtest
	fmt.Println("\n--- COMPARISON: OLD vs ENHANCED ---")
	oldConfig := patterns.BacktestConfig{
		WindowSize:          30,
		ConfidenceThreshold: 0.35,
		MinRR:               1.5,
		MaxBarsToExit:       20,
		Lookback:            5,
		CooldownBars:        10,
		SpreadPips:          0.2,
		SlippagePips:        0.3,
		TPSlippagePips:      0.2,
	}
	
	oldMetrics, oldTrades := patterns.RunBacktest(candles, oldConfig)
	if oldMetrics != nil {
		fmt.Printf("\nOLD BACKTEST (pattern only):\n")
		fmt.Printf("  Trades: %d | Win Rate: %.1f%% | P&L: %.2fR\n",
			len(oldTrades), oldMetrics.WinRate*100, oldMetrics.TotalPnLR)
		
		fmt.Printf("\nENHANCED BACKTEST (with new components):\n")
		fmt.Printf("  Trades: %d | Win Rate: %.1f%% | P&L: %.2fR\n",
			len(trades), result.WinRate*100, result.TotalPnLR)
		
		// Calculate improvement
		if oldMetrics.TotalPnLR != 0 {
			improvement := ((result.TotalPnLR - oldMetrics.TotalPnLR) / abs(oldMetrics.TotalPnLR)) * 100
			fmt.Printf("\n→ P&L Improvement: %.1f%%\n", improvement)
		}
		if oldMetrics.WinRate != 0 {
			wrImprovement := result.WinRate - oldMetrics.WinRate
			fmt.Printf("→ Win Rate Change: %+.1f%%\n", wrImprovement*100)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func loadH4Candles(ctx context.Context, queries *db.Queries) ([]patterns.Candle, error) {
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	dbCandles, err := queries.GetCandlesH4BySymbolAndRange(ctx, db.GetCandlesH4BySymbolAndRangeParams{
		Symbol:         "EURUSD",
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
		}
	}
	return candles, nil
}
