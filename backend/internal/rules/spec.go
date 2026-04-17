package rules

import "time"

// Timeframe represents a trading timeframe
type Timeframe string

const (
	W1 Timeframe = "W1"
	D1 Timeframe = "D1"
	H4 Timeframe = "H4"
)

// RuleCode uniquely identifies a rule
type RuleCode string

const (
	W1TrendBullish    RuleCode = "W1_TREND_BULLISH"
	H4TrendFollowing  RuleCode = "H4_TREND_FOLLOWING"
	H4SupplyDemand    RuleCode = "H4_SUPPLY_DEMAND"
	H4LondonBreakout  RuleCode = "H4_LONDON_BREAKOUT"
	SAFChecklist            RuleCode = "SAF_CHECKLIST"
	SAFW1EmaRejection       RuleCode = "SAF_W1_EMA_REJECTION"
	SAFD1RejectionStructure RuleCode = "SAF_D1_REJECTION_STRUCTURE"
	SAFW1RejectionD1AOI     RuleCode = "SAF_W1_REJECTION_D1_AOI"
)

// RuleSpec defines an immutable rule specification
type RuleSpec struct {
	Code        RuleCode
	Name        string
	Description string
	Timeframe   Timeframe // ✅ FIXED: Now typed, not string
	Conditions  []ConditionCode
}

// ConditionCode identifies a condition
type ConditionCode string

const (
	EMA50GtEMA200      ConditionCode = "ema50_gt_ema200"
	CloseGtEMA50       ConditionCode = "close_gt_ema50"
	EMA50SlopePositive ConditionCode = "ema50_slope_positive"
)

// RuleRegistry is the immutable registry of all rules
var RuleRegistry = map[RuleCode]RuleSpec{
	W1TrendBullish: {
		Code:        W1TrendBullish,
		Name:        "Weekly Trend Bullish",
		Description: "Weekly bullish trend confirmation: EMA50 > EMA200, Close > EMA50, EMA50 rising",
		Timeframe:   W1,
		Conditions: []ConditionCode{
			EMA50GtEMA200,
			CloseGtEMA50,
			EMA50SlopePositive,
		},
	},
	H4TrendFollowing: {
		Code:        H4TrendFollowing,
		Name:        "H4 Trend Following",
		Description: "H4 bullish trend: EMA50 > EMA200, Close > EMA50, EMA50 rising. Same conditions as W1_TREND_BULLISH evaluated against H4 indicators.",
		Timeframe:   H4,
		Conditions: []ConditionCode{
			EMA50GtEMA200,
			CloseGtEMA50,
			EMA50SlopePositive,
		},
	},
}

// RuleResult represents the outcome of rule evaluation
type RuleResult struct {
	RuleCode       RuleCode
	Result         string  // "PASS" or "FAIL"
	Confidence     float64 // 0.0 to 1.0
	ConditionsMet  []ConditionCode
	ConditionsFail []ConditionCode
	// ✅ FIXED: Removed Explanation - frontend can build it from structured data
}

// Candle represents candle data for rule evaluation
type Candle struct {
	Open         float64
	High         float64
	Low          float64
	Close        float64
	TimestampUTC time.Time // ✅ FIXED: Added for session derivation
}

// Indicators represents indicator data for rule evaluation
type Indicators struct {
	EMA20      float64
	EMA50      float64
	EMA200     float64
	EMA50Prev  *float64 // ✅ FIXED: Pointer to distinguish nil from 0
	RangeSize  float64
	BodySize   float64
	UpperWick  float64
	LowerWick  float64
	MidPrice   float64
	// HTF trend indicators — populated by synthesis.BuildFullIndicatorSeries.
	// Zero = not yet warmed up (bars before EMA period).
	D1EMA50    float64
	D1EMA200   float64
	W1EMA50    float64
	W1EMA200   float64
}
