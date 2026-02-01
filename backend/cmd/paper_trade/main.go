package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/confluence"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

var MAJOR_PAIRS = []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD"}

const (
	CONFLUENCE_THRESHOLD = 57.0 // 57% = ~1.57 signals/week (optimal)
	CHECK_INTERVAL       = 4 * time.Hour // Check every 4h (H4 candle close)
)

type SignalLog struct {
	file *os.File
}

func NewSignalLog(path string) (*SignalLog, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &SignalLog{file: f}, nil
}

func (s *SignalLog) Log(symbol, direction string, price, confluence float64, score *confluence.ChecklistScore) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("%s | %s %s @ %.5f | Confluence: %.1f%%\n",
		timestamp, symbol, direction, price, confluence)
	
	// Detailed breakdown
	details := fmt.Sprintf("  W1: Favor=%.1f%% AOI=%.1f%% EMA=%.1f%% Struct=%.1f%% Pattern=%.1f%%\n",
		score.WeeklyFavor*100, score.WeeklyAOI*100, score.WeeklyEMA*100,
		score.WeeklyStruct*100, score.WeeklyPattern*100)
	details += fmt.Sprintf("  D1: Favor=%.1f%% AOI=%.1f%% EMA=%.1f%% Struct=%.1f%% Pattern=%.1f%%\n",
		score.DailyFavor*100, score.DailyAOI*100, score.DailyEMA*100,
		score.DailyStruct*100, score.DailyPattern*100)
	details += fmt.Sprintf("  H4: Favor=%.1f%% EMA=%.1f%% Reject=%.1f%% Struct=%.1f%% Pattern=%.1f%%\n",
		score.H4Favor*100, score.H4EMA*100, score.H4Reject*100,
		score.H4Struct*100, score.H4Pattern*100)
	
	s.file.WriteString(entry)
	s.file.WriteString(details)
	s.file.WriteString("\n")
	s.file.Sync()
	
	// Also print to console
	fmt.Print(entry)
	fmt.Print(details)
}

func (s *SignalLog) Close() {
	s.file.Close()
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("           PAPER TRADING - CONFLUENCE SCORING ENGINE")
	fmt.Println("================================================================================")
	fmt.Printf("Threshold: %.0f%% | Check Interval: %v\n", CONFLUENCE_THRESHOLD, CHECK_INTERVAL)
	fmt.Printf("Pairs: %v\n", MAJOR_PAIRS)
	fmt.Println("================================================================================")
	fmt.Println()

	// Database connection
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

	// Signal logger
	signalLog, err := NewSignalLog("signals.log")
	if err != nil {
		log.Fatalf("Failed to create signal log: %v", err)
	}
	defer signalLog.Close()

	// Confluence scorer
	config := confluence.ConfluenceConfig{
		MinConfluencePercent: CONFLUENCE_THRESHOLD,
	}
	scorer := confluence.NewConfluenceScorer(config)

	// Stats tracking
	signalCount := 0
	startTime := time.Now()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🚀 Paper trading started. Press Ctrl+C to stop.")
	fmt.Println()

	// Initial check
	checkAllPairs(ctx, queries, scorer, signalLog, &signalCount)

	// Main loop
	ticker := time.NewTicker(CHECK_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkAllPairs(ctx, queries, scorer, signalLog, &signalCount)
			
		case <-sigChan:
			elapsed := time.Since(startTime)
			fmt.Println("\n================================================================================")
			fmt.Println("           PAPER TRADING SESSION SUMMARY")
			fmt.Println("================================================================================")
			fmt.Printf("Duration: %v\n", elapsed.Round(time.Second))
			fmt.Printf("Total Signals: %d\n", signalCount)
			if elapsed.Hours() > 0 {
				fmt.Printf("Signals/Hour: %.2f\n", float64(signalCount)/elapsed.Hours())
				fmt.Printf("Projected Signals/Week: %.1f\n", float64(signalCount)/elapsed.Hours()*168)
			}
			fmt.Println("\nSignals logged to: signals.log")
			return
		}
	}
}

func checkAllPairs(ctx context.Context, queries *db.Queries, scorer *confluence.ConfluenceScorer, signalLog *SignalLog, signalCount *int) {
	fmt.Printf("[%s] Checking all pairs...\n", time.Now().UTC().Format("15:04:05"))
	
	for _, symbol := range MAJOR_PAIRS {
		checkPairConfluence(ctx, queries, scorer, signalLog, symbol, signalCount)
	}
	
	fmt.Printf("[%s] Check complete. Next check in %v\n\n", time.Now().UTC().Format("15:04:05"), CHECK_INTERVAL)
}

func checkPairConfluence(ctx context.Context, queries *db.Queries, scorer *confluence.ConfluenceScorer, signalLog *SignalLog, symbol string, signalCount *int) {
	// Load latest candles for each timeframe
	weeklyCandles, err := loadLatestWeeklyCandles(ctx, queries, symbol, 60)
	if err != nil || len(weeklyCandles) < 30 {
		fmt.Printf("  %s: Weekly data issue (%v candles)\n", symbol, len(weeklyCandles))
		return
	}

	dailyCandles, err := loadLatestDailyCandles(ctx, queries, symbol, 300)
	if err != nil || len(dailyCandles) < 60 {
		fmt.Printf("  %s: Daily data issue (%v candles)\n", symbol, len(dailyCandles))
		return
	}

	h4Candles, err := loadLatestH4Candles(ctx, queries, symbol, 500)
	if err != nil || len(h4Candles) < 120 {
		fmt.Printf("  %s: H4 data issue (%v candles)\n", symbol, len(h4Candles))
		return
	}

	// Calculate confluence
	score, signal := scorer.CalculateConfluence(
		weeklyCandles, dailyCandles, h4Candles,
		nil, nil, nil, // No H2, H1, M30 yet
		symbol,
	)

	// Debug: Show all confluence scores
	fmt.Printf("  %s: Confluence=%.1f%% (W1:%d D1:%d H4:%d)\n", 
		symbol, score.TotalPercent, len(weeklyCandles), len(dailyCandles), len(h4Candles))

	// Check if tradeable
	if score.TotalPercent >= CONFLUENCE_THRESHOLD {
		direction := "LONG"
		if signal != nil {
			direction = signal.Direction
		} else {
			// Determine direction from structure
			lastCandle := h4Candles[len(h4Candles)-1]
			if score.WeeklyFavor > 0 && lastCandle.Close < lastCandle.Open {
				direction = "SHORT"
			}
		}
		
		currentPrice := h4Candles[len(h4Candles)-1].Close
		signalLog.Log(symbol, direction, currentPrice, score.TotalPercent, score)
		*signalCount++
	}
}

func loadLatestWeeklyCandles(ctx context.Context, queries *db.Queries, symbol string, limit int) ([]patterns.Candle, error) {
	// Use current date range (data now goes to Feb 2026)
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(-3, 0, 0)) // 3 years back

	dbCandles, err := queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	// Take last N candles
	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
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

func loadLatestDailyCandles(ctx context.Context, queries *db.Queries, symbol string, limit int) ([]patterns.Candle, error) {
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(-2, 0, 0)) // 2 years back

	dbCandles, err := queries.GetCandlesD1BySymbolAndRange(ctx, db.GetCandlesD1BySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
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

func loadLatestH4Candles(ctx context.Context, queries *db.Queries, symbol string, limit int) ([]patterns.Candle, error) {
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(0, -6, 0)) // 6 months back

	dbCandles, err := queries.GetCandlesH4BySymbolAndRange(ctx, db.GetCandlesH4BySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
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
