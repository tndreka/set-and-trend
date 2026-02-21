#!/usr/bin/env python3
"""
COMPREHENSIVE MULTI-TIMEFRAME IMPORT
=====================================
Imports ALL symbols from forex_prices.db to PostgreSQL
Creates tables: candles_w1, candles_d1, candles_h4, candles_h1, candles_m30, candles_m5
Each with: DateTime, Open, High, Low, Close, Volume, Symbol
"""

import sqlite3
import psycopg2
from psycopg2.extras import execute_values
from datetime import datetime, timedelta
import os
from dotenv import load_dotenv
import sys

# Load environment variables
load_dotenv('/home/set-and-trend/backend/.env')

# Database connections
SQLITE_PATH = '/home/set-and-trend/backend/forex_prices.db'

PG_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'dbname': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD', ''),
}

# Mapping: PostgreSQL table -> SQLite table
TIMEFRAME_MAP = {
    'candles_w1': None,           # Generated from daily_prices
    'candles_d1': 'daily_prices',
    'candles_h4': 'h4_prices',
    'candles_h1': 'hourly_prices',
    'candles_m30': 'm30_prices',
    'candles_m5': 'm5_prices',
}

def connect_sqlite():
    """Connect to SQLite database."""
    return sqlite3.connect(SQLITE_PATH)

def connect_postgres():
    """Connect to PostgreSQL database."""
    return psycopg2.connect(**PG_CONFIG)

def create_all_tables(pg_conn):
    """Create all candle tables for all timeframes if they don't exist."""
    cursor = pg_conn.cursor()
    
    for table_name in TIMEFRAME_MAP.keys():
        print(f"Creating table {table_name} if not exists...")
        
        cursor.execute(f'''
            CREATE TABLE IF NOT EXISTS {table_name} (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                symbol TEXT NOT NULL,
                timestamp_utc TIMESTAMP WITH TIME ZONE NOT NULL,
                open DECIMAL(12,5) NOT NULL,
                high DECIMAL(12,5) NOT NULL,
                low DECIMAL(12,5) NOT NULL,
                close DECIMAL(12,5) NOT NULL,
                volume BIGINT DEFAULT 0,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                UNIQUE(symbol, timestamp_utc)
            );
        ''')
        
        cursor.execute(f'''
            CREATE INDEX IF NOT EXISTS idx_{table_name}_timestamp 
            ON {table_name}(timestamp_utc);
        ''')
        
        cursor.execute(f'''
            CREATE INDEX IF NOT EXISTS idx_{table_name}_symbol 
            ON {table_name}(symbol);
        ''')
        
        cursor.execute(f'''
            CREATE INDEX IF NOT EXISTS idx_{table_name}_symbol_ts 
            ON {table_name}(symbol, timestamp_utc DESC);
        ''')
    
    pg_conn.commit()
    print("✅ All tables created/verified")

def get_sqlite_symbols(sqlite_conn):
    """Get all unique symbols from SQLite."""
    cursor = sqlite_conn.cursor()
    cursor.execute("SELECT DISTINCT symbol FROM daily_prices ORDER BY symbol")
    return [row[0] for row in cursor.fetchall()]

def get_pg_symbols(pg_conn, table_name):
    """Get all unique symbols from PostgreSQL table."""
    cursor = pg_conn.cursor()
    cursor.execute(f"SELECT DISTINCT symbol FROM {table_name} ORDER BY symbol")
    return [row[0] for row in cursor.fetchall()]

def get_pg_count(pg_conn, table_name, symbol):
    """Get count of candles for a symbol in PostgreSQL."""
    cursor = pg_conn.cursor()
    cursor.execute(f"SELECT COUNT(*) FROM {table_name} WHERE symbol = %s", (symbol,))
    return cursor.fetchone()[0]

