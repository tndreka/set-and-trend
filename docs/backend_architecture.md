# Set The Trend — Backend Architecture (MVP, FIXED)

## Go Project Structure

backend/
├── cmd/api/main.go # HTTP server entrypoint (Gin)
├── internal/
│ ├── config/ # Env var loading, DB config
│ │ └── config.go
│ ├── domain/ # Entities, enums, validation invariants
│ │ ├── candle.go
│ │ ├── trade.go
│ │ ├── rule.go
│ │ └── enums.go # TradeResult, Session, Emotion enums
│ ├── services/ # Business logic (pure functions)
│ │ ├── marketdata.go # Candle → indicators + swings
│ │ ├── rules.go # Rule evaluation
│ │ └── trade.go # Risk calc, lifecycle
│ ├── repositories/ # DB CRUD (SQLC or pgx)
│ │ ├── candles.go
│ │ ├── indicators.go
│ │ ├── rules.go
│ │ ├── trades.go
│ │ └── feedback.go
│ ├── handlers/ # HTTP handlers (Gin)
│ │ ├── candles.go
│ │ ├── rules.go
│ │ ├── trades.go
│ │ └── analytics.go
│ └── constants/ # EURUSD pip value, fixed params
│ └── forex.go # PipValueEURUSD = 0.0001
├── migrations/ # SQL migrations
├── pkg/ # Shared utilities (validation, etc.)
└── go.mod


## Ownership 

| Responsibility | Owner | Inputs | Outputs | When it runs |
|---------------|--------|--------|---------|--------------|
| **Candle ingestion** | `repositories.Candles` | Raw OHLC JSON/CSV | `candles_weekly` row | **On-demand** (POST /candles/weekly) |
| **Indicator computation** | `services.MarketData.ComputeIndicators(candle_id)` | `candles_weekly` row | `indicators_weekly` row | **On-demand** (after candle ingest, sync) |
| **Swing detection** | `services.MarketData.DetectSwings(candles []Candle)` | Recent candles | Swing high/low prices | **On-demand** (inside ComputeIndicators) |
| **Rule evaluation** | `services.Rules.EvaluateAll(candle_id)` | `indicators_weekly` row | Multiple `rule_results` rows | **On-demand** (POST /rules/evaluate/{candle_id}, sync) |
| **Trade creation** | `services.Trade.CreateSetup` | Trade params + account snapshot + **candle_id rule_results check** | `trades` row (planned fields) | **On-demand** (POST /trades, sync) |
| **Trade execution** | `services.Trade.Execute` | Trade ID + actual prices | Update `trades` (actual fields) | **On-demand** (PATCH /trades/{id}/execute, sync) |
| **Trade close** | `services.Trade.Close` | Trade ID + close price | Update `trades` (outcome fields) | **On-demand** (PATCH /trades/{id}/close, sync) |
| **Feedback storage** | `repositories.TradeFeedback` | Feedback JSON | `trade_feedback` row | **On-demand** (POST /trades/{id}/feedback, sync) |
| **Analytics** | `repositories.Analytics` (pure SQL) | user_id + filters | JSON {win_rate, avg_rr, streaks} | **On-demand** (GET /analytics/{user_id}, sync) |

**Rule results are written once per candle and never updated. Re-evaluation requires deleting results explicitly (admin-only, manual).**

## Execution Flow (All Synchronous)


1. POST /candles/weekly
    → repositories.Candles.Create()
    → services.MarketData.ComputeIndicators() [auto-called, includes swings]
    → repositories.Indicators.Create()

2. POST /rules/evaluate/{candle_id}
    → services.Rules.EvaluateAll()
    → repositories.RuleResults.CreateMany()

3. POST /trades
    → services.Trade.CreateSetup()
    ↓ Validates: rule_results exist for candle_id
    → repositories.Trades.Create()

4. PATCH /trades/{id}/execute
    → services.Trade.Execute()
    → repositories.Trades.Update()

5. PATCH /trades/{id}/close
    → services.Trade.Close() [compute pips, rr]
    → repositories.Trades.Update()

6. POST /trades/{id}/feedback
    → repositories.TradeFeedback.Create()


## Sync vs Async (MVP = 100% Sync)

- **All operations synchronous** — no background jobs, no queues.
- **No real-time** — weekly candles only, manual triggers.
- **No caching** — recompute indicators/rules on every request (data small).
- **DB transactions** per endpoint — candle+indicators in one TX.

## Service Boundaries (Strict)

services.MarketData
├── ComputeIndicators(candle_id) → indicators_weekly row
│ └── DetectSwings(candles []Candle) → swing prices (internal)
└── constants.PipValueEURUSD → 0.0001 (from constants/forex.go)

services.Rules
├── EvaluateAll(candle_id) → []rule_results
└── EvaluateSingle(rule_code, indicators) → PASS/FAIL

services.Trade
├── CreateSetup(params, account, candle_id) → validated trade
│ └── Reject if: no rule_results for candle_id
├── ComputePositionSize(risk, sl_distance) → lots
├── Execute(trade_id, actuals) → updated trade
└── Close(trade_id, close_price) → outcome fields

repositories.* [SQLC generated]
├── Create/Read/Update/List only
└── No business logic

domain.*
├── TradeResult enum (win/loss/breakeven)
├── Session enum (london/new_york/asian/custom)
├── Emotion enum (calm/anxious/fomo/revenge/other)
└── Validation methods (no DB, no HTTP)

