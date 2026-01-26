package database

import "time"

// Candle represents OHLCV data for a single time period
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Timeframe represents available data timeframes
type Timeframe string

const (
	TimeframeM1     Timeframe = "m1"
	TimeframeM5     Timeframe = "m5"
	TimeframeM15    Timeframe = "m15"
	TimeframeM30    Timeframe = "m30"
	TimeframeH1     Timeframe = "hourly"
	TimeframeH4     Timeframe = "h4"
	TimeframeDaily  Timeframe = "daily"
)

// TableName returns the database table name for a timeframe
func (t Timeframe) TableName() string {
	switch t {
	case TimeframeM1:
		return "m1_prices"
	case TimeframeM5:
		return "m5_prices"
	case TimeframeM15:
		return "m15_prices"
	case TimeframeM30:
		return "m30_prices"
	case TimeframeH1:
		return "hourly_prices"
	case TimeframeH4:
		return "h4_prices"
	case TimeframeDaily:
		return "daily_prices"
	default:
		return "h4_prices"
	}
}

// Symbol metadata
type SymbolInfo struct {
	Symbol           string
	Description      string
	Category         SymbolCategory
	PipValue         float64
	TypicalSpread    float64
	CorrelationGroup int
}

type SymbolCategory string

const (
	CategoryMajorForex SymbolCategory = "major_forex"
	CategoryCrossPair  SymbolCategory = "cross_pair"
	CategoryCommodity  SymbolCategory = "commodity"
	CategoryIndex      SymbolCategory = "index"
)

// DateRange for filtering data
type DateRange struct {
	Start time.Time
	End   time.Time
}
