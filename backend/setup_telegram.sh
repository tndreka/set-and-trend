#!/bin/bash
# ============================================================================
# LIVE SCANNER SETUP GUIDE
# ============================================================================

echo "========================================"
echo "   SET & TREND - LIVE SCANNER SETUP"
echo "========================================"
echo ""

# ============================================================================
# STEP 1: Create Telegram Bot
# ============================================================================
echo "STEP 1: Create Telegram Bot"
echo "----------------------------"
echo "1. Open Telegram and search for @BotFather"
echo "2. Send /newbot"
echo "3. Choose a name: 'Set and Trend Trading Bot'"
echo "4. Choose a username: 'SetAndTrendBot' (must be unique)"
echo "5. Copy the API token that BotFather gives you"
echo ""
echo "Example token: 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
echo ""

# ============================================================================
# STEP 2: Get Your Chat ID
# ============================================================================
echo "STEP 2: Get Your Chat ID"
echo "-------------------------"
echo "1. Start a chat with your new bot"
echo "2. Send any message to the bot"
echo "3. Visit: https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates"
echo "4. Find 'chat':{'id': YOUR_CHAT_ID}"
echo ""
echo "Or use this command after setting TELEGRAM_BOT_TOKEN:"
echo "  curl 'https://api.telegram.org/bot\$TELEGRAM_BOT_TOKEN/getUpdates'"
echo ""

# ============================================================================
# STEP 3: Configure Environment
# ============================================================================
echo "STEP 3: Configure Environment"
echo "------------------------------"
echo "Add these to your ~/.bashrc or run before starting:"
echo ""
echo "  export TELEGRAM_BOT_TOKEN='your_bot_token_here'"
echo "  export TELEGRAM_CHAT_ID='your_chat_id_here'"
echo ""

# ============================================================================
# STEP 4: Start the Scanner
# ============================================================================
echo "STEP 4: Start the Scanner"
echo "--------------------------"
echo "  cd /home/set-and-trend/backend"
echo "  ./live_scanner"
echo ""
echo "Or run in background with tmux:"
echo "  tmux new -s scanner"
echo "  ./live_scanner"
echo "  # Press Ctrl+B then D to detach"
echo ""

# ============================================================================
# Configuration
# ============================================================================
echo "========================================"
echo "   CURRENT CONFIGURATION"
echo "========================================"
echo ""
echo "MT5 Account: 5045820618 @ MetaQuotes-Demo"
echo "Pairs: EURUSD, GBPUSD, USDJPY, USDCHF, AUDUSD"
echo "Threshold: 50% (targets 2-3 signals/week)"
echo "Check Interval: Every 4 hours (H4 close)"
echo ""
echo "Telegram Bot Token: ${TELEGRAM_BOT_TOKEN:-NOT SET}"
echo "Telegram Chat ID: ${TELEGRAM_CHAT_ID:-NOT SET}"
echo ""
