-- =====================================================================
-- 000014_add_strategies_and_signals
-- =====================================================================
-- Adds the multi-strategy lab tables on top of the restored journal
-- schema (000013). Enables running 2–3 trading strategies in parallel,
-- each mapped to its own broker demo account, with push-alert signals
-- when any strategy's rule fires on a new H4 bar.
--
--   1. strategies        — high-level strategy definition (name, code,
--                          timeframe, symbol, enabled flag)
--   2. strategy_rules    — M2M between strategies and the existing
--                          rules table; one strategy can combine rules
--                          (Phase 1 uses one rule per strategy)
--   3. accounts.strategy_id  — nullable FK so each demo account declares
--                              which strategy it's running
--   4. setup_signals     — output table the candle-sync job writes to
--                          when a rule evaluates PASS on a fresh bar;
--                          OpenClaw's setup-watcher skill polls this
--                          table every 5 minutes and pushes unnotified
--                          rows to Telegram
-- =====================================================================

-- ----------------------------------------------------------------------
-- STRATEGIES
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS strategies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT NOT NULL UNIQUE,        -- 'H4_TREND_FOLLOWING' etc
    name        TEXT NOT NULL,
    description TEXT NOT NULL,
    symbol      TEXT NOT NULL DEFAULT 'EURUSD',
    timeframe   rule_timeframe NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategies_enabled ON strategies(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_strategies_symbol_tf ON strategies(symbol, timeframe);

-- Seed the three starter strategies. Idempotent via ON CONFLICT.
INSERT INTO strategies (code, name, description, symbol, timeframe) VALUES
    ('H4_TREND_FOLLOWING',
     'H4 Trend Following',
     'EMA50>EMA200, close above EMA50, rising EMA50 slope. Entry on prior H4 high break for longs. SL below swing low. TP 2R.',
     'EURUSD', 'H4'),
    ('H4_SUPPLY_DEMAND',
     'H4 Supply & Demand',
     'Reaction at unmitigated demand zone with bullish rejection. Weekly trend confluence filter. SL below zone, TP at nearest supply.',
     'EURUSD', 'H4'),
    ('H4_LONDON_BREAKOUT',
     'London Session Breakout',
     'Asian session range (22:00-07:00 UTC) broken during London hours (07:00-10:00 UTC). SL on opposite boundary, TP at 1.5x range.',
     'EURUSD', 'H4')
ON CONFLICT (code) DO NOTHING;

-- ----------------------------------------------------------------------
-- STRATEGY_RULES (M2M strategies <-> rules)
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS strategy_rules (
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    rule_id     UUID NOT NULL REFERENCES rules(id)      ON DELETE CASCADE,
    PRIMARY KEY (strategy_id, rule_id)
);

-- ----------------------------------------------------------------------
-- ACCOUNTS.strategy_id  (ALTER existing table from 000013)
-- ----------------------------------------------------------------------
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS strategy_id UUID REFERENCES strategies(id);

CREATE INDEX IF NOT EXISTS idx_accounts_strategy ON accounts(strategy_id);

-- ----------------------------------------------------------------------
-- SETUP_SIGNALS
-- ----------------------------------------------------------------------
-- Written by backend/internal/services/signal_evaluator.go whenever a
-- strategy's rule evaluates PASS on a newly synced H4 bar.
-- Consumed by OpenClaw's setup-watcher skill (polls /api/signals/active
-- every 5 minutes, pushes message to Telegram, calls /signals/:id/notified).
CREATE TABLE IF NOT EXISTS setup_signals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id  UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    symbol       TEXT NOT NULL,
    timeframe    rule_timeframe NOT NULL,
    candle_id    UUID NOT NULL,                 -- points into candles_{tf}; no FK (multi-tf)
    direction    trade_direction NOT NULL,
    entry        NUMERIC(12,5) NOT NULL CHECK (entry > 0),
    stop_loss    NUMERIC(12,5) NOT NULL CHECK (stop_loss > 0),
    take_profit  NUMERIC(12,5) NOT NULL CHECK (take_profit > 0),
    rr           NUMERIC(6,2)  NOT NULL CHECK (rr > 0),
    confidence   NUMERIC(3,2)  NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    details      JSONB,                         -- strategy-specific extras (rule reasons, indicators)
    notified     BOOLEAN NOT NULL DEFAULT FALSE,
    notified_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A strategy can only emit one signal per candle; prevents duplicates
    -- from repeated candle-sync runs on the same bar.
    UNIQUE (strategy_id, candle_id)
);

-- Hot-path index for the setup-watcher poller.
CREATE INDEX IF NOT EXISTS idx_setup_signals_unnotified
    ON setup_signals(created_at)
    WHERE notified = FALSE;

CREATE INDEX IF NOT EXISTS idx_setup_signals_strategy
    ON setup_signals(strategy_id, created_at DESC);
