package rules

// ConditionFunc is a pure boolean function that evaluates a condition
type ConditionFunc func(c Candle, ind Indicators) bool

// ConditionRegistry maps condition codes to their evaluation functions
var ConditionRegistry = map[ConditionCode]ConditionFunc{
	EMA50GtEMA200:      conditionEMA50GtEMA200,
	CloseGtEMA50:       conditionCloseGtEMA50,
	EMA50SlopePositive: conditionEMA50SlopePositive,
}

// conditionEMA50GtEMA200 checks if EMA50 is above EMA200 (bullish structure)
func conditionEMA50GtEMA200(c Candle, ind Indicators) bool {
	return ind.EMA50 > ind.EMA200
}

// conditionCloseGtEMA50 checks if close is above EMA50 (price in trend)
func conditionCloseGtEMA50(c Candle, ind Indicators) bool {
	return c.Close > ind.EMA50
}

// conditionEMA50SlopePositive checks if EMA50 is rising (trend momentum)
func conditionEMA50SlopePositive(c Candle, ind Indicators) bool {
	// ✅ FIXED: Check for nil instead of 0 sentinel
	if ind.EMA50Prev == nil {
		return false
	}
	return ind.EMA50 > *ind.EMA50Prev
}

// EvaluateCondition evaluates a single condition
func EvaluateCondition(code ConditionCode, c Candle, ind Indicators) bool {
	fn, exists := ConditionRegistry[code]
	if !exists {
		return false
	}
	return fn(c, ind)
}
