package analyzer

import (
	"fmt"
	"math"
	"strings"

	"set-and-trend/backend/internal/engine"
	"set-and-trend/backend/internal/patterns"
)

// ChartVisualizer creates ASCII charts for signal analysis
type ChartVisualizer struct {
	width  int
	height int
}

func NewChartVisualizer(width, height int) *ChartVisualizer {
	return &ChartVisualizer{
		width:  width,
		height: height,
	}
}

// VisualizeSignal creates ASCII chart showing the signal
func (cv *ChartVisualizer) VisualizeSignal(sig *engine.BacktestSignal, numCandles int) string {
	if len(sig.FutureCandles) == 0 {
		return "No candles available for visualization"
	}
	
	// Limit candles to show
	candles := sig.FutureCandles
	if numCandles > 0 && len(candles) > numCandles {
		candles = candles[:numCandles]
	}
	
	var output strings.Builder
	
	// Header
	output.WriteString("╔" + strings.Repeat("═", cv.width-2) + "╗\n")
	output.WriteString(fmt.Sprintf("║ %s %s @ %.5f (R:R %.2f, Conf %.0f%%) %s ║\n",
		sig.Symbol, sig.Direction, sig.EntryPrice, sig.RiskReward, 
		sig.Confidence*100, 
		strings.Repeat(" ", max(0, cv.width-len(sig.Symbol)-50))))
	output.WriteString("╠" + strings.Repeat("═", cv.width-2) + "╣\n")
	
	// Find price range
	minPrice, maxPrice := cv.findPriceRange(candles, sig)
	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		priceRange = 0.0001
	}
	
	// Draw chart
	for row := 0; row < cv.height; row++ {
		price := maxPrice - (float64(row)/float64(cv.height-1))*priceRange
		line := "║ "
		
		// Price scale on left
		line += fmt.Sprintf("%8.5f ", price)
		
		// Draw candles
		for col, candle := range candles {
			if col >= cv.width-15 {
				break
			}
			
			char := cv.getCandleChar(candle, price, priceRange, sig)
			line += string(char)
		}
		
		// Key levels on right
		if math.Abs(price-sig.TakeProfit) < priceRange*0.02 {
			line += " ← TP"
		} else if math.Abs(price-sig.EntryPrice) < priceRange*0.02 {
			line += " ← Entry"
		} else if math.Abs(price-sig.StopLoss) < priceRange*0.02 {
			line += " ← SL"
		}
		
		// Pad to width
		for len(line) < cv.width-2 {
			line += " "
		}
		line += " ║\n"
		output.WriteString(line)
	}
	
	// Footer with outcome
	output.WriteString("╠" + strings.Repeat("═", cv.width-2) + "╣\n")
	
	// Find outcome
	outcome := cv.findOutcome(sig)
	outcomeStr := fmt.Sprintf("Outcome: %s", outcome)
	padding := strings.Repeat(" ", max(0, cv.width-len(outcomeStr)-4))
	output.WriteString("║ " + outcomeStr + padding + " ║\n")
	
	output.WriteString("╚" + strings.Repeat("═", cv.width-2) + "╝\n")
	
	return output.String()
}

func (cv *ChartVisualizer) findPriceRange(candles []patterns.Candle, sig *engine.BacktestSignal) (float64, float64) {
	minPrice := candles[0].Low
	maxPrice := candles[0].High
	
	for _, c := range candles {
		if c.Low < minPrice {
			minPrice = c.Low
		}
		if c.High > maxPrice {
			maxPrice = c.High
		}
	}
	
	// Include entry, SL, TP in range
	minPrice = math.Min(minPrice, math.Min(sig.StopLoss, math.Min(sig.EntryPrice, sig.TakeProfit)))
	maxPrice = math.Max(maxPrice, math.Max(sig.StopLoss, math.Max(sig.EntryPrice, sig.TakeProfit)))
	
	// Add 2% padding
	padding := (maxPrice - minPrice) * 0.02
	return minPrice - padding, maxPrice + padding
}

func (cv *ChartVisualizer) getCandleChar(candle patterns.Candle, price, priceRange float64, sig *engine.BacktestSignal) rune {
	tolerance := priceRange * 0.01
	
	// Check if this row intersects the candle
	withinHigh := price <= candle.High+tolerance
	withinLow := price >= candle.Low-tolerance
	withinBody := (price <= math.Max(candle.Open, candle.Close)+tolerance) && 
	              (price >= math.Min(candle.Open, candle.Close)-tolerance)
	
	if !withinHigh || !withinLow {
		// Check key levels
		if math.Abs(price-sig.TakeProfit) < tolerance {
			return '━'  // TP line
		} else if math.Abs(price-sig.EntryPrice) < tolerance {
			return '─'  // Entry line
		} else if math.Abs(price-sig.StopLoss) < tolerance {
			return '━'  // SL line
		}
		return ' '
	}
	
	// Candle visualization
	if withinBody {
		if candle.Close > candle.Open {
			return '█'  // Bullish body
		} else {
			return '▓'  // Bearish body
		}
	} else {
		return '│'  // Wick
	}
}

func (cv *ChartVisualizer) findOutcome(sig *engine.BacktestSignal) string {
	for i, candle := range sig.FutureCandles {
		hitTP := false
		hitSL := false
		
		if sig.Direction == patterns.DirectionLong {
			hitTP = candle.High >= sig.TakeProfit
			hitSL = candle.Low <= sig.StopLoss
		} else {
			hitTP = candle.Low <= sig.TakeProfit
			hitSL = candle.High >= sig.StopLoss
		}
		
		if hitTP && hitSL {
			return fmt.Sprintf("🔴 SL Hit (candle %d - both hit, SL first)", i)
		}
		if hitTP {
			return fmt.Sprintf("🟢 TP Hit (candle %d)", i)
		}
		if hitSL {
			return fmt.Sprintf("🔴 SL Hit (candle %d)", i)
		}
	}
	
	return "⏳ Timeout (neither TP nor SL hit)"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// VisualizeBestAndWorst creates charts for best and worst signals
func VisualizeBestAndWorst(signals []*engine.BacktestSignal, numCandles int) string {
	if len(signals) == 0 {
		return "No signals to visualize"
	}
	
	var output strings.Builder
	visualizer := NewChartVisualizer(80, 20)
	
	output.WriteString("\n╔" + strings.Repeat("═", 78) + "╗\n")
	output.WriteString("║" + center("📊 VISUAL SIGNAL ANALYSIS", 78) + "║\n")
	output.WriteString("╚" + strings.Repeat("═", 78) + "╝\n\n")
	
	// Show first few signals
	count := min(len(signals), 5)
	for i := 0; i < count; i++ {
		output.WriteString(fmt.Sprintf("\n[Signal %d/%d]\n", i+1, len(signals)))
		output.WriteString(visualizer.VisualizeSignal(signals[i], numCandles))
		output.WriteString("\n")
	}
	
	return output.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func center(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}