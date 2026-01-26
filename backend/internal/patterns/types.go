package patterns

import "time"

// Timeframe represents a trading timeframe
type Timeframe string

const (
	TF_W1 Timeframe = "W1"
	TF_D1 Timeframe = "D1"
	TF_H4 Timeframe = "H4"
	TF_H1 Timeframe = "H1"
)

// LookbackByTimeframe returns the recommended lookback for swing detection
func LookbackByTimeframe(tf Timeframe) int {
	switch tf {
	case TF_W1:
		return 3
	case TF_D1:
		return 5
	case TF_H4:
		return 8
	case TF_H1:
		return 12
	default:
		return 5
	}
}

// Candle represents OHLCV data
type Candle struct {
	Index     int
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
}

// SwingPoint represents a detected swing high or low
type SwingPoint struct {
	Index     int
	Price     float64
	IsPeak    bool    // true = swing high, false = swing low
	Strength  float64 // 0-1, how extreme vs neighbors
	Timestamp time.Time
}

// DetectedStructure contains all H&S metrics (objective, no vibes)
type DetectedStructure struct {
	// Price levels and indices
	LeftShoulderPrice float64
	LeftShoulderIdx   int
	HeadPrice         float64
	HeadIdx           int
	RightShoulderPrice float64
	RightShoulderIdx   int
	NecklinePrice     float64
	NecklineIdx       int

	// Trough points (for neckline)
	Trough1Price float64
	Trough1Idx   int
	Trough2Price float64
	Trough2Idx   int

	// Component scores (0.0-1.0)
	ShoulderSymmetry float64 // How similar are LS and RS?
	HeadProminence   float64 // How much taller is head?
	TimeSymmetry     float64 // Are peaks evenly spaced?
	VolumeProfile    float64 // Is volume declining?
	NecklineQuality  float64 // How flat/horizontal is neckline?

	// Summary
	OverallConfidence float64 // Geometric confidence before context
	FinalConfidence   float64 // After context adjustments
	PatternType       string  // "H&S", "IHS", "WEAK"
}

// MarketContext holds market state for context-based confidence
type MarketContext struct {
	Trend              string  // STRONG_UP, WEAK_UP, SIDEWAYS, WEAK_DOWN, STRONG_DOWN
	VolatilityRegime   string  // COMPRESSION, EXPANSION, NORMAL
	DistanceFromEMA200 float64 // Relative distance from 200 EMA
	RecentSwings       int     // Number of swings in last N bars
}

// TradeSignal represents entry/exit parameters
type TradeSignal struct {
	Symbol      string
	Direction   string  // "SHORT" for H&S, "LONG" for IHS
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	RiskReward  float64
	Confidence  float64
	PatternType string
}

// PatternType constants
const (
	PatternHS   = "H&S"
	PatternIHS  = "IHS"
	PatternWeak = "WEAK"
)

// Direction constants
const (
	DirectionShort = "SHORT"
	DirectionLong  = "LONG"
)

// Trend constants
const (
	TrendStrongUp   = "STRONG_UP"
	TrendWeakUp     = "WEAK_UP"
	TrendSideways   = "SIDEWAYS"
	TrendWeakDown   = "WEAK_DOWN"
	TrendStrongDown = "STRONG_DOWN"
)

// Volatility regime constants
const (
	VolatilityCompression = "COMPRESSION"
	VolatilityExpansion   = "EXPANSION"
	VolatilityNormal      = "NORMAL"
)
