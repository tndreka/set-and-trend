package backtest

import (
	"context"
	"fmt"
	"log"
	//"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/analyzer"
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

	// 1. Initialize Portfolio ($2000, 500 leverage)
	p := portfolio.New(2000, 3, 0.01)

	// 2. Get Data
	symbols := getAllSymbols(ctx, pool)
	weeks := getWeeks(pool)

	// Map: WeekKey -> Signals
	signalsByWeek := make(map[string][]*engine.BacktestSignal)

	// ========== PHASE 1: Generate Signals ==========
	log.Printf("📊 PHASE 1: Generating H4 signals for %d symbols...", len(symbols))

	for _, symbol := range symbols {
		// Panic recovery for missing symbol configs
		var eng *engine.TradingEngine
		func() {
			defer func() { recover() }()
			eng = engine.NewTradingEngine(symbol, engine.DefaultEngineConfig())
		}()

		if eng == nil {
			continue
		}

		candles := loadH4History(ctx, pool, symbol)
		if len(candles) < 200 {
			continue
		}

		signals := eng.RunBacktest(candles)
		for _, sig := range signals {
			y, w := sig.Timestamp.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", y, w)
			signalsByWeek[weekKey] = append(signalsByWeek[weekKey], sig)
		}
	}

	// ========== PHASE 1.5: Quality Analysis ==========
	log.Println("\n🔍 PHASE 1.5: Analyzing Signal Quality...")

	allSignalsList := []*engine.BacktestSignal{}
	for _, signals := range signalsByWeek {
		allSignalsList = append(allSignalsList, signals...)
	}

	batchAnalyzer := analyzer.NewBatchAnalyzer()
	analysisResult := batchAnalyzer.AnalyzeAll(allSignalsList)
	analysisResult.PrintSummary()

	// ========== PHASE 2: Filtering & Execution ==========
	log.Println("\n🎯 FILTERING & SIMULATING PORTFOLIO...")

	for _, week := range weeks {
		weekSignals := signalsByWeek[week.WeekKey]
		if len(weekSignals) == 0 {
			continue
		}

		filtered := []*engine.BacktestSignal{}
		for _, sig := range weekSignals {
			var report *analyzer.SignalQualityReport
			for _, r := range analysisResult.Reports {
				if r.Symbol == sig.Symbol && r.SignalTime.Equal(sig.Timestamp) {
					report = r
					break
				}
			}

			// Loosen the filter: If the engine is 75%+ confident, we take it 
			// unless the analyzer says it's a total failure.
			if report != nil && report.RiskScore >= 50 { 
				filtered = append(filtered, sig)
			}
		}

		if len(filtered) == 0 {
			continue
		}

		sel := selector.New(week.WeekKey, 2)
		for _, s := range filtered {
			sel.Add(s)
		}
		
		top2 := sel.Top(2)
		if len(top2) > 0 {
			// Remove the debug noise, just log the execution
			p.Execute(top2)
		}
	}

	p.Report()
}

// --- Helper Functions using Direct SQL ---

func loadH4History(ctx context.Context, pool *pgxpool.Pool, symbol string) []patterns.Candle {
	query := `SELECT timestamp_utc, open, high, low, close FROM candles_h4 WHERE symbol = $1 ORDER BY timestamp_utc ASC`
	rows, err := pool.Query(ctx, query, symbol)
	if err != nil {
		return []patterns.Candle{}
	}
	defer rows.Close()

	var candles []patterns.Candle
	for i := 0; rows.Next(); i++ {
		var c patterns.Candle
		if err := rows.Scan(&c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close); err == nil {
			c.Index = i
			candles = append(candles, c)
		}
	}
	return candles
}

func getAllSymbols(ctx context.Context, pool *pgxpool.Pool) []string {
	var symbols []string
	pool.QueryRow(ctx, "SELECT array_agg(DISTINCT symbol) FROM candles_h4").Scan(&symbols)
	return symbols
}

