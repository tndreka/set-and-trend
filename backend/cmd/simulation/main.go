package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"set-and-trend/backend/internal/database"
	"set-and-trend/backend/internal/indicators"
	"set-and-trend/backend/internal/mtf"
	"set-and-trend/backend/internal/scoring"
)

// SimulationConfig holds simulation parameters
type SimulationConfig struct {
	Symbol              string
	Timeframe           database.Timeframe
	ConfidenceThreshold float64
	MinRR               float64
	MaxBarsToHold       int
	CooldownBars        int
	SwingStrength       int
	UseSMCFilters       bool
	MinGrade            string // Minimum grade to trade (A, B, C)
}

// Position represents an open trade
type Position struct {
	Direction   int
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	EntryBar    int
	PatternType string
	Grade       string
}

// TradeResult represents a closed trade
type TradeResult struct {
	Symbol      string
	PatternType string
	Direction   int
	EntryPrice  float64
	ExitPrice   float64
	StopLoss    float64
	TakeProfit  float64
	EntryBar    int
	ExitBar     int
	PnLR        float64
	Outcome     string
	BarsHeld    int
	Grade       string
	Score       int
}

// MarketFeeder provides bar-by-bar data WITHOUT lookahead
type MarketFeeder struct {
	candles      []database.Candle
	currentIndex int
}

func NewMarketFeeder(candles []database.Candle) *MarketFeeder {
	copied := make([]database.Candle, len(candles))
	copy(copied, candles)
	return &MarketFeeder{candles: copied, currentIndex: 0}
}

func (f *MarketFeeder) HasMoreBars() bool {
	return f.currentIndex < len(f.candles)
}

func (f *MarketFeeder) NextBar() *database.Candle {
	if !f.HasMoreBars() {
		return nil
	}
	bar := f.candles[f.currentIndex]
	f.currentIndex++
	return &bar
}

func (f *MarketFeeder) GetVisibleWindow(size int) []database.Candle {
	if f.currentIndex == 0 {
		return nil
	}
	start := f.currentIndex - size
	if start < 0 {
		start = 0
	}
	visible := make([]database.Candle, f.currentIndex-start)
	copy(visible, f.candles[start:f.currentIndex])
	return visible
}

func (f *MarketFeeder) CurrentIndex() int {
	return f.currentIndex
}

func (f *MarketFeeder) TotalBars() int {
	return len(f.candles)
}

// SwingPoint represents a significant high or low
type SwingPoint struct {
	Index  int
	Price  float64
	IsHigh bool
}

func detectSwings(candles []database.Candle, strength int) []SwingPoint {
	var swings []SwingPoint
	for i := strength; i < len(candles)-strength; i++ {
		isHigh := true
		isLow := true
		for j := 1; j <= strength; j++ {
			if candles[i-j].High >= candles[i].High || candles[i+j].High >= candles[i].High {
				isHigh = false
			}
			if candles[i-j].Low <= candles[i].Low || candles[i+j].Low <= candles[i].Low {
				isLow = false
			}
		}
		if isHigh {
			swings = append(swings, SwingPoint{Index: i, Price: candles[i].High, IsHigh: true})
		}
		if isLow {
			swings = append(swings, SwingPoint{Index: i, Price: candles[i].Low, IsHigh: false})
		}
	}
	return swings
}

// Signal represents a trading signal
type Signal struct {
	PatternType string
	Direction   int
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	Confidence  float64
	RiskReward  float64
	BarIndex    int
}

