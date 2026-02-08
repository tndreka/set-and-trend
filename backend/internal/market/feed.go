package market

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"set-and-trend/backend/internal/patterns"
)

var ErrNoCandles = errors.New("no candles provided")
var ErrUnsortedCandles = errors.New("candles must be sorted by timestamp ascending")
var ErrFeedExhausted = errors.New("feed exhausted")
var ErrFeedClosed = errors.New("feed is closed")

type CandleEvent struct {
	Candle    patterns.Candle `json:"candle"`
	Timestamp time.Time       `json:"timestamp"`
	IsClosed  bool            `json:"is_closed"`
	Sequence  int64           `json:"sequence"`
	BarIndex  int             `json:"bar_index"`
}

type CandleFeed interface {
	Subscribe(ctx context.Context) (<-chan CandleEvent, error)
	Next() (*patterns.Candle, error)
	Current() *patterns.Candle
	Peek() *patterns.Candle
	HasMore() bool
	GetWindow(size int) []patterns.Candle
	GetAllVisible() []patterns.Candle
	Position() int
	TotalCandles() int
	Progress() float64
	Reset()
	Seek(position int) bool
	SetSpeed(multiplier float64)
	Close() error
}

type HistoricalFeed struct {
	mu         sync.RWMutex
	candles    []patterns.Candle
	currentIdx int
	sequence   int64
	speed      float64
	closed     bool
}

func NewHistoricalFeed(candles []patterns.Candle) (*HistoricalFeed, error) {
	if len(candles) == 0 {
		return nil, ErrNoCandles
	}

	copied := make([]patterns.Candle, len(candles))
	copy(copied, candles)

	if !isSorted(copied) {
		return nil, ErrUnsortedCandles
	}

	for i := range copied {
		copied[i].Index = i
	}

	return &HistoricalFeed{
		candles:    copied,
		currentIdx: -1,
		speed:      0,
	}, nil
}

func isSorted(candles []patterns.Candle) bool {
	return sort.SliceIsSorted(candles, func(i, j int) bool {
		return candles[i].Timestamp.Before(candles[j].Timestamp)
	})
}

func (h *HistoricalFeed) Subscribe(ctx context.Context) (<-chan CandleEvent, error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrFeedClosed
	}
	h.mu.Unlock()

	ch := make(chan CandleEvent, 1)

	go func() {
		defer close(ch)

		var ticker *time.Ticker
		if h.speed > 0 {
			interval := time.Duration(float64(time.Second) / h.speed)
			ticker = time.NewTicker(interval)
			defer ticker.Stop()
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if ticker != nil {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}

			candle, err := h.Next()
			if err != nil {
				return
			}

			h.mu.Lock()
			h.sequence++
			seq := h.sequence
			pos := h.currentIdx
			h.mu.Unlock()

			event := CandleEvent{
				Candle:    *candle,
				Timestamp: time.Now(),
				IsClosed:  true,
				Sequence:  seq,
				BarIndex:  pos,
			}

			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (h *HistoricalFeed) Next() (*patterns.Candle, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrFeedClosed
	}

	if h.currentIdx >= len(h.candles)-1 {
		return nil, ErrFeedExhausted
	}

	h.currentIdx++
	return &h.candles[h.currentIdx], nil
}

func (h *HistoricalFeed) Current() *patterns.Candle {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.currentIdx < 0 || h.currentIdx >= len(h.candles) {
		return nil
	}
	return &h.candles[h.currentIdx]
}

func (h *HistoricalFeed) Peek() *patterns.Candle {
	h.mu.RLock()
	defer h.mu.RUnlock()

	nextIdx := h.currentIdx + 1
	if nextIdx >= len(h.candles) {
		return nil
	}
	return &h.candles[nextIdx]
}

func (h *HistoricalFeed) HasMore() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentIdx < len(h.candles)-1
}

func (h *HistoricalFeed) GetWindow(size int) []patterns.Candle {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.currentIdx < 0 {
		return nil
	}

	start := h.currentIdx - size + 1
	if start < 0 {
		start = 0
	}

	end := h.currentIdx + 1
	window := make([]patterns.Candle, end-start)
	copy(window, h.candles[start:end])
	return window
}

func (h *HistoricalFeed) GetAllVisible() []patterns.Candle {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.currentIdx < 0 {
		return nil
	}

	visible := make([]patterns.Candle, h.currentIdx+1)
	copy(visible, h.candles[:h.currentIdx+1])
	return visible
}

func (h *HistoricalFeed) Position() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentIdx
}

func (h *HistoricalFeed) TotalCandles() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.candles)
}

func (h *HistoricalFeed) Progress() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.candles) == 0 {
		return 0
	}
	return float64(h.currentIdx+1) / float64(len(h.candles)) * 100
}

func (h *HistoricalFeed) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.currentIdx = -1
	h.sequence = 0
}

func (h *HistoricalFeed) Seek(position int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if position < -1 || position >= len(h.candles) {
		return false
	}
	h.currentIdx = position
	return true
}

func (h *HistoricalFeed) SetSpeed(multiplier float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if multiplier >= 0 {
		h.speed = multiplier
	}
}

func (h *HistoricalFeed) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	return nil
}

var _ CandleFeed = (*HistoricalFeed)(nil)

type FeedStats struct {
	Timeframe    string        `json:"timeframe"`
	TotalCandles int           `json:"total_candles"`
	Position     int           `json:"position"`
	Progress     float64       `json:"progress"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
}

func (h *HistoricalFeed) GetStats() FeedStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := FeedStats{
		TotalCandles: len(h.candles),
		Position:     h.currentIdx,
		Progress:     float64(h.currentIdx+1) / float64(len(h.candles)) * 100,
	}

	if len(h.candles) > 0 {
		stats.StartTime = h.candles[0].Timestamp
		stats.EndTime = h.candles[len(h.candles)-1].Timestamp
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
	}

	return stats
}
