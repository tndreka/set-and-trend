package engine

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/patterns"
)

// ================================================================================
// UNIFIED TRADING ENGINE - Backtest + Live = SAME CODE
// ================================================================================

// BacktestSignal - Signal + bound future candles (bulletproof mapping)
type BacktestSignal struct {
    *patterns.TradeSignal
    FutureCandles []patterns.Candle `json:"-"`
	Timestamp     time.Time         `json:"timestamp"`     //  for selector.timeBonus()
	ATR           float64           `json:"atr"`           //  for selector.volatilityBonus()
}


// DataSource abstracts the candle source (backtest or live)
type DataSource interface {
	// NextCandle blocks until new candle available
	NextCandle(ctx context.Context) (patterns.Candle, error)
	Symbol() string
	IsLive() bool
}

// CandleFeeder interface for WebSocket integration
type CandleFeeder interface {
	// FeedCandle simulates WebSocket message
	FeedCandle(candle patterns.Candle)
	// NextCandle blocks until a candle is available
	NextCandle(ctx context.Context) (patterns.Candle, error)
	Close()
}

// TradingEngine is the unified engine for backtest and live trading
type TradingEngine struct {
	symbol  string
	pipSize float64
	isLive  bool

	// Incremental state (O(1) updates)
	candles    []patterns.Candle
	maxCandles int // Keep only recent N candles
	state      EngineState

	// Pattern detectors (incremental)
	detectors map[string]*PatternDetector

	// Risk management
	positions      []*Position
	maxPositions   int
	maxRiskPercent float64
	accountBalance float64

	// Callbacks
	onSignal         func(*patterns.TradeSignal)
	onPositionUpdate func([]*Position)

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// SwingPoint represents a detected swing high/low
type SwingPoint struct {
	Price float64
	Index int
}

// IncrementalState holds O(1) rolling state
type IncrementalState struct {
	recentHighs    []SwingPoint
	recentLows     []SwingPoint
	lastZones      []patterns.SupplyDemandZone
	necklineLevel  float64
	breakDirection string
	liquidityLevel float64
	recentSwings   int
}

// EngineState holds incremental computation state
type EngineState struct {
	CurrentBar    int
	LastTimestamp time.Time
	ATR           float64
	Direction     string // Current detected direction
	Incremental   IncrementalState
}

// PatternDetector wraps pattern logic incrementally
type PatternDetector struct {
	Name     string
	Score    float64
	Updated  bool
	LastCalc time.Time
}

// Position represents an open or closed trade position
type Position struct {
	Signal     *patterns.TradeSignal
	EntryPrice float64
	Size       float64
	OpenTime   time.Time
	PnL        float64
	IsClosed   bool
	ExitPrice  float64
	ExitTime   time.Time
	ExitReason string
}

// EngineConfig for customization
type EngineConfig struct {
	MaxHistoryBars int
	MaxPositions   int
	MaxRiskPercent float64
	AccountBalance float64
}

// DefaultEngineConfig returns sensible defaults
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxHistoryBars: 500,
		MaxPositions:   3,
		MaxRiskPercent: 2.0,
		AccountBalance: 10000.0,
	}
}

// NewTradingEngine creates unified engine
func NewTradingEngine(symbol string, config EngineConfig) *TradingEngine {
	sym := constants.MustGetSymbolConfig(symbol)

	if config.MaxHistoryBars == 0 {
		config = DefaultEngineConfig()
	}

	engine := &TradingEngine{
		symbol:         symbol,
		pipSize:        sym.PipSize,
		maxCandles:     config.MaxHistoryBars,
		maxPositions:   config.MaxPositions,
		maxRiskPercent: config.MaxRiskPercent,
		accountBalance: config.AccountBalance,
		detectors:      make(map[string]*PatternDetector),
		candles:        make([]patterns.Candle, 0, config.MaxHistoryBars),
		positions:      make([]*Position, 0),
		state: EngineState{
			Incremental: IncrementalState{
				recentHighs: make([]SwingPoint, 0, 10),
				recentLows:  make([]SwingPoint, 0, 10),
				lastZones:   make([]patterns.SupplyDemandZone, 0, 5),
			},
		},
	}

	// Initialize all detectors
	engine.initDetectors()

	return engine
}

