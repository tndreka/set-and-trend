package mtf

import (
	"time"

	"set-and-trend/backend/internal/candlestick"
	"set-and-trend/backend/internal/database"
	"set-and-trend/backend/internal/indicators"
	"set-and-trend/backend/internal/levels"
	"set-and-trend/backend/internal/smc"
)

// TimeframeContext holds SMC analysis for a single timeframe
type TimeframeContext struct {
	Timeframe        database.Timeframe
	Candles          []database.Candle
	
	// Trend
	Trend            int     // 1=bullish, -1=bearish, 0=ranging
	EMA50            float64
	EMA200           float64
	
	// SMC Elements
	ActiveFVGs       []smc.FVG
	ActiveOBs        []smc.OrderBlock
	LiquidityZones   []smc.LiquidityZone
	MarketStructure  *smc.MarketStructure
	RecentBOS        *smc.StructureShift
	RecentCHoCH      *smc.StructureShift
	
	// Candlestick Patterns
	RecentPinBars    []candlestick.PinBar
	RecentEngulfing  []candlestick.EngulfingPattern
	
	// Current state
	LastClose        float64
	AtFVG            *smc.FVG
	AtOrderBlock     *smc.OrderBlock
	NearPsychLevel   *levels.PsychLevel
}

// MTFContext holds multi-timeframe analysis
type MTFContext struct {
	Symbol    string
	Weekly    *TimeframeContext
	Daily     *TimeframeContext
	H4        *TimeframeContext
	Timestamp time.Time
}

// MTFContextBuilder builds multi-timeframe context
type MTFContextBuilder struct {
	Loader       *database.SQLiteLoader
	SwingLookback int
	MinFVGPercent float64
	MinOBPercent  float64
}

// NewMTFContextBuilder creates a new builder with default settings
func NewMTFContextBuilder(loader *database.SQLiteLoader) *MTFContextBuilder {
	return &MTFContextBuilder{
		Loader:        loader,
		SwingLookback: 5,
		MinFVGPercent: 0.001,
		MinOBPercent:  0.005,
	}
}

// BuildContext builds the complete multi-timeframe context
func (b *MTFContextBuilder) BuildContext(symbol string, currentTime time.Time) *MTFContext {
	ctx := &MTFContext{
		Symbol:    symbol,
		Timestamp: currentTime,
	}

	// Build context for each timeframe
	// Weekly: look back ~52 weeks (1 year)
	weeklyRange := &database.DateRange{
		Start: currentTime.AddDate(-1, 0, 0),
		End:   currentTime,
	}
	ctx.Weekly = b.buildTimeframeContext(symbol, database.TimeframeDaily, weeklyRange, 7) // Use daily and resample

	// Daily: look back ~6 months
	dailyRange := &database.DateRange{
		Start: currentTime.AddDate(0, -6, 0),
		End:   currentTime,
	}
	ctx.Daily = b.buildTimeframeContext(symbol, database.TimeframeDaily, dailyRange, 1)

	// H4: look back ~3 months
	h4Range := &database.DateRange{
		Start: currentTime.AddDate(0, -3, 0),
		End:   currentTime,
	}
	ctx.H4 = b.buildTimeframeContext(symbol, database.TimeframeH4, h4Range, 1)

	return ctx
}

// BuildContextFromCandles builds context from pre-loaded candles (no DB access)
func (b *MTFContextBuilder) BuildContextFromCandles(symbol string, h4Candles []database.Candle, dailyCandles []database.Candle) *MTFContext {
	ctx := &MTFContext{
		Symbol: symbol,
	}

	if len(h4Candles) > 0 {
		ctx.Timestamp = h4Candles[len(h4Candles)-1].Timestamp
		ctx.H4 = b.analyzeCandles(h4Candles, database.TimeframeH4, symbol)
	}

	if len(dailyCandles) > 0 {
		ctx.Daily = b.analyzeCandles(dailyCandles, database.TimeframeDaily, symbol)
		
		// Create weekly by resampling daily (every 5 days)
		weeklyCandles := resampleToWeekly(dailyCandles)
		if len(weeklyCandles) >= 20 {
			ctx.Weekly = b.analyzeCandles(weeklyCandles, database.TimeframeDaily, symbol) // Use daily TF type but weekly data
		}
	}

	return ctx
}

