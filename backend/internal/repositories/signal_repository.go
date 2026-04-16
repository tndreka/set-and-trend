package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type SignalRepository struct {
	pool *pgxpool.Pool
}

func NewSignalRepository(pool *pgxpool.Pool) *SignalRepository {
	return &SignalRepository{pool: pool}
}

type SetupSignal struct {
	ID          uuid.UUID       `json:"id"`
	StrategyID  uuid.UUID       `json:"strategy_id"`
	Symbol      string          `json:"symbol"`
	Timeframe   string          `json:"timeframe"`
	CandleID    uuid.UUID       `json:"candle_id"`
	Direction   string          `json:"direction"`
	Entry       decimal.Decimal `json:"entry"`
	StopLoss    decimal.Decimal `json:"stop_loss"`
	TakeProfit  decimal.Decimal `json:"take_profit"`
	RR          decimal.Decimal `json:"rr"`
	Confidence  decimal.Decimal `json:"confidence"`
	Details     json.RawMessage `json:"details,omitempty"`
	Notified    bool            `json:"notified"`
	NotifiedAt  *time.Time      `json:"notified_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CreateSignalParams struct {
	StrategyID uuid.UUID
	Symbol     string
	Timeframe  string
	CandleID   uuid.UUID
	Direction  string
	Entry      float64
	StopLoss   float64
	TakeProfit float64
	RR         float64
	Confidence float64
	Details    json.RawMessage
}

const signalColumns = `id, strategy_id, symbol, timeframe, candle_id, direction, entry, stop_loss, take_profit, rr, confidence, details, notified, notified_at, created_at`

func scanSignal(row interface {
	Scan(dest ...any) error
}) (*SetupSignal, error) {
	var s SetupSignal
	var details []byte
	var notifiedAt *time.Time
	if err := row.Scan(
		&s.ID, &s.StrategyID, &s.Symbol, &s.Timeframe, &s.CandleID,
		&s.Direction, &s.Entry, &s.StopLoss, &s.TakeProfit, &s.RR, &s.Confidence,
		&details, &s.Notified, &notifiedAt, &s.CreatedAt,
	); err != nil {
		return nil, err
	}
	if details != nil {
		s.Details = json.RawMessage(details)
	}
	s.NotifiedAt = notifiedAt
	return &s, nil
}

// CreateSignal inserts a new setup_signals row. The (strategy_id, candle_id)
// unique constraint deduplicates repeated evaluator runs against the same bar.
func (r *SignalRepository) CreateSignal(ctx context.Context, p CreateSignalParams) (*SetupSignal, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO setup_signals
		    (strategy_id, symbol, timeframe, candle_id, direction, entry, stop_loss, take_profit, rr, confidence, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+signalColumns,
		p.StrategyID, p.Symbol, p.Timeframe, p.CandleID, p.Direction,
		decimal.NewFromFloat(p.Entry), decimal.NewFromFloat(p.StopLoss),
		decimal.NewFromFloat(p.TakeProfit), decimal.NewFromFloat(p.RR),
		decimal.NewFromFloat(p.Confidence), p.Details,
	)
	return scanSignal(row)
}

// ListUnnotified returns up to `limit` unnotified signals oldest-first
// (the OpenClaw setup-watcher polls this every 5 minutes).
func (r *SignalRepository) ListUnnotified(ctx context.Context, limit int32) ([]*SetupSignal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+signalColumns+` FROM setup_signals
		WHERE notified = FALSE
		ORDER BY created_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*SetupSignal, 0)
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListByStrategy returns recent signals for a single strategy (notified or not).
func (r *SignalRepository) ListByStrategy(ctx context.Context, strategyID uuid.UUID, limit int32) ([]*SetupSignal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+signalColumns+` FROM setup_signals
		WHERE strategy_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, strategyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*SetupSignal, 0)
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// MarkNotified flips the notified flag and stamps notified_at.
func (r *SignalRepository) MarkNotified(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE setup_signals
		SET notified = TRUE, notified_at = NOW()
		WHERE id = $1 AND notified = FALSE`, id)
	return err
}

// ListUnprocessed returns signals that have not yet been composed by the AI
// orchestrator. Caller passes an age threshold in seconds so the poller can
// wait for specialists to write their verdicts before sweeping a signal.
func (r *SignalRepository) ListUnprocessed(ctx context.Context, olderThanSec int, limit int32) ([]*SetupSignal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+signalColumns+` FROM setup_signals
		WHERE ai_composed_at IS NULL
		  AND created_at < NOW() - make_interval(secs => $1)
		ORDER BY created_at ASC
		LIMIT $2`, olderThanSec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*SetupSignal, 0)
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// MarkComposed stores the composer's summary and stamps ai_composed_at.
func (r *SignalRepository) MarkComposed(ctx context.Context, id uuid.UUID, summary string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE setup_signals
		SET ai_summary = $2, ai_composed_at = NOW()
		WHERE id = $1 AND ai_composed_at IS NULL`, id, summary)
	return err
}