func (e *TradingEngine) initDetectors() {
	detectorNames := []string{
		"bos",
		"sd_zones",
		"retest",
		"liquidity",
		"double_top",
		"double_bottom",
		"triple_top",
		"triple_bottom",
	}

	for _, name := range detectorNames {
		e.detectors[name] = &PatternDetector{Name: name}
	}
}

// SetSignalCallback sets the callback for new signals
func (e *TradingEngine) SetSignalCallback(cb func(*patterns.TradeSignal)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onSignal = cb
}

// SetPositionCallback sets the callback for position updates
func (e *TradingEngine) SetPositionCallback(cb func([]*Position)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onPositionUpdate = cb
}

// Start engine with data source (backtest OR live)
func (e *TradingEngine) Start(ctx context.Context, source DataSource) error {
	e.isLive = source.IsLive()
	e.ctx, e.cancel = context.WithCancel(ctx)

	modeStr := "BACKTEST"
	if e.isLive {
		modeStr = "LIVE"
	}
	log.Printf("[ENGINE] Starting %s mode for %s", modeStr, e.symbol)

	for {
		select {
		case <-e.ctx.Done():
			log.Printf("[ENGINE] Shutting down %s", e.symbol)
			return e.ctx.Err()
		default:
			candle, err := source.NextCandle(e.ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				// End of data for backtest
				if err.Error() == "end of history" {
					log.Printf("[ENGINE] Backtest complete for %s", e.symbol)
					return nil
				}
				// Transient error - wait and retry
				time.Sleep(100 * time.Millisecond)
				continue
			}

			e.ProcessCandle(candle)
		}
	}
}

// Stop gracefully stops the engine
func (e *TradingEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

// ProcessCandle - THE CORE METHOD (identical for backtest/live)
func (e *TradingEngine) ProcessCandle(candle patterns.Candle) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Incremental history (no full rescan)
	e.candles = append(e.candles, candle)
	if len(e.candles) > e.maxCandles {
		e.candles = e.candles[1:]
	}

	// 2. Update incremental state O(1)
	e.state.CurrentBar++
	e.state.LastTimestamp = candle.Timestamp

	// Calculate ATR if enough data
	if len(e.candles) >= 14 {
		e.state.ATR = calculateATR(e.candles, 14)
	}

	// 3. Update detectors (uses existing pattern functions)
	e.updateAllDetectors()

	// 4. Determine direction from detector consensus
	e.state.Direction = e.determineDirection()

	// 5. Generate signal if confluence high
	if signal := e.generateSignal(); signal != nil {
		e.handleSignal(signal)
	}

	// 6. Manage positions
	e.managePositions(candle.Close)
}

// TRUE O(1) updateAllDetectors - incremental updates only
func (e *TradingEngine) updateAllDetectors() {
	if len(e.candles) < 20 { // Minimum for swings
		return
	}

	candle := e.candles[len(e.candles)-1]

	// ========== 1. BOS (O(1)) ==========
	if d, ok := e.detectors["bos"]; ok {
		d.Score = e.incrementalBOS(candle)
		d.Updated = true
	}

	// ========== 2. S/D ZONES (O(1)) ==========
	if d, ok := e.detectors["sd_zones"]; ok {
		d.Score = e.incrementalZoneScore(candle)
		d.Updated = true
	}

	// ========== 3. RETEST (O(1)) ==========
	if d, ok := e.detectors["retest"]; ok {
		d.Score = e.incrementalRetest(candle)
		d.Updated = true
	}

	// ========== 4. LIQUIDITY (O(1)) ==========
	if d, ok := e.detectors["liquidity"]; ok {
		d.Score = e.incrementalLiquidity(candle)
		d.Updated = true
	}

	// ========== 5-8. PATTERNS (O(1) swing check) ==========
	e.incrementalPatterns(candle)
}

