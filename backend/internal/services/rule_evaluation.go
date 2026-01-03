package services

import (
	"context"
	"fmt"
	"strconv"
	"time"
	
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/rules"
)

type RuleEvaluationService struct {
	candleRepo      *repositories.CandleRepository
	indicatorRepo   *repositories.IndicatorRepository
	ruleResultRepo  *repositories.RuleResultRepository
}

func NewRuleEvaluationService(
	candleRepo *repositories.CandleRepository,
	indicatorRepo *repositories.IndicatorRepository,
	ruleResultRepo *repositories.RuleResultRepository,
) *RuleEvaluationService {
	return &RuleEvaluationService{
		candleRepo:     candleRepo,
		indicatorRepo:  indicatorRepo,
		ruleResultRepo: ruleResultRepo,
	}
}

func (s *RuleEvaluationService) EvaluateCandle(
	ctx context.Context,
	candleID uuid.UUID,
) error {
	// 1. Load candle data
	candle, err := s.candleRepo.GetCandleByID(ctx, candleID)
	if err != nil {
		return fmt.Errorf("failed to load candle: %w", err)
	}

	// 2. Load indicators
	indicator, err := s.indicatorRepo.GetIndicatorByCandleID(ctx, candleID)
	if err != nil {
		return fmt.Errorf("failed to load indicators: %w", err)
	}

	// 3. Convert to rule evaluation types
	ruleCandle, err := s.convertToRuleCandle(*candle)
	if err != nil {
		return fmt.Errorf("failed to convert candle: %w", err)
	}

	ruleIndicators, err := s.convertToRuleIndicators(ctx, indicator, candle.TimestampUTC)
	if err != nil {
		return fmt.Errorf("failed to convert indicators: %w", err)
	}

	// 4. Evaluate all rules
	results := rules.EvaluateAllRules(ruleCandle, ruleIndicators)

	// 5. Persist results
	for ruleCode, result := range results {
		err := s.ruleResultRepo.CreateRuleResult(ctx, repositories.RuleResultCreateParams{
			RuleCode:   ruleCode,
			CandleID:   candleID,
			Result:     result.Result,
			Confidence: result.Confidence,
		})
		if err != nil {
			log.Warn().
				Err(err).
				Str("rule_code", string(ruleCode)).
				Str("candle_id", candleID.String()).
				Msg("Failed to persist rule result")
			// Continue with other rules
		}
	}

	log.Info().
		Str("candle_id", candleID.String()).
		Int("rules_evaluated", len(results)).
		Msg("Rule evaluation complete")

	return nil
}

// Helper: Convert repository candle to rules candle
func (s *RuleEvaluationService) convertToRuleCandle(c repositories.Candle) (rules.Candle, error) {
	open, err := strconv.ParseFloat(c.Open, 64)
	if err != nil {
		return rules.Candle{}, err
	}
	high, err := strconv.ParseFloat(c.High, 64)
	if err != nil {
		return rules.Candle{}, err
	}
	low, err := strconv.ParseFloat(c.Low, 64)
	if err != nil {
		return rules.Candle{}, err
	}
	close, err := strconv.ParseFloat(c.Close, 64)
	if err != nil {
		return rules.Candle{}, err
	}

	return rules.Candle{
		Open:         open,
		High:         high,
		Low:          low,
		Close:        close,
		TimestampUTC: c.TimestampUTC,
	}, nil
}


func (s *RuleEvaluationService) convertToRuleIndicators(
	ctx context.Context,
	i *repositories.Indicator,
	candleTimestamp time.Time,
) (rules.Indicators, error) {
	ema20, err := strconv.ParseFloat(i.EMA20, 64)
	if err != nil {
		return rules.Indicators{}, err
	}
	ema50, err := strconv.ParseFloat(i.EMA50, 64)
	if err != nil {
		return rules.Indicators{}, err
	}
	ema200, err := strconv.ParseFloat(i.EMA200, 64)
	if err != nil {
		return rules.Indicators{}, err
	}

	rangeSize, _ := strconv.ParseFloat(i.RangeSize, 64)
	bodySize, _ := strconv.ParseFloat(i.BodySize, 64)
	upperWick, _ := strconv.ParseFloat(i.UpperWick, 64)
	lowerWick, _ := strconv.ParseFloat(i.LowerWick, 64)
	midPrice, _ := strconv.ParseFloat(i.MidPrice, 64)

	// ✅ CRITICAL: Fetch previous EMA50
	var ema50Prev *float64
	prevIndicator, err := s.indicatorRepo.GetPreviousIndicatorByTimestamp(ctx, candleTimestamp)
	if err == nil {
		// Previous indicator exists
		prevEma50, parseErr := strconv.ParseFloat(prevIndicator.EMA50, 64)
		if parseErr == nil {
			ema50Prev = &prevEma50
		}
	}
	// If err != nil, it's the first candle, ema50Prev stays nil (correct)

	return rules.Indicators{
		EMA20:      ema20,
		EMA50:      ema50,
		EMA200:     ema200,
		EMA50Prev:  ema50Prev, // ✅ NOW PROPERLY POPULATED
		RangeSize:  rangeSize,
		BodySize:   bodySize,
		UpperWick:  upperWick,
		LowerWick:  lowerWick,
		MidPrice:   midPrice,
	}, nil
}