func detectPatterns(candles []database.Candle, config SimulationConfig, adx *indicators.ADX) *Signal {
	if len(candles) < 30 {
		return nil
	}

	swings := detectSwings(candles, config.SwingStrength)

	var highs, lows []SwingPoint
	for _, s := range swings {
		if s.IsHigh {
			highs = append(highs, s)
		} else {
			lows = append(lows, s)
		}
	}

	// Check H&S (bearish)
	if len(highs) >= 3 {
		for i := len(highs) - 3; i >= 0; i-- {
			left := highs[i]
			head := highs[i+1]
			right := highs[i+2]

			if head.Price <= left.Price || head.Price <= right.Price {
				continue
			}

			leftH := head.Price - left.Price
			rightH := head.Price - right.Price
			symmetry := 1 - math.Abs(leftH-rightH)/((leftH+rightH)/2)
			if symmetry < 0.6 {
				continue
			}

			var neckLows []SwingPoint
			for _, l := range lows {
				if l.Index > left.Index && l.Index < right.Index {
					neckLows = append(neckLows, l)
				}
			}
			if len(neckLows) < 2 {
				continue
			}

			neckline := (neckLows[0].Price + neckLows[len(neckLows)-1].Price) / 2
			lastClose := candles[len(candles)-1].Close

			if lastClose > neckline*1.01 || lastClose < neckline*0.95 {
				continue
			}

			if adx != nil && adx.IsValid() && !adx.IsTrending() {
				continue
			}

			entryPrice := neckline * 0.999
			stopLoss := head.Price * 1.01
			risk := stopLoss - entryPrice
			takeProfit := entryPrice - risk*2

			return &Signal{
				PatternType: "H&S",
				Direction:   -1,
				EntryPrice:  entryPrice,
				StopLoss:    stopLoss,
				TakeProfit:  takeProfit,
				Confidence:  symmetry,
				RiskReward:  2.0,
				BarIndex:    len(candles) - 1,
			}
		}
	}

	// Check IHS (bullish)
	if len(lows) >= 3 {
		for i := len(lows) - 3; i >= 0; i-- {
			left := lows[i]
			head := lows[i+1]
			right := lows[i+2]

			if head.Price >= left.Price || head.Price >= right.Price {
				continue
			}

			leftD := left.Price - head.Price
			rightD := right.Price - head.Price
			symmetry := 1 - math.Abs(leftD-rightD)/((leftD+rightD)/2)
			if symmetry < 0.6 {
				continue
			}

			var neckHighs []SwingPoint
			for _, h := range highs {
				if h.Index > left.Index && h.Index < right.Index {
					neckHighs = append(neckHighs, h)
				}
			}
			if len(neckHighs) < 2 {
				continue
			}

			neckline := (neckHighs[0].Price + neckHighs[len(neckHighs)-1].Price) / 2
			lastClose := candles[len(candles)-1].Close

			if lastClose < neckline*0.99 || lastClose > neckline*1.05 {
				continue
			}

			if adx != nil && adx.IsValid() && !adx.IsTrending() {
				continue
			}

			entryPrice := neckline * 1.001
			stopLoss := head.Price * 0.99
			risk := entryPrice - stopLoss
			takeProfit := entryPrice + risk*2

			return &Signal{
				PatternType: "IHS",
				Direction:   1,
				EntryPrice:  entryPrice,
				StopLoss:    stopLoss,
				TakeProfit:  takeProfit,
				Confidence:  symmetry,
				RiskReward:  2.0,
				BarIndex:    len(candles) - 1,
			}
		}
	}

	return nil
}

// convertSignalForScoring converts our signal to scoring.Signal
func convertSignalForScoring(s *Signal) *scoring.Signal {
	return &scoring.Signal{
		PatternType: s.PatternType,
		Direction:   s.Direction,
		EntryPrice:  s.EntryPrice,
		StopLoss:    s.StopLoss,
		TakeProfit:  s.TakeProfit,
		BarIndex:    s.BarIndex,
	}
}

// shouldTakeGrade checks if the grade meets minimum requirements
func shouldTakeGrade(grade, minGrade string) bool {
	grades := map[string]int{"A": 4, "B": 3, "C": 2, "F": 1}
	return grades[grade] >= grades[minGrade]
}

