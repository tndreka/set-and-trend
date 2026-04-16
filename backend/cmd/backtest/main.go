package main

import (
	"context"
	"flag"
	"log"

	"set-and-trend/backend/internal/backtest"
	"set-and-trend/backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

func main() {
	engine := flag.String("engine", "legacy", "backtest engine: legacy (detector consensus) or saf (Set-and-Forget H4 rules)")
	iter := flag.Int("iter", 0, "iteration number (saf only)")
	iterName := flag.String("name", "baseline", "iteration short name (saf only)")
	outPath := flag.String("out", "results/iter_00_baseline.json", "output JSON path (saf only)")
	allSymbols := flag.Bool("all-symbols", true, "saf: run every enabled rule against every pair in candles_h4 (research mode)")
	flag.Parse()

	ctx := context.Background()
	connString := "postgresql://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	switch *engine {
	case "legacy":
		log.Println("🎯 LEGACY detector-consensus backtest")
		backtest.Run(pool, queries)
	case "saf":
		log.Println("🎯 SaF (Set-and-Forget) rule-engine backtest")
		if err := backtest.RunSaF(ctx, pool, *iter, *iterName, *outPath, *allSymbols); err != nil {
			log.Fatalf("SaF backtest failed: %v", err)
		}
	default:
		log.Fatalf("unknown engine: %s (expected 'legacy' or 'saf')", *engine)
	}
}
