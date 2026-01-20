package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/handlers"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
	"set-and-trend/backend/internal/services/auth"
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
	authService := auth.NewAuthService(queries)
	//userHandler := handlers.NewUserHandler(userRepo)
	userHandler := handlers.NewUserHandler(authService)
	accountHandler := handlers.NewAccountHandler(accountRepo, userRepo)
	candleHandler := handlers.NewCandleHandler(candleRepo)
	indicatorHandler := handlers.NewIndicatorHandler(indicatorRepo, candleRepo)
	tradeHandler := handlers.NewTradeHandler(tradeService)
	executionHandler := handlers.NewExecutionHandler(executionService, executionRepo)
	
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://164.92.229.200:3000", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		// api.POST("/users", userHandler.CreateUser)
		api.POST("/auth/signup", userHandler.SignUp)
		api.POST("/auth/login", userHandler.Login)
		api.POST("/accounts", accountHandler.CreateAccount)
		api.POST("/candles", candleHandler.CreateCandle)
		api.GET("/candles/latest", candleHandler.GetLatestCandles)
		api.GET("/indicators/latest", indicatorHandler.GetLatestIndicators)
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
