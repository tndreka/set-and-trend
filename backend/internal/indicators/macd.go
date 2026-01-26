package indicators

import "set-and-trend/backend/internal/database"

type MACD struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	MACDLine     []float64
	SignalLine   []float64
	Histogram    []float64
}

func NewMACD(fast, slow, signal int) *MACD {
	if fast <= 0 {
		fast = 12
	}
	if slow <= 0 {
		slow = 26
	}
	if signal <= 0 {
		signal = 9
	}
	return &MACD{FastPeriod: fast, SlowPeriod: slow, SignalPeriod: signal}
}

func (m *MACD) Calculate(candles []database.Candle) {
	if len(candles) < m.SlowPeriod {
		return
	}

	n := len(candles)
	closes := make([]float64, n)
	for i, c := range candles {
		closes[i] = c.Close
	}

	fastEMA := m.calculateEMA(closes, m.FastPeriod)
	slowEMA := m.calculateEMA(closes, m.SlowPeriod)

	m.MACDLine = make([]float64, n)
	for i := m.SlowPeriod - 1; i < n; i++ {
		m.MACDLine[i] = fastEMA[i] - slowEMA[i]
	}

	m.SignalLine = m.calculateEMA(m.MACDLine, m.SignalPeriod)

	m.Histogram = make([]float64, n)
	startIdx := m.SlowPeriod + m.SignalPeriod - 2
	for i := startIdx; i < n; i++ {
		m.Histogram[i] = m.MACDLine[i] - m.SignalLine[i]
	}
}

func (m *MACD) calculateEMA(data []float64, period int) []float64 {
	if len(data) < period {
		return nil
	}
	ema := make([]float64, len(data))
	multiplier := 2.0 / float64(period+1)

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	ema[period-1] = sum / float64(period)

	for i := period; i < len(data); i++ {
		ema[i] = (data[i]-ema[i-1])*multiplier + ema[i-1]
	}
	return ema
}

func (m *MACD) GetCurrentHistogram() float64 {
	if len(m.Histogram) == 0 {
		return 0
	}
	return m.Histogram[len(m.Histogram)-1]
}

func (m *MACD) IsBullish() bool {
	return m.GetCurrentHistogram() > 0
}

func (m *MACD) IsBearish() bool {
	return m.GetCurrentHistogram() < 0
}

func (m *MACD) HasBullishCrossover(lookback int) bool {
	if len(m.Histogram) < lookback+1 {
		return false
	}
	for i := len(m.Histogram) - 1; i >= len(m.Histogram)-lookback && i > 0; i-- {
		if m.Histogram[i] > 0 && m.Histogram[i-1] <= 0 {
			return true
		}
	}
	return false
}

func (m *MACD) HasBearishCrossover(lookback int) bool {
	if len(m.Histogram) < lookback+1 {
		return false
	}
	for i := len(m.Histogram) - 1; i >= len(m.Histogram)-lookback && i > 0; i-- {
		if m.Histogram[i] < 0 && m.Histogram[i-1] >= 0 {
			return true
		}
	}
	return false
}

func (m *MACD) ConfirmsPatternBreakout(isShort bool, lookback int) bool {
	if isShort {
		return m.IsBearish() || m.HasBearishCrossover(lookback)
	}
	return m.IsBullish() || m.HasBullishCrossover(lookback)
}

func (m *MACD) IsValid() bool {
	return len(m.Histogram) > m.SlowPeriod+m.SignalPeriod
}
