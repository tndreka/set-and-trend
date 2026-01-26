-- H&S Pattern Detection Schema
-- Phase 1: Multi-timeframe candles, indicators, patterns, signals, backtesting

-- ============================
-- EXTEND TIMEFRAME ENUM
-- ============================
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'D1';
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'H4';
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'H1';

-- ============================
-- CANDLE TABLES (Multi-TF)
-- ============================
CREATE TABLE IF NOT EXISTS candles_d1 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,
    open NUMERIC(12,5) NOT NULL,
    high NUMERIC(12,5) NOT NULL,
    low NUMERIC(12,5) NOT NULL,
    close NUMERIC(12,5) NOT NULL,
    volume BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_candles_d1_timestamp ON candles_d1(timestamp_utc);

CREATE TABLE IF NOT EXISTS candles_h4 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,
    open NUMERIC(12,5) NOT NULL,
    high NUMERIC(12,5) NOT NULL,
    low NUMERIC(12,5) NOT NULL,
    close NUMERIC(12,5) NOT NULL,
    volume BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_candles_h4_timestamp ON candles_h4(timestamp_utc);

CREATE TABLE IF NOT EXISTS candles_h1 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,
    open NUMERIC(12,5) NOT NULL,
    high NUMERIC(12,5) NOT NULL,
    low NUMERIC(12,5) NOT NULL,
    close NUMERIC(12,5) NOT NULL,
    volume BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_candles_h1_timestamp ON candles_h1(timestamp_utc);

-- ============================
-- INDICATOR TABLES (Multi-TF)
-- ============================
CREATE TABLE IF NOT EXISTS indicators_d1 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id UUID NOT NULL REFERENCES candles_d1(id) ON DELETE CASCADE UNIQUE,
    ema20 NUMERIC(12,5) NOT NULL,
    ema50 NUMERIC(12,5) NOT NULL,
    ema200 NUMERIC(12,5) NOT NULL,
    range_size NUMERIC(12,5) NOT NULL,
    body_size NUMERIC(12,5) NOT NULL,
    upper_wick NUMERIC(12,5) NOT NULL,
    lower_wick NUMERIC(12,5) NOT NULL,
    mid_price NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price NUMERIC(12,5) NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_indicators_d1_candle ON indicators_d1(candle_id);

CREATE TABLE IF NOT EXISTS indicators_h4 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id UUID NOT NULL REFERENCES candles_h4(id) ON DELETE CASCADE UNIQUE,
    ema20 NUMERIC(12,5) NOT NULL,
    ema50 NUMERIC(12,5) NOT NULL,
    ema200 NUMERIC(12,5) NOT NULL,
    range_size NUMERIC(12,5) NOT NULL,
    body_size NUMERIC(12,5) NOT NULL,
    upper_wick NUMERIC(12,5) NOT NULL,
    lower_wick NUMERIC(12,5) NOT NULL,
    mid_price NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price NUMERIC(12,5) NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_indicators_h4_candle ON indicators_h4(candle_id);

CREATE TABLE IF NOT EXISTS indicators_h1 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id UUID NOT NULL REFERENCES candles_h1(id) ON DELETE CASCADE UNIQUE,
    ema20 NUMERIC(12,5) NOT NULL,
    ema50 NUMERIC(12,5) NOT NULL,
    ema200 NUMERIC(12,5) NOT NULL,
    range_size NUMERIC(12,5) NOT NULL,
    body_size NUMERIC(12,5) NOT NULL,
    upper_wick NUMERIC(12,5) NOT NULL,
    lower_wick NUMERIC(12,5) NOT NULL,
    mid_price NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price NUMERIC(12,5) NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_indicators_h1_candle ON indicators_h1(candle_id);

-- ============================
-- PATTERN DETECTIONS
-- ============================
CREATE TABLE IF NOT EXISTS pattern_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    detected_candle_idx INT NOT NULL,
    
    -- Structure points
    left_shoulder_price NUMERIC(12,5) NOT NULL,
    left_shoulder_idx INT NOT NULL,
    head_price NUMERIC(12,5) NOT NULL,
    head_idx INT NOT NULL,
    right_shoulder_price NUMERIC(12,5) NOT NULL,
    right_shoulder_idx INT NOT NULL,
    neckline_price NUMERIC(12,5) NOT NULL,
    neckline_idx INT NOT NULL,
    
    -- Component scores (0.0-1.0)
    shoulder_symmetry NUMERIC(5,4) NOT NULL,
    head_prominence NUMERIC(5,4) NOT NULL,
    time_symmetry NUMERIC(5,4) NOT NULL,
    volume_profile NUMERIC(5,4) NOT NULL,
    neckline_quality NUMERIC(5,4) NOT NULL,
    
    -- Context
    context_trend VARCHAR(20) NOT NULL,
    volatility_regime VARCHAR(20) NOT NULL,
    context_dist_ema200 NUMERIC(8,5) NOT NULL,
    recent_swings INT NOT NULL,
    
    -- Confidence
    overall_confidence NUMERIC(6,4) NOT NULL,
    final_confidence NUMERIC(6,4) NOT NULL,
    pattern_type VARCHAR(10) NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pattern_detections_symbol_tf ON pattern_detections(symbol, timeframe);
