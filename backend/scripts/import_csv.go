package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config.Load:", err)
	}

	// Connect to database
	queries, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal("database:", err)
	}

	candleRepo := repositories.NewCandleRepository(queries)
	indicatorRepo := repositories.NewIndicatorRepository(queries)

	// Open CSV file - UPDATE THIS PATH IF NEEDED
	csvPath := "/home/set-and-trend/backend/mt4_ready/EURUSD_weekly_2015_2025.csv"
	file, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("Failed to open CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read CSV header: %v", err)
	}
	fmt.Printf("📋 CSV Header: %v\n\n", header)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV: %v", err)
	}

	fmt.Printf("🚀 Starting import of %d candles from 2015-2025\n\n", len(records))

	successCount := 0
	errorCount := 0

	for i, record := range records {
		// CSV format: DateTime,Open,High,Low,Close,Volume,EMA12,EMA26
		if len(record) < 6 {
			log.Printf("❌ Row %d: Invalid format (too few columns)\n", i+2)
			errorCount++
			continue
		}

		dateStr := record[0]
		openStr := record[1]
		highStr := record[2]
		lowStr := record[3]
		closeStr := record[4]
		volumeStr := record[5]

		// Parse timestamp (format: 2015-01-04)
		timestamp, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("❌ Row %d: Failed to parse date '%s': %v\n", i+2, dateStr, err)
			errorCount++
			continue
		}

		// Parse volume
		volume, err := strconv.ParseInt(volumeStr, 10, 64)
		if err != nil {
			log.Printf("⚠️  Row %d: Invalid volume '%s', using 0\n", i+2, volumeStr)
			volume = 0
		}

		// Insert candle
		candle, err := candleRepo.CreateCandle(ctx, repositories.CandleCreateParams{
			ID:           uuid.New(),
			TimestampUTC: timestamp,
			Open:         openStr,
			High:         highStr,
			Low:          lowStr,
			Close:        closeStr,
			Volume:       &volume,
		})
		if err != nil {
			log.Printf("❌ Row %d: Failed to insert candle %s: %v\n", i+2, dateStr, err)
			errorCount++
			continue
		}

		// Parse OHLC for indicators
		open, _ := strconv.ParseFloat(openStr, 64)
		high, _ := strconv.ParseFloat(highStr, 64)
		low, _ := strconv.ParseFloat(lowStr, 64)
		close, _ := strconv.ParseFloat(closeStr, 64)

		// Compute basic indicators
		indicators := services.ComputeBasicIndicators(services.Candle{
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: &volume,
		})

		// Insert indicators
		_, err = indicatorRepo.CreateIndicator(ctx, repositories.IndicatorCreateParams{
			ID:                 uuid.New(),
			CandleID:           candle.ID,
			EMA20:              indicators.EMA20,
			EMA50:              indicators.EMA50,
			EMA200:             indicators.EMA200,
			RangeSize:          indicators.RangeSize,
			BodySize:           indicators.BodySize,
			UpperWick:          indicators.UpperWick,
			LowerWick:          indicators.LowerWick,
			MidPrice:           indicators.MidPrice,
			LastSwingHighPrice: indicators.LastSwingHighPrice,
			LastSwingLowPrice:  indicators.LastSwingLowPrice,
		})
		if err != nil {
			log.Printf("⚠️  Row %d: Failed to insert indicators for %s: %v\n", i+2, dateStr, err)
		}

		successCount++
		
		// Progress indicator every 50 candles
		if (i+1)%50 == 0 {
			fmt.Printf("✅ Processed %d/%d candles...\n", i+1, len(records))
		}
	}

	fmt.Printf("\n=====================================================================")
	fmt.Printf("✅ Import Complete!\n")
	fmt.Printf("📊 Total rows: %d\n", len(records))
	fmt.Printf("✅ Successful: %d\n", successCount)
	fmt.Printf("❌ Errors: %d\n", errorCount)
	fmt.Printf("📅 Date range: 2015-2025 (10 years of weekly data)\n")
	fmt.Printf("\n========================================================================")
}
