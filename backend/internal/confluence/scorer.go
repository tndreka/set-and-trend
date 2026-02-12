package confluence

import (
	"math"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/patterns"
)

// ConfluenceScorer calculates multi-timeframe confluence scores
type ConfluenceScorer struct {
	config ConfluenceConfig
}

// NewConfluenceScorer creates a new scorer with config
func NewConfluenceScorer(config ConfluenceConfig) *ConfluenceScorer {
	return &ConfluenceScorer{
		config: config,
	}
}

// CalculateConfluence analyzes all timeframes and returns total score
func (cs *ConfluenceScorer) CalculateConfluence(
	weeklyCandles []patterns.Candle,
	dailyCandles []patterns.Candle,
	h4Candles []patterns.Candle,
	h2Candles []patterns.Candle,
	h1Candles []patterns.Candle,
	m30Candles []patterns.Candle,
	symbol string,
) (*ChecklistScore, *patterns.TradeSignal) {

	score := &ChecklistScore{}

	// === WEEKLY ANALYSIS (30.6% normalized) ===
	if len(weeklyCandles) >= 55 { // Need 50 for EMA200 + 5 buffer
		wCtx := patterns.DetermineMarketContext(weeklyCandles)

		// Trend in Favor (5.6%)
		score.WeeklyFavor = cs.calculateTrendScore(wCtx, weeklyCandles) * 0.056

		// At/Rejected AOI (5.6%)
		score.WeeklyAOI = cs.calculateAOIScore(weeklyCandles) * 0.056

		// Touching EMA200 (2.8%)
		score.WeeklyEMA = cs.calculateEMATouchScore(weeklyCandles, 200) * 0.028

		// Candlestick Rejection (5.6%)
		score.WeeklyReject = cs.calculateRejectionScore(weeklyCandles) * 0.056

		// Structure Rejection (5.6%)
		score.WeeklyStruct = cs.calculateStructureScore(weeklyCandles) * 0.056

		// Pattern Detection (5.6%) - H&S or IHS
		wPattern := cs.detectBestPattern(weeklyCandles, symbol)
		if wPattern != nil {
			score.WeeklyPattern = 0.056
		}
	}

	// === DAILY ANALYSIS (30.6% normalized) ===
	if len(dailyCandles) >= 55 {
		dCtx := patterns.DetermineMarketContext(dailyCandles)

		score.DailyFavor = cs.calculateTrendScore(dCtx, dailyCandles) * 0.056
		score.DailyAOI = cs.calculateAOIScore(dailyCandles) * 0.056
		score.DailyEMA = cs.calculateEMATouchScore(dailyCandles, 200) * 0.028
		score.DailyReject = cs.calculateRejectionScore(dailyCandles) * 0.056
		score.DailyStruct = cs.calculateStructureScore(dailyCandles) * 0.056

		dPattern := cs.detectBestPattern(dailyCandles, symbol)
		if dPattern != nil {
			score.DailyPattern = 0.056
		}
	}

	// === 4H ANALYSIS (25% normalized) ===
	if len(h4Candles) >= 45 {
		h4Ctx := patterns.DetermineMarketContext(h4Candles)

		score.H4Favor = cs.calculateTrendScore(h4Ctx, h4Candles) * 0.056
		score.H4EMA = cs.calculateEMATouchScore(h4Candles, 200) * 0.028
		score.H4Reject = cs.calculateRejectionScore(h4Candles) * 0.056
		score.H4Struct = cs.calculateStructureScore(h4Candles) * 0.056

		h4Pattern := cs.detectBestPattern(h4Candles, symbol)
		if h4Pattern != nil {
			score.H4Pattern = 0.056
		}
	}

	// === LOWER TFs (13.9% normalized) ===
	// 2H Psychological Level (2.8%)
	if len(h2Candles) >= 20 {
		score.H2Psych = cs.calculatePsychLevelScore(h2Candles, symbol) * 0.028
	}

	// 1H Shift of Structure (5.6%)
	if len(h1Candles) >= 20 {
		score.H1SOS = cs.calculateSOSScore(h1Candles) * 0.056
	}

	// M30 Engulfing Pattern (5.6%)
	if len(m30Candles) >= 10 {
		score.M30Engulf = cs.calculateEngulfingScore(m30Candles) * 0.056
	}

	// Calculate total percentage (0-100%)
	score.TotalPercent = (score.WeeklyFavor + score.WeeklyAOI + score.WeeklyEMA +
		score.WeeklyReject + score.WeeklyStruct + score.WeeklyPattern +
		score.DailyFavor + score.DailyAOI + score.DailyEMA +
		score.DailyReject + score.DailyStruct + score.DailyPattern +
		score.H4Favor + score.H4EMA + score.H4Reject +
		score.H4Struct + score.H4Pattern +
		score.H2Psych + score.H1SOS + score.M30Engulf) * 100

	// Generate signal only if confluence >= threshold
	var signal *patterns.TradeSignal
	if score.TotalPercent >= cs.config.MinConfluencePercent {
		signal = cs.generateSignalFromConfluence(score, weeklyCandles, dailyCandles, h4Candles, symbol)
	}

	return score, signal
}

