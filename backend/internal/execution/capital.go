// Package execution provides trade execution, capital simulation, and risk management.
//
// CRITICAL: This module implements production-grade capital simulation for backtesting
// with $1000 starting balance and 1:3 leverage as per specification.
package execution

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"set-and-trend/backend/internal/constants"
)

// CapitalConfig holds capital simulation parameters.
// All values must be validated before use.
type CapitalConfig struct {
	StartingBalance  float64 // Starting account balance (e.g., 1000.0)
	Leverage         float64 // Account leverage (e.g., 3.0 for 1:3)
	RiskPerTrade     float64 // Risk per trade as % of balance (e.g., 0.01 = 1%)
	MaxMarginPercent float64 // Maximum margin usage (e.g., 0.30 = 30%)
	MinLotSize       float64 // Minimum lot size (e.g., 0.01)
	LotStep          float64 // Lot size increment (e.g., 0.01)
	MaxLotSize       float64 // Maximum lot size per trade (e.g., 1.0)
	MaxDrawdownPct   float64 // Maximum allowed drawdown before stopping (e.g., 0.20 = 20%)
}

// DefaultCapitalConfig returns sensible defaults for $1000 account with 1:3 leverage.
func DefaultCapitalConfig() CapitalConfig {
	return CapitalConfig{
		StartingBalance:  1000.0,
		Leverage:         3.0,
		RiskPerTrade:     0.01, // 1% risk per trade
		MaxMarginPercent: 0.30, // 30% max margin usage
		MinLotSize:       0.01,
		LotStep:          0.01,
		MaxLotSize:       1.0,
		MaxDrawdownPct:   0.20, // Stop at 20% drawdown
	}
}

// Validate checks all config values for validity.
func (c CapitalConfig) Validate() error {
	if c.StartingBalance <= 0 {
		return errors.New("starting balance must be positive")
	}
	if c.Leverage <= 0 {
		return errors.New("leverage must be positive")
	}
	if c.RiskPerTrade <= 0 || c.RiskPerTrade > 0.10 {
		return errors.New("risk per trade must be between 0 and 10%")
	}
	if c.MaxMarginPercent <= 0 || c.MaxMarginPercent > 1.0 {
		return errors.New("max margin percent must be between 0 and 100%")
	}
	if c.MinLotSize <= 0 {
		return errors.New("min lot size must be positive")
	}
	if c.LotStep <= 0 {
		return errors.New("lot step must be positive")
	}
	if c.MaxLotSize <= 0 || c.MaxLotSize < c.MinLotSize {
		return errors.New("max lot size must be >= min lot size")
	}
	if c.MaxDrawdownPct <= 0 || c.MaxDrawdownPct > 1.0 {
		return errors.New("max drawdown must be between 0 and 100%")
	}
	return nil
}

// Position represents an open trading position with full capital metrics.
type Position struct {
	ID            int       `json:"id"`
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"` // "LONG" or "SHORT"
	EntryPrice    float64   `json:"entry_price"`
	StopLoss      float64   `json:"stop_loss"`
	TakeProfit    float64   `json:"take_profit"`
	LotSize       float64   `json:"lot_size"`
	RiskAmount    float64   `json:"risk_amount"`  // $ amount risked
	RiskPercent   float64   `json:"risk_percent"` // % of balance risked
	MarginUsed    float64   `json:"margin_used"`  // Margin required for this position
	EntryTime     time.Time `json:"entry_time"`
	EntryBar      int       `json:"entry_bar"`
	PatternType   string    `json:"pattern_type"`
	Confidence    float64   `json:"confidence"`
	RiskReward    float64   `json:"risk_reward"`
	UnrealizedPnL float64   `json:"unrealized_pnl"` // Current floating P&L
}

// Validate checks if the position parameters are valid.
func (p Position) Validate() error {
	if p.Symbol == "" {
		return errors.New("symbol is required")
	}
	if p.Direction != "LONG" && p.Direction != "SHORT" {
		return errors.New("direction must be LONG or SHORT")
	}
	if p.EntryPrice <= 0 {
		return errors.New("entry price must be positive")
	}
	if p.LotSize <= 0 {
		return errors.New("lot size must be positive")
	}

	// Validate stop loss is on correct side
	if p.Direction == "LONG" {
		if p.StopLoss >= p.EntryPrice {
			return errors.New("LONG stop loss must be below entry price")
		}
		if p.TakeProfit <= p.EntryPrice {
			return errors.New("LONG take profit must be above entry price")
		}
	} else {
		if p.StopLoss <= p.EntryPrice {
			return errors.New("SHORT stop loss must be above entry price")
		}
		if p.TakeProfit >= p.EntryPrice {
			return errors.New("SHORT take profit must be below entry price")
		}
	}
	return nil
}

