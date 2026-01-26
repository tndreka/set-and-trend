package simulation

import (
	"fmt"
	"math"
	"strings"

	"set-and-trend/backend/internal/patterns"
)

// Visualizer handles ASCII chart rendering
type Visualizer struct {
	config    SimulationConfig
	width     int
	height    int
}

// NewVisualizer creates a new chart visualizer
func NewVisualizer(config SimulationConfig) *Visualizer {
	return &Visualizer{
		config: config,
		width:  80,
		height: 20,
	}
}

// RenderChart creates an ASCII representation of price action
func (v *Visualizer) RenderChart(candles []patterns.Candle, state *SimulationState, currentBar *patterns.Candle) string {
	if len(candles) == 0 {
		return ""
	}
	
	var sb strings.Builder
	
	// Clear screen effect (ANSI escape)
	sb.WriteString("\033[2J\033[H")
	
	// Find price range
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
	
	// Add some padding
	priceRange := maxPrice - minPrice
	minPrice -= priceRange * 0.05
	maxPrice += priceRange * 0.05
	priceRange = maxPrice - minPrice
	
	// Create chart grid
	chartWidth := min(len(candles), v.width-10)
	chart := make([][]rune, v.height)
	for i := range chart {
		chart[i] = make([]rune, chartWidth)
		for j := range chart[i] {
			chart[i][j] = ' '
		}
	}
	
	// Plot candles (simplified - just use close price dots)
	startIdx := 0
	if len(candles) > chartWidth {
		startIdx = len(candles) - chartWidth
	}
	
	for i := startIdx; i < len(candles); i++ {
		c := candles[i]
		col := i - startIdx
		
		// Calculate row positions
		highRow := v.priceToRow(c.High, minPrice, priceRange)
		lowRow := v.priceToRow(c.Low, minPrice, priceRange)
		closeRow := v.priceToRow(c.Close, minPrice, priceRange)
		_ = highRow // Suppress unused warning
		
		// Draw high-low range
		for row := lowRow; row >= highRow; row-- {
			if row >= 0 && row < v.height {
				chart[row][col] = '|'
			}
		}
		
		// Mark close/open with directional character
		if closeRow >= 0 && closeRow < v.height {
			if c.Close > c.Open {
				chart[closeRow][col] = '^' // Bullish
			} else if c.Close < c.Open {
				chart[closeRow][col] = 'v' // Bearish
			} else {
				chart[closeRow][col] = '-' // Doji
			}
		}
	}
	
	// Mark trade entry/exit if in position
	if state.OpenPosition != nil {
		pos := state.OpenPosition
		
		// Entry level
		entryRow := v.priceToRow(pos.EntryPrice, minPrice, priceRange)
		if entryRow >= 0 && entryRow < v.height {
			for i := 0; i < chartWidth; i++ {
				if chart[entryRow][i] == ' ' {
					chart[entryRow][i] = '-'
				}
			}
		}
		
		// SL level
		slRow := v.priceToRow(pos.StopLoss, minPrice, priceRange)
		if slRow >= 0 && slRow < v.height {
			for i := 0; i < chartWidth; i++ {
				if chart[slRow][i] == ' ' {
					chart[slRow][i] = '.'
				}
			}
		}
		
		// TP level
		tpRow := v.priceToRow(pos.TakeProfit, minPrice, priceRange)
		if tpRow >= 0 && tpRow < v.height {
			for i := 0; i < chartWidth; i++ {
				if chart[tpRow][i] == ' ' {
					chart[tpRow][i] = '.'
				}
			}
		}
	}
	
	// Header
	sb.WriteString("===============================================================================\n")
	
	// Render chart with price axis
	for row := 0; row < v.height; row++ {
		// Price label (every 4 rows)
		if row%4 == 0 {
			price := v.rowToPrice(row, minPrice, priceRange)
			sb.WriteString(fmt.Sprintf("%7.5f ", price))
		} else {
			sb.WriteString("        ")
		}
		
		// Chart row
		sb.WriteString(string(chart[row]))
		
		// Right side annotations
		if state.OpenPosition != nil {
			pos := state.OpenPosition
			if row == v.priceToRow(pos.EntryPrice, minPrice, priceRange) {
				sb.WriteString(fmt.Sprintf(" < ENTRY %.5f", pos.EntryPrice))
			} else if row == v.priceToRow(pos.StopLoss, minPrice, priceRange) {
				sb.WriteString(fmt.Sprintf(" < SL %.5f", pos.StopLoss))
			} else if row == v.priceToRow(pos.TakeProfit, minPrice, priceRange) {
				sb.WriteString(fmt.Sprintf(" < TP %.5f", pos.TakeProfit))
			}
		}
		
		sb.WriteString("\n")
	}
	
	sb.WriteString("===============================================================================\n")
	
	// Current price
	if currentBar != nil {
		sb.WriteString(fmt.Sprintf("Current: %.5f | O: %.5f H: %.5f L: %.5f C: %.5f\n",
			currentBar.Close, currentBar.Open, currentBar.High, currentBar.Low, currentBar.Close))
	}
	
	return sb.String()
}

// priceToRow converts price to chart row (0 = top)
func (v *Visualizer) priceToRow(price, minPrice, priceRange float64) int {
	if priceRange == 0 {
		return v.height / 2
	}
	// Invert: high prices at top (row 0)
	normalized := (price - minPrice) / priceRange
	row := int((1 - normalized) * float64(v.height-1))
	return max(0, min(v.height-1, row))
}

// rowToPrice converts chart row to price
func (v *Visualizer) rowToPrice(row int, minPrice, priceRange float64) float64 {
	normalized := 1 - (float64(row) / float64(v.height-1))
	return minPrice + (normalized * priceRange)
}

// RenderTradeTable shows recent trades in table format
func (v *Visualizer) RenderTradeTable(trades []TradeResult, limit int) string {
	var sb strings.Builder
	
	sb.WriteString("\nRECENT TRADES:\n")
	sb.WriteString("-----------------------------------------------------------------------------\n")
	sb.WriteString("| # | Type | Dir   | Entry   | Exit    | P&L (R) | Bars | Result     |\n")
	sb.WriteString("-----------------------------------------------------------------------------\n")
	
	start := 0
	if len(trades) > limit {
		start = len(trades) - limit
	}
	
	for i := start; i < len(trades); i++ {
		t := trades[i]
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %.5f | %.5f | %+.2f   | %2d   | %-10s |\n",
			i+1,
			t.PatternType,
			t.Direction,
			t.EntryPrice,
			t.ExitPrice,
			t.PnLR,
			t.BarsHeld,
			t.Outcome,
		))
	}
	
	sb.WriteString("-----------------------------------------------------------------------------\n")
	
	return sb.String()
}

// RenderProgressBar shows simulation progress
func (v *Visualizer) RenderProgressBar(current, total int) string {
	percent := float64(current) / float64(total) * 100
	filled := int(math.Round(percent / 2))
	
	bar := strings.Repeat("#", filled) + strings.Repeat("-", 50-filled)
	return fmt.Sprintf("[%s] %.1f%%", bar, percent)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
