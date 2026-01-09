package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	
	"github.com/shopspring/decimal" 
	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/db"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config.Load:", err)
	}

	queries, _, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal("database:", err)
	}

	fmt.Println("🚀 Computing EMAs for all candles...")

	// Get all candles ordered by time (oldest first)
	candles, err := queries.GetAllCandlesOrdered(ctx)
	if err != nil {
		log.Fatalf("Failed to get candles: %v", err)
	}

	fmt.Printf("📊 Found %d candles\n\n", len(candles))

	// Extract close prices
	var closePrices []float64
	for _, c := range candles {
		closeFloat, _ := strconv.ParseFloat(c.Close.String(), 64)
		closePrices = append(closePrices, closeFloat)
	}

	// Compute EMAs for each period
	ema20Values := computeEMA(closePrices, 20)
	ema50Values := computeEMA(closePrices, 50)
	ema200Values := computeEMA(closePrices, 200)

	// Update indicators for each candle
	successCount := 0
	for i, candle := range candles {
		// Get indicator for this candle
		indicator, err := queries.GetIndicatorByCandleID(ctx, candle.ID)
		if err != nil {
			log.Printf("⚠️  No indicator found for candle %s", candle.ID)
			continue
		}

		// Update with computed EMAs
		err = queries.UpdateIndicatorEMAs(ctx, db.UpdateIndicatorEMAsParams{
			ID:     indicator.ID,
			Ema20:  decimal.NewFromFloat(ema20Values[i]),
			Ema50:  decimal.NewFromFloat(ema50Values[i]),
			Ema200: decimal.NewFromFloat(ema200Values[i]),
		})
		if err != nil {
			log.Printf("❌ Failed to update indicator %s: %v", indicator.ID, err)
			continue
		}

		successCount++
		if (i+1)%50 == 0 {
			fmt.Printf("✅ Updated %d/%d indicators with EMAs\n", i+1, len(candles))
		}
	}

	fmt.Println("\n============================================================")
	fmt.Printf("✅ EMA Computation Complete!\n")
	fmt.Printf("📊 Updated: %d/%d indicators\n", successCount, len(candles))
	fmt.Println("============================================================")
}

// computeEMA calculates EMA for entire price series
func computeEMA(prices []float64, period int) []float64 {
	if len(prices) == 0 {
		return []float64{}
	}

	emas := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// Not enough data for first candles
	for i := 0; i < period && i < len(prices); i++ {
		emas[i] = 0 // Insufficient data
	}

	// Calculate initial SMA as first EMA
	if len(prices) >= period {
		sum := 0.0
		for i := 0; i < period; i++ {
			sum += prices[i]
		}
		emas[period-1] = sum / float64(period)

		// Calculate subsequent EMAs
		for i := period; i < len(prices); i++ {
			emas[i] = (prices[i] * multiplier) + (emas[i-1] * (1 - multiplier))
		}
	}

	return emas
}