// ================================================================================
// O(1) INCREMENTAL DETECTOR METHODS
// ================================================================================

// incrementalBOS - Check if current candle breaks recent swing (O(1))
func (e *TradingEngine) incrementalBOS(candle patterns.Candle) float64 {
	if len(e.state.Incremental.recentHighs) == 0 || len(e.state.Incremental.recentLows) == 0 {
		return 0
	}

	lastHigh := e.state.Incremental.recentHighs[0]
	lastLow := e.state.Incremental.recentLows[0]

	// Bullish BOS: break above recent swing high
	if candle.Close > lastHigh.Price && candle.Low < lastHigh.Price*1.001 {
		e.updateSwingHighs(len(e.candles)-1, candle.High)
		e.state.Incremental.breakDirection = "bullish"
		if e.state.ATR > 0 {
			return math.Min(0.8+(candle.Close-lastHigh.Price)/(e.state.ATR*0.5), 1.0)
		}
		return 0.8
	}

	// Bearish BOS: break below recent swing low
	if candle.Close < lastLow.Price && candle.High > lastLow.Price*0.999 {
		e.updateSwingLows(len(e.candles)-1, candle.Low)
		e.state.Incremental.breakDirection = "bearish"
		if e.state.ATR > 0 {
			return math.Min(0.8+(lastLow.Price-candle.Close)/(e.state.ATR*0.5), 1.0)
		}
		return 0.8
	}

	return 0
}

// incrementalZoneScore - Distance to last zone (O(1))
func (e *TradingEngine) incrementalZoneScore(candle patterns.Candle) float64 {
	// Check proximity to recent zones (max 3)
	zonesCount := len(e.state.Incremental.lastZones)
	if zonesCount > 3 {
		zonesCount = 3
	}

	for i := 0; i < zonesCount; i++ {
		zone := e.state.Incremental.lastZones[i]
		// Direct zone touch
		if candle.Close >= zone.ProximalPrice && candle.Close <= zone.DistalPrice {
			return zone.Confidence * 0.9
		}
		if candle.Close <= zone.ProximalPrice && candle.Close >= zone.DistalPrice {
			return zone.Confidence * 0.9
		}
		// Zone proximity bonus
		if e.state.ATR > 0 {
			distance := math.Abs(candle.Close-zone.ProximalPrice) / e.state.ATR
			if distance < 1.0 {
				return zone.Confidence * (1.0 - distance) * 0.6
			}
		}
	}

	// Check if CURRENT candle creates new zone (strong impulse)
	if e.isStrongBullishCandle(candle, 0.6) {
		e.addDemandZone(candle)
		return 0.4
	}
	if e.isStrongBearishCandle(candle, 0.6) {
		e.addSupplyZone(candle)
		return 0.4
	}

	return 0
}

// incrementalRetest - Check last 5 candles only (O(1))
func (e *TradingEngine) incrementalRetest(candle patterns.Candle) float64 {
	if e.state.Incremental.necklineLevel == 0 {
		return 0
	}

	tolerance := 15 * e.pipSize
	neckline := e.state.Incremental.necklineLevel

	if math.Abs(candle.High-neckline) <= tolerance || math.Abs(candle.Low-neckline) <= tolerance {
		// Rejection strength
		rejection := math.Abs(candle.High-candle.Close) / e.pipSize
		if rejection > 10 {
			return math.Min(rejection/30.0, 1.0) * 0.9
		}
	}
	return 0
}

