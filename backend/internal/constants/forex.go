package constants

import (
	"fmt"
	"strings"
)

// DEPRECATED: Use domain.GetSymbolConfig() for per-symbol config
const (
	TimeframeW1          = "W1"
	MinStopLossPips      = 5
	MaxStopLossPips      = 500
	MinimumRR            = 1.5
	MaxEntrySlippagePips = 20.0

	// DefaultFXPipSize is the standard pip size for major FX pairs (non-JPY)
	// Used ONLY as fallback when symbol is unavailable - prefer MustGet(symbol).PipSize
	DefaultFXPipSize = 0.0001
)

// SymbolConfig defines trading parameters per symbol
type SymbolConfig struct {
	Symbol         string
	PipSize        float64
	PipValue       float64 // Value per pip per standard lot (in USD)
	SpreadPips     float64 // Typical spread for cost modeling
	PipLocation    int     // 4 or 2
	ContractSize   float64 // Units per lot
	MinLot         float64 // Minimum lot size
	LotStep        float64 // Lot size increment
	MarginRate     float64 // Margin % required (1.0 = 100%)
	PricePrecision int
	CurrencyBase   string
	CurrencyQuote  string
}

// Registry - EXPLICITLY DEFINED, NOT AUTO-DETECTED
var SymbolRegistry = map[string]SymbolConfig{
	// Major FX Pairs
	"EURUSD": {
		Symbol: "EURUSD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.2, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "EUR", CurrencyQuote: "USD",
	},
	"GBPUSD": {
		Symbol: "GBPUSD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.3, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "GBP", CurrencyQuote: "USD",
	},
	"USDJPY": {
		Symbol: "USDJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.2, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "USD", CurrencyQuote: "JPY",
	},
	"USDCHF": {
		Symbol: "USDCHF", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.3, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "USD", CurrencyQuote: "CHF",
	},
	"AUDUSD": {
		Symbol: "AUDUSD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.2, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "AUD", CurrencyQuote: "USD",
	},
	"NZDUSD": {
		Symbol: "NZDUSD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.3, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "NZD", CurrencyQuote: "USD",
	},
	"USDCAD": {
		Symbol: "USDCAD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.3, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "USD", CurrencyQuote: "CAD",
	},

	// Cross Pairs
	"AUDJPY": {
		Symbol: "AUDJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.5, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "AUD", CurrencyQuote: "JPY",
	},
	"CHFJPY": {
		Symbol: "CHFJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.5, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "CHF", CurrencyQuote: "JPY",
	},
	"EURAUD": {
		Symbol: "EURAUD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.5, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "EUR", CurrencyQuote: "AUD",
	},
	"EURCAD": {
		Symbol: "EURCAD", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.4, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "EUR", CurrencyQuote: "CAD",
	},
	"EURCHF": {
		Symbol: "EURCHF", PipSize: 0.0001, PipValue: 10.0, SpreadPips: 0.4, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "EUR", CurrencyQuote: "CHF",
	},
	"EURGBP": {
		Symbol: "EURGBP", PipSize: 0.0001, PipValue: 12.5, SpreadPips: 0.3, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "EUR", CurrencyQuote: "GBP",
	},
	"EURJPY": {
		Symbol: "EURJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.4, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "EUR", CurrencyQuote: "JPY",
	},
	"GBPJPY": {
		Symbol: "GBPJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.8, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "GBP", CurrencyQuote: "JPY",
	},

	// Metals
	"XAUUSD": {
		Symbol: "XAUUSD", PipSize: 1.0, PipValue: 10.0, SpreadPips: 2.0, PipLocation: 0,
		ContractSize: 100, MinLot: 0.01, LotStep: 0.01, MarginRate: 2.0,
		PricePrecision: 2, CurrencyBase: "XAU", CurrencyQuote: "USD",
	},
	"XAGUSD": {
		Symbol: "XAGUSD", PipSize: 0.1, PipValue: 5.0, SpreadPips: 2.0, PipLocation: 1,
		ContractSize: 5000, MinLot: 0.01, LotStep: 0.01, MarginRate: 2.0,
		PricePrecision: 3, CurrencyBase: "XAG", CurrencyQuote: "USD",
	},
	
	// Indices (CFD - price per point, not per pip)
	"USA30IDXUSD": {
		Symbol: "USA30IDXUSD", PipSize: 1.0, PipValue: 1.0, SpreadPips: 2.0, PipLocation: 0,
		ContractSize: 1, MinLot: 0.1, LotStep: 0.1, MarginRate: 5.0,
		PricePrecision: 1, CurrencyBase: "USA30", CurrencyQuote: "USD",
	},
	"USA500IDXUSD": {
		Symbol: "USA500IDXUSD", PipSize: 0.1, PipValue: 1.0, SpreadPips: 0.5, PipLocation: 1,
		ContractSize: 1, MinLot: 0.1, LotStep: 0.1, MarginRate: 5.0,
		PricePrecision: 2, CurrencyBase: "USA500", CurrencyQuote: "USD",
	},
	"USATECHIDXUSD": {
		Symbol: "USATECHIDXUSD", PipSize: 0.1, PipValue: 1.0, SpreadPips: 1.0, PipLocation: 1,
		ContractSize: 1, MinLot: 0.1, LotStep: 0.1, MarginRate: 5.0,
		PricePrecision: 2, CurrencyBase: "USATECH", CurrencyQuote: "USD",
	},
	"AUDCHF": {
		Symbol: "AUDCHF", PipSize: 0.0001, PipValue: 11.0, SpreadPips: 0.5, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "AUD", CurrencyQuote: "CHF",
	},
	"AUDCAD": {
		Symbol: "AUDCAD", PipSize: 0.0001, PipValue: 7.5, SpreadPips: 0.5, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "AUD", CurrencyQuote: "CAD",
	},
	"AUDNZD": {
		Symbol: "AUDNZD", PipSize: 0.0001, PipValue: 6.5, SpreadPips: 0.5, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "AUD", CurrencyQuote: "NZD",
	},
	"CADJPY": {
		Symbol: "CADJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.5, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "CAD", CurrencyQuote: "JPY",
	},
	"NZDJPY": {
		Symbol: "NZDJPY", PipSize: 0.01, PipValue: 9.0, SpreadPips: 0.5, PipLocation: 2,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 3, CurrencyBase: "NZD", CurrencyQuote: "JPY",
	},
	"GBPCHF": {
		Symbol: "GBPCHF", PipSize: 0.0001, PipValue: 11.0, SpreadPips: 0.7, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "GBP", CurrencyQuote: "CHF",
	},
	"GBPAUD": {
		Symbol: "GBPAUD", PipSize: 0.0001, PipValue: 6.5, SpreadPips: 0.7, PipLocation: 4,
		ContractSize: 100000, MinLot: 0.01, LotStep: 0.01, MarginRate: 1.0,
		PricePrecision: 5, CurrencyBase: "GBP", CurrencyQuote: "AUD",
	},
}

