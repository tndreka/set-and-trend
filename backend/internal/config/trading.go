// Package config provides centralized configuration for the trading system.
//
// CRITICAL: This module centralizes ALL magic numbers and configuration values
// to ensure consistency across the codebase and enable parameter tuning.
package config

import (
	"errors"
	"fmt"

	"set-and-trend/backend/internal/execution"
)

// TradingConfig holds all trading system configuration.
// This is the single source of truth for all trading parameters.
type TradingConfig struct {
	// Symbol configuration
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"` // Primary timeframe for trading

	// Capital management
	Capital execution.CapitalConfig `json:"capital"`

	// Trade frequency control
	Frequency execution.FrequencyConfig `json:"frequency"`

	// Pattern detection
	Pattern PatternConfig `json:"pattern"`

	// Backtesting
	Backtest BacktestConfig `json:"backtest"`

	// Logging
	Logging LoggingConfig `json:"logging"`
}

// PatternConfig holds pattern detection parameters.
type PatternConfig struct {
	// Window and lookback
	WindowSize int `json:"window_size"` // Number of bars to analyze (default: 30)
	Lookback   int `json:"lookback"`    // Swing detection lookback (default: 3)

	// Confidence thresholds
	MinConfidence float64 `json:"min_confidence"` // Minimum confidence (default: 0.55)

	// Risk-reward requirements
	MinRiskReward float64 `json:"min_risk_reward"` // Minimum R:R (default: 1.5)
	MaxRiskReward float64 `json:"max_risk_reward"` // Maximum R:R - suspiciously good (default: 5.0)

	// Pattern geometry
	MinShoulderSymmetry  float64 `json:"min_shoulder_symmetry"`   // Min symmetry (default: 0.80)
	MinHeadProminence    float64 `json:"min_head_prominence"`     // Min head height % (default: 0.02)
	MinPatternHeightPips float64 `json:"min_pattern_height_pips"` // Min pattern height in pips (default: 50)

	// Entry/exit buffers (in pips)
	EntryBufferPips    float64 `json:"entry_buffer_pips"`     // Entry offset from neckline (default: 10)
	StopLossBufferPips float64 `json:"stop_loss_buffer_pips"` // SL buffer above/below shoulder (default: 20)

	// Cooldown between trades
	CooldownBars int `json:"cooldown_bars"` // Minimum bars between trades (default: 10)
}

// BacktestConfig holds backtesting parameters.
type BacktestConfig struct {
	// Exit configuration
	MaxBarsToExit int `json:"max_bars_to_exit"` // Maximum bars to hold trade (default: 20)

	// Execution costs (all in pips)
	SpreadPips     float64 `json:"spread_pips"`      // Typical spread (default: 0.2 for EURUSD)
	SlippagePips   float64 `json:"slippage_pips"`    // SL slippage (default: 0.3)
	TPSlippagePips float64 `json:"tp_slippage_pips"` // TP slippage (default: 0.2)
	CommissionPips float64 `json:"commission_pips"`  // Commission if applicable (default: 0.0)

	// Simulation mode
	UseRealTimeSimulation bool `json:"use_realtime_simulation"` // Use streaming feed
	SimulationSpeedMs     int  `json:"simulation_speed_ms"`     // Ms per bar in auto mode
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Enabled       bool   `json:"enabled"`
	CandleLogPath string `json:"candle_log_path"`
	TradeLogPath  string `json:"trade_log_path"`
	EngineLogPath string `json:"engine_log_path"`
	Append        bool   `json:"append"` // Append to existing logs
}

// DefaultTradingConfig returns sensible defaults for production trading.
func DefaultTradingConfig() TradingConfig {
	return TradingConfig{
		Symbol:    "EURUSD",
		Timeframe: "H4",

		Capital:   execution.DefaultCapitalConfig(),
		Frequency: execution.DefaultFrequencyConfig(),

		Pattern: PatternConfig{
			WindowSize:           30,
			Lookback:             3,
			MinConfidence:        0.55,
			MinRiskReward:        1.5,
			MaxRiskReward:        5.0,
			MinShoulderSymmetry:  0.80,
			MinHeadProminence:    0.02,
			MinPatternHeightPips: 50,
			EntryBufferPips:      10,
			StopLossBufferPips:   20,
			CooldownBars:         10,
		},

		Backtest: BacktestConfig{
			MaxBarsToExit:         20,
			SpreadPips:            0.2,
			SlippagePips:          0.3,
			TPSlippagePips:        0.2,
			CommissionPips:        0.0,
			UseRealTimeSimulation: false,
			SimulationSpeedMs:     500,
		},

		Logging: LoggingConfig{
			Enabled:       true,
			CandleLogPath: "logs/candles.jsonl",
			TradeLogPath:  "logs/trades.jsonl",
			EngineLogPath: "logs/engine.jsonl",
			Append:        false,
		},
	}
}