// TradeRecord represents a completed trade with full capital details.
type TradeRecord struct {
	ID            int       `json:"id"`
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"`
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	StopLoss      float64   `json:"stop_loss"`
	TakeProfit    float64   `json:"take_profit"`
	LotSize       float64   `json:"lot_size"`
	RiskAmount    float64   `json:"risk_amount"`  // $ risked
	RiskPercent   float64   `json:"risk_percent"` // % of balance risked
	PnL           float64   `json:"pnl"`          // Profit/Loss in $
	PnLPips       float64   `json:"pnl_pips"`
	PnLR          float64   `json:"pnl_r"`   // R multiple
	PnLPct        float64   `json:"pnl_pct"` // % return on balance
	MarginUsed    float64   `json:"margin_used"`
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
	EntryBar      int       `json:"entry_bar"`
	ExitBar       int       `json:"exit_bar"`
	BarsHeld      int       `json:"bars_held"`
	PatternType   string    `json:"pattern_type"`
	Confidence    float64   `json:"confidence"`
	ExitReason    string    `json:"exit_reason"` // "SL_HIT", "TP_HIT", "TIMEOUT"
	Result        string    `json:"result"`      // "WIN", "LOSS", "BREAKEVEN"
	BalanceBefore float64   `json:"balance_before"`
	BalanceAfter  float64   `json:"balance_after"`
}

// EquityPoint represents a single point on the equity curve.
type EquityPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	BarIndex    int       `json:"bar_index"`
	Balance     float64   `json:"balance"`
	Equity      float64   `json:"equity"`
	Drawdown    float64   `json:"drawdown"`
	DrawdownPct float64   `json:"drawdown_pct"`
	OpenPnL     float64   `json:"open_pnl"`
}

// CapitalState tracks the simulated account state.
type CapitalState struct {
	Config            CapitalConfig `json:"config"`
	Balance           float64       `json:"balance"`          // Current account balance
	Equity            float64       `json:"equity"`           // Balance + open PnL
	MarginUsed        float64       `json:"margin_used"`      // Current margin used
	MarginAvailable   float64       `json:"margin_available"` // Available margin
	PeakEquity        float64       `json:"peak_equity"`      // Highest equity reached
	MaxDrawdown       float64       `json:"max_drawdown"`     // Maximum drawdown in $
	MaxDrawdownPct    float64       `json:"max_drawdown_pct"` // Maximum drawdown as %
	OpenPosition      *Position     `json:"open_position"`
	ClosedTrades      []TradeRecord `json:"closed_trades"`
	EquityCurve       []EquityPoint `json:"equity_curve"`
	TradeCount        int           `json:"trade_count"`
	WinCount          int           `json:"win_count"`
	LossCount         int           `json:"loss_count"`
	BreakevenCount    int           `json:"breakeven_count"`
	TotalPnL          float64       `json:"total_pnl"`
	TotalPnLR         float64       `json:"total_pnl_r"` // Total R multiples
	GrossProfit       float64       `json:"gross_profit"`
	GrossLoss         float64       `json:"gross_loss"`
	LargestWin        float64       `json:"largest_win"`
	LargestLoss       float64       `json:"largest_loss"`
	ConsecutiveWins   int           `json:"consecutive_wins"`
	ConsecutiveLosses int           `json:"consecutive_losses"`
	CurrentStreak     int           `json:"current_streak"` // Positive = wins, negative = losses
	IsStopped         bool          `json:"is_stopped"`     // True if max drawdown reached
	StopReason        string        `json:"stop_reason"`
}

// CapitalSimulator manages capital simulation for backtesting.
// Thread-safe for concurrent access.
type CapitalSimulator struct {
	mu       sync.RWMutex
	state    *CapitalState
	config   CapitalConfig
	tradeSeq int // Trade ID sequence
}

// NewCapitalSimulator creates a new capital simulator.
func NewCapitalSimulator(config CapitalConfig) (*CapitalSimulator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &CapitalSimulator{
		config: config,
		state: &CapitalState{
			Config:          config,
			Balance:         config.StartingBalance,
			Equity:          config.StartingBalance,
			PeakEquity:      config.StartingBalance,
			MarginAvailable: config.StartingBalance * config.Leverage,
			ClosedTrades:    make([]TradeRecord, 0, 100),
			EquityCurve:     make([]EquityPoint, 0, 1000),
		},
		tradeSeq: 0,
	}, nil
}

