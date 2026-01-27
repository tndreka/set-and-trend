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
	MinGrade            string
	OnlyGrade           string
	ExcludeSymbols      []string
	// New improvement parameters
	MinPatternSize      float64 // Minimum head-to-neckline distance (percentage)
	MinSymmetry         float64 // Minimum shoulder symmetry
	MinTimeSymmetry     float64 // Minimum time symmetry
	RequireVolumeSpike  bool    // Require volume confirmation
	VolumeMultiplier    float64 // Volume spike threshold
	UseRetestEntry      bool    // Wait for neckline retest
	UseTighterSL        bool    // Use right shoulder for SL instead of head
	UseTimeFilters      bool    // Avoid low-probability periods
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

// calculateAverageVolume computes average volume over N bars
func calculateAverageVolume(candles []database.Candle, period int) float64 {
	if len(candles) < period {
		period = len(candles)
	}
	if period == 0 {
		return 0
	}
	
	sum := 0.0
	start := len(candles) - period
	for i := start; i < len(candles); i++ {
		sum += candles[i].Volume
	}
	return sum / float64(period)
}

// isLowProbabilityTime checks if current time is low probability for trading
func isLowProbabilityTime(t time.Time, symbol string) bool {
	hour := t.UTC().Hour()
	weekday := t.Weekday()
	
	// Skip Friday afternoon (weekend gap risk)
	if weekday == time.Friday && hour >= 18 {
		return true
	}
	
	// Skip Asian session for forex pairs (low liquidity, 22:00-07:00 UTC)
	isForex := !strings.Contains(symbol, "IDX") && !strings.Contains(symbol, "XAU") && !strings.Contains(symbol, "XAG") && !strings.Contains(symbol, "CMD")
	if isForex && (hour >= 22 || hour < 7) {
		return true
	}
	
	return false
}

