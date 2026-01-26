#!/usr/bin/env python3
"""
Import multi-timeframe candle data into PostgreSQL
"""

import pandas as pd
import psycopg2
from psycopg2.extras import execute_values
import sys
import os

# Database connection
DB_CONFIG = {
    'host': 'localhost',
    'port': 5432,
    'user': 'stt_user',
    'password': 'lantidhe42H@$@',
    'dbname': 'set_the_trend'
}

DATA_DIR = '/home/set-and-trend/backend/mt4_ready'

TIMEFRAMES = {
    'H1': ('EURUSD_H1_2015_2025.csv', 'candles_h1'),
    'H4': ('EURUSD_H4_2015_2025.csv', 'candles_h4'),
    'D1': ('EURUSD_D1_2015_2025.csv', 'candles_d1'),
}

def import_candles(conn, timeframe, csv_file, table_name):
    """Import candles from CSV to database table"""
    filepath = os.path.join(DATA_DIR, csv_file)
    
    print(f"\n{'='*50}")
    print(f"Importing {timeframe} from {csv_file}")
    
    # Read CSV
    df = pd.read_csv(filepath)
    df['DateTime'] = pd.to_datetime(df['DateTime'])
    
    print(f"  Loaded {len(df)} rows")
    print(f"  Date range: {df['DateTime'].min()} to {df['DateTime'].max()}")
    
    # Prepare data for insertion
    records = []
    for _, row in df.iterrows():
        records.append((
            row['DateTime'],
            float(row['Open']),
            float(row['High']),
            float(row['Low']),
            float(row['Close']),
            int(row['Volume']) if pd.notna(row['Volume']) else None
        ))
    
    # Insert in batches
    cur = conn.cursor()
    
    # Clear existing data
    cur.execute(f"DELETE FROM {table_name}")
    print(f"  Cleared existing data from {table_name}")
    
    # Insert new data
    insert_query = f"""
        INSERT INTO {table_name} (timestamp_utc, open, high, low, close, volume)
        VALUES %s
        ON CONFLICT (timestamp_utc) DO NOTHING
    """
    
    batch_size = 5000
    total_inserted = 0
    
    for i in range(0, len(records), batch_size):
        batch = records[i:i+batch_size]
        execute_values(cur, insert_query, batch)
        total_inserted += len(batch)
        print(f"  Inserted {total_inserted}/{len(records)} records...", end='\r')
    
    conn.commit()
    print(f"  ✅ Inserted {total_inserted} records into {table_name}")
    
    # Verify
    cur.execute(f"SELECT COUNT(*) FROM {table_name}")
    count = cur.fetchone()[0]
    print(f"  Verified: {count} records in {table_name}")
    
    cur.close()
    return count

def main():
    print("Multi-Timeframe Candle Importer")
    print("="*50)
    
    # Connect to database
    print("\nConnecting to database...")
    conn = psycopg2.connect(**DB_CONFIG)
    print("✅ Connected")
    
    results = {}
    
    for tf, (csv_file, table_name) in TIMEFRAMES.items():
        try:
            count = import_candles(conn, tf, csv_file, table_name)
            results[tf] = count
        except Exception as e:
            print(f"❌ Error importing {tf}: {e}")
            results[tf] = 0
    
    conn.close()
    
    # Summary
    print("\n" + "="*50)
    print("IMPORT SUMMARY")
    print("="*50)
    for tf, count in results.items():
        print(f"  {tf}: {count:,} candles")
    print("="*50)

if __name__ == '__main__':
    main()
