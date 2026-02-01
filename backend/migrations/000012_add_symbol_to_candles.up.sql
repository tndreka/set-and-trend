-- Add symbol column to multi-timeframe candle tables
-- This enables storing multiple forex pairs in the same tables

-- candles_weekly
ALTER TABLE candles_weekly ADD COLUMN IF NOT EXISTS symbol TEXT NOT NULL DEFAULT 'EURUSD';
ALTER TABLE candles_weekly DROP CONSTRAINT IF EXISTS candles_weekly_timestamp_utc_key;
ALTER TABLE candles_weekly ADD CONSTRAINT candles_weekly_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
CREATE INDEX IF NOT EXISTS idx_candles_weekly_symbol ON candles_weekly(symbol);

-- candles_d1
ALTER TABLE candles_d1 ADD COLUMN IF NOT EXISTS symbol TEXT NOT NULL DEFAULT 'EURUSD';
ALTER TABLE candles_d1 DROP CONSTRAINT IF EXISTS candles_d1_timestamp_utc_key;
ALTER TABLE candles_d1 ADD CONSTRAINT candles_d1_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
CREATE INDEX IF NOT EXISTS idx_candles_d1_symbol ON candles_d1(symbol);

-- candles_h4
ALTER TABLE candles_h4 ADD COLUMN IF NOT EXISTS symbol TEXT NOT NULL DEFAULT 'EURUSD';
ALTER TABLE candles_h4 DROP CONSTRAINT IF EXISTS candles_h4_timestamp_utc_key;
ALTER TABLE candles_h4 ADD CONSTRAINT candles_h4_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
CREATE INDEX IF NOT EXISTS idx_candles_h4_symbol ON candles_h4(symbol);

-- candles_h1
ALTER TABLE candles_h1 ADD COLUMN IF NOT EXISTS symbol TEXT NOT NULL DEFAULT 'EURUSD';
ALTER TABLE candles_h1 DROP CONSTRAINT IF EXISTS candles_h1_timestamp_utc_key;
ALTER TABLE candles_h1 ADD CONSTRAINT candles_h1_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
CREATE INDEX IF NOT EXISTS idx_candles_h1_symbol ON candles_h1(symbol);
