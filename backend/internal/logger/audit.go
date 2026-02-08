package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

type CandleLog struct {
	Timestamp       time.Time              `json:"timestamp"`
	Sequence        int64                  `json:"sequence"`
	Symbol          string                 `json:"symbol"`
	Timeframe       string                 `json:"timeframe"`
	BarIndex        int                    `json:"bar_index"`
	Open            float64                `json:"open"`
	High            float64                `json:"high"`
	Low             float64                `json:"low"`
	Close           float64                `json:"close"`
	Volume          int64                  `json:"volume"`
	Patterns        []PatternLog           `json:"patterns_detected"`
	ChecklistScore  *ChecklistLog          `json:"checklist_score,omitempty"`
	SignalDecision  string                 `json:"signal_decision"`
	RejectReason    string                 `json:"reject_reason,omitempty"`
	Confidence      float64                `json:"confidence"`
	OpenPosition    bool                   `json:"open_position"`
	PositionSide    string                 `json:"position_side,omitempty"`
	UnrealizedPnL   float64                `json:"unrealized_pnl,omitempty"`
	BalanceSnapshot float64                `json:"balance_snapshot"`
	EquitySnapshot  float64                `json:"equity_snapshot"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type PatternLog struct {
	Type               string  `json:"type"`
	Confidence         float64 `json:"confidence"`
	HeadPrice          float64 `json:"head_price,omitempty"`
	NecklinePrice      float64 `json:"neckline_price,omitempty"`
	LeftShoulderPrice  float64 `json:"left_shoulder_price,omitempty"`
	RightShoulderPrice float64 `json:"right_shoulder_price,omitempty"`
	ShoulderSymmetry   float64 `json:"shoulder_symmetry,omitempty"`
	HeadProminence     float64 `json:"head_prominence,omitempty"`
	TimeSymmetry       float64 `json:"time_symmetry,omitempty"`
	VolumeProfile      float64 `json:"volume_profile,omitempty"`
	NecklineQuality    float64 `json:"neckline_quality,omitempty"`
}

type ChecklistLog struct {
	WeeklyFavor   float64 `json:"weekly_favor"`
	WeeklyAOI     float64 `json:"weekly_aoi"`
	WeeklyEMA     float64 `json:"weekly_ema"`
	WeeklyReject  float64 `json:"weekly_reject"`
	WeeklyStruct  float64 `json:"weekly_struct"`
	WeeklyPattern float64 `json:"weekly_pattern"`
	DailyFavor    float64 `json:"daily_favor"`
	DailyAOI      float64 `json:"daily_aoi"`
	DailyEMA      float64 `json:"daily_ema"`
	DailyReject   float64 `json:"daily_reject"`
	DailyStruct   float64 `json:"daily_struct"`
	DailyPattern  float64 `json:"daily_pattern"`
	H4Favor       float64 `json:"h4_favor"`
	H4EMA         float64 `json:"h4_ema"`
	H4Reject      float64 `json:"h4_reject"`
	H4Struct      float64 `json:"h4_struct"`
	H4Pattern     float64 `json:"h4_pattern"`
	H2Psych       float64 `json:"h2_psych"`
	H1SOS         float64 `json:"h1_sos"`
	M30Engulf     float64 `json:"m30_engulf"`
	TotalPercent  float64 `json:"total_percent"`
}

type TradeLog struct {
	Timestamp     time.Time     `json:"timestamp"`
	Event         string        `json:"event"`
	TradeID       int           `json:"trade_id"`
	Symbol        string        `json:"symbol"`
	Direction     string        `json:"direction"`
	PatternType   string        `json:"pattern_type"`
	EntryPrice    float64       `json:"entry_price,omitempty"`
	ExitPrice     float64       `json:"exit_price,omitempty"`
	StopLoss      float64       `json:"stop_loss"`
	TakeProfit    float64       `json:"take_profit"`
	LotSize       float64       `json:"lot_size,omitempty"`
	RiskAmount    float64       `json:"risk_amount,omitempty"`
	RiskPercent   float64       `json:"risk_percent,omitempty"`
	RiskReward    float64       `json:"risk_reward,omitempty"`
	PnL           float64       `json:"pnl,omitempty"`
	PnLPips       float64       `json:"pnl_pips,omitempty"`
	PnLR          float64       `json:"pnl_r,omitempty"`
	PnLPct        float64       `json:"pnl_pct,omitempty"`
	ExitReason    string        `json:"exit_reason,omitempty"`
	Result        string        `json:"result,omitempty"`
	Checklist     *ChecklistLog `json:"checklist,omitempty"`
	Confidence    float64       `json:"confidence"`
	BalanceBefore float64       `json:"balance_before,omitempty"`
	BalanceAfter  float64       `json:"balance_after,omitempty"`
	EquityBefore  float64       `json:"equity_before,omitempty"`
	EquityAfter   float64       `json:"equity_after,omitempty"`
	BarsHeld      int           `json:"bars_held,omitempty"`
	BarIndex      int           `json:"bar_index"`
}

type EngineLog struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      LogLevel               `json:"level"`
	Event      string                 `json:"event"`
	Message    string                 `json:"message"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type AuditLoggerConfig struct {
	CandleLogPath string
	TradeLogPath  string
	EngineLogPath string
	Append        bool
}

type AuditLogger struct {
	candleWriter io.Writer
	tradeWriter  io.Writer
	engineWriter io.Writer
	mu           sync.Mutex
	sequence     int64
	closed       bool
	candleFile   *os.File
	tradeFile    *os.File
	engineFile   *os.File
}

func NewAuditLogger(config AuditLoggerConfig) (*AuditLogger, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if config.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	for _, path := range []string{config.CandleLogPath, config.TradeLogPath, config.EngineLogPath} {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
	}

	candleFile, err := os.OpenFile(config.CandleLogPath, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open candle log: %w", err)
	}

	tradeFile, err := os.OpenFile(config.TradeLogPath, flags, 0644)
	if err != nil {
		candleFile.Close()
		return nil, fmt.Errorf("failed to open trade log: %w", err)
	}

	engineFile, err := os.OpenFile(config.EngineLogPath, flags, 0644)
	if err != nil {
		candleFile.Close()
		tradeFile.Close()
		return nil, fmt.Errorf("failed to open engine log: %w", err)
	}

	return &AuditLogger{
		candleWriter: candleFile,
		tradeWriter:  tradeFile,
		engineWriter: engineFile,
		candleFile:   candleFile,
		tradeFile:    tradeFile,
		engineFile:   engineFile,
	}, nil
}

func NewAuditLoggerWithWriters(candleWriter, tradeWriter, engineWriter io.Writer) *AuditLogger {
	return &AuditLogger{
		candleWriter: candleWriter,
		tradeWriter:  tradeWriter,
		engineWriter: engineWriter,
	}
}

func (al *AuditLogger) LogCandle(log CandleLog) error {
	al.mu.Lock()
	defer al.mu.Unlock()
	if al.closed {
		return fmt.Errorf("logger is closed")
	}
	al.sequence++
	log.Sequence = al.sequence
	return al.writeJSON(al.candleWriter, log)
}

func (al *AuditLogger) LogTrade(log TradeLog) error {
	al.mu.Lock()
	defer al.mu.Unlock()
	if al.closed {
		return fmt.Errorf("logger is closed")
	}
	return al.writeJSON(al.tradeWriter, log)
}

func (al *AuditLogger) LogEngine(log EngineLog) error {
	al.mu.Lock()
	defer al.mu.Unlock()
	if al.closed {
		return fmt.Errorf("logger is closed")
	}
	return al.writeJSON(al.engineWriter, log)
}

func (al *AuditLogger) LogInfo(event, message string) error {
	return al.LogEngine(EngineLog{
		Timestamp: time.Now().UTC(),
		Level:     LevelInfo,
		Event:     event,
		Message:   message,
	})
}

func (al *AuditLogger) LogWarn(event, message string) error {
	return al.LogEngine(EngineLog{
		Timestamp: time.Now().UTC(),
		Level:     LevelWarn,
		Event:     event,
		Message:   message,
	})
}

func (al *AuditLogger) LogError(event string, err error) error {
	return al.LogEngine(EngineLog{
		Timestamp: time.Now().UTC(),
		Level:     LevelError,
		Event:     event,
		Error:     err.Error(),
	})
}

func (al *AuditLogger) LogLatency(event string, duration time.Duration) error {
	return al.LogEngine(EngineLog{
		Timestamp: time.Now().UTC(),
		Level:     LevelInfo,
		Event:     event,
		Message:   fmt.Sprintf("completed in %v", duration),
		Duration:  duration,
	})
}

func (al *AuditLogger) writeJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func (al *AuditLogger) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.closed = true

	var errs []error
	if al.candleFile != nil {
		if err := al.candleFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close candle log: %w", err))
		}
	}
	if al.tradeFile != nil {
		if err := al.tradeFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close trade log: %w", err))
		}
	}
	if al.engineFile != nil {
		if err := al.engineFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close engine log: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}
	return nil
}

func (al *AuditLogger) Flush() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	var errs []error
	if al.candleFile != nil {
		if err := al.candleFile.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	if al.tradeFile != nil {
		if err := al.tradeFile.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	if al.engineFile != nil {
		if err := al.engineFile.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors flushing log files: %v", errs)
	}
	return nil
}

func (al *AuditLogger) GetSequence() int64 {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.sequence
}

type NullLogger struct{}

func (nl *NullLogger) LogCandle(log CandleLog) error { return nil }
func (nl *NullLogger) LogTrade(log TradeLog) error   { return nil }
func (nl *NullLogger) LogEngine(log EngineLog) error { return nil }
func (nl *NullLogger) Close() error                  { return nil }

type Logger interface {
	LogCandle(log CandleLog) error
	LogTrade(log TradeLog) error
	LogEngine(log EngineLog) error
	Close() error
}

var _ Logger = (*AuditLogger)(nil)
var _ Logger = (*NullLogger)(nil)
