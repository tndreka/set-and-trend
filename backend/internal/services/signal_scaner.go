package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/rules"
)

type SignalScannerService struct {
	candleRepo     *repositories.CandleRepository
	indicatorRepo  *repositories.IndicatorRepository
	ruleResultRepo *repositories.RuleResultRepository
}

func NewSignalScannerService(
	candleRepo *repositories.CandleRepository,
	indicatorRepo *repositories.IndicatorRepository,
	ruleResultRepo *repositories.RuleResultRepository,
) *SignalScannerService {
	return &SignalScannerService{
		candleRepo:     candleRepo,
		indicatorRepo:  indicatorRepo,
		ruleResultRepo: ruleResultRepo,
	}
}

// Signal represents a trading signal
type Signal struct {
	CandleID         uuid.UUID            `json:"candle_id"`
	Timestamp        time.Time            `json:"timestamp"`
	Symbol           string               `json:"symbol"`
	Timeframe        string               `json:"timeframe"`
	Bias             string               `json:"bias"`
	RulesPassed      []string             `json:"rules_passed"`
	RulesFailed      []string             `json:"rules_failed"`
	Confidence       float64              `json:"confidence"`
	SuggestedEntry   *float64             `json:"suggested_entry,omitempty"`
	SuggestedSL      *float64             `json:"suggested_sl,omitempty"`
	SuggestedTP      *float64             `json:"suggested_tp,omitempty"`
	CurrentPrice     float64              `json:"current_price"`
	EMA50            float64              `json:"ema50"`
	EMA200           float64              `json:"ema200"`
	Indicators       map[string]float64   `json:"indicators"`
}

// ScanLatest scans the most recent candle for signals
func (s *SignalScannerService) ScanLatest(ctx context.Context) (*Signal, error) {
	// Get latest candle
	candles, err := s.candleRepo.GetLatestCandles(ctx, 1)
	if err != nil || len(candles) == 0 {
		return nil, fmt.Errorf("no candles found: %w", err)
	}

	candle := candles[0]
	
	// Get indicators for this candle
	indicator, err := s.indicatorRepo.GetIndicatorByCandleID(ctx, candle.ID)
	if err != nil {
		return nil, fmt.Errorf("no indicators found: %w", err)
	}

	// Get rule results
	ruleResults, err := s.ruleResultRepo.GetRuleResultsByCandleID(ctx, candle.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule results: %w", err)
	}

	// Parse data
	closePrice := parseFloat(candle.Close)
	ema50 := parseFloat(indicator.EMA50)
	ema200 := parseFloat(indicator.EMA200)

	// Determine bias from rules
	bias := "neutral"
	rulesPassed := []string{}
	rulesFailed := []string{}
	totalConfidence := 0.0

	for _, result := range ruleResults {
		if result.Result == "PASS" {
			rulesPassed = append(rulesPassed, result.RuleCode)
			totalConfidence += result.ConfidenceScore.InexactFloat64()
			
			// W1_TREND_BULLISH means long bias
			if result.RuleCode == "W1_TREND_BULLISH" {
				bias = "long"
			}
		} else {
			rulesFailed = append(rulesFailed, result.RuleCode)
		}
	}

	// Calculate average confidence
	avgConfidence := 0.0
	if len(rulesPassed) > 0 {
		avgConfidence = totalConfidence / float64(len(rulesPassed))
	}

	// Generate suggested levels based on bias
	var suggestedEntry, suggestedSL, suggestedTP *float64
	
	if bias == "long" && ema50 > 0 {
		// Entry: Current price or slight pullback to EMA50
		entry := closePrice
		suggestedEntry = &entry
		
		// SL: Below EMA50 or recent swing low
		sl := ema50 * 0.995 // 0.5% below EMA50
		suggestedSL = &sl
		
		// TP: 2-3x risk
		riskPips := (entry - sl) / 0.0001
		tp := entry + (riskPips * 3 * 0.0001) // 3:1 RR
		suggestedTP = &tp
	}

	signal := &Signal{
		CandleID:       candle.ID,
		Timestamp:      candle.TimestampUTC,
		Symbol:         "EURUSD",
		Timeframe:      "W1",
		Bias:           bias,
		RulesPassed:    rulesPassed,
		RulesFailed:    rulesFailed,
		Confidence:     avgConfidence,
		SuggestedEntry: suggestedEntry,
		SuggestedSL:    suggestedSL,
		SuggestedTP:    suggestedTP,
		CurrentPrice:   closePrice,
		EMA50:          ema50,
		EMA200:         ema200,
		Indicators: map[string]float64{
			"ema20":       parseFloat(indicator.EMA20),
			"ema50":       ema50,
			"ema200":      ema200,
			"range_size":  parseFloat(indicator.RangeSize),
			"body_size":   parseFloat(indicator.BodySize),
		},
	}

	return signal, nil
}

// Helper to parse float from string
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
