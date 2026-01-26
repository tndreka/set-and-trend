package simulation

import (
	"fmt"
	"strings"

	"set-and-trend/backend/internal/patterns"
)

// Executor manages the simulation execution flow
type Executor struct {
	bot        *TradingBot
	feeder     *MarketFeeder
	config     SimulationConfig
	visualizer *Visualizer
}

// NewExecutor creates a new simulation executor
func NewExecutor(candles []patterns.Candle, config SimulationConfig) *Executor {
	feeder := NewMarketFeeder(candles)
	bot := NewTradingBot(feeder, config)
	visualizer := NewVisualizer(config)

	return &Executor{
		bot:        bot,
		feeder:     feeder,
		config:     config,
		visualizer: visualizer,
	}
}

// SkipTo advances simulation to a specific bar without processing
func (e *Executor) SkipTo(barIndex int) {
	e.feeder.Seek(barIndex)
	e.bot.state.CurrentBar = barIndex
}

// Step processes one bar and returns display output
func (e *Executor) Step() (output string, done bool) {
	if !e.feeder.HasMoreBars() {
		return e.GetFinalReport(), true
	}

	barIdx := e.feeder.CurrentIndex()
	currentBar := e.feeder.PeekCurrentBar()

	// Process the bar - now returns 4 values
	signal, closedTrade, orderFilled, errMsg := e.bot.ProcessBar()

	if errMsg != "" {
		return errMsg, true
	}

	var sb strings.Builder

	if e.config.ShowChart {
		visible := e.feeder.GetVisibleWindow(40)
		chartOutput := e.visualizer.RenderChart(visible, e.bot.state, currentBar)
		sb.WriteString(chartOutput)
	}

	stats := e.bot.GetStats()
	sb.WriteString(fmt.Sprintf("\n%s %s - Bar %d/%d | Account: %.2fR | Win Rate: %.1f%%\n",
		e.config.Symbol,
		e.config.Timeframe,
		barIdx+1,
		e.feeder.TotalBars(),
		stats.TotalPnLR,
		stats.WinRate*100,
	))

	if closedTrade != nil {
		sb.WriteString(fmt.Sprintf("\n[CLOSED] %s %s @ %.5f | P&L: %.2fR | %s\n",
			closedTrade.Direction,
			closedTrade.PatternType,
			closedTrade.ExitPrice,
			closedTrade.PnLR,
			closedTrade.Outcome,
		))
	}

	if orderFilled && e.bot.state.OpenPosition != nil {
		pos := e.bot.state.OpenPosition
		sb.WriteString(fmt.Sprintf("\n[FILLED] %s @ %.5f | SL: %.5f | TP: %.5f | RR: %.2f\n",
			pos.Direction,
			pos.EntryPrice,
			pos.StopLoss,
			pos.TakeProfit,
			pos.RiskReward,
		))
	}

	if signal != nil && !orderFilled {
		sb.WriteString(fmt.Sprintf("\n[PENDING ORDER] %s @ %.5f | SL: %.5f | TP: %.5f | Waiting for fill...\n",
			signal.Direction,
			signal.EntryPrice,
			signal.StopLoss,
			signal.TakeProfit,
		))
	}

	if e.bot.state.OpenPosition != nil {
		pos := e.bot.state.OpenPosition
		barsHeld := barIdx - pos.EntryBar
		sb.WriteString(fmt.Sprintf("\n[OPEN] %s %s @ %.5f | SL: %.5f | TP: %.5f | Bars: %d/%d\n",
			pos.Direction,
			pos.PatternType,
			pos.EntryPrice,
			pos.StopLoss,
			pos.TakeProfit,
			barsHeld,
			e.config.MaxBarsToHold,
		))
	}

	return sb.String(), false
}

// RunFast runs entire simulation without display and returns final report
func (e *Executor) RunFast() string {
	for e.feeder.HasMoreBars() {
		e.bot.ProcessBar()
	}

	return e.GetFinalReport()
}

