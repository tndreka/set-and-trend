-- Migration 005: Trade Lifecycle Status

-- Add lifecycle status for terminal non-execution states
ALTER TABLE trades ADD COLUMN lifecycle_status trade_lifecycle_status;
ALTER TABLE trades ADD COLUMN lifecycle_changed_at TIMESTAMPTZ;
ALTER TABLE trades ADD COLUMN lifecycle_reason TEXT;

-- Index for lifecycle queries
CREATE INDEX idx_trades_lifecycle_status ON trades(lifecycle_status) 
WHERE lifecycle_status IS NOT NULL;

-- Constraint: Cannot have executions if lifecycle is terminal
-- (Enforced at service layer, not DB)
