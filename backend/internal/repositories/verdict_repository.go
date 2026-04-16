package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AIVerdict is one agent's opinion on a setup signal. Rows are immutable;
// re-running an agent against the same signal is a no-op thanks to the
// (signal_id, agent_id) UNIQUE constraint.
type AIVerdict struct {
	ID         uuid.UUID        `json:"id"`
	SignalID   uuid.UUID        `json:"signal_id"`
	AgentID    string           `json:"agent_id"`
	Verdict    string           `json:"verdict"`
	Confidence *decimal.Decimal `json:"confidence,omitempty"`
	Reason     *string          `json:"reason,omitempty"`
	RawOutput  json.RawMessage  `json:"raw_output,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

type CreateVerdictParams struct {
	SignalID   uuid.UUID
	AgentID    string
	Verdict    string
	Confidence *float64
	Reason     *string
	RawOutput  json.RawMessage
}

type VerdictRepository struct {
	pool *pgxpool.Pool
}

func NewVerdictRepository(pool *pgxpool.Pool) *VerdictRepository {
	return &VerdictRepository{pool: pool}
}

const verdictColumns = `id, signal_id, agent_id, verdict, confidence, reason, raw_output, created_at`

func scanVerdict(row interface {
	Scan(dest ...any) error
}) (*AIVerdict, error) {
	var v AIVerdict
	var confidence *decimal.Decimal
	var reason *string
	var raw []byte
	if err := row.Scan(&v.ID, &v.SignalID, &v.AgentID, &v.Verdict, &confidence, &reason, &raw, &v.CreatedAt); err != nil {
		return nil, err
	}
	v.Confidence = confidence
	v.Reason = reason
	if raw != nil {
		v.RawOutput = json.RawMessage(raw)
	}
	return &v, nil
}

// CreateVerdict upserts — if the agent already voted on this signal, the
// existing row is returned unchanged (idempotent replay of composer runs).
func (r *VerdictRepository) CreateVerdict(ctx context.Context, p CreateVerdictParams) (*AIVerdict, error) {
	var confidence *decimal.Decimal
	if p.Confidence != nil {
		d := decimal.NewFromFloat(*p.Confidence)
		confidence = &d
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO ai_verdicts (signal_id, agent_id, verdict, confidence, reason, raw_output)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (signal_id, agent_id) DO UPDATE SET agent_id = EXCLUDED.agent_id
		RETURNING `+verdictColumns,
		p.SignalID, p.AgentID, p.Verdict, confidence, p.Reason, p.RawOutput,
	)
	return scanVerdict(row)
}

func (r *VerdictRepository) ListBySignal(ctx context.Context, signalID uuid.UUID) ([]*AIVerdict, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+verdictColumns+` FROM ai_verdicts
		WHERE signal_id = $1
		ORDER BY created_at ASC`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AIVerdict, 0)
	for rows.Next() {
		v, err := scanVerdict(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *VerdictRepository) CountForSignal(ctx context.Context, signalID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai_verdicts WHERE signal_id = $1`, signalID).Scan(&n)
	return n, err
}