// GetState returns a copy of the current capital state.
func (cs *CapitalSimulator) GetState() CapitalState {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Return copy to prevent external modification
	stateCopy := *cs.state
	if cs.state.OpenPosition != nil {
		posCopy := *cs.state.OpenPosition
		stateCopy.OpenPosition = &posCopy
	}
	// Note: ClosedTrades and EquityCurve are slices (reference types)
	// For full isolation, would need deep copy
	return stateCopy
}

// GetBalance returns current account balance.
func (cs *CapitalSimulator) GetBalance() float64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state.Balance
}

// GetEquity returns current equity (balance + unrealized P&L).
func (cs *CapitalSimulator) GetEquity() float64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state.Equity
}

// HasOpenPosition returns true if there's an open position.
func (cs *CapitalSimulator) HasOpenPosition() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state.OpenPosition != nil
}

// IsStopped returns true if simulation was stopped (e.g., max drawdown).
func (cs *CapitalSimulator) IsStopped() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state.IsStopped
}

// CalculatePositionSize determines lot size based on risk amount and stop loss distance.
// Returns (lotSize, riskAmount, error)
func (cs *CapitalSimulator) CalculatePositionSize(
	entryPrice float64,
	stopLoss float64,
	symbol string,
) (float64, float64, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.state.IsStopped {
		return 0, 0, errors.New("simulation stopped due to max drawdown")
	}

	// Risk amount = current balance * risk per trade
	riskAmount := cs.state.Balance * cs.config.RiskPerTrade

	// Calculate price risk in pips
	priceRisk := math.Abs(entryPrice - stopLoss)
	if priceRisk < 1e-10 {
		return 0, 0, errors.New("invalid stop loss: price risk is zero")
	}

	// Get pip size for symbol
	pipSize := getPipSizeForSymbol(symbol)
	pipRisk := priceRisk / pipSize

	// Value per pip for 0.01 lot (micro lot)
	// Standard: 1 lot = 100,000 units, 1 pip = $10
	// Micro: 0.01 lot = 1,000 units, 1 pip = $0.10
	valuePerPipMicro := 0.10

	// Calculate lot size to risk exactly riskAmount
	// riskAmount = lotSize * 100 * pipRisk * valuePerPipMicro
	lotSize := riskAmount / (pipRisk * valuePerPipMicro * 100)

	// Round to lot step
	lotSize = math.Floor(lotSize/cs.config.LotStep) * cs.config.LotStep

	// Check minimum lot size
	if lotSize < cs.config.MinLotSize {
		return 0, 0, fmt.Errorf("calculated lot size %.4f below minimum %.4f",
			lotSize, cs.config.MinLotSize)
	}

	// Check maximum lot size
	if lotSize > cs.config.MaxLotSize {
		lotSize = cs.config.MaxLotSize
	}

	// Check margin requirements
	marginRequired := cs.calculateMarginRequired(entryPrice, lotSize, symbol)
	maxMargin := cs.state.Balance * cs.config.MaxMarginPercent

	if marginRequired > maxMargin {
		// Reduce lot size to fit margin constraint
		maxLotSize := (maxMargin * cs.config.Leverage) / (entryPrice * 100000)
		lotSize = math.Floor(maxLotSize/cs.config.LotStep) * cs.config.LotStep

		if lotSize < cs.config.MinLotSize {
			return 0, 0, errors.New("insufficient margin for minimum lot size")
		}
	}

	// Recalculate actual risk amount with final lot size
	actualRisk := lotSize * 100 * pipRisk * valuePerPipMicro

	return lotSize, actualRisk, nil
}

// calculateMarginRequired calculates margin for a position.
func (cs *CapitalSimulator) calculateMarginRequired(price, lotSize float64, symbol string) float64 {
	contractSize := 100000.0 // Standard forex contract
	notionalValue := lotSize * contractSize * price
	return notionalValue / cs.config.Leverage
}

