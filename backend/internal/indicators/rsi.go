package indicators

import "set-and-trend/backend/internal/database"

type RSI struct {
	Period int
	Values []float64
}

func NewRSI(period int) *RSI {
	if period <= 0 {
		period = 14
	}
	return &RSI{Period: period, Values: make([]float64, 0)}
}

func (r *RSI) Calculate(candles []database.Candle) []float64 {
	if len(candles) < r.Period+1 {
		return nil
	}

	r.Values = make([]float64, len(candles))
	gains := make([]float64, len(candles))
	losses := make([]float64, len(candles))

	for i := 1; i < len(candles); i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	var avgGain, avgLoss float64
	for i := 1; i <= r.Period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(r.Period)
	avgLoss /= float64(r.Period)

	if avgLoss == 0 {
		r.Values[r.Period] = 100
	} else {
		rs := avgGain / avgLoss
		r.Values[r.Period] = 100 - (100 / (1 + rs))
	}

	for i := r.Period + 1; i < len(candles); i++ {
		avgGain = (avgGain*float64(r.Period-1) + gains[i]) / float64(r.Period)
		avgLoss = (avgLoss*float64(r.Period-1) + losses[i]) / float64(r.Period)

		if avgLoss == 0 {
			r.Values[i] = 100
		} else {
			rs := avgGain / avgLoss
			r.Values[i] = 100 - (100 / (1 + rs))
		}
	}
	return r.Values
}

func (r *RSI) GetCurrent() float64 {
	if len(r.Values) == 0 {
		return 50
	}
	return r.Values[len(r.Values)-1]
}

func (r *RSI) IsOverbought(level float64) bool {
	if level == 0 {
		level = 70
	}
	return r.GetCurrent() > level
}

func (r *RSI) IsOversold(level float64) bool {
	if level == 0 {
		level = 30
	}
	return r.GetCurrent() < level
}

func (r *RSI) HasBearishDivergence(candles []database.Candle, lookback int) bool {
	if len(candles) < lookback || len(r.Values) < len(candles) {
		return false
	}
	endIdx := len(candles) - 1
	startIdx := endIdx - lookback

	var priceHighs []struct {
		idx   int
		price float64
		rsi   float64
	}

	for i := startIdx + 1; i < endIdx; i++ {
		if candles[i].High > candles[i-1].High && candles[i].High > candles[i+1].High {
			priceHighs = append(priceHighs, struct {
				idx   int
				price float64
				rsi   float64
			}{i, candles[i].High, r.Values[i]})
		}
	}

	if len(priceHighs) < 2 {
		return false
	}

	for i := 0; i < len(priceHighs)-1; i++ {
		for j := i + 1; j < len(priceHighs); j++ {
			if priceHighs[j].price > priceHighs[i].price && priceHighs[j].rsi < priceHighs[i].rsi {
				return true
			}
		}
	}
	return false
}

func (r *RSI) HasBullishDivergence(candles []database.Candle, lookback int) bool {
	if len(candles) < lookback || len(r.Values) < len(candles) {
		return false
	}
	endIdx := len(candles) - 1
	startIdx := endIdx - lookback

	var priceLows []struct {
		idx   int
		price float64
		rsi   float64
	}

	for i := startIdx + 1; i < endIdx; i++ {
		if candles[i].Low < candles[i-1].Low && candles[i].Low < candles[i+1].Low {
			priceLows = append(priceLows, struct {
				idx   int
				price float64
				rsi   float64
			}{i, candles[i].Low, r.Values[i]})
		}
	}

	if len(priceLows) < 2 {
		return false
	}

	for i := 0; i < len(priceLows)-1; i++ {
		for j := i + 1; j < len(priceLows); j++ {
			if priceLows[j].price < priceLows[i].price && priceLows[j].rsi > priceLows[i].rsi {
				return true
			}
		}
	}
	return false
}

func (r *RSI) GetValueAt(index int) float64 {
	if index < 0 || index >= len(r.Values) {
		return 50
	}
	return r.Values[index]
}

func (r *RSI) IsValid() bool {
	return len(r.Values) > r.Period
}