func getWeeks(pool *pgxpool.Pool) []Week {
	ctx := context.Background()
	query := `
		SELECT DISTINCT 
			TO_CHAR(DATE_TRUNC('week', timestamp_utc), 'YYYY-"W"IW') as wk,
			DATE_TRUNC('week', timestamp_utc) as start_date,
			DATE_TRUNC('week', timestamp_utc) + INTERVAL '6 days 23:59:59' as end_date
		FROM candles_h4 
		ORDER BY start_date ASC 
	`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return []Week{}
	}
	defer rows.Close()

	var weeks []Week
	for rows.Next() {
		var w Week
		if err := rows.Scan(&w.WeekKey, &w.Start, &w.End); err == nil {
			weeks = append(weeks, w)
		}
	}
	return weeks
}

/*
 4H trades

*/
/*
package backtest

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/engine"
	"set-and-trend/backend/internal/patterns"
	"set-and-trend/backend/internal/portfolio"
)

type Week struct {
	WeekKey string
	Start   time.Time
	End     time.Time
}

func Run(pool *pgxpool.Pool, queries *db.Queries) {
	ctx := context.Background()
	p := portfolio.New(2000.0, 100)

	symbols := getAllSymbols(ctx, pool)
	weeks := getWeeks(pool)

	signalMap := make(map[string][]*engine.BacktestSignal)

	log.Printf("📊 PHASE 1: Generating H4 signals for %d symbols...", len(symbols))

	for _, symbol := range symbols {
		// ✅ Panic Recovery: If a symbol is missing from SymbolRegistry, it won't crash the whole app
		var eng *engine.TradingEngine
		func() {
			defer func() {
				if r := recover(); r != nil {
					eng = nil // Mark as failed
				}
			}()
			eng = engine.NewTradingEngine(symbol, engine.DefaultEngineConfig())
		}()

		if eng == nil {
			log.Printf("⚠️  Skipping %s: Not configured in constants/forex.go", symbol)
			continue
		}

		candles := loadH4History(ctx, pool, symbol)
		if len(candles) < 200 {
			continue
		}
p := port
		allSignals := eng.RunBacktest(candles)
		for _, sig := range allSignals {
			y, w := sig.Timestamp.ISOWeek()
			actualKey := fmt.Sprintf("%d-W%02d", y, w)
			signalMap[actualKey] = append(signalMap[actualKey], sig)
		}
	}

	log.Printf("💰 PHASE 2: Running H4 Portfolio Simulation...")

	for _, week := range weeks {
		signalsThisWeek := signalMap[week.WeekKey]
		if len(signalsThisWeek) == 0 {
			continue
		}

		sort.Slice(signalsThisWeek, func(i, j int) bool {
			return signalsThisWeek[i].Confidence > signalsThisWeek[j].Confidence
		})

		var top2 []*engine.BacktestSignal
		if len(signalsThisWeek) > 2 {
			top2 = signalsThisWeek[:2]
		} else {
			top2 = signalsThisWeek
		}

		log.Printf("🎯 %s: Found %d H4 signals, trading top %d", week.WeekKey, len(signalsThisWeek), len(top2))
		// ✅ FIX: Convert []*engine.BacktestSignal to []*patterns.TradeSignal
		
		var signalsToExecute []*patterns.TradeSignal
		for _, bts := range top2 {
			signalsToExecute = append(signalsToExecute, bts.TradeSignal)
		}

		// Pass the basic signals to the portfolio
		p.Execute(top2)
	}
	
	p.Report()
}

// ✅ FIXED: Uses direct pool query to bypass missing SQLC methods
func loadH4History(ctx context.Context, pool *pgxpool.Pool, symbol string) []patterns.Candle {
	query := `
		SELECT timestamp_utc, open, high, low, close 
		FROM candles_h4 
		WHERE symbol = $1 
		ORDER BY timestamp_utc ASC
	`
	rows, err := pool.Query(ctx, query, symbol)
	if err != nil {
		log.Printf("⚠️ Error loading H4 for %s: %v", symbol, err)
		return []patterns.Candle{}
	}
	defer rows.Close()

	var candles []patterns.Candle
	i := 0
	for rows.Next() {
		var c patterns.Candle
		var ts time.Time
		if err := rows.Scan(&ts, &c.Open, &c.High, &c.Low, &c.Close); err != nil {
			continue
		}
		c.Timestamp = ts
		c.Index = i
		candles = append(candles, c)
		i++
	}
	return candles
}

// ✅ FIXED: Uses direct pool query
func getAllSymbols(ctx context.Context, pool *pgxpool.Pool) []string {
	var symbols []string
	query := `SELECT DISTINCT symbol FROM candles_h4`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			symbols = append(symbols, s)
		}
	}
	return symbols
}

func getWeeks(pool *pgxpool.Pool) []Week {
	ctx := context.Background()
	query := `
		SELECT DISTINCT 
			TO_CHAR(DATE_TRUNC('week', timestamp_utc), 'YYYY-"W"IW') as wk,
			DATE_TRUNC('week', timestamp_utc) as start_date,
			DATE_TRUNC('week', timestamp_utc) + INTERVAL '6 days 23:59:59' as end_date
		FROM candles_h4 
		ORDER BY start_date ASC 
	`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return []Week{}
	}
	defer rows.Close()
	
	var weeks []Week
	for rows.Next() {
		var w Week
		if err := rows.Scan(&w.WeekKey, &w.Start, &w.End); err == nil {
			weeks = append(weeks, w)
		}
	}
	return weeks
}

*/






