-- Reverses 000014_add_strategies_and_signals.

DROP TABLE IF EXISTS setup_signals  CASCADE;

ALTER TABLE accounts DROP COLUMN IF EXISTS strategy_id;

DROP TABLE IF EXISTS strategy_rules CASCADE;
DROP TABLE IF EXISTS strategies     CASCADE;