// OpenPosition opens a new trading position.
func (cs *CapitalSimulator) OpenPosition(
	symbol string,
	direction string,
	entryPrice float64,
	stopLoss float64,
	takeProfit float64,
	entryTime time.Time,
	entryBar int,
	patternType string,
	confidence float64,
) (*Position, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.state.IsStopped {
		return nil, errors.New("simulation stopped due to max drawdown")
	}

	if cs.state.OpenPosition != nil {
		return nil, errors.New("position already open")
	}

	// Validate direction and stop loss position
	if direction == "LONG" {
		if stopLoss >= entryPrice {
			return nil, errors.New("LONG stop loss must be below entry price")
		}
		if takeProfit <= entryPrice {
			return nil, errors.New("LONG take profit must be above entry price")
		}
	} else if direction == "SHORT" {
		if stopLoss <= entryPrice {
			return nil, errors.New("SHORT stop loss must be above entry price")
		}
		if takeProfit >= entryPrice {
			return nil, errors.New("SHORT take profit must be below entry price")
		}
	} else {
		return nil, errors.New("direction must be LONG or SHORT")
	}

	// Calculate position size (unlock temporarily for nested call)
	cs.mu.Unlock()
	lotSize, riskAmount, err := cs.CalculatePositionSize(entryPrice, stopLoss, symbol)
	cs.mu.Lock()

	if err != nil {
		return nil, fmt.Errorf("position sizing failed: %w", err)
	}

	// Calculate risk-reward ratio
	var risk, reward float64
	if direction == "SHORT" {
		risk = stopLoss - entryPrice
		reward = entryPrice - takeProfit
	} else {
		risk = entryPrice - stopLoss
		reward = takeProfit - entryPrice
	}

	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	// Calculate margin
	marginRequired := cs.calculateMarginRequired(entryPrice, lotSize, symbol)

	cs.tradeSeq++
	position := &Position{
		ID:          cs.tradeSeq,
		Symbol:      symbol,
		Direction:   direction,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		LotSize:     lotSize,
		RiskAmount:  riskAmount,
		RiskPercent: riskAmount / cs.state.Balance,
		MarginUsed:  marginRequired,
		EntryTime:   entryTime,
		EntryBar:    entryBar,
		PatternType: patternType,
		Confidence:  confidence,
		RiskReward:  riskReward,
	}

	cs.state.OpenPosition = position
	cs.state.MarginUsed = marginRequired
	cs.state.MarginAvailable = (cs.state.Balance * cs.config.Leverage) - marginRequired

	return position, nil
}

// UpdateEquity updates the equity based on current price.
func (cs *CapitalSimulator) UpdateEquity(currentPrice float64, barIndex int, timestamp time.Time) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	openPnL := 0.0

	if cs.state.OpenPosition != nil {
		pos := cs.state.OpenPosition
		pipSize := getPipSizeForSymbol(pos.Symbol)

		var pnl float64
		if pos.Direction == "LONG" {
			pnl = (currentPrice - pos.EntryPrice) / pipSize
		} else {
			pnl = (pos.EntryPrice - currentPrice) / pipSize
		}

		// Convert pip P&L to dollar P&L
		// 0.01 lot = $0.10 per pip
		openPnL = pnl * 0.10 * pos.LotSize * 100
		pos.UnrealizedPnL = openPnL
	}

	cs.state.Equity = cs.state.Balance + openPnL

	// Update peak and drawdown
	if cs.state.Equity > cs.state.PeakEquity {
		cs.state.PeakEquity = cs.state.Equity
	}

	drawdown := cs.state.PeakEquity - cs.state.Equity
	drawdownPct := 0.0
	if cs.state.PeakEquity > 0 {
		drawdownPct = drawdown / cs.state.PeakEquity
	}

	if drawdown > cs.state.MaxDrawdown {
		cs.state.MaxDrawdown = drawdown
		cs.state.MaxDrawdownPct = drawdownPct
	}

	// Check max drawdown stop
	if drawdownPct >= cs.config.MaxDrawdownPct {
		cs.state.IsStopped = true
		cs.state.StopReason = fmt.Sprintf("max drawdown %.2f%% reached", drawdownPct*100)
	}

	// Record equity point
	point := EquityPoint{
		Timestamp:   timestamp,
		BarIndex:    barIndex,
		Balance:     cs.state.Balance,
		Equity:      cs.state.Equity,
		Drawdown:    drawdown,
		DrawdownPct: drawdownPct,
		OpenPnL:     openPnL,
	}
	cs.state.EquityCurve = append(cs.state.EquityCurve, point)
}

