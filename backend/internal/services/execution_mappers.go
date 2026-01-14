package services

import (
	"fmt"
	"strconv"

	"set-and-trend/backend/internal/repositories"
)

// parseRequiredFloat converts *string → float64 or panics
func parseRequiredFloat(val *string, field string, execID any) float64 {
	if val == nil {
		panic(fmt.Sprintf("execution %v has NULL %s", execID, field))
	}

	f, err := strconv.ParseFloat(*val, 64)
	if err != nil {
		panic(fmt.Sprintf(
			"execution %v has invalid %s value '%s': %v",
			execID, field, *val, err,
		))
	}

	return f
}

// parseOptionalFloat converts *string → float64 (optional)
func parseOptionalFloat(val *string, field string, execID any) float64 {
	if val == nil {
		return 0
	}

	f, err := strconv.ParseFloat(*val, 64)
	if err != nil {
		panic(fmt.Sprintf(
			"execution %v has invalid %s value '%s': %v",
			execID, field, *val, err,
		))
	}

	return f
}

// Converts repository executions → domain executions
func mapToTradeExecutions(execs []repositories.TradeExecution) []TradeExecution {
	result := make([]TradeExecution, 0, len(execs))

	for _, e := range execs {
		te := TradeExecution{
			EventType:    e.EventType,
			Price:        parseRequiredFloat(e.Price, "price", e.ID),
			PositionSize: parseRequiredFloat(e.PositionSize, "position_size", e.ID),
			ExecutedAt:   e.ExecutedAt,
			PnL:          parseOptionalFloat(e.PnL, "pnl", e.ID),
			PnLPips:      parseOptionalFloat(e.PnLPips, "pnl_pips", e.ID),
		}

		result = append(result, te)
	}

	return result
}

// Converts repository intent → domain intent
func mapToTradeIntent(i *repositories.TradeIntent) *TradeIntent {
	if i == nil {
		return nil
	}

	return &TradeIntent{
		IntentType: i.IntentType,
		Reason:     i.Reason,
		CreatedAt:  i.CreatedAt,
	}
}
