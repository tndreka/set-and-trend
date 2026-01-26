package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

// PatternService orchestrates pattern detection with database operations
type PatternService struct {
	queries *db.Queries
}

// NewPatternService creates a new PatternService
func NewPatternService(queries *db.Queries) *PatternService {
	return &PatternService{queries: queries}
}

// DetectionResult holds the result of pattern detection
type DetectionResult struct {
	Structure *patterns.DetectedStructure
	Context   patterns.MarketContext
	Signal    *patterns.TradeSignal
	PatternID uuid.UUID
	SignalID  uuid.UUID
}

// DetectPatternsD1 detects H&S patterns on D1 timeframe
func (s *PatternService) DetectPatternsD1(ctx context.Context, windowSize int, minConfidence float64, minRR float64) ([]DetectionResult, error) {
	// Load candles from DB
	dbCandles, err := s.queries.GetCandlesD1Last(ctx, int32(windowSize))
	if err != nil {
		return nil, fmt.Errorf("failed to load D1 candles: %w", err)
	}

	if len(dbCandles) < 20 {
		return nil, fmt.Errorf("insufficient data: need at least 20 candles, got %d", len(dbCandles))
	}

	// Reverse to chronological order (DB returns DESC)
	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[len(dbCandles)-1-i] = dbCandleToPatternCandle(c, len(dbCandles)-1-i)
	}

	return s.detectPatterns(ctx, candles, patterns.TF_D1, minConfidence, minRR)
}

// DetectPatternsH4 detects H&S patterns on H4 timeframe
func (s *PatternService) DetectPatternsH4(ctx context.Context, windowSize int, minConfidence float64, minRR float64) ([]DetectionResult, error) {
	dbCandles, err := s.queries.GetCandlesH4Last(ctx, int32(windowSize))
	if err != nil {
		return nil, fmt.Errorf("failed to load H4 candles: %w", err)
	}

	if len(dbCandles) < 20 {
		return nil, fmt.Errorf("insufficient data: need at least 20 candles, got %d", len(dbCandles))
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[len(dbCandles)-1-i] = dbCandleH4ToPatternCandle(c, len(dbCandles)-1-i)
	}

	return s.detectPatterns(ctx, candles, patterns.TF_H4, minConfidence, minRR)
}

// DetectPatternsH1 detects H&S patterns on H1 timeframe
func (s *PatternService) DetectPatternsH1(ctx context.Context, windowSize int, minConfidence float64, minRR float64) ([]DetectionResult, error) {
	dbCandles, err := s.queries.GetCandlesH1Last(ctx, int32(windowSize))
	if err != nil {
		return nil, fmt.Errorf("failed to load H1 candles: %w", err)
	}

	if len(dbCandles) < 20 {
		return nil, fmt.Errorf("insufficient data: need at least 20 candles, got %d", len(dbCandles))
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[len(dbCandles)-1-i] = dbCandleH1ToPatternCandle(c, len(dbCandles)-1-i)
	}

	return s.detectPatterns(ctx, candles, patterns.TF_H1, minConfidence, minRR)
}

