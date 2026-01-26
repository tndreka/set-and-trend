package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"set-and-trend/backend/internal/config"
	"set-and-trend/backend/internal/patterns"
	"set-and-trend/backend/internal/simulation"
)

func main() {
	// Parse command line flags
	mode := flag.String("mode", "step", "Simulation mode: step, auto, fast")
	startDate := flag.String("start", "2015-01-01", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end", "2025-12-31", "End date (YYYY-MM-DD)")
	confidence := flag.Float64("confidence", 0.55, "Minimum confidence threshold (0.0-1.0)")
	minRR := flag.Float64("min-rr", 1.5, "Minimum risk/reward ratio")
	maxBars := flag.Int("max-bars", 20, "Maximum bars to hold position")
	cooldown := flag.Int("cooldown", 10, "Cooldown bars between trades")
	autoDelay := flag.Int("delay", 500, "Delay in ms between bars (auto mode)")
	noChart := flag.Bool("no-chart", false, "Disable chart visualization")
	flag.Parse()

	fmt.Println("\n===============================================================================")
	fmt.Println("              H&S PATTERN TRADING SIMULATION")
	fmt.Println("===============================================================================")
	fmt.Println("\nInitializing...")

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Invalid end date: %v", err)
	}

	// Determine simulation mode
	var simMode simulation.SimulationMode
	switch *mode {
	case "step":
		simMode = simulation.ModeStep
	case "auto":
		simMode = simulation.ModeAuto
	case "fast":
		simMode = simulation.ModeFast
	default:
		log.Fatalf("Invalid mode: %s. Use step, auto, or fast", *mode)
	}

	// Build configuration
	simConfig := simulation.SimulationConfig{
		Mode:                simMode,
		StartDate:           start,
		EndDate:             end,
		ConfidenceThreshold: *confidence,
		MinRR:               *minRR,
		MaxBarsToHold:       *maxBars,
		Lookback:            3,
		CooldownBars:        *cooldown,
		AutoDelayMs:         *autoDelay,
		Symbol:              "EURUSD",
		Timeframe:           patterns.TF_W1,
		ShowChart:           !*noChart,
	}

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Mode: %s\n", simConfig.Mode)
	fmt.Printf("  Period: %s to %s\n", simConfig.StartDate.Format("2006-01-02"), simConfig.EndDate.Format("2006-01-02"))
	fmt.Printf("  Confidence Threshold: %.0f%%\n", simConfig.ConfidenceThreshold*100)
	fmt.Printf("  Min R:R: %.2f\n", simConfig.MinRR)
	fmt.Printf("  Max Bars to Hold: %d\n", simConfig.MaxBarsToHold)
	fmt.Printf("  Cooldown: %d bars\n\n", simConfig.CooldownBars)

	// Load candles from database
	candles, err := loadCandlesFromDB(simConfig.StartDate, simConfig.EndDate)
	if err != nil {
		log.Fatalf("Failed to load candles: %v", err)
	}

	if len(candles) == 0 {
		log.Fatal("No candles found for the specified date range")
	}

	fmt.Printf("Loaded %d candles from database\n", len(candles))

	// Create and run simulation
	sim := simulation.NewSimulation(candles, simConfig)
	
	if err := sim.Run(); err != nil {
		log.Fatalf("Simulation error: %v", err)
	}
}

// loadCandlesFromDB fetches candles from the database
func loadCandlesFromDB(start, end time.Time) ([]patterns.Candle, error) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load: %w", err)
	}

	_, pool, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database connection: %w", err)
	}
	defer pool.Close()

	// Fetch candles from the weekly table
	rows, err := pool.Query(ctx, `
		SELECT id, timestamp_utc, open, high, low, close, COALESCE(volume, 0)
		FROM candles_weekly
		WHERE timestamp_utc >= $1 AND timestamp_utc <= $2
		ORDER BY timestamp_utc ASC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	var candles []patterns.Candle
	idx := 0
	for rows.Next() {
		var id string
		var timestamp time.Time
		var open, high, low, close float64
		var volume int64

		if err := rows.Scan(&id, &timestamp, &open, &high, &low, &close, &volume); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		candles = append(candles, patterns.Candle{
			Index:     idx,
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
		idx++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return candles, nil
}

func init() {
	// Set working directory if running from a different location
	if envDir := os.Getenv("APP_DIR"); envDir != "" {
		os.Chdir(envDir)
	}
}