// GetSymbolConfig retrieves config for a symbol
func GetSymbolConfig(symbol string) (SymbolConfig, error) {
	config, ok := SymbolRegistry[strings.ToUpper(symbol)]
	if !ok {
		return SymbolConfig{}, fmt.Errorf("symbol %s not configured. Add to SymbolRegistry in constants/forex.go", symbol)
	}
	return config, nil
}

// MustGetSymbolConfig panics if symbol not found
func MustGetSymbolConfig(symbol string) SymbolConfig {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		panic(err)
	}
	return cfg
}

// ValidateSymbol checks if symbol is supported
func ValidateSymbol(symbol string) bool {
	_, exists := SymbolRegistry[strings.ToUpper(symbol)]
	return exists
}

// GetPipSizeForSymbol returns pip size for a symbol
func GetPipSizeForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 0.0001
	}
	return cfg.PipSize
}

// GetContractSizeForSymbol returns contract size per lot
func GetContractSizeForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 100000
	}
	return cfg.ContractSize
}

// GetPipValueForSymbol returns USD value per pip per standard lot
func GetPipValueForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 10.0
	}
	return cfg.PipValue
}

// GetMinLotForSymbol returns minimum lot size
func GetMinLotForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 0.01
	}
	return cfg.MinLot
}

// GetLotStepForSymbol returns lot size increment
func GetLotStepForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 0.01
	}
	return cfg.LotStep
}

// GetMarginRateForSymbol returns margin rate multiplier
func GetMarginRateForSymbol(symbol string) float64 {
	cfg, err := GetSymbolConfig(symbol)
	if err != nil {
		return 1.0
	}
	return cfg.MarginRate
}
