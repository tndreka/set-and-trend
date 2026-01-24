package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

// TradeSummary contains overall trading statistics
type TradeSummary struct {
	TotalTrades   int     `json:"total_trades"`
	WinCount      int     `json:"win_count"`
	LossCount     int     `json:"loss_count"`
	BreakevenCount int    `json:"breakeven_count"`
	WinRate       float64 `json:"win_rate"`
	AvgRR         float64 `json:"avg_rr"`
	TotalPnL      float64 `json:"total_pnl"`
	BestTrade     float64 `json:"best_trade"`
	WorstTrade    float64 `json:"worst_trade"`
	WinStreak     int     `json:"win_streak"`
	LossStreak    int     `json:"loss_streak"`
}

// RuleStats contains statistics per rule
type RuleStats struct {
	RuleCode    string  `json:"rule_code"`
	RuleName    string  `json:"rule_name"`
	TotalTrades int     `json:"total_trades"`
	WinCount    int     `json:"win_count"`
	LossCount   int     `json:"loss_count"`
	WinRate     float64 `json:"win_rate"`
	AvgRR       float64 `json:"avg_rr"`
	TotalPnL    float64 `json:"total_pnl"`
}

// SessionStats contains statistics per trading session
type SessionStats struct {
	Session     string  `json:"session"`
	TotalTrades int     `json:"total_trades"`
	WinCount    int     `json:"win_count"`
	LossCount   int     `json:"loss_count"`
	WinRate     float64 `json:"win_rate"`
	AvgRR       float64 `json:"avg_rr"`
	TotalPnL    float64 `json:"total_pnl"`
}

// EmotionStats contains statistics by emotion
type EmotionStats struct {
	Emotion       string  `json:"emotion"`
	Phase         string  `json:"phase"` // before, during, after
	TotalTrades   int     `json:"total_trades"`
	WinCount      int     `json:"win_count"`
	LossCount     int     `json:"loss_count"`
	WinRate       float64 `json:"win_rate"`
	FollowedPlan  int     `json:"followed_plan_count"`
	PlanRate      float64 `json:"plan_adherence_rate"`
}

// GetTradeSummary returns overall trading statistics
func (r *AnalyticsRepository) GetTradeSummary(ctx context.Context, userID *string) (*TradeSummary, error) {
	query := `
		WITH trade_stats AS (
			SELECT 
				COUNT(*) as total_trades,
				COUNT(*) FILTER (WHERE te.execution_type IN ('TP_HIT')) as win_count,
				COUNT(*) FILTER (WHERE te.execution_type IN ('SL_HIT')) as loss_count,
				COUNT(*) FILTER (WHERE te.execution_type IN ('MANUAL_CLOSE')) as manual_count
			FROM trades t
			LEFT JOIN trade_executions te ON t.id = te.trade_id 
				AND te.execution_type IN ('TP_HIT', 'SL_HIT', 'MANUAL_CLOSE')
			WHERE ($1::text IS NULL OR t.user_id::text = $1)
		),
		pnl_stats AS (
			SELECT 
				COALESCE(SUM(
					CASE 
						WHEN te.execution_type = 'TP_HIT' THEN (te.price - t.planned_entry) * te.quantity
						WHEN te.execution_type = 'SL_HIT' THEN (te.price - t.planned_entry) * te.quantity
						WHEN te.execution_type = 'MANUAL_CLOSE' THEN (te.price - t.planned_entry) * te.quantity
						ELSE 0 
					END
				), 0) as total_pnl,
				COALESCE(MAX(
					CASE WHEN t.direction = 'LONG' THEN (te.price - t.planned_entry) * te.quantity
					     ELSE (t.planned_entry - te.price) * te.quantity END
				), 0) as best_trade,
				COALESCE(MIN(
					CASE WHEN t.direction = 'LONG' THEN (te.price - t.planned_entry) * te.quantity
					     ELSE (t.planned_entry - te.price) * te.quantity END
				), 0) as worst_trade
			FROM trades t
			LEFT JOIN trade_executions te ON t.id = te.trade_id 
				AND te.execution_type IN ('TP_HIT', 'SL_HIT', 'MANUAL_CLOSE')
			WHERE ($1::text IS NULL OR t.user_id::text = $1)
		)
		SELECT 
			ts.total_trades,
			ts.win_count,
			ts.loss_count,
			ts.manual_count as breakeven_count,
			CASE WHEN ts.total_trades > 0 
				THEN ROUND((ts.win_count::numeric / ts.total_trades) * 100, 2) 
				ELSE 0 END as win_rate,
			0 as avg_rr,
			ps.total_pnl,
			ps.best_trade,
			ps.worst_trade,
			0 as win_streak,
			0 as loss_streak
		FROM trade_stats ts, pnl_stats ps
	`

	var summary TradeSummary
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&summary.TotalTrades,
		&summary.WinCount,
		&summary.LossCount,
		&summary.BreakevenCount,
		&summary.WinRate,
		&summary.AvgRR,
		&summary.TotalPnL,
		&summary.BestTrade,
		&summary.WorstTrade,
		&summary.WinStreak,
		&summary.LossStreak,
	)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetStatsByRule returns statistics grouped by rule
