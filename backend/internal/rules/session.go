package rules

import "time"

type Session string

const (
	SessionSydney  Session = "sydney"
	SessionAsia    Session = "asia"
	SessionLondon  Session = "london"
	SessionNewYork Session = "new_york"
)

// SessionSpec defines a trading session window (UTC)
type SessionSpec struct {
	Name     Session
	StartUTC int // Minutes since midnight
	EndUTC   int // Minutes since midnight (can wrap)
}

// SessionRegistry - IMMUTABLE, FROZEN
// Change these and you corrupt historical analytics
var SessionRegistry = []SessionSpec{
	{
		Name:     SessionSydney,
		StartUTC: 21 * 60,      // 21:00 UTC
		EndUTC:   5*60 + 30,    // 05:30 UTC (wraps)
	},
	{
		Name:     SessionAsia,
		StartUTC: 0 * 60,       // 00:00 UTC
		EndUTC:   9 * 60,       // 09:00 UTC
	},
	{
		Name:     SessionLondon,
		StartUTC: 8 * 60,       // 08:00 UTC
		EndUTC:   16*60 + 30,   // 16:30 UTC
	},
	{
		Name:     SessionNewYork,
		StartUTC: 13 * 60,      // 13:00 UTC
		EndUTC:   21 * 60,      // 21:00 UTC
	},
}

// DeriveSessions returns ALL active sessions at time T
// NO DOMINANCE. Overlaps are preserved.
// Returns empty slice for weekends.
func DeriveSessions(t time.Time) []Session {
	weekday := t.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return []Session{}
	}

	minutesSinceMidnight := t.Hour()*60 + t.Minute()
	
	var active []Session
	for _, spec := range SessionRegistry {
		if isInSession(minutesSinceMidnight, spec) {
			active = append(active, spec.Name)
		}
	}
	
	return active
}

// isInSession checks if time falls within session window
// Handles wraparound (Sydney crosses midnight)
func isInSession(minutes int, spec SessionSpec) bool {
	if spec.StartUTC < spec.EndUTC {
		// Normal: same-day window
		return minutes >= spec.StartUTC && minutes < spec.EndUTC
	}
	
	// Wraparound: crosses midnight
	return minutes >= spec.StartUTC || minutes < spec.EndUTC
}
