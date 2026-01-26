package patterns

import (
	"fmt"
	"math"
	"time"
)

// BacktestConfig holds backtesting parameters
type BacktestConfig struct {
	WindowSize          int
	ConfidenceThreshold float64
	MinRR               float64
	MaxBarsToExit       int
	Lookback            int
	CooldownBars        int // Minimum bars between trades to avoid duplicates
}

// DefaultBacktestConfig returns sensible defaults
func DefaultBacktestConfig() BacktestConfig {
	return BacktestConfig{
		WindowSize:          30,
		ConfidenceThreshold: 0.60,
		MinRR:               1.5,
		MaxBarsToExit:       20,
		Lookback:            3,
		CooldownBars:        10, // Don't take another trade for 10 bars
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

// SimulateExit simulates trade exit on future candles
func SimulateExit(signal *TradeSignal, futureCandles []Candle, maxBars int) *ExitResult {
	if signal == nil || len(futureCandles) == 0 {
		return nil
	}

	barsToCheck := len(futureCandles)
	if maxBars > 0 && maxBars < barsToCheck {
		barsToCheck = maxBars
	}

	for i := 0; i < barsToCheck; i++ {
		candle := futureCandles[i]

		if signal.Direction == DirectionShort {
			// SHORT: Check SL first (conservative)
			if candle.High >= signal.StopLoss {
				return &ExitResult{
					ExitPrice: signal.StopLoss,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "SL_HIT",
					PnL:       signal.EntryPrice - signal.StopLoss,
					PnLPips:   (signal.EntryPrice - signal.StopLoss) / PipSize,
				}
			}
			if candle.Low <= signal.TakeProfit {
				return &ExitResult{
					ExitPrice: signal.TakeProfit,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "TP_HIT",
					PnL:       signal.EntryPrice - signal.TakeProfit,
					PnLPips:   (signal.EntryPrice - signal.TakeProfit) / PipSize,
				}
			}
		} else if signal.Direction == DirectionLong {
			// LONG: Check SL first
			if candle.Low <= signal.StopLoss {
				return &ExitResult{
					ExitPrice: signal.StopLoss,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "SL_HIT",
					PnL:       signal.StopLoss - signal.EntryPrice,
					PnLPips:   (signal.StopLoss - signal.EntryPrice) / PipSize,
				}
			}
			if candle.High >= signal.TakeProfit {
				return &ExitResult{
					ExitPrice: signal.TakeProfit,
					ExitTime:  candle.Timestamp,
					ExitBar:   i,
					Reason:    "TP_HIT",
					PnL:       signal.TakeProfit - signal.EntryPrice,
					PnLPips:   (signal.TakeProfit - signal.EntryPrice) / PipSize,
				}
			}
		}
	}

	// Timeout exit
	lastCandle := futureCandles[barsToCheck-1]
	var pnl float64
	if signal.Direction == DirectionShort {
		pnl = signal.EntryPrice - lastCandle.Close
	} else {
		pnl = lastCandle.Close - signal.EntryPrice
	}

	return &ExitResult{
		ExitPrice: lastCandle.Close,
		ExitTime:  lastCandle.Timestamp,
		ExitBar:   barsToCheck - 1,
		Reason:    "TIMEOUT",
		PnL:       pnl,
		PnLPips:   pnl / PipSize,
	}
}

// RunBacktest executes backtest on historical data with cooldown to prevent duplicates
func RunBacktest(candles []Candle, config BacktestConfig) (*BacktestMetrics, []BacktestTrade) {
	if len(candles) < config.WindowSize+config.MaxBarsToExit {
		return nil, nil
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

	if signal.RiskReward < config.MinRR {
		return nil
	}

	exit := SimulateExit(signal, futureData, config.MaxBarsToExit)
	if exit == nil {
		return nil
	}

	risk := math.Abs(signal.StopLoss - signal.EntryPrice)
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
		EntryPrice:  signal.EntryPrice,
		ExitPrice:   exit.ExitPrice,
		StopLoss:    signal.StopLoss,
		TakeProfit:  signal.TakeProfit,
		Direction:   signal.Direction,
		PatternType: structure.PatternType,
		Confidence:  finalConfidence,
		RiskReward:  signal.RiskReward,
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

func PrintBacktestReport(metrics *BacktestMetrics, trades []BacktestTrade) string {
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
		metrics.AvgWin, metrics.AvgWin/PipSize,
		metrics.AvgLoss, metrics.AvgLoss/PipSize,
		metrics.AvgRR,
		metrics.Expectancy, metrics.Expectancy/PipSize,
		metrics.ExpectancyR,
		metrics.ProfitFactor,
		metrics.TotalPnL, metrics.TotalPnLPips,
		metrics.TotalPnLR,
		metrics.MaxDrawdown, metrics.MaxDrawdownR,
		metrics.LargestWin, metrics.LargestWin/PipSize,
		metrics.LargestLoss, metrics.LargestLoss/PipSize,
		metrics.ConsecutiveWins,
		metrics.ConsecutiveLosses,
		metrics.AvgBarsInTrade,
	)

	return report
}
