-- Rollback H&S Pattern Detection Schema

DROP TABLE IF EXISTS walkforward_validation;
DROP TABLE IF EXISTS backtest_runs;
DROP TABLE IF EXISTS backtest_trades;
DROP TABLE IF EXISTS signals;
DROP TABLE IF EXISTS pattern_detections;

DROP TABLE IF EXISTS indicators_h1;
DROP TABLE IF EXISTS indicators_h4;
DROP TABLE IF EXISTS indicators_d1;

DROP TABLE IF EXISTS candles_h1;
DROP TABLE IF EXISTS candles_h4;
DROP TABLE IF EXISTS candles_d1;

-- Note: Cannot remove enum values in PostgreSQL
-- D1, H4, H1 will remain in rule_timeframe enum
