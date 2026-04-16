-- Migration 000016: AI layer for multi-agent signal enrichment
-- Adds ai_verdicts (per-agent verdicts per signal) and composer output columns on setup_signals.

BEGIN;

-- One row per (signal, agent) pair. Agents POST their verdict here.
CREATE TABLE IF NOT EXISTS ai_verdicts (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  signal_id   UUID NOT NULL REFERENCES setup_signals(id) ON DELETE CASCADE,
  agent_id    TEXT NOT NULL,
  verdict     TEXT NOT NULL,
  confidence  NUMERIC(3,2),
  reason      TEXT,
  raw_output  JSONB,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ai_verdicts_signal_agent_uniq UNIQUE (signal_id, agent_id),
  CONSTRAINT ai_verdicts_confidence_range  CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1))
);

CREATE INDEX IF NOT EXISTS idx_ai_verdicts_signal ON ai_verdicts(signal_id);
CREATE INDEX IF NOT EXISTS idx_ai_verdicts_agent  ON ai_verdicts(agent_id, created_at DESC);

-- Composer output persisted on the signal itself.
ALTER TABLE setup_signals
  ADD COLUMN IF NOT EXISTS ai_summary      TEXT,
  ADD COLUMN IF NOT EXISTS ai_composed_at  TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS shadow_mode     BOOLEAN NOT NULL DEFAULT TRUE;

-- Composer's polling query filters on this.
CREATE INDEX IF NOT EXISTS idx_setup_signals_unprocessed
  ON setup_signals(created_at)
  WHERE ai_composed_at IS NULL;

COMMIT;
