-- ============================
-- PHASE 3 CLEAN CANONICAL SCHEMA
-- ============================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================
-- ENUMS
-- ============================

CREATE TYPE trade_direction AS ENUM ('LONG', 'SHORT');

CREATE TYPE execution_type AS ENUM (
    'ENTRY_FILLED',
    'PARTIAL_EXIT',
    'TP_HIT',
    'SL_HIT',
    'MANUAL_CLOSE'
);

-- ============================
-- TRADES (INTENT ONLY)
-- ============================

CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    direction trade_direction NOT NULL,

    planned_entry NUMERIC(12,5) NOT NULL CHECK (planned_entry > 0),
    stop_loss NUMERIC(12,5) NOT NULL CHECK (stop_loss > 0),
    take_profit NUMERIC(12,5) NOT NULL CHECK (take_profit > 0),

    risk_percent NUMERIC(5,2) NOT NULL CHECK (risk_percent > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================
-- TRADE EXECUTIONS (FACTS)
-- ============================

CREATE TABLE trade_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,

    execution_type execution_type NOT NULL,
    price NUMERIC(12,5) NOT NULL CHECK (price > 0),
    quantity NUMERIC(12,4) NOT NULL CHECK (quantity > 0),

    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trade_executions_trade_id
ON trade_executions(trade_id);

CREATE INDEX idx_trade_executions_executed_at
ON trade_executions(executed_at);

CREATE UNIQUE INDEX idx_trade_single_entry
ON trade_executions(trade_id)
WHERE execution_type = 'ENTRY_FILLED';

-- ============================
-- CANDLES (FROM PHASE 2)
-- ============================

CREATE TABLE candles (
    id BIGSERIAL PRIMARY KEY,

    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    open_time TIMESTAMPTZ NOT NULL,

    open NUMERIC(12,5) NOT NULL,
    high NUMERIC(12,5) NOT NULL,
    low NUMERIC(12,5) NOT NULL,
    close NUMERIC(12,5) NOT NULL,
    volume NUMERIC(20,2) NOT NULL,

    UNIQUE(symbol, timeframe, open_time)
);
