package simulation

import (
	"time"

	"set-and-trend/backend/internal/patterns"
)

// SimulationMode defines how the simulation runs
type SimulationMode string

const (
	ModeStep SimulationMode = "step" // Press ENTER for each bar
	ModeAuto SimulationMode = "auto" // Runs with configurable delay
	ModeFast SimulationMode = "fast" // Runs as fast as possible
)

// SimulationConfig holds all configuration for the simulation
type SimulationConfig struct {
	Mode                SimulationMode
	StartDate           time.Time
	EndDate             time.Time
	ConfidenceThreshold float64
	MinRR               float64
	MaxBarsToHold       int
	Lookback            int
	CooldownBars        int
	AutoDelayMs         int // Delay between bars in auto mode
	Symbol              string
	Timeframe           patterns.Timeframe
	ShowChart           bool
}

// DefaultSimulationConfig returns sensible defaults
func DefaultSimulationConfig() SimulationConfig {
	return SimulationConfig{
		Mode:                ModeStep,
		ConfidenceThreshold: 0.55,
		MinRR:               1.5,
		MaxBarsToHold:       20,
		Lookback:            3,
		CooldownBars:        10,
		AutoDelayMs:         500,
		Symbol:              "EURUSD",
		Timeframe:           patterns.TF_W1,
		ShowChart:           true,
	}
}

// Position represents an open trading position
type Position struct {
	ID          int
	Direction   string  // "LONG" or "SHORT"
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	EntryBar    int
	EntryTime   time.Time
	PatternType string
	Confidence  float64
	RiskReward  float64
}

// TradeResult represents a closed trade outcome
type TradeResult struct {
	Position
	ExitPrice   float64
	ExitBar     int
	ExitTime    time.Time
	PnL         float64  // Absolute P&L
	PnLPips     float64
	PnLR        float64  // P&L in R multiples
	Outcome     string   // "WIN", "LOSS", "TIMEOUT"
	ExitReason string  // ADD THIS: SL_HIT, TP_HIT, TIMEOUT
	BarsHeld    int
}

// SimulationState tracks current simulation progress
type SimulationState struct {
	CurrentBar      int
	TotalBars       int
	OpenPosition    *Position
	ClosedTrades    []TradeResult
	AccountBalanceR float64
	MaxDrawdownR    float64
	PeakBalanceR    float64
	LastTradeBar    map[string]int // Track last trade bar for each pattern type
}

// NewSimulationState creates initial state
func NewSimulationState(totalBars int) *SimulationState {
	return &SimulationState{
		CurrentBar:      0,
		TotalBars:       totalBars,
		ClosedTrades:    make([]TradeResult, 0),
		AccountBalanceR: 0.0,
		MaxDrawdownR:    0.0,
		PeakBalanceR:    0.0,
		LastTradeBar:    make(map[string]int),
	}
}

// SimulationStats calculates running statistics
type SimulationStats struct {
	TotalTrades      int
	WinningTrades    int
	LosingTrades     int
	WinRate          float64
	TotalPnLR        float64
	MaxDrawdownR     float64
	Expectancy       float64
	AvgBarsHeld      float64
	HSWins           int
	HSLosses         int
	IHSWins          int
	IHSLosses        int
	ConsecutiveWins  int
	ConsecutiveLosses int
	CurrentStreak    int // Positive = wins, negative = losses
}
