package indicators

import "set-and-trend/backend/internal/database"

type EMA struct {
	Period int
	Values []float64
}

func NewEMA(period int) *EMA {
	if period <= 0 {
		period = 20
	}
	return &EMA{Period: period, Values: make([]float64, 0)}
}

func (e *EMA) Calculate(candles []database.Candle) []float64 {
	if len(candles) < e.Period {
		return nil
	}
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}
	return e.CalculateFromPrices(closes)
}

func (e *EMA) CalculateFromPrices(prices []float64) []float64 {
	if len(prices) < e.Period {
		return nil
	}
	e.Values = make([]float64, len(prices))
	multiplier := 2.0 / float64(e.Period+1)

	sum := 0.0
	for i := 0; i < e.Period; i++ {
		sum += prices[i]
	}
	e.Values[e.Period-1] = sum / float64(e.Period)

	for i := e.Period; i < len(prices); i++ {
		e.Values[i] = (prices[i]-e.Values[i-1])*multiplier + e.Values[i-1]
	}
	return e.Values
}

func (e *EMA) GetCurrent() float64 {
	if len(e.Values) == 0 {
		return 0
	}
	return e.Values[len(e.Values)-1]
}

func (e *EMA) IsValid() bool {
	return len(e.Values) >= e.Period
}

type TrendAnalyzer struct {
	FastEMA *EMA
	SlowEMA *EMA
}

func NewTrendAnalyzer(fastPeriod, slowPeriod int) *TrendAnalyzer {
	return &TrendAnalyzer{FastEMA: NewEMA(fastPeriod), SlowEMA: NewEMA(slowPeriod)}
}

func (t *TrendAnalyzer) Calculate(candles []database.Candle) {
	t.FastEMA.Calculate(candles)
	t.SlowEMA.Calculate(candles)
}

func (t *TrendAnalyzer) GetTrend() int {
	if !t.FastEMA.IsValid() || !t.SlowEMA.IsValid() {
		return 0
	}
	fast := t.FastEMA.GetCurrent()
	slow := t.SlowEMA.GetCurrent()
	if fast > slow {
		return 1
	} else if fast < slow {
		return -1
	}
	return 0
}

func (t *TrendAnalyzer) IsUptrend() bool {
	return t.GetTrend() == 1
}

func (t *TrendAnalyzer) IsDowntrend() bool {
	return t.GetTrend() == -1
}

func (t *TrendAnalyzer) ConfirmsPattern(patternIsShort bool) bool {
	if patternIsShort {
		return t.IsUptrend()
	}
	return t.IsDowntrend()
}
