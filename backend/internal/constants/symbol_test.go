package constants

import (
	"math"
	"testing"
)

// TestPipSizeConsistency validates all symbols have correct pip sizes
func TestPipSizeConsistency(t *testing.T) {
	tests := []struct {
		symbol  string
		wantPip float64
		class   AssetClass
	}{
		// Majors
		{"EURUSD", 0.0001, FXMajor},
		{"USDJPY", 0.01, FXMajor},
		// Metals
		{"XAUUSD", 0.01, Metal},
		{"XAGUSD", 0.001, Metal},
		// Indices
		{"USA30IDXUSD", 1.0, Index},
		{"USA500IDXUSD", 0.1, Index},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			sym, ok := Get(tt.symbol)
			if !ok {
				t.Fatalf("symbol %s not found", tt.symbol)
			}
			if sym.PipSize != tt.wantPip {
				t.Errorf("PipSize = %v, want %v", sym.PipSize, tt.wantPip)
			}
			if sym.AssetClass != tt.class {
				t.Errorf("AssetClass = %v, want %v", sym.AssetClass, tt.class)
			}
		})
	}
}

// TestPipConversions validates PriceToPips / PipsToPrice round-trip
func TestPipConversions(t *testing.T) {
	tests := []struct {
		symbol    string
		priceDiff float64
		wantPips  float64
	}{
		{"EURUSD", 0.0020, 20},      // 20 pips
		{"USDJPY", 0.50, 50},        // 50 pips
		{"XAUUSD", 5.0, 500},        // 500 pips (gold)
		{"USA500IDXUSD", 15.0, 150}, // 150 points (S&P)
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			sym := MustGet(tt.symbol)

			// Test PriceToPips
			gotPips := sym.PriceToPips(tt.priceDiff)
			if math.Abs(gotPips-tt.wantPips) > 0.001 {
				t.Errorf("PriceToPips(%v) = %v, want %v", tt.priceDiff, gotPips, tt.wantPips)
			}

			// Test round-trip: PipsToPrice(PriceToPips(x)) ≈ x
			backToPrice := sym.PipsToPrice(gotPips)
			if math.Abs(backToPrice-tt.priceDiff) > 0.0001 {
				t.Errorf("Round-trip failed: %v -> %v pips -> %v price",
					tt.priceDiff, gotPips, backToPrice)
			}
		})
	}
}

// TestVolatilityScaling ensures different assets get different tolerances
func TestVolatilityScaling(t *testing.T) {
	eur := MustGet("EURUSD")
	xau := MustGet("XAUUSD")
	spx := MustGet("USA500IDXUSD")

	basePips := 20.0

	eurScaled := eur.GetScaledPips(basePips)
	xauScaled := xau.GetScaledPips(basePips)
	spxScaled := spx.GetScaledPips(basePips)

	// EURUSD: 20 pips (1.0x)
	if eurScaled != 20.0 {
		t.Errorf("EURUSD scaled pips = %v, want 20", eurScaled)
	}

	// XAUUSD: 30 pips (1.5x)
	if xauScaled != 30.0 {
		t.Errorf("XAUUSD scaled pips = %v, want 30", xauScaled)
	}

	// SPX: 40 pips (2.0x)
	if spxScaled != 40.0 {
		t.Errorf("SPX scaled pips = %v, want 40", spxScaled)
	}

	t.Logf("20 pip tolerance scales to: EUR=%.1f, XAU=%.1f, SPX=%.1f",
		eurScaled, xauScaled, spxScaled)
}

// TestCrossAssetStopDistance validates that 20-pip risk = different price distances
func TestCrossAssetStopDistance(t *testing.T) {
	// Simulate: 20 pip stop loss on each asset
	riskPips := 20.0

	eur := MustGet("EURUSD")
	jpy := MustGet("USDJPY")
	xau := MustGet("XAUUSD")
	spx := MustGet("USA500IDXUSD")

	eurDistance := eur.PipsToPrice(riskPips) // 0.0020
	jpyDistance := jpy.PipsToPrice(riskPips) // 0.20
	xauDistance := xau.PipsToPrice(riskPips) // 0.20
	spxDistance := spx.PipsToPrice(riskPips) // 2.0

	// Verify: Same 20-pip risk = vastly different price distances
	tests := []struct {
		name     string
		got      float64
		expected float64
	}{
		{"EURUSD", eurDistance, 0.0020},
		{"USDJPY", jpyDistance, 0.20},
		{"XAUUSD", xauDistance, 0.20},
		{"SPX", spxDistance, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if math.Abs(tt.got-tt.expected) > 0.0001 {
				t.Errorf("20-pip distance = %v, want %v", tt.got, tt.expected)
			}
		})
	}

	t.Logf("20-pip stop distance: EUR=%.4f, JPY=%.2f, XAU=%.2f, SPX=%.1f",
		eurDistance, jpyDistance, xauDistance, spxDistance)
}
