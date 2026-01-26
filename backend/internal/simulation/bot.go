package simulation

import (
	"math"

	"set-and-trend/backend/internal/patterns"
)

// TradingBot makes trading decisions based only on visible data
type TradingBot struct {
	feeder  *MarketFeeder
	state   *SimulationState
	config  SimulationConfig
}

// NewTradingBot creates a new bot instance
func NewTradingBot(feeder *MarketFeeder, config SimulationConfig) *TradingBot {
	return &TradingBot{
		feeder: feeder,
		state:  NewSimulationState(feeder.TotalBars()),
		config: config,
	}
}

// GetState returns current simulation state
func (b *TradingBot) GetState() *SimulationState {
	return b.state
}

// ProcessBar processes the current bar and returns any action taken
// Returns: (signal, trade closed this bar, error message)
func (b *TradingBot) ProcessBar() (*patterns.TradeSignal, *TradeResult, string) {
	// Get the current bar (this "reveals" it to the simulation)
	currentBar := b.feeder.NextBar()
	if currentBar == nil {
		return nil, nil, "No more bars"
	}
	
	b.state.CurrentBar = b.feeder.CurrentIndex()
	
	// First: Check if we need to exit an open position
	var closedTrade *TradeResult
	if b.state.OpenPosition != nil {
		closedTrade = b.checkExits(currentBar)
	}
	
	// Second: If no open position, look for new signals
	var newSignal *patterns.TradeSignal
	if b.state.OpenPosition == nil {
		newSignal = b.analyzeForSignals()
	}
	
	return newSignal, closedTrade, ""
}

// analyzeForSignals looks for H&S or IHS patterns in visible data
func (b *TradingBot) analyzeForSignals() *patterns.TradeSignal {
	// Get visible window for analysis
	window := b.feeder.GetVisibleWindow(30) // Use 30 bars for pattern detection
	if len(window) < 20 {
		return nil // Not enough data
	}
	
	lookback := patterns.LookbackByTimeframe(b.config.Timeframe)
	currentBarIdx := b.feeder.CurrentIndex()
	
	// Check cooldown for H&S
	lastHS, hasHS := b.state.LastTradeBar[patterns.PatternHS]
	if !hasHS || currentBarIdx-lastHS > b.config.CooldownBars {
		// Detect H&S (bearish)
		hsStructure := patterns.DetectHeadAndShoulders(window, lookback)
		if hsStructure != nil {
			signal := b.processStructure(hsStructure, window)
			if signal != nil {
				return signal
			}
		}
	}
	
	// Check cooldown for IHS
	lastIHS, hasIHS := b.state.LastTradeBar[patterns.PatternIHS]
	if !hasIHS || currentBarIdx-lastIHS > b.config.CooldownBars {
		// Detect IHS (bullish)
		ihsStructure := patterns.DetectInverseHeadAndShoulders(window, lookback)
		if ihsStructure != nil {
			signal := b.processStructure(ihsStructure, window)
			if signal != nil {
				return signal
			}
		}
	}
	
	return nil
}

// processStructure validates and generates signal from detected pattern
func (b *TradingBot) processStructure(structure *patterns.DetectedStructure, window []patterns.Candle) *patterns.TradeSignal {
	// Calculate context-adjusted confidence
	ctx := patterns.DetermineMarketContext(window)
	finalConfidence := patterns.FinalConfidence(structure, ctx)
	
	// Check confidence threshold
	if finalConfidence < b.config.ConfidenceThreshold {
		return nil
	}
	
	// Generate signal
	signal := patterns.GenerateTradeSignal(structure, b.config.Symbol, finalConfidence)
	if signal == nil {
		return nil
	}
	
	// Check R:R threshold
	if signal.RiskReward < b.config.MinRR {
		return nil
	}
	
	// Validate signal sanity
	if !patterns.ValidateSignal(signal, b.config.MinRR, b.config.ConfidenceThreshold) {
		return nil
	}
	
	return signal
}

// ExecuteSignal opens a new position based on signal
func (b *TradingBot) ExecuteSignal(signal *patterns.TradeSignal) {
	if signal == nil || b.state.OpenPosition != nil {
		return
	}
	
	visible := b.feeder.GetVisibleCandles()
	entryTime := visible[len(visible)-1].Timestamp
	
	b.state.OpenPosition = &Position{
		ID:          len(b.state.ClosedTrades) + 1,
		Direction:   signal.Direction,
		EntryPrice:  signal.EntryPrice,
		StopLoss:    signal.StopLoss,
		TakeProfit:  signal.TakeProfit,
		EntryBar:    b.state.CurrentBar,
		EntryTime:   entryTime,
		PatternType: signal.PatternType,
		Confidence:  signal.Confidence,
		RiskReward:  signal.RiskReward,
	}
	
	// Update last trade bar for this pattern type
	b.state.LastTradeBar[signal.PatternType] = b.state.CurrentBar
}

