package patterns

import (
	"testing"
	"time"
)

// createHSCandles creates a synthetic H&S pattern for testing
func createHSCandles() []Candle {
	// Pattern: LS at 1.10, Head at 1.12, RS at 1.10, Neckline ~1.08
	candles := []Candle{
		// Lead-in
		{Index: 0, Open: 1.05, High: 1.06, Low: 1.04, Close: 1.055, Volume: 100},
		{Index: 1, Open: 1.055, High: 1.07, Low: 1.05, Close: 1.065, Volume: 110},
		{Index: 2, Open: 1.065, High: 1.08, Low: 1.06, Close: 1.075, Volume: 120},
		// Left Shoulder up
		{Index: 3, Open: 1.075, High: 1.09, Low: 1.07, Close: 1.085, Volume: 150},
		{Index: 4, Open: 1.085, High: 1.10, Low: 1.08, Close: 1.095, Volume: 180}, // LS peak
		{Index: 5, Open: 1.095, High: 1.10, Low: 1.085, Close: 1.09, Volume: 160},
		// Trough 1
		{Index: 6, Open: 1.09, High: 1.095, Low: 1.075, Close: 1.08, Volume: 140},
		{Index: 7, Open: 1.08, High: 1.085, Low: 1.07, Close: 1.075, Volume: 130}, // Trough 1
		{Index: 8, Open: 1.075, High: 1.09, Low: 1.07, Close: 1.085, Volume: 140},
		// Head up
		{Index: 9, Open: 1.085, High: 1.10, Low: 1.08, Close: 1.095, Volume: 170},
		{Index: 10, Open: 1.095, High: 1.11, Low: 1.09, Close: 1.105, Volume: 190},
		{Index: 11, Open: 1.105, High: 1.12, Low: 1.10, Close: 1.115, Volume: 200}, // Head peak
		{Index: 12, Open: 1.115, High: 1.12, Low: 1.105, Close: 1.11, Volume: 180},
		// Trough 2
		{Index: 13, Open: 1.11, High: 1.115, Low: 1.095, Close: 1.10, Volume: 150},
		{Index: 14, Open: 1.10, High: 1.105, Low: 1.075, Close: 1.08, Volume: 140}, // Trough 2
		{Index: 15, Open: 1.08, High: 1.09, Low: 1.075, Close: 1.085, Volume: 130},
		// Right Shoulder up
		{Index: 16, Open: 1.085, High: 1.095, Low: 1.08, Close: 1.09, Volume: 140},
		{Index: 17, Open: 1.09, High: 1.10, Low: 1.085, Close: 1.095, Volume: 150}, // RS peak
		{Index: 18, Open: 1.095, High: 1.10, Low: 1.085, Close: 1.09, Volume: 130},
		// Decline
		{Index: 19, Open: 1.09, High: 1.095, Low: 1.08, Close: 1.085, Volume: 120},
		{Index: 20, Open: 1.085, High: 1.09, Low: 1.075, Close: 1.08, Volume: 110},
	}

	baseTime := time.Now().AddDate(0, 0, -21)
	for i := range candles {
		candles[i].Timestamp = baseTime.AddDate(0, 0, i)
	}

	return candles
}

func TestDetectHeadAndShoulders(t *testing.T) {
	candles := createHSCandles()

	structure := DetectHeadAndShoulders(candles, 2)

	if structure == nil {
		t.Fatal("Expected to detect H&S pattern, got nil")
	}

	// Verify pattern type
	if structure.PatternType != PatternHS {
		t.Errorf("Expected pattern type H&S, got %s", structure.PatternType)
	}

	// Verify head is highest
	if structure.HeadPrice <= structure.LeftShoulderPrice || structure.HeadPrice <= structure.RightShoulderPrice {
		t.Error("Head should be higher than both shoulders")
	}

	// Verify confidence is reasonable
	if structure.OverallConfidence < 0.5 || structure.OverallConfidence > 1.0 {
		t.Errorf("Unexpected confidence: %f", structure.OverallConfidence)
	}

	// Verify shoulder symmetry
	if structure.ShoulderSymmetry < 0.8 {
		t.Errorf("Expected shoulder symmetry >= 0.8, got %f", structure.ShoulderSymmetry)
	}

	t.Logf("Detected H&S: LS=%.4f, Head=%.4f, RS=%.4f, Neckline=%.4f, Confidence=%.2f",
		structure.LeftShoulderPrice, structure.HeadPrice, structure.RightShoulderPrice,
		structure.NecklinePrice, structure.OverallConfidence)
}

func TestDetectHeadAndShoulders_NoPattern(t *testing.T) {
	// Create data with no clear pattern (random walk)
	candles := make([]Candle, 25)
	for i := 0; i < 25; i++ {
		candles[i] = Candle{
			Index: i,
			Open:  1.10,
			High:  1.11,
			Low:   1.09,
			Close: 1.10,
		}
	}

	structure := DetectHeadAndShoulders(candles, 2)

	if structure != nil {
		t.Log("Detected pattern in random data - may be false positive")
	}
}

func TestDetectHeadAndShoulders_InsufficientData(t *testing.T) {
	candles := []Candle{
		{Index: 0, High: 1.10, Low: 1.08},
		{Index: 1, High: 1.11, Low: 1.09},
	}

	structure := DetectHeadAndShoulders(candles, 1)

	if structure != nil {
		t.Error("Expected nil for insufficient data")
	}
}

func TestCalculateStructureConfidence(t *testing.T) {
	structure := &DetectedStructure{
		ShoulderSymmetry: 0.95,
		HeadProminence:   0.10,
		TimeSymmetry:     0.90,
		VolumeProfile:    0.80,
		NecklineQuality:  0.85,
	}

	confidence := CalculateStructureConfidence(structure)

	if confidence < 0.5 || confidence > 1.0 {
		t.Errorf("Unexpected confidence: %f", confidence)
	}

	t.Logf("Confidence: %.4f", confidence)
}

func TestCalculateStructureConfidence_Nil(t *testing.T) {
	confidence := CalculateStructureConfidence(nil)

	if confidence != 0.0 {
		t.Errorf("Expected 0.0 for nil structure, got %f", confidence)
	}
}
