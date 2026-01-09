-- Prevent duplicate trades on same account + candle + bias
-- This is the final authority, not application code
CREATE UNIQUE INDEX IF NOT EXISTS uniq_trade_account_candle_bias
ON trades (account_id, candle_id, bias);
