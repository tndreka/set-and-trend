package confluence

import (
	"set-and-trend/backend/internal/patterns"
)

// ConfluenceBacktester wraps backtest with confluence filtering
type ConfluenceBacktester struct {
	scorer *ConfluenceScorer
	config BacktestConfig
}

// BacktestConfig for confluence backtesting
type BacktestConfig struct {
	MinConfluence float64
	PatternConfig patterns.BacktestConfig
}

// NewConfluenceBacktester creates backtester with confluence
func NewConfluenceBacktester(config BacktestConfig) *ConfluenceBacktester {
	return &ConfluenceBacktester{
		scorer: NewConfluenceScorer(DefaultConfluenceConfig()),
		config: config,
	}
}

// RunConfluenceBacktest runs backtest with confluence filtering
func (cb *ConfluenceBacktester) RunConfluenceBacktest(
	weeklyCandles []patterns.Candle,
	dailyCandles []patterns.Candle,
	h4Candles []patterns.Candle,
	symbol string,
) (*patterns.BacktestMetrics, []patterns.BacktestTrade, []ChecklistScore) {
	
	var confluenceTrades []patterns.BacktestTrade
	var scores []ChecklistScore
	totalPatterns := 0
	filteredByConfluence := 0
	
	// Align timeframes - use daily as base for bar-by-bar analysis
	for i := 60; i < len(dailyCandles)-20; i++ {
		
		// Get aligned candles for each timeframe
		dWindow := dailyCandles[i-30 : i]
		wIdx := i / 5 // Approximate weekly alignment
		h4Idx := i * 6 // Approximate 4H alignment
		
		var wCandles, h4Window []patterns.Candle
		
		if wIdx >= 20 && wIdx < len(weeklyCandles) {
			wCandles = weeklyCandles[wIdx-20 : wIdx]
		}
		if h4Idx >= 40 && h4Idx < len(h4Candles) {
			h4Window = h4Candles[h4Idx-40 : h4Idx]
		}
		
		// Calculate confluence
		score, signal := cb.scorer.CalculateConfluence(
			wCandles, dWindow, h4Window, nil, nil, nil, symbol,
		)
		
		if score != nil {
			scores = append(scores, *score)
			
			// Check if we have a valid pattern
			hs := patterns.DetectHeadAndShoulders(dWindow, 3)
			ihs := patterns.DetectInverseHeadAndShoulders(dWindow, 3)
			
			if hs != nil || ihs != nil {
				totalPatterns++
				
				// Filter by confluence
				if score.TotalPercent < cb.config.MinConfluence {
					filteredByConfluence++
					continue
				}
				
				// Process trade if confluence >= threshold
				if signal != nil {
					trade := cb.processConfluenceTrade(signal, dailyCandles[i:i+20], i)
					if trade != nil {
						confluenceTrades = append(confluenceTrades, *trade)
					}
				}
			}
		}
	}
	
	metrics := patterns.CalculateBacktestMetrics(confluenceTrades)
	metrics.TotalPatterns = totalPatterns
	metrics.FilteredPatterns = filteredByConfluence
	metrics.TradedPatterns = len(confluenceTrades)
	
	return metrics, confluenceTrades, scores
}

func (cb *ConfluenceBacktester) processConfluenceTrade(
	signal *patterns.TradeSignal,
	futureData []patterns.Candle,
	barIndex int,
) *patterns.BacktestTrade {
	
	// Use existing SimulateExit logic
	exit := patterns.SimulateExit(signal, signal.EntryPrice, futureData, cb.config.PatternConfig)
	if exit == nil {
		return nil
	}
	
	// Calculate R-multiple
	risk := signal.StopLoss - signal.EntryPrice
	if signal.Direction == patterns.DirectionLong {
		risk = signal.EntryPrice - signal.StopLoss
	}
	
	pnlR := 0.0
	if risk != 0 {
		pnlR = exit.PnL / risk
	}
	
	result := "BREAKEVEN"
	if exit.Reason == "TP_HIT" {
		result = "WIN"
	} else if exit.Reason == "SL_HIT" {
		result = "LOSS"
	}
	
	return &patterns.BacktestTrade{
		EntryTime:   futureData[0].Timestamp,
		ExitTime:    exit.ExitTime,
		EntryPrice:  signal.EntryPrice,
		ExitPrice:   exit.ExitPrice,
		StopLoss:    signal.StopLoss,
		TakeProfit:  signal.TakeProfit,
		Direction:   signal.Direction,
		Confidence:  signal.Confidence, // This is now confluence score
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