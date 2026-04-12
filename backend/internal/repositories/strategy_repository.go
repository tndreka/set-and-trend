package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StrategyRepository struct {
	pool *pgxpool.Pool
}

func NewStrategyRepository(pool *pgxpool.Pool) *StrategyRepository {
	return &StrategyRepository{pool: pool}
}

type Strategy struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Symbol      string    `json:"symbol"`
	Timeframe   string    `json:"timeframe"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

const strategyColumns = `id, code, name, description, symbol, timeframe, enabled, created_at`

func scanStrategy(row interface {
	Scan(dest ...any) error
}) (*Strategy, error) {
	var s Strategy
	if err := row.Scan(&s.ID, &s.Code, &s.Name, &s.Description, &s.Symbol, &s.Timeframe, &s.Enabled, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StrategyRepository) ListStrategies(ctx context.Context, enabledOnly bool) ([]*Strategy, error) {
	q := `SELECT ` + strategyColumns + ` FROM strategies`
	if enabledOnly {
		q += ` WHERE enabled = TRUE`
	}
	q += ` ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*Strategy, 0)
	for rows.Next() {
		s, err := scanStrategy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *StrategyRepository) GetStrategyByID(ctx context.Context, id uuid.UUID) (*Strategy, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+strategyColumns+` FROM strategies WHERE id = $1`, id)
	return scanStrategy(row)
}

func (r *StrategyRepository) GetStrategyByCode(ctx context.Context, code string) (*Strategy, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+strategyColumns+` FROM strategies WHERE code = $1`, code)
	return scanStrategy(row)
}
