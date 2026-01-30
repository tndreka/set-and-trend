import psycopg2
import pandas as pd
import os
from dotenv import load_dotenv
from pathlib import Path

# Load environment variables from .env file
env_path = Path(__file__).parent / '.env'
load_dotenv(dotenv_path=env_path)

# Build DB config from environment
DB_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5433'),
    'database': os.getenv('DB_NAME', 'set_the_trend'),
    'user': os.getenv('DB_USER', 'stt_user'),
    'password': os.getenv('DB_PASSWORD')
}

def connect_db():
    """Connect to PostgreSQL database"""
    if not DB_CONFIG['password']:
        raise ValueError("DB_PASSWORD not set in .env file!")
    
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        print(f"✓ Connected to {DB_CONFIG['database']} on port {DB_CONFIG['port']}")
        return conn
    except Exception as e:
        print(f"✗ Connection failed: {e}")
        raise

def import_candles(df, timeframe='W1'):
    """Import candles from DataFrame to database"""
    conn = connect_db()
    cur = conn.cursor()
    
    # Map timeframe to table name
    table_map = {
        'W1': 'candles_weekly',
        'D1': 'candles_d1',
        'H4': 'candles_h4',
        'H1': 'candles_h1'
    }
    table = table_map.get(timeframe, 'candles_weekly')
    
    inserted = 0
    for _, row in df.iterrows():
        try:
            cur.execute(f"""
                INSERT INTO {table} (timestamp_utc, open, high, low, close, volume)
                VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT (timestamp_utc) DO NOTHING
            """, (
                row['timestamp'],
                row['open'],
                row['high'],
                row['low'],
                row['close'],
                row.get('volume', 0)
            ))
            inserted += 1
        except Exception as e:
            print(f"Error inserting row: {e}")
            continue
    
    conn.commit()
    cur.close()
    conn.close()
    print(f"✓ Imported {inserted} candles to {table}")

if __name__ == "__main__":
    # Example usage
    print("Script configured to use environment variables from .env file")
    print(f"DB_HOST: {DB_CONFIG['host']}")
    print(f"DB_PORT: {DB_CONFIG['port']}")
    print(f"DB_USER: {DB_CONFIG['user']}")
    print(f"DB_NAME: {DB_CONFIG['database']}")
    print("Password: [REDACTED]" if DB_CONFIG['password'] else "Password: NOT SET!")