// ClosePosition closes the current position and records the trade.
func (cs *CapitalSimulator) ClosePosition(
	exitPrice float64,
	exitTime time.Time,
	exitBar int,
	exitReason string,
) (*TradeRecord, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.state.OpenPosition == nil {
		return nil, errors.New("no position to close")
	}

	pos := cs.state.OpenPosition
	pipSize := getPipSizeForSymbol(pos.Symbol)

	// Calculate P&L
	var pnlPips float64
	if pos.Direction == "LONG" {
		pnlPips = (exitPrice - pos.EntryPrice) / pipSize
	} else {
		pnlPips = (pos.EntryPrice - exitPrice) / pipSize
	}

	// Dollar P&L: pip_pnl * $0.10 per pip per micro lot * (lot_size * 100)
	pnl := pnlPips * 0.10 * pos.LotSize * 100

	// R multiple
	pnlR := 0.0
	if pos.RiskAmount > 0 {
		pnlR = pnl / pos.RiskAmount
	}

	// Percentage return
	balanceBefore := cs.state.Balance
	pnlPct := pnl / balanceBefore

	// Update balance
	cs.state.Balance += pnl
	cs.state.Equity = cs.state.Balance
	cs.state.TotalPnL += pnl
	cs.state.TotalPnLR += pnlR

	// Determine result
	result := "BREAKEVEN"
	if exitReason == "TP_HIT" || pnlR >= 0.5 {
		result = "WIN"
		cs.state.WinCount++
		cs.state.GrossProfit += pnl
		if pnl > cs.state.LargestWin {
			cs.state.LargestWin = pnl
		}
		// Update streak
		if cs.state.CurrentStreak > 0 {
			cs.state.CurrentStreak++
		} else {
			cs.state.CurrentStreak = 1
		}
		if cs.state.CurrentStreak > cs.state.ConsecutiveWins {
			cs.state.ConsecutiveWins = cs.state.CurrentStreak
		}
	} else if exitReason == "SL_HIT" || pnlR <= -0.5 {
		result = "LOSS"
		cs.state.LossCount++
		cs.state.GrossLoss += math.Abs(pnl)
		if pnl < cs.state.LargestLoss {
			cs.state.LargestLoss = pnl
		}
		// Update streak
		if cs.state.CurrentStreak < 0 {
			cs.state.CurrentStreak--
		} else {
			cs.state.CurrentStreak = -1
		}
		if -cs.state.CurrentStreak > cs.state.ConsecutiveLosses {
			cs.state.ConsecutiveLosses = -cs.state.CurrentStreak
		}
	} else {
		cs.state.BreakevenCount++
	}

	cs.state.TradeCount++

	// Create trade record
	record := TradeRecord{
		ID:            pos.ID,
		Symbol:        pos.Symbol,
		Direction:     pos.Direction,
		EntryPrice:    pos.EntryPrice,
		ExitPrice:     exitPrice,
		StopLoss:      pos.StopLoss,
		TakeProfit:    pos.TakeProfit,
		LotSize:       pos.LotSize,
		RiskAmount:    pos.RiskAmount,
		RiskPercent:   pos.RiskPercent,
		PnL:           pnl,
		PnLPips:       pnlPips,
		PnLR:          pnlR,
		PnLPct:        pnlPct,
		MarginUsed:    pos.MarginUsed,
		EntryTime:     pos.EntryTime,
		ExitTime:      exitTime,
		EntryBar:      pos.EntryBar,
		ExitBar:       exitBar,
		BarsHeld:      exitBar - pos.EntryBar,
		PatternType:   pos.PatternType,
		Confidence:    pos.Confidence,
		ExitReason:    exitReason,
		Result:        result,
		BalanceBefore: balanceBefore,
		BalanceAfter:  cs.state.Balance,
	}

	cs.state.ClosedTrades = append(cs.state.ClosedTrades, record)
	cs.state.OpenPosition = nil
	cs.state.MarginUsed = 0
	cs.state.MarginAvailable = cs.state.Balance * cs.config.Leverage

	// Update peak after trade
	if cs.state.Balance > cs.state.PeakEquity {
		cs.state.PeakEquity = cs.state.Balance
	}

	return &record, nil
}

