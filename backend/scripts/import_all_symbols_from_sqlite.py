#!/usr/bin/env python3
"""
Import ALL forex price data from SQLite (forex_prices.db) to PostgreSQL.
Imports all symbols and all timeframes including generating W1 from D1.
"""

import sqlite3
import psycopg2
from datetime import datetime, timezone
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv('/home/set-and-trend/backend/.env')

# Database connections
SQLITE_PATH = '/home/set-and-trend/backend/forex_prices.db'

# PostgreSQL connection from local .env
PG_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'dbname': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD', ''),
}

def get_sqlite_symbols(sqlite_conn, table_name):
    """Get all unique symbols from a SQLite table."""
    cursor = sqlite_conn.cursor()
    cursor.execute(f"SELECT DISTINCT symbol FROM {table_name} ORDER BY symbol")
    return [row[0] for row in cursor.fetchall()]

def import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table, symbol):
    """Import data from SQLite table to PostgreSQL table for a specific symbol."""
    sqlite_cursor = sqlite_conn.cursor()
    pg_cursor = pg_conn.cursor()
    
    # Get data from SQLite
    sqlite_cursor.execute(f'''
        SELECT datetime, open, high, low, close, volume
        FROM {sqlite_table}
        WHERE symbol = ?
        ORDER BY datetime ASC
    ''', (symbol,))
    
    rows = sqlite_cursor.fetchall()
    
    if not rows:
        return 0
    
    # Insert into PostgreSQL
    inserted = 0
    batch_size = 1000
    
    for i in range(0, len(rows), batch_size):
        batch = rows[i:i+batch_size]
        values = []
        
        for row in batch:
            timestamp_ms_str, open_p, high_p, low_p, close_p, volume = row
            # Convert milliseconds timestamp (stored as text) to datetime
            timestamp_ms = int(timestamp_ms_str)
            timestamp = datetime.fromtimestamp(timestamp_ms / 1000, tz=timezone.utc)
            values.append((symbol, timestamp, open_p, high_p, low_p, close_p, volume or 0))
        
        # Bulk insert with ON CONFLICT DO NOTHING
        args_str = ','.join(
            pg_cursor.mogrify("(%s, %s, %s, %s, %s, %s, %s)", v).decode('utf-8')
            for v in values
        )
        
        pg_cursor.execute(f'''
            INSERT INTO {pg_table} (symbol, timestamp_utc, open, high, low, close, volume)
            VALUES {args_str}
            ON CONFLICT (symbol, timestamp_utc) DO NOTHING
        ''')
        
        inserted += pg_cursor.rowcount
    
    pg_conn.commit()
    return inserted

def generate_weekly_candles(pg_conn, symbol):
    """Generate weekly candles from D1 candles for a symbol."""
    pg_cursor = pg_conn.cursor()
    
    # Get all D1 candles ordered by date
    pg_cursor.execute('''
        SELECT timestamp_utc, open, high, low, close, volume
        FROM candles_d1
        WHERE symbol = %s
        ORDER BY timestamp_utc ASC
    ''', (symbol,))
    
    rows = pg_cursor.fetchall()
    if not rows:
        return 0
    
    # Group by week (Monday-based weeks)
    weekly_candles = {}
    for row in rows:
        timestamp, open_p, high_p, low_p, close_p, volume = row
        # Get the Monday of the week
        week_start = timestamp - timedelta(days=timestamp.weekday())
        week_start = week_start.replace(hour=0, minute=0, second=0, microsecond=0)
        
        if week_start not in weekly_candles:
            weekly_candles[week_start] = {
                'open': open_p,
                'high': high_p,
                'low': low_p,
                'close': close_p,
                'volume': volume or 0
            }
        else:
            wc = weekly_candles[week_start]
            wc['high'] = max(wc['high'], high_p)
            wc['low'] = min(wc['low'], low_p)
            wc['close'] = close_p
            wc['volume'] += volume or 0
    
    # Insert weekly candles
    inserted = 0
    values = []
    for week_start, candle in sorted(weekly_candles.items()):
        values.append((
            symbol,
            week_start,
            float(candle['open']),
            float(candle['high']),
            float(candle['low']),
            float(candle['close']),
            candle['volume']
        ))
    
    if values:
        args_str = ','.join(
            pg_cursor.mogrify("(%s, %s, %s, %s, %s, %s, %s)", v).decode('utf-8')
            for v in values
        )
        
        pg_cursor.execute(f'''
            INSERT INTO candles_weekly (symbol, timestamp_utc, open, high, low, close, volume)
            VALUES {args_str}
            ON CONFLICT (symbol, timestamp_utc) DO NOTHING
        ''')
        
        inserted = pg_cursor.rowcount
    
    pg_conn.commit()
    return inserted