func runSimulationForSymbol(loader *database.SQLiteLoader, symbol string, timeframe database.Timeframe, dateRange *database.DateRange, config SimulationConfig, verbose bool) []TradeResult {
	h4Candles, err := loader.GetCandles(symbol, database.TimeframeH4, dateRange)
	if err != nil || len(h4Candles) == 0 {
		return nil
	}

	// Load daily candles for MTF context
	var dailyCandles []database.Candle
	if config.UseSMCFilters {
		dailyCandles, _ = loader.GetCandles(symbol, database.TimeframeDaily, dateRange)
	}

	if verbose {
		fmt.Printf("\nProcessing %s: %d candles...", symbol, len(h4Candles))
		if config.UseSMCFilters {
			fmt.Printf(" (SMC filters enabled, min grade: %s)\n", config.MinGrade)
		} else {
			fmt.Println()
		}
	}

	config.Symbol = symbol
	feeder := NewMarketFeeder(h4Candles)
	adx := indicators.NewADX(14)

	// MTF context builder
	var mtfBuilder *mtf.MTFContextBuilder
	if config.UseSMCFilters {
		mtfBuilder = mtf.NewMTFContextBuilder(loader)
	}

	var trades []TradeResult
	var position *Position
	var pendingSignal *Signal
	var pendingGrade string
	var pendingScore int // Confluence score for display
	lastTradeBar := make(map[string]int)

	// Grade statistics
	gradeStats := map[string]struct{ total, wins int }{
		"A": {}, "B": {}, "C": {}, "F": {},
	}
	filteredByGrade := 0

	for feeder.HasMoreBars() {
		currentBar := feeder.NextBar()
		barIdx := feeder.CurrentIndex()

		// 1. Check exits first
		if position != nil {
			var exitPrice float64
			var outcome string
			barsHeld := barIdx - position.EntryBar

			if position.Direction == -1 {
				slHit := currentBar.High >= position.StopLoss
				tpHit := currentBar.Low <= position.TakeProfit

				if slHit && tpHit {
					exitPrice = position.StopLoss
					outcome = "SL_HIT"
				} else if slHit {
					exitPrice = position.StopLoss
					outcome = "SL_HIT"
				} else if tpHit {
					exitPrice = position.TakeProfit
					outcome = "TP_HIT"
				} else if barsHeld >= config.MaxBarsToHold {
					exitPrice = currentBar.Close
					outcome = "TIMEOUT"
				}
			} else {
				slHit := currentBar.Low <= position.StopLoss
				tpHit := currentBar.High >= position.TakeProfit

				if slHit && tpHit {
					exitPrice = position.StopLoss
					outcome = "SL_HIT"
				} else if slHit {
					exitPrice = position.StopLoss
					outcome = "SL_HIT"
				} else if tpHit {
					exitPrice = position.TakeProfit
					outcome = "TP_HIT"
				} else if barsHeld >= config.MaxBarsToHold {
					exitPrice = currentBar.Close
					outcome = "TIMEOUT"
				}
			}

			if outcome != "" {
				var pnl float64
				if position.Direction == -1 {
					pnl = position.EntryPrice - exitPrice
				} else {
					pnl = exitPrice - position.EntryPrice
				}
				risk := math.Abs(position.StopLoss - position.EntryPrice)
				pnlR := pnl / risk

				resultOutcome := "LOSS"
				if outcome == "TP_HIT" {
					resultOutcome = "WIN"
				} else if outcome == "TIMEOUT" && pnlR > 0 {
					resultOutcome = "WIN"
				}

				trade := TradeResult{
					Symbol:      symbol,
					PatternType: position.PatternType,
					Direction:   position.Direction,
					EntryPrice:  position.EntryPrice,
					ExitPrice:   exitPrice,
					StopLoss:    position.StopLoss,
					TakeProfit:  position.TakeProfit,
					EntryBar:    position.EntryBar,
					ExitBar:     barIdx,
					PnLR:        pnlR,
					Outcome:     resultOutcome,
					BarsHeld:    barsHeld,
					Grade:       position.Grade,
				}
				trades = append(trades, trade)

				// Update grade stats
				if position.Grade != "" {
					stat := gradeStats[position.Grade]
					stat.total++
					if resultOutcome == "WIN" {
						stat.wins++
					}
					gradeStats[position.Grade] = stat
				}

				position = nil
			}
		}

		// 2. Check pending order fill
		if position == nil && pendingSignal != nil {
			filled := false
			fillPrice := pendingSignal.EntryPrice

			if pendingSignal.Direction == -1 {
				if currentBar.Open <= pendingSignal.EntryPrice {
					filled = true
					fillPrice = currentBar.Open
				} else if currentBar.Low <= pendingSignal.EntryPrice {
					filled = true
				}
			} else {
				if currentBar.Open >= pendingSignal.EntryPrice {
					filled = true
					fillPrice = currentBar.Open
				} else if currentBar.High >= pendingSignal.EntryPrice {
					filled = true
				}
			}

			if filled {
				position = &Position{
					Direction:   pendingSignal.Direction,
					EntryPrice:  fillPrice,
					StopLoss:    pendingSignal.StopLoss,
					TakeProfit:  pendingSignal.TakeProfit,
					EntryBar:    barIdx,
					PatternType: pendingSignal.PatternType,
					Grade:       pendingGrade,
				}
			}
			pendingSignal = nil
			pendingGrade = ""
			_ = pendingScore // Reset
		}

		// 3. Look for new signals
		if position == nil && pendingSignal == nil {
			visible := feeder.GetVisibleWindow(100)
			if len(visible) >= 50 {
				adx.Calculate(visible)

				lastHS := lastTradeBar["H&S"]
				lastIHS := lastTradeBar["IHS"]
				hsOK := barIdx-lastHS > config.CooldownBars
				ihsOK := barIdx-lastIHS > config.CooldownBars

				if hsOK || ihsOK {
					signal := detectPatterns(visible, config, adx)
					if signal != nil {
						if (signal.PatternType == "H&S" && hsOK) || (signal.PatternType == "IHS" && ihsOK) {
							
							// Apply SMC filters if enabled
							grade := "B" // Default grade if filters disabled
							score := 0
							
							if config.UseSMCFilters {
								// Build MTF context from visible data
								visibleDaily := make([]database.Candle, 0)
								if len(dailyCandles) > 0 {
									// Find daily candles up to current time
									for _, dc := range dailyCandles {
										if dc.Timestamp.Before(currentBar.Timestamp) || dc.Timestamp.Equal(currentBar.Timestamp) {
											visibleDaily = append(visibleDaily, dc)
										}
									}
								}

								// Build context from visible candles only (no lookahead)
								mtfCtx := mtfBuilder.BuildContextFromCandles(symbol, visible, visibleDaily)
								
								// Calculate confluence score
								scoringSignal := convertSignalForScoring(signal)
								confluenceScore := scoring.CalculateConfluence(mtfCtx, scoringSignal)
								
								grade = confluenceScore.Grade
								score = confluenceScore.Total
								
								// Filter by grade
								if !shouldTakeGrade(grade, config.MinGrade) {
									filteredByGrade++
									lastTradeBar[signal.PatternType] = barIdx
									continue
								}
							}
							
							pendingSignal = signal
							pendingGrade = grade
							pendingScore = score
							lastTradeBar[signal.PatternType] = barIdx
						}
					}
				}
			}
		}
	}

	if verbose && len(trades) > 0 {
		wins := 0
		totalR := 0.0
		for _, t := range trades {
			totalR += t.PnLR
			if t.Outcome == "WIN" {
				wins++
			}
		}
		winRate := float64(wins) / float64(len(trades)) * 100
		fmt.Printf("  %s: %d trades, %d wins (%.1f%%), %.2fR", symbol, len(trades), wins, winRate, totalR)
		
		if config.UseSMCFilters && filteredByGrade > 0 {
			fmt.Printf(" [filtered: %d]", filteredByGrade)
		}
		fmt.Println()
		
		// Print grade breakdown
		if config.UseSMCFilters {
			for g, stat := range gradeStats {
				if stat.total > 0 {
					gWinRate := float64(stat.wins) / float64(stat.total) * 100
					fmt.Printf("    Grade %s: %d trades, %.1f%% win\n", g, stat.total, gWinRate)
				}
			}
		}
	}

	return trades
}

