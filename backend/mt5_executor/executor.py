#!/usr/bin/env python3
"""
MT5 Trade Executor for Set & Trend
===================================
Monitors for signals and executes trades on MT5 demo account.

Requirements:
- MetaTrader5 terminal must be running on a Windows machine
- This script connects via the MT5 Python API

For Linux: You need to run MT5 on Windows/Wine or use a VPS with Windows.
Alternative: Use the Telegram signals for manual trading.
"""

import os
import sys
import json
import time
import logging
from datetime import datetime
from pathlib import Path

# MT5 Configuration
MT5_LOGIN = 5045820618
MT5_PASSWORD = "J*CmHaR1"
MT5_SERVER = "MetaQuotes-Demo"

# Trading Configuration  
RISK_PERCENT = 1.0  # Risk 1% per trade
DEFAULT_SL_PIPS = 50  # Default stop loss in pips
DEFAULT_TP_MULTIPLIER = 2.0  # TP = SL * 2 (1:2 RR)

# Telegram for notifications
TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN", "8587109876:AAEX86NYkrGAzcTKGpe-kKd66MxlGa3HrV8")
TELEGRAM_CHAT_ID = os.getenv("TELEGRAM_CHAT_ID", "8164222762")

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s',
    handlers=[
        logging.FileHandler('mt5_executor.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)


def send_telegram(message: str):
    """Send notification to Telegram"""
    import requests
    try:
        url = f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage"
        data = {"chat_id": TELEGRAM_CHAT_ID, "text": message, "parse_mode": "HTML"}
        requests.post(url, json=data, timeout=10)
    except Exception as e:
        logger.error(f"Telegram error: {e}")


def check_mt5_available():
    """Check if MT5 Python package is available"""
    try:
        import MetaTrader5 as mt5
        return True
    except ImportError:
        return False


def connect_mt5():
    """Connect to MT5 terminal"""
    import MetaTrader5 as mt5
    
    if not mt5.initialize():
        logger.error(f"MT5 initialize failed: {mt5.last_error()}")
        return False
    
    # Login to account
    if not mt5.login(MT5_LOGIN, password=MT5_PASSWORD, server=MT5_SERVER):
        logger.error(f"MT5 login failed: {mt5.last_error()}")
        mt5.shutdown()
        return False
    
    account_info = mt5.account_info()
    logger.info(f"Connected to MT5: {account_info.name} | Balance: {account_info.balance} {account_info.currency}")
    return True


def calculate_lot_size(symbol: str, sl_pips: float, risk_percent: float = RISK_PERCENT):
    """Calculate lot size based on risk percentage"""
    import MetaTrader5 as mt5
    
    account = mt5.account_info()
    if not account:
        return 0.01  # Default minimum
    
    balance = account.balance
    risk_amount = balance * (risk_percent / 100)
    
    # Get symbol info
    symbol_info = mt5.symbol_info(symbol)
    if not symbol_info:
        return 0.01
    
    # Calculate pip value
    pip_size = 0.0001 if "JPY" not in symbol else 0.01
    point = symbol_info.point
    pip_value = (pip_size / point) * symbol_info.trade_tick_value
    
    # Lot size = Risk Amount / (SL pips * pip value)
    if sl_pips > 0 and pip_value > 0:
        lot_size = risk_amount / (sl_pips * pip_value)
        # Round to valid lot step
        lot_step = symbol_info.volume_step
        lot_size = round(lot_size / lot_step) * lot_step
        lot_size = max(symbol_info.volume_min, min(lot_size, symbol_info.volume_max))
        return lot_size
    
    return 0.01


def execute_trade(symbol: str, direction: str, entry: float, sl: float, tp: float):
    """Execute a trade on MT5"""
    import MetaTrader5 as mt5
    
    # Get current price
    tick = mt5.symbol_info_tick(symbol)
    if not tick:
        logger.error(f"Failed to get tick for {symbol}")
        return None
    
    # Determine order type
    if direction.upper() == "LONG":
        order_type = mt5.ORDER_TYPE_BUY
        price = tick.ask
    else:
        order_type = mt5.ORDER_TYPE_SELL
        price = tick.bid
    
    # Calculate SL distance in pips
    pip_size = 0.0001 if "JPY" not in symbol else 0.01
    sl_pips = abs(entry - sl) / pip_size
    
    # Calculate lot size
    lot = calculate_lot_size(symbol, sl_pips)
    
    # Prepare request
    request = {
        "action": mt5.TRADE_ACTION_DEAL,
        "symbol": symbol,
        "volume": lot,
        "type": order_type,
        "price": price,
        "sl": sl,
        "tp": tp,
        "deviation": 20,
        "magic": 20260201,  # Magic number for identification
        "comment": "SetAndTrend",
        "type_time": mt5.ORDER_TIME_GTC,
        "type_filling": mt5.ORDER_FILLING_IOC,
    }
    
    # Send order
    result = mt5.order_send(request)
    
    if result.retcode != mt5.TRADE_RETCODE_DONE:
        logger.error(f"Order failed: {result.retcode} - {result.comment}")
        send_telegram(f"❌ <b>Trade Failed</b>\n{symbol} {direction}\nError: {result.comment}")
        return None
    
    logger.info(f"Order executed: {symbol} {direction} {lot} lots @ {price}")
    
    # Send success notification
    send_telegram(f"""✅ <b>TRADE EXECUTED</b>

<b>Symbol:</b> {symbol}
<b>Direction:</b> {direction}
<b>Lots:</b> {lot}
<b>Entry:</b> {price:.5f}
<b>SL:</b> {sl:.5f}
<b>TP:</b> {tp:.5f}

<i>Order ID: {result.order}</i>""")
    
    return result


def watch_signals_file(filepath: str = "/home/set-and-trend/backend/live_signals.log"):
    """Watch signals file for new trades to execute"""
    logger.info(f"Watching for signals in: {filepath}")
    
    last_position = 0
    processed_signals = set()
    
    while True:
        try:
            if not Path(filepath).exists():
                time.sleep(10)
                continue
            
            with open(filepath, 'r') as f:
                f.seek(last_position)
                new_lines = f.readlines()
                last_position = f.tell()
            
            for line in new_lines:
                line = line.strip()
                if not line or line in processed_signals:
                    continue
                
                # Parse signal: "2026-02-01 12:00:00 | EURUSD LONG | Entry: 1.10500 | Conf: 55.0%"
                try:
                    parts = line.split('|')
                    if len(parts) >= 3:
                        symbol_dir = parts[1].strip().split()
                        symbol = symbol_dir[0]
                        direction = symbol_dir[1]
                        entry = float(parts[2].split(':')[1].strip())
                        
                        # Calculate SL/TP (simplified)
                        pip_size = 0.0001 if "JPY" not in symbol else 0.01
                        sl_distance = DEFAULT_SL_PIPS * pip_size
                        tp_distance = sl_distance * DEFAULT_TP_MULTIPLIER
                        
                        if direction.upper() == "LONG":
                            sl = entry - sl_distance
                            tp = entry + tp_distance
                        else:
                            sl = entry + sl_distance
                            tp = entry - tp_distance
                        
                        logger.info(f"New signal: {symbol} {direction} @ {entry}")
                        execute_trade(symbol, direction, entry, sl, tp)
                        processed_signals.add(line)
                        
                except Exception as e:
                    logger.error(f"Failed to parse signal: {line} - {e}")
            
            time.sleep(5)  # Check every 5 seconds
            
        except Exception as e:
            logger.error(f"Error watching signals: {e}")
            time.sleep(10)


def main():
    print("=" * 60)
    print("   SET & TREND - MT5 TRADE EXECUTOR")
    print("=" * 60)
    print(f"Account: {MT5_LOGIN} @ {MT5_SERVER}")
    print(f"Risk: {RISK_PERCENT}% per trade")
    print("=" * 60)
    
    if not check_mt5_available():
        print("""
⚠️  MetaTrader5 Python package not available.

This is because MT5 only works on Windows.
Options:
1. Run this script on a Windows VPS with MT5 installed
2. Use Wine on Linux (experimental)
3. Trade manually using Telegram signals

For now, signals will be sent to Telegram.
You can manually execute them on your MT5 terminal.
""")
        print("Telegram signals are active. Manual trading mode.")
        return
    
    # Connect to MT5
    if not connect_mt5():
        print("Failed to connect to MT5. Check if terminal is running.")
        return
    
    print("✅ Connected to MT5")
    print("Watching for signals...")
    
    try:
        watch_signals_file()
    except KeyboardInterrupt:
        print("\nShutting down...")
    finally:
        import MetaTrader5 as mt5
        mt5.shutdown()


if __name__ == "__main__":
    main()
