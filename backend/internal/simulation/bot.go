package simulation

import (
	"math"
	"time"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/patterns"
)

// TradingBot makes trading decisions based only on visible data
type TradingBot struct {
	feeder       *MarketFeeder
	state        *SimulationState
	config       SimulationConfig
	pendingOrder *patterns.TradeSignal // Order waiting to be filled
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
// Returns: (new signal generated, trade closed this bar, order filled this bar, error message)
func (b *TradingBot) ProcessBar() (*patterns.TradeSignal, *TradeResult, bool, string) {
	currentBar := b.feeder.NextBar()
	if currentBar == nil {
		return nil, nil, false, "No more bars"
	}

	b.state.CurrentBar = b.feeder.CurrentIndex()

	var closedTrade *TradeResult
	orderFilled := false

	// Step 1: Check exits on open positions FIRST
	if b.state.OpenPosition != nil {
		closedTrade = b.checkExits(currentBar)
	}

	// Step 2: Check if pending order fills (only if no open position)
	if b.state.OpenPosition == nil && b.pendingOrder != nil {
		filled, fillPrice := b.checkOrderFill(currentBar, b.pendingOrder)
		if filled {
			b.executeAtPrice(b.pendingOrder, fillPrice, currentBar)
			orderFilled = true
		}
		// Cancel unfilled order after this bar (could make this configurable)
		b.pendingOrder = nil
	}

	// Step 3: Look for new signals (only if no position and no pending order)
	var newSignal *patterns.TradeSignal
	if b.state.OpenPosition == nil && b.pendingOrder == nil {
		newSignal = b.analyzeForSignals()
		if newSignal != nil {
			b.pendingOrder = newSignal
		}
	}

	return newSignal, closedTrade, orderFilled, ""
}

// checkOrderFill checks if a pending order would fill on this bar
// Returns (filled, actualFillPrice)
func (b *TradingBot) checkOrderFill(bar *patterns.Candle, order *patterns.TradeSignal) (bool, float64) {
	if order == nil {
		return false, 0
	}

	if order.Direction == patterns.DirectionShort {
		// SHORT entry: price must go DOWN to entry level
		// Entry is below neckline, so we need Low <= EntryPrice

		// Case 1: Bar opens at or below entry - fill at open (slippage against us)
		if bar.Open <= order.EntryPrice {
			return true, bar.Open
		}

		// Case 2: Bar's low reaches entry level - fill at entry price
		if bar.Low <= order.EntryPrice {
			return true, order.EntryPrice
		}

		// Case 3: Entry not reached
		return false, 0

	} else { // LONG
		// LONG entry: price must go UP to entry level
		// Entry is above neckline, so we need High >= EntryPrice

		// Case 1: Bar opens at or above entry - fill at open (slippage against us)
		if bar.Open >= order.EntryPrice {
			return true, bar.Open
		}

		// Case 2: Bar's high reaches entry level - fill at entry price
		if bar.High >= order.EntryPrice {
			return true, order.EntryPrice
		}

		// Case 3: Entry not reached
		return false, 0
	}
}

// executeAtPrice opens position at the actual fill price
func (b *TradingBot) executeAtPrice(signal *patterns.TradeSignal, fillPrice float64, bar *patterns.Candle) {
	// Recalculate R:R based on actual fill price
	var risk, reward float64
	if signal.Direction == patterns.DirectionShort {
		risk = signal.StopLoss - fillPrice
		reward = fillPrice - signal.TakeProfit
	} else {
		risk = fillPrice - signal.StopLoss
		reward = signal.TakeProfit - fillPrice
	}

	actualRR := 0.0
	if risk > 0 {
		actualRR = reward / risk
	}

	b.state.OpenPosition = &Position{
		ID:          len(b.state.ClosedTrades) + 1,
		Direction:   signal.Direction,
		EntryPrice:  fillPrice, // Use actual fill price, not signal price
		StopLoss:    signal.StopLoss,
		TakeProfit:  signal.TakeProfit,
		EntryBar:    b.state.CurrentBar,
		EntryTime:   bar.Timestamp,
		PatternType: signal.PatternType,
		Confidence:  signal.Confidence,
		RiskReward:  actualRR,
	}

	b.state.LastTradeBar[signal.PatternType] = b.state.CurrentBar
}

// checkExits checks if SL, TP, or timeout is hit
// IMPORTANT: For realism, if both SL and TP could be hit in same bar,
// we assume SL hit first (conservative/worst case)
func (b *TradingBot) checkExits(currentBar *patterns.Candle) *TradeResult {
	pos := b.state.OpenPosition
	if pos == nil {
		return nil
	}

	barsHeld := b.state.CurrentBar - pos.EntryBar

	if pos.Direction == patterns.DirectionShort {
		// SHORT position
		slHit := currentBar.High >= pos.StopLoss
		tpHit := currentBar.Low <= pos.TakeProfit

		if slHit && tpHit {
			// Both hit in same bar - ASSUME SL HIT FIRST (conservative)
			// This is the worst case assumption
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		} else if slHit {
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		} else if tpHit {
			return b.closePosition(pos.TakeProfit, currentBar.Timestamp, "TP_HIT", barsHeld)
		}

	} else { // LONG position
		slHit := currentBar.Low <= pos.StopLoss
		tpHit := currentBar.High >= pos.TakeProfit

		if slHit && tpHit {
			// Both hit in same bar - ASSUME SL HIT FIRST (conservative)
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		} else if slHit {
			return b.closePosition(pos.StopLoss, currentBar.Timestamp, "SL_HIT", barsHeld)
		} else if tpHit {
			return b.closePosition(pos.TakeProfit, currentBar.Timestamp, "TP_HIT", barsHeld)
		}
	}

	// Check timeout
	if barsHeld >= b.config.MaxBarsToHold {
		return b.closePosition(currentBar.Close, currentBar.Timestamp, "TIMEOUT", barsHeld)
	}

	return nil
}

// analyzeForSignals looks for H&S or IHS patterns in visible data
func (b *TradingBot) analyzeForSignals() *patterns.TradeSignal {
	window := b.feeder.GetVisibleWindow(30)
	if len(window) < 20 {
		return nil
	}

	lookback := patterns.LookbackByTimeframe(b.config.Timeframe)
	currentBarIdx := b.feeder.CurrentIndex()

	// Check cooldown for H&S
	lastHS, hasHS := b.state.LastTradeBar[patterns.PatternHS]
	if !hasHS || currentBarIdx-lastHS > b.config.CooldownBars {
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
	ctx := patterns.DetermineMarketContext(window)
	finalConfidence := patterns.FinalConfidence(structure, ctx)

	if finalConfidence < b.config.ConfidenceThreshold {
		return nil
	}

	signal := patterns.GenerateTradeSignal(structure, b.config.Symbol, finalConfidence)
	if signal == nil {
		return nil
	}

	if signal.RiskReward < b.config.MinRR {
		return nil
	}


	return signal
}

// ExecuteSignal is now deprecated - orders are placed as pending
// and filled via checkOrderFill on the next bar
func (b *TradingBot) ExecuteSignal(signal *patterns.TradeSignal) {
	// This is now handled internally via pendingOrder
	// Kept for backward compatibility but does nothing
}

// closePosition closes the current position and calculates results
func (b *TradingBot) closePosition(exitPrice float64, exitTime time.Time, reason string, barsHeld int) *TradeResult {
	pos := b.state.OpenPosition
	if pos == nil {
		return nil
	}

	var pnl float64
	if pos.Direction == patterns.DirectionShort {
		pnl = pos.EntryPrice - exitPrice
	} else {
		pnl = exitPrice - pos.EntryPrice
	}

	    pnlPips := pnl / constants.GetPipSizeForSymbol(b.config.Symbol)

	risk := math.Abs(pos.StopLoss - pos.EntryPrice)
	pnlR := 0.0
	if risk > 0 {
		pnlR = pnl / risk
	}

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
		ExitReason: reason,
		BarsHeld:  barsHeld,
	}

	b.state.ClosedTrades = append(b.state.ClosedTrades, result)
	b.state.OpenPosition = nil
	b.state.AccountBalanceR += pnlR

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
