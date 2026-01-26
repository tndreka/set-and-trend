import pandas as pd
import sys

print("Loading M5 data...")
df = pd.read_csv('/home/set-and-trend/backend/mt4_ready/eurusd-m5-bid-2015-01-01-2025-08-08_MT4.csv',
                 names=['DateTime', 'Open', 'High', 'Low', 'Close', 'Volume'])

# Parse datetime
df['DateTime'] = pd.to_datetime(df['DateTime'], format='%Y.%m.%d %H:%M')
df.set_index('DateTime', inplace=True)

print(f"Loaded {len(df)} M5 candles")
print(f"Date range: {df.index.min()} to {df.index.max()}")

# Resampling rules
agg_rules = {
    'Open': 'first',
    'High': 'max',
    'Low': 'min',
    'Close': 'last',
    'Volume': 'sum'
}

# H1: 1 hour candles
print("\nConverting to H1...")
h1 = df.resample('1h').agg(agg_rules).dropna()
h1.to_csv('/home/set-and-trend/backend/mt4_ready/EURUSD_H1_2015_2025.csv')
print(f"✅ H1 saved: {len(h1)} candles")

# H4: 4 hour candles
print("\nConverting to H4...")
h4 = df.resample('4h').agg(agg_rules).dropna()
h4.to_csv('/home/set-and-trend/backend/mt4_ready/EURUSD_H4_2015_2025.csv')
print(f"✅ H4 saved: {len(h4)} candles")

# D1: Daily candles
print("\nConverting to D1...")
d1 = df.resample('1D').agg(agg_rules).dropna()
d1.to_csv('/home/set-and-trend/backend/mt4_ready/EURUSD_D1_2015_2025.csv')
print(f"✅ D1 saved: {len(d1)} candles")

# W1: Weekly candles (regenerate for consistency)
print("\nConverting to W1...")
w1 = df.resample('W').agg(agg_rules).dropna()
w1.to_csv('/home/set-and-trend/backend/mt4_ready/EURUSD_W1_2015_2025.csv')
print(f"✅ W1 saved: {len(w1)} candles")

print("\n" + "="*50)
print("Summary:")
print(f"  H1: {len(h1)} candles")
print(f"  H4: {len(h4)} candles")
print(f"  D1: {len(d1)} candles")
print(f"  W1: {len(w1)} candles")
print("="*50)