def main():
    from datetime import timedelta
    # Make timedelta available in generate_weekly_candles
    global timedelta
    
    print("=" * 70)
    print("  SQLITE → POSTGRESQL IMPORT (ALL SYMBOLS)")
    print("=" * 70)
    
    # Connect to SQLite
    print(f"\n📁 SQLite source: {SQLITE_PATH}")
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    
    # Connect to PostgreSQL
    print(f"🐘 PostgreSQL target: {PG_CONFIG['host']}:{PG_CONFIG['port']}/{PG_CONFIG['dbname']}")
    pg_conn = psycopg2.connect(**PG_CONFIG)
    
    try:
        # Mapping: SQLite table -> PostgreSQL table
        mappings = [
            ('daily_prices', 'candles_d1'),
            ('h4_prices', 'candles_h4'),
            ('hourly_prices', 'candles_h1'),
        ]
        
        # Get all symbols from D1
        symbols = get_sqlite_symbols(sqlite_conn, 'daily_prices')
        print(f"\n📊 Found {len(symbols)} symbols to import:")
        print(f"   {', '.join(symbols)}\n")
        
        total_imported = {}
        
        for sqlite_table, pg_table in mappings:
            print(f"\n{'='*70}")
            print(f"  Importing {sqlite_table} → {pg_table}")
            print(f"{'='*70}")
            
            table_total = 0
            symbols_in_table = get_sqlite_symbols(sqlite_conn, sqlite_table)
            
            for symbol in symbols_in_table:
                try:
                    count = import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table, symbol)
                    if count > 0:
                        print(f"  ✓ {symbol}: +{count} rows")
                    else:
                        print(f"  - {symbol}: already up to date")
                    table_total += count
                except Exception as e:
                    print(f"  ✗ {symbol}: Error - {e}")
                    pg_conn.rollback()
            
            total_imported[pg_table] = table_total
            print(f"\n  Total new rows in {pg_table}: {table_total}")
        
        # Generate weekly candles
        print(f"\n{'='*70}")
        print(f"  Generating Weekly (W1) candles from D1")
        print(f"{'='*70}")
        
        weekly_total = 0
        for symbol in symbols:
            try:
                count = generate_weekly_candles(pg_conn, symbol)
                if count > 0:
                    print(f"  ✓ {symbol}: +{count} weekly candles")
                else:
                    print(f"  - {symbol}: already up to date")
                weekly_total += count
            except Exception as e:
                print(f"  ✗ {symbol}: Error - {e}")
                pg_conn.rollback()
        
        total_imported['candles_weekly'] = weekly_total
        
        # Show final counts
        print(f"\n{'='*70}")
        print(f"  IMPORT COMPLETE - SUMMARY")
        print(f"{'='*70}")
        
        pg_cursor = pg_conn.cursor()
        for table in ['candles_d1', 'candles_h4', 'candles_h1', 'candles_weekly']:
            pg_cursor.execute(f"SELECT symbol, COUNT(*) FROM {table} GROUP BY symbol ORDER BY symbol")
            rows = pg_cursor.fetchall()
            total = sum(r[1] for r in rows)
            print(f"\n  {table}: {total} total rows ({len(rows)} symbols)")
            for symbol, count in rows:
                print(f"    {symbol}: {count}")
        
    finally:
        sqlite_conn.close()
        pg_conn.close()
    
    print(f"\n✅ All data imported successfully!")

if __name__ == '__main__':
    main()