func main() {
	dbPath := flag.String("db", "forex_prices.db", "Path to database")
	symbol := flag.String("symbol", "EURUSD", "Symbol to trade (or 'all' for all symbols)")
	tf := flag.String("tf", "h4", "Timeframe")
	mode := flag.String("mode", "fast", "Mode: step, auto, fast")
	startDate := flag.String("start", "2020-01-01", "Start date")
	endDate := flag.String("end", "2025-12-31", "End date")
	useSMC := flag.Bool("smc", false, "Enable SMC confluence filters")
	minGrade := flag.String("grade", "B", "Minimum grade to trade (A, B, C)")
	flag.Parse()

	fmt.Println("\n===============================================================================")
	fmt.Println("         H&S PATTERN TRADING SIMULATION (NO LOOKAHEAD BIAS)")
	if *useSMC {
		fmt.Println("                  [SMC CONFLUENCE FILTERS ENABLED]")
	}
	fmt.Println("===============================================================================")

	loader, err := database.NewSQLiteLoader(*dbPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer loader.Close()

	var timeframe database.Timeframe
	switch strings.ToLower(*tf) {
	case "h4":
		timeframe = database.TimeframeH4
	case "h1":
		timeframe = database.TimeframeH1
	case "daily":
		timeframe = database.TimeframeDaily
	default:
		timeframe = database.TimeframeH4
	}

	start, _ := time.Parse("2006-01-02", *startDate)
	end, _ := time.Parse("2006-01-02", *endDate)
	dateRange := &database.DateRange{Start: start, End: end}

	config := SimulationConfig{
		Timeframe:           timeframe,
		ConfidenceThreshold: 0.6,
		MinRR:               1.5,
		MaxBarsToHold:       50,
		CooldownBars:        10,
		SwingStrength:       5,
		UseSMCFilters:       *useSMC,
		MinGrade:            *minGrade,
	}

	var symbols []string
	if strings.ToLower(*symbol) == "all" {
		symbols = database.GetAllSymbols()
	} else {
		symbols = []string{*symbol}
	}

	fmt.Printf("\nSymbols: %d\n", len(symbols))
	fmt.Printf("Timeframe: %s\n", *tf)
	fmt.Printf("Period: %s to %s\n", *startDate, *endDate)
	fmt.Printf("Mode: %s\n", *mode)
	if *useSMC {
		fmt.Printf("SMC Filters: ENABLED (min grade: %s)\n", *minGrade)
	}

	fmt.Println("\n===============================================================================")
	fmt.Println("                         SIMULATION RUNNING")
	fmt.Println("===============================================================================")
	fmt.Println("\nProcessing (bot can only see past data)...")

	startTime := time.Now()

	var allTrades []TradeResult
	for _, sym := range symbols {
		trades := runSimulationForSymbol(loader, sym, timeframe, dateRange, config, len(symbols) > 1)
		allTrades = append(allTrades, trades...)
	}

	elapsed := time.Since(startTime)

	fmt.Println("\n===============================================================================")
	fmt.Println("                         SIMULATION COMPLETE")
	fmt.Println("===============================================================================")
	fmt.Printf("\nCompleted in %v\n", elapsed)
	fmt.Printf("Symbols processed: %d\n", len(symbols))

	if len(allTrades) == 0 {
		fmt.Println("\nNo trades executed.")
		return
	}

	wins := 0
	losses := 0
	totalR := 0.0
	hsWins, hsLosses := 0, 0
	ihsWins, ihsLosses := 0, 0
	gradeStats := map[string]struct{ total, wins int; pnl float64 }{
		"A": {}, "B": {}, "C": {}, "F": {},
	}
	symbolStats := make(map[string]struct{ wins, losses int; pnl float64 })

	for _, t := range allTrades {
		totalR += t.PnLR
		stat := symbolStats[t.Symbol]
		stat.pnl += t.PnLR
		
		// Grade stats
		gStat := gradeStats[t.Grade]
		gStat.total++
		gStat.pnl += t.PnLR
		
		if t.Outcome == "WIN" {
			wins++
			stat.wins++
			gStat.wins++
			if t.PatternType == "H&S" {
				hsWins++
			} else {
				ihsWins++
			}
		} else {
			losses++
			stat.losses++
			if t.PatternType == "H&S" {
				hsLosses++
			} else {
				ihsLosses++
			}
		}
		symbolStats[t.Symbol] = stat
		gradeStats[t.Grade] = gStat
	}

	fmt.Printf("\nTRADING RESULTS:\n")
	fmt.Printf("  Total Trades: %d\n", len(allTrades))
	fmt.Printf("  Winners: %d (%.1f%%)\n", wins, float64(wins)/float64(len(allTrades))*100)
	fmt.Printf("  Losers: %d (%.1f%%)\n", losses, float64(losses)/float64(len(allTrades))*100)
	fmt.Printf("\n  Total P&L: %.2fR\n", totalR)
	fmt.Printf("  Expectancy: %.2fR per trade\n", totalR/float64(len(allTrades)))

	fmt.Printf("\nPATTERN BREAKDOWN:\n")
	if hsWins+hsLosses > 0 {
		fmt.Printf("  H&S (Bearish): %d trades, %d wins (%.1f%%)\n", hsWins+hsLosses, hsWins, float64(hsWins)/float64(hsWins+hsLosses)*100)
	}
	if ihsWins+ihsLosses > 0 {
		fmt.Printf("  IHS (Bullish): %d trades, %d wins (%.1f%%)\n", ihsWins+ihsLosses, ihsWins, float64(ihsWins)/float64(ihsWins+ihsLosses)*100)
	}

	// Grade breakdown
	if *useSMC {
		fmt.Printf("\nGRADE BREAKDOWN:\n")
		for _, g := range []string{"A", "B", "C", "F"} {
			stat := gradeStats[g]
			if stat.total > 0 {
				winRate := float64(stat.wins) / float64(stat.total) * 100
				avgR := stat.pnl / float64(stat.total)
				fmt.Printf("  Grade %s: %d trades, %.1f%% win, %.2fR avg, %.2fR total\n", g, stat.total, winRate, avgR, stat.pnl)
			}
		}
	}

	if len(symbols) > 1 {
		fmt.Printf("\nTOP PERFORMING SYMBOLS:\n")
		type symbolResult struct {
			symbol string
			wins   int
			losses int
			pnl    float64
		}
		var results []symbolResult
		for sym, stat := range symbolStats {
			results = append(results, symbolResult{sym, stat.wins, stat.losses, stat.pnl})
		}
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].pnl > results[i].pnl {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
		shown := 5
		if len(results) < 5 {
			shown = len(results)
		}
		for i := 0; i < shown; i++ {
			r := results[i]
			total := r.wins + r.losses
			fmt.Printf("  %s: %d trades, %.1f%% win, %.2fR\n", r.symbol, total, float64(r.wins)/float64(total)*100, r.pnl)
		}
		fmt.Printf("\nWORST PERFORMING SYMBOLS:\n")
		for i := len(results) - 1; i >= len(results)-shown && i >= 0; i-- {
			r := results[i]
			total := r.wins + r.losses
			if total > 0 {
				fmt.Printf("  %s: %d trades, %.1f%% win, %.2fR\n", r.symbol, total, float64(r.wins)/float64(total)*100, r.pnl)
			}
		}
	}

	fmt.Println("\nNO-LOOKAHEAD VERIFICATION:")
	fmt.Println("  - Bot only sees GetVisibleWindow() data (past bars)")
	fmt.Println("  - Orders are pending until next bar fills them")
	fmt.Println("  - If SL and TP both hit same bar, SL assumed first (conservative)")
	fmt.Println("  - Entry slippage: if bar opens past entry, fill at open")
	if *useSMC {
		fmt.Println("  - SMC context built from visible data only (no future data)")
	}

	fmt.Println("\n===============================================================================")
}
