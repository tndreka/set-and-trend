-- Migration 003: Trade Execution Events (Phase 3)
-- Date: 2026-01-09
-- Description: Append-only event log for trade lifecycle

-- Drop existing if any (clean slate)
DROP TABLE IF EXISTS trade_executions CASCADE;
DROP TYPE IF EXISTS execution_event_type CASCADE;
DROP TYPE IF EXISTS trade_lifecycle_status CASCADE;

-- Execution event types (lowercase to match Go code)
CREATE TYPE execution_event_type AS ENUM (
    'entry',
    'partial_close',
    'tp_hit',
    'sl_hit',
    'manual_close',
    'cancel',
    'invalidate'
);

-- Lifecycle status for terminal non-execution states
CREATE TYPE trade_lifecycle_status AS ENUM (
    'cancelled',
    'invalidated'
);

-- Trade executions table (append-only event log)
CREATE TABLE trade_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    
    -- Event details
    event_type execution_event_type NOT NULL,
    price DECIMAL(12,5),
    position_size DECIMAL(12,8),
    
    -- PnL (computed server-side, nullable until close)
    pnl DECIMAL(12,2),
    pnl_pips DECIMAL(12,2),
    
    -- Timing
    executed_at TIMESTAMPTZ NOT NULL,
    
    -- Context
    session session_type,
    reason TEXT,
    slippage_pips DECIMAL(8,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_trade_executions_trade_id ON trade_executions(trade_id);
CREATE INDEX idx_trade_executions_executed_at ON trade_executions(executed_at);
CREATE INDEX idx_executions_trade_time ON trade_executions(trade_id, executed_at);

-- Prevent duplicate entries (can only enter once)
CREATE UNIQUE INDEX idx_trade_executions_unique_entry 
ON trade_executions(trade_id, event_type) 
WHERE event_type = 'entry';
