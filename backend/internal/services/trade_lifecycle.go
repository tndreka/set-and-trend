package services

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// TradeState represents the current state of a trade
type TradeState string

const (
	StatePlanned     TradeState = "planned"
	StateOpen        TradeState = "open"
	StatePartial     TradeState = "partial"
	StateClosed      TradeState = "closed"
	StateCancelled   TradeState = "cancelled"
	StateInvalidated TradeState = "invalidated"
)

// TradeExecution represents an execution event
type TradeExecution struct {
	EventType    string
	Price        float64
	PositionSize float64
	ExecutedAt   time.Time
	PnL          float64
	PnLPips      float64
}

// DeriveTradeState determines current state from ordered execution history
func DeriveTradeState(
	lifecycleStatus *string,
	executions []TradeExecution,
) (TradeState, error) {
	// Terminal non-execution states
	if lifecycleStatus != nil {
		switch *lifecycleStatus {
		case "cancelled":
			return StateCancelled, nil
		case "invalidated":
			return StateInvalidated, nil
		}
	}
	
	// No executions yet
	if len(executions) == 0 {
		return StatePlanned, nil
	}
	
	// Sort by execution time (CRITICAL)
	sortedExecs := make([]TradeExecution, len(executions))
	copy(sortedExecs, executions)
	sort.Slice(sortedExecs, func(i, j int) bool {
		return sortedExecs[i].ExecutedAt.Before(sortedExecs[j].ExecutedAt)
	})
	
	// Replay state transitions
	currentState := StatePlanned
	
	for i, exec := range sortedExecs {
		// Validate transition is legal
		if err := CanTransition(currentState, exec.EventType); err != nil {
			return "", fmt.Errorf(
				"invalid execution sequence at index %d: %w",
				i, err,
			)
		}
		
		// Apply transition
		switch exec.EventType {
		case "entry":
			currentState = StateOpen
		case "partial_close":
			if currentState == StateOpen {
				currentState = StatePartial
			}
			// If already partial, stays partial
		case "tp_hit", "sl_hit", "manual_close":
			currentState = StateClosed
		}
	}
	
	return currentState, nil
}

// CanTransition validates if a state transition is allowed
func CanTransition(currentState TradeState, eventType string) error {
	validTransitions := map[TradeState][]string{
		StatePlanned: {
			"entry",      // → open
			"cancel",     // → cancelled
			"invalidate", // → invalidated
		},
		StateOpen: {
			"partial_close", // → partial
			"tp_hit",        // → closed
			"sl_hit",        // → closed
			"manual_close",  // → closed
		},
		StatePartial: {
			"partial_close", // → partial (additional)
			"tp_hit",        // → closed
			"sl_hit",        // → closed
			"manual_close",  // → closed
		},
		StateClosed:      {},
		StateCancelled:   {},
		StateInvalidated: {},
	}
	
	allowed := validTransitions[currentState]
	for _, valid := range allowed {
		if valid == eventType {
			return nil
		}
	}
	
	return fmt.Errorf(
		"invalid transition: cannot %s from %s state",
		eventType,
		currentState,
	)
}

// ComputeRemainingPosition calculates position size remaining after executions
func ComputeRemainingPosition(
	plannedPositionSize float64,
	executions []TradeExecution,
) (float64, error) {
	if plannedPositionSize <= 0 {
		return 0, errors.New("planned position size must be positive")
	}
	
	// Sort by execution time
	sortedExecs := make([]TradeExecution, len(executions))
	copy(sortedExecs, executions)
	sort.Slice(sortedExecs, func(i, j int) bool {
		return sortedExecs[i].ExecutedAt.Before(sortedExecs[j].ExecutedAt)
	})
	
	remaining := plannedPositionSize
	entryFilled := false
	
	for _, exec := range sortedExecs {
		switch exec.EventType {
		case "entry":
			entryFilled = true
			
		case "partial_close":
			if !entryFilled {
				return 0, errors.New("partial close before entry")
			}
			if exec.PositionSize <= 0 {
				return 0, errors.New("partial close size must be positive")
			}
			if exec.PositionSize >= remaining {
				return 0, fmt.Errorf(
					"partial close %.4f exceeds remaining %.4f",
					exec.PositionSize,
					remaining,
				)
			}
			remaining -= exec.PositionSize
			
		case "tp_hit", "sl_hit", "manual_close":
			if !entryFilled {
				return 0, errors.New("close event before entry")
			}
			if exec.PositionSize != remaining {
				return 0, fmt.Errorf(
					"close size %.4f does not match remaining %.4f",
					exec.PositionSize,
					remaining,
				)
			}
			remaining = 0
		}
	}
	
	return remaining, nil
}

// ValidateExecutionSize checks if a new execution size is valid
func ValidateExecutionSize(
	eventType string,
	executionSize float64,
	plannedSize float64,
	existingExecutions []TradeExecution,
) error {
	remaining, err := ComputeRemainingPosition(plannedSize, existingExecutions)
	if err != nil {
		return fmt.Errorf("compute remaining: %w", err)
	}
	
	switch eventType {
	case "entry":
		if executionSize != plannedSize {
			return fmt.Errorf(
				"entry size %.4f must match planned %.4f",
				executionSize,
				plannedSize,
			)
		}
		
	case "partial_close":
		if executionSize <= 0 {
			return errors.New("partial close size must be positive")
		}
		if executionSize >= remaining {
			return fmt.Errorf(
				"partial close %.4f must be less than remaining %.4f",
				executionSize,
				remaining,
			)
		}
		
	case "tp_hit", "sl_hit", "manual_close":
		if executionSize != remaining {
			return fmt.Errorf(
				"close size %.4f must match remaining %.4f",
				executionSize,
				remaining,
			)
		}
	}
	
	return nil
}

// ValidateTradeExecutable checks if trade can accept execution events
func ValidateTradeExecutable(
	lifecycleStatus *string,
	executions []TradeExecution,
) error {
	// Cannot execute if lifecycle status is terminal
	if lifecycleStatus != nil {
		return fmt.Errorf(
			"cannot execute: trade is %s",
			*lifecycleStatus,
		)
	}
	
	// Derive current state from executions
	state, err := DeriveTradeState(lifecycleStatus, executions)
	if err != nil {
		return fmt.Errorf("derive state: %w", err)
	}
	
	// Cannot execute if already closed
	if state == StateClosed {
		return errors.New("cannot execute: trade is closed")
	}
	
	return nil
}
