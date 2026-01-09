-- Migration 008: Drop lifecycle_status columns
-- Date: 2026-01-09
ALTER TABLE trades 
DROP COLUMN IF EXISTS lifecycle_status CASCADE,
DROP COLUMN IF EXISTS lifecycle_changed_at CASCADE,
DROP COLUMN IF EXISTS lifecycle_reason CASCADE;

DROP TYPE IF EXISTS trade_lifecycle_status CASCADE;

COMMENT ON TABLE trades IS 'Trade state derived from trade_executions and trade_intents. ';
