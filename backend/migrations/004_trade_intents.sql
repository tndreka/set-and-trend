-- Migration 004: Trade Intents (User Actions, NOT Executions)
-- Date: 2026-01-09
-- Description: Captures user intent to cancel or invalidate trades

DROP TABLE IF EXISTS trade_intents CASCADE;

-- Trade intents table (cancel / invalidate)
-- These are NOT executions - they're user/system decisions
CREATE TABLE trade_intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    
    -- Intent type (CANCEL or INVALIDATE)
    intent_type VARCHAR(20) NOT NULL CHECK (intent_type IN ('cancel', 'invalidate')),
    
    -- Why this intent was set
    reason TEXT NOT NULL,
    
    -- When intent was recorded
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Business rule: One intent per trade
CREATE UNIQUE INDEX idx_trade_intents_unique ON trade_intents(trade_id);

-- DB-LEVEL INVARIANT: Cannot set intent if already executed
CREATE OR REPLACE FUNCTION prevent_intent_after_execution()
RETURNS trigger AS $$
BEGIN
    -- Check if trade has any executions
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW.trade_id
    ) THEN
        RAISE EXCEPTION 'Cannot set intent: trade % has already been executed', NEW.trade_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_intent_after_execution
BEFORE INSERT ON trade_intents
FOR EACH ROW EXECUTE FUNCTION prevent_intent_after_execution();

COMMENT ON TABLE trade_intents IS 'Records user/system intent to cancel or invalidate trades. Separate from executions because these are NOT market interactions. ';
