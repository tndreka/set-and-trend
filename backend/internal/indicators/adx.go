package indicators

import (
	"math"
	"set-and-trend/backend/internal/database"
)

type ADX struct {
	Period  int
	Values  []float64
	PlusDI  []float64
	MinusDI []float64
}

func NewADX(period int) *ADX {
	if period <= 0 {
		period = 14
	}
	return &ADX{Period: period, Values: make([]float64, 0), PlusDI: make([]float64, 0), MinusDI: make([]float64, 0)}
}

func (a *ADX) Calculate(candles []database.Candle) []float64 {
	if len(candles) < a.Period*2 {
		return nil
	}

	n := len(candles)
	a.Values = make([]float64, n)
	a.PlusDI = make([]float64, n)
	a.MinusDI = make([]float64, n)

	tr := make([]float64, n)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)

	for i := 1; i < n; i++ {
		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		tr[i] = math.Max(highLow, math.Max(highClose, lowClose))

		upMove := candles[i].High - candles[i-1].High
		downMove := candles[i-1].Low - candles[i].Low

		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}

	smoothedTR := make([]float64, n)
	smoothedPlusDM := make([]float64, n)
	smoothedMinusDM := make([]float64, n)

	for i := 1; i <= a.Period; i++ {
		smoothedTR[a.Period] += tr[i]
		smoothedPlusDM[a.Period] += plusDM[i]
		smoothedMinusDM[a.Period] += minusDM[i]
	}

	for i := a.Period + 1; i < n; i++ {
		smoothedTR[i] = smoothedTR[i-1] - smoothedTR[i-1]/float64(a.Period) + tr[i]
		smoothedPlusDM[i] = smoothedPlusDM[i-1] - smoothedPlusDM[i-1]/float64(a.Period) + plusDM[i]
		smoothedMinusDM[i] = smoothedMinusDM[i-1] - smoothedMinusDM[i-1]/float64(a.Period) + minusDM[i]
	}

	dx := make([]float64, n)
	for i := a.Period; i < n; i++ {
		if smoothedTR[i] > 0 {
			a.PlusDI[i] = 100 * smoothedPlusDM[i] / smoothedTR[i]
			a.MinusDI[i] = 100 * smoothedMinusDM[i] / smoothedTR[i]
		}
		diSum := a.PlusDI[i] + a.MinusDI[i]
		if diSum > 0 {
			dx[i] = 100 * math.Abs(a.PlusDI[i]-a.MinusDI[i]) / diSum
		}
	}

	if n >= a.Period*2 {
		sum := 0.0
		for i := a.Period; i < a.Period*2; i++ {
			sum += dx[i]
		}
		a.Values[a.Period*2-1] = sum / float64(a.Period)

		for i := a.Period * 2; i < n; i++ {
			a.Values[i] = (a.Values[i-1]*float64(a.Period-1) + dx[i]) / float64(a.Period)
		}
	}
	return a.Values
}

func (a *ADX) GetCurrent() float64 {
	if len(a.Values) == 0 {
		return 0
	}
	return a.Values[len(a.Values)-1]
}

func (a *ADX) IsTrending() bool {
	return a.GetCurrent() > 25
}

func (a *ADX) IsStrongTrend() bool {
	return a.GetCurrent() > 40
}

func (a *ADX) IsRanging() bool {
	return a.GetCurrent() < 20
}

func (a *ADX) GetTrendDirection() int {
	if len(a.PlusDI) == 0 || len(a.MinusDI) == 0 {
		return 0
	}
	plusDI := a.PlusDI[len(a.PlusDI)-1]
	minusDI := a.MinusDI[len(a.MinusDI)-1]

	if plusDI > minusDI {
		return 1
	} else if minusDI > plusDI {
		return -1
	}
	return 0
}

func (a *ADX) GetValueAt(index int) float64 {
	if index < 0 || index >= len(a.Values) {
		return 0
	}
	return a.Values[index]
}

func (a *ADX) ShouldTakeSignal(patternIsShort bool) bool {
	if !a.IsTrending() {
		return false
	}
	direction := a.GetTrendDirection()
	if patternIsShort {
		return direction == 1
	}
	return direction == -1
}

func (a *ADX) IsValid() bool {
	return len(a.Values) >= a.Period*2
}
