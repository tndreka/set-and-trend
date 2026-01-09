package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
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

	candleID := uuid.MustParse("a5f889bf-b77d-46b0-a953-4a05f2c1d037")

	fmt.Printf("🧪 Testing rule evaluation for candle: %s\n\n", candleID)

	err = ruleService.EvaluateCandle(ctx, candleID)
	if err != nil {
		log.Fatalf("❌ Evaluation failed: %v", err)
	}

	fmt.Println("✅ Rule evaluation completed successfully!")
	
	// Check results
	results, err := ruleResultRepo.GetRuleResultsByCandleID(ctx, candleID)
	if err != nil {
		log.Fatalf("❌ Failed to fetch results: %v", err)
	}

	fmt.Printf("\n📊 Results for candle %s:\n", candleID)
	for _, r := range results {
		fmt.Printf("  - %s: %s (confidence: %.2f)\n", r.RuleCode, r.Result, r.ConfidenceScore.InexactFloat64())
	}
}
