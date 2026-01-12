-- ====================================
-- COMPLETE SCHEMA (Phase 2 + Phase 3)
-- ====================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ====================================
-- ENUMS
-- ====================================

CREATE TYPE account_type AS ENUM ('demo', 'live');
CREATE TYPE session_type AS ENUM ('london', 'new_york', 'asian', 'custom');
CREATE TYPE trade_bias AS ENUM ('long', 'short');
CREATE TYPE trade_result AS ENUM ('win', 'loss', 'breakeven');
CREATE TYPE emotion_type AS ENUM ('calm', 'anxious', 'fomo', 'revenge', 'other');
CREATE TYPE execution_event_type AS ENUM ('entry', 'partial_close', 'tp_hit', 'sl_hit', 'manual_close');
CREATE TYPE rule_result_type AS ENUM ('PASS', 'FAIL');
CREATE TYPE rule_timeframe AS ENUM ('W1');

-- ====================================
-- USERS
-- ====================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ====================================
-- ACCOUNTS
-- ====================================

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

-- ====================================
-- CANDLES
-- ====================================

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

-- ====================================
-- INDICATORS
-- ====================================

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

-- ====================================
-- RULES
-- ====================================

CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed initial rules
INSERT INTO rules (id, code, name, timeframe, description) VALUES
    (gen_random_uuid(), 'W1_TREND_BULLISH', 'Weekly Trend Bullish', 'W1', 'Weekly bullish trend confirmation: EMA50 > EMA200, Close > EMA50, EMA50 rising')
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

-- ====================================
-- TRADES (PHASE 3)
-- ====================================

CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    candle_id UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    setup_timestamp_utc TIMESTAMPTZ NOT NULL,
    
    -- Account snapshot at setup
    account_balance_at_setup NUMERIC(12,2) NOT NULL,
    leverage_at_setup INT NOT NULL,
    max_risk_per_trade_pct_at_setup NUMERIC(5,2) NOT NULL,
    timezone_at_setup TEXT NOT NULL,
    
    -- Trade plan
    bias trade_bias NOT NULL,
    planned_entry NUMERIC(12,5) NOT NULL,
    planned_sl NUMERIC(12,5) NOT NULL,
    planned_tp NUMERIC(12,5) NOT NULL,
    planned_rr NUMERIC(6,2) NOT NULL,
    planned_risk_pct NUMERIC(5,2) NOT NULL,
    planned_risk_amount NUMERIC(12,2) NOT NULL,
    planned_position_size NUMERIC(12,2) NOT NULL,
    reason_for_trade TEXT NOT NULL,
    
    -- Actual execution (filled on execution)
    actual_entry NUMERIC(12,5),
    actual_sl NUMERIC(12,5),
    actual_tp NUMERIC(12,5),
    actual_risk_pct NUMERIC(5,2),
    actual_risk_amount NUMERIC(12,2),
    actual_position_size NUMERIC(12,2),
    execution_timestamp_utc TIMESTAMPTZ,
    
    -- Closure (filled on close)
    close_timestamp_utc TIMESTAMPTZ,
    close_price NUMERIC(12,5),
    result trade_result,
    pips_gained NUMERIC(10,2),
    money_gained NUMERIC(12,2),
    rr_realized NUMERIC(6,2),
    duration_seconds INT,
    session session_type,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Prevent duplicate trades on same candle/account/bias
    UNIQUE(account_id, candle_id, bias)
);

CREATE INDEX idx_trades_user ON trades(user_id);
CREATE INDEX idx_trades_account ON trades(account_id);
CREATE INDEX idx_trades_candle ON trades(candle_id);

-- ====================================
-- TRADE EXECUTIONS (Phase 3)
-- ====================================

CREATE TABLE trade_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    event_type execution_event_type NOT NULL,
    price NUMERIC(12,5),
    position_size NUMERIC(12,2),
    executed_at TIMESTAMPTZ NOT NULL,
    session session_type,
    reason TEXT,
    slippage_pips NUMERIC(10,2),
    pnl NUMERIC(12,2),
    pnl_pips NUMERIC(10,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trade_executions_trade ON trade_executions(trade_id);
CREATE INDEX idx_trade_executions_executed_at ON trade_executions(executed_at);

-- ====================================
-- TRADE INTENTS (Phase 3)
-- ====================================

CREATE TABLE trade_intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE UNIQUE,
    intent_type TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trade_intents_trade ON trade_intents(trade_id);

-- ====================================
-- TRADE FEEDBACK
-- ====================================

CREATE TABLE trade_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE UNIQUE,
    followed_plan BOOLEAN NOT NULL,
    emotion_before emotion_type NOT NULL,
    emotion_during emotion_type NOT NULL,
    emotion_after emotion_type NOT NULL,
    biggest_mistake TEXT,
    screenshot_url TEXT,
    feedback_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