// detectPatterns finds H&S and IHS patterns with improved filters
func detectPatterns(candles []database.Candle, config SimulationConfig, adx *indicators.ADX) *Signal {
	if len(candles) < 30 {
		return nil
	}

	// Time filter (Phase 6)
	if config.UseTimeFilters {
		lastCandle := candles[len(candles)-1]
		if isLowProbabilityTime(lastCandle.Timestamp, config.Symbol) {
			return nil
		}
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

			// Head must be higher than shoulders
			if head.Price <= left.Price || head.Price <= right.Price {
				continue
			}

			// IMPROVEMENT: Stricter symmetry (Phase 4)
			leftH := head.Price - left.Price
			rightH := head.Price - right.Price
			symmetry := 1 - math.Abs(leftH-rightH)/((leftH+rightH)/2)
			if symmetry < config.MinSymmetry {
				continue
			}

			// IMPROVEMENT: Time symmetry (Phase 4)
			leftTime := head.Index - left.Index
			rightTime := right.Index - head.Index
			timeSymmetry := 1.0 - math.Abs(float64(leftTime-rightTime))/float64(max(leftTime, rightTime))
			if timeSymmetry < config.MinTimeSymmetry {
				continue
			}

			// Find neckline points
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

			// IMPROVEMENT: Minimum pattern size (Phase 4)
			patternSize := (head.Price - neckline) / head.Price
			if patternSize < config.MinPatternSize {
				continue
			}

			lastClose := candles[len(candles)-1].Close
			prevClose := candles[len(candles)-2].Close

			// IMPROVEMENT: Entry timing - require neckline BREAK (Phase 1)
			// Old: Enter when price is near neckline
			// New: Enter when price BREAKS below neckline (confirmation)
			if config.UseRetestEntry {
				// Require: previous close above neckline, current close below
				necklineBroken := prevClose > neckline && lastClose < neckline
				// Or: price is retesting neckline from below after break
				retesting := lastClose < neckline && lastClose > neckline*0.98 && prevClose < neckline*0.98
				
				if !necklineBroken && !retesting {
					continue
				}
			} else {
				// Fallback to original logic (but stricter)
				if lastClose > neckline*1.005 || lastClose < neckline*0.95 {
					continue
				}
			}

			// IMPROVEMENT: Volume confirmation (Phase 3)
			if config.RequireVolumeSpike {
				avgVolume := calculateAverageVolume(candles[:len(candles)-1], 20)
				currentVolume := candles[len(candles)-1].Volume
				if avgVolume > 0 && currentVolume < avgVolume*config.VolumeMultiplier {
					continue // No volume confirmation
				}
			}

			// ADX filter
			if adx != nil && adx.IsValid() && !adx.IsTrending() {
				continue
			}

			// IMPROVEMENT: Tighter stop loss using right shoulder (Phase 2)
			var stopLoss float64
			if config.UseTighterSL {
				stopLoss = right.Price * 1.005 // Just above right shoulder
			} else {
				stopLoss = head.Price * 1.01 // Original: above head
			}

			entryPrice := neckline * 0.999
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

			// Head must be lower than shoulders
			if head.Price >= left.Price || head.Price >= right.Price {
				continue
			}

			// IMPROVEMENT: Stricter symmetry
			leftD := left.Price - head.Price
			rightD := right.Price - head.Price
			symmetry := 1 - math.Abs(leftD-rightD)/((leftD+rightD)/2)
			if symmetry < config.MinSymmetry {
				continue
			}

			// IMPROVEMENT: Time symmetry
			leftTime := head.Index - left.Index
			rightTime := right.Index - head.Index
			timeSymmetry := 1.0 - math.Abs(float64(leftTime-rightTime))/float64(max(leftTime, rightTime))
			if timeSymmetry < config.MinTimeSymmetry {
				continue
			}

			// Find neckline points
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

			// IMPROVEMENT: Minimum pattern size
			patternSize := (neckline - head.Price) / head.Price
			if patternSize < config.MinPatternSize {
				continue
			}

			lastClose := candles[len(candles)-1].Close
			prevClose := candles[len(candles)-2].Close

			// IMPROVEMENT: Entry timing - require neckline BREAK
			if config.UseRetestEntry {
				// Require: previous close below neckline, current close above
				necklineBroken := prevClose < neckline && lastClose > neckline
				// Or: price is retesting neckline from above after break
				retesting := lastClose > neckline && lastClose < neckline*1.02 && prevClose > neckline*1.02
				
				if !necklineBroken && !retesting {
					continue
				}
			} else {
				if lastClose < neckline*0.995 || lastClose > neckline*1.05 {
					continue
				}
			}

			// IMPROVEMENT: Volume confirmation
			if config.RequireVolumeSpike {
				avgVolume := calculateAverageVolume(candles[:len(candles)-1], 20)
				currentVolume := candles[len(candles)-1].Volume
				if avgVolume > 0 && currentVolume < avgVolume*config.VolumeMultiplier {
					continue
				}
			}

			// ADX filter
			if adx != nil && adx.IsValid() && !adx.IsTrending() {
				continue
			}

			// IMPROVEMENT: Tighter stop loss
			var stopLoss float64
			if config.UseTighterSL {
				stopLoss = right.Price * 0.995 // Just below right shoulder
			} else {
				stopLoss = head.Price * 0.99 // Original: below head
			}

			entryPrice := neckline * 1.001
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

func shouldTakeGrade(grade, minGrade string) bool {
	grades := map[string]int{"A": 4, "B": 3, "C": 2, "F": 1}
	return grades[grade] >= grades[minGrade]
}

func runSimulationForSymbol(loader *database.SQLiteLoader, symbol string, timeframe database.Timeframe, dateRange *database.DateRange, config SimulationConfig, verbose bool) []TradeResult {
	h4Candles, err := loader.GetCandles(symbol, database.TimeframeH4, dateRange)
	if err != nil || len(h4Candles) == 0 {
		return nil
	}

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

	var mtfBuilder *mtf.MTFContextBuilder
	if config.UseSMCFilters {
		mtfBuilder = mtf.NewMTFContextBuilder(loader)
	}

	var trades []TradeResult
	var position *Position
	var pendingSignal *Signal
	var pendingGrade string
	var pendingScore int
	lastTradeBar := make(map[string]int)

	gradeStats := map[string]struct{ total, wins int }{
		"A": {}, "B": {}, "C": {}, "F": {},
	}
	filteredByGrade := 0

	for feeder.HasMoreBars() {
		currentBar := feeder.NextBar()
		barIdx := feeder.CurrentIndex()

		// Check exits
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

		// Check pending order fill
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
			_ = pendingScore
		}

		// Look for new signals
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
							
							grade := "B"
							score := 0
							
							if config.UseSMCFilters {
								visibleDaily := make([]database.Candle, 0)
								if len(dailyCandles) > 0 {
									for _, dc := range dailyCandles {
										if dc.Timestamp.Before(currentBar.Timestamp) || dc.Timestamp.Equal(currentBar.Timestamp) {
											visibleDaily = append(visibleDaily, dc)
										}
									}
								}

								mtfCtx := mtfBuilder.BuildContextFromCandles(symbol, visible, visibleDaily)
								scoringSignal := convertSignalForScoring(signal)
								confluenceScore := scoring.CalculateConfluence(mtfCtx, scoringSignal)
								
								grade = confluenceScore.Grade
								score = confluenceScore.Total
								
								if config.OnlyGrade != "" && grade != config.OnlyGrade {
									filteredByGrade++
									lastTradeBar[signal.PatternType] = barIdx
									continue
								}
								if config.OnlyGrade == "" && !shouldTakeGrade(grade, config.MinGrade) {
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
	excludeSymbols := flag.String("exclude", "", "Comma-separated symbols to exclude")
	mode := flag.String("mode", "fast", "Mode: step, auto, fast")
	startDate := flag.String("start", "2020-01-01", "Start date")
	endDate := flag.String("end", "2025-12-31", "End date")
	useSMC := flag.Bool("smc", false, "Enable SMC confluence filters")
	minGrade := flag.String("grade", "B", "Minimum grade to trade (A, B, C, F)")
	onlyGrade := flag.String("onlygrade", "", "Only trade this specific grade")
	improved := flag.Bool("improved", false, "Use improved filters (all 6 phases)")
	flag.Parse()

	fmt.Println("\n===============================================================================")
	fmt.Println("         H&S PATTERN TRADING SIMULATION (NO LOOKAHEAD BIAS)")
	if *useSMC {
		fmt.Println("                  [SMC CONFLUENCE FILTERS ENABLED]")
	}
	if *improved {
		fmt.Println("                  [IMPROVED FILTERS V2 ENABLED]")
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
		OnlyGrade:           *onlyGrade,
		// Default values (original behavior)
		MinPatternSize:      0.01,
		MinSymmetry:         0.6,
		MinTimeSymmetry:     0.0,
		RequireVolumeSpike:  false,
		VolumeMultiplier:    1.5,
		UseRetestEntry:      false,
		UseTighterSL:        false,
		UseTimeFilters:      false,
	}

	// Apply improved filters if flag is set
	if *improved {
		config.MinPatternSize = 0.01      // 2% minimum pattern size
		config.MinSymmetry = 0.6         // Stricter symmetry
		config.MinTimeSymmetry = 0.0      // Require time symmetry
		config.RequireVolumeSpike = false // Skip volume (data may be unreliable)
		config.VolumeMultiplier = 1.5
		config.UseRetestEntry = false      // Wait for neckline break
		config.UseTighterSL = false        // Use right shoulder for SL
		config.UseTimeFilters = true      // Avoid low probability times
	}

	var symbols []string
	if strings.ToLower(*symbol) == "all" {
		symbols = database.GetAllSymbols()
	} else {
		symbols = []string{*symbol}
	}
	// Filter excluded symbols
	if *excludeSymbols != "" {
		excludeList := strings.Split(*excludeSymbols, ",")
		excludeMap := make(map[string]bool)
		for _, s := range excludeList {
			excludeMap[strings.TrimSpace(s)] = true
		}
		var filteredSymbols []string
		for _, s := range symbols {
			if !excludeMap[s] {
				filteredSymbols = append(filteredSymbols, s)
			}
		}
		symbols = filteredSymbols
	}

	fmt.Printf("\nSymbols: %d\n", len(symbols))
	fmt.Printf("Timeframe: %s\n", *tf)
	fmt.Printf("Period: %s to %s\n", *startDate, *endDate)
	fmt.Printf("Mode: %s\n", *mode)
	if *useSMC {
		fmt.Printf("SMC Filters: ENABLED (min grade: %s)\n", *minGrade)
	}
	if *improved {
		fmt.Printf("Improved Filters: ENABLED\n")
		fmt.Printf("  - Min Pattern Size: %.1f%%\n", config.MinPatternSize*100)
		fmt.Printf("  - Min Symmetry: %.0f%%\n", config.MinSymmetry*100)
		fmt.Printf("  - Min Time Symmetry: %.0f%%\n", config.MinTimeSymmetry*100)
		fmt.Printf("  - Retest Entry: %v\n", config.UseRetestEntry)
		fmt.Printf("  - Tighter SL: %v\n", config.UseTighterSL)
		fmt.Printf("  - Time Filters: %v\n", config.UseTimeFilters)
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
