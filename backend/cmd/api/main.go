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
	"set-and-trend/backend/internal/middleware"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
	"set-and-trend/backend/internal/services/auth"
)

func main() {
	ctx := context.Background()//root context

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config.Load:", err)
	}

	queries, pool, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		log.Fatal("database:", err)
	}
	defer pool.Close()


	//Repository Layer
	userRepo := repositories.NewUserRepository(queries)
	accountRepo := repositories.NewAccountRepository(queries)
	candleRepo := repositories.NewCandleRepository(queries)
	indicatorRepo := repositories.NewIndicatorRepository(queries)
	tradeRepo := repositories.NewTradeRepository(queries, pool)
	executionRepo := repositories.NewExecutionRepository(pool)
	intentRepo := repositories.NewIntentRepository(pool)
	feedbackRepo := repositories.NewFeedbackRepository(queries)
	analyticsRepo := repositories.NewAnalyticsRepository(pool)
	strategyRepo := repositories.NewStrategyRepository(pool)
	signalRepo := repositories.NewSignalRepository(pool)
	verdictRepo := repositories.NewVerdictRepository(pool)
	//service layer
	tradeService := services.NewTradeService(tradeRepo, accountRepo, candleRepo)
	executionService := services.NewExecutionService(tradeRepo, executionRepo, intentRepo, pool)
	authService := auth.NewAuthService(queries)
	//handler layer
	userHandler := handlers.NewUserHandler(authService)
	accountHandler := handlers.NewAccountHandler(accountRepo, userRepo)
	candleHandler := handlers.NewCandleHandler(candleRepo)
	indicatorHandler := handlers.NewIndicatorHandler(indicatorRepo, candleRepo)
	tradeHandler := handlers.NewTradeHandler(tradeService, tradeRepo, candleRepo)
	executionHandler := handlers.NewExecutionHandler(executionService, executionRepo)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackRepo, tradeRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo)
	strategyHandler := handlers.NewStrategyHandler(strategyRepo, signalRepo)
	verdictHandler := handlers.NewVerdictHandler(verdictRepo, signalRepo)
	
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://164.92.229.200:3000", "http://164.92.229.200:3001", "http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		// PUBLIC ROUTES (No auth required)
		api.POST("/auth/signup", userHandler.SignUp)
		api.POST("/auth/login", userHandler.Login)
		
		// PROTECTED ROUTES (Auth required)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Accounts
			protected.POST("/accounts", accountHandler.CreateAccount)
			
			// Candles (read-only, could be public but safer protected)
			protected.POST("/candles", candleHandler.CreateCandle)
			protected.GET("/candles/latest", candleHandler.GetLatestCandles)
			
			// Indicators
			protected.GET("/indicators/latest", indicatorHandler.GetLatestIndicators)
			protected.POST("/indicators/compute", indicatorHandler.ComputeIndicator)
			
			// Trades
			protected.GET("/trades", tradeHandler.ListTrades)
			protected.POST("/trades", tradeHandler.CreateTrade)
			protected.POST("/trades/simple", tradeHandler.CreateSimpleTrade)
			protected.POST("/trades/:id/execute", executionHandler.ExecuteTrade)
			protected.POST("/trades/:id/close", executionHandler.CloseTrade)
			protected.POST("/trades/:id/cancel", executionHandler.CancelTrade)
			protected.GET("/trades/:id/state", executionHandler.GetTradeState)
			protected.GET("/trades/:id/executions", executionHandler.GetTradeExecutions)
			
			// Trade Feedback
			protected.POST("/trades/:id/feedback", feedbackHandler.CreateFeedback)
			protected.GET("/trades/:id/feedback", feedbackHandler.GetFeedback)
			protected.PUT("/trades/:id/feedback", feedbackHandler.UpdateFeedback)
			protected.DELETE("/trades/:id/feedback", feedbackHandler.DeleteFeedback)
			
			// Strategies + signals (multi-strategy lab)
			protected.GET("/strategies", strategyHandler.ListStrategies)
			protected.GET("/strategies/:id/signals", strategyHandler.GetStrategySignals)
			protected.GET("/signals/active", strategyHandler.GetActiveSignals)
			protected.POST("/signals/:id/notified", strategyHandler.MarkSignalNotified)

			// AI orchestration layer (composer + specialist agents)
			protected.GET("/signals/unprocessed", verdictHandler.ListUnprocessedSignals)
			protected.POST("/signals/:id/composed", verdictHandler.MarkComposed)
			protected.POST("/signals/:id/verdicts", verdictHandler.CreateVerdict)
			protected.GET("/signals/:id/verdicts", verdictHandler.ListVerdicts)

			// Analytics
			protected.GET("/analytics/summary", analyticsHandler.GetSummary)
			protected.GET("/analytics/by-rule", analyticsHandler.GetByRule)
			protected.GET("/analytics/by-strategy", analyticsHandler.GetByStrategy)
			protected.GET("/analytics/by-session", analyticsHandler.GetBySession)
			protected.GET("/analytics/by-emotion", analyticsHandler.GetByEmotion)
		}
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