// Package execution provides trade frequency control to enforce sniper trading discipline.
//
// CRITICAL: This module enforces 8-12 trades per month maximum as per specification.
// When approaching limits, quality thresholds dynamically increase.
package execution

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// FrequencyConfig controls trade frequency limits.
type FrequencyConfig struct {
	MinTradesPerMonth     int     // Minimum trades expected (e.g., 8)
	MaxTradesPerMonth     int     // Maximum trades allowed (e.g., 12)
	BaseQualityThreshold  float64 // Minimum confidence normally (e.g., 0.55)
	HighQualityThreshold  float64 // Confidence when approaching limit (e.g., 0.75)
	FinalQualityThreshold float64 // Confidence for last allowed trade (e.g., 0.80)
	RejectBelowThreshold  bool    // Whether to reject or just log warnings
}

// DefaultFrequencyConfig returns sensible defaults for sniper trading.
func DefaultFrequencyConfig() FrequencyConfig {
	return FrequencyConfig{
		MinTradesPerMonth:     8,
		MaxTradesPerMonth:     12,
		BaseQualityThreshold:  0.55,
		HighQualityThreshold:  0.70, // Raised from 0.55 when 3 trades remaining
		FinalQualityThreshold: 0.80, // Required for last 1-2 trades
		RejectBelowThreshold:  true,
	}
}

// Validate checks all config values.
func (c FrequencyConfig) Validate() error {
	if c.MinTradesPerMonth <= 0 {
		return errors.New("min trades per month must be positive")
	}
	if c.MaxTradesPerMonth <= 0 {
		return errors.New("max trades per month must be positive")
	}
	if c.MaxTradesPerMonth < c.MinTradesPerMonth {
		return errors.New("max trades must be >= min trades")
	}
	if c.BaseQualityThreshold < 0 || c.BaseQualityThreshold > 1 {
		return errors.New("base quality threshold must be 0-1")
	}
	if c.HighQualityThreshold < c.BaseQualityThreshold {
		return errors.New("high quality threshold must be >= base threshold")
	}
	if c.FinalQualityThreshold < c.HighQualityThreshold {
		return errors.New("final quality threshold must be >= high threshold")
	}
	return nil
}

// TradeDecision represents the outcome of a frequency check.
type TradeDecision struct {
	Allowed         bool    `json:"allowed"`
	Reason          string  `json:"reason"`
	TradesThisMonth int     `json:"trades_this_month"`
	TradesRemaining int     `json:"trades_remaining"`
	RequiredScore   float64 `json:"required_score"`
	ActualScore     float64 `json:"actual_score"`
}

// MonthlyStats tracks monthly trading statistics.
type MonthlyStats struct {
	Month              int            // 1-12
	Year               int            // e.g., 2025
	TradeCount         int            // Total trades this month
	AcceptedCount      int            // Signals accepted
	RejectedCount      int            // Signals rejected
	WinCount           int            // Winning trades
	LossCount          int            // Losing trades
	TotalPnL           float64        // Total P&L this month
	RejectionsByReason map[string]int // Count by rejection reason
}

// MonthlyTracker tracks trades per month with quality filtering.
// Thread-safe for concurrent access.
type MonthlyTracker struct {
	config       FrequencyConfig
	mu           sync.RWMutex
	currentMonth int
	currentYear  int
	stats        MonthlyStats
	history      []MonthlyStats // Historical monthly stats
}

// NewMonthlyTracker creates a new monthly trade tracker.
func NewMonthlyTracker(config FrequencyConfig) (*MonthlyTracker, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	now := time.Now().UTC()
	return &MonthlyTracker{
		config:       config,
		currentMonth: int(now.Month()),
		currentYear:  now.Year(),
		stats: MonthlyStats{
			Month:              int(now.Month()),
			Year:               now.Year(),
			RejectionsByReason: make(map[string]int),
		},
		history: make([]MonthlyStats, 0, 12),
	}, nil
}

// checkAndResetMonth resets counters if we're in a new month.
// Must be called with lock held.
func (mt *MonthlyTracker) checkAndResetMonth(t time.Time) {
	month := int(t.Month())
	year := t.Year()

	if month != mt.currentMonth || year != mt.currentYear {
		// Archive current stats
		if mt.stats.TradeCount > 0 || mt.stats.RejectedCount > 0 {
			mt.history = append(mt.history, mt.stats)
		}

		// Reset for new month
		mt.currentMonth = month
		mt.currentYear = year
		mt.stats = MonthlyStats{
			Month:              month,
			Year:               year,
			RejectionsByReason: make(map[string]int),
		}
	}
}

