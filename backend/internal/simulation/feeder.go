package simulation

import (
	"set-and-trend/backend/internal/patterns"
)

// MarketFeeder provides bar-by-bar data access without lookahead
// This is the CRITICAL component that prevents lookahead bias
type MarketFeeder struct {
	allCandles   []patterns.Candle // Full dataset (PRIVATE - bot cannot access)
	currentIndex int               // Current position
}

// NewMarketFeeder creates a feeder with historical data
// The candles slice is copied to prevent external modification
func NewMarketFeeder(candles []patterns.Candle) *MarketFeeder {
	// Deep copy to prevent external access
	copied := make([]patterns.Candle, len(candles))
	copy(copied, candles)
	
	return &MarketFeeder{
		allCandles:   copied,
		currentIndex: 0,
	}
}

// Reset moves back to the start
func (f *MarketFeeder) Reset() {
	f.currentIndex = 0
}

// Seek moves to a specific bar index
func (f *MarketFeeder) Seek(index int) bool {
	if index < 0 || index >= len(f.allCandles) {
		return false
	}
	f.currentIndex = index
	return true
}

// HasMoreBars returns true if more bars are available
func (f *MarketFeeder) HasMoreBars() bool {
	return f.currentIndex < len(f.allCandles)
}

// NextBar advances to the next bar and returns it
// Returns nil if no more bars available
func (f *MarketFeeder) NextBar() *patterns.Candle {
	if !f.HasMoreBars() {
		return nil
	}
	
	bar := f.allCandles[f.currentIndex]
	f.currentIndex++
	return &bar
}

// PeekCurrentBar returns current bar without advancing
func (f *MarketFeeder) PeekCurrentBar() *patterns.Candle {
	if !f.HasMoreBars() {
		return nil
	}
	return &f.allCandles[f.currentIndex]
}

// GetVisibleCandles returns all candles UP TO (not including) currentIndex
// This is what the bot can "see" - only historical data
func (f *MarketFeeder) GetVisibleCandles() []patterns.Candle {
	if f.currentIndex == 0 {
		return nil
	}
	
	// Return copy to prevent modification
	visible := make([]patterns.Candle, f.currentIndex)
	copy(visible, f.allCandles[:f.currentIndex])
	return visible
}

// GetVisibleWindow returns the last N visible candles for analysis
// This is the typical window a pattern detector would use
func (f *MarketFeeder) GetVisibleWindow(windowSize int) []patterns.Candle {
	if f.currentIndex == 0 {
		return nil
	}
	
	start := f.currentIndex - windowSize
	if start < 0 {
		start = 0
	}
	
	visible := make([]patterns.Candle, f.currentIndex-start)
	copy(visible, f.allCandles[start:f.currentIndex])
	return visible
}

// CurrentIndex returns the current bar index (0-based)
func (f *MarketFeeder) CurrentIndex() int {
	return f.currentIndex
}

// TotalBars returns the total number of bars
func (f *MarketFeeder) TotalBars() int {
	return len(f.allCandles)
}

// Progress returns current progress as percentage
func (f *MarketFeeder) Progress() float64 {
	if len(f.allCandles) == 0 {
		return 0.0
	}
	return float64(f.currentIndex) / float64(len(f.allCandles)) * 100.0
}

// VERIFICATION: These methods exist solely to prove no lookahead bias

// VerifyNoLookahead confirms the bot timestamp is always <= current bar
// This should be called after every pattern detection
func (f *MarketFeeder) VerifyNoLookahead(detectionTime int) bool {
	// Detection time (bar index) must be less than current position
	return detectionTime < f.currentIndex
}
