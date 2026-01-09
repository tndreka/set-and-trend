package domain

// ExecutionEventType represents market interaction events ONLY
type ExecutionEventType string

const (
	EventEntry        ExecutionEventType = "entry"
	EventPartialClose ExecutionEventType = "partial_close"
	EventTPHit        ExecutionEventType = "tp_hit"
	EventSLHit        ExecutionEventType = "sl_hit"
	EventManualClose  ExecutionEventType = "manual_close"
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