// detectPatterns is the core detection logic shared across timeframes
func (s *PatternService) detectPatterns(ctx context.Context, candles []patterns.Candle, tf patterns.Timeframe, minConfidence float64, minRR float64) ([]DetectionResult, error) {
	var results []DetectionResult
	lookback := patterns.LookbackByTimeframe(tf)

	// Detect H&S (bearish)
	hsStructure := patterns.DetectHeadAndShoulders(candles, lookback)
	if hsStructure != nil {
		result, err := s.processDetectedStructure(ctx, hsStructure, candles, tf, minConfidence, minRR)
		if err == nil && result != nil {
			results = append(results, *result)
		}
	}

	// Detect IHS (bullish)
	ihsStructure := patterns.DetectInverseHeadAndShoulders(candles, lookback)
	if ihsStructure != nil {
		result, err := s.processDetectedStructure(ctx, ihsStructure, candles, tf, minConfidence, minRR)
		if err == nil && result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// processDetectedStructure processes a detected structure, calculates context, and optionally saves
func (s *PatternService) processDetectedStructure(ctx context.Context, structure *patterns.DetectedStructure, candles []patterns.Candle, tf patterns.Timeframe, minConfidence float64, minRR float64) (*DetectionResult, error) {
	// Calculate market context
	marketCtx := patterns.DetermineMarketContext(candles)

	// Calculate final confidence with context
	finalConfidence := patterns.FinalConfidence(structure, marketCtx)
	structure.FinalConfidence = finalConfidence

	// Filter by minimum confidence
	if finalConfidence < minConfidence {
		return nil, nil
	}

	// Generate trade signal
	signal := patterns.GenerateTradeSignal(structure, "EURUSD", finalConfidence)

	// Validate signal
	if !patterns.ValidateSignal(signal, minRR, minConfidence) {
		return nil, nil
	}

	// Save pattern detection to DB
	patternID, err := s.savePatternDetection(ctx, structure, marketCtx, tf, len(candles)-1)
	if err != nil {
		return nil, fmt.Errorf("failed to save pattern: %w", err)
	}

	// Save signal to DB
	signalID, err := s.saveSignal(ctx, patternID, signal, tf)
	if err != nil {
		return nil, fmt.Errorf("failed to save signal: %w", err)
	}

	return &DetectionResult{
		Structure: structure,
		Context:   marketCtx,
		Signal:    signal,
		PatternID: patternID,
		SignalID:  signalID,
	}, nil
}

// savePatternDetection saves a detected pattern to the database
func (s *PatternService) savePatternDetection(ctx context.Context, structure *patterns.DetectedStructure, marketCtx patterns.MarketContext, tf patterns.Timeframe, candleIdx int) (uuid.UUID, error) {
	params := db.InsertPatternDetectionParams{
		Symbol:             "EURUSD",
		Timeframe:          db.RuleTimeframe(tf),
		DetectedCandleIdx:  int32(candleIdx),
		LeftShoulderPrice:  decimal.NewFromFloat(structure.LeftShoulderPrice),
		LeftShoulderIdx:    int32(structure.LeftShoulderIdx),
		HeadPrice:          decimal.NewFromFloat(structure.HeadPrice),
		HeadIdx:            int32(structure.HeadIdx),
		RightShoulderPrice: decimal.NewFromFloat(structure.RightShoulderPrice),
		RightShoulderIdx:   int32(structure.RightShoulderIdx),
		NecklinePrice:      decimal.NewFromFloat(structure.NecklinePrice),
		NecklineIdx:        int32(structure.NecklineIdx),
		ShoulderSymmetry:   decimal.NewFromFloat(structure.ShoulderSymmetry),
		HeadProminence:     decimal.NewFromFloat(structure.HeadProminence),
		TimeSymmetry:       decimal.NewFromFloat(structure.TimeSymmetry),
		VolumeProfile:      decimal.NewFromFloat(structure.VolumeProfile),
		NecklineQuality:    decimal.NewFromFloat(structure.NecklineQuality),
		ContextTrend:       marketCtx.Trend,
		VolatilityRegime:   marketCtx.VolatilityRegime,
		ContextDistEma200:  decimal.NewFromFloat(marketCtx.DistanceFromEMA200),
		RecentSwings:       int32(marketCtx.RecentSwings),
		OverallConfidence:  decimal.NewFromFloat(structure.OverallConfidence),
		FinalConfidence:    decimal.NewFromFloat(structure.FinalConfidence),
		PatternType:        structure.PatternType,
	}

	result, err := s.queries.InsertPatternDetection(ctx, params)
	if err != nil {
		return uuid.Nil, err
	}

	return result.ID, nil
}

// saveSignal saves a trade signal to the database
func (s *PatternService) saveSignal(ctx context.Context, patternID uuid.UUID, signal *patterns.TradeSignal, tf patterns.Timeframe) (uuid.UUID, error) {
	direction := db.TradeDirectionSHORT
	if signal.Direction == patterns.DirectionLong {
		direction = db.TradeDirectionLONG
	}

	// Execution timeframe is typically one level down
	execTF := db.RuleTimeframeH4
	if tf == patterns.TF_H4 {
		execTF = db.RuleTimeframeH1
	} else if tf == patterns.TF_H1 {
		execTF = db.RuleTimeframeH1
	}

	params := db.InsertSignalParams{
		PatternDetectionID: patternID,
		Symbol:             signal.Symbol,
		Timeframe:          db.RuleTimeframe(tf),
		ExecutionTimeframe: execTF,
		Direction:          direction,
		DetectedPrice:      decimal.NewFromFloat(signal.EntryPrice),
		TheoreticalSl:      decimal.NewFromFloat(signal.StopLoss),
		TheoreticalTp:      decimal.NewFromFloat(signal.TakeProfit),
		ProjectedRr:        decimal.NewFromFloat(signal.RiskReward),
		Confidence:         decimal.NewFromFloat(signal.Confidence),
		Status:             "PENDING",
	}

	result, err := s.queries.InsertSignal(ctx, params)
	if err != nil {
		return uuid.Nil, err
	}

	return result.ID, nil
}

// Helper functions to convert DB types to pattern types
func dbCandleToPatternCandle(c db.CandlesD1, idx int) patterns.Candle {
	return patterns.Candle{
		Index:     idx,
		Timestamp: c.TimestampUtc.Time,
		Open:      c.Open.InexactFloat64(),
		High:      c.High.InexactFloat64(),
		Low:       c.Low.InexactFloat64(),
		Close:     c.Close.InexactFloat64(),
		Volume:    volumeToInt64(c.Volume),
	}
}

func dbCandleH4ToPatternCandle(c db.CandlesH4, idx int) patterns.Candle {
	return patterns.Candle{
		Index:     idx,
		Timestamp: c.TimestampUtc.Time,
		Open:      c.Open.InexactFloat64(),
		High:      c.High.InexactFloat64(),
		Low:       c.Low.InexactFloat64(),
		Close:     c.Close.InexactFloat64(),
		Volume:    volumeToInt64(c.Volume),
	}
}

func dbCandleH1ToPatternCandle(c db.CandlesH1, idx int) patterns.Candle {
	return patterns.Candle{
		Index:     idx,
		Timestamp: c.TimestampUtc.Time,
		Open:      c.Open.InexactFloat64(),
		High:      c.High.InexactFloat64(),
		Low:       c.Low.InexactFloat64(),
		Close:     c.Close.InexactFloat64(),
		Volume:    volumeToInt64(c.Volume),
	}
}

func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}
