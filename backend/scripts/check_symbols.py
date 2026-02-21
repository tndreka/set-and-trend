#!/usr/bin/env python3
import psycopg2

conn = psycopg2.connect(
    host='localhost', 
    database='set_the_trend', 
    user='stt_user', 
    password='taulantdhe42H@$D'
)
cur = conn.cursor()

# Check each timeframe table
tables = ['candles_d1', 'candles_h4', 'candles_h1', 'candles_weekly']

for table in tables:
    try:
        cur.execute(f'SELECT DISTINCT symbol FROM {table} ORDER BY symbol')
        rows = cur.fetchall()
        print(f'\n=== {table.upper()} ===')
        for i, row in enumerate(rows, 1):
            # Get count for this symbol
            cur.execute(f"SELECT COUNT(*) FROM {table} WHERE symbol = %s", (row[0],))
            count = cur.fetchone()[0]
            print(f'  {i}. {row[0]:20} ({count:,} candles)')
        print(f'  Total: {len(rows)} symbols')
    except Exception as e:
        print(f'{table}: Error - {e}')

conn.close()