// WEEKLY TRADES


// package backtest

// import (
// 	"context"
// 	"fmt" // ✅ Added for WeekKey formatting
// 	"log"
// 	"sort"
// 	"time"

// 	"github.com/jackc/pgx/v5/pgtype"
// 	"github.com/jackc/pgx/v5/pgxpool"

// 	"set-and-trend/backend/internal/db"
// 	"set-and-trend/backend/internal/engine" // ✅ Contains BacktestSignal
// 	"set-and-trend/backend/internal/patterns"
// 	"set-and-trend/backend/internal/portfolio"
// )

// type Week struct {
// 	WeekKey string
// 	Start   time.Time
// 	End     time.Time
// }

// func Run(pool *pgxpool.Pool, queries *db.Queries) {
// 	ctx := context.Background()
	
// 	// Initializing portfolio: $2000 balance, 3% risk per trade
// 	p := portfolio.New(2000.0, 3.0)

// 	symbols := getAllSymbols(ctx, queries)
// 	weeks := getWeeks(pool)

// 	// Map to store: WeekKey -> List of Signals from all symbols found for that week
// 	signalMap := make(map[string][]*engine.BacktestSignal)

// 	log.Printf("📊 PHASE 1: Generating signals for %d symbols...", len(symbols))

// 	for _, symbol := range symbols {
// 		// Load full historical candles once per symbol
// 		candles := loadFullHistory(ctx, queries, symbol)
// 		if len(candles) < 200 {
// 			continue
// 		}

// 		// Run the unified trading engine
// 		eng := engine.NewTradingEngine(symbol, engine.DefaultEngineConfig())
		
// 		// RunBacktest is a pure function that returns all historical signals for this symbol
// 		allSignals := eng.RunBacktest(candles)
		
// 		for _, sig := range allSignals {
// 			// Generate WeekKey (e.g., "2024-W05") to match database keys
// 			y, w := sig.Timestamp.ISOWeek()
// 			actualKey := fmt.Sprintf("%d-W%02d", y, w)
// 			signalMap[actualKey] = append(signalMap[actualKey], sig)
// 		}
// 	}

