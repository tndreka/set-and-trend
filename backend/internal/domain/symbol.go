package domain

import (
	"fmt"
	"strings"
)

type SymbolConfig struct {
	Symbol         string
	PipSize        float64   // e.g., 0.0001 or 0.01
	PipLocation    int       // 4 or 2
	ContractSize   float64
	PricePrecision int
	CurrencyBase   string
	CurrencyQuote  string
}

// Registry - EXPLICITLY DEFINED, NOT AUTO-DETECTED
var SymbolRegistry = map[string]SymbolConfig{
	"EURUSD": {
		Symbol: "EURUSD", PipSize: 0.0001, PipLocation: 4,
		ContractSize: 100000, PricePrecision: 5,
		CurrencyBase: "EUR", CurrencyQuote: "USD",
	},
	"GBPUSD": {
		Symbol: "GBPUSD", PipSize: 0.0001, PipLocation: 4,
		ContractSize: 100000, PricePrecision: 5,
		CurrencyBase: "GBP", CurrencyQuote: "USD",
	},
	"USDJPY": {
		Symbol: "USDJPY", PipSize: 0.01, PipLocation: 2,
		ContractSize: 100000, PricePrecision: 3,
		CurrencyBase: "USD", CurrencyQuote: "JPY",
	},
	"GBPJPY": {
		Symbol: "GBPJPY", PipSize: 0.01, PipLocation: 2,
		ContractSize: 100000, PricePrecision: 3,
		CurrencyBase: "GBP", CurrencyQuote: "JPY",
	},
	"AUDUSD": {
		Symbol: "AUDUSD", PipSize: 0.0001, PipLocation: 4,
		ContractSize: 100000, PricePrecision: 5,
		CurrencyBase: "AUD", CurrencyQuote: "USD",
	},
	"USDCAD": {
		Symbol: "USDCAD", PipSize: 0.0001, PipLocation: 4,
		ContractSize: 100000, PricePrecision: 5,
		CurrencyBase: "USD", CurrencyQuote: "CAD",
	},
}

// GetSymbolConfig retrieves config for a symbol
func GetSymbolConfig(symbol string) (SymbolConfig, error) {
	config, ok := SymbolRegistry[strings.ToUpper(symbol)]
	if !ok {
		return SymbolConfig{}, fmt.Errorf("symbol %s not configured", symbol)
	}
	return config, nil
}