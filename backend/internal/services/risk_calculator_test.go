package services

import (
	"math"
	"testing"
)

const floatTolerance = 0.00001

func almostEqualRisk(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestComputeRiskAmount(t *testing.T) {
	tests := []struct {
		name     string
		balance  float64
		riskPct  float64
		expected float64
		wantErr  bool
	}{
		{"1% of $10,000", 10000, 1.0, 100.0, false},
		{"2% of $5,000", 5000, 2.0, 100.0, false},
		{"0.5% of $20,000", 20000, 0.5, 100.0, false},
		{"negative balance", -10000, 1.0, 0, true},
		{"risk > 100%", 10000, 150, 0, true},
		{"zero balance", 0, 1.0, 0, true},
		{"negative risk", 10000, -1.0, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComputeRiskAmount(tt.balance, tt.riskPct)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !almostEqualRisk(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComputeStopDistance(t *testing.T) {
	tests := []struct {
		name     string
		entry    float64
		sl       float64
		expected float64
		wantErr  bool
	}{
		{"50 pips distance", 1.1050, 1.1000, 0.0050, false},
		{"100 pips distance", 1.1100, 1.1000, 0.0100, false},
		{"same price", 1.1000, 1.1000, 0, true},
		{"negative price", -1.1000, 1.1050, 0, true},
		{"zero price", 0, 1.1000, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComputeStopDistance(tt.entry, tt.sl)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !almostEqualRisk(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComputeStopDistancePips(t *testing.T) {
	tests := []struct {
		name         string
		stopDistance float64
		pipValue     float64
		expected     float64
		wantErr      bool
	}{
		{"EURUSD 50 pips", 0.0050, 0.0001, 50.0, false},
		{"EURUSD 100 pips", 0.0100, 0.0001, 100.0, false},
		{"zero pip value", 0.0050, 0, 0, true},
		{"zero distance", 0, 0.0001, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComputeStopDistancePips(tt.stopDistance, tt.pipValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !almostEqualRisk(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComputePositionSize(t *testing.T) {
	tests := []struct {
		name             string
		riskAmount       float64
		stopDistancePips float64
		pipValuePerLot   float64
		expected         float64
		wantErr          bool
	}{
		{"$100 risk, 50 pips, $10/pip", 100.0, 50.0, 10.0, 0.2, false},
		{"$200 risk, 100 pips, $10/pip", 200.0, 100.0, 10.0, 0.2, false},
		{"zero risk", 0, 50.0, 10.0, 0, true},
		{"zero stop distance", 100.0, 0, 10.0, 0, true},
		{"zero pip value", 100.0, 50.0, 0, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComputePositionSize(tt.riskAmount, tt.stopDistancePips, tt.pipValuePerLot)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !almostEqualRisk(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComputeRR(t *testing.T) {
	tests := []struct {
		name     string
		entry    float64
		sl       float64
		tp       float64
		bias     string
		expected float64
		wantErr  bool
	}{
		{
			name:     "Long 1:2 RR",
			entry:    1.1050,
			sl:       1.1000, // 50 pips risk
			tp:       1.1150, // 100 pips reward
			bias:     "long",
			expected: 2.0,
			wantErr:  false,
		},
		{
			name:     "Short 1:3 RR",
			entry:    1.1050,
			sl:       1.1100, // 50 pips risk
			tp:       1.0900, // 150 pips reward
			bias:     "short",
			expected: 3.0,
			wantErr:  false,
		},
		{
			name:     "Long 1:1 RR",
			entry:    1.1050,
			sl:       1.1000,
			tp:       1.1100,
			bias:     "long",
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "Invalid bias",
			entry:    1.1050,
			sl:       1.1000,
			tp:       1.1150,
			bias:     "sideways",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Zero price",
			entry:    0,
			sl:       1.1000,
			tp:       1.1150,
			bias:     "long",
			expected: 0,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComputeRR(tt.entry, tt.sl, tt.tp, tt.bias)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && !almostEqualRisk(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateTradeGeometry(t *testing.T) {
	tests := []struct {
		name    string
		entry   float64
		sl      float64
		tp      float64
		bias    string
		wantErr bool
	}{
		{"Valid long", 1.1050, 1.1000, 1.1200, "long", false},
		{"Valid short", 1.1050, 1.1100, 1.0900, "short", false},
		{"Long SL above entry", 1.1050, 1.1100, 1.1200, "long", true},
		{"Long SL equal entry", 1.1050, 1.1050, 1.1200, "long", true},
		{"Long TP below entry", 1.1050, 1.1000, 1.1040, "long", true},
		{"Long TP equal entry", 1.1050, 1.1000, 1.1050, "long", true},
		{"Short SL below entry", 1.1050, 1.1000, 1.0900, "short", true},
		{"Short SL equal entry", 1.1050, 1.1050, 1.0900, "short", true},
		{"Short TP above entry", 1.1050, 1.1100, 1.1060, "short", true},
		{"Short TP equal entry", 1.1050, 1.1100, 1.1050, "short", true},
		{"Invalid bias", 1.1050, 1.1000, 1.1200, "neutral", true},
		{"Zero entry", 0, 1.1000, 1.1200, "long", true},
		{"Negative price", 1.1050, -1.1000, 1.1200, "long", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTradeGeometry(tt.entry, tt.sl, tt.tp, tt.bias)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
		})
	}
}

// Test that proves ValidateTradeGeometry is deterministic
func TestValidateTradeGeometry_Deterministic(t *testing.T) {
	entry, sl, tp, bias := 1.1050, 1.1000, 1.1200, "long"
	
	err1 := ValidateTradeGeometry(entry, sl, tp, bias)
	err2 := ValidateTradeGeometry(entry, sl, tp, bias)
	
	if (err1 == nil) != (err2 == nil) {
		t.Error("ValidateTradeGeometry is not deterministic")
	}
}
