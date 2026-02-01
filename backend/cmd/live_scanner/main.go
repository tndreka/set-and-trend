package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"set-and-trend/backend/internal/confluence"
	"set-and-trend/backend/internal/db"
	"set-and-trend/backend/internal/patterns"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

var MAJOR_PAIRS = []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD"}

const (
	// 50% threshold = ~2.5 signals/week (optimal for 2-3 trades)
	CONFLUENCE_THRESHOLD = 50.0
	CHECK_INTERVAL       = 4 * time.Hour // H4 candle closes
)

// MT5 Demo Account Info
const (
	MT5_SERVER  = "MetaQuotes-Demo"
	MT5_LOGIN   = "5045820618"
	MT5_ACCOUNT = "Forex Hedged EUR"
)

// ============================================================================
// TELEGRAM INTEGRATION
// ============================================================================

type TelegramBot struct {
	token  string
	chatID string
}

func NewTelegramBot() *TelegramBot {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	
	if token == "" || chatID == "" {
		log.Println("⚠️  Telegram not configured. Set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID")
		return nil
	}
	
	return &TelegramBot{token: token, chatID: chatID}
}

func (t *TelegramBot) SendMessage(text string) error {
	if t == nil {
		return nil
	}
	
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	
	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram error: %s", string(respBody))
	}
	
	return nil
}

func (t *TelegramBot) SendSignal(symbol, direction string, entry, confluence float64, score *confluence.ChecklistScore) error {
	emoji := "🔴"
	if direction == "LONG" {
		emoji = "🟢"
	}
	
	msg := fmt.Sprintf(`%s <b>TRADE SIGNAL</b> %s

<b>Symbol:</b> %s
<b>Direction:</b> %s
<b>Confluence:</b> %.1f%%

<b>Entry Zone:</b> %.5f

<b>Checklist Breakdown:</b>
━━━━━━━━━━━━━━━━━━
W1: %.1f%% (Favor + AOI + EMA + Struct + Pattern)
D1: %.1f%% (Favor + AOI + EMA + Struct + Pattern)
H4: %.1f%% (EMA + Struct + Pattern)

<i>Account: %s @ %s</i>
<i>Time: %s UTC</i>

⚠️ Set SL above/below recent swing high/low`,
		emoji, emoji,
		symbol,
		direction,
		confluence,
		entry,
		(score.WeeklyFavor+score.WeeklyAOI+score.WeeklyEMA+score.WeeklyStruct+score.WeeklyPattern)*100,
		(score.DailyFavor+score.DailyAOI+score.DailyEMA+score.DailyStruct+score.DailyPattern)*100,
		(score.H4EMA+score.H4Struct+score.H4Pattern)*100,
		MT5_ACCOUNT, MT5_SERVER,
		time.Now().UTC().Format("2006-01-02 15:04"),
	)
	
	return t.SendMessage(msg)
}

// ============================================================================
// SIGNAL LOGGING
// ============================================================================

type SignalLog struct {
	file *os.File
}

func NewSignalLog(path string) (*SignalLog, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &SignalLog{file: f}, nil
}

func (s *SignalLog) Log(symbol, direction string, entry, confluence float64) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("%s | %s %s | Entry: %.5f | Conf: %.1f%%\n",
		timestamp, symbol, direction, entry, confluence)
	s.file.WriteString(line)
	s.file.Sync()
}

func (s *SignalLog) Close() {
	s.file.Close()
}

// ============================================================================
// SCANNER
// ============================================================================

type LiveScanner struct {
	pool        *pgxpool.Pool
	queries     *db.Queries
	scorer      *confluence.ConfluenceScorer
	telegram    *TelegramBot
	signalLog   *SignalLog
	lastSignals map[string]time.Time // Prevent duplicate signals
	signalCount int
	startTime   time.Time
}

