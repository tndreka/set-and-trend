package simulation

import (
	"testing"
	"time"
	"set-and-trend/backend/internal/patterns"
)

// TestNoLookaheadBias verifies pattern detection only uses past data
func TestNoLookaheadBias(t *testing.T) {
	// Create 100 candles with known pattern
	candles := make([]patterns.Candle, 100)
	for i := 0; i < 100; i++ {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      1.1000,
			High:      1.1050,
			Low:       1.0950,
			Close:     1.1000,
		}
	}
	
	feeder := NewMarketFeeder(candles)
	
	// Advance to bar 50
	for i := 0; i < 50; i++ {
		feeder.NextBar()
	}
	
	// Get visible window - should NOT include bar 50 or later
	visible := feeder.GetVisibleCandles()
	
	if len(visible) != 50 {
		t.Errorf("Expected 50 visible candles, got %d", len(visible))
	}
	
	// Check that no candle in visible has index >= 50
	for _, c := range visible {
		if c.Index >= 50 {
			t.Errorf("LOOKAHEAD DETECTED: Visible candle has index %d (current position is 50)", c.Index)
		}
	}
	
	// Verify GetVisibleWindow also respects boundary
	window := feeder.GetVisibleWindow(30)
	for _, c := range window {
		if c.Index >= 50 {
			t.Errorf("LOOKAHEAD DETECTED in window: Candle index %d >= current position 50", c.Index)
		}
	}
	
	t.Logf("PASS: Pattern detection window contains only bars 0-49 (current bar 50 not visible)")
}

// TestRealisticEntryFill verifies orders only fill when price is actually reached
func TestRealisticEntryFill(t *testing.T) {
	config := DefaultSimulationConfig()
	
	// Create a bot with mock feeder
	candles := make([]patterns.Candle, 10)
	for i := range candles {
		candles[i] = patterns.Candle{
			Index: i,
			Open:  1.1000,
			High:  1.1050,
			Low:   1.0950,
			Close: 1.1000,
		}
	}
	
	feeder := NewMarketFeeder(candles)
	bot := NewTradingBot(feeder, config)
	
	// Create a SHORT signal at 1.0900 (below current price range)
	signal := &patterns.TradeSignal{
		Direction:  patterns.DirectionShort,
		EntryPrice: 1.0900, // Entry below the bar's low of 1.0950
		StopLoss:   1.1100,
		TakeProfit: 1.0700,
	}
	
	// Bar that does NOT reach entry price
	barNoFill := &patterns.Candle{
		Open:  1.1000,
		High:  1.1050,
		Low:   1.0950, // Doesn't go below 1.0900
		Close: 1.1000,
	}
	
	filled, _ := bot.checkOrderFill(barNoFill, signal)
	if filled {
		t.Error("Order should NOT fill - price never reached 1.0900")
	}
	
	// Bar that DOES reach entry price
	barFill := &patterns.Candle{
		Open:  1.1000,
		High:  1.1050,
		Low:   1.0850, // Goes below 1.0900
		Close: 1.0900,
	}
	
	filled, fillPrice := bot.checkOrderFill(barFill, signal)
	if !filled {
		t.Error("Order SHOULD fill - price reached 1.0900")
	}
	if fillPrice != 1.0900 {
		t.Errorf("Fill price should be 1.0900, got %.4f", fillPrice)
	}
	
	// Bar that gaps past entry - fill at open (slippage)
	barGap := &patterns.Candle{
		Open:  1.0850, // Opens below entry
		High:  1.0900,
		Low:   1.0800,
		Close: 1.0850,
	}
	
	filled, fillPrice = bot.checkOrderFill(barGap, signal)
	if !filled {
		t.Error("Order SHOULD fill on gap")
	}
	if fillPrice != 1.0850 {
		t.Errorf("Fill price should be bar open 1.0850 (slippage), got %.4f", fillPrice)
	}
	
	t.Log("PASS: Realistic entry fill logic verified")
}

// TestConservativeExitOnSameBar verifies SL assumed hit first when ambiguous
func TestConservativeExitOnSameBar(t *testing.T) {
	config := DefaultSimulationConfig()
	candles := make([]patterns.Candle, 50)
	for i := range candles {
		candles[i] = patterns.Candle{Index: i, Open: 1.1, High: 1.11, Low: 1.09, Close: 1.1}
	}
	
	feeder := NewMarketFeeder(candles)
	bot := NewTradingBot(feeder, config)
	
	// Manually set up an open SHORT position
	bot.state.OpenPosition = &Position{
		Direction:  patterns.DirectionShort,
		EntryPrice: 1.1000,
		StopLoss:   1.1100, // SL above entry
		TakeProfit: 1.0800, // TP below entry
	}
	bot.state.CurrentBar = 10
	
	// Bar where BOTH SL and TP are hit (High >= SL AND Low <= TP)
	ambiguousBar := &patterns.Candle{
		Open:  1.1000,
		High:  1.1200, // Hits SL at 1.1100
		Low:   1.0700, // Hits TP at 1.0800
		Close: 1.1000,
	}
	
	result := bot.checkExits(ambiguousBar)
	
	if result == nil {
		t.Fatal("Expected trade to close")
	}
	
	if result.Outcome != "LOSS" {
		t.Errorf("Expected LOSS (conservative SL first), got %s", result.Outcome)
	}
	
	if result.ExitPrice != 1.1100 {
		t.Errorf("Expected exit at SL 1.1100, got %.4f", result.ExitPrice)
	}
	
	t.Log("PASS: Conservative exit (SL first when ambiguous) verified")
}
