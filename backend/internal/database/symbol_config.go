package database

var SymbolConfig = map[string]SymbolInfo{
	"EURUSD": {Symbol: "EURUSD", Description: "Euro / US Dollar", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 0.8, CorrelationGroup: 1},
	"GBPUSD": {Symbol: "GBPUSD", Description: "British Pound / US Dollar", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 1.2, CorrelationGroup: 1},
	"USDJPY": {Symbol: "USDJPY", Description: "US Dollar / Japanese Yen", Category: CategoryMajorForex, PipValue: 0.01, TypicalSpread: 1.0, CorrelationGroup: 2},
	"USDCHF": {Symbol: "USDCHF", Description: "US Dollar / Swiss Franc", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 1.5, CorrelationGroup: 2},
	"USDCAD": {Symbol: "USDCAD", Description: "US Dollar / Canadian Dollar", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 1.8, CorrelationGroup: 2},
	"AUDUSD": {Symbol: "AUDUSD", Description: "Australian Dollar / US Dollar", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 1.2, CorrelationGroup: 1},
	"NZDUSD": {Symbol: "NZDUSD", Description: "New Zealand Dollar / US Dollar", Category: CategoryMajorForex, PipValue: 0.0001, TypicalSpread: 1.8, CorrelationGroup: 1},
	"EURGBP": {Symbol: "EURGBP", Description: "Euro / British Pound", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 1.5, CorrelationGroup: 5},
	"EURJPY": {Symbol: "EURJPY", Description: "Euro / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 1.8, CorrelationGroup: 6},
	"EURCHF": {Symbol: "EURCHF", Description: "Euro / Swiss Franc", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 2.0, CorrelationGroup: 5},
	"EURAUD": {Symbol: "EURAUD", Description: "Euro / Australian Dollar", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 2.5, CorrelationGroup: 5},
	"EURCAD": {Symbol: "EURCAD", Description: "Euro / Canadian Dollar", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 2.5, CorrelationGroup: 5},
	"GBPJPY": {Symbol: "GBPJPY", Description: "British Pound / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 2.5, CorrelationGroup: 6},
	"GBPCHF": {Symbol: "GBPCHF", Description: "British Pound / Swiss Franc", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 3.0, CorrelationGroup: 5},
	"GBPAUD": {Symbol: "GBPAUD", Description: "British Pound / Australian Dollar", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 3.0, CorrelationGroup: 5},
	"AUDJPY": {Symbol: "AUDJPY", Description: "Australian Dollar / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 2.0, CorrelationGroup: 6},
	"AUDNZD": {Symbol: "AUDNZD", Description: "Australian Dollar / New Zealand Dollar", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 2.5, CorrelationGroup: 7},
	"AUDCHF": {Symbol: "AUDCHF", Description: "Australian Dollar / Swiss Franc", Category: CategoryCrossPair, PipValue: 0.0001, TypicalSpread: 2.5, CorrelationGroup: 7},
	"CADJPY": {Symbol: "CADJPY", Description: "Canadian Dollar / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 2.5, CorrelationGroup: 6},
	"CHFJPY": {Symbol: "CHFJPY", Description: "Swiss Franc / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 2.5, CorrelationGroup: 6},
	"NZDJPY": {Symbol: "NZDJPY", Description: "New Zealand Dollar / Japanese Yen", Category: CategoryCrossPair, PipValue: 0.01, TypicalSpread: 2.5, CorrelationGroup: 6},
	"XAUUSD": {Symbol: "XAUUSD", Description: "Gold / US Dollar", Category: CategoryCommodity, PipValue: 0.01, TypicalSpread: 25, CorrelationGroup: 3},
	"XAGUSD": {Symbol: "XAGUSD", Description: "Silver / US Dollar", Category: CategoryCommodity, PipValue: 0.001, TypicalSpread: 2.0, CorrelationGroup: 3},
	"BRENTCMDUSD": {Symbol: "BRENTCMDUSD", Description: "Brent Crude Oil", Category: CategoryCommodity, PipValue: 0.01, TypicalSpread: 3.0, CorrelationGroup: 8},
	"LIGHTCMDUSD": {Symbol: "LIGHTCMDUSD", Description: "WTI Crude Oil", Category: CategoryCommodity, PipValue: 0.01, TypicalSpread: 3.0, CorrelationGroup: 8},
	"USA30IDXUSD":   {Symbol: "USA30IDXUSD", Description: "Dow Jones", Category: CategoryIndex, PipValue: 1.0, TypicalSpread: 2.0, CorrelationGroup: 4},
	"USA500IDXUSD":  {Symbol: "USA500IDXUSD", Description: "S&P 500", Category: CategoryIndex, PipValue: 0.1, TypicalSpread: 0.5, CorrelationGroup: 4},
	"USATECHIDXUSD": {Symbol: "USATECHIDXUSD", Description: "Nasdaq 100", Category: CategoryIndex, PipValue: 0.1, TypicalSpread: 1.0, CorrelationGroup: 4},
}

var CorrelationMatrix = map[string]map[string]float64{
	"EURUSD": {"GBPUSD": 0.95, "USDCHF": -0.95, "AUDUSD": 0.85, "NZDUSD": 0.80, "USDJPY": -0.40, "XAUUSD": 0.60},
	"GBPUSD": {"EURUSD": 0.95, "EURGBP": -0.90, "AUDUSD": 0.75, "USDCHF": -0.85},
	"USDJPY": {"USDCHF": 0.60, "USDCAD": 0.50, "EURUSD": -0.40},
	"XAUUSD": {"XAGUSD": 0.90, "USDCHF": -0.85, "EURUSD": 0.60},
	"XAGUSD": {"XAUUSD": 0.90},
	"USA30IDXUSD": {"USA500IDXUSD": 0.95, "USATECHIDXUSD": 0.90},
	"USA500IDXUSD": {"USA30IDXUSD": 0.95, "USATECHIDXUSD": 0.95},
	"USATECHIDXUSD": {"USA500IDXUSD": 0.95, "USA30IDXUSD": 0.90},
	"BRENTCMDUSD": {"LIGHTCMDUSD": 0.98, "USDCAD": -0.60},
	"LIGHTCMDUSD": {"BRENTCMDUSD": 0.98, "USDCAD": -0.60},
}

func GetCorrelation(symbol1, symbol2 string) float64 {
	if symbol1 == symbol2 {
		return 1.0
	}
	if corr, ok := CorrelationMatrix[symbol1][symbol2]; ok {
		return corr
	}
	if corr, ok := CorrelationMatrix[symbol2][symbol1]; ok {
		return corr
	}
	info1, ok1 := SymbolConfig[symbol1]
	info2, ok2 := SymbolConfig[symbol2]
	if ok1 && ok2 && info1.CorrelationGroup == info2.CorrelationGroup {
		return 0.70
	}
	return 0.0
}

func IsHighlyCorrelated(symbol1, symbol2 string, threshold float64) bool {
	corr := GetCorrelation(symbol1, symbol2)
	if corr < 0 {
		corr = -corr
	}
	return corr >= threshold
}
