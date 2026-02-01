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
)

// SymbolConfig defines trading parameters per symbol
type SymbolConfig struct {
    Symbol         string
    PipSize        float64
    SpreadPips     float64  // ADD THIS LINE - Typical spread for cost modeling
    PipLocation    int      // 4 or 2
    ContractSize   float64
    PricePrecision int
    CurrencyBase   string
    CurrencyQuote  string
}

// Registry - EXPLICITLY DEFINED, NOT AUTO-DETECTED
var SymbolRegistry = map[string]SymbolConfig{
    "EURUSD": {
        Symbol: "EURUSD", PipSize: 0.0001, SpreadPips: 0.2, PipLocation: 4,
        ContractSize: 100000, PricePrecision: 5,
        CurrencyBase: "EUR", CurrencyQuote: "USD",
    },
    "GBPUSD": {
        Symbol: "GBPUSD", PipSize: 0.0001, SpreadPips: 0.3, PipLocation: 4,
        ContractSize: 100000, PricePrecision: 5,
        CurrencyBase: "GBP", CurrencyQuote: "USD",
    },
    "USDJPY": {
        Symbol: "USDJPY", PipSize: 0.01, SpreadPips: 0.2, PipLocation: 2,
        ContractSize: 100000, PricePrecision: 3,
        CurrencyBase: "USD", CurrencyQuote: "JPY",
    },
    "GBPJPY": {
        Symbol: "GBPJPY", PipSize: 0.01, SpreadPips: 0.8, PipLocation: 2,
        ContractSize: 100000, PricePrecision: 3,
        CurrencyBase: "GBP", CurrencyQuote: "JPY",
    },
    "AUDUSD": {
        Symbol: "AUDUSD", PipSize: 0.0001, SpreadPips: 0.2, PipLocation: 4,
        ContractSize: 100000, PricePrecision: 5,
        CurrencyBase: "AUD", CurrencyQuote: "USD",
    },
    "USDCHF": {
        Symbol: "USDCHF", PipSize: 0.0001, SpreadPips: 0.3, PipLocation: 4,
        ContractSize: 100000, PricePrecision: 5,
        CurrencyBase: "USD", CurrencyQuote: "CHF",
    },
    "USDCAD": {
        Symbol: "USDCAD", PipSize: 0.0001, SpreadPips: 0.3, PipLocation: 4,
        ContractSize: 100000, PricePrecision: 5,
        CurrencyBase: "USD", CurrencyQuote: "CAD",
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
// Temporary bridge function until all code is refactored
func GetPipSizeForSymbol(symbol string) float64 {
    cfg, err := GetSymbolConfig(symbol)
    if err != nil {
        // Fallback for backward compatibility
        return 0.0001
    }
    return cfg.PipSize
}