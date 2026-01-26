package patterns

import (
	"testing"
	"time"
)

func TestFindSwingHighs(t *testing.T) {
	// Create sample data with clear swing highs
	candles := []Candle{
		{Index: 0, High: 100, Low: 95, Timestamp: time.Now()},
		{Index: 1, High: 102, Low: 97, Timestamp: time.Now()},
		{Index: 2, High: 110, Low: 100, Timestamp: time.Now()}, // Swing high
		{Index: 3, High: 108, Low: 98, Timestamp: time.Now()},
		{Index: 4, High: 105, Low: 90, Timestamp: time.Now()},
		{Index: 5, High: 107, Low: 92, Timestamp: time.Now()},
		{Index: 6, High: 115, Low: 105, Timestamp: time.Now()}, // Swing high
		{Index: 7, High: 113, Low: 103, Timestamp: time.Now()},
		{Index: 8, High: 110, Low: 100, Timestamp: time.Now()},
	}

	highs := FindSwingHighs(candles, 1)

	if len(highs) != 2 {
		t.Errorf("Expected 2 swing highs, got %d", len(highs))
	}

	// Verify first swing high
	if len(highs) > 0 && highs[0].Index != 2 {
		t.Errorf("Expected first swing high at index 2, got %d", highs[0].Index)
	}

	// Verify second swing high
	if len(highs) > 1 && highs[1].Index != 6 {
		t.Errorf("Expected second swing high at index 6, got %d", highs[1].Index)
	}
}

func TestFindSwingLows(t *testing.T) {
	candles := []Candle{
		{Index: 0, High: 110, Low: 105, Timestamp: time.Now()},
		{Index: 1, High: 108, Low: 103, Timestamp: time.Now()},
		{Index: 2, High: 105, Low: 95, Timestamp: time.Now()}, // Swing low
		{Index: 3, High: 107, Low: 97, Timestamp: time.Now()},
		{Index: 4, High: 112, Low: 102, Timestamp: time.Now()},
		{Index: 5, High: 110, Low: 100, Timestamp: time.Now()},
		{Index: 6, High: 106, Low: 90, Timestamp: time.Now()}, // Swing low
		{Index: 7, High: 108, Low: 92, Timestamp: time.Now()},
		{Index: 8, High: 115, Low: 100, Timestamp: time.Now()},
	}

	lows := FindSwingLows(candles, 1)

	if len(lows) != 2 {
		t.Errorf("Expected 2 swing lows, got %d", len(lows))
	}

	if len(lows) > 0 && lows[0].Index != 2 {
		t.Errorf("Expected first swing low at index 2, got %d", lows[0].Index)
	}

	if len(lows) > 1 && lows[1].Index != 6 {
		t.Errorf("Expected second swing low at index 6, got %d", lows[1].Index)
	}
}

func TestFindSwingHighs_InsufficientData(t *testing.T) {
	candles := []Candle{
		{Index: 0, High: 100, Low: 95},
		{Index: 1, High: 102, Low: 97},
	}

	highs := FindSwingHighs(candles, 2)

	if highs != nil {
		t.Errorf("Expected nil for insufficient data, got %v", highs)
	}
}

func TestFindMinBetweenPeaks(t *testing.T) {
	candles := []Candle{
		{Index: 0, High: 110, Low: 105},
		{Index: 1, High: 108, Low: 100},
		{Index: 2, High: 105, Low: 95}, // Min
		{Index: 3, High: 107, Low: 98},
		{Index: 4, High: 112, Low: 102},
	}

	minPoint := FindMinBetweenPeaks(candles, 0, 4)

	if minPoint == nil {
		t.Fatal("Expected to find minimum, got nil")
	}

	if minPoint.Index != 2 {
		t.Errorf("Expected min at index 2, got %d", minPoint.Index)
	}

	if minPoint.Price != 95 {
		t.Errorf("Expected min price 95, got %f", minPoint.Price)
	}
}

func TestLookbackByTimeframe(t *testing.T) {
	tests := []struct {
		tf       Timeframe
		expected int
	}{
		{TF_W1, 3},
		{TF_D1, 5},
		{TF_H4, 8},
		{TF_H1, 12},
	}

	for _, tt := range tests {
		result := LookbackByTimeframe(tt.tf)
		if result != tt.expected {
			t.Errorf("LookbackByTimeframe(%s) = %d, expected %d", tt.tf, result, tt.expected)
		}
	}
}
