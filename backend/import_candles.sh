#!/bin/bash

DB_URL="postgres://stt_user:lantidhe42H%40%24%40@localhost:5432/set_the_trend?sslmode=disable"
CSV_FILE="/home/set-and-trend/backend/mt4_ready/EURUSD_weekly_2015_2025.csv"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  📊 IMPORTING WEEKLY CANDLE DATA"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "CSV file: $CSV_FILE"
echo ""

# Import CSV
psql "$DB_URL" << EOF
-- Create temp table for CSV import
CREATE TEMP TABLE temp_candles (
    datetime TEXT,
    open DECIMAL(12,5),
    high DECIMAL(12,5),
    low DECIMAL(12,5),
    close DECIMAL(12,5),
    volume BIGINT,
    ema12 DECIMAL(12,5),
    ema26 DECIMAL(12,5)
);

-- Import CSV (skip header)
\COPY temp_candles FROM '$CSV_FILE' WITH (FORMAT csv, HEADER true);

-- Show what we got
SELECT COUNT(*) as csv_rows FROM temp_candles;

-- Insert into candles_weekly
INSERT INTO candles_weekly (timestamp_utc, open, high, low, close, volume)
SELECT 
    datetime:: timestamp AT TIME ZONE 'UTC',
    open,
    high,
    low,
    close,
    volume
FROM temp_candles
ORDER BY datetime::timestamp;

-- Show result
SELECT 
    COUNT(*) as imported_candles,
    MIN(timestamp_utc) as first_candle,
    MAX(timestamp_utc) as last_candle
FROM candles_weekly;
EOF

echo ""
echo "✅ Import complete!"