func (r *AnalyticsRepository) GetStatsByRule(ctx context.Context, userID *string) ([]RuleStats, error) {
	query := `
		SELECT 
			r.code as rule_code,
			r.name as rule_name,
			COUNT(DISTINCT t.id) as total_trades,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT') as win_count,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'SL_HIT') as loss_count,
			CASE WHEN COUNT(DISTINCT t.id) > 0 
				THEN ROUND((COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT')::numeric / COUNT(DISTINCT t.id)) * 100, 2) 
				ELSE 0 END as win_rate,
			0 as avg_rr,
			0 as total_pnl
		FROM rules r
		LEFT JOIN rule_results rr ON r.id = rr.rule_id AND rr.result = 'PASS'
		LEFT JOIN trades t ON rr.candle_id = t.candle_id
			AND ($1::text IS NULL OR t.user_id::text = $1)
		LEFT JOIN trade_executions te ON t.id = te.trade_id 
			AND te.execution_type IN ('TP_HIT', 'SL_HIT', 'MANUAL_CLOSE')
		GROUP BY r.id, r.code, r.name
		ORDER BY total_trades DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []RuleStats
	for rows.Next() {
		var s RuleStats
		err := rows.Scan(
			&s.RuleCode,
			&s.RuleName,
			&s.TotalTrades,
			&s.WinCount,
			&s.LossCount,
			&s.WinRate,
			&s.AvgRR,
			&s.TotalPnL,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetStatsBySession returns statistics grouped by trading session
func (r *AnalyticsRepository) GetStatsBySession(ctx context.Context, userID *string) ([]SessionStats, error) {
	query := `
		SELECT 
			COALESCE(a.preferred_session::text, 'unknown') as session,
			COUNT(DISTINCT t.id) as total_trades,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT') as win_count,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'SL_HIT') as loss_count,
			CASE WHEN COUNT(DISTINCT t.id) > 0 
				THEN ROUND((COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT')::numeric / COUNT(DISTINCT t.id)) * 100, 2) 
				ELSE 0 END as win_rate,
			0 as avg_rr,
			0 as total_pnl
		FROM trades t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN trade_executions te ON t.id = te.trade_id 
			AND te.execution_type IN ('TP_HIT', 'SL_HIT', 'MANUAL_CLOSE')
		WHERE ($1::text IS NULL OR t.user_id::text = $1)
		GROUP BY a.preferred_session
		ORDER BY total_trades DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []SessionStats
	for rows.Next() {
		var s SessionStats
		err := rows.Scan(
			&s.Session,
			&s.TotalTrades,
			&s.WinCount,
			&s.LossCount,
			&s.WinRate,
			&s.AvgRR,
			&s.TotalPnL,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetStatsByEmotion returns statistics grouped by emotion (before trade)
func (r *AnalyticsRepository) GetStatsByEmotion(ctx context.Context, userID *string) ([]EmotionStats, error) {
	query := `
		SELECT 
			tf.emotion_before::text as emotion,
			'before' as phase,
			COUNT(DISTINCT t.id) as total_trades,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT') as win_count,
			COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'SL_HIT') as loss_count,
			CASE WHEN COUNT(DISTINCT t.id) > 0 
				THEN ROUND((COUNT(DISTINCT t.id) FILTER (WHERE te.execution_type = 'TP_HIT')::numeric / COUNT(DISTINCT t.id)) * 100, 2) 
				ELSE 0 END as win_rate,
			COUNT(*) FILTER (WHERE tf.followed_plan = true) as followed_plan_count,
			CASE WHEN COUNT(*) > 0 
				THEN ROUND((COUNT(*) FILTER (WHERE tf.followed_plan = true)::numeric / COUNT(*)) * 100, 2) 
				ELSE 0 END as plan_adherence_rate
		FROM trades t
		JOIN trade_feedback tf ON t.id = tf.trade_id
		LEFT JOIN trade_executions te ON t.id = te.trade_id 
			AND te.execution_type IN ('TP_HIT', 'SL_HIT', 'MANUAL_CLOSE')
		WHERE ($1::text IS NULL OR t.user_id::text = $1)
		GROUP BY tf.emotion_before
		ORDER BY total_trades DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []EmotionStats
	for rows.Next() {
		var s EmotionStats
		err := rows.Scan(
			&s.Emotion,
			&s.Phase,
			&s.TotalTrades,
			&s.WinCount,
			&s.LossCount,
			&s.WinRate,
			&s.FollowedPlan,
			&s.PlanRate,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}