// IsValidConfluence checks if score meets minimum threshold
func (cs *ConfluenceScorer) IsValidConfluence(score *ChecklistScore) bool {
	return score != nil && score.TotalPercent >= cs.config.MinConfluencePercent
}

// Helper: Calculate trend score (1.0 = in favor, 0.0 = against)
func (cs *ConfluenceScorer) calculateTrendScore(ctx patterns.MarketContext, candles []patterns.Candle) float64 {
	// Strong trend alignment = 1.0, weak = 0.5, against = 0.0
	switch ctx.Trend {
	case patterns.TrendStrongUp, patterns.TrendStrongDown:
		return 1.0
	case patterns.TrendWeakUp, patterns.TrendWeakDown:
		return 0.5
	default:
		return 0.0
	}
}

// Helper: Calculate AOI (Area of Interest) score
func (cs *ConfluenceScorer) calculateAOIScore(candles []patterns.Candle) float64 {
	if len(candles) < 20 {
		return 0.0
	}

	// Find recent swing high/low
	highs := patterns.FindSwingHighs(candles, 3)
	lows := patterns.FindSwingLows(candles, 3)

	if len(highs) == 0 && len(lows) == 0 {
		return 0.0
	}

	current := candles[len(candles)-1].Close
	pipSize := 0.0001 // Default, should use constants

	// Check if price is within 20 pips of recent swing
	for _, h := range highs {
		if math.Abs(current-h.Price) <= 20*pipSize {
			return 1.0
		}
	}
	for _, l := range lows {
		if math.Abs(current-l.Price) <= 20*pipSize {
			return 1.0
		}
	}

	return 0.0
}

// Helper: Calculate EMA touch score
func (cs *ConfluenceScorer) calculateEMATouchScore(candles []patterns.Candle, period int) float64 {
	if len(candles) < period {
		return 0.0
	}

	ema := patterns.CalculateEMA(candles, period)
	if ema <= 0 {
		return 0.0
	}

	current := candles[len(candles)-1]
	pipSize := 0.0001 // Default

	// Check if price touched EMA (high or low within 20 pips)
	if math.Abs(current.High-ema) <= 20*pipSize || math.Abs(current.Low-ema) <= 20*pipSize {
		return 1.0
	}

	return 0.0
}

// Helper: Calculate candlestick rejection score
func (cs *ConfluenceScorer) calculateRejectionScore(candles []patterns.Candle) float64 {
	if len(candles) < 2 {
		return 0.0
	}

	current := candles[len(candles)-1]
	body := math.Abs(current.Close - current.Open)
	range_ := current.High - current.Low

	if range_ == 0 {
		return 0.0
	}

	upperWick := current.High - math.Max(current.Open, current.Close)
	lowerWick := math.Min(current.Open, current.Close) - current.Low

	// Pinbar: wick >= 2x body
	if upperWick >= 2*body || lowerWick >= 2*body {
		return 1.0
	}

	// Hammer/shooting star: small body, long wick
	if body <= range_*0.3 && (upperWick >= range_*0.6 || lowerWick >= range_*0.6) {
		return 1.0
	}

	return 0.0
}

// Helper: Calculate structure rejection score
func (cs *ConfluenceScorer) calculateStructureScore(candles []patterns.Candle) float64 {
	if len(candles) < 10 {
		return 0.0
	}

	// Find recent structure points
	highs := patterns.FindSwingHighs(candles, 2)
	lows := patterns.FindSwingLows(candles, 2)

	if len(highs) == 0 && len(lows) == 0 {
		return 0.0
	}

	current := candles[len(candles)-1]
	pipSize := 0.0001 // Default

	// Check rejection from recent structure
	for _, h := range highs {
		if math.Abs(current.High-h.Price) <= 10*pipSize {
			return 1.0
		}
	}
	for _, l := range lows {
		if math.Abs(current.Low-l.Price) <= 10*pipSize {
			return 1.0
		}
	}

	return 0.0
}

