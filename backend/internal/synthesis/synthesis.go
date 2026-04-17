package synthesis

import (
	"fmt"
	"time"

	"set-and-trend/backend/internal/rules"
)

// MultiTFData holds precomputed multi-timeframe candles, EMAs, and index maps.
type MultiTFData struct {
	H4Indicators []rules.Indicators
	D1Candles    []rules.Candle
	D1EMA50      []float64
	D1EMA200     []float64
	W1Candles    []rules.Candle
	W1EMA50      []float64
	W1EMA200     []float64
	H4ToD1       map[int]int
	D1ToW1       map[int]int
}

// BuildFullIndicatorSeries computes EMA50, EMA200, and EMA50Prev for every H4
// bar and synthesizes D1/W1 bars with their own EMAs. Used by both the backtest
// runner and the live signal evaluator.
func BuildFullIndicatorSeries(candles []rules.Candle) MultiTFData {
	n := len(candles)
	out := make([]rules.Indicators, n)
	if n == 0 {
		return MultiTFData{H4Indicators: out}
	}
	ema50 := EMASeries(candles, 50)
	ema200 := EMASeries(candles, 200)

	d1Candles, h4ToD1 := SynthesizeD1(candles)
	d1ema50 := EMASeries(d1Candles, 50)
	d1ema200 := EMASeries(d1Candles, 200)

	w1Candles, d1ToW1 := SynthesizeW1(d1Candles)
	w1ema50 := EMASeries(w1Candles, 50)
	w1ema200 := EMASeries(w1Candles, 200)

	for i := 0; i < n; i++ {
		var prev *float64
		if i > 0 {
			v := ema50[i-1]
			prev = &v
		}
		out[i] = rules.Indicators{
			EMA50:     ema50[i],
			EMA200:    ema200[i],
			EMA50Prev: prev,
		}
		if d1Idx, ok := h4ToD1[i]; ok && d1Idx < len(d1ema50) {
			out[i].D1EMA50 = d1ema50[d1Idx]
			out[i].D1EMA200 = d1ema200[d1Idx]
		} else if i > 0 {
			out[i].D1EMA50 = out[i-1].D1EMA50
			out[i].D1EMA200 = out[i-1].D1EMA200
		}
		if d1Idx, ok := h4ToD1[i]; ok {
			if w1Idx, ok2 := d1ToW1[d1Idx]; ok2 && w1Idx < len(w1ema50) {
				out[i].W1EMA50 = w1ema50[w1Idx]
				out[i].W1EMA200 = w1ema200[w1Idx]
			}
		}
		if out[i].W1EMA50 == 0 && i > 0 {
			out[i].W1EMA50 = out[i-1].W1EMA50
			out[i].W1EMA200 = out[i-1].W1EMA200
		}
	}
	return MultiTFData{
		H4Indicators: out,
		D1Candles:    d1Candles,
		D1EMA50:      d1ema50,
		D1EMA200:     d1ema200,
		W1Candles:    w1Candles,
		W1EMA50:      w1ema50,
		W1EMA200:     w1ema200,
		H4ToD1:       h4ToD1,
		D1ToW1:       d1ToW1,
	}
}

// SynthesizeD1 aggregates H4 bars into daily OHLC bars. Returns the D1 candle
// slice and a map[h4Index] -> d1Index so each H4 bar can look up its daily EMAs.
func SynthesizeD1(h4 []rules.Candle) ([]rules.Candle, map[int]int) {
	type dayAcc struct {
		open, high, low, close float64
		ts                     time.Time
		firstH4Idx             int
	}
	days := make([]dayAcc, 0, len(h4)/6+1)
	h4ToD1 := make(map[int]int, len(h4))

	var cur *dayAcc
	for i, c := range h4 {
		dayKey := c.TimestampUTC.Truncate(24 * time.Hour)
		if cur == nil || !cur.ts.Equal(dayKey) {
			if cur != nil {
				days = append(days, *cur)
			}
			cur = &dayAcc{
				open: c.Open, high: c.High, low: c.Low, close: c.Close,
				ts: dayKey, firstH4Idx: i,
			}
		} else {
			if c.High > cur.high {
				cur.high = c.High
			}
			if c.Low < cur.low {
				cur.low = c.Low
			}
			cur.close = c.Close
		}
		h4ToD1[i] = len(days)
	}
	if cur != nil {
		days = append(days, *cur)
	}

	out := make([]rules.Candle, len(days))
	for i, d := range days {
		out[i] = rules.Candle{
			Open: d.open, High: d.high, Low: d.low, Close: d.close,
			TimestampUTC: d.ts,
		}
	}
	return out, h4ToD1
}

// SynthesizeW1 aggregates D1 candles into weekly bars (ISO week grouping).
// Returns the W1 candle slice and a map[d1Index] -> w1Index.
func SynthesizeW1(d1 []rules.Candle) ([]rules.Candle, map[int]int) {
	type weekAcc struct {
		open, high, low, close float64
		ts                     time.Time
	}
	weeks := make([]weekAcc, 0, len(d1)/5+1)
	d1ToW1 := make(map[int]int, len(d1))

	var cur *weekAcc
	prevKey := ""
	for i, c := range d1 {
		y, w := c.TimestampUTC.ISOWeek()
		key := fmt.Sprintf("%d-%02d", y, w)
		if key != prevKey {
			if cur != nil {
				weeks = append(weeks, *cur)
			}
			cur = &weekAcc{
				open: c.Open, high: c.High, low: c.Low, close: c.Close,
				ts: c.TimestampUTC,
			}
			prevKey = key
		} else {
			if c.High > cur.high {
				cur.high = c.High
			}
			if c.Low < cur.low {
				cur.low = c.Low
			}
			cur.close = c.Close
		}
		d1ToW1[i] = len(weeks)
	}
	if cur != nil {
		weeks = append(weeks, *cur)
	}

	out := make([]rules.Candle, len(weeks))
	for i, w := range weeks {
		out[i] = rules.Candle{
			Open: w.open, High: w.high, Low: w.low, Close: w.close,
			TimestampUTC: w.ts,
		}
	}
	return out, d1ToW1
}

// EMASeries returns the full EMA series. Bars before `period` are seeded with
// a simple moving average, then the recursive EMA formula takes over.
func EMASeries(candles []rules.Candle, period int) []float64 {
	n := len(candles)
	out := make([]float64, n)
	if n < period {
		return out
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += candles[i].Close
	}
	sma := sum / float64(period)
	for i := 0; i < period; i++ {
		out[i] = 0
	}
	out[period-1] = sma
	k := 2.0 / float64(period+1)
	for i := period; i < n; i++ {
		out[i] = candles[i].Close*k + out[i-1]*(1-k)
	}
	return out
}
