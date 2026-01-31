package patterns

import (
	"fmt"
	"math"
	"time"
	
	"set-and-trend/backend/internal/constants"
)

// BacktestConfig holds backtesting parameters
type BacktestConfig struct {
	WindowSize          int
	ConfidenceThreshold float64
	MinRR               float64
	MaxBarsToExit       int
	Lookback            int
	CooldownBars        int // Minimum bars between trades to avoid duplicates
	// Execution costs - CRITICAL for realistic backtesting
	SpreadPips     float64 // e.g., 0.2 for EURUSD, 0.8 for GBPJPY
	SlippagePips   float64 // Slippage on SL hits (conservative: 0.5)
	TPSlippagePips float64 // Slippage on TP hits (lower: 0.2)
	CommissionPips float64 // If applicable (e.g., 0.0 for spread-only, 0.3 for ECN)
}

// DefaultBacktestConfig returns sensible defaults with realistic costs
func DefaultBacktestConfig() BacktestConfig {
	return BacktestConfig{
		WindowSize:          30,
		ConfidenceThreshold: 0.60,
		MinRR:               1.5,
		MaxBarsToExit:       20,
		Lookback:            3,
		CooldownBars:        10,
		// Realistic EURUSD costs - adjust for your symbol
		SpreadPips:     0.2,  // Typical EURUSD spread
		SlippagePips:   0.3,  // Conservative SL slippage in volatile H4
		TPSlippagePips: 0.2,  // TP fills better than SL but still pay spread
		CommissionPips: 0.0,  // Add if using raw spread + commission broker
	}
}

// BacktestTrade represents a single backtest trade
type BacktestTrade struct {
	EntryTime   time.Time
	ExitTime    time.Time
	EntryPrice  float64
	ExitPrice   float64
	StopLoss    float64
	TakeProfit  float64
	Direction   string
	PatternType string
	Confidence  float64
	RiskReward  float64
	PnL         float64
	PnLPips     float64
	PnLR        float64
	Result      string
	Reason      string
	BarIndex    int
	ExitBar     int
}

// BacktestMetrics holds aggregate backtest results
type BacktestMetrics struct {
	TotalPatterns     int
	TradedPatterns    int
	FilteredPatterns  int
	TotalTrades       int
	WinningTrades     int
	LosingTrades      int
	BreakevenTrades   int
	TimeoutTrades     int
	WinRate           float64
	AvgWin            float64
	AvgLoss           float64
	AvgRR             float64
	Expectancy        float64
	ExpectancyR       float64 // Expectancy in R multiples
	ProfitFactor      float64
	MaxDrawdown       float64
	MaxDrawdownR      float64
	TotalPnL          float64
	TotalPnLPips      float64
	TotalPnLR         float64
	LargestWin        float64
	LargestLoss       float64
	AvgBarsInTrade    float64
	ConsecutiveWins   int
	ConsecutiveLosses int
}

// ExitResult holds the outcome of a simulated exit
type ExitResult struct {
	ExitPrice float64
	ExitTime  time.Time
	ExitBar   int
	Reason    string
	PnL       float64
	PnLPips   float64
}

