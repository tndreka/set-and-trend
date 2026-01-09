-- Migration 003: Trade Execution Events (Phase 3 - PRODUCTION)
-- Date: 2026-01-09
-- Description: Append-only execution log - market interactions ONLY

DROP TABLE IF EXISTS trade_executions CASCADE;
DROP TYPE IF EXISTS execution_event_type CASCADE;

-- Execution event types (MARKET INTERACTIONS ONLY)
-- REMOVED: cancel, invalidate (those are intents, not executions)
CREATE TYPE execution_event_type AS ENUM (
    'entry',
    'partial_close',
    'tp_hit',
    'sl_hit',
    'manual_close'
);

-- Trade executions table (append-only, market events only)
CREATE TABLE trade_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    
    -- Event type (market interaction)
    event_type execution_event_type NOT NULL,
    
    -- Raw execution data (AUTHORITATIVE)
    price DECIMAL(12,5) NOT NULL,
    position_size DECIMAL(12,8) NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL,
    
    -- Context
    session session_type,
    reason TEXT,
    slippage_pips DECIMAL(8,2),
    
    -- PnL (DERIVED - recompute in analytics using ACTUAL entry price)
    pnl DECIMAL(12,2),
    pnl_pips DECIMAL(12,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraint: Price and size are mandatory for all executions
    CONSTRAINT valid_execution_data CHECK (price > 0 AND position_size > 0)
);

-- Performance indexes
CREATE INDEX idx_trade_executions_trade_id ON trade_executions(trade_id);
CREATE INDEX idx_trade_executions_executed_at ON trade_executions(executed_at);
CREATE INDEX idx_trade_executions_event_type ON trade_executions(event_type);
CREATE INDEX idx_executions_trade_time ON trade_executions(trade_id, executed_at);

-- Business rule: One entry per trade
CREATE UNIQUE INDEX idx_trade_executions_unique_entry 
ON trade_executions(trade_id, event_type) 
WHERE event_type = 'entry';

-- DB-LEVEL INVARIANT:  Prevent execution after close
CREATE OR REPLACE FUNCTION prevent_execution_after_close()
RETURNS trigger AS $$
BEGIN
    -- Check if trade is already closed
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW.trade_id
        AND event_type IN ('tp_hit', 'sl_hit', 'manual_close')
    ) THEN
        RAISE EXCEPTION 'Cannot execute:  trade % is already closed', NEW.trade_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_execution_after_close
BEFORE INSERT ON trade_executions
FOR EACH ROW EXECUTE FUNCTION prevent_execution_after_close();

-- DB-LEVEL INVARIANT:  Prevent entry if already entered
CREATE OR REPLACE FUNCTION prevent_duplicate_entry()
RETURNS trigger AS $$
BEGIN
    IF NEW.event_type = 'entry' THEN
        IF EXISTS (
            SELECT 1 FROM trade_executions
            WHERE trade_id = NEW.trade_id
            AND event_type = 'entry'
        ) THEN
            RAISE EXCEPTION 'Cannot enter: trade % already has entry', NEW.trade_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_duplicate_entry
BEFORE INSERT ON trade_executions
FOR EACH ROW EXECUTE FUNCTION prevent_duplicate_entry();

COMMENT ON TABLE trade_executions IS 'Append-only execution event log.  Contains MARKET INTERACTIONS only.  State is computed via:  SELECT event_type FROM trade_executions WHERE trade_id = ?  ORDER BY executed_at';
