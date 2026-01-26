package simulation

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"set-and-trend/backend/internal/patterns"
)

// Simulation is the main orchestrator for the trading simulation game
type Simulation struct {
	executor *Executor
	config   SimulationConfig
	running  bool
}

// NewSimulation creates a new simulation with the given candles and config
func NewSimulation(candles []patterns.Candle, config SimulationConfig) *Simulation {
	return &Simulation{
		executor: NewExecutor(candles, config),
		config:   config,
		running:  false,
	}
}

// Run starts the simulation in the configured mode
func (s *Simulation) Run() error {
	s.running = true

	switch s.config.Mode {
	case ModeStep:
		return s.runStepMode()
	case ModeAuto:
		return s.runAutoMode()
	case ModeFast:
		return s.runFastMode()
	default:
		return s.runStepMode()
	}
}

// runStepMode runs interactively, waiting for user input between bars
func (s *Simulation) runStepMode() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n===============================================================================")
	fmt.Println("              H&S PATTERN TRADING SIMULATION - STEP MODE")
	fmt.Println("===============================================================================")
	fmt.Println("\nControls:")
	fmt.Println("  ENTER  - Next bar")
	fmt.Println("  f      - Fast forward (auto mode)")
	fmt.Println("  s      - Skip 10 bars")
	fmt.Println("  j      - Jump to specific bar")
	fmt.Println("  t      - Show trade table")
	fmt.Println("  q      - Quit")
	fmt.Println("\nPress ENTER to start...")
	reader.ReadString('\n')

	for s.running {
		output, done := s.executor.Step()
		fmt.Print(output)

		if done {
			break
		}

		fmt.Print("\n> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "q", "quit":
			s.running = false
			fmt.Println(s.executor.GetFinalReport())
		case "f":
			fmt.Println("Switching to auto mode...")
			s.config.Mode = ModeAuto
			return s.runAutoMode()
		case "s":
			for i := 0; i < 10 && s.executor.feeder.HasMoreBars(); i++ {
				s.executor.bot.ProcessBar()
			}
		case "j":
			fmt.Print("Jump to bar: ")
			var barNum int
			fmt.Scanf("%d", &barNum)
			if barNum > 0 && barNum < s.executor.feeder.TotalBars() {
				s.executor.SkipTo(barNum)
				fmt.Printf("Jumped to bar %d\n", barNum)
			}
		case "t":
			trades := s.executor.bot.state.ClosedTrades
			vis := s.executor.visualizer
			fmt.Print(vis.RenderTradeTable(trades, 10))
		default:
			// ENTER - continue to next bar
		}
	}

	return nil
}

// runAutoMode runs automatically with delays between bars
func (s *Simulation) runAutoMode() error {
	delay := time.Duration(s.config.AutoDelayMs) * time.Millisecond

	fmt.Println("\n===============================================================================")
	fmt.Println("              H&S PATTERN TRADING SIMULATION - AUTO MODE")
	fmt.Printf("              Running at %dms/bar. Press Ctrl+C to stop.\n", s.config.AutoDelayMs)
	fmt.Println("===============================================================================")

	for s.running && s.executor.feeder.HasMoreBars() {
		output, done := s.executor.Step()
		fmt.Print(output)

		if done {
			break
		}

		time.Sleep(delay)
	}

	fmt.Println(s.executor.GetFinalReport())
	return nil
}

// runFastMode runs as fast as possible without visualization
func (s *Simulation) runFastMode() error {
	fmt.Println("\n===============================================================================")
	fmt.Println("              H&S PATTERN TRADING SIMULATION - FAST MODE")
	fmt.Println("===============================================================================")
	fmt.Println("\nRunning simulation...")

	start := time.Now()

	for s.executor.feeder.HasMoreBars() {
		s.executor.bot.ProcessBar()
	}

	elapsed := time.Since(start)

	fmt.Printf("\nCompleted in %v\n", elapsed)
	fmt.Println(s.executor.GetFinalReport())

	return nil
}

// GetExecutor returns the executor for external access
func (s *Simulation) GetExecutor() *Executor {
	return s.executor
}

// Stop stops the running simulation
func (s *Simulation) Stop() {
	s.running = false
}

// ConvertDBCandles converts database candle format to patterns.Candle
func ConvertDBCandles(dbCandles []DBCandle) []patterns.Candle {
	result := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		result[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUTC,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
		}
	}
	return result
}

// DBCandle represents the database candle format
type DBCandle struct {
	ID           string
	TimestampUTC time.Time
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       int64
}
