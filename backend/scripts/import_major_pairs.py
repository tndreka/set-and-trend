#!/usr/bin/env python3
"""
Import 5 major forex pairs from SQLite to PostgreSQL.
Pairs: EURUSD, GBPUSD, USDJPY, USDCHF, AUDUSD
Timeframes: D1, H4, H1
"""

import sqlite3
import psycopg2
from datetime import datetime
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv('/home/set-and-trend/backend/.env')

# Database connections
SQLITE_PATH = '/home/set-and-trend/backend/forex_prices.db'

# PostgreSQL connection
PG_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'dbname': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD', ''),
}

# Top 5 major pairs
MAJOR_PAIRS = ['EURUSD', 'GBPUSD', 'USDJPY', 'USDCHF', 'AUDUSD']

def ensure_tables_exist(pg_conn):
    """Create multi-symbol candle tables if they don't exist."""
    cursor = pg_conn.cursor()
    
    for tf in ['d1', 'h4', 'h1']:
        table_name = f'candles_{tf}'
        
        # Check if symbol column exists
        cursor.execute(f"""
            SELECT column_name FROM information_schema.columns 
            WHERE table_name = '{table_name}' AND column_name = 'symbol';
        """)
        
        if cursor.fetchone() is None:
            # Add symbol column if missing
            try:
                cursor.execute(f"ALTER TABLE {table_name} ADD COLUMN symbol TEXT NOT NULL DEFAULT 'EURUSD';")
                print(f"  Added symbol column to {table_name}")
            except:
                pass
        
        # Drop old unique constraint if exists and create new one
        try:
            cursor.execute(f"ALTER TABLE {table_name} DROP CONSTRAINT IF EXISTS {table_name}_timestamp_utc_key;")
            cursor.execute(f"ALTER TABLE {table_name} DROP CONSTRAINT IF EXISTS {table_name}_symbol_timestamp_key;")
            cursor.execute(f"""
                ALTER TABLE {table_name} 
                ADD CONSTRAINT {table_name}_symbol_timestamp_key UNIQUE(symbol, timestamp_utc);
            """)
        except Exception as e:
            print(f"  Constraint update for {table_name}: {e}")
        
        # Create index on symbol
        try:
            cursor.execute(f"CREATE INDEX IF NOT EXISTS idx_{table_name}_symbol ON {table_name}(symbol);")
        except:
            pass
    
    pg_conn.commit()
    print("✓ Tables ready for multi-symbol data")

def import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table, symbols):
    """Import data from SQLite to PostgreSQL for multiple symbols."""
    sqlite_cursor = sqlite_conn.cursor()
    pg_cursor = pg_conn.cursor()
    
    total_inserted = 0
    
    for symbol in symbols:
        # Get data from SQLite
        sqlite_cursor.execute(f'''
            SELECT datetime, open, high, low, close, volume
            FROM {sqlite_table}
            WHERE symbol = ?
            ORDER BY datetime ASC
        ''', (symbol,))
        
        rows = sqlite_cursor.fetchall()
        print(f"  {symbol}: Found {len(rows)} rows in {sqlite_table}")
        
        if not rows:
            continue
        
        # Insert into PostgreSQL in batches
        batch_size = 1000
        inserted = 0
        
        for i in range(0, len(rows), batch_size):
            batch = rows[i:i+batch_size]
            values = []
            
            for row in batch:
                timestamp_ms, open_p, high_p, low_p, close_p, volume = row
                # Handle timestamp as string or int
                if isinstance(timestamp_ms, str):
                    timestamp_ms = int(timestamp_ms)
                timestamp = datetime.utcfromtimestamp(timestamp_ms / 1000)
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
        print(f"    ✓ Inserted {inserted} new rows")
        total_inserted += inserted
    
    return total_inserted

def main():
    print("=" * 70)
    print("  IMPORT 5 MAJOR PAIRS: EURUSD, GBPUSD, USDJPY, USDCHF, AUDUSD")
    print("=" * 70)
    
    # Connect to SQLite
    print(f"\nConnecting to SQLite: {SQLITE_PATH}")
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    
    # Connect to PostgreSQL
    print(f"Connecting to PostgreSQL: {PG_CONFIG['host']}:{PG_CONFIG['port']}/{PG_CONFIG['dbname']}")
    pg_conn = psycopg2.connect(**PG_CONFIG)
    
    try:
        # Ensure tables support multi-symbol
        print("\nPreparing tables...")
        ensure_tables_exist(pg_conn)
        
        # Mapping: SQLite table -> PostgreSQL table
        mappings = [
            ('daily_prices', 'candles_d1', 'Daily'),
            ('h4_prices', 'candles_h4', 'H4'),
            ('hourly_prices', 'candles_h1', 'Hourly'),
        ]
        
        print("\nImporting data...")
        for sqlite_table, pg_table, tf_name in mappings:
            print(f"\n{tf_name} ({sqlite_table} → {pg_table}):")
            try:
                count = import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table, MAJOR_PAIRS)
                print(f"  Total new rows: {count}")
            except Exception as e:
                print(f"  ✗ Error: {e}")
                pg_conn.rollback()
        
        print("\n" + "=" * 70)
        print("  IMPORT COMPLETE - FINAL COUNTS")
        print("=" * 70)
        
        # Show final counts per symbol
        pg_cursor = pg_conn.cursor()
        for tf_name, pg_table in [('D1', 'candles_d1'), ('H4', 'candles_h4'), ('H1', 'candles_h1')]:
            print(f"\n{tf_name} ({pg_table}):")
            for symbol in MAJOR_PAIRS:
                pg_cursor.execute(f"SELECT COUNT(*) FROM {pg_table} WHERE symbol = %s", (symbol,))
                count = pg_cursor.fetchone()[0]
                print(f"  {symbol}: {count:,} candles")
        
    finally:
        sqlite_conn.close()
        pg_conn.close()

if __name__ == '__main__':
    main()
