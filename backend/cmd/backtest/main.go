package main

import (
	"context"
	"log"
	
	"set-and-trend/backend/internal/backtest"
	"set-and-trend/backend/internal/db"
	
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()
	
	// ✅ Direct connection string
	connString := "postgresql://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()
	
	// Create queries instance
	queries := db.New(pool)
	
	log.Println("🎯 ELITE TOP-2/WEEK BACKTEST - PRODUCTION READY")
	
	// Run backtest
	backtest.Run(pool, queries)
}