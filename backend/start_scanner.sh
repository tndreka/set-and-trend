#!/bin/bash
# ============================================================================
# START LIVE SCANNER
# ============================================================================

cd /home/set-and-trend/backend

# Load Telegram credentials if saved
if [ -f ~/.stt_telegram ]; then
    source ~/.stt_telegram
fi

# Check for credentials
if [ -z "$TELEGRAM_BOT_TOKEN" ] || [ -z "$TELEGRAM_CHAT_ID" ]; then
    echo "⚠️  Telegram not configured. Signals will only be logged to file."
    echo ""
    echo "To enable Telegram alerts:"
    echo "  1. Create a bot via @BotFather on Telegram"
    echo "  2. Run: ./setup_telegram.sh for instructions"
    echo "  3. Set environment variables:"
    echo "     export TELEGRAM_BOT_TOKEN='your_token'"
    echo "     export TELEGRAM_CHAT_ID='your_chat_id'"
    echo ""
    echo "Or save to ~/.stt_telegram:"
    echo "  echo 'export TELEGRAM_BOT_TOKEN=your_token' >> ~/.stt_telegram"
    echo "  echo 'export TELEGRAM_CHAT_ID=your_chat_id' >> ~/.stt_telegram"
    echo ""
fi

echo "Starting Live Scanner..."
echo "Signals will be logged to: live_signals.log"
echo ""

./live_scanner
