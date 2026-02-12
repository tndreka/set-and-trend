package selector

import (
	"fmt"
	"math"
	"sort"
	"time"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/engine"
)

type WeeklySelector struct {
	weekKey  string
	max      int
	signals  []*engine.BacktestSignal
	selected []*engine.BacktestSignal
}

func New(weekKey string, maxTrades int) *WeeklySelector {
	return &WeeklySelector{
		weekKey: weekKey, 
		max:     maxTrades,
		signals: make([]*engine.BacktestSignal, 0),
	}
}

func (s *WeeklySelector) Add(sig *engine.BacktestSignal) {
	s.signals = append(s.signals, sig)
}

func (s *WeeklySelector) Top(n int) []*engine.BacktestSignal {
	if len(s.signals) == 0 {
		return []*engine.BacktestSignal{}
	}
	
	// Sort by score descending
	sort.Slice(s.signals, func(i, j int) bool {
		return s.score(s.signals[i]) > s.score(s.signals[j])
	})
	
	num := min(n, len(s.signals))
	s.selected = s.signals[:num]
	return s.selected
}

func (s *WeeklySelector) score(sig *engine.BacktestSignal) float64 {
	score := sig.Confidence * 0.60
	score += math.Max(0, sig.RiskReward-1.5) * 0.25
	score += s.timeBonus(sig.Timestamp) * 0.10
	score += s.volatilityBonus(sig) * 0.05
	return score
}

func (s *WeeklySelector) timeBonus(t time.Time) float64 {
	hour := t.Hour()
	// London session (8-12 UTC) or New York session (13-17 UTC)
	if (hour >= 8 && hour <= 12) || (hour >= 13 && hour <= 17) {
		return 1.0
	}
	return 0.8
}

func (s *WeeklySelector) volatilityBonus(sig *engine.BacktestSignal) float64 {
	// ✅ FIXED: Use symbol-specific pip size instead of hardcoded 0.0001
	symbolConfig := constants.MustGetSymbolConfig(sig.Symbol)
	pipSize := symbolConfig.PipSize
	
	// Convert ATR to pips
	atrPips := sig.ATR / pipSize
	
	// Ideal volatility range: 15-80 pips
	if atrPips > 15 && atrPips < 80 {
		return 1.0
	}
	return 0.7
}

func (s *WeeklySelector) Print() {
	if len(s.signals) == 0 {
		fmt.Printf("\n📊 WEEK %s: No signals\n", s.weekKey)
		return
	}
	
	fmt.Printf("\n📊 WEEK %s: %d signals → TOP %d\n", 
		s.weekKey, len(s.signals), len(s.selected))
	
	for i, sig := range s.selected {
		fmt.Printf("  🥇 #%d %.0f%% | %s %s@%.5f | R:R=%.1f | ATR=%.0fpips\n",
			i+1, s.score(sig)*100, sig.Symbol, sig.Direction, 
			sig.EntryPrice, sig.RiskReward, sig.ATR/constants.MustGetSymbolConfig(sig.Symbol).PipSize)
	}
}

func min(a, b int) int {
	if a < b { 
		return a 
	}
	return b
}