// incrementalLiquidity - Current candle vs recent extreme (O(1))
func (e *TradingEngine) incrementalLiquidity(candle patterns.Candle) float64 {
	liqLevel := e.state.Incremental.liquidityLevel
	if liqLevel == 0 {
		// Initialize from recent swings
		if len(e.state.Incremental.recentLows) > 0 {
			liqLevel = e.state.Incremental.recentLows[0].Price
		} else {
			return 0
		}
	}

	// Bullish sweep: wick below recent low, close above
	if candle.Low < liqLevel && candle.Close > liqLevel &&
		(candle.Close-candle.Low)/e.pipSize > 5 {
		sweepDepth := (liqLevel - candle.Low) / e.pipSize
		e.state.Incremental.liquidityLevel = candle.Low // Update liq level
		return math.Min(sweepDepth/20.0, 1.0) * 0.85
	}

	// Bearish sweep: wick above recent high, close below
	if len(e.state.Incremental.recentHighs) > 0 {
		highLevel := e.state.Incremental.recentHighs[0].Price
		if candle.High > highLevel && candle.Close < highLevel &&
			(candle.High-candle.Close)/e.pipSize > 5 {
			sweepDepth := (candle.High - highLevel) / e.pipSize
			e.state.Incremental.liquidityLevel = candle.High
			return math.Min(sweepDepth/20.0, 1.0) * 0.85
		}
	}

	return 0
}

// incrementalPatterns - Recent swings only (O(10) = O(1))
func (e *TradingEngine) incrementalPatterns(candle patterns.Candle) {
	e.updateSwings(candle) // O(1) rolling window

	highs := e.state.Incremental.recentHighs
	lows := e.state.Incremental.recentLows

	// Double Top: 2 recent highs within 1%
	if len(highs) >= 2 {
		if math.Abs(highs[0].Price-highs[1].Price)/highs[0].Price < 0.01 &&
			candle.Close < highs[0].Price*0.995 { // Break below
			if d, ok := e.detectors["double_top"]; ok {
				d.Score = 0.75
				d.Updated = true
			}
		}
	}

	// Double Bottom: 2 recent lows within 1%
	if len(lows) >= 2 {
		if math.Abs(lows[0].Price-lows[1].Price)/lows[0].Price < 0.01 &&
			candle.Close > lows[0].Price*1.005 { // Break above
			if d, ok := e.detectors["double_bottom"]; ok {
				d.Score = 0.75
				d.Updated = true
			}
		}
	}

	// Triple Top: 3 recent highs within 1%
	if len(highs) >= 3 {
		if math.Abs(highs[0].Price-highs[1].Price)/highs[0].Price < 0.01 &&
			math.Abs(highs[1].Price-highs[2].Price)/highs[1].Price < 0.01 &&
			candle.Close < highs[0].Price*0.995 {
			if d, ok := e.detectors["triple_top"]; ok {
				d.Score = 0.85
				d.Updated = true
			}
		}
	}

	// Triple Bottom: 3 recent lows within 1%
	if len(lows) >= 3 {
		if math.Abs(lows[0].Price-lows[1].Price)/lows[0].Price < 0.01 &&
			math.Abs(lows[1].Price-lows[2].Price)/lows[1].Price < 0.01 &&
			candle.Close > lows[0].Price*1.005 {
			if d, ok := e.detectors["triple_bottom"]; ok {
				d.Score = 0.85
				d.Updated = true
			}
		}
	}
}

// ================================================================================
// O(1) SWING TRACKING HELPERS
// ================================================================================

// updateSwings - O(1) rolling window max 10 swings
func (e *TradingEngine) updateSwings(candle patterns.Candle) {
	if len(e.candles) < 5 {
		return
	}

	idx := len(e.candles) - 1

	// Detect swing high: higher than 2 neighbors
	if candle.High > e.candles[idx-1].High && candle.High > e.candles[idx-2].High {
		e.updateSwingHighs(idx, candle.High)
	}

	// Detect swing low: lower than 2 neighbors
	if candle.Low < e.candles[idx-1].Low && candle.Low < e.candles[idx-2].Low {
		e.updateSwingLows(idx, candle.Low)
	}
}

