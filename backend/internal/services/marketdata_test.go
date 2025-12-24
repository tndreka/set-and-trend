package services

import (
	"math"
	"testing"

	"set-and-trend/backend/internal/domain"
)

// helper to compare floats with tolerance
func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestComputeIndicators(t *testing.T) {
	candles := []domain.Candle{
		{Open: 1.1000, High: 1.1050, Low: 1.0950, Close: 1.1025},
	}

	indicators := ComputeIndicators(candles, 0)

	tol := 1e-9 // tolerance for float comparison

	expectedRange := 1.1050 - 1.0950
	expectedBody := math.Abs(1.1025 - 1.1000)
	expectedUpperWick := 1.1050 - math.Max(1.1000, 1.1025)
	expectedLowerWick := math.Min(1.1000, 1.1025) - 1.0950
	expectedMidPrice := (1.1050 + 1.0950) / 2.0

	if !almostEqual(indicators.RangeSize, expectedRange, tol) {
		t.Errorf("expected RangeSize %v, got %v", expectedRange, indicators.RangeSize)
	}
	if !almostEqual(indicators.BodySize, expectedBody, tol) {
		t.Errorf("expected BodySize %v, got %v", expectedBody, indicators.BodySize)
	}
	if !almostEqual(indicators.UpperWick, expectedUpperWick, tol) {
		t.Errorf("expected UpperWick %v, got %v", expectedUpperWick, indicators.UpperWick)
	}
	if !almostEqual(indicators.LowerWick, expectedLowerWick, tol) {
		t.Errorf("expected LowerWick %v, got %v", expectedLowerWick, indicators.LowerWick)
	}
	if !almostEqual(indicators.MidPrice, expectedMidPrice, tol) {
		t.Errorf("expected MidPrice %v, got %v", expectedMidPrice, indicators.MidPrice)
	}
}
