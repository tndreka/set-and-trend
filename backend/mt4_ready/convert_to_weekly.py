import pandas as pd

# Load your MT4 CSV (adjust filename if needed)
df = pd.read_csv('mt4_ready/eurusd-m5-bid-2015-01-01-2025-08-08_MT4.csv', 
                 names=['DateTime', 'Open', 'High', 'Low', 'Close', 'Volume'])

# Parse datetime (your format: 2025.08.07 20:50)
df['DateTime'] = pd.to_datetime(df['DateTime'], format='%Y.%m.%d %H:%M')

# Set index and resample to WEEKLY
df.set_index('DateTime', inplace=True)
weekly = df.resample('W').agg({
    'Open': 'first',
    'High': 'max', 
    'Low': 'min',
    'Close': 'last',
    'Volume': 'sum'
}).dropna()

# Compute EMAs (12 and 26 week)
weekly['EMA12'] = weekly['Close'].ewm(span=12, adjust=False).mean()
weekly['EMA26'] = weekly['Close'].ewm(span=26, adjust=False).mean()

# Save weekly data
weekly.to_csv('EURUSD_weekly_2015_2025.csv')
print(f"✅ Weekly data saved: {len(weekly)} weeks")
print(weekly.tail())
