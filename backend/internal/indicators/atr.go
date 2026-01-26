package indicators

import (
	"math"
	"set-and-trend/backend/internal/database"
)

type ATR struct {
	Period int
	Values []float64
}

func NewATR(period int) *ATR {
	if period <= 0 {
		period = 14
	}
	return &ATR{Period: period, Values: make([]float64, 0)}
}

func (a *ATR) Calculate(candles []database.Candle) []float64 {
	if len(candles) < 2 {
		return nil
	}

	trueRanges := make([]float64, len(candles))
	trueRanges[0] = candles[0].High - candles[0].Low

	for i := 1; i < len(candles); i++ {
		tr := a.calculateTrueRange(candles[i], candles[i-1])
		trueRanges[i] = tr
	}

	a.Values = make([]float64, len(candles))

	if len(candles) >= a.Period {
		sum := 0.0
		for i := 0; i < a.Period; i++ {
			sum += trueRanges[i]
		}
		a.Values[a.Period-1] = sum / float64(a.Period)

		for i := a.Period; i < len(candles); i++ {
			a.Values[i] = (a.Values[i-1]*float64(a.Period-1) + trueRanges[i]) / float64(a.Period)
		}
	}
	return a.Values
}

func (a *ATR) calculateTrueRange(current, previous database.Candle) float64 {
	highLow := current.High - current.Low
	highClose := math.Abs(current.High - previous.Close)
	lowClose := math.Abs(current.Low - previous.Close)
	return math.Max(highLow, math.Max(highClose, lowClose))
}

func (a *ATR) GetCurrent() float64 {
	if len(a.Values) == 0 {
		return 0
	}
	return a.Values[len(a.Values)-1]
}

func (a *ATR) GetStopDistance(multiplier float64) float64 {
	return a.GetCurrent() * multiplier
}

func (a *ATR) GetStopLoss(entryPrice float64, isLong bool, multiplier float64) float64 {
	stopDistance := a.GetStopDistance(multiplier)
	if isLong {
		return entryPrice - stopDistance
	}
	return entryPrice + stopDistance
}

func (a *ATR) GetValueAt(index int) float64 {
	if index < 0 || index >= len(a.Values) {
		return 0
	}
	return a.Values[index]
}

func (a *ATR) IsValid() bool {
	return len(a.Values) >= a.Period
}