// SimulateExit simulates trade exit on future candles with realistic costs
func SimulateExit(signal *TradeSignal, entryPrice float64, futureCandles []Candle, config BacktestConfig) *ExitResult {
	if signal == nil || len(futureCandles) == 0 {
		return nil
	}

	barsToCheck := len(futureCandles)
	if config.MaxBarsToExit > 0 && config.MaxBarsToExit < barsToCheck {
		barsToCheck = config.MaxBarsToExit
	}

	pipSize := constants.GetPipSizeForSymbol(signal.Symbol)
	spreadCost := config.SpreadPips * pipSize
	slSlippage := config.SlippagePips * pipSize
	tpSlippage := config.TPSlippagePips * pipSize

	for i := 0; i < barsToCheck; i++ {
		candle := futureCandles[i]

		if signal.Direction == DirectionShort {
			// SHORT: Entry at Bid, Exit at Ask (worse by spread)
			
			// SL HIT: High >= StopLoss (Bid price hit SL)
			// Actual fill: Buy at Ask = StopLoss + spread + slippage
			if candle.High >= signal.StopLoss {
				exitPrice := signal.StopLoss + spreadCost + slSlippage
				// Cap at candle high (can't fill worse than the bar's extreme + slippage)
				if exitPrice > candle.High+slSlippage {
					exitPrice = candle.High + slSlippage
				}
				
				pnl := entryPrice - exitPrice
				return &ExitResult{
					ExitPrice: exitPrice,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "SL_HIT",
					PnL:       pnl,
					PnLPips:   pnl / pipSize,
				}
			}
			
			// TP HIT: Low <= TakeProfit (Bid price hit TP)
			// Actual fill: Buy at Ask = TakeProfit + spread + small slippage
			if candle.Low <= signal.TakeProfit {
				exitPrice := signal.TakeProfit + spreadCost + tpSlippage
				if exitPrice < candle.Low { // Sanity check
					exitPrice = candle.Low + spreadCost
				}
				
				pnl := entryPrice - exitPrice
				return &ExitResult{
					ExitPrice: exitPrice,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "TP_HIT",
					PnL:       pnl,
					PnLPips:   pnl / pipSize,
				}
			}
			
		} else if signal.Direction == DirectionLong {
			// LONG: Entry at Ask, Exit at Bid (worse by spread)
			
			// SL HIT: Low <= StopLoss (Bid price hit SL)
			// Actual fill: Sell at Bid = StopLoss - spread - slippage
			if candle.Low <= signal.StopLoss {
				exitPrice := signal.StopLoss - spreadCost - slSlippage
				if exitPrice < candle.Low-slSlippage {
					exitPrice = candle.Low - slSlippage
				}
				
				pnl := exitPrice - entryPrice
				return &ExitResult{
					ExitPrice: exitPrice,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "SL_HIT",
					PnL:       pnl,
					PnLPips:   pnl / pipSize,
				}
			}
			
			// TP HIT: High >= TakeProfit (Bid price hit TP)
			// Actual fill: Sell at Bid = TakeProfit - spread - small slippage  
			if candle.High >= signal.TakeProfit {
				exitPrice := signal.TakeProfit - spreadCost - tpSlippage
				if exitPrice > candle.High {
					exitPrice = candle.High - spreadCost
				}
				
				pnl := exitPrice - entryPrice
				return &ExitResult{
					ExitPrice: exitPrice,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "TP_HIT",
					PnL:       pnl,
					PnLPips:   pnl / pipSize,
				}
			}
		}
	}

	// TIMEOUT EXIT - Apply spread + slippage (market order at close)
	lastCandle := futureCandles[barsToCheck-1]
	var pnl float64
	var exitPrice float64
	exitSlippage := config.SlippagePips * pipSize * 0.5  // Half slippage on exit
	
	if signal.Direction == DirectionShort {
		// Close short by buying at Ask (Close + spread + slippage)
		exitPrice = lastCandle.Close + spreadCost + exitSlippage
		pnl = entryPrice - exitPrice
	} else {
		// Close long by selling at Bid (Close - spread - slippage)
		exitPrice = lastCandle.Close - spreadCost - exitSlippage
		pnl = exitPrice - entryPrice
	}

	return &ExitResult{
		ExitPrice: exitPrice,
		ExitTime:  lastCandle.Timestamp,
		ExitBar:   barsToCheck - 1,
		Reason:    "TIMEOUT",
		PnL:       pnl,
		PnLPips:   pnl / pipSize,
	}
}

