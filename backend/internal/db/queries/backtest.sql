-- name: InsertBacktestRun :one
INSERT INTO backtest_runs (
    strategy_name, symbol, timeframe, execution_timeframe,
    start_date, end_date,
    total_patterns, traded_patterns, filtered_patterns,
    total_trades, winning_trades, losing_trades, breakeven_trades,
    win_rate, avg_win, avg_loss, avg_rr, expectancy,
    profit_factor, max_drawdown, sharpe_ratio, total_pnl,
    confidence_threshold, min_rr
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
)
RETURNING *;

-- name: GetBacktestRunByID :one
SELECT * FROM backtest_runs WHERE id = $1;

-- name: GetBacktestRunsByStrategy :many
SELECT * FROM backtest_runs 
WHERE strategy_name = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetLatestBacktestRun :one
SELECT * FROM backtest_runs 
WHERE strategy_name = $1 AND symbol = $2 AND timeframe = $3
ORDER BY created_at DESC
LIMIT 1;

-- name: InsertBacktestTrade :one
INSERT INTO backtest_trades (
    backtest_run_id, pattern_detection_id, signal_id,
    symbol, timeframe, execution_timeframe,
    entry_time, exit_time, entry_price, exit_price,
    pnl, pnl_r, rr_ratio, trade_result, reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetBacktestTradesByRun :many
SELECT * FROM backtest_trades 
WHERE backtest_run_id = $1
ORDER BY entry_time ASC;

-- name: GetBacktestTradesByResult :many
SELECT * FROM backtest_trades 
WHERE backtest_run_id = $1 AND trade_result = $2
ORDER BY entry_time ASC;

-- name: CountBacktestTradesByResult :one
SELECT COUNT(*) FROM backtest_trades 
WHERE backtest_run_id = $1 AND trade_result = $2;

-- name: InsertWalkforwardValidation :one
INSERT INTO walkforward_validation (
    strategy_name, symbol, timeframe, execution_timeframe,
    in_sample_start, in_sample_end, in_sample_expectancy, in_sample_trades,
    out_sample_start, out_sample_end, out_sample_expectancy, out_sample_trades,
    walkforward_efficiency, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetWalkforwardValidationsByStrategy :many
SELECT * FROM walkforward_validation 
WHERE strategy_name = $1
ORDER BY in_sample_start ASC;

-- name: GetWalkforwardValidationsByStatus :many
SELECT * FROM walkforward_validation 
WHERE strategy_name = $1 AND status = $2
ORDER BY created_at DESC;

-- name: CountWalkforwardValidationsByStatus :one
SELECT COUNT(*) FROM walkforward_validation 
WHERE strategy_name = $1 AND status = $2;