// buildTimeframeContext loads data and builds context for a single timeframe
func (b *MTFContextBuilder) buildTimeframeContext(symbol string, tf database.Timeframe, dateRange *database.DateRange, resampleFactor int) *TimeframeContext {
	candles, err := b.Loader.GetCandles(symbol, tf, dateRange)
	if err != nil || len(candles) < 50 {
		return nil
	}

	// Resample if needed (for weekly from daily)
	if resampleFactor > 1 {
		candles = resampleCandles(candles, resampleFactor)
	}

	return b.analyzeCandles(candles, tf, symbol)
}

// analyzeCandles performs SMC analysis on candle data
func (b *MTFContextBuilder) analyzeCandles(candles []database.Candle, tf database.Timeframe, symbol string) *TimeframeContext {
	if len(candles) < 50 {
		return nil
	}

	ctx := &TimeframeContext{
		Timeframe: tf,
		Candles:   candles,
		LastClose: candles[len(candles)-1].Close,
	}

	// Calculate EMAs for trend
	ema50 := indicators.NewEMA(50)
	ema200 := indicators.NewEMA(200)
	ema50.Calculate(candles)
	ema200.Calculate(candles)
	
	if ema50.IsValid() {
		ctx.EMA50 = ema50.GetCurrent()
	}
	if ema200.IsValid() {
		ctx.EMA200 = ema200.GetCurrent()
	}
	
	// Determine trend
	if ctx.EMA50 > ctx.EMA200 {
		ctx.Trend = 1
	} else if ctx.EMA50 < ctx.EMA200 {
		ctx.Trend = -1
	}

	// Find swing points for SMC analysis
	swingHighs := findSwingHighs(candles, b.SwingLookback)
	swingLows := findSwingLows(candles, b.SwingLookback)
	
	// Convert to smc.SwingPoint
	smcHighs := convertToSMCSwings(swingHighs, true)
	smcLows := convertToSMCSwings(swingLows, false)

	// Detect FVGs
	allFVGs := smc.DetectFVGs(candles, b.MinFVGPercent)
	allFVGs = smc.UpdateFVGMitigation(allFVGs, candles, 0)
	ctx.ActiveFVGs = smc.GetActiveFVGs(allFVGs)

	// Detect Order Blocks
	allOBs := smc.DetectOrderBlocksWithSwings(candles, smcHighs, smcLows)
	allOBs = smc.UpdateOrderBlockStatus(allOBs, candles, 0)
	ctx.ActiveOBs = smc.GetActiveOrderBlocks(allOBs)

	// Detect Liquidity Zones
	allLiqZones := smc.DetectLiquidityZones(candles, smcHighs, smcLows, 0.001)
	allLiqZones = smc.UpdateLiquiditySweeps(allLiqZones, candles, 0)
	ctx.LiquidityZones = smc.GetActiveZones(allLiqZones)

	// Analyze Market Structure
	ctx.MarketStructure = smc.AnalyzeMarketStructure(candles, b.SwingLookback)
	if ctx.MarketStructure != nil {
		ctx.RecentBOS = smc.GetLastBOS(ctx.MarketStructure.RecentShifts)
		ctx.RecentCHoCH = smc.GetLastCHoCH(ctx.MarketStructure.RecentShifts)
	}

	// Detect Candlestick Patterns (last 20 bars)
	recentCandles := candles
	if len(candles) > 20 {
		recentCandles = candles[len(candles)-20:]
	}
	ctx.RecentPinBars = candlestick.DetectPinBars(recentCandles, 0.60, 0.30)
	ctx.RecentEngulfing = candlestick.DetectEngulfing(recentCandles)

	// Check current price position
	currentPrice := ctx.LastClose
	ctx.AtFVG = smc.IsPriceInFVG(currentPrice, ctx.ActiveFVGs)
	ctx.AtOrderBlock = smc.IsPriceAtOrderBlock(currentPrice, ctx.ActiveOBs)
	ctx.NearPsychLevel = levels.GetNearPsychLevelInfo(currentPrice, symbol, 0.002)

	return ctx
}

