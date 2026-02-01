package confluence

import (
	"time"

	"set-and-trend/backend/internal/patterns"
)

// ChecklistScore holds all confluence components
type ChecklistScore struct {
	// Weekly (30.6% normalized | 55% raw)
	WeeklyFavor    float64 // 5.6%
	WeeklyAOI      float64 // 5.6%
	WeeklyEMA      float64 // 2.8%
	WeeklyReject   float64 // 5.6%
	WeeklyStruct   float64 // 5.6%
	WeeklyPattern  float64 // 5.6%

	// Daily (30.6% normalized | 55% raw)
	DailyFavor     float64 // 5.6%
	DailyAOI       float64 // 5.6%
	DailyEMA       float64 // 2.8%
	DailyReject    float64 // 5.6%
	DailyStruct    float64 // 5.6%
	DailyPattern   float64 // 5.6%

	// 4H (25% normalized | 45% raw)
	H4Favor        float64 // 5.6%
	H4EMA          float64 // 2.8%
	H4Reject       float64 // 5.6%
	H4Struct       float64 // 5.6%
	H4Pattern      float64 // 5.6%

	// Lower TFs (13.9% normalized | 25% raw)
	H2Psych        float64 // 2.8%
	H1SOS          float64 // 5.6%
	M30Engulf      float64 // 5.6%

	// Final score
	TotalPercent   float64 // 0-100%
}

// ConfluenceStream handles real-time confluence updates
type ConfluenceStream struct {
	Scorer        *ConfluenceScorer
	CurrentScore  *ChecklistScore
	LastUpdate    time.Time
	Symbol        string
}

// ConfluenceUpdate is sent when score changes significantly
type ConfluenceUpdate struct {
	Score       *ChecklistScore
	Signal      *patterns.TradeSignal
	Timestamp   time.Time
	IsValid     bool // Above 70% threshold
}

// Timeframe represents supported timeframes for confluence
type Timeframe string

const (
	TF_W1  Timeframe = "W1"
	TF_D1  Timeframe = "D1"
	TF_H4  Timeframe = "H4"
	TF_H2  Timeframe = "H2"
	TF_H1  Timeframe = "H1"
	TF_M30 Timeframe = "M30"
)

// ConfluenceConfig holds scoring thresholds
type ConfluenceConfig struct {
	MinConfluencePercent float64 // Default: 70%
}

// DefaultConfluenceConfig returns default config
func DefaultConfluenceConfig() ConfluenceConfig {
	return ConfluenceConfig{
		MinConfluencePercent: 70.0,
	}
}