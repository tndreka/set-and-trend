# MT5 → Go Backend Integration

## Architecture (Option A - ZeroMQ)

```
┌─────────────────┐      ZeroMQ       ┌──────────────────────┐
│  MT5 Terminal   │ ←──────────────→  │  Go Backend (VPS)    │
│  (Your PC)      │   TCP:5555        │  - Confluence Scorer │
│                 │                    │  - Signal Generator  │
│  EA: ZMQ Client │ ←── Signals ────  │  - Position Manager  │
└─────────────────┘                    └──────────────────────┘
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │ Telegram Bot │
                                       │ Take/Ignore  │
                                       └──────────────┘
```

## MT5 EA Code (for ZeroMQ)

Save as `ConfluenceExecutor.mq5`:

```mql5
#include <Zmq/Zmq.mqh>

input string ZMQ_HOST = "tcp://your-vps-ip:5555";
input double RISK_PERCENT = 1.0;

Context context("confluence");
Socket subscriber(context, ZMQ_SUB);

int OnInit() {
   subscriber.connect(ZMQ_HOST);
   subscriber.subscribe("");
   Print("Connected to confluence engine");
   return INIT_SUCCEEDED;
}

void OnTick() {
   ZmqMsg message;
   if (subscriber.recv(message, ZMQ_NOBLOCK)) {
      string signal = message.getData();
      ProcessSignal(signal);
   }
}

void ProcessSignal(string signal) {
   // Format: "EURUSD|SHORT|1.0850|1.0900|1.0750|60.5"
   string parts[];
   StringSplit(signal, '|', parts);
   
   string symbol = parts[0];
   string direction = parts[1];
   double entry = StringToDouble(parts[2]);
   double sl = StringToDouble(parts[3]);
   double tp = StringToDouble(parts[4]);
   double confluence = StringToDouble(parts[5]);
   
   double lots = CalculateLots(symbol, sl, RISK_PERCENT);
   
   MqlTradeRequest request = {};
   request.symbol = symbol;
   request.volume = lots;
   request.type = direction == "LONG" ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
   request.price = direction == "LONG" ? SymbolInfoDouble(symbol, SYMBOL_ASK) : SymbolInfoDouble(symbol, SYMBOL_BID);
   request.sl = sl;
   request.tp = tp;
   request.comment = StringFormat("Confluence %.1f%%", confluence);
   
   MqlTradeResult result;
   OrderSend(request, result);
}

double CalculateLots(string symbol, double sl, double riskPct) {
   double balance = AccountInfoDouble(ACCOUNT_BALANCE);
   double riskAmount = balance * riskPct / 100;
   double tickValue = SymbolInfoDouble(symbol, SYMBOL_TRADE_TICK_VALUE);
   double tickSize = SymbolInfoDouble(symbol, SYMBOL_TRADE_TICK_SIZE);
   double slPips = MathAbs(SymbolInfoDouble(symbol, SYMBOL_BID) - sl) / tickSize;
   
   double lots = riskAmount / (slPips * tickValue);
   double minLot = SymbolInfoDouble(symbol, SYMBOL_VOLUME_MIN);
   double maxLot = SymbolInfoDouble(symbol, SYMBOL_VOLUME_MAX);
   
   return MathMin(maxLot, MathMax(minLot, NormalizeDouble(lots, 2)));
}
```

## Go ZeroMQ Publisher

```go
package zeromq

import (
    zmq "github.com/pebbe/zmq4"
)

type SignalPublisher struct {
    socket *zmq.Socket
}

func NewSignalPublisher(port int) (*SignalPublisher, error) {
    socket, err := zmq.NewSocket(zmq.PUB)
    if err != nil {
        return nil, err
    }
    socket.Bind(fmt.Sprintf("tcp://*:%d", port))
    return &SignalPublisher{socket: socket}, nil
}

func (p *SignalPublisher) Publish(signal TradeSignal) error {
    msg := fmt.Sprintf("%s|%s|%.5f|%.5f|%.5f|%.1f",
        signal.Symbol, signal.Direction,
        signal.Entry, signal.StopLoss, signal.TakeProfit,
        signal.Confluence)
    _, err := p.socket.Send(msg, 0)
    return err
}
```

## Telegram Bot Integration

```go
// internal/telegram/bot.go
package telegram

import (
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendSignal(bot *tgbotapi.BotAPI, chatID int64, signal TradeSignal) {
    text := fmt.Sprintf(`
🎯 *NEW SIGNAL*

Symbol: %s
Direction: %s
Entry: %.5f
SL: %.5f
TP: %.5f
R:R: %.2f
Confluence: %.1f%%
`,
        signal.Symbol, signal.Direction,
        signal.Entry, signal.StopLoss, signal.TakeProfit,
        signal.RiskReward, signal.Confluence)

    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("✅ TAKE", "take_"+signal.ID),
            tgbotapi.NewInlineKeyboardButtonData("❌ IGNORE", "ignore_"+signal.ID),
        ),
    )

    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "Markdown"
    msg.ReplyMarkup = keyboard
    bot.Send(msg)
}
```

## Quick Start Commands

```bash
# 1. Paper trade (current - no real money)
cd /home/set-and-trend/backend
./paper_sim  # Historical simulation

# 2. Run live paper trade (logs only, no execution)
./paper_trade  # Checks every 4 hours

# 3. Deploy to VPS (Phase 2)
# Install ZeroMQ: apt install libzmq3-dev
# go get github.com/pebbe/zmq4
# Build and run on VPS

# 4. Connect MT5 (Phase 3)
# Install ZeroMQ MQL5 library
# Compile ConfluenceExecutor.mq5
# Attach to chart
```

## Confluence Threshold Settings

| Threshold | Signals/Week | Notes |
|-----------|-------------|-------|
| 55% | 3.38 | Too many |
| 57% | 1.57 | ✅ Optimal |
| 60% | 0.92 | Conservative |
| 65% | 0.33 | Very selective |
