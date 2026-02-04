package patterns

import (
	"time"
)

// ================================================================================
// KILL ZONE SESSIONS - Institutional Volume Periods
// ================================================================================
// WHY: London/NY = 70% daily volume. Asia signals = noise.
// HOW: Filter signals by session timing for higher probability setups.
// ================================================================================

// SessionType represents trading sessions
type SessionType string

const (
	SessionAsian       SessionType = "ASIAN"
	SessionLondon      SessionType = "LONDON"
	SessionNewYork     SessionType = "NEW_YORK"
	SessionLondonNY    SessionType = "LONDON_NY"    // Overlap (highest volume)
	SessionOffHours    SessionType = "OFF_HOURS"
)

// KillZone represents a high-probability trading window
type KillZone struct {
	Name        string
	Session     SessionType
	StartHour   int // UTC hour
	StartMinute int
	EndHour     int
	EndMinute   int
	Weight      float64 // How important is this session (0.0-1.0)
	VolumePct   float64 // % of daily volume
}

// KillZoneResult holds the analysis of current timing
type KillZoneResult struct {
	CurrentSession    SessionType
	IsInKillZone      bool
	KillZoneName      string
	SessionWeight     float64
	MinutesToOpen     int   // Minutes until next kill zone
	MinutesToClose    int   // Minutes until current kill zone ends
	NextKillZone      string
	DayOfWeek         time.Weekday
	IsWeekendClose    bool  // Friday close / Sunday open
}

// Standard Kill Zones (UTC times)
var StandardKillZones = []KillZone{
	// Asian Session - Low volume, range-bound
	{
		Name:        "Asian Session",
		Session:     SessionAsian,
		StartHour:   0,
		StartMinute: 0,
		EndHour:     8,
		EndMinute:   0,
		Weight:      0.3,
		VolumePct:   15,
	},
	// London Kill Zone - High probability
	{
		Name:        "London Open Kill Zone",
		Session:     SessionLondon,
		StartHour:   7,
		StartMinute: 0,
		EndHour:     10,
		EndMinute:   0,
		Weight:      0.9,
		VolumePct:   30,
	},
	// London Session (extended)
	{
		Name:        "London Session",
		Session:     SessionLondon,
		StartHour:   8,
		StartMinute: 0,
		EndHour:     16,
		EndMinute:   0,
		Weight:      0.7,
		VolumePct:   35,
	},
	// London/NY Overlap - HIGHEST volume
	{
		Name:        "London/NY Overlap",
		Session:     SessionLondonNY,
		StartHour:   13,
		StartMinute: 0,
		EndHour:     16,
		EndMinute:   0,
		Weight:      1.0,
		VolumePct:   25,
	},
	// NY Kill Zone
	{
		Name:        "New York Open Kill Zone",
		Session:     SessionNewYork,
		StartHour:   12,
		StartMinute: 0,
		EndHour:     15,
		EndMinute:   0,
		Weight:      0.85,
		VolumePct:   25,
	},
	// NY Session (extended)
	{
		Name:        "New York Session",
		Session:     SessionNewYork,
		StartHour:   13,
		StartMinute: 0,
		EndHour:     21,
		EndMinute:   0,
		Weight:      0.6,
		VolumePct:   30,
	},
}

// AnalyzeKillZone determines if current time is in a high-probability window
func AnalyzeKillZone(t time.Time) *KillZoneResult {
	// Convert to UTC for consistent analysis
	utc := t.UTC()
	hour := utc.Hour()
	minute := utc.Minute()
	currentMinutes := hour*60 + minute
	dayOfWeek := utc.Weekday()

	result := &KillZoneResult{
		CurrentSession: SessionOffHours,
		IsInKillZone:   false,
		SessionWeight:  0.0,
		DayOfWeek:      dayOfWeek,
	}

	// Check for weekend
	if dayOfWeek == time.Saturday {
		result.IsWeekendClose = true
		result.NextKillZone = "Sunday Open (22:00 UTC)"
		return result
	}
	if dayOfWeek == time.Sunday && hour < 22 {
		result.IsWeekendClose = true
		result.NextKillZone = "Sunday Open (22:00 UTC)"
		result.MinutesToOpen = (22-hour)*60 - minute
		return result
	}

	// Friday close warning
	if dayOfWeek == time.Friday && hour >= 20 {
		result.IsWeekendClose = true
	}

	// Find current kill zone
	bestWeight := 0.0
	for _, kz := range StandardKillZones {
		kzStart := kz.StartHour*60 + kz.StartMinute
		kzEnd := kz.EndHour*60 + kz.EndMinute

		if currentMinutes >= kzStart && currentMinutes < kzEnd {
			if kz.Weight > bestWeight {
				bestWeight = kz.Weight
				result.CurrentSession = kz.Session
				result.IsInKillZone = true
				result.KillZoneName = kz.Name
				result.SessionWeight = kz.Weight
				result.MinutesToClose = kzEnd - currentMinutes
			}
		}
	}

	// If not in kill zone, find next one
	if !result.IsInKillZone {
		minWait := 24 * 60 // Max wait
		for _, kz := range StandardKillZones {
			kzStart := kz.StartHour*60 + kz.StartMinute
			
			wait := kzStart - currentMinutes
			if wait < 0 {
				wait += 24 * 60 // Next day
			}

			if wait < minWait && kz.Weight >= 0.8 { // Only consider high-quality zones
				minWait = wait
				result.NextKillZone = kz.Name
				result.MinutesToOpen = wait
			}
		}
	}

	return result
}

