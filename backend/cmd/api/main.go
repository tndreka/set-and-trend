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

	queries, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal("database:", err)
	}

	log.Println("✓ Database connected with 100 connection pool")

	userRepo := repositories.NewUserRepository(queries)
	accountRepo := repositories.NewAccountRepository(queries)
	userHandler := handlers.NewUserHandler(userRepo)
	accountHandler := handlers.NewAccountHandler(accountRepo, userRepo)
	candleRepo := repositories.NewCandleRepository(queries)
	candleHandler := handlers.NewCandleHandler(candleRepo)
	indicatorRepo := repositories.NewIndicatorRepository(queries)
	indicatorHandler := handlers.NewIndicatorHandler(indicatorRepo, candleRepo)
	tradeRepo := repositories.NewTradeRepository(queries)
	tradeService := services.NewTradeService(tradeRepo, accountRepo, candleRepo)
	tradeHandler := handlers.NewTradeHandler(tradeService)

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