// GetFinalReport generates the complete simulation report
func (e *Executor) GetFinalReport() string {
	stats := e.bot.GetStats()
	trades := e.bot.state.ClosedTrades

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("===============================================================================\n")
	sb.WriteString("                    H&S PATTERN TRADING SIMULATION COMPLETE\n")
	sb.WriteString("===============================================================================\n\n")

	sb.WriteString(fmt.Sprintf("Period: %s to %s (%d bars)\n",
		e.config.StartDate.Format("2006-01-02"),
		e.config.EndDate.Format("2006-01-02"),
		e.feeder.TotalBars(),
	))
	sb.WriteString(fmt.Sprintf("Timeframe: %s\n\n", e.config.Timeframe))

	sb.WriteString("TRADING RESULTS:\n")
	sb.WriteString(fmt.Sprintf("  Total Trades: %d\n", stats.TotalTrades))
	sb.WriteString(fmt.Sprintf("  Winners: %d (%.1f%%)\n", stats.WinningTrades, stats.WinRate*100))
	sb.WriteString(fmt.Sprintf("  Losers: %d (%.1f%%)\n\n", stats.LosingTrades, (1-stats.WinRate)*100))

	sb.WriteString(fmt.Sprintf("  Total P&L: %+.2fR\n", stats.TotalPnLR))

	avgWinR := 0.0
	avgLossR := 0.0
	if stats.WinningTrades > 0 {
		var totalWin float64
		for _, t := range trades {
			if t.Outcome == "WIN" {
				totalWin += t.PnLR
			}
		}
		avgWinR = totalWin / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		var totalLoss float64
		for _, t := range trades {
			if t.Outcome == "LOSS" {
				totalLoss += t.PnLR
			}
		}
		avgLossR = totalLoss / float64(stats.LosingTrades)
	}

	sb.WriteString(fmt.Sprintf("  Average Win: %+.2fR\n", avgWinR))
	sb.WriteString(fmt.Sprintf("  Average Loss: %.2fR\n", avgLossR))
	sb.WriteString(fmt.Sprintf("  Expectancy: %+.2fR per trade\n\n", stats.Expectancy))

	sb.WriteString(fmt.Sprintf("  Max Drawdown: -%.2fR\n", stats.MaxDrawdownR))
	sb.WriteString(fmt.Sprintf("  Max Consecutive Wins: %d\n", stats.ConsecutiveWins))
	sb.WriteString(fmt.Sprintf("  Max Consecutive Losses: %d\n\n", stats.ConsecutiveLosses))

	sb.WriteString("PATTERN BREAKDOWN:\n")
	hsTrades := stats.HSWins + stats.HSLosses
	ihsTrades := stats.IHSWins + stats.IHSLosses

	if hsTrades > 0 {
		hsWinRate := float64(stats.HSWins) / float64(hsTrades) * 100
		sb.WriteString(fmt.Sprintf("  H&S (Bearish): %d trades, %d wins (%.1f%%)\n", hsTrades, stats.HSWins, hsWinRate))
	}
	if ihsTrades > 0 {
		ihsWinRate := float64(stats.IHSWins) / float64(ihsTrades) * 100
		sb.WriteString(fmt.Sprintf("  IHS (Bullish): %d trades, %d wins (%.1f%%)\n", ihsTrades, stats.IHSWins, ihsWinRate))
	}

	sb.WriteString("\nEXECUTION MODEL:\n")
	sb.WriteString("  - Entry: Limit order at signal price, fills only if price reached\n")
	sb.WriteString("  - Slippage: If bar opens past entry, fill at open (worse price)\n")
	sb.WriteString("  - SL/TP: Conservative - if both hit same bar, assume SL first\n")

	sb.WriteString("\n===============================================================================\n")

	return sb.String()
}

// GetBot returns the trading bot for direct access
func (e *Executor) GetBot() *TradingBot {
	return e.bot
}

// GetFeeder returns the market feeder
func (e *Executor) GetFeeder() *MarketFeeder {
	return e.feeder
}
