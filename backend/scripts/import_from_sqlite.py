#!/usr/bin/env python3
"""
Import forex price data from SQLite (forex_prices.db) to PostgreSQL.
This enables running simulations with multi-timeframe data.
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

# PostgreSQL connection from local .env
PG_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'dbname': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD', ''),
}

def create_multi_tf_tables(pg_conn):
    """Create tables for multi-timeframe candles if they don't exist."""
    cursor = pg_conn.cursor()
    
    # Create tables for each timeframe
    for tf in ['daily', 'h4', 'h1', 'm30']:
        table_name = f'candles_{tf}'
        cursor.execute(f'''
            CREATE TABLE IF NOT EXISTS {table_name} (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                symbol TEXT NOT NULL DEFAULT 'EURUSD',
                timestamp_utc TIMESTAMP WITH TIME ZONE NOT NULL,
                open DECIMAL(12,5) NOT NULL,
                high DECIMAL(12,5) NOT NULL,
                low DECIMAL(12,5) NOT NULL,
                close DECIMAL(12,5) NOT NULL,
                volume BIGINT,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                UNIQUE(symbol, timestamp_utc)
            );
            
            CREATE INDEX IF NOT EXISTS idx_{table_name}_timestamp 
            ON {table_name}(timestamp_utc);
            
            CREATE INDEX IF NOT EXISTS idx_{table_name}_symbol 
            ON {table_name}(symbol);
        ''')
    
    pg_conn.commit()
    print("✓ Created multi-timeframe tables")

def import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table, symbol='EURUSD'):
    """Import data from SQLite table to PostgreSQL table."""
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
    print(f"  Found {len(rows)} rows in {sqlite_table} for {symbol}")
    
    if not rows:
        return 0
    
    # Insert into PostgreSQL
    inserted = 0
    batch_size = 1000
    
    for i in range(0, len(rows), batch_size):
        batch = rows[i:i+batch_size]
        values = []
        
        for row in batch:
            timestamp_ms, open_p, high_p, low_p, close_p, volume = row
            # Convert milliseconds timestamp to datetime
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
        
        if (i + batch_size) % 10000 == 0:
            print(f"    Processed {i + batch_size} rows...")
    
    pg_conn.commit()
    return inserted

def main():
    print("=" * 60)
    print("  SQLITE → POSTGRESQL IMPORT")
    print("=" * 60)
    
    # Connect to SQLite
    print(f"\nConnecting to SQLite: {SQLITE_PATH}")
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    
    # Connect to PostgreSQL
    print(f"Connecting to PostgreSQL: {PG_CONFIG['host']}:{PG_CONFIG['port']}/{PG_CONFIG['dbname']}")
    pg_conn = psycopg2.connect(**PG_CONFIG)
    
    try:
        # Create tables
        create_multi_tf_tables(pg_conn)
        
        # Mapping: SQLite table -> PostgreSQL table
        mappings = [
            ('daily_prices', 'candles_daily'),
            ('h4_prices', 'candles_h4'),
            ('hourly_prices', 'candles_h1'),
            ('m30_prices', 'candles_m30'),
        ]
        
        print("\nImporting data...")
        for sqlite_table, pg_table in mappings:
            print(f"\n{sqlite_table} → {pg_table}")
            try:
                count = import_timeframe(sqlite_conn, pg_conn, sqlite_table, pg_table)
                print(f"  ✓ Inserted {count} new rows")
            except Exception as e:
                print(f"  ✗ Error: {e}")
                pg_conn.rollback()
        
        print("\n" + "=" * 60)
        print("  IMPORT COMPLETE")
        print("=" * 60)
        
        # Show final counts
        pg_cursor = pg_conn.cursor()
        for _, pg_table in mappings:
            pg_cursor.execute(f"SELECT COUNT(*) FROM {pg_table}")
            count = pg_cursor.fetchone()[0]
            print(f"  {pg_table}: {count} rows")
        
    finally:
        sqlite_conn.close()
        pg_conn.close()

if __name__ == '__main__':
    main()
