package patterns

import (
	"set-and-trend/backend/internal/constants"
)

// GenerateTradeSignal creates entry/exit parameters from detected structure
func GenerateTradeSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	if s == nil {
		return nil
	}

	if s.PatternType == PatternHS {
		return generateHSSignal(s, symbol, confidence)
	} else if s.PatternType == PatternIHS {
		return generateIHSSignal(s, symbol, confidence)
	} else if s.PatternType == "DOUBLE_TOP" {
		return generateDoubleTopSignal(s, symbol, confidence)
	} else if s.PatternType == "DOUBLE_BOTTOM" {
		return generateDoubleBottomSignal(s, symbol, confidence)
	} else if s.PatternType == "TRIPLE_TOP" {
		return generateTripleTopSignal(s, symbol, confidence)
	} else if s.PatternType == "TRIPLE_BOTTOM" {
		return generateTripleBottomSignal(s, symbol, confidence)
	}

	return nil
}

// generateHSSignal creates a SHORT signal for bearish H&S
func generateHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Get pip size for this symbol from registry
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Pattern height = Head - Neckline
	patternHeight := s.HeadPrice - s.NecklinePrice

	// Entry: Neckline break (with buffer)
	entryPrice := s.NecklinePrice - (10 * pipSize)

	// Stop Loss: Above right shoulder (with buffer)
	stopLoss := s.RightShoulderPrice + (20 * pipSize)

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
		Symbol:     symbol,
		Direction:  DirectionShort,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		RiskReward: riskReward,
		Confidence: confidence,
	}
}

// generateIHSSignal creates a LONG signal for bullish inverse H&S
func generateIHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Get pip size for this symbol from registry
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Pattern height = Neckline - Head (inverse)
	patternHeight := s.NecklinePrice - s.HeadPrice

	// Entry: Neckline break (with buffer)
	entryPrice := s.NecklinePrice + (10 * pipSize)

	// Stop Loss: Below right shoulder (with buffer)
	stopLoss := s.RightShoulderPrice - (20 * pipSize)

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
		Symbol:     symbol,
		Direction:  DirectionLong,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		RiskReward: riskReward,
		Confidence: confidence,
	}
}

// ValidateSignal checks if a signal meets minimum requirements
func ValidateSignal(signal *TradeSignal, minRR float64, minConfidence float64) bool {
	if signal == nil {
		return false
	}
	return signal.RiskReward >= minRR && signal.Confidence >= minConfidence
}

// generateDoubleTopSignal creates a SHORT signal for bearish double top
func generateDoubleTopSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Entry: Neckline break (trough level)
	entryPrice := s.NecklinePrice - (10 * pipSize)

	// Stop Loss: Above the tops
	topPrice := s.LeftShoulderPrice // Average of both tops stored in LeftShoulder
	stopLoss := topPrice + (20 * pipSize)

	// Take Profit: Pattern height projected below neckline
	patternHeight := topPrice - s.NecklinePrice
	takeProfit := s.NecklinePrice - patternHeight

	// Calculate risk-reward
	risk := stopLoss - entryPrice
	reward := entryPrice - takeProfit
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return &TradeSignal{
		Symbol:     symbol,
		Direction:  DirectionShort,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		RiskReward: riskReward,
		Confidence: confidence,
	}
}

// generateDoubleBottomSignal creates a LONG signal for bullish double bottom
func generateDoubleBottomSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Entry: Neckline break (peak level)
	entryPrice := s.NecklinePrice + (10 * pipSize)

	// Stop Loss: Below the bottoms
	bottomPrice := s.LeftShoulderPrice // Average of both bottoms stored in LeftShoulder
	stopLoss := bottomPrice - (20 * pipSize)

	// Take Profit: Pattern height projected above neckline
	patternHeight := s.NecklinePrice - bottomPrice
	takeProfit := s.NecklinePrice + patternHeight

	// Calculate risk-reward
	risk := entryPrice - stopLoss
	reward := takeProfit - entryPrice
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return &TradeSignal{
		Symbol:     symbol,
		Direction:  DirectionLong,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		RiskReward: riskReward,
		Confidence: confidence,
	}
}

// generateTripleTopSignal creates a SHORT signal for bearish triple top
func generateTripleTopSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Entry: Neckline break
	entryPrice := s.NecklinePrice - (10 * pipSize)

	// Stop Loss: Above the tops (use middle top as reference)
	stopLoss := s.HeadPrice + (20 * pipSize)

	// Take Profit: Pattern height projected below neckline (triple patterns often exceed 1:1)
	patternHeight := s.HeadPrice - s.NecklinePrice
	takeProfit := s.NecklinePrice - (patternHeight * 1.2) // 20% bonus target

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
		PatternType: "TRIPLE_TOP",
	}
}

// generateTripleBottomSignal creates a LONG signal for bullish triple bottom
func generateTripleBottomSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	sym := constants.MustGet(symbol)
	pipSize := sym.PipSize

	// Entry: Neckline break
	entryPrice := s.NecklinePrice + (10 * pipSize)

	// Stop Loss: Below the bottoms (use middle bottom as reference)
	stopLoss := s.HeadPrice - (20 * pipSize)

	// Take Profit: Pattern height projected above neckline
	patternHeight := s.NecklinePrice - s.HeadPrice
	takeProfit := s.NecklinePrice + (patternHeight * 1.2) // 20% bonus target

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
		PatternType: "TRIPLE_BOTTOM",
	}
}
