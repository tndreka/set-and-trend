package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"set-and-trend/backend/internal/database"
	"set-and-trend/backend/internal/indicators"
)

func main() {
	dbPath := flag.String("db", "forex_prices.db", "Path to forex_prices.db SQLite database")
	symbol := flag.String("symbol", "", "Single symbol to backtest (e.g., EURUSD)")
	allSymbols := flag.Bool("all", false, "Run backtest on all symbols")
	timeframe := flag.String("tf", "h4", "Timeframe: m30, h1, h4, daily")
	useADX := flag.Bool("adx", true, "Use ADX trend filter")
	checkData := flag.Bool("check-data", false, "Only check database contents")

	flag.Parse()

	loader, err := database.NewSQLiteLoader(*dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer loader.Close()

	fmt.Printf("Database: %s\n", *dbPath)

	var tf database.Timeframe
	switch strings.ToLower(*timeframe) {
	case "m30":
		tf = database.TimeframeM30
	case "h1", "hourly":
		tf = database.TimeframeH1
	case "h4":
		tf = database.TimeframeH4
	case "daily", "d1":
		tf = database.TimeframeDaily
	default:
		tf = database.TimeframeH4
	}

	if *checkData {
		checkDataContents(loader)
		return
	}

	fmt.Printf("\n=== H&S Pattern Backtester ===\n")
	fmt.Printf("Timeframe: %s\n", tf)
	fmt.Printf("ADX Filter: %v\n", *useADX)

	if *allSymbols {
		fmt.Println("\nRunning backtest on all symbols...")
		runAllSymbols(loader, tf, *useADX)
	} else if *symbol != "" {
		fmt.Printf("\nRunning backtest on %s...\n", *symbol)
		runSingleSymbol(loader, strings.ToUpper(*symbol), tf, *useADX)
	} else {
		fmt.Println("\nUsage:")
		fmt.Println("  -symbol EURUSD  : Backtest single symbol")
		fmt.Println("  -all            : Backtest all symbols")
		fmt.Println("  -check-data     : Show database contents")
		fmt.Println("\nExample:")
		fmt.Println("  ./backtest -db forex_prices.db -all -tf h4")
	}
}

func checkDataContents(loader *database.SQLiteLoader) {
	fmt.Println("\n=== Database Contents ===\n")
	stats, err := loader.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		return
	}

	for tf, tfStats := range stats.TimeframeStats {
		fmt.Printf("%s:\n", tf.TableName())
		fmt.Printf("  Total bars: %d\n", tfStats.TotalBars)
		fmt.Printf("  Symbols: %d\n", tfStats.SymbolCount)
		if !tfStats.StartDate.IsZero() {
			fmt.Printf("  Date range: %s to %s\n",
				tfStats.StartDate.Format("2006-01-02"),
				tfStats.EndDate.Format("2006-01-02"))
		}
		fmt.Println()
	}

	symbols, err := loader.GetSymbols(database.TimeframeH4)
	if err == nil {
		fmt.Printf("H4 Symbols (%d):\n", len(symbols))
		for i, s := range symbols {
			fmt.Printf("  %s", s)
			if (i+1)%7 == 0 {
				fmt.Println()
			}
		}
		fmt.Println()
	}
}

type BacktestResult struct {
	Symbol        string
	TotalPatterns int
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnLR     float64
	Expectancy    float64
}

func runSingleSymbol(loader *database.SQLiteLoader, symbol string, tf database.Timeframe, useADX bool) {
	candles, err := loader.GetCandles(symbol, tf, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(candles) < 100 {
		fmt.Printf("Insufficient data: only %d candles\n", len(candles))
		return
	}

	result := runBacktest(candles, symbol, useADX)
	printResult(result, candles[0].Timestamp, candles[len(candles)-1].Timestamp)
}

func runAllSymbols(loader *database.SQLiteLoader, tf database.Timeframe, useADX bool) {
	symbols, err := loader.GetSymbols(tf)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var results []BacktestResult
	totalPatterns := 0
	totalTrades := 0
	totalWins := 0
	totalLosses := 0
	totalPnLR := 0.0

	for _, symbol := range symbols {
		candles, err := loader.GetCandles(symbol, tf, nil)
		if err != nil || len(candles) < 100 {
			continue
		}

		result := runBacktest(candles, symbol, useADX)
		results = append(results, result)

		totalPatterns += result.TotalPatterns
		totalTrades += result.TotalTrades
		totalWins += result.WinningTrades
		totalLosses += result.LosingTrades
		totalPnLR += result.TotalPnLR
	}

	fmt.Printf("\n=== Portfolio Backtest Results ===\n")
	fmt.Printf("Symbols Tested: %d\n", len(results))
	fmt.Printf("\nTotal Patterns: %d\n", totalPatterns)
	fmt.Printf("Total Trades: %d\n", totalTrades)
	fmt.Printf("Winning Trades: %d\n", totalWins)
	fmt.Printf("Losing Trades: %d\n", totalLosses)

	if totalTrades > 0 {
		fmt.Printf("\nWin Rate: %.1f%%\n", float64(totalWins)/float64(totalTrades)*100)
		fmt.Printf("Expectancy: %.2fR\n", totalPnLR/float64(totalTrades))
		fmt.Printf("Total P&L (R): %.2fR\n", totalPnLR)
	}

	fmt.Printf("\n--- Per Symbol ---\n")
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalTrades > results[j].TotalTrades
	})
	for _, r := range results {
		if r.TotalTrades > 0 {
			fmt.Printf("%s: %d trades, %.0f%% win rate, %.2fR\n",
				r.Symbol, r.TotalTrades, r.WinRate*100, r.TotalPnLR)
		}
	}
}