// Helper: Detect best pattern and return signal
func (cs *ConfluenceScorer) detectBestPattern(candles []patterns.Candle, symbol string) *patterns.TradeSignal {
	lookback := 3
	if len(candles) > 100 {
		lookback = 5
	}

	// Try H&S first
	hs := patterns.DetectHeadAndShoulders(candles, lookback)
	if hs != nil {
		ctx := patterns.DetermineMarketContext(candles)
		confidence := patterns.FinalConfidence(hs, ctx)
		return patterns.GenerateTradeSignal(hs, symbol, confidence)
	}

	// Try IHS
	ihs := patterns.DetectInverseHeadAndShoulders(candles, lookback)
	if ihs != nil {
		ctx := patterns.DetermineMarketContext(candles)
		confidence := patterns.FinalConfidence(ihs, ctx)
		return patterns.GenerateTradeSignal(ihs, symbol, confidence)
	}

	// Try Double Top (bearish)
	dt := patterns.DetectDoubleTop(candles, 50, symbol)
	if dt != nil {
		ctx := patterns.DetermineMarketContext(candles)
		confidence := patterns.FinalConfidence(dt, ctx)
		return patterns.GenerateTradeSignal(dt, symbol, confidence)
	}

	// Try Double Bottom (bullish)
	db := patterns.DetectDoubleBottom(candles, 50, symbol)
	if db != nil {
		ctx := patterns.DetermineMarketContext(candles)
		confidence := patterns.FinalConfidence(db, ctx)
		return patterns.GenerateTradeSignal(db, symbol, confidence)
	}

	return nil
}

// Helper: Calculate psychological level score
func (cs *ConfluenceScorer) calculatePsychLevelScore(candles []patterns.Candle, symbol string) float64 {
	if len(candles) < 1 {
		return 0.0
	}

	sym := constants.MustGet(symbol)
	price := candles[len(candles)-1].Close

	// Round to nearest 100 pips for this symbol
	psychLevel := sym.PipsToPrice(100)
	rounded := math.Round(price/psychLevel) * psychLevel

	// Within 5 pips of psych level
	if math.Abs(price-rounded) <= sym.PipsToPrice(5) {
		return 1.0
	}

	return 0.0
}

// Helper: Calculate Shift of Structure score
func (cs *ConfluenceScorer) calculateSOSScore(candles []patterns.Candle) float64 {
	if len(candles) < 20 {
		return 0.0
	}

	// Find previous structure
	prevHighs := patterns.FindSwingHighs(candles[:len(candles)-5], 2)
	prevLows := patterns.FindSwingLows(candles[:len(candles)-5], 2)

	if len(prevHighs) == 0 && len(prevLows) == 0 {
		return 0.0
	}

	current := candles[len(candles)-1]

	// Bullish SOS: broke above previous high
	for _, h := range prevHighs {
		if current.High > h.Price {
			return 1.0
		}
	}

	// Bearish SOS: broke below previous low
	for _, l := range prevLows {
		if current.Low < l.Price {
			return 1.0
		}
	}

	return 0.0
}

// Helper: Calculate engulfing pattern score
func (cs *ConfluenceScorer) calculateEngulfingScore(candles []patterns.Candle) float64 {
	if len(candles) < 3 {
		return 0.0
	}

	prev1 := candles[len(candles)-2]
	_ = candles[len(candles)-3] // prev2 reserved for future 3-candle patterns
	current := candles[len(candles)-1]

	// Bullish engulfing
	bullEngulf := prev1.Close < prev1.Open && // Prev bearish
		current.Open < prev1.Close && // Current open below prev close
		current.Close > prev1.Open // Current close above prev open

	// Bearish engulfing
	bearEngulf := prev1.Close > prev1.Open && // Prev bullish
		current.Open > prev1.Close && // Current open above prev close
		current.Close < prev1.Open // Current close below prev open

	if bullEngulf || bearEngulf {
		return 1.0
	}

	return 0.0
}

