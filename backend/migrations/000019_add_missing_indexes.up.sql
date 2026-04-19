-- 000019: Add composite indexes for common query patterns.
CREATE INDEX IF NOT EXISTS idx_trades_account_candle ON trades(account_id, candle_id);
CREATE INDEX IF NOT EXISTS idx_setup_signals_created ON setup_signals(created_at DESC);