def get_sqlite_count(sqlite_conn, table_name, symbol):
    """Get count of candles for a symbol in SQLite."""
    cursor = sqlite_conn.cursor()
    cursor.execute(f"SELECT COUNT(*) FROM {table_name} WHERE symbol = ?", (symbol,))
    return cursor.fetchone()[0]

def import_candles(sqlite_conn, pg_conn, pg_table, sqlite_table, symbol, batch_size=1000):
    """Import candles from SQLite to PostgreSQL for a specific symbol."""
    sqlite_cursor = sqlite_conn.cursor()
    pg_cursor = pg_conn.cursor()
    
    # Get existing timestamps in PostgreSQL to avoid duplicates
    pg_cursor.execute(f"""
        SELECT timestamp_utc FROM {pg_table} 
        WHERE symbol = %s
    """, (symbol,))
    existing_timestamps = set(row[0] for row in pg_cursor.fetchall())
    
    # Fetch from SQLite
    sqlite_cursor.execute(f"""
        SELECT datetime, open, high, low, close, volume
        FROM {sqlite_table}
        WHERE symbol = ?
        ORDER BY datetime
    """, (symbol,))
    
    rows = sqlite_cursor.fetchall()
    
    # Filter out existing and prepare for insert
    to_insert = []
    for row in rows:
        dt_str, open_, high, low, close, volume = row
        
        # Parse datetime
        try:
            if 'T' in dt_str:
                ts = datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
            else:
                ts = datetime.strptime(dt_str, '%Y-%m-%d %H:%M:%S')
        except:
            try:
                ts = datetime.strptime(dt_str, '%Y-%m-%d')
            except:
                continue
        
        # Skip if already exists
        if ts in existing_timestamps:
            continue
        
        to_insert.append((
            symbol,
            ts,
            float(open_) if open_ else 0,
            float(high) if high else 0,
            float(low) if low else 0,
            float(close) if close else 0,
            int(volume) if volume else 0,
        ))
    
    if not to_insert:
        return 0
    
    # Batch insert
    insert_sql = f"""
        INSERT INTO {pg_table} (symbol, timestamp_utc, open, high, low, close, volume)
        VALUES %s
        ON CONFLICT (symbol, timestamp_utc) DO NOTHING
    """
    
    for i in range(0, len(to_insert), batch_size):
        batch = to_insert[i:i+batch_size]
        execute_values(pg_cursor, insert_sql, batch)
    
    pg_conn.commit()
    return len(to_insert)

def generate_weekly_candles(pg_conn, symbol):
    """Generate weekly candles from daily candles."""
    cursor = pg_conn.cursor()
    
    # Get existing weekly candles
    cursor.execute("""
        SELECT timestamp_utc FROM candles_w1 WHERE symbol = %s
    """, (symbol,))
    existing_weeks = set(row[0] for row in cursor.fetchall())
    
    # Get daily candles grouped by week
    cursor.execute("""
        SELECT 
            date_trunc('week', timestamp_utc) as week_start,
            (array_agg(open ORDER BY timestamp_utc))[1] as open,
            MAX(high) as high,
            MIN(low) as low,
            (array_agg(close ORDER BY timestamp_utc DESC))[1] as close,
            SUM(volume) as volume
        FROM candles_d1
        WHERE symbol = %s
        GROUP BY week_start
        ORDER BY week_start
    """, (symbol,))
    
    rows = cursor.fetchall()
    
    to_insert = []
    for row in rows:
        week_start, open_, high, low, close, volume = row
        
        if week_start in existing_weeks:
            continue
        
        to_insert.append((
            symbol,
            week_start,
            float(open_),
            float(high),
            float(low),
            float(close),
            int(volume) if volume else 0,
        ))
    
    if not to_insert:
        return 0
    
    insert_sql = """
        INSERT INTO candles_w1 (symbol, timestamp_utc, open, high, low, close, volume)
        VALUES %s
        ON CONFLICT (symbol, timestamp_utc) DO NOTHING
    """
    execute_values(cursor, insert_sql, to_insert)
    pg_conn.commit()
    
    return len(to_insert)

