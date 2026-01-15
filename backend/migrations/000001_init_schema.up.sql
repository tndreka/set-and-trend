-- Phase 2 Context + Phase 3 Clean Trades
-- ============================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================
-- ENUMS
-- ============================
CREATE TYPE account_type AS ENUM ('demo', 'live');
CREATE TYPE session_type AS ENUM ('london', 'new_york', 'asian', 'custom');
CREATE TYPE trade_direction AS ENUM ('LONG', 'SHORT');
CREATE TYPE execution_type AS ENUM (
    'ENTRY_FILLED',
    'PARTIAL_EXIT',
    'TP_HIT',
    'SL_HIT',
    'MANUAL_CLOSE'
);
CREATE TYPE rule_result_type AS ENUM ('PASS', 'FAIL');
CREATE TYPE rule_timeframe AS ENUM ('W1');

-- ============================
-- USERS (Phase 2)
-- ============================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================
-- ACCOUNTS (Phase 2)
-- ============================
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type account_type NOT NULL,
    broker_name TEXT NOT NULL,
    currency TEXT NOT NULL,
    balance NUMERIC(12,2) NOT NULL,
    leverage INT NOT NULL,
    max_risk_per_trade_pct NUMERIC(5,2) NOT NULL,
    max_daily_risk_pct NUMERIC(5,2) NOT NULL,
    timezone TEXT NOT NULL,
    preferred_session session_type NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================
-- CANDLES (Phase 2)
-- ============================
CREATE TABLE candles_weekly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,
    open NUMERIC(12,5) NOT NULL,
    high NUMERIC(12,5) NOT NULL,
    low NUMERIC(12,5) NOT NULL,
    close NUMERIC(12,5) NOT NULL,
    volume BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_candles_weekly_timestamp ON candles_weekly(timestamp_utc);

-- ============================
-- INDICATORS (Phase 2)
-- ============================
CREATE TABLE indicators_weekly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE UNIQUE,
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

CREATE INDEX idx_indicators_weekly_candle ON indicators_weekly(candle_id);

-- ============================
-- RULES (Phase 2)
-- ============================
CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO rules (id, code, name, timeframe, description) VALUES
    (gen_random_uuid(), 'W1_TREND_BULLISH', 'Weekly Trend Bullish', 'W1', 'Weekly bullish trend confirmation')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE rule_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    candle_id UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE,
    result rule_result_type NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confidence_score NUMERIC(5,4) NOT NULL,
    UNIQUE(rule_id, candle_id)
);

CREATE INDEX idx_rule_results_candle ON rule_results(candle_id);
CREATE INDEX idx_rule_results_rule ON rule_results(rule_id);

-- ============================
-- TRADES (PHASE 3 - INTENT ONLY)
-- ============================
CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    candle_id UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    direction trade_direction NOT NULL,
    planned_entry NUMERIC(12,5) NOT NULL CHECK (planned_entry > 0),
    stop_loss NUMERIC(12,5) NOT NULL CHECK (stop_loss > 0),
    take_profit NUMERIC(12,5) NOT NULL CHECK (take_profit > 0),
    risk_percent NUMERIC(5,2) NOT NULL CHECK (risk_percent > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, candle_id, direction)
);

CREATE INDEX idx_trades_user ON trades(user_id);
CREATE INDEX idx_trades_account ON trades(account_id);
CREATE INDEX idx_trades_candle ON trades(candle_id);

-- ============================
-- TRADE EXECUTIONS (PHASE 3 - APPEND-ONLY FACTS)
-- ============================
CREATE TABLE trade_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    execution_type execution_type NOT NULL,
    price NUMERIC(12,5) NOT NULL CHECK (price > 0),
    quantity NUMERIC(12,4) NOT NULL CHECK (quantity > 0),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trade_executions_trade_id ON trade_executions(trade_id);
CREATE INDEX idx_trade_executions_executed_at ON trade_executions(executed_at);
CREATE UNIQUE INDEX idx_trade_single_entry ON trade_executions(trade_id)
WHERE execution_type = 'ENTRY_FILLED';