func (e *TradingEngine) updateSwingHighs(idx int, price float64) {
	e.state.Incremental.recentHighs = append([]SwingPoint{{
		Price: price,
		Index: idx,
	}}, e.state.Incremental.recentHighs...)
	if len(e.state.Incremental.recentHighs) > 10 {
		e.state.Incremental.recentHighs = e.state.Incremental.recentHighs[:10]
	}
}

func (e *TradingEngine) updateSwingLows(idx int, price float64) {
	e.state.Incremental.recentLows = append([]SwingPoint{{
		Price: price,
		Index: idx,
	}}, e.state.Incremental.recentLows...)
	if len(e.state.Incremental.recentLows) > 10 {
		e.state.Incremental.recentLows = e.state.Incremental.recentLows[:10]
	}
}

// ================================================================================
// O(1) ZONE HELPERS
// ================================================================================

func (e *TradingEngine) isStrongBullishCandle(candle patterns.Candle, threshold float64) bool {
	body := candle.Close - candle.Open
	range_ := candle.High - candle.Low
	if range_ == 0 {
		return false
	}
	return body > 0 && body/range_ >= threshold
}

func (e *TradingEngine) isStrongBearishCandle(candle patterns.Candle, threshold float64) bool {
	body := candle.Open - candle.Close
	range_ := candle.High - candle.Low
	if range_ == 0 {
		return false
	}
	return body > 0 && body/range_ >= threshold
}

func (e *TradingEngine) addDemandZone(candle patterns.Candle) {
	zone := patterns.SupplyDemandZone{
		Type:          patterns.ZoneDemand,
		ProximalPrice: candle.Low,
		DistalPrice:   candle.Open,
		Confidence:    0.6,
	}
	e.state.Incremental.lastZones = append([]patterns.SupplyDemandZone{zone}, e.state.Incremental.lastZones...)
	if len(e.state.Incremental.lastZones) > 5 {
		e.state.Incremental.lastZones = e.state.Incremental.lastZones[:5]
	}
}

func (e *TradingEngine) addSupplyZone(candle patterns.Candle) {
	zone := patterns.SupplyDemandZone{
		Type:          patterns.ZoneSupply,
		ProximalPrice: candle.High,
		DistalPrice:   candle.Open,
		Confidence:    0.6,
	}
	e.state.Incremental.lastZones = append([]patterns.SupplyDemandZone{zone}, e.state.Incremental.lastZones...)
	if len(e.state.Incremental.lastZones) > 5 {
		e.state.Incremental.lastZones = e.state.Incremental.lastZones[:5]
	}
}

func (e *TradingEngine) determineDirection() string {
	// Use detector scores to determine bias
	bullishScore := 0.0
	bearishScore := 0.0

	// Double/Triple bottom = bullish
	if d, ok := e.detectors["double_bottom"]; ok && d.Score > 0 {
		bullishScore += d.Score
	}
	if d, ok := e.detectors["triple_bottom"]; ok && d.Score > 0 {
		bullishScore += d.Score
	}

	// Double/Triple top = bearish
	if d, ok := e.detectors["double_top"]; ok && d.Score > 0 {
		bearishScore += d.Score
	}
	if d, ok := e.detectors["triple_top"]; ok && d.Score > 0 {
		bearishScore += d.Score
	}

	// BOS direction from incremental state
	if d, ok := e.detectors["bos"]; ok && d.Score > 0.5 {
		// Use break direction from incremental state
		if e.state.Incremental.breakDirection == "bullish" {
			bullishScore += d.Score
		} else if e.state.Incremental.breakDirection == "bearish" {
			bearishScore += d.Score
		}
	}

	if bullishScore > bearishScore && bullishScore > 0.3 {
		return patterns.DirectionLong
	}
	if bearishScore > bullishScore && bearishScore > 0.3 {
		return patterns.DirectionShort
	}

	return "" // No clear direction
}

