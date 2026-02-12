package backtest

import (
	"context"
	"log"
	"time"
	
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"  // ✅ ADDED
	
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/engine"
	"set-and-trend/backend/internal/patterns"
	"set-and-trend/backend/internal/portfolio"
	"set-and-trend/backend/internal/selector"
)

type Week struct {
	WeekKey string
	Start   time.Time
	End     time.Time
}

func Run(pool *pgxpool.Pool, queries *db.Queries) {
	ctx := context.Background()
	
	p := portfolio.New(2000.0, 3.0)
	
	// Get all unique symbols from database
	symbols := getAllSymbols(ctx, queries)
	
	// Get all weeks
	weeks := getWeeks(pool)
	
	log.Printf("🚀 ELITE BACKTEST: %d weeks × %d symbols", len(weeks), len(symbols))
	
	for i, week := range weeks {
		if i%52 == 0 { 
			log.Printf("⏳ Year %d/%d", i/52+1, len(weeks)/52) 
		}
		
		sel := selector.New(week.WeekKey, 2)
		
		for _, symbol := range symbols {
			// Load weekly candles for this symbol
			candles := loadWeek(ctx, queries, symbol, week.Start, week.End)
			
			if len(candles) < 200 { 
				continue 
			}
			
			// Run trading engine on weekly data
			eng := engine.NewTradingEngine(symbol, engine.DefaultEngineConfig())
			signals := eng.RunBacktest(candles)
			
			for _, sig := range signals {
				sel.Add(sig)
			}
		}
		
		// Get top 2 signals for this week
		top2 := sel.Top(2)
		sel.Print()
		
		// Execute trades on top 2
		p.Execute(top2)
	}
	
	// Print final report
	p.Report()
}

// getAllSymbols retrieves all unique symbols from candles_weekly table
func getAllSymbols(ctx context.Context, queries *db.Queries) []string {
	rows, err := queries.GetDistinctSymbolsFromWeekly(ctx)
	if err != nil {
		log.Printf("⚠️  Error fetching symbols: %v", err)
		return []string{}
	}
	return rows
}

// getWeeks retrieves all weeks from the database
func getWeeks(pool *pgxpool.Pool) []Week {
	ctx := context.Background()
	
	query := `
		SELECT DISTINCT 
			TO_CHAR(DATE_TRUNC('week', timestamp_utc), 'YYYY-"W"IW'),
			DATE_TRUNC('week', timestamp_utc),
			DATE_TRUNC('week', timestamp_utc) + INTERVAL '6 days 23:59:59'
		FROM candles_weekly 
		ORDER BY 2 DESC 
		LIMIT 520
	`
	
	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Printf("⚠️  Error fetching weeks: %v", err)
		return []Week{}
	}
	defer rows.Close()
	
	var weeks []Week
	for rows.Next() {
		var w Week
		if err := rows.Scan(&w.WeekKey, &w.Start, &w.End); err != nil {
			log.Printf("⚠️  Error scanning week: %v", err)
			continue
		}
		weeks = append(weeks, w)
	}
	
	return weeks
}

// loadWeek loads weekly candles for a symbol within a date range
func loadWeek(ctx context.Context, queries *db.Queries, symbol string, start, end time.Time) []patterns.Candle {
	// ✅ FIXED: Convert time.Time to pgtype.Timestamptz
	var startPg, endPg pgtype.Timestamptz
	startPg.Scan(start)
	endPg.Scan(end)
	
	// Fetch candles from database
	dbCandles, err := queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startPg,
		TimestampUtc_2: endPg,
	})
	
	if err != nil {
		log.Printf("⚠️  Error loading candles for %s: %v", symbol, err)
		return []patterns.Candle{}
	}
	
	// Convert to patterns.Candle format
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

// volumeToInt64 converts pgtype.Int8 to int64
func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}