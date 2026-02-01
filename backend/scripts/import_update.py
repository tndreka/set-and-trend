#!/usr/bin/env python3
"""Import updated forex data from Dukascopy CSV to PostgreSQL"""

import os
import glob
import psycopg2
from datetime import datetime

DB_URL = "postgresql://stt_user:taulantdhe42H@$D@localhost:5432/set_the_trend"
DOWNLOAD_DIR = "/home/lanti/Get-MT4-Historical-Data/download"

def parse_symbol(filename):
    """Extract symbol from filename like eurusd-h4-bid-2025-08-01-2026-02-01.csv"""
    base = os.path.basename(filename)
    parts = base.split('-')
    return parts[0].upper()

def parse_timeframe(filename):
    """Extract timeframe from filename"""
    base = os.path.basename(filename)
    parts = base.split('-')
    return parts[1].upper()

def import_csv(conn, filepath):
    """Import a single CSV file to the appropriate table"""
    symbol = parse_symbol(filepath)
    tf = parse_timeframe(filepath)
    
    table_map = {
        'H4': 'candles_h4',
        'D1': 'candles_d1',
    }
    
    if tf not in table_map:
        print(f"  Skipping {tf} (not H4 or D1)")
        return 0
    
    table = table_map[tf]
    
    inserted = 0
    skipped = 0
    
    with open(filepath, 'r') as f:
        # Skip header
        next(f)
        
        cur = conn.cursor()
        for line in f:
            parts = line.strip().split(',')
            if len(parts) < 5:
                continue
            
            ts_ms = int(parts[0])
            ts = datetime.utcfromtimestamp(ts_ms / 1000)
            open_p = float(parts[1])
            high_p = float(parts[2])
            low_p = float(parts[3])
            close_p = float(parts[4])
            
            # Check if exists
            cur.execute(f"""
                SELECT 1 FROM {table} 
                WHERE timestamp_utc = %s AND symbol = %s
            """, (ts, symbol))
            
            if cur.fetchone():
                skipped += 1
                continue
            
            # Insert
            cur.execute(f"""
                INSERT INTO {table} (timestamp_utc, open, high, low, close, volume, symbol)
                VALUES (%s, %s, %s, %s, %s, 0, %s)
            """, (ts, open_p, high_p, low_p, close_p, symbol))
            inserted += 1
        
        conn.commit()
    
    print(f"  {symbol} {tf}: +{inserted} new, {skipped} skipped")
    return inserted

def generate_weekly_from_daily(conn, symbol):
    """Generate weekly candles from daily data"""
    cur = conn.cursor()
    
    # Get daily candles after Aug 2025
    cur.execute("""
        SELECT timestamp_utc, open, high, low, close 
        FROM candles_d1 
        WHERE symbol = %s AND timestamp_utc >= '2025-08-01'
        ORDER BY timestamp_utc
    """, (symbol,))
    
    daily = cur.fetchall()
    if not daily:
        return 0
    
    # Group by week
    weekly_candles = {}
    for ts, o, h, l, c in daily:
        # Get Monday of that week
        week_start = ts - __import__('datetime').timedelta(days=ts.weekday())
        week_start = week_start.replace(hour=0, minute=0, second=0, microsecond=0)
        
        if week_start not in weekly_candles:
            weekly_candles[week_start] = {
                'open': o, 'high': h, 'low': l, 'close': c
            }
        else:
            w = weekly_candles[week_start]
            w['high'] = max(w['high'], h)
            w['low'] = min(w['low'], l)
            w['close'] = c
    
    inserted = 0
    for week_ts, candle in weekly_candles.items():
        # Check if exists
        cur.execute("""
            SELECT 1 FROM candles_weekly 
            WHERE timestamp_utc = %s AND symbol = %s
        """, (week_ts, symbol))
        
        if cur.fetchone():
            continue
        
        cur.execute("""
            INSERT INTO candles_weekly (timestamp_utc, open, high, low, close, volume, symbol)
            VALUES (%s, %s, %s, %s, %s, 0, %s)
        """, (week_ts, candle['open'], candle['high'], candle['low'], candle['close'], symbol))
        inserted += 1
    
    conn.commit()
    return inserted

def main():
    print("=" * 60)
    print("  IMPORTING UPDATED FOREX DATA (Aug 2025 - Feb 2026)")
    print("=" * 60)
    
    conn = psycopg2.connect(
        host="localhost",
        database="set_the_trend",
        user="stt_user",
        password="taulantdhe42H@$D"
    )
    
    # Find all update CSVs
    pattern = os.path.join(DOWNLOAD_DIR, "*-2025-08-01-2026-02-01.csv")
    files = glob.glob(pattern)
    
    print(f"\nFound {len(files)} CSV files to import\n")
    
    total_inserted = 0
    for f in sorted(files):
        inserted = import_csv(conn, f)
        total_inserted += inserted
    
    # Generate weekly candles
    print("\nGenerating weekly candles...")
    for symbol in ['EURUSD', 'GBPUSD', 'USDJPY', 'USDCHF', 'AUDUSD']:
        w = generate_weekly_from_daily(conn, symbol)
        print(f"  {symbol} W1: +{w} new")
        total_inserted += w
    
    conn.close()
    
    print(f"\n✅ Total new candles: {total_inserted}")
    print("=" * 60)

if __name__ == "__main__":
    main()
