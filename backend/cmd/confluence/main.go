package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/confluence"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("           CONFLUENCE SCORING BACKTEST")
	fmt.Println("================================================================================")
	fmt.Println()

	connStr := "postgres://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	ctx := context.Background()

	// Load candles for all timeframes
	fmt.Println("Loading candles...")
	
	weeklyCandles, err := loadWeeklyCandles(ctx, queries)
	if err != nil {
		log.Fatalf("Failed to load weekly: %v", err)
	}
	fmt.Printf("  Weekly: %d candles\n", len(weeklyCandles))

	dailyCandles, err := loadDailyCandles(ctx, queries)
	if err != nil {
		log.Fatalf("Failed to load daily: %v", err)
	}
	fmt.Printf("  Daily: %d candles\n", len(dailyCandles))

	h4Candles, err := loadH4Candles(ctx, queries)
	if err != nil {
		log.Fatalf("Failed to load H4: %v", err)
	}
	fmt.Printf("  H4: %d candles\n", len(h4Candles))

	// Create confluence scorer
	config := confluence.ConfluenceConfig{
		MinConfluencePercent: 70.0,
	}
	scorer := confluence.NewConfluenceScorer(config)

	// Run confluence analysis at various points
	fmt.Println("\n================================================================================")
	fmt.Println("           CONFLUENCE ANALYSIS (Last 5 Weekly Bars)")
	fmt.Println("================================================================================")

	for i := 5; i >= 1; i-- {
		wIdx := len(weeklyCandles) - i
		dIdx := len(dailyCandles) - (i * 5) // Approx 5 daily per weekly
		h4Idx := len(h4Candles) - (i * 30)   // Approx 30 H4 per weekly

		if wIdx < 60 || dIdx < 60 || h4Idx < 60 {
			continue
		}

		wWindow := weeklyCandles[:wIdx]
		dWindow := dailyCandles[:dIdx]
		h4Window := h4Candles[:h4Idx]

		score, signal := scorer.CalculateConfluence(
			wWindow, dWindow, h4Window,
			nil, nil, nil, // No H2, H1, M30 data
			"EURUSD",
		)

		bar := weeklyCandles[wIdx-1]
		fmt.Printf("\n[%s] Close: %.5f\n", bar.Timestamp.Format("2006-01-02"), bar.Close)
		fmt.Println("─────────────────────────────────────────────────────")
		
		printScoreBreakdown(score)

		if signal != nil {
			fmt.Printf("\n🎯 SIGNAL: %s at %.5f | SL: %.5f | TP: %.5f | R:R: %.2f\n",
				signal.Direction, signal.EntryPrice, signal.StopLoss, signal.TakeProfit, signal.RiskReward)
		}
	}

	// Count signals at different thresholds across ALL 5 pairs
	fmt.Println("\n================================================================================")
	fmt.Println("           CONFLUENCE DISTRIBUTION (All 5 Major Pairs)")
	fmt.Println("================================================================================")

	symbols := []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD"}
	thresholdSignals := map[float64]int{50: 0, 55: 0, 60: 0, 65: 0, 70: 0, 75: 0}
	totalBarsAnalyzed := 0
	
	buckets := map[string]int{
		"0-20%":   0,
		"20-40%":  0,
		"40-60%":  0,
		"60-80%":  0,
		"80-100%": 0,
	}

	for _, symbol := range symbols {
		for i := 60; i < len(weeklyCandles); i++ {
			dIdx := (i * 5)
			h4Idx := (i * 30)

			if dIdx >= len(dailyCandles) {
				dIdx = len(dailyCandles) - 1
			}
			if h4Idx >= len(h4Candles) {
				h4Idx = len(h4Candles) - 1
			}

			wWindow := weeklyCandles[:i]
			dWindow := dailyCandles[:dIdx]
			h4Window := h4Candles[:h4Idx]

			score, _ := scorer.CalculateConfluence(
				wWindow, dWindow, h4Window,
				nil, nil, nil,
				symbol,
			)

			totalBarsAnalyzed++
			pct := score.TotalPercent
			
			// Count for each threshold
			for threshold := range thresholdSignals {
				if pct >= threshold {
					thresholdSignals[threshold]++
				}
			}

			switch {
			case pct < 20:
				buckets["0-20%"]++
			case pct < 40:
				buckets["20-40%"]++
			case pct < 60:
				buckets["40-60%"]++
			case pct < 80:
				buckets["60-80%"]++
			default:
				buckets["80-100%"]++
			}
		}
	}

	fmt.Printf("\nTotal Bars Analyzed: %d (5 pairs x %d bars)\n\n", totalBarsAnalyzed, (len(weeklyCandles)-60))
	fmt.Println("Confluence Distribution:")
	for _, bucket := range []string{"0-20%", "20-40%", "40-60%", "60-80%", "80-100%"} {
		count := buckets[bucket]
		pct := float64(count) / float64(totalBarsAnalyzed) * 100
		bar := ""
		for j := 0; j < int(pct/2); j++ {
			bar += "█"
		}
		fmt.Printf("  %s: %4d (%5.1f%%) %s\n", bucket, count, pct, bar)
	}

	fmt.Println("\n📊 SIGNAL COUNTS BY THRESHOLD:")
	yearsOfData := 10.0
	weeksOfData := yearsOfData * 52
	for _, threshold := range []float64{50, 55, 60, 65, 70, 75} {
		count := thresholdSignals[threshold]
		perYear := float64(count) / yearsOfData
		perWeek := float64(count) / weeksOfData
		fmt.Printf("  ≥%2.0f%%: %4d signals | %.1f/year | %.2f/week\n", threshold, count, perYear, perWeek)
	}
}