func (e *TradingEngine) generateSignal() *patterns.TradeSignal {
	if e.state.Direction == "" {
		return nil // No clear direction
	}

	totalConfluence := 0.0
	activeDetectors := 0

	// Sum all detector scores relevant to our direction
	for name, d := range e.detectors {
		if !d.Updated || d.Score == 0 {
			continue
		}

		// Filter detectors by direction
		if e.state.Direction == patterns.DirectionLong {
			if name == "double_top" || name == "triple_top" {
				continue // Skip bearish patterns for long
			}
		} else if e.state.Direction == patterns.DirectionShort {
			if name == "double_bottom" || name == "triple_bottom" {
				continue // Skip bullish patterns for short
			}
		}

		totalConfluence += d.Score
		activeDetectors++
	}

	avgConfluence := 0.0
	if activeDetectors > 0 {
		avgConfluence = totalConfluence / float64(activeDetectors)
	}

	// Only generate if confluence > threshold
	if avgConfluence < 0.65 {
		return nil
	}

	currentPrice := e.candles[len(e.candles)-1].Close

	// Calculate stop loss and take profit
	stopDistance := e.state.ATR * 1.5
	if stopDistance == 0 {
		stopDistance = 20 * e.pipSize // Fallback
	}

	var stopLoss, takeProfit float64
	if e.state.Direction == patterns.DirectionLong {
		stopLoss = currentPrice - stopDistance
		takeProfit = currentPrice + (stopDistance * 2.0) // 2:1 R:R
	} else {
		stopLoss = currentPrice + stopDistance
		takeProfit = currentPrice - (stopDistance * 2.0)
	}

	// Calculate risk-reward
	risk := math.Abs(currentPrice - stopLoss)
	reward := math.Abs(takeProfit - currentPrice)
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return &patterns.TradeSignal{
		Symbol:      e.symbol,
		Direction:   e.state.Direction,
		EntryPrice:  currentPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		RiskReward:  riskReward,
		Confidence:  avgConfluence,
		PatternType: fmt.Sprintf("CONFLUENCE_%.0f", avgConfluence*100),
	}
}

func (e *TradingEngine) handleSignal(signal *patterns.TradeSignal) {
	// Check if we can open more positions
	openCount := 0
	for _, p := range e.positions {
		if !p.IsClosed {
			openCount++
		}
	}
	if openCount >= e.maxPositions {
		return // Max positions reached
	}

	// Position sizing based on risk
	positionSize := e.calculatePositionSize(signal)
	if positionSize <= 0 {
		return
	}

	position := &Position{
		Signal:     signal,
		EntryPrice: signal.EntryPrice,
		Size:       positionSize,
		OpenTime:   e.state.LastTimestamp,
	}

	e.positions = append(e.positions, position)

	if e.onSignal != nil {
		e.onSignal(signal)
	}

	log.Printf("[SIGNAL] %s %s @ %.5f (conf: %.1f%%, R:R: %.2f)",
		signal.Direction, e.symbol, signal.EntryPrice, signal.Confidence*100, signal.RiskReward)
}

func (e *TradingEngine) calculatePositionSize(signal *patterns.TradeSignal) float64 {
	if e.accountBalance <= 0 {
		return 0
	}

	// Risk amount = account * risk%
	riskAmount := e.accountBalance * (e.maxRiskPercent / 100.0)

	// Stop distance in pips
	stopDistance := math.Abs(signal.EntryPrice - signal.StopLoss)
	stopPips := stopDistance / e.pipSize

	if stopPips <= 0 {
		return 0
	}

	// Get pip value for this symbol using helper functions
	pipValue := constants.GetPipValueForSymbol(e.symbol)

	// Position size = risk amount / (stop pips * pip value)
	if pipValue <= 0 {
		pipValue = 10.0 // Default $10 per pip per lot
	}

	lots := riskAmount / (stopPips * pipValue)

	// Round to lot step
	lotStep := constants.GetLotStepForSymbol(e.symbol)
	if lotStep <= 0 {
		lotStep = 0.01
	}
	lots = math.Floor(lots/lotStep) * lotStep

	// Ensure minimum lot size
	minLot := constants.GetMinLotForSymbol(e.symbol)
	if lots < minLot {
		lots = minLot
	}

	return lots
}

