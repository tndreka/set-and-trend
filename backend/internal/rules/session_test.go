package rules

import (
	"testing"
	"time"
)

func TestDeriveSessions_Weekend(t *testing.T) {
	saturday := time.Date(2024, 12, 28, 12, 0, 0, 0, time.UTC)
	sessions := DeriveSessions(saturday)
	if len(sessions) != 0 {
		t.Errorf("Expected no sessions on Saturday, got %v", sessions)
	}

	sunday := time.Date(2024, 12, 29, 12, 0, 0, 0, time.UTC)
	sessions = DeriveSessions(sunday)
	if len(sessions) != 0 {
		t.Errorf("Expected no sessions on Sunday, got %v", sessions)
	}
}

func TestDeriveSessions_SingleSession(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		expected Session
	}{
		{
			name:     "London only",
			t:        time.Date(2024, 12, 25, 9, 0, 0, 0, time.UTC),
			expected: SessionLondon,
		},
		{
			name:     "Sydney only",
			t:        time.Date(2024, 12, 25, 22, 0, 0, 0, time.UTC),
			expected: SessionSydney,
		},
		{
			name:     "Asia only",
			t:        time.Date(2024, 12, 25, 6, 0, 0, 0, time.UTC),
			expected: SessionAsia,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := DeriveSessions(tt.t)
			if len(sessions) != 1 {
				t.Fatalf("Expected 1 session, got %d: %v", len(sessions), sessions)
			}
			if sessions[0] != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, sessions[0])
			}
		})
	}
}

func TestDeriveSessions_Overlaps(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		expected []Session
	}{
		{
			name:     "London + New York overlap",
			t:        time.Date(2024, 12, 25, 14, 0, 0, 0, time.UTC), // 14:00 UTC
			expected: []Session{SessionLondon, SessionNewYork},
		},
		{
			name:     "Sydney + Asia overlap",
			t:        time.Date(2024, 12, 25, 2, 0, 0, 0, time.UTC), // 02:00 UTC
			expected: []Session{SessionSydney, SessionAsia},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := DeriveSessions(tt.t)
			if len(sessions) != len(tt.expected) {
				t.Fatalf("Expected %d sessions, got %d: %v", len(tt.expected), len(sessions), sessions)
			}
			
			// Check all expected sessions are present (order-independent)
			for _, exp := range tt.expected {
				found := false
				for _, s := range sessions {
					if s == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected session %v not found in %v", exp, sessions)
				}
			}
		})
	}
}

func TestDeriveSessions_Deterministic(t *testing.T) {
	timestamp := time.Date(2024, 12, 25, 15, 30, 0, 0, time.UTC)
	
	sessions1 := DeriveSessions(timestamp)
	sessions2 := DeriveSessions(timestamp)
	
	if len(sessions1) != len(sessions2) {
		t.Fatalf("Non-deterministic: %v vs %v", sessions1, sessions2)
	}
	
	for i := range sessions1 {
		if sessions1[i] != sessions2[i] {
			t.Fatalf("Non-deterministic: %v vs %v", sessions1, sessions2)
		}
	}
}