// CanTrade checks if a trade is allowed based on frequency limits and quality.
// Returns a decision with detailed reasoning.
func (mt *MonthlyTracker) CanTrade(confidence float64, timestamp time.Time) TradeDecision {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.checkAndResetMonth(timestamp)

	tradesRemaining := mt.config.MaxTradesPerMonth - mt.stats.TradeCount

	decision := TradeDecision{
		TradesThisMonth: mt.stats.TradeCount,
		TradesRemaining: tradesRemaining,
		ActualScore:     confidence,
	}

	// Check if we've hit the maximum
	if tradesRemaining <= 0 {
		decision.Allowed = false
		decision.Reason = "MAX_TRADES_REACHED"
		decision.RequiredScore = 1.0 // Impossible
		mt.stats.RejectedCount++
		mt.stats.RejectionsByReason["MAX_TRADES_REACHED"]++
		return decision
	}

	// Determine required quality threshold based on remaining trades
	var requiredThreshold float64
	var thresholdReason string

	switch {
	case tradesRemaining <= 1:
		// Last trade of the month - require exceptional quality
		requiredThreshold = mt.config.FinalQualityThreshold
		thresholdReason = "FINAL_TRADE_REQUIRES_HIGH_QUALITY"
	case tradesRemaining <= 2:
		// Second to last - require high quality
		requiredThreshold = (mt.config.HighQualityThreshold + mt.config.FinalQualityThreshold) / 2
		thresholdReason = "NEAR_LIMIT_ELEVATED_THRESHOLD"
	case tradesRemaining <= 3:
		// Getting close to limit - elevated threshold
		requiredThreshold = mt.config.HighQualityThreshold
		thresholdReason = "APPROACHING_LIMIT"
	default:
		// Plenty of trades remaining - base threshold
		requiredThreshold = mt.config.BaseQualityThreshold
		thresholdReason = "NORMAL_THRESHOLD"
	}

	decision.RequiredScore = requiredThreshold

	// Check if confidence meets threshold
	if confidence < requiredThreshold {
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("QUALITY_TOO_LOW_%s", thresholdReason)
		mt.stats.RejectedCount++
		mt.stats.RejectionsByReason[decision.Reason]++
		return decision
	}

	// Trade is allowed
	decision.Allowed = true
	decision.Reason = "APPROVED"

	return decision
}

// RecordTrade records a trade execution.
func (mt *MonthlyTracker) RecordTrade(timestamp time.Time, isWin bool, pnl float64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.checkAndResetMonth(timestamp)

	mt.stats.TradeCount++
	mt.stats.AcceptedCount++
	mt.stats.TotalPnL += pnl

	if isWin {
		mt.stats.WinCount++
	} else {
		mt.stats.LossCount++
	}
}

// RecordRejection records a rejected signal.
func (mt *MonthlyTracker) RecordRejection(timestamp time.Time, reason string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.checkAndResetMonth(timestamp)

	mt.stats.RejectedCount++
	mt.stats.RejectionsByReason[reason]++
}

// GetCurrentStats returns current month statistics.
func (mt *MonthlyTracker) GetCurrentStats() MonthlyStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Return a copy
	stats := mt.stats
	stats.RejectionsByReason = make(map[string]int)
	for k, v := range mt.stats.RejectionsByReason {
		stats.RejectionsByReason[k] = v
	}
	return stats
}

// GetHistory returns historical monthly statistics.
func (mt *MonthlyTracker) GetHistory() []MonthlyStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	history := make([]MonthlyStats, len(mt.history))
	copy(history, mt.history)
	return history
}

// GetRequiredThreshold returns the current required quality threshold.
func (mt *MonthlyTracker) GetRequiredThreshold(timestamp time.Time) float64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	tradesRemaining := mt.config.MaxTradesPerMonth - mt.stats.TradeCount

	switch {
	case tradesRemaining <= 1:
		return mt.config.FinalQualityThreshold
	case tradesRemaining <= 2:
		return (mt.config.HighQualityThreshold + mt.config.FinalQualityThreshold) / 2
	case tradesRemaining <= 3:
		return mt.config.HighQualityThreshold
	default:
		return mt.config.BaseQualityThreshold
	}
}

// Reset resets all statistics.
func (mt *MonthlyTracker) Reset() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	now := time.Now().UTC()
	mt.currentMonth = int(now.Month())
	mt.currentYear = now.Year()
	mt.stats = MonthlyStats{
		Month:              mt.currentMonth,
		Year:               mt.currentYear,
		RejectionsByReason: make(map[string]int),
	}
	mt.history = make([]MonthlyStats, 0, 12)
}

// Summary generates a summary string for logging.
func (mt *MonthlyTracker) Summary() string {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return fmt.Sprintf(
		"Month %d/%d: Trades=%d/%d (remaining=%d), Accepted=%d, Rejected=%d, Win=%d, Loss=%d, PnL=$%.2f",
		mt.stats.Month, mt.stats.Year,
		mt.stats.TradeCount, mt.config.MaxTradesPerMonth,
		mt.config.MaxTradesPerMonth-mt.stats.TradeCount,
		mt.stats.AcceptedCount, mt.stats.RejectedCount,
		mt.stats.WinCount, mt.stats.LossCount,
		mt.stats.TotalPnL,
	)
}

// IsAtRisk returns true if we're close to trade limit or underperforming.
func (mt *MonthlyTracker) IsAtRisk(timestamp time.Time) (bool, string) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Check overtrading
	remaining := mt.config.MaxTradesPerMonth - mt.stats.TradeCount
	if remaining <= 2 {
		return true, fmt.Sprintf("NEAR_LIMIT: only %d trades remaining", remaining)
	}

	// Check undertrading (based on day of month)
	dayOfMonth := timestamp.Day()
	if dayOfMonth >= 20 && mt.stats.TradeCount < mt.config.MinTradesPerMonth/2 {
		return true, fmt.Sprintf("UNDERTRADING: only %d trades with %d days remaining",
			mt.stats.TradeCount, 30-dayOfMonth)
	}

	return false, ""
}
