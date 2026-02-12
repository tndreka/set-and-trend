package constants

type Symbol struct {
	Code         string
	PipSize      float64
	ContractSize float64
	AssetClass   AssetClass
	// NEW: Volatility scaling factor for tolerance settings
	ATRMultiplier float64 // Scales default pips for this asset class
}

type AssetClass string

const (
	FXMajor AssetClass = "FX_MAJOR"
	FXCross AssetClass = "FX_CROSS"
	Metal   AssetClass = "METAL"
	Index   AssetClass = "INDEX"
)

// Registry with ATR-aware multipliers
// FX: 1x | Metals: 1.5x (more volatile) | Indices: 2x (much more volatile)
var Registry = map[string]Symbol{
	// Majors (1x multiplier)
	"EURUSD": {Code: "EURUSD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"GBPUSD": {Code: "GBPUSD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"USDJPY": {Code: "USDJPY", PipSize: 0.01, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"USDCHF": {Code: "USDCHF", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"AUDUSD": {Code: "AUDUSD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"NZDUSD": {Code: "NZDUSD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},
	"USDCAD": {Code: "USDCAD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXMajor, ATRMultiplier: 1.0},

	// Crosses (1x multiplier)
	"AUDJPY": {Code: "AUDJPY", PipSize: 0.01, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"CHFJPY": {Code: "CHFJPY", PipSize: 0.01, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"EURAUD": {Code: "EURAUD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"EURCAD": {Code: "EURCAD", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"EURCHF": {Code: "EURCHF", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"EURGBP": {Code: "EURGBP", PipSize: 0.0001, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"EURJPY": {Code: "EURJPY", PipSize: 0.01, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},
	"GBPJPY": {Code: "GBPJPY", PipSize: 0.01, ContractSize: 100000, AssetClass: FXCross, ATRMultiplier: 1.0},

	// Metals (1.5x multiplier - more volatile)
	"XAUUSD": {Code: "XAUUSD", PipSize: 0.01, ContractSize: 100, AssetClass: Metal, ATRMultiplier: 1.5},
	"XAGUSD": {Code: "XAGUSD", PipSize: 0.001, ContractSize: 5000, AssetClass: Metal, ATRMultiplier: 1.5},

	// Indices (2x multiplier - much more volatile)
	"USA30IDXUSD":   {Code: "USA30IDXUSD", PipSize: 1.0, ContractSize: 1, AssetClass: Index, ATRMultiplier: 2.0},
	"USA500IDXUSD":  {Code: "USA500IDXUSD", PipSize: 0.1, ContractSize: 1, AssetClass: Index, ATRMultiplier: 2.0},
	"USATECHIDXUSD": {Code: "USATECHIDXUSD", PipSize: 0.1, ContractSize: 1, AssetClass: Index, ATRMultiplier: 2.0},
}

// GetScaledPips returns pip value adjusted for asset volatility
// Use this for tolerance settings, not for price calculations
func (s Symbol) GetScaledPips(basePips float64) float64 {
	return basePips * s.ATRMultiplier
}

func (s Symbol) PriceToPips(priceDiff float64) float64 {
	if s.PipSize == 0 {
		return 0
	}
	return priceDiff / s.PipSize
}

func (s Symbol) PipsToPrice(pips float64) float64 {
	return pips * s.PipSize
}

func Get(symbol string) (Symbol, bool) {
	s, ok := Registry[symbol]
	return s, ok
}

func MustGet(symbol string) Symbol {
	s, ok := Registry[symbol]
	if !ok {
		panic("unknown symbol: " + symbol)
	}
	return s
}