def print_summary(pg_conn):
    """Print summary of all data in PostgreSQL."""
    cursor = pg_conn.cursor()
    
    print("\n" + "="*70)
    print("📊 POSTGRESQL CANDLE DATA SUMMARY")
    print("="*70)
    
    for table_name in TIMEFRAME_MAP.keys():
        cursor.execute(f"""
            SELECT symbol, COUNT(*), MIN(timestamp_utc), MAX(timestamp_utc)
            FROM {table_name}
            GROUP BY symbol
            ORDER BY symbol
        """)
        rows = cursor.fetchall()
        
        if rows:
            print(f"\n📈 {table_name.upper()}:")
            print("-"*70)
            for symbol, count, min_ts, max_ts in rows:
                print(f"  {symbol:18} | {count:>7} candles | {min_ts.date()} → {max_ts.date()}")
        else:
            print(f"\n📈 {table_name.upper()}: (empty)")

def main():
    print("="*70)
    print("🚀 MULTI-TIMEFRAME CANDLE IMPORT")
    print("   SQLite → PostgreSQL")
    print("="*70)
    
    # Connect to databases
    sqlite_conn = connect_sqlite()
    pg_conn = connect_postgres()
    
    print("\n✅ Connected to SQLite:", SQLITE_PATH)
    print("✅ Connected to PostgreSQL:", PG_CONFIG['dbname'])
    
    # Create all tables
    print("\n📦 Creating tables...")
    create_all_tables(pg_conn)
    
    # Get all symbols from SQLite
    symbols = get_sqlite_symbols(sqlite_conn)
    print(f"\n📋 Found {len(symbols)} symbols in SQLite:")
    print("   " + ", ".join(symbols))
    
    # Import each timeframe for each symbol
    total_imported = 0
    
    for pg_table, sqlite_table in TIMEFRAME_MAP.items():
        if sqlite_table is None:
            continue  # Skip W1 (generated)
        
        print(f"\n{'='*70}")
        print(f"📥 Importing {pg_table} from {sqlite_table}")
        print("-"*70)
        
        for symbol in symbols:
            sqlite_count = get_sqlite_count(sqlite_conn, sqlite_table, symbol)
            
            if sqlite_count == 0:
                continue
            
            pg_count_before = get_pg_count(pg_conn, pg_table, symbol)
            imported = import_candles(sqlite_conn, pg_conn, pg_table, sqlite_table, symbol)
            pg_count_after = get_pg_count(pg_conn, pg_table, symbol)
            
            if imported > 0:
                print(f"  ✅ {symbol:18} | +{imported:>6} candles | Total: {pg_count_after}")
                total_imported += imported
            else:
                print(f"  ⏭️  {symbol:18} | Already synced ({pg_count_after} candles)")
    
    # Generate weekly candles from daily
    print(f"\n{'='*70}")
    print("📥 Generating candles_w1 from candles_d1")
    print("-"*70)
    
    for symbol in symbols:
        d1_count = get_pg_count(pg_conn, 'candles_d1', symbol)
        if d1_count == 0:
            continue
        
        w1_before = get_pg_count(pg_conn, 'candles_w1', symbol)
        generated = generate_weekly_candles(pg_conn, symbol)
        w1_after = get_pg_count(pg_conn, 'candles_w1', symbol)
        
        if generated > 0:
            print(f"  ✅ {symbol:18} | +{generated:>6} candles | Total: {w1_after}")
            total_imported += generated
        else:
            print(f"  ⏭️  {symbol:18} | Already synced ({w1_after} candles)")
    
    # Print summary
    print_summary(pg_conn)
    
    print(f"\n{'='*70}")
    print(f"🎉 IMPORT COMPLETE! Total new candles: {total_imported}")
    print("="*70)
    
    # Close connections
    sqlite_conn.close()
    pg_conn.close()

if __name__ == '__main__':
    main()
