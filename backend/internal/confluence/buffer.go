package confluence

import (
	"sync"

	"set-and-trend/backend/internal/patterns"
)

// CandleBuffer thread-safe rolling buffer for live candles
type CandleBuffer struct {
	mu       sync.RWMutex
	candles  []patterns.Candle
	maxSize  int
	timeframe Timeframe
}

// NewCandleBuffer creates buffer for specific timeframe
func NewCandleBuffer(tf Timeframe, maxSize int) *CandleBuffer {
	return &CandleBuffer{
		candles:   make([]patterns.Candle, 0, maxSize),
		maxSize:   maxSize,
		timeframe: tf,
	}
}

// AddCandle adds new candle (called from WebSocket handler)
func (cb *CandleBuffer) AddCandle(candle patterns.Candle) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.candles = append(cb.candles, candle)
	
	// Keep only last maxSize candles
	if len(cb.candles) > cb.maxSize {
		cb.candles = cb.candles[len(cb.candles)-cb.maxSize:]
	}
	
	// Reindex
	for i := range cb.candles {
		cb.candles[i].Index = i
	}
}

// GetCandles returns copy of candles (thread-safe)
func (cb *CandleBuffer) GetCandles() []patterns.Candle {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	result := make([]patterns.Candle, len(cb.candles))
	copy(result, cb.candles)
	return result
}

// GetLatest returns most recent candle
func (cb *CandleBuffer) GetLatest() (patterns.Candle, bool) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if len(cb.candles) == 0 {
		return patterns.Candle{}, false
	}
	return cb.candles[len(cb.candles)-1], true
}

// Len returns current buffer size
func (cb *CandleBuffer) Len() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return len(cb.candles)
}

// IsReady returns true if buffer has enough data for analysis
func (cb *CandleBuffer) IsReady(minBars int) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return len(cb.candles) >= minBars
}