-- Migration 009: Separate Trade Intents from Executions (Phase 3 Fix)
-- Date: 2026-01-09
-- Description: Cancel/invalidate are USER INTENTS, not market executions

-- Step 1: Create trade_intents table (should already exist from migration 004)
-- If it doesn't exist, create it: 
CREATE TABLE IF NOT EXISTS trade_intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    intent_type VARCHAR(20) NOT NULL CHECK (intent_type IN ('cancel', 'invalidate')),
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- One intent per trade maximum
CREATE UNIQUE INDEX IF NOT EXISTS idx_trade_intents_unique ON trade_intents(trade_id);

-- Step 2: Remove cancel/invalidate from execution_event_type enum
-- First, check if any data exists with these values
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM trade_executions 
        WHERE event_type IN ('cancel', 'invalidate')
    ) THEN
        RAISE EXCEPTION 'Cannot migrate:  existing executions with cancel/invalidate found.  Migrate data first.';
    END IF;
END $$;

-- Drop and recreate the enum without cancel/invalidate
ALTER TYPE execution_event_type RENAME TO execution_event_type_old;

CREATE TYPE execution_event_type AS ENUM (
    'entry',
    'partial_close',
    'tp_hit',
    'sl_hit',
    'manual_close'
);

-- Update the column type
ALTER TABLE trade_executions 
    ALTER COLUMN event_type TYPE execution_event_type 
    USING event_type::text::execution_event_type;

-- Drop old enum
DROP TYPE execution_event_type_old;

-- Step 3: Add DB trigger to prevent execution after close
CREATE OR REPLACE FUNCTION prevent_execution_after_close()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if trade is already closed
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW. trade_id
          AND event_type IN ('tp_hit', 'sl_hit', 'manual_close')
    ) THEN
        RAISE EXCEPTION 'Cannot execute:  trade % is already closed', NEW.trade_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_execution_after_close
    BEFORE INSERT ON trade_executions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_execution_after_close();

-- Step 4: Add DB trigger to prevent intent after execution
CREATE OR REPLACE FUNCTION prevent_intent_after_execution()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if trade has any executions
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW.trade_id
          AND event_type = 'entry'
    ) THEN
        RAISE EXCEPTION 'Cannot cancel/invalidate: trade % has been executed', NEW.trade_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_intent_after_execution
    BEFORE INSERT ON trade_intents
    FOR EACH ROW
    EXECUTE FUNCTION prevent_intent_after_execution();

-- Step 5: Add validation for execution data
ALTER TABLE trade_executions 
    ADD CONSTRAINT valid_execution_data 
    CHECK (price > 0 AND position_size > 0);

-- Step 6: Remove lifecycle_status columns (already done in migration 008)
-- This is just documentation that lifecycle_status should not exist

COMMENT ON TABLE trade_executions IS 'Append-only log of market interactions.  State derived from this table + trade_intents. ';
COMMENT ON TABLE trade_intents IS 'User decisions to cancel or invalidate trades before execution.';
