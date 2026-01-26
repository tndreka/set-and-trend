package patterns

// PipSize for EURUSD
const PipSize = 0.0001

// GenerateTradeSignal creates entry/exit parameters from detected structure
func GenerateTradeSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	if s == nil {
		return nil
	}

	if s.PatternType == PatternHS {
		return generateHSSignal(s, symbol, confidence)
	} else if s.PatternType == PatternIHS {
		return generateIHSSignal(s, symbol, confidence)
	}

	return nil
}

// generateHSSignal creates a SHORT signal for bearish H&S
func generateHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Pattern height = Head - Neckline
	patternHeight := s.HeadPrice - s.NecklinePrice

	// Entry: Neckline break (with buffer)
	entryPrice := s.NecklinePrice - (10 * PipSize)

	// Stop Loss: Above right shoulder (with buffer)
	stopLoss := s.RightShoulderPrice + (20 * PipSize)

	// Take Profit: Pattern height projected below neckline
	takeProfit := s.NecklinePrice - patternHeight

	// Calculate risk-reward
	risk := stopLoss - entryPrice
	reward := entryPrice - takeProfit
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return &TradeSignal{
		Symbol:      symbol,
		Direction:   DirectionShort,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		RiskReward:  riskReward,
		Confidence:  confidence,
		PatternType: PatternHS,
	}
}

// generateIHSSignal creates a LONG signal for bullish IHS
func generateIHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Pattern height = Neckline - Head (head is lower in IHS)
	patternHeight := s.NecklinePrice - s.HeadPrice

	// Entry: Neckline break (with buffer)
	entryPrice := s.NecklinePrice + (10 * PipSize)

	// Stop Loss: Below right shoulder (with buffer)
	stopLoss := s.RightShoulderPrice - (20 * PipSize)

	// Take Profit: Pattern height projected above neckline
	takeProfit := s.NecklinePrice + patternHeight

	// Calculate risk-reward
	risk := entryPrice - stopLoss
	reward := takeProfit - entryPrice
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return &TradeSignal{
		Symbol:      symbol,
		Direction:   DirectionLong,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		RiskReward:  riskReward,
		Confidence:  confidence,
		PatternType: PatternIHS,
	}
}

// ValidateSignal checks if a signal meets minimum criteria
func ValidateSignal(signal *TradeSignal, minRR float64, minConfidence float64) bool {
	if signal == nil {
		return false
	}

	if signal.RiskReward < minRR {
		return false
	}

	if signal.Confidence < minConfidence {
		return false
	}

	// Sanity checks
	if signal.Direction == DirectionShort {
		// For SHORT: SL > Entry > TP
		if signal.StopLoss <= signal.EntryPrice || signal.EntryPrice <= signal.TakeProfit {
			return false
		}
	} else if signal.Direction == DirectionLong {
		// For LONG: TP > Entry > SL
		if signal.TakeProfit <= signal.EntryPrice || signal.EntryPrice <= signal.StopLoss {
			return false
		}
	}

	return true
}