// GetSessionForCandle determines what session a candle was in
func GetSessionForCandle(c Candle) SessionType {
	result := AnalyzeKillZone(c.Timestamp)
	return result.CurrentSession
}

// IsHighVolumeSession returns true if in London or NY session
func IsHighVolumeSession(t time.Time) bool {
	result := AnalyzeKillZone(t)
	return result.CurrentSession == SessionLondon ||
		   result.CurrentSession == SessionNewYork ||
		   result.CurrentSession == SessionLondonNY
}

// IsKillZoneOpen returns true if in a kill zone (open period)
func IsKillZoneOpen(t time.Time) bool {
	result := AnalyzeKillZone(t)
	return result.IsInKillZone && result.SessionWeight >= 0.8
}

// CalculateSessionScore returns confluence score (0.0-1.0) for session timing
func CalculateSessionScore(t time.Time) float64 {
	result := AnalyzeKillZone(t)

	if result.IsWeekendClose {
		return 0.0 // Never trade near weekend close
	}

	if !result.IsInKillZone {
		return 0.1 // Base score for off-hours
	}

	// Return the session weight (already 0.0-1.0)
	return result.SessionWeight
}

// FilterCandlesBySession returns only candles from specified sessions
func FilterCandlesBySession(candles []Candle, sessions ...SessionType) []Candle {
	sessionSet := make(map[SessionType]bool)
	for _, s := range sessions {
		sessionSet[s] = true
	}

	var filtered []Candle
	for _, c := range candles {
		session := GetSessionForCandle(c)
		if sessionSet[session] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// AnalyzeSessionDistribution analyzes how signals are distributed across sessions
func AnalyzeSessionDistribution(signals []TradeSignal, candles []Candle) map[SessionType]int {
	dist := make(map[SessionType]int)

	for _, c := range candles {
		session := GetSessionForCandle(c)
		dist[session]++
	}

	return dist
}

// GetOptimalEntryWindow suggests when to look for entries based on pattern
func GetOptimalEntryWindow(patternType string, currentTime time.Time) *KillZoneResult {
	result := AnalyzeKillZone(currentTime)

	// H&S patterns work best in London/NY
	if patternType == PatternHS || patternType == PatternIHS {
		if result.CurrentSession == SessionAsian {
			// Suggest waiting for London
			result.NextKillZone = "London Open Kill Zone"
			result.MinutesToOpen = calculateMinutesToLondon(currentTime)
		}
	}

	return result
}

func calculateMinutesToLondon(t time.Time) int {
	utc := t.UTC()
	hour := utc.Hour()
	minute := utc.Minute()
	
	londonOpen := 7 * 60 // 07:00 UTC
	currentMinutes := hour*60 + minute

	if currentMinutes < londonOpen {
		return londonOpen - currentMinutes
	}
	// Wait until tomorrow
	return (24*60 - currentMinutes) + londonOpen
}

// DayOfWeekScore returns a score modifier based on day of week
func DayOfWeekScore(t time.Time) float64 {
	day := t.UTC().Weekday()
	
	switch day {
	case time.Monday:
		return 0.7 // Monday can be slow, gaps
	case time.Tuesday, time.Wednesday, time.Thursday:
		return 1.0 // Best trading days
	case time.Friday:
		hour := t.UTC().Hour()
		if hour >= 18 {
			return 0.3 // Friday late = avoid
		}
		return 0.8 // Friday morning still okay
	case time.Saturday, time.Sunday:
		return 0.0 // No trading
	}
	return 0.5
}

// ShouldAvoidEntry returns true if timing suggests avoiding new entries
func ShouldAvoidEntry(t time.Time) bool {
	result := AnalyzeKillZone(t)
	
	// Avoid weekend close
	if result.IsWeekendClose {
		return true
	}
	
	// Avoid very low-weight sessions
	if result.SessionWeight < 0.3 {
		return true
	}
	
	// Avoid major news times (would need external data)
	// This is a placeholder for news filter integration
	
	return false
}

// GetKillZoneMultiplier returns a multiplier for confluence based on session
func GetKillZoneMultiplier(t time.Time) float64 {
	result := AnalyzeKillZone(t)
	
	if result.IsWeekendClose {
		return 0.5 // Half weight near weekend
	}
	
	dayScore := DayOfWeekScore(t)
	sessionScore := result.SessionWeight
	
	// Combine day and session scores
	return (dayScore * 0.3) + (sessionScore * 0.7)
}