func (e *TradingEngine) managePositions(currentPrice float64) {
	for _, pos := range e.positions {
		if pos.IsClosed {
			continue
		}

		// Calculate current P&L
		if pos.Signal.Direction == patterns.DirectionLong {
			pos.PnL = (currentPrice - pos.EntryPrice) / e.pipSize * pos.Size
		} else {
			pos.PnL = (pos.EntryPrice - currentPrice) / e.pipSize * pos.Size
		}

		// Check stop loss
		if pos.Signal.Direction == patterns.DirectionLong && currentPrice <= pos.Signal.StopLoss {
			e.closePosition(pos, currentPrice, "STOP_LOSS")
		} else if pos.Signal.Direction == patterns.DirectionShort && currentPrice >= pos.Signal.StopLoss {
			e.closePosition(pos, currentPrice, "STOP_LOSS")
		}

		// Check take profit
		if pos.Signal.Direction == patterns.DirectionLong && currentPrice >= pos.Signal.TakeProfit {
			e.closePosition(pos, currentPrice, "TAKE_PROFIT")
		} else if pos.Signal.Direction == patterns.DirectionShort && currentPrice <= pos.Signal.TakeProfit {
			e.closePosition(pos, currentPrice, "TAKE_PROFIT")
		}
	}

	// Notify position updates
	if e.onPositionUpdate != nil {
		e.onPositionUpdate(e.positions)
	}
}

func (e *TradingEngine) closePosition(pos *Position, exitPrice float64, reason string) {
	pos.IsClosed = true
	pos.ExitPrice = exitPrice
	pos.ExitTime = e.state.LastTimestamp
	pos.ExitReason = reason

	// Final P&L
	if pos.Signal.Direction == patterns.DirectionLong {
		pos.PnL = (exitPrice - pos.EntryPrice) / e.pipSize * pos.Size
	} else {
		pos.PnL = (pos.EntryPrice - exitPrice) / e.pipSize * pos.Size
	}

	// Update account balance
	e.accountBalance += pos.PnL

	log.Printf("[CLOSE] %s %s @ %.5f (%s) P&L: %.2f",
		pos.Signal.Direction, e.symbol, exitPrice, reason, pos.PnL)
}

// GetState returns current engine state (thread-safe)
func (e *TradingEngine) GetState() EngineState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// GetPositions returns all positions (thread-safe)
func (e *TradingEngine) GetPositions() []*Position {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent data races
	result := make([]*Position, len(e.positions))
	copy(result, e.positions)
	return result
}

// GetOpenPositions returns only open positions
func (e *TradingEngine) GetOpenPositions() []*Position {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var open []*Position
	for _, p := range e.positions {
		if !p.IsClosed {
			open = append(open, p)
		}
	}
	return open
}

// GetAccountBalance returns current balance
func (e *TradingEngine) GetAccountBalance() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.accountBalance
}

// ================================================================================
// HELPER FUNCTIONS
// ================================================================================

func calculateATR(candles []patterns.Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	var trSum float64
	startIdx := len(candles) - period - 1

	for i := startIdx + 1; i < len(candles); i++ {
		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		tr := math.Max(highLow, math.Max(highClose, lowClose))
		trSum += tr
	}

	return trSum / float64(period)
}

// ================================================================================
// DATA SOURCE ADAPTERS - Backtest & Live
// ================================================================================

// HistoricalDataSource feeds historical CSV/database data
type HistoricalDataSource struct {
	symbol  string
	candles []patterns.Candle
	index   int
	delay   time.Duration
}

// NewHistoricalDataSource creates a backtest data source
func NewHistoricalDataSource(symbol string, candles []patterns.Candle) *HistoricalDataSource {
	return &HistoricalDataSource{
		symbol:  symbol,
		candles: candles,
		delay:   10 * time.Millisecond, // Fast backtest
	}
}

