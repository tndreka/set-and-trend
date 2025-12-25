package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/handlers"
	"set-and-trend/backend/internal/repositories"
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
	
	log.Println("✓ SQLC connected via .env")
	
	// ✅ BOTH repos use queries (NOT pool)
	userRepo := repositories.NewUserRepository(queries)
	userHandler := handlers.NewUserHandler(userRepo)
	
	accountRepo := repositories.NewAccountRepository(queries)
	accountHandler := handlers.NewAccountHandler(accountRepo, userRepo)
 
	r := gin.Default()
	api := r.Group("/api")
	{
		api.POST("/users", userHandler.CreateUser)
		api.POST("/accounts", accountHandler.CreateAccount)
	}
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
	})
	
	log.Printf("🚀 http://localhost:%d", cfg.Port)
	log.Fatal(r.Run(fmt.Sprintf(":%d", cfg.Port)))
}
