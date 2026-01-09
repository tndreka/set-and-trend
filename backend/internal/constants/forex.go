package constants

const (
	// EURUSD specifics
	SymbolEURUSD = "EURUSD"

	// Timeframes
	TimeframeW1 = "W1"

	// Pip definition
	PipValueEURUSD = 0.0001

	// Broker conventions
	PricePrecisionEURUSD = 5

	// Risk math guards
	MinStopLossPips = 5
	MaxStopLossPips = 500
	
	// Contract sizes (units per lot)
	ContractSizeEURUSD = 100000.0  // Standard lot = 100,000 units
	// Risk/Reward guards
	MinimumRR = 1.5
	// Entry execution tolerances (pips)
	MaxEntrySlippagePips = 20.0 // Max 20 pips slippage allowed
)