// RunBacktest executes backtest on historical data with cooldown to prevent duplicates
func RunBacktest(candles []Candle, config BacktestConfig) (*BacktestMetrics, []BacktestTrade) {
	if len(candles) < config.WindowSize+config.MaxBarsToExit {
		return nil, nil
	}

	// Set default execution costs if not specified (backward compatibility)
	if config.SpreadPips == 0 {
		config.SpreadPips = 0.2  // Default EURUSD spread
	}
	if config.SlippagePips == 0 {
		config.SlippagePips = 0.5  // Conservative default
	}
	if config.TPSlippagePips == 0 {
		config.TPSlippagePips = 0.2  // TP fills better than SL
	}

	var trades []BacktestTrade
	totalPatterns := 0
	filteredPatterns := 0
	
	// Track last trade bar for each direction to prevent duplicates
	lastHSTradeBar := -config.CooldownBars - 1
	lastIHSTradeBar := -config.CooldownBars - 1

	for i := config.WindowSize; i < len(candles)-config.MaxBarsToExit; i++ {
		window := candles[i-config.WindowSize : i]
		futureData := candles[i : i+config.MaxBarsToExit]

		// Detect H&S (bearish) - only if cooldown passed
		if i-lastHSTradeBar > config.CooldownBars {
			hsStructure := DetectHeadAndShoulders(window, config.Lookback)
			if hsStructure != nil {
				totalPatterns++
				trade := processStructureForBacktest(hsStructure, window, futureData, config, i)
				if trade != nil {
					trades = append(trades, *trade)
					lastHSTradeBar = i
				} else {
					filteredPatterns++
				}
			}
		}

		// Detect IHS (bullish) - only if cooldown passed
		if i-lastIHSTradeBar > config.CooldownBars {
			ihsStructure := DetectInverseHeadAndShoulders(window, config.Lookback)
			if ihsStructure != nil {
				totalPatterns++
				trade := processStructureForBacktest(ihsStructure, window, futureData, config, i)
				if trade != nil {
					trades = append(trades, *trade)
					lastIHSTradeBar = i
				} else {
					filteredPatterns++
				}
			}
		}
	}

	metrics := CalculateBacktestMetrics(trades)
	metrics.TotalPatterns = totalPatterns
	metrics.FilteredPatterns = filteredPatterns
	metrics.TradedPatterns = len(trades)

	return metrics, trades
}

func processStructureForBacktest(structure *DetectedStructure, window []Candle, futureData []Candle, config BacktestConfig, barIndex int) *BacktestTrade {
	ctx := DetermineMarketContext(window)
	finalConfidence := FinalConfidence(structure, ctx)
	structure.FinalConfidence = finalConfidence

	if finalConfidence < config.ConfidenceThreshold {
		return nil
	}

	signal := GenerateTradeSignal(structure, "EURUSD", finalConfidence)
	if signal == nil {
		return nil
	}

		// Calculate realistic entry price with spread and slippage
	pipSize := constants.GetPipSizeForSymbol(signal.Symbol)
	spreadCost := config.SpreadPips * pipSize
	entrySlippage := config.SlippagePips * pipSize * 0.5  // Half slippage on entry

	var actualEntry float64

	if signal.Direction == DirectionShort {
		// SHORT: Sell at Bid, but we get filled slightly worse (slippage)
		// Signal.EntryPrice is theoretical neckline break
		// Actual fill is lower (worse) by slippage
		actualEntry = signal.EntryPrice - entrySlippage
	} else {
		// LONG: Buy at Ask (Bid + spread) + slippage
		// We pay spread AND get slippage against us
		actualEntry = signal.EntryPrice + spreadCost + entrySlippage
	}

	// Recalculate Risk with ACTUAL entry vs original SL
	// The SL level stays the same, but our risk amount changes based on fill
	var risk float64
	if signal.Direction == DirectionShort {
		risk = signal.StopLoss - actualEntry  // SL is higher than entry for short
	} else {
		risk = actualEntry - signal.StopLoss  // SL is lower than entry for long
	}

	if risk <= 0 {
		return nil  // Invalid: SL on wrong side or too tight after slippage
	}

	// Recalculate Reward based on actual entry vs original TP
	var reward float64
	if signal.Direction == DirectionShort {
		reward = actualEntry - signal.TakeProfit
	} else {
		reward = signal.TakeProfit - actualEntry
	}

	actualRR := reward / risk
	
	// CRITICAL: Filter if spread/slippage killed the R:R ratio
	// Example: Signal shows 1.5 RR, but after costs it's 1.2 -> Filter out
	if actualRR < config.MinRR {
		return nil
	}

	// Simulate exit with realistic costs
	exit := SimulateExit(signal, actualEntry, futureData, config)
	if exit == nil {
		return nil
	}

	// Calculate R-multiple based on actual risk
	pnlR := 0.0
	if risk > 0 {
		pnlR = exit.PnL / risk
	}

	result := "BREAKEVEN"
	if exit.Reason == "TP_HIT" {
		result = "WIN"
	} else if exit.Reason == "SL_HIT" {
		result = "LOSS"
	} else if exit.Reason == "TIMEOUT" {
		if pnlR > 0.5 {
			result = "WIN"
		} else if pnlR < -0.5 {
			result = "LOSS"
		}
	}

	return &BacktestTrade{
		EntryTime:   window[len(window)-1].Timestamp,
		ExitTime:    exit.ExitTime,
		EntryPrice:  actualEntry, // Use the realistic fill price
		ExitPrice:   exit.ExitPrice,
		StopLoss:    signal.StopLoss,
		TakeProfit:  signal.TakeProfit,
		Direction:   signal.Direction,
		PatternType: structure.PatternType,
		Confidence:  finalConfidence,
		RiskReward:  actualRR, // Use realistic RR
		PnL:         exit.PnL,
		PnLPips:     exit.PnLPips,
		PnLR:        pnlR,
		Result:      result,
		Reason:      exit.Reason,
		BarIndex:    barIndex,
		ExitBar:     exit.ExitBar,
	}
}

