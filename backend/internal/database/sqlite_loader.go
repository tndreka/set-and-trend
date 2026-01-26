package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// SQLiteLoader loads candle data from a SQLite database
type SQLiteLoader struct {
	db     *sql.DB
	dbPath string
}

// NewSQLiteLoader creates a new loader for the given database path
func NewSQLiteLoader(dbPath string) (*SQLiteLoader, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteLoader{
		db:     db,
		dbPath: dbPath,
	}, nil
}

func (l *SQLiteLoader) Close() error {
	return l.db.Close()
}

func (l *SQLiteLoader) GetSymbols(tf Timeframe) ([]string, error) {
	query := fmt.Sprintf("SELECT DISTINCT symbol FROM %s ORDER BY symbol", tf.TableName())
	rows, err := l.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbols: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func (l *SQLiteLoader) GetCandles(symbol string, tf Timeframe, dateRange *DateRange) ([]Candle, error) {
	query := fmt.Sprintf("SELECT symbol, datetime, open, high, low, close, volume FROM %s WHERE symbol = ?", tf.TableName())
	args := []interface{}{symbol}

	if dateRange != nil {
		startMs := dateRange.Start.UnixMilli()
		endMs := dateRange.End.UnixMilli()
		query += " AND datetime >= ? AND datetime <= ?"
		args = append(args, startMs, endMs)
	}

	query += " ORDER BY datetime ASC"
	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query candles: %w", err)
	}
	defer rows.Close()

	var candles []Candle
	for rows.Next() {
		var c Candle
		var datetimeMs int64
		var open, high, low, close, volume sql.NullFloat64

		if err := rows.Scan(&c.Symbol, &datetimeMs, &open, &high, &low, &close, &volume); err != nil {
			return nil, fmt.Errorf("failed to scan candle: %w", err)
		}

		c.Timestamp = time.UnixMilli(datetimeMs)
		c.Open = open.Float64
		c.High = high.Float64
		c.Low = low.Float64
		c.Close = close.Float64
		c.Volume = volume.Float64
		candles = append(candles, c)
	}
	return candles, nil
}

func (l *SQLiteLoader) GetCandleCount(symbol string, tf Timeframe) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE symbol = ?", tf.TableName())
	var count int
	err := l.db.QueryRow(query, symbol).Scan(&count)
	return count, err
}

func (l *SQLiteLoader) GetDateRange(symbol string, tf Timeframe) (*DateRange, error) {
	query := fmt.Sprintf("SELECT MIN(datetime), MAX(datetime) FROM %s WHERE symbol = ?", tf.TableName())
	var minMs, maxMs sql.NullInt64
	err := l.db.QueryRow(query, symbol).Scan(&minMs, &maxMs)
	if err != nil {
		return nil, err
	}
	if !minMs.Valid || !maxMs.Valid {
		return nil, fmt.Errorf("no data found for symbol %s", symbol)
	}
	return &DateRange{
		Start: time.UnixMilli(minMs.Int64),
		End:   time.UnixMilli(maxMs.Int64),
	}, nil
}

func (l *SQLiteLoader) GetStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{TimeframeStats: make(map[Timeframe]*TimeframeStats)}
	timeframes := []Timeframe{TimeframeDaily, TimeframeH4, TimeframeH1, TimeframeM30}

	for _, tf := range timeframes {
		tfStats := &TimeframeStats{SymbolCounts: make(map[string]int)}
		var totalCount int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tf.TableName())
		if err := l.db.QueryRow(countQuery).Scan(&totalCount); err != nil {
			continue
		}
		tfStats.TotalBars = totalCount

		symbols, err := l.GetSymbols(tf)
		if err != nil {
			continue
		}
		tfStats.SymbolCount = len(symbols)

		if len(symbols) > 0 {
			dateRange, err := l.GetDateRange(symbols[0], tf)
			if err == nil {
				tfStats.StartDate = dateRange.Start
				tfStats.EndDate = dateRange.End
			}
		}
		stats.TimeframeStats[tf] = tfStats
	}
	return stats, nil
}

type DatabaseStats struct {
	TimeframeStats map[Timeframe]*TimeframeStats
}

type TimeframeStats struct {
	TotalBars    int
	SymbolCount  int
	SymbolCounts map[string]int
	StartDate    time.Time
	EndDate      time.Time
}