func NewLiveScanner(pool *pgxpool.Pool) (*LiveScanner, error) {
	signalLog, err := NewSignalLog("live_signals.log")
	if err != nil {
		return nil, err
	}
	
	config := confluence.ConfluenceConfig{
		MinConfluencePercent: CONFLUENCE_THRESHOLD,
	}
	
	return &LiveScanner{
		pool:        pool,
		queries:     db.New(pool),
		scorer:      confluence.NewConfluenceScorer(config),
		telegram:    NewTelegramBot(),
		signalLog:   signalLog,
		lastSignals: make(map[string]time.Time),
		startTime:   time.Now(),
	}, nil
}

func (s *LiveScanner) ScanSymbol(ctx context.Context, symbol string) {
	// Check cooldown (no duplicate signals within 24h for same symbol)
	if lastTime, exists := s.lastSignals[symbol]; exists {
		if time.Since(lastTime) < 24*time.Hour {
			return
		}
	}
	
	// Load candles
	weeklyCandles, err := s.loadWeeklyCandles(ctx, symbol, 60)
	if err != nil || len(weeklyCandles) < 30 {
		return
	}
	
	dailyCandles, err := s.loadDailyCandles(ctx, symbol, 300)
	if err != nil || len(dailyCandles) < 60 {
		return
	}
	
	h4Candles, err := s.loadH4Candles(ctx, symbol, 500)
	if err != nil || len(h4Candles) < 120 {
		return
	}
	
	// Calculate confluence using the existing scorer
	score, signal := s.scorer.CalculateConfluence(
		weeklyCandles, dailyCandles, h4Candles,
		nil, nil, nil, // No H2, H1, M30 yet
		symbol,
	)
	
	if signal != nil && score.TotalPercent >= CONFLUENCE_THRESHOLD {
		currentPrice := h4Candles[len(h4Candles)-1].Close
		direction := "LONG"
		if signal.Direction == "short" || signal.Direction == "SHORT" {
			direction = "SHORT"
		}
		
		s.emitSignal(symbol, direction, currentPrice, score.TotalPercent, score)
	}
}

func (s *LiveScanner) emitSignal(symbol, direction string, entry, confluence float64, score *confluence.ChecklistScore) {
	// Log to file
	s.signalLog.Log(symbol, direction, entry, confluence)
	
	// Calculate TF scores
	weeklyScore := (score.WeeklyFavor + score.WeeklyAOI + score.WeeklyEMA + score.WeeklyStruct + score.WeeklyPattern) * 100
	dailyScore := (score.DailyFavor + score.DailyAOI + score.DailyEMA + score.DailyStruct + score.DailyPattern) * 100
	h4Score := (score.H4EMA + score.H4Struct + score.H4Pattern) * 100
	
	// Print to console
	fmt.Printf("\n🚨 SIGNAL: %s %s @ %.5f | Confluence: %.1f%%\n", symbol, direction, entry, confluence)
	fmt.Printf("   W1: %.1f%% | D1: %.1f%% | H4: %.1f%%\n", weeklyScore, dailyScore, h4Score)
	
	// Send to Telegram
	if s.telegram != nil {
		if err := s.telegram.SendSignal(symbol, direction, entry, confluence, score); err != nil {
			log.Printf("⚠️  Telegram error: %v", err)
		} else {
			fmt.Println("   ✅ Sent to Telegram")
		}
	}
	
	// Update cooldown
	s.lastSignals[symbol] = time.Now()
	s.signalCount++
}