// Validate checks all configuration values for validity.
func (c TradingConfig) Validate() error {
	if c.Symbol == "" {
		return errors.New("symbol is required")
	}
	if c.Timeframe == "" {
		return errors.New("timeframe is required")
	}

	// Validate capital config
	if err := c.Capital.Validate(); err != nil {
		return fmt.Errorf("capital config: %w", err)
	}

	// Validate frequency config
	if err := c.Frequency.Validate(); err != nil {
		return fmt.Errorf("frequency config: %w", err)
	}

	// Validate pattern config
	if err := c.Pattern.Validate(); err != nil {
		return fmt.Errorf("pattern config: %w", err)
	}

	// Validate backtest config
	if err := c.Backtest.Validate(); err != nil {
		return fmt.Errorf("backtest config: %w", err)
	}

	return nil
}

// Validate checks pattern configuration.
func (p PatternConfig) Validate() error {
	if p.WindowSize < 10 {
		return errors.New("window size must be >= 10")
	}
	if p.Lookback < 1 {
		return errors.New("lookback must be >= 1")
	}
	if p.MinConfidence < 0 || p.MinConfidence > 1 {
		return errors.New("min confidence must be between 0 and 1")
	}
	if p.MinRiskReward <= 0 {
		return errors.New("min risk-reward must be positive")
	}
	if p.MaxRiskReward <= p.MinRiskReward {
		return errors.New("max risk-reward must be > min risk-reward")
	}
	if p.MinShoulderSymmetry < 0 || p.MinShoulderSymmetry > 1 {
		return errors.New("min shoulder symmetry must be between 0 and 1")
	}
	if p.MinHeadProminence < 0 || p.MinHeadProminence > 1 {
		return errors.New("min head prominence must be between 0 and 1")
	}
	if p.MinPatternHeightPips <= 0 {
		return errors.New("min pattern height must be positive")
	}
	if p.EntryBufferPips < 0 {
		return errors.New("entry buffer cannot be negative")
	}
	if p.StopLossBufferPips < 0 {
		return errors.New("stop loss buffer cannot be negative")
	}
	if p.CooldownBars < 0 {
		return errors.New("cooldown bars cannot be negative")
	}
	return nil
}

// Validate checks backtest configuration.
func (b BacktestConfig) Validate() error {
	if b.MaxBarsToExit < 1 {
		return errors.New("max bars to exit must be >= 1")
	}
	if b.SpreadPips < 0 {
		return errors.New("spread pips cannot be negative")
	}
	if b.SlippagePips < 0 {
		return errors.New("slippage pips cannot be negative")
	}
	if b.TPSlippagePips < 0 {
		return errors.New("TP slippage pips cannot be negative")
	}
	if b.CommissionPips < 0 {
		return errors.New("commission pips cannot be negative")
	}
	return nil
}

// ProductionConfig returns a strict configuration for live trading.
// More conservative than defaults.
func ProductionConfig() TradingConfig {
	config := DefaultTradingConfig()

	// Stricter confidence requirements
	config.Pattern.MinConfidence = 0.60

	// Higher R:R requirement
	config.Pattern.MinRiskReward = 1.8

	// Longer cooldown
	config.Pattern.CooldownBars = 15

	// Fewer trades per month
	config.Frequency.MaxTradesPerMonth = 10
	config.Frequency.HighQualityThreshold = 0.75
	config.Frequency.FinalQualityThreshold = 0.85

	// Lower risk per trade
	config.Capital.RiskPerTrade = 0.005 // 0.5% instead of 1%

	return config
}

// BacktestOptimizationConfig returns a config suitable for parameter optimization.
func BacktestOptimizationConfig() TradingConfig {
	config := DefaultTradingConfig()

	// Disable logging during optimization for speed
	config.Logging.Enabled = false

	// Faster simulation
	config.Backtest.SimulationSpeedMs = 0

	return config
}