func CalculateBacktestMetrics(trades []BacktestTrade) *BacktestMetrics {
	if len(trades) == 0 {
		return &BacktestMetrics{}
	}

	metrics := &BacktestMetrics{
		TotalTrades: len(trades),
	}

	var totalWinPnL, totalLossPnL float64
	var totalWinR, totalLossR float64
	var totalRR float64
	var totalBars int
	var consecutiveWins, consecutiveLosses, maxConsecWins, maxConsecLosses int

	for _, trade := range trades {
		totalRR += trade.RiskReward
		totalBars += trade.ExitBar + 1

		switch trade.Result {
		case "WIN":
			metrics.WinningTrades++
			totalWinPnL += trade.PnL
			totalWinR += trade.PnLR
			if trade.PnL > metrics.LargestWin {
				metrics.LargestWin = trade.PnL
			}
			consecutiveWins++
			consecutiveLosses = 0
			if consecutiveWins > maxConsecWins {
				maxConsecWins = consecutiveWins
			}
		case "LOSS":
			metrics.LosingTrades++
			totalLossPnL += math.Abs(trade.PnL)
			totalLossR += math.Abs(trade.PnLR)
			if trade.PnL < metrics.LargestLoss {
				metrics.LargestLoss = trade.PnL
			}
			consecutiveLosses++
			consecutiveWins = 0
			if consecutiveLosses > maxConsecLosses {
				maxConsecLosses = consecutiveLosses
			}
		case "BREAKEVEN":
			metrics.BreakevenTrades++
		}

		if trade.Reason == "TIMEOUT" {
			metrics.TimeoutTrades++
		}

		metrics.TotalPnL += trade.PnL
		metrics.TotalPnLPips += trade.PnLPips
		metrics.TotalPnLR += trade.PnLR
	}

	if metrics.WinningTrades > 0 {
		metrics.AvgWin = totalWinPnL / float64(metrics.WinningTrades)
	}
	if metrics.LosingTrades > 0 {
		metrics.AvgLoss = totalLossPnL / float64(metrics.LosingTrades)
	}
	if metrics.TotalTrades > 0 {
		metrics.WinRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades)
		metrics.AvgRR = totalRR / float64(metrics.TotalTrades)
		metrics.AvgBarsInTrade = float64(totalBars) / float64(metrics.TotalTrades)
	}

	lossRate := float64(metrics.LosingTrades) / float64(metrics.TotalTrades)
	metrics.Expectancy = (metrics.WinRate * metrics.AvgWin) - (lossRate * metrics.AvgLoss)
	
	// Expectancy in R
	avgWinR := 0.0
	avgLossR := 0.0
	if metrics.WinningTrades > 0 {
		avgWinR = totalWinR / float64(metrics.WinningTrades)
	}
	if metrics.LosingTrades > 0 {
		avgLossR = totalLossR / float64(metrics.LosingTrades)
	}
	metrics.ExpectancyR = (metrics.WinRate * avgWinR) - (lossRate * avgLossR)

	if totalLossPnL > 0 {
		metrics.ProfitFactor = totalWinPnL / totalLossPnL
	}

	metrics.MaxDrawdown, metrics.MaxDrawdownR = calculateMaxDrawdown(trades)
	metrics.ConsecutiveWins = maxConsecWins
	metrics.ConsecutiveLosses = maxConsecLosses

	return metrics
}

