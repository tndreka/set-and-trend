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
	}

	return nil
}

// generateHSSignal creates a SHORT signal for bearish H&S
func generateHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Get pip size for this symbol (JPY pairs use 0.01, others 0.0001)
	pipSize := constants.GetPipSizeForSymbol(symbol)
	
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
		Symbol:      symbol,
		Direction:   DirectionShort,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		RiskReward:  riskReward,
		Confidence:  confidence,
	}
}

// generateIHSSignal creates a LONG signal for bullish inverse H&S
func generateIHSSignal(s *DetectedStructure, symbol string, confidence float64) *TradeSignal {
	// Get pip size for this symbol
	pipSize := constants.GetPipSizeForSymbol(symbol)
	
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
		Symbol:      symbol,
		Direction:   DirectionLong,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		RiskReward:  riskReward,
		Confidence:  confidence,
	}
}