// Helper: find swing highs
func findSwingHighs(candles []database.Candle, lookback int) []SwingPoint {
	var swings []SwingPoint
	for i := lookback; i < len(candles)-lookback; i++ {
		isHigh := true
		for j := 1; j <= lookback; j++ {
			if candles[i-j].High >= candles[i].High || candles[i+j].High >= candles[i].High {
				isHigh = false
				break
			}
		}
		if isHigh {
			swings = append(swings, SwingPoint{
				Index:  i,
				Price:  candles[i].High,
				IsHigh: true,
			})
		}
	}
	return swings
}

// Helper: find swing lows
func findSwingLows(candles []database.Candle, lookback int) []SwingPoint {
	var swings []SwingPoint
	for i := lookback; i < len(candles)-lookback; i++ {
		isLow := true
		for j := 1; j <= lookback; j++ {
			if candles[i-j].Low <= candles[i].Low || candles[i+j].Low <= candles[i].Low {
				isLow = false
				break
			}
		}
		if isLow {
			swings = append(swings, SwingPoint{
				Index:  i,
				Price:  candles[i].Low,
				IsHigh: false,
			})
		}
	}
	return swings
}

// SwingPoint for local use
type SwingPoint struct {
	Index  int
	Price  float64
	IsHigh bool
}

// Convert to SMC swing points
func convertToSMCSwings(swings []SwingPoint, isHigh bool) []smc.SwingPoint {
	result := make([]smc.SwingPoint, len(swings))
	for i, s := range swings {
		result[i] = smc.SwingPoint{
			Index:  s.Index,
			Price:  s.Price,
			IsHigh: s.IsHigh,
		}
	}
	return result
}

// resampleCandles aggregates candles (e.g., daily -> weekly)
func resampleCandles(candles []database.Candle, factor int) []database.Candle {
	if factor <= 1 || len(candles) < factor {
		return candles
	}

	var resampled []database.Candle
	for i := 0; i < len(candles); i += factor {
		end := i + factor
		if end > len(candles) {
			end = len(candles)
		}
		
		chunk := candles[i:end]
		if len(chunk) == 0 {
			continue
		}

		// Aggregate OHLCV
		aggregated := database.Candle{
			Symbol:    chunk[0].Symbol,
			Timestamp: chunk[0].Timestamp,
			Open:      chunk[0].Open,
			High:      chunk[0].High,
			Low:       chunk[0].Low,
			Close:     chunk[len(chunk)-1].Close,
			Volume:    0,
		}

		for _, c := range chunk {
			if c.High > aggregated.High {
				aggregated.High = c.High
			}
			if c.Low < aggregated.Low {
				aggregated.Low = c.Low
			}
			aggregated.Volume += c.Volume
		}

		resampled = append(resampled, aggregated)
	}

	return resampled
}

// resampleToWeekly creates weekly candles from daily
func resampleToWeekly(dailyCandles []database.Candle) []database.Candle {
	return resampleCandles(dailyCandles, 5) // 5 trading days per week
}

// GetTrendAlignment checks if all timeframes are aligned
func (ctx *MTFContext) GetTrendAlignment() int {
	if ctx.Weekly == nil || ctx.Daily == nil || ctx.H4 == nil {
		return 0
	}

	if ctx.Weekly.Trend == ctx.Daily.Trend && ctx.Daily.Trend == ctx.H4.Trend {
		return ctx.H4.Trend
	}
	return 0
}

// HasStructureSupport checks if there's structure support for a direction
func (ctx *MTFContext) HasStructureSupport(direction int) bool {
	// Check for recent BOS in the right direction
	if ctx.H4 != nil && ctx.H4.RecentBOS != nil && ctx.H4.RecentBOS.Direction == direction {
		return true
	}
	if ctx.Daily != nil && ctx.Daily.RecentBOS != nil && ctx.Daily.RecentBOS.Direction == direction {
		return true
	}
	return false
}

// HasCHoCHSignal checks if there's a change of character signal
func (ctx *MTFContext) HasCHoCHSignal(direction int) bool {
	if ctx.H4 != nil && ctx.H4.RecentCHoCH != nil && ctx.H4.RecentCHoCH.Direction == direction {
		return true
	}
	return false
}
