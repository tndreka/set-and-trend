-- =====================================================================
-- 000013_restore_journal_schema
-- =====================================================================
-- Restores the trade-journal layer of the Set & Trend backend.
--
-- Context: the public schema in the live database currently contains
-- only the four candle tables (candles_weekly, candles_d1, candles_h4,
-- candles_h1). The sqlc-generated models in backend/internal/db/models.go
-- require many additional tables (users, accounts, trades, rules, etc.)
-- that were dropped at some point. Without them the API crashes at
-- runtime the first time any protected endpoint is hit.
--
-- This migration:
--   * creates every enum type referenced by sqlc models.go
--   * creates every non-candle table the Go code expects
--   * merges the relevant pieces of historic migrations 000001, 000006,
--     000007, 000008, and 000011 into a single idempotent restore
--   * never touches candles_weekly / candles_d1 / candles_h4 / candles_h1
--     — those rows (1.3M+) are the valuable asset and must be preserved
--   * removes the candles_weekly FK that 000001 originally added on
--     trades.candle_id and rule_results.candle_id so that multi-timeframe
--     trades (H4, D1, H1) can be logged against the correct candle table
--   * seeds the canonical W1_TREND_BULLISH rule
--
-- Idempotency: everything is wrapped in IF NOT EXISTS / DO blocks so it
-- can be applied against a partially-restored schema without failing.
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ----------------------------------------------------------------------
-- ENUMS
-- ----------------------------------------------------------------------
DO $$ BEGIN
    CREATE TYPE account_type AS ENUM ('demo', 'live');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE session_type AS ENUM ('london', 'new_york', 'asian', 'custom');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE trade_direction AS ENUM ('LONG', 'SHORT');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE execution_type AS ENUM (
        'ENTRY_FILLED',
        'PARTIAL_EXIT',
        'TP_HIT',
        'SL_HIT',
        'MANUAL_CLOSE'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE rule_result_type AS ENUM ('PASS', 'FAIL');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE rule_timeframe AS ENUM ('W1', 'D1', 'H4', 'H1');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Extend rule_timeframe if it already exists but only has W1
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'D1';
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'H4';
ALTER TYPE rule_timeframe ADD VALUE IF NOT EXISTS 'H1';

DO $$ BEGIN
    CREATE TYPE emotion_type AS ENUM ('calm', 'anxious', 'fomo', 'revenge', 'other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------------
-- USERS (merges 000001 + 000006 auth fields)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username                 VARCHAR(50)  UNIQUE NOT NULL,
    email                    VARCHAR(255) UNIQUE NOT NULL,
    password_hash            VARCHAR(255) NOT NULL,
    name                     VARCHAR(100),
    surname                  VARCHAR(100),
    is_email_verified        BOOLEAN      NOT NULL DEFAULT false,
    email_verification_token VARCHAR(255),
    password_reset_token     VARCHAR(255),
    password_reset_expires   TIMESTAMPTZ,
    last_login               TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username                 ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email                    ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token);
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token     ON users(password_reset_token);

-- ----------------------------------------------------------------------
-- ACCOUNTS (merges 000001 + 000011 check constraints)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type                    account_type NOT NULL,
    broker_name             TEXT NOT NULL,
    currency                TEXT NOT NULL,
    balance                 NUMERIC(12,2) NOT NULL,
    leverage                INT NOT NULL
        CHECK (leverage > 0 AND leverage <= 1000),
    max_risk_per_trade_pct  NUMERIC(5,2) NOT NULL
        CHECK (max_risk_per_trade_pct > 0 AND max_risk_per_trade_pct <= 100),
    max_daily_risk_pct      NUMERIC(5,2) NOT NULL
        CHECK (max_daily_risk_pct > 0 AND max_daily_risk_pct <= 100),
    timezone                TEXT NOT NULL,
    preferred_session       session_type NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----------------------------------------------------------------------
-- INDICATORS (4 tables, one per candle timeframe)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS indicators_weekly (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id             UUID NOT NULL REFERENCES candles_weekly(id) ON DELETE CASCADE UNIQUE,
    ema20                 NUMERIC(12,5) NOT NULL CHECK (ema20  > 0),
    ema50                 NUMERIC(12,5) NOT NULL CHECK (ema50  > 0),
    ema200                NUMERIC(12,5) NOT NULL CHECK (ema200 > 0),
    range_size            NUMERIC(12,5) NOT NULL,
    body_size             NUMERIC(12,5) NOT NULL,
    upper_wick            NUMERIC(12,5) NOT NULL,
    lower_wick            NUMERIC(12,5) NOT NULL,
    mid_price             NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price  NUMERIC(12,5) NOT NULL,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_indicators_weekly_candle ON indicators_weekly(candle_id);

CREATE TABLE IF NOT EXISTS indicators_d1 (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id             UUID NOT NULL REFERENCES candles_d1(id) ON DELETE CASCADE UNIQUE,
    ema20                 NUMERIC(12,5) NOT NULL CHECK (ema20  > 0),
    ema50                 NUMERIC(12,5) NOT NULL CHECK (ema50  > 0),
    ema200                NUMERIC(12,5) NOT NULL CHECK (ema200 > 0),
    range_size            NUMERIC(12,5) NOT NULL,
    body_size             NUMERIC(12,5) NOT NULL,
    upper_wick            NUMERIC(12,5) NOT NULL,
    lower_wick            NUMERIC(12,5) NOT NULL,
    mid_price             NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price  NUMERIC(12,5) NOT NULL,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_indicators_d1_candle ON indicators_d1(candle_id);

CREATE TABLE IF NOT EXISTS indicators_h4 (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id             UUID NOT NULL REFERENCES candles_h4(id) ON DELETE CASCADE UNIQUE,
    ema20                 NUMERIC(12,5) NOT NULL CHECK (ema20  > 0),
    ema50                 NUMERIC(12,5) NOT NULL CHECK (ema50  > 0),
    ema200                NUMERIC(12,5) NOT NULL CHECK (ema200 > 0),
    range_size            NUMERIC(12,5) NOT NULL,
    body_size             NUMERIC(12,5) NOT NULL,
    upper_wick            NUMERIC(12,5) NOT NULL,
    lower_wick            NUMERIC(12,5) NOT NULL,
    mid_price             NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price  NUMERIC(12,5) NOT NULL,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_indicators_h4_candle ON indicators_h4(candle_id);

CREATE TABLE IF NOT EXISTS indicators_h1 (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candle_id             UUID NOT NULL REFERENCES candles_h1(id) ON DELETE CASCADE UNIQUE,
    ema20                 NUMERIC(12,5) NOT NULL CHECK (ema20  > 0),
    ema50                 NUMERIC(12,5) NOT NULL CHECK (ema50  > 0),
    ema200                NUMERIC(12,5) NOT NULL CHECK (ema200 > 0),
    range_size            NUMERIC(12,5) NOT NULL,
    body_size             NUMERIC(12,5) NOT NULL,
    upper_wick            NUMERIC(12,5) NOT NULL,
    lower_wick            NUMERIC(12,5) NOT NULL,
    mid_price             NUMERIC(12,5) NOT NULL,
    last_swing_high_price NUMERIC(12,5) NOT NULL,
    last_swing_low_price  NUMERIC(12,5) NOT NULL,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_indicators_h1_candle ON indicators_h1(candle_id);

-- ----------------------------------------------------------------------
-- RULES + RULE RESULTS
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    timeframe   rule_timeframe NOT NULL,
    description TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO rules (id, code, name, timeframe, description) VALUES
    (gen_random_uuid(), 'W1_TREND_BULLISH', 'Weekly Trend Bullish', 'W1', 'Weekly bullish trend confirmation')
ON CONFLICT (code) DO NOTHING;

-- candle_id is NOT referenced to candles_weekly — rule results can
-- target any candle table (W1/D1/H4/H1). Integrity is enforced at the
-- application layer via the rules.timeframe column.
CREATE TABLE IF NOT EXISTS rule_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id          UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    candle_id        UUID NOT NULL,
    result           rule_result_type NOT NULL,
    evaluated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confidence_score NUMERIC(5,4) NOT NULL
        CHECK (confidence_score >= 0 AND confidence_score <= 1),
    UNIQUE (rule_id, candle_id)
);
CREATE INDEX IF NOT EXISTS idx_rule_results_candle ON rule_results(candle_id);
CREATE INDEX IF NOT EXISTS idx_rule_results_rule   ON rule_results(rule_id);

-- ----------------------------------------------------------------------
-- TRADES (merges 000001 + 000011 check constraints)
-- ----------------------------------------------------------------------
-- candle_id is NOT FK'd to candles_weekly so H4/D1/H1 trades are supported.
CREATE TABLE IF NOT EXISTS trades (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    account_id    UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    candle_id     UUID NOT NULL,
    symbol        TEXT NOT NULL,
    timeframe     TEXT NOT NULL,
    direction     trade_direction NOT NULL,
    planned_entry NUMERIC(12,5) NOT NULL CHECK (planned_entry > 0),
    stop_loss     NUMERIC(12,5) NOT NULL CHECK (stop_loss     > 0),
    take_profit   NUMERIC(12,5) NOT NULL CHECK (take_profit   > 0),
    risk_percent  NUMERIC(5,2)  NOT NULL
        CHECK (risk_percent > 0 AND risk_percent <= 100),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (stop_loss != take_profit),
    UNIQUE (account_id, candle_id, direction)
);
CREATE INDEX IF NOT EXISTS idx_trades_user    ON trades(user_id);
CREATE INDEX IF NOT EXISTS idx_trades_account ON trades(account_id);
CREATE INDEX IF NOT EXISTS idx_trades_candle  ON trades(candle_id);

-- ----------------------------------------------------------------------
-- TRADE EXECUTIONS (append-only fact table)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS trade_executions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id       UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    execution_type execution_type NOT NULL,
    price          NUMERIC(12,5) NOT NULL CHECK (price > 0),
    quantity       NUMERIC(12,4) NOT NULL CHECK (quantity > 0),
    executed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
        CHECK (executed_at <= NOW() + INTERVAL '1 minute')
);
CREATE INDEX IF NOT EXISTS idx_trade_executions_trade_id   ON trade_executions(trade_id);
CREATE INDEX IF NOT EXISTS idx_trade_executions_executed_at ON trade_executions(executed_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_trade_single_entry
    ON trade_executions(trade_id)
    WHERE execution_type = 'ENTRY_FILLED';

-- ----------------------------------------------------------------------
-- TRADE INTENTS (used by repositories/intent_repository.go)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS trade_intents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id    UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    intent_type TEXT NOT NULL,
    reason      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trade_intents_trade_id ON trade_intents(trade_id);

-- ----------------------------------------------------------------------
-- TRADE FEEDBACK (from 000007)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS trade_feedback (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id         UUID NOT NULL UNIQUE REFERENCES trades(id) ON DELETE CASCADE,
    followed_plan    BOOLEAN NOT NULL,
    emotion_before   emotion_type NOT NULL,
    emotion_during   emotion_type NOT NULL,
    emotion_after    emotion_type NOT NULL,
    biggest_mistake  TEXT,
    screenshot_url   TEXT,
    feedback_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trade_feedback_trade_id ON trade_feedback(trade_id);

-- ----------------------------------------------------------------------
-- PATTERN DETECTIONS + SIGNALS (from 000008)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pattern_detections (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol               TEXT NOT NULL,
    timeframe            rule_timeframe NOT NULL,
    detected_candle_idx  INT NOT NULL,
    left_shoulder_price  NUMERIC(12,5) NOT NULL,
    left_shoulder_idx    INT NOT NULL,
    head_price           NUMERIC(12,5) NOT NULL,
    head_idx             INT NOT NULL,
    right_shoulder_price NUMERIC(12,5) NOT NULL,
    right_shoulder_idx   INT NOT NULL,
    neckline_price       NUMERIC(12,5) NOT NULL,
    neckline_idx         INT NOT NULL,
    shoulder_symmetry    NUMERIC(5,4)  NOT NULL,
    head_prominence      NUMERIC(5,4)  NOT NULL,
    time_symmetry        NUMERIC(5,4)  NOT NULL,
    volume_profile       NUMERIC(5,4)  NOT NULL,
    neckline_quality     NUMERIC(5,4)  NOT NULL,
    context_trend        VARCHAR(20)   NOT NULL,
    volatility_regime    VARCHAR(20)   NOT NULL,
    context_dist_ema200  NUMERIC(8,5)  NOT NULL,
    recent_swings        INT NOT NULL,
    overall_confidence   NUMERIC(6,4)  NOT NULL,
    final_confidence     NUMERIC(6,4)  NOT NULL,
    pattern_type         VARCHAR(10)   NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pattern_detections_symbol_tf   ON pattern_detections(symbol, timeframe);
CREATE INDEX IF NOT EXISTS idx_pattern_detections_confidence  ON pattern_detections(final_confidence);
CREATE INDEX IF NOT EXISTS idx_pattern_detections_created     ON pattern_detections(created_at);

CREATE TABLE IF NOT EXISTS signals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_detection_id  UUID NOT NULL REFERENCES pattern_detections(id) ON DELETE CASCADE,
    symbol                TEXT NOT NULL,
    timeframe             rule_timeframe NOT NULL,
    execution_timeframe   rule_timeframe NOT NULL,
    direction             trade_direction NOT NULL,
    detected_price        NUMERIC(12,5) NOT NULL,
    theoretical_sl        NUMERIC(12,5) NOT NULL,
    theoretical_tp        NUMERIC(12,5) NOT NULL,
    projected_rr          NUMERIC(5,2)  NOT NULL,
    confidence            NUMERIC(6,4)  NOT NULL,
    status                TEXT NOT NULL DEFAULT 'BACKTEST_ONLY',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_signals_pattern   ON signals(pattern_detection_id);
CREATE INDEX IF NOT EXISTS idx_signals_status    ON signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_symbol_tf ON signals(symbol, timeframe);
