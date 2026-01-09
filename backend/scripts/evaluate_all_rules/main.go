package main

import (
	"context"
	"fmt"
	"log"

	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
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

	// Initialize repositories
	candleRepo := repositories.NewCandleRepository(queries)
	indicatorRepo := repositories.NewIndicatorRepository(queries)
	ruleResultRepo := repositories.NewRuleResultRepository(queries)

	// Initialize service
	ruleService := services.NewRuleEvaluationService(
		candleRepo,
		indicatorRepo,
		ruleResultRepo,
	)

	fmt.Println("🚀 Evaluating rules for all candles...")

	// Get all candles
	candles, err := candleRepo.GetLatestCandles(ctx, 10000) // Or use GetAllCandlesOrdered
	if err != nil {
		log.Fatalf("Failed to get candles: %v", err)
	}

	fmt.Printf("📊 Found %d candles\n\n", len(candles))

	successCount := 0
	errorCount := 0

	for i, candle := range candles {
		err := ruleService.EvaluateCandle(ctx, candle.ID)
		if err != nil {
			log.Printf("❌ Failed to evaluate candle %s: %v", candle.ID, err)
			errorCount++
			continue
		}

		successCount++
		if (i+1)%50 == 0 {
			fmt.Printf("✅ Evaluated %d/%d candles\n", i+1, len(candles))
		}
	}

	fmt.Println("\n============================================================")
	fmt.Printf("✅ Rule Evaluation Complete!\n")
	fmt.Printf("📊 Success: %d/%d\n", successCount, len(candles))
	fmt.Printf("❌ Errors: %d\n", errorCount)
	fmt.Println("============================================================")
}