CREATE INDEX idx_pattern_detections_confidence ON pattern_detections(final_confidence);
CREATE INDEX idx_pattern_detections_created ON pattern_detections(created_at);

-- ============================
-- SIGNALS
-- ============================
CREATE TABLE IF NOT EXISTS signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_detection_id UUID NOT NULL REFERENCES pattern_detections(id) ON DELETE CASCADE,
    
    symbol TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    execution_timeframe rule_timeframe NOT NULL,
    
    direction trade_direction NOT NULL,
    detected_price NUMERIC(12,5) NOT NULL,
    theoretical_sl NUMERIC(12,5) NOT NULL,
    theoretical_tp NUMERIC(12,5) NOT NULL,
    projected_rr NUMERIC(5,2) NOT NULL,
    confidence NUMERIC(6,4) NOT NULL,
    
    status TEXT NOT NULL DEFAULT 'BACKTEST_ONLY',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_signals_pattern ON signals(pattern_detection_id);
CREATE INDEX idx_signals_status ON signals(status);
CREATE INDEX idx_signals_symbol_tf ON signals(symbol, timeframe);

-- ============================
-- BACKTEST TRADES
-- ============================
CREATE TABLE IF NOT EXISTS backtest_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID,
    pattern_detection_id UUID REFERENCES pattern_detections(id),
    signal_id UUID REFERENCES signals(id),
    
    symbol TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    execution_timeframe rule_timeframe NOT NULL,
    
    entry_time TIMESTAMPTZ NOT NULL,
    exit_time TIMESTAMPTZ NOT NULL,
    entry_price NUMERIC(12,5) NOT NULL,
    exit_price NUMERIC(12,5) NOT NULL,
    
    pnl NUMERIC(14,5) NOT NULL,
    pnl_r NUMERIC(8,4) NOT NULL,
    rr_ratio NUMERIC(5,2) NOT NULL,
    trade_result TEXT NOT NULL,
    reason TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backtest_trades_run ON backtest_trades(backtest_run_id);
CREATE INDEX idx_backtest_trades_pattern ON backtest_trades(pattern_detection_id);
CREATE INDEX idx_backtest_trades_result ON backtest_trades(trade_result);

-- ============================
-- BACKTEST RUNS
-- ============================
CREATE TABLE IF NOT EXISTS backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_name VARCHAR(100) NOT NULL,
    symbol TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    execution_timeframe rule_timeframe NOT NULL,
    
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    
    total_patterns INT NOT NULL DEFAULT 0,
    traded_patterns INT NOT NULL DEFAULT 0,
    filtered_patterns INT NOT NULL DEFAULT 0,
    
    total_trades INT NOT NULL DEFAULT 0,
    winning_trades INT NOT NULL DEFAULT 0,
    losing_trades INT NOT NULL DEFAULT 0,
    breakeven_trades INT NOT NULL DEFAULT 0,
    
    win_rate NUMERIC(6,4),
    avg_win NUMERIC(12,5),
    avg_loss NUMERIC(12,5),
    avg_rr NUMERIC(5,2),
    expectancy NUMERIC(8,4),
    profit_factor NUMERIC(8,4),
    max_drawdown NUMERIC(8,4),
    sharpe_ratio NUMERIC(8,4),
    total_pnl NUMERIC(14,5),
    
    confidence_threshold NUMERIC(6,4) NOT NULL,
    min_rr NUMERIC(5,2) NOT NULL DEFAULT 1.5,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backtest_runs_strategy ON backtest_runs(strategy_name);
CREATE INDEX idx_backtest_runs_symbol_tf ON backtest_runs(symbol, timeframe);

-- ============================
-- WALK-FORWARD VALIDATION
-- ============================
CREATE TABLE IF NOT EXISTS walkforward_validation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_name VARCHAR(100) NOT NULL,
    symbol TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    execution_timeframe rule_timeframe NOT NULL,
    
    in_sample_start DATE NOT NULL,
    in_sample_end DATE NOT NULL,
    in_sample_expectancy NUMERIC(8,4) NOT NULL,
    in_sample_trades INT NOT NULL,
    
    out_sample_start DATE NOT NULL,
    out_sample_end DATE NOT NULL,
    out_sample_expectancy NUMERIC(8,4) NOT NULL,
    out_sample_trades INT NOT NULL,
    
    walkforward_efficiency NUMERIC(8,4) NOT NULL,
    status VARCHAR(20) NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wfv_strategy ON walkforward_validation(strategy_name);
CREATE INDEX idx_wfv_status ON walkforward_validation(status);