// Generate signal from highest timeframe pattern with confluence
func (cs *ConfluenceScorer) generateSignalFromConfluence(
	score *ChecklistScore,
	weeklyCandles []patterns.Candle,
	dailyCandles []patterns.Candle,
	h4Candles []patterns.Candle,
	symbol string,
) *patterns.TradeSignal {

	// Priority: Weekly > Daily > 4H
	var signal *patterns.TradeSignal

	// Check weekly first
	if score.WeeklyPattern > 0 && len(weeklyCandles) > 0 {
		signal = cs.detectBestPattern(weeklyCandles, symbol)
		if signal != nil {
			// Enhance signal with confluence info
			signal.Confidence = score.TotalPercent / 100
			return signal
		}
	}

	// Then daily
	if score.DailyPattern > 0 && len(dailyCandles) > 0 {
		signal = cs.detectBestPattern(dailyCandles, symbol)
		if signal != nil {
			signal.Confidence = score.TotalPercent / 100
			return signal
		}
	}

	// Then 4H
	if score.H4Pattern > 0 && len(h4Candles) > 0 {
		signal = cs.detectBestPattern(h4Candles, symbol)
		if signal != nil {
			signal.Confidence = score.TotalPercent / 100
			return signal
		}
	}

	return nil
}

// CalculateConfluenceFromCandleBuffers uses pre-aggregated candle buffers
// This is for WebSocket live trading where you maintain rolling buffers
func (cs *ConfluenceScorer) CalculateConfluenceFromBuffers(
	weeklyBuffer *CandleBuffer,
	dailyBuffer *CandleBuffer,
	h4Buffer *CandleBuffer,
	h2Buffer *CandleBuffer,
	h1Buffer *CandleBuffer,
	m30Buffer *CandleBuffer,
	symbol string,
) (*ChecklistScore, *patterns.TradeSignal) {

	// Get slices from buffers (thread-safe)
	weeklyCandles := weeklyBuffer.GetCandles()
	dailyCandles := dailyBuffer.GetCandles()
	h4Candles := h4Buffer.GetCandles()
	h2Candles := h2Buffer.GetCandles()
	h1Candles := h1Buffer.GetCandles()
	m30Candles := m30Buffer.GetCandles()

	return cs.CalculateConfluence(
		weeklyCandles,
		dailyCandles,
		h4Candles,
		h2Candles,
		h1Candles,
		m30Candles,
		symbol,
	)
}

// UpdateConfluenceOnNewCandle recalculates only affected timeframe
// Called when WebSocket receives new candle close
func (cs *ConfluenceScorer) UpdateConfluenceOnNewCandle(
	tf Timeframe,
	candles []patterns.Candle,
	currentScore *ChecklistScore,
	symbol string,
) (*ChecklistScore, bool) {

	// Only recalculate the specific timeframe component
	switch tf {
	case TF_W1:
		// Recalculate weekly components only
		if len(candles) >= 55 {
			wCtx := patterns.DetermineMarketContext(candles)
			currentScore.WeeklyFavor = cs.calculateTrendScore(wCtx, candles) * 0.056
			currentScore.WeeklyAOI = cs.calculateAOIScore(candles) * 0.056
			currentScore.WeeklyEMA = cs.calculateEMATouchScore(candles, 200) * 0.028
			currentScore.WeeklyReject = cs.calculateRejectionScore(candles) * 0.056
			currentScore.WeeklyStruct = cs.calculateStructureScore(candles) * 0.056

			wPattern := cs.detectBestPattern(candles, symbol)
			if wPattern != nil {
				currentScore.WeeklyPattern = 0.056
			} else {
				currentScore.WeeklyPattern = 0
			}
		}
		// Similar for D1, H4, etc...
	}

	// Recalculate total
	currentScore.TotalPercent = cs.recalculateTotal(currentScore)

	isValid := currentScore.TotalPercent >= cs.config.MinConfluencePercent
	return currentScore, isValid
}

func (cs *ConfluenceScorer) recalculateTotal(score *ChecklistScore) float64 {
	return (score.WeeklyFavor + score.WeeklyAOI + score.WeeklyEMA +
		score.WeeklyReject + score.WeeklyStruct + score.WeeklyPattern +
		score.DailyFavor + score.DailyAOI + score.DailyEMA +
		score.DailyReject + score.DailyStruct + score.DailyPattern +
		score.H4Favor + score.H4EMA + score.H4Reject +
		score.H4Struct + score.H4Pattern +
		score.H2Psych + score.H1SOS + score.M30Engulf) * 100
}