func (s *LiveScanner) Run(ctx context.Context) {
	ticker := time.NewTicker(CHECK_INTERVAL)
	defer ticker.Stop()
	
	// Run immediately on start
	s.scan(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *LiveScanner) scan(ctx context.Context) {
	fmt.Printf("\n[%s] Scanning %d pairs...\n", 
		time.Now().UTC().Format("2006-01-02 15:04"), len(MAJOR_PAIRS))
	
	for _, symbol := range MAJOR_PAIRS {
		s.ScanSymbol(ctx, symbol)
	}
	
	fmt.Printf("Scan complete. Next scan in %v\n", CHECK_INTERVAL)
}

func (s *LiveScanner) PrintSummary() {
	elapsed := time.Since(s.startTime)
	fmt.Println("\n================================================================================")
	fmt.Println("           LIVE SCANNER SESSION SUMMARY")
	fmt.Println("================================================================================")
	fmt.Printf("Duration: %v\n", elapsed.Round(time.Second))
	fmt.Printf("Total Signals: %d\n", s.signalCount)
	if elapsed.Hours() > 0 {
		fmt.Printf("Signals/Hour: %.2f\n", float64(s.signalCount)/elapsed.Hours())
		fmt.Printf("Projected Signals/Week: %.1f\n", float64(s.signalCount)/elapsed.Hours()*168)
	}
	fmt.Println("\nSignals logged to: live_signals.log")
}

func (s *LiveScanner) Close() {
	s.signalLog.Close()
}

// ============================================================================
// CANDLE LOADERS
// ============================================================================

func (s *LiveScanner) loadWeeklyCandles(ctx context.Context, symbol string, limit int) ([]patterns.Candle, error) {
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(-3, 0, 0))

	dbCandles, err := s.queries.GetCandlesWeeklyBySymbolAndRange(ctx, db.GetCandlesWeeklyBySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func (s *LiveScanner) loadDailyCandles(ctx context.Context, symbol string, limit int) ([]patterns.Candle, error) {
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(-2, 0, 0))

	dbCandles, err := s.queries.GetCandlesD1BySymbolAndRange(ctx, db.GetCandlesD1BySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func (s *LiveScanner) loadH4Candles(ctx context.Context, symbol string, limit int) ([]patterns.Candle, error) {
	endTime := pgtype.Timestamptz{}
	endTime.Scan(time.Now().UTC())
	startTime := pgtype.Timestamptz{}
	startTime.Scan(time.Now().UTC().AddDate(0, -6, 0))

	dbCandles, err := s.queries.GetCandlesH4BySymbolAndRange(ctx, db.GetCandlesH4BySymbolAndRangeParams{
		Symbol:         symbol,
		TimestampUtc:   startTime,
		TimestampUtc_2: endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(dbCandles) > limit {
		dbCandles = dbCandles[len(dbCandles)-limit:]
	}

	candles := make([]patterns.Candle, len(dbCandles))
	for i, c := range dbCandles {
		candles[i] = patterns.Candle{
			Index:     i,
			Timestamp: c.TimestampUtc.Time,
			Open:      c.Open.InexactFloat64(),
			High:      c.High.InexactFloat64(),
			Low:       c.Low.InexactFloat64(),
			Close:     c.Close.InexactFloat64(),
			Volume:    volumeToInt64(c.Volume),
		}
	}
	return candles, nil
}

func volumeToInt64(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	fmt.Println("================================================================================")
	fmt.Println("       LIVE TRADING SCANNER - SET & TREND")
	fmt.Println("================================================================================")
	fmt.Printf("MT5 Account: %s @ %s\n", MT5_LOGIN, MT5_SERVER)
	fmt.Printf("Threshold: %.0f%% | Interval: %v | Target: 2-3 signals/week\n", 
		CONFLUENCE_THRESHOLD, CHECK_INTERVAL)
	fmt.Printf("Pairs: %v\n", MAJOR_PAIRS)
	fmt.Println("================================================================================")
	
	// Check Telegram config
	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		fmt.Println("\n⚠️  TELEGRAM NOT CONFIGURED")
		fmt.Println("To enable Telegram alerts, set:")
		fmt.Println("  export TELEGRAM_BOT_TOKEN=your_bot_token")
		fmt.Println("  export TELEGRAM_CHAT_ID=your_chat_id")
		fmt.Println("")
	} else {
		fmt.Println("✅ Telegram configured")
	}
	
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://stt_user:taulantdhe42H%40%24D@localhost:5432/set_the_trend?sslmode=disable"
	}
	
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	
	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	fmt.Println("✅ Database connected")
	
	scanner, err := NewLiveScanner(pool)
	if err != nil {
		log.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()
	
	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down...")
		cancel()
	}()
	
	fmt.Println("\n🚀 Scanner started. Press Ctrl+C to stop.")
	scanner.Run(ctx)
	
	scanner.PrintSummary()
}