func printScoreBreakdown(score *confluence.ChecklistScore) {
	fmt.Println("WEEKLY (30.6% max):")
	fmt.Printf("  ├─ Favor:     %.1f%% %s\n", score.WeeklyFavor*100, check(score.WeeklyFavor > 0))
	fmt.Printf("  ├─ AOI:       %.1f%% %s\n", score.WeeklyAOI*100, check(score.WeeklyAOI > 0))
	fmt.Printf("  ├─ EMA:       %.1f%% %s\n", score.WeeklyEMA*100, check(score.WeeklyEMA > 0))
	fmt.Printf("  ├─ Reject:    %.1f%% %s\n", score.WeeklyReject*100, check(score.WeeklyReject > 0))
	fmt.Printf("  ├─ Structure: %.1f%% %s\n", score.WeeklyStruct*100, check(score.WeeklyStruct > 0))
	fmt.Printf("  └─ Pattern:   %.1f%% %s\n", score.WeeklyPattern*100, check(score.WeeklyPattern > 0))

	fmt.Println("DAILY (30.6% max):")
	fmt.Printf("  ├─ Favor:     %.1f%% %s\n", score.DailyFavor*100, check(score.DailyFavor > 0))
	fmt.Printf("  ├─ AOI:       %.1f%% %s\n", score.DailyAOI*100, check(score.DailyAOI > 0))
	fmt.Printf("  ├─ EMA:       %.1f%% %s\n", score.DailyEMA*100, check(score.DailyEMA > 0))
	fmt.Printf("  ├─ Reject:    %.1f%% %s\n", score.DailyReject*100, check(score.DailyReject > 0))
	fmt.Printf("  ├─ Structure: %.1f%% %s\n", score.DailyStruct*100, check(score.DailyStruct > 0))
	fmt.Printf("  └─ Pattern:   %.1f%% %s\n", score.DailyPattern*100, check(score.DailyPattern > 0))

	fmt.Println("4H (25% max):")
	fmt.Printf("  ├─ Favor:     %.1f%% %s\n", score.H4Favor*100, check(score.H4Favor > 0))
	fmt.Printf("  ├─ EMA:       %.1f%% %s\n", score.H4EMA*100, check(score.H4EMA > 0))
	fmt.Printf("  ├─ Reject:    %.1f%% %s\n", score.H4Reject*100, check(score.H4Reject > 0))
	fmt.Printf("  ├─ Structure: %.1f%% %s\n", score.H4Struct*100, check(score.H4Struct > 0))
	fmt.Printf("  └─ Pattern:   %.1f%% %s\n", score.H4Pattern*100, check(score.H4Pattern > 0))

	fmt.Println("LOWER TFs (13.9% max):")
	fmt.Printf("  ├─ 2H Psych:  %.1f%% %s\n", score.H2Psych*100, check(score.H2Psych > 0))
	fmt.Printf("  ├─ 1H SOS:    %.1f%% %s\n", score.H1SOS*100, check(score.H1SOS > 0))
	fmt.Printf("  └─ M30 Engulf:%.1f%% %s\n", score.M30Engulf*100, check(score.M30Engulf > 0))

	fmt.Printf("\n📊 TOTAL CONFLUENCE: %.1f%%", score.TotalPercent)
	if score.TotalPercent >= 70 {
		fmt.Print(" ✅ TRADEABLE")
	} else {
		fmt.Print(" ❌ WAIT")
	}
	fmt.Println()
}

func check(v bool) string {
	if v {
		return "✓"
	}
	return "✗"
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

func loadDailyCandles(ctx context.Context, queries *db.Queries) ([]patterns.Candle, error) {
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

func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}
