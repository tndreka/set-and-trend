package domain

// ExecutionEventType represents market interaction events ONLY
type ExecutionEventType string

const (
	EventEntry        ExecutionEventType = "ENTRY_FILLED"
	EventPartialClose ExecutionEventType = "PARTIAL_EXIT"
	EventTPHit        ExecutionEventType = "TP_HIT"
	EventSLHit        ExecutionEventType = "SL_HIT"
	EventManualClose  ExecutionEventType = "MANUAL_CLOSE"
)

// TradeIntentType represents user decisions (NOT market events)
type TradeIntentType string

const (
	IntentCancel     TradeIntentType = "cancel"
	IntentInvalidate TradeIntentType = "invalidate"
)

// IsValidExecutionEvent checks if event type is a valid market execution
func IsValidExecutionEvent(eventType string) bool {
	switch ExecutionEventType(eventType) {
	case EventEntry, EventPartialClose, EventTPHit, EventSLHit, EventManualClose:
		return true
	default:
		return false
	}
}

// IsValidIntent checks if intent type is valid
func IsValidIntent(intentType string) bool {
	switch TradeIntentType(intentType) {
	case IntentCancel, IntentInvalidate:
		return true
	default:
		return false
	}
}

// IsClosingEvent returns true if this event closes the position
func IsClosingEvent(eventType ExecutionEventType) bool {
	return eventType == EventTPHit ||
		eventType == EventSLHit ||
		eventType == EventManualClose
}