// SetDelay sets delay between candles (for realistic simulation)
func (s *HistoricalDataSource) SetDelay(d time.Duration) {
	s.delay = d
}

func (s *HistoricalDataSource) NextCandle(ctx context.Context) (patterns.Candle, error) {
	select {
	case <-ctx.Done():
		return patterns.Candle{}, ctx.Err()
	default:
	}

	if s.index >= len(s.candles) {
		return patterns.Candle{}, fmt.Errorf("end of history")
	}

	candle := s.candles[s.index]
	s.index++

	// Simulate realistic timing
	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	return candle, nil
}

func (s *HistoricalDataSource) Symbol() string { return s.symbol }
func (s *HistoricalDataSource) IsLive() bool   { return false }

// Progress returns backtest progress as 0.0-1.0
func (s *HistoricalDataSource) Progress() float64 {
	if len(s.candles) == 0 {
		return 0
	}
	return float64(s.index) / float64(len(s.candles))
}

// WebSocketDataSource feeds live WebSocket candles
type WebSocketDataSource struct {
	symbol string
	feeder CandleFeeder
}

// NewWebSocketDataSource creates a live data source
func NewWebSocketDataSource(symbol string, feeder CandleFeeder) *WebSocketDataSource {
	return &WebSocketDataSource{
		symbol: symbol,
		feeder: feeder,
	}
}

func (s *WebSocketDataSource) NextCandle(ctx context.Context) (patterns.Candle, error) {
	return s.feeder.NextCandle(ctx)
}

func (s *WebSocketDataSource) Symbol() string { return s.symbol }
func (s *WebSocketDataSource) IsLive() bool   { return true }

// ================================================================================
// CHANNEL-BASED CANDLE FEEDER (for WebSocket integration)
// ================================================================================

// ChannelFeeder implements CandleFeeder using channels
type ChannelFeeder struct {
	candles chan patterns.Candle
	closed  bool
	mu      sync.Mutex
}

// NewChannelFeeder creates a channel-based feeder
func NewChannelFeeder(bufferSize int) *ChannelFeeder {
	return &ChannelFeeder{
		candles: make(chan patterns.Candle, bufferSize),
	}
}

// FeedCandle sends a candle to the engine (called by WebSocket handler)
func (f *ChannelFeeder) FeedCandle(candle patterns.Candle) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	select {
	case f.candles <- candle:
	default:
		// Channel full, drop oldest
		select {
		case <-f.candles:
		default:
		}
		f.candles <- candle
	}
}

// NextCandle blocks until a candle is available
func (f *ChannelFeeder) NextCandle(ctx context.Context) (patterns.Candle, error) {
	select {
	case <-ctx.Done():
		return patterns.Candle{}, ctx.Err()
	case candle, ok := <-f.candles:
		if !ok {
			return patterns.Candle{}, fmt.Errorf("feeder closed")
		}
		return candle, nil
	}
}

// Close closes the feeder
func (f *ChannelFeeder) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.closed {
		f.closed = true
		close(f.candles)
	}
}

// RunBacktest - PURE FUNCTION for backtesting (NO async/callbacks externally)
func (e *TradingEngine) RunBacktest(candles []patterns.Candle) []*BacktestSignal {
    var signals []*BacktestSignal
    origCallback := e.onSignal
    
    // Capture ALL signals with bound candles
    e.SetSignalCallback(func(sig *patterns.TradeSignal) {
        signals = append(signals, &BacktestSignal{
            TradeSignal:  sig,
            FutureCandles: candles, // ✅ PERFECT binding
			Timestamp:     e.state.LastTimestamp, // ✅ ADDED
			ATR:           e.state.ATR,           // ✅ ADDED
        })
    })
    
    // Run synchronously (zero delay)
    source := NewHistoricalDataSource(e.symbol, candles)
    source.SetDelay(0)
    ctx := context.Background()
    e.Start(ctx, source)
    
    // Restore callback
    e.SetSignalCallback(origCallback)
    return signals
}