func runBacktest(candles []database.Candle, symbol string, useADX bool) BacktestResult {
	result := BacktestResult{Symbol: symbol}

	// Calculate indicators
	adx := indicators.NewADX(14)
	adx.Calculate(candles)

	atr := indicators.NewATR(14)
	atr.Calculate(candles)

	// Detect swing points and patterns (simplified)
	swingStrength := 5
	var swingHighs, swingLows []int

	for i := swingStrength; i < len(candles)-swingStrength; i++ {
		isHigh := true
		isLow := true
		for j := 1; j <= swingStrength; j++ {
			if candles[i-j].High >= candles[i].High || candles[i+j].High >= candles[i].High {
				isHigh = false
			}
			if candles[i-j].Low <= candles[i].Low || candles[i+j].Low <= candles[i].Low {
				isLow = false
			}
		}
		if isHigh {
			swingHighs = append(swingHighs, i)
		}
		if isLow {
			swingLows = append(swingLows, i)
		}
	}

	// Look for H&S patterns
	for i := 0; i < len(swingHighs)-2; i++ {
		leftIdx := swingHighs[i]
		headIdx := swingHighs[i+1]
		rightIdx := swingHighs[i+2]

		leftPrice := candles[leftIdx].High
		headPrice := candles[headIdx].High
		rightPrice := candles[rightIdx].High

		// Head must be higher
		if headPrice <= leftPrice || headPrice <= rightPrice {
			continue
		}

		// Check symmetry
		leftHeight := headPrice - leftPrice
		rightHeight := headPrice - rightPrice
		symmetry := 1 - math.Abs(leftHeight-rightHeight)/((leftHeight+rightHeight)/2)
		if symmetry < 0.6 {
			continue
		}

		result.TotalPatterns++

		// Apply ADX filter
		if useADX && !adx.IsTrending() {
			continue
		}

		// Simulate trade
		entryIdx := rightIdx + swingStrength
		if entryIdx >= len(candles) {
			continue
		}

		entryPrice := candles[entryIdx].Close
		stopLoss := headPrice * 1.01
		risk := stopLoss - entryPrice
		takeProfit := entryPrice - risk*2

		// Find exit
		var exitPrice float64
		var isWin bool
		for j := entryIdx + 1; j < len(candles); j++ {
			if candles[j].High >= stopLoss {
				exitPrice = stopLoss
				isWin = false
				break
			}
			if candles[j].Low <= takeProfit {
				exitPrice = takeProfit
				isWin = true
				break
			}
			if j-entryIdx > 100 {
				exitPrice = candles[j].Close
				isWin = exitPrice < entryPrice
				break
			}
		}

		result.TotalTrades++
		if isWin {
			result.WinningTrades++
			result.TotalPnLR += 2.0
		} else {
			result.LosingTrades++
			result.TotalPnLR -= 1.0
		}
	}

	// Look for IHS patterns
	for i := 0; i < len(swingLows)-2; i++ {
		leftIdx := swingLows[i]
		headIdx := swingLows[i+1]
		rightIdx := swingLows[i+2]

		leftPrice := candles[leftIdx].Low
		headPrice := candles[headIdx].Low
		rightPrice := candles[rightIdx].Low

		// Head must be lower
		if headPrice >= leftPrice || headPrice >= rightPrice {
			continue
		}

		// Check symmetry
		leftDepth := leftPrice - headPrice
		rightDepth := rightPrice - headPrice
		symmetry := 1 - math.Abs(leftDepth-rightDepth)/((leftDepth+rightDepth)/2)
		if symmetry < 0.6 {
			continue
		}

		result.TotalPatterns++

		// Apply ADX filter
		if useADX && !adx.IsTrending() {
			continue
		}

		// Simulate trade
		entryIdx := rightIdx + swingStrength
		if entryIdx >= len(candles) {
			continue
		}

		entryPrice := candles[entryIdx].Close
		stopLoss := headPrice * 0.99
		risk := entryPrice - stopLoss
		takeProfit := entryPrice + risk*2

		// Find exit
		var exitPrice float64
		var isWin bool
		for j := entryIdx + 1; j < len(candles); j++ {
			if candles[j].Low <= stopLoss {
				exitPrice = stopLoss
				isWin = false
				break
			}
			if candles[j].High >= takeProfit {
				exitPrice = takeProfit
				isWin = true
				break
			}
			if j-entryIdx > 100 {
				exitPrice = candles[j].Close
				isWin = exitPrice > entryPrice
				break
			}
		}

		result.TotalTrades++
		if isWin {
			result.WinningTrades++
			result.TotalPnLR += 2.0
		} else {
			result.LosingTrades++
			result.TotalPnLR -= 1.0
		}
	}

	if result.TotalTrades > 0 {
		result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades)
		result.Expectancy = result.TotalPnLR / float64(result.TotalTrades)
	}

	return result
}

func printResult(result BacktestResult, startDate, endDate time.Time) {
	fmt.Printf("\n=== %s Backtest Results ===\n", result.Symbol)
	fmt.Printf("Period: %s to %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	fmt.Printf("\nPatterns Detected: %d\n", result.TotalPatterns)
	fmt.Printf("Total Trades: %d\n", result.TotalTrades)
	fmt.Printf("Winning Trades: %d\n", result.WinningTrades)
	fmt.Printf("Losing Trades: %d\n", result.LosingTrades)
	fmt.Printf("\nWin Rate: %.1f%%\n", result.WinRate*100)
	fmt.Printf("Expectancy: %.2fR\n", result.Expectancy)
	fmt.Printf("Total P&L (R): %.2fR\n", result.TotalPnLR)
}
