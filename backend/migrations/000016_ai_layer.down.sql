-- Reverse of 000016_ai_layer.up.sql

BEGIN;

DROP INDEX IF EXISTS idx_setup_signals_unprocessed;

ALTER TABLE setup_signals
  DROP COLUMN IF EXISTS shadow_mode,
  DROP COLUMN IF EXISTS ai_composed_at,
  DROP COLUMN IF EXISTS ai_summary;

DROP TABLE IF EXISTS ai_verdicts;

COMMIT;
