package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/handlers"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config.Load:", err)
	}

	queries, pool, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal("database:", err)
	}
	defer pool.Close()

	log.Println("✅ Database connected with 100 connection pool")

	userRepo := repositories.NewUserRepository(queries)
	accountRepo := repositories.NewAccountRepository(queries)
	candleRepo := repositories.NewCandleRepository(queries)
	indicatorRepo := repositories.NewIndicatorRepository(queries)
	tradeRepo := repositories.NewTradeRepository(queries)
	executionRepo := repositories.NewExecutionRepository(pool)
	intentRepo := repositories.NewIntentRepository(pool)

	tradeService := services.NewTradeService(tradeRepo, accountRepo, candleRepo)
	executionService := services.NewExecutionService(tradeRepo, executionRepo, intentRepo, pool)

	userHandler := handlers.NewUserHandler(userRepo)
	accountHandler := handlers.NewAccountHandler(accountRepo, userRepo)
	candleHandler := handlers.NewCandleHandler(candleRepo)
	indicatorHandler := handlers.NewIndicatorHandler(indicatorRepo, candleRepo)
	tradeHandler := handlers.NewTradeHandler(tradeService)
	executionHandler := handlers.NewExecutionHandler(executionService, executionRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/users", userHandler.CreateUser)
		api.POST("/accounts", accountHandler.CreateAccount)
		api.POST("/candles", candleHandler.CreateCandle)
		api.GET("/candles/latest", candleHandler.GetLatestCandles)
		api.POST("/indicators/compute", indicatorHandler.ComputeIndicator)
		api.POST("/trades", tradeHandler.CreateTrade)
		api.POST("/trades/:id/execute", executionHandler.ExecuteTrade)
		api.POST("/trades/:id/close", executionHandler.CloseTrade)
		api.POST("/trades/:id/cancel", executionHandler.CancelTrade)
		api.GET("/trades/:id/state", executionHandler.GetTradeState)
		api.GET("/trades/:id/executions", executionHandler.GetTradeExecutions)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Port),
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("🚀 Server ready for 10k+ requests on :%d", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