// checkExits checks if SL, TP, or timeout is hit
func (b *TradingBot) checkExits(currentBar *patterns.Candle) *TradeResult {
	pos := b.state.OpenPosition
	if pos == nil {
		return nil
	}
	
	barsHeld := b.state.CurrentBar - pos.EntryBar
	
	// Check exit conditions based on direction
	if pos.Direction == patterns.DirectionShort {
		// SHORT: Check SL first (conservative - assume worst case)
		if currentBar.High >= pos.StopLoss {
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		}
		// Check TP
		if currentBar.Low <= pos.TakeProfit {
			return b.closePosition(pos.TakeProfit, currentBar.Timestamp, "TP_HIT", barsHeld)
		}
	} else { // LONG
		// LONG: Check SL first
		if currentBar.Low <= pos.StopLoss {
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		}
		// Check TP
		if currentBar.High >= pos.TakeProfit {
			return b.closePosition(pos.TakeProfit, currentBar.Timestamp, "TP_HIT", barsHeld)
		}
	}
	
	// Check timeout
	if barsHeld >= b.config.MaxBarsToHold {
		return b.closePosition(currentBar.Close, currentBar.Timestamp, "TIMEOUT", barsHeld)
	}
	
	return nil
}

// closePosition closes the current position and calculates results
func (b *TradingBot) closePosition(exitPrice float64, exitTime interface{}, reason string, barsHeld int) *TradeResult {
	pos := b.state.OpenPosition
	if pos == nil {
		return nil
	}
	
	// Calculate P&L
	var pnl float64
	if pos.Direction == patterns.DirectionShort {
		pnl = pos.EntryPrice - exitPrice
	} else {
		pnl = exitPrice - pos.EntryPrice
	}
	
	pnlPips := pnl / patterns.PipSize
	
	// Calculate R multiple
	risk := math.Abs(pos.StopLoss - pos.EntryPrice)
	pnlR := 0.0
	if risk > 0 {
		pnlR = pnl / risk
	}
	
	// Determine outcome
	outcome := "BREAKEVEN"
	if reason == "TP_HIT" {
		outcome = "WIN"
	} else if reason == "SL_HIT" {
		outcome = "LOSS"
	} else if reason == "TIMEOUT" {
		if pnlR > 0.5 {
			outcome = "WIN"
		} else if pnlR < -0.5 {
			outcome = "LOSS"
		}
	}
	
	result := TradeResult{
		Position:  *pos,
		ExitPrice: exitPrice,
		ExitBar:   b.state.CurrentBar,
		PnL:       pnl,
		PnLPips:   pnlPips,
		PnLR:      pnlR,
		Outcome:   outcome,
		BarsHeld:  barsHeld,
	}
	
	// Update state
	b.state.ClosedTrades = append(b.state.ClosedTrades, result)
	b.state.OpenPosition = nil
	b.state.AccountBalanceR += pnlR
	
	// Update peak and drawdown
	if b.state.AccountBalanceR > b.state.PeakBalanceR {
		b.state.PeakBalanceR = b.state.AccountBalanceR
	}
	drawdown := b.state.PeakBalanceR - b.state.AccountBalanceR
	if drawdown > b.state.MaxDrawdownR {
		b.state.MaxDrawdownR = drawdown
	}
	
	return &result
}

// GetStats calculates current statistics
func (b *TradingBot) GetStats() SimulationStats {
	stats := SimulationStats{}
	trades := b.state.ClosedTrades
	
	if len(trades) == 0 {
		return stats
	}
	
	stats.TotalTrades = len(trades)
	var totalBars int
	
	for _, t := range trades {
		totalBars += t.BarsHeld
		
		if t.Outcome == "WIN" {
			stats.WinningTrades++
			if t.PatternType == patterns.PatternHS {
				stats.HSWins++
			} else {
				stats.IHSWins++
			}
		} else if t.Outcome == "LOSS" {
			stats.LosingTrades++
			if t.PatternType == patterns.PatternHS {
				stats.HSLosses++
			} else {
				stats.IHSLosses++
			}
		}
	}
	
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
		stats.AvgBarsHeld = float64(totalBars) / float64(stats.TotalTrades)
	}
	
	stats.TotalPnLR = b.state.AccountBalanceR
	stats.MaxDrawdownR = b.state.MaxDrawdownR
	
	// Calculate expectancy
	lossRate := float64(stats.LosingTrades) / float64(stats.TotalTrades)
	avgWinR := 0.0
	avgLossR := 0.0
	if stats.WinningTrades > 0 {
		var totalWinR float64
		for _, t := range trades {
			if t.Outcome == "WIN" {
				totalWinR += t.PnLR
			}
		}
		avgWinR = totalWinR / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		var totalLossR float64
		for _, t := range trades {
			if t.Outcome == "LOSS" {
				totalLossR += math.Abs(t.PnLR)
			}
		}
		avgLossR = totalLossR / float64(stats.LosingTrades)
	}
	stats.Expectancy = (stats.WinRate * avgWinR) - (lossRate * avgLossR)
	
	// Calculate streaks
	currentStreak := 0
	maxWinStreak := 0
	maxLossStreak := 0
	
	for _, t := range trades {
		if t.Outcome == "WIN" {
			if currentStreak > 0 {
				currentStreak++
			} else {
				currentStreak = 1
			}
			if currentStreak > maxWinStreak {
				maxWinStreak = currentStreak
			}
		} else if t.Outcome == "LOSS" {
			if currentStreak < 0 {
				currentStreak--
			} else {
				currentStreak = -1
			}
			if -currentStreak > maxLossStreak {
				maxLossStreak = -currentStreak
			}
		}
	}
	
	stats.ConsecutiveWins = maxWinStreak
	stats.ConsecutiveLosses = maxLossStreak
	stats.CurrentStreak = currentStreak
	
	return stats
}