// 	log.Printf("💰 PHASE 2: Running 10-Year Portfolio Simulation...")

// 	// Iterate through weeks in chronological order
// 	for _, week := range weeks {
// 		signalsThisWeek := signalMap[week.WeekKey]
		
// 		if len(signalsThisWeek) == 0 {
// 			continue
// 		}

// 		// Sort signals by confidence (Highest confidence first)
// 		sort.Slice(signalsThisWeek, func(i, j int) bool {
// 			return signalsThisWeek[i].Confidence > signalsThisWeek[j].Confidence
// 		})

// 		// Select Top 2 signals to trade this week
// 		var top2 []*engine.BacktestSignal
// 		if len(signalsThisWeek) > 2 {
// 			top2 = signalsThisWeek[:2]
// 		} else {
// 			top2 = signalsThisWeek
// 		}

// 		log.Printf("🎯 %s: Found %d signals, trading top %d", week.WeekKey, len(signalsThisWeek), len(top2))
		
// 		// Note: p.Execute must be able to handle []*engine.BacktestSignal
// 		// If it only takes *patterns.TradeSignal, you may need a loop here.
// 		p.Execute(top2)
// 	}
	
// 	p.Report()
// }

// func loadFullHistory(ctx context.Context, queries *db.Queries, symbol string) []patterns.Candle {
// 	// Query dates spanning from year 2000 to now
// 	startPg := pgtype.Timestamptz{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
// 	endPg := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	
// 	dbCandles, err := queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
// 		Symbol:         symbol,
// 		TimestampUtc:   startPg,
// 		TimestampUtc_2: endPg,
// 	})
	
// 	if err != nil {
// 		log.Printf("⚠️ Error loading candles for %s: %v", symbol, err)
// 		return []patterns.Candle{}
// 	}
	
// 	candles := make([]patterns.Candle, len(dbCandles))
// 	for i, c := range dbCandles {
// 		candles[i] = patterns.Candle{
// 			Index:     i,
// 			Timestamp: c.TimestampUtc.Time,
// 			Open:      c.Open.InexactFloat64(),
// 			High:      c.High.InexactFloat64(),
// 			Low:       c.Low.InexactFloat64(),
// 			Close:     c.Close.InexactFloat64(),
// 			Volume:    volumeToInt64(c.Volume),
// 		}
// 	}
// 	return candles
// }

// func getAllSymbols(ctx context.Context, queries *db.Queries) []string {
// 	rows, err := queries.GetDistinctSymbolsFromWeekly(ctx)
// 	if err != nil {
// 		log.Printf("⚠️ Error fetching symbols: %v", err)
// 		return []string{}
// 	}
// 	return rows
// }

// func getWeeks(pool *pgxpool.Pool) []Week {
// 	ctx := context.Background()
	
// 	query := `
// 		SELECT DISTINCT 
// 			TO_CHAR(DATE_TRUNC('week', timestamp_utc), 'YYYY-"W"IW') as wk,
// 			DATE_TRUNC('week', timestamp_utc) as start_date,
// 			DATE_TRUNC('week', timestamp_utc) + INTERVAL '6 days 23:59:59' as end_date
// 		FROM candles_weekly 
// 		ORDER BY start_date ASC 
// 	`
	
// 	rows, err := pool.Query(ctx, query)
// 	if err != nil {
// 		log.Printf("⚠️ Error fetching weeks: %v", err)
// 		return []Week{}
// 	}
// 	defer rows.Close()
	
// 	var weeks []Week
// 	for rows.Next() {
// 		var w Week
// 		if err := rows.Scan(&w.WeekKey, &w.Start, &w.End); err != nil {
// 			continue
// 		}
// 		weeks = append(weeks, w)
// 	}
	
// 	return weeks
// }

// func volumeToInt64(v pgtype.Int8) int64 {
// 	if v.Valid {
// 		return v.Int64
// 	}
// 	return 0
// }