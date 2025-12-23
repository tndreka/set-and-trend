-- Set The Trend MVP — PostgreSQL Schema

-- Enums (domain layer)
CREATE TYPE trade_bias AS ENUM ('long', 'short');
CREATE TYPE trade_result AS ENUM ('win', 'loss', 'breakeven');
CREATE TYPE account_type AS ENUM ('demo', 'live');
CREATE TYPE session_type AS ENUM ('london', 'new_york', 'asian', 'custom');
CREATE TYPE emotion_type AS ENUM ('calm', 'anxious', 'fomo', 'revenge', 'other');
CREATE TYPE rule_result_type AS ENUM ('PASS', 'FAIL');
CREATE TYPE rule_timeframe AS ENUM ('W1');

-- Core tables

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type account_type NOT NULL,
    broker_name TEXT NOT NULL,
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    balance DECIMAL(15,2) NOT NULL CHECK (balance >= 0),
    leverage INTEGER NOT NULL CHECK (leverage > 0),
    max_risk_per_trade_pct DECIMAL(5,2) NOT NULL CHECK (max_risk_per_trade_pct BETWEEN 0 AND 100),
    max_daily_risk_pct DECIMAL(5,2) NOT NULL CHECK (max_daily_risk_pct BETWEEN 0 AND 100),
    timezone TEXT NOT NULL CHECK (timezone ~ '^[^/]+(/[A-Za-z_/-]+)*$'), -- IANA TZ
    preferred_session session_type NOT NULL DEFAULT 'london',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE candles_weekly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp_utc TIMESTAMP WITH TIME ZONE NOT NULL UNIQUE,
    open DECIMAL(12,5) NOT NULL,
    high DECIMAL(12,5) NOT NULL,
    low DECIMAL(12,5) NOT NULL CHECK (low <= high),
    close DECIMAL(12,5) NOT NULL,
    volume BIGINT, -- optional
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE indicators_weekly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id UUID NOT NULL UNIQUE REFERENCES candles_weekly(id) ON DELETE CASCADE,
    ema20 DECIMAL(12,5) NOT NULL,
    ema50 DECIMAL(12,5) NOT NULL,
    ema200 DECIMAL(12,5) NOT NULL,
    range_size DECIMAL(12,5) NOT NULL,
    body_size DECIMAL(12,5) NOT NULL,
    upper_wick DECIMAL(12,5) NOT NULL,
    lower_wick DECIMAL(12,5) NOT NULL,
    mid_price DECIMAL(12,5) NOT NULL,
    last_swing_high_price DECIMAL(12,5),
    last_swing_low_price DECIMAL(12,5),
    computed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE, -- "W1_TREND_BULLISH"
    name TEXT NOT NULL,
    timeframe rule_timeframe NOT NULL DEFAULT 'W1',
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE rule_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES rules(id),
    candle_id UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE,
    result rule_result_type NOT NULL,
    evaluated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confidence_score DECIMAL(3,2), -- 0.0 to 1.0 optional
    UNIQUE(rule_id, candle_id) -- One result per rule per candle
);

CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    candle_id UUID NOT NULL REFERENCES candles_weekly(id),
    symbol TEXT NOT NULL DEFAULT 'EURUSD' CHECK (symbol = 'EURUSD'),
    timeframe TEXT NOT NULL DEFAULT 'W1' CHECK (timeframe = 'W1'),
    setup_timestamp_utc TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- ACCOUNT SNAPSHOTS (immutable)
    account_balance_at_setup DECIMAL(15,2) NOT NULL,
    leverage_at_setup INTEGER NOT NULL,
    max_risk_per_trade_pct_at_setup DECIMAL(5,2) NOT NULL,
    timezone_at_setup TEXT NOT NULL,
    
    -- PLANNED
    bias trade_bias NOT NULL,
    planned_entry DECIMAL(12,5) NOT NULL,
    planned_sl DECIMAL(12,5) NOT NULL,
    planned_tp DECIMAL(12,5) NOT NULL,
    planned_rr DECIMAL(5,2) NOT NULL CHECK (planned_rr > 0),
    planned_risk_pct DECIMAL(5,2) NOT NULL,
    planned_risk_amount DECIMAL(15,2) NOT NULL,
    planned_position_size DECIMAL(10,5) NOT NULL,
    reason_for_trade TEXT NOT NULL,
    
    -- ACTUAL
    actual_entry DECIMAL(12,5),
    actual_sl DECIMAL(12,5),
    actual_tp DECIMAL(12,5),
    actual_risk_pct DECIMAL(5,2),
    actual_risk_amount DECIMAL(15,2),
    actual_position_size DECIMAL(10,5),
    execution_timestamp_utc TIMESTAMP WITH TIME ZONE,
    close_timestamp_utc TIMESTAMP WITH TIME ZONE,
    close_price DECIMAL(12,5),
    result trade_result,
    pips_gained DECIMAL(8,2),
    money_gained DECIMAL(15,2),
    rr_realized DECIMAL(5,2),
    duration_seconds INTEGER,
    session session_type,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE trade_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL UNIQUE REFERENCES trades(id) ON DELETE CASCADE,
    followed_plan BOOLEAN NOT NULL,
    emotion_before emotion_type NOT NULL,
    emotion_during emotion_type NOT NULL,
    emotion_after emotion_type NOT NULL,
    biggest_mistake TEXT,
    screenshot_url TEXT,
    feedback_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for analytics (critical performance)
CREATE INDEX idx_candles_timestamp ON candles_weekly(timestamp_utc);
CREATE INDEX idx_trades_user_id ON trades(user_id);
CREATE INDEX idx_trades_candle_id ON trades(candle_id);
CREATE INDEX idx_trades_result ON trades(result);
CREATE INDEX idx_trades_bias ON trades(bias);
CREATE INDEX idx_trades_session ON trades(session);
CREATE INDEX idx_rule_results_candle_id ON rule_results(candle_id);
CREATE INDEX idx_rule_results_rule_id ON rule_results(rule_id);
CREATE INDEX idx_trade_feedback_trade_id ON trade_feedback(trade_id);

-- Sample data (for testing)
INSERT INTO users (id) VALUES (gen_random_uuid()) ON CONFLICT DO NOTHING;
INSERT INTO rules (code, name, timeframe, description) VALUES
    ('W1_TREND_BULLISH', 'Weekly Trend Bullish', 'W1', 'EMA50 > EMA200 AND Close > EMA50'),
    ('W1_TREND_BEARISH', 'Weekly Trend Bearish', 'W1', 'EMA50 < EMA200 AND Close < EMA50'),
    ('W1_TOUCH_EMA20', 'Weekly EMA20 Touch', 'W1', 'Price touches EMA20 with proximity filter')
ON CONFLICT DO NOTHING;
