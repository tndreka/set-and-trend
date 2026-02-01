#!/usr/bin/env python3
"""
Generate W1 (weekly) candles from D1 (daily) data for all 5 major pairs.
Also adds symbol column to candles_weekly if missing.
"""

import psycopg2
from datetime import datetime, timedelta
import os
from dotenv import load_dotenv

load_dotenv('/home/set-and-trend/backend/.env')

PG_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'dbname': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD', ''),
}

MAJOR_PAIRS = ['EURUSD', 'GBPUSD', 'USDJPY', 'USDCHF', 'AUDUSD']

def ensure_weekly_table_ready(conn):
    """Add symbol column to candles_weekly if missing."""
    cursor = conn.cursor()
    
    # Check if symbol column exists
    cursor.execute("""
        SELECT column_name FROM information_schema.columns 
        WHERE table_name = 'candles_weekly' AND column_name = 'symbol';
    """)
    
    if cursor.fetchone() is None:
        print("Adding symbol column to candles_weekly...")
        cursor.execute("ALTER TABLE candles_weekly ADD COLUMN symbol TEXT NOT NULL DEFAULT 'EURUSD';")
        
        # Update unique constraint
        cursor.execute("ALTER TABLE candles_weekly DROP CONSTRAINT IF EXISTS candles_weekly_timestamp_utc_key;")
        cursor.execute("""
            ALTER TABLE candles_weekly 
            ADD CONSTRAINT candles_weekly_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
        """)
        
        # Add index
        cursor.execute("CREATE INDEX IF NOT EXISTS idx_candles_weekly_symbol ON candles_weekly(symbol);")
        conn.commit()
        print("✓ Symbol column added")
    else:
        print("✓ Symbol column already exists")
    
    return cursor

def get_week_start(dt):
    """Get Monday 00:00 UTC for the week containing dt."""
    # Forex week starts Sunday, but we'll use Monday for simplicity
    days_since_monday = dt.weekday()
    monday = dt - timedelta(days=days_since_monday)
    return monday.replace(hour=0, minute=0, second=0, microsecond=0)

def generate_weekly_from_daily(conn, symbol):
    """Aggregate daily candles into weekly candles."""
    cursor = conn.cursor()
    
    # Get all daily candles for this symbol
    cursor.execute("""
        SELECT timestamp_utc, open, high, low, close, volume
        FROM candles_d1
        WHERE symbol = %s
        ORDER BY timestamp_utc ASC
    """, (symbol,))
    
    daily_rows = cursor.fetchall()
    print(f"  {symbol}: {len(daily_rows)} daily candles")
    
    if not daily_rows:
        return 0
    
    # Group by week
    weeks = {}
    for row in daily_rows:
        ts, open_p, high_p, low_p, close_p, volume = row
        week_start = get_week_start(ts)
        
        if week_start not in weeks:
            weeks[week_start] = {
                'open': open_p,
                'high': high_p,
                'low': low_p,
                'close': close_p,
                'volume': volume or 0,
                'first_ts': ts
            }
        else:
            w = weeks[week_start]
            w['high'] = max(w['high'], high_p)
            w['low'] = min(w['low'], low_p)
            w['close'] = close_p  # Last close of week
            w['volume'] += volume or 0
    
    print(f"    Aggregated into {len(weeks)} weekly candles")
    
    # Insert weekly candles
    inserted = 0
    for week_start, w in weeks.items():
        try:
            cursor.execute("""
                INSERT INTO candles_weekly (symbol, timestamp_utc, open, high, low, close, volume)
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (symbol, timestamp_utc) DO NOTHING
            """, (symbol, week_start, w['open'], w['high'], w['low'], w['close'], w['volume']))
            inserted += cursor.rowcount
        except Exception as e:
            print(f"    Error inserting week {week_start}: {e}")
    
    conn.commit()
    print(f"    ✓ Inserted {inserted} new weekly candles")
    return inserted

def main():
    print("=" * 70)
    print("  GENERATE W1 CANDLES FROM D1 DATA")
    print("=" * 70)
    
    conn = psycopg2.connect(**PG_CONFIG)
    
    try:
        ensure_weekly_table_ready(conn)
        
        print("\nGenerating weekly candles...")
        total = 0
        for symbol in MAJOR_PAIRS:
            count = generate_weekly_from_daily(conn, symbol)
            total += count
        
        print(f"\nTotal new weekly candles: {total}")
        
        # Show final counts
        print("\n" + "=" * 70)
        print("  FINAL W1 COUNTS")
        print("=" * 70)
        
        cursor = conn.cursor()
        for symbol in MAJOR_PAIRS:
            cursor.execute("SELECT COUNT(*) FROM candles_weekly WHERE symbol = %s", (symbol,))
            count = cursor.fetchone()[0]
            
            cursor.execute("""
                SELECT MIN(timestamp_utc), MAX(timestamp_utc) 
                FROM candles_weekly WHERE symbol = %s
            """, (symbol,))
            min_ts, max_ts = cursor.fetchone()
            
            print(f"  {symbol}: {count:,} weeks ({min_ts.strftime('%Y-%m-%d') if min_ts else 'N/A'} → {max_ts.strftime('%Y-%m-%d') if max_ts else 'N/A'})")
    
    finally:
        conn.close()

if __name__ == '__main__':
    main()