// GetMetrics returns calculated trading metrics.
func (cs *CapitalSimulator) GetMetrics() TradingMetrics {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	s := cs.state

	metrics := TradingMetrics{
		StartingBalance:   cs.config.StartingBalance,
		EndingBalance:     s.Balance,
		TotalPnL:          s.TotalPnL,
		TotalPnLPct:       (s.Balance - cs.config.StartingBalance) / cs.config.StartingBalance * 100,
		TotalPnLR:         s.TotalPnLR,
		TradeCount:        s.TradeCount,
		WinCount:          s.WinCount,
		LossCount:         s.LossCount,
		BreakevenCount:    s.BreakevenCount,
		GrossProfit:       s.GrossProfit,
		GrossLoss:         s.GrossLoss,
		LargestWin:        s.LargestWin,
		LargestLoss:       s.LargestLoss,
		MaxDrawdown:       s.MaxDrawdown,
		MaxDrawdownPct:    s.MaxDrawdownPct * 100,
		ConsecutiveWins:   s.ConsecutiveWins,
		ConsecutiveLosses: s.ConsecutiveLosses,
	}

	if s.TradeCount > 0 {
		metrics.WinRate = float64(s.WinCount) / float64(s.TradeCount) * 100
		metrics.Expectancy = s.TotalPnL / float64(s.TradeCount)
		metrics.ExpectancyR = s.TotalPnLR / float64(s.TradeCount)
	}

	if s.GrossLoss > 0 {
		metrics.ProfitFactor = s.GrossProfit / s.GrossLoss
	}

	// Calculate average win/loss
	if s.WinCount > 0 {
		metrics.AvgWin = s.GrossProfit / float64(s.WinCount)
	}
	if s.LossCount > 0 {
		metrics.AvgLoss = s.GrossLoss / float64(s.LossCount)
	}

	// Calculate average R
	if s.TradeCount > 0 {
		var totalWinR, totalLossR float64
		for _, t := range s.ClosedTrades {
			if t.Result == "WIN" {
				totalWinR += t.PnLR
			} else if t.Result == "LOSS" {
				totalLossR += math.Abs(t.PnLR)
			}
		}
		if s.WinCount > 0 {
			metrics.AvgWinR = totalWinR / float64(s.WinCount)
		}
		if s.LossCount > 0 {
			metrics.AvgLossR = totalLossR / float64(s.LossCount)
		}
	}

	return metrics
}

// TradingMetrics contains calculated performance statistics.
type TradingMetrics struct {
	StartingBalance   float64 `json:"starting_balance"`
	EndingBalance     float64 `json:"ending_balance"`
	TotalPnL          float64 `json:"total_pnl"`
	TotalPnLPct       float64 `json:"total_pnl_pct"`
	TotalPnLR         float64 `json:"total_pnl_r"`
	TradeCount        int     `json:"trade_count"`
	WinCount          int     `json:"win_count"`
	LossCount         int     `json:"loss_count"`
	BreakevenCount    int     `json:"breakeven_count"`
	WinRate           float64 `json:"win_rate"`
	GrossProfit       float64 `json:"gross_profit"`
	GrossLoss         float64 `json:"gross_loss"`
	AvgWin            float64 `json:"avg_win"`
	AvgLoss           float64 `json:"avg_loss"`
	AvgWinR           float64 `json:"avg_win_r"`
	AvgLossR          float64 `json:"avg_loss_r"`
	LargestWin        float64 `json:"largest_win"`
	LargestLoss       float64 `json:"largest_loss"`
	ProfitFactor      float64 `json:"profit_factor"`
	Expectancy        float64 `json:"expectancy"`
	ExpectancyR       float64 `json:"expectancy_r"`
	MaxDrawdown       float64 `json:"max_drawdown"`
	MaxDrawdownPct    float64 `json:"max_drawdown_pct"`
	ConsecutiveWins   int     `json:"consecutive_wins"`
	ConsecutiveLosses int     `json:"consecutive_losses"`
}

// Reset resets the simulator to initial state.
func (cs *CapitalSimulator) Reset() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.state = &CapitalState{
		Config:          cs.config,
		Balance:         cs.config.StartingBalance,
		Equity:          cs.config.StartingBalance,
		PeakEquity:      cs.config.StartingBalance,
		MarginAvailable: cs.config.StartingBalance * cs.config.Leverage,
		ClosedTrades:    make([]TradeRecord, 0, 100),
		EquityCurve:     make([]EquityPoint, 0, 1000),
	}
	cs.tradeSeq = 0
}

// Helper function to get pip size (bridge to constants package)
func getPipSizeForSymbol(symbol string) float64 {
	cfg, err := constants.GetSymbolConfig(symbol)
	if err != nil {
		// Default for unknown symbols
		return 0.0001
	}
	return cfg.PipSize
}
