package patterns

import "testing"

func TestGenerateTradeSignal_HS(t *testing.T) {
	structure := &DetectedStructure{
		HeadPrice:          1.1200,
		LeftShoulderPrice:  1.1000,
		RightShoulderPrice: 1.1000,
		NecklinePrice:      1.0800,
		PatternType:        PatternHS,
		OverallConfidence:  0.75,
	}

	signal := GenerateTradeSignal(structure, "EURUSD", 0.75)

	if signal == nil {
		t.Fatal("Expected signal, got nil")
	}

	if signal.Direction != DirectionShort {
		t.Errorf("Expected SHORT direction, got %s", signal.Direction)
	}

	// Entry should be below neckline
	if signal.EntryPrice >= structure.NecklinePrice {
		t.Errorf("Entry should be below neckline: %f >= %f", signal.EntryPrice, structure.NecklinePrice)
	}

	// SL should be above right shoulder
	if signal.StopLoss <= structure.RightShoulderPrice {
		t.Errorf("SL should be above right shoulder: %f <= %f", signal.StopLoss, structure.RightShoulderPrice)
	}

	// TP should be below entry
	if signal.TakeProfit >= signal.EntryPrice {
		t.Errorf("TP should be below entry for SHORT: %f >= %f", signal.TakeProfit, signal.EntryPrice)
	}

	// RR should be positive
	if signal.RiskReward <= 0 {
		t.Errorf("Expected positive RR, got %f", signal.RiskReward)
	}

	t.Logf("Signal: Entry=%.5f, SL=%.5f, TP=%.5f, RR=%.2f",
		signal.EntryPrice, signal.StopLoss, signal.TakeProfit, signal.RiskReward)
}

func TestGenerateTradeSignal_IHS(t *testing.T) {
	structure := &DetectedStructure{
		HeadPrice:          1.0800,
		LeftShoulderPrice:  1.1000,
		RightShoulderPrice: 1.1000,
		NecklinePrice:      1.1200,
		PatternType:        PatternIHS,
		OverallConfidence:  0.70,
	}

	signal := GenerateTradeSignal(structure, "EURUSD", 0.70)

	if signal == nil {
		t.Fatal("Expected signal, got nil")
	}

	if signal.Direction != DirectionLong {
		t.Errorf("Expected LONG direction, got %s", signal.Direction)
	}

	// Entry should be above neckline
	if signal.EntryPrice <= structure.NecklinePrice {
		t.Errorf("Entry should be above neckline: %f <= %f", signal.EntryPrice, structure.NecklinePrice)
	}

	// SL should be below right shoulder
	if signal.StopLoss >= structure.RightShoulderPrice {
		t.Errorf("SL should be below right shoulder: %f >= %f", signal.StopLoss, structure.RightShoulderPrice)
	}

	// TP should be above entry
	if signal.TakeProfit <= signal.EntryPrice {
		t.Errorf("TP should be above entry for LONG: %f <= %f", signal.TakeProfit, signal.EntryPrice)
	}

	t.Logf("Signal: Entry=%.5f, SL=%.5f, TP=%.5f, RR=%.2f",
		signal.EntryPrice, signal.StopLoss, signal.TakeProfit, signal.RiskReward)
}

func TestValidateSignal(t *testing.T) {
	tests := []struct {
		name          string
		signal        *TradeSignal
		minRR         float64
		minConfidence float64
		expected      bool
	}{
		{
			name: "Valid SHORT signal",
			signal: &TradeSignal{
				Direction:  DirectionShort,
				EntryPrice: 1.0800,
				StopLoss:   1.1000,
				TakeProfit: 1.0400,
				RiskReward: 2.0,
				Confidence: 0.70,
			},
			minRR:         1.5,
			minConfidence: 0.60,
			expected:      true,
		},
		{
			name: "Low RR",
			signal: &TradeSignal{
				Direction:  DirectionShort,
				EntryPrice: 1.0800,
				StopLoss:   1.1000,
				TakeProfit: 1.0600,
				RiskReward: 1.0,
				Confidence: 0.70,
			},
			minRR:         1.5,
			minConfidence: 0.60,
			expected:      false,
		},
		{
			name: "Low confidence",
			signal: &TradeSignal{
				Direction:  DirectionShort,
				EntryPrice: 1.0800,
				StopLoss:   1.1000,
				TakeProfit: 1.0400,
				RiskReward: 2.0,
				Confidence: 0.50,
			},
			minRR:         1.5,
			minConfidence: 0.60,
			expected:      false,
		},
		{
			name:          "Nil signal",
			signal:        nil,
			minRR:         1.5,
			minConfidence: 0.60,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSignal(tt.signal, tt.minRR, tt.minConfidence)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