func calculateMaxDrawdown(trades []BacktestTrade) (float64, float64) {
	if len(trades) == 0 {
		return 0.0, 0.0
	}

	equity := 0.0
	equityR := 0.0
	peak := 0.0
	peakR := 0.0
	maxDD := 0.0
	maxDDR := 0.0

	for _, trade := range trades {
		equity += trade.PnL
		equityR += trade.PnLR
		
		if equity > peak {
			peak = equity
		}
		if equityR > peakR {
			peakR = equityR
		}
		
		dd := peak - equity
		ddR := peakR - equityR
		
		if dd > maxDD {
			maxDD = dd
		}
		if ddR > maxDDR {
			maxDDR = ddR
		}
	}

	return maxDD, maxDDR
}

func PrintBacktestReport(metrics *BacktestMetrics, trades []BacktestTrade, symbol string) string {
        reportPipSize := constants.GetPipSizeForSymbol(symbol)
	report := fmt.Sprintf(`
================================================================================
                         BACKTEST REPORT - H&S PATTERN DETECTION
================================================================================

PATTERN STATISTICS
------------------
Total Patterns Detected:    %d
Patterns Traded:            %d
Patterns Filtered:          %d (below confidence/RR threshold)

TRADE STATISTICS
----------------
Total Trades:               %d
Winning Trades:             %d (%.1f%%)
Losing Trades:              %d (%.1f%%)
Breakeven Trades:           %d
Timeout Exits:              %d

PERFORMANCE METRICS
-------------------
Win Rate:                   %.2f%%
Average Win:                %.5f (%.1f pips)
Average Loss:               %.5f (%.1f pips)
Average R:R Ratio:          %.2f

EXPECTANCY
----------
Expectancy (per trade):     %.5f (%.1f pips)
Expectancy (R):             %.2fR per trade
Profit Factor:              %.2f

P&L SUMMARY
-----------
Total P&L:                  %.5f (%.1f pips)
Total P&L (R):              %.2fR
Max Drawdown:               %.5f (%.2fR)

TRADE QUALITY
-------------
Largest Win:                %.5f (%.1f pips)
Largest Loss:               %.5f (%.1f pips)
Max Consecutive Wins:       %d
Max Consecutive Losses:     %d
Avg Bars in Trade:          %.1f

================================================================================
`,
		metrics.TotalPatterns,
		metrics.TradedPatterns,
		metrics.FilteredPatterns,
		metrics.TotalTrades,
		metrics.WinningTrades, metrics.WinRate*100,
		metrics.LosingTrades, (1-metrics.WinRate)*100,
		metrics.BreakevenTrades,
		metrics.TimeoutTrades,
		metrics.WinRate*100,
		metrics.AvgWin, metrics.AvgWin/reportPipSize,
		metrics.AvgLoss, metrics.AvgLoss/reportPipSize,
		metrics.AvgRR,
		metrics.Expectancy, metrics.Expectancy/reportPipSize,
		metrics.ExpectancyR,
		metrics.ProfitFactor,
		metrics.TotalPnL, metrics.TotalPnLPips,
		metrics.TotalPnLR,
		metrics.MaxDrawdown, metrics.MaxDrawdownR,
		metrics.LargestWin, metrics.LargestWin/reportPipSize,
		metrics.LargestLoss, metrics.LargestLoss/reportPipSize,
		metrics.ConsecutiveWins,
		metrics.ConsecutiveLosses,
		metrics.AvgBarsInTrade,
	)

	return report
}
