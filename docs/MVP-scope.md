# Set The Trend — MVP Scope (Phase 1, FIXED)

## Entities (tables that exist in MVP)

### Core entities
- `users` — Single row for the trader (you). Fields: `id`, `created_at`.
- `accounts` — Single row per account (demo/live). Fields: `id`, `user_id`, `type` (demo/live), `broker_name`, `currency`, `balance`, `leverage`, `max_risk_per_trade_pct`, `max_daily_risk_pct`, `timezone`, `preferred_session`, `updated_at`.
- `candles_weekly` — Weekly EURUSD OHLC. Fields: `id`, `timestamp_utc`, `open`, `high`, `low`, `close`, `volume` (optional).
- `indicators_weekly` — Precomputed indicators per weekly candle. **Immutable, no user edits.** Fields: `id`, `candle_id`, `ema20`, `ema50`, `ema200`, `range_size`, `body_size`, `upper_wick`, `lower_wick`, `mid_price`, `last_swing_high_price`, `last_swing_low_price`, `computed_at`.
- `rules` — Static rule metadata. **No expressions, code-driven only.** Fields: `id`, `code` (e.g. "W1_TREND_BULLISH"), `name`, `timeframe` ("W1"), `description`.
- `rule_results` — Rule evaluations **per candle, not per trade**. Fields: `id`, `rule_id`, `candle_id`, `result` (PASS/FAIL), `evaluated_at`, `confidence_score` (optional float).
- `trades` — One row per trade lifecycle. Fields: `id`, `user_id`, `account_id`, `candle_id` (FK to candles_weekly), `symbol`, `timeframe`, `setup_timestamp_utc`, **snapshot fields** (`account_balance_at_setup`, `leverage_at_setup`, `max_risk_per_trade_pct_at_setup`, `timezone_at_setup`), `bias` (long/short), `planned_entry`, `planned_sl`, `planned_tp`, `planned_rr`, `planned_risk_pct`, `planned_risk_amount`, `planned_position_size`, `reason_for_trade`, `actual_entry`, `actual_sl`, `actual_tp`, `actual_risk_pct`, `actual_risk_amount`, `actual_position_size`, `execution_timestamp_utc`, `close_timestamp_utc`, `close_price`, `result` (win/loss/breakeven), `pips_gained`, `money_gained`, `rr_realized`, `duration_seconds`, `session`.
- `trade_feedback` — Post-trade behavioral data. Fields: `id`, `trade_id`, `followed_plan` (bool), `emotion_before` (enum), `emotion_during` (enum), `emotion_after` (enum), `biggest_mistake` (text), `screenshot_url` (text nullable), `feedback_at`.

**Total: 8 tables. No more.**

## What does NOT exist (Phase 2+)

- No Daily or 4H candles/indicators
- No other symbols (GBPUSD, etc.)
- No multi-timeframe rules
- No AOI (areas of interest) tables
- No session analytics tables
- No advanced patterns (head & shoulders, engulfing)
- No user permissions/roles
- No audit logs
- No file uploads (screenshots stored as URLs only)
- No real-time price feeds
- No broker integrations

## Hard-coded (never configurable)

- Symbol: `EURUSD`
- Timeframe: `Weekly` (W1)
- **EMA periods: 20, 50, 200 (fixed)**
- Swing definition: 2-bar lookback/forward (`H_t > max(H_{t-1},H_{t-2}) ∧ H_t > max(H_{t+1},H_{t+2})`)
- EMA touch proximity: θ = 0.3 × Range_t (fixed)
- Pip value calculation: standard forex (0.0001 for most pairs)
- Sessions: london/new_york/asian/custom enums
- Emotions: calm/anxious/fomo/revenge/other enums
- **Rule logic: Go switch-case on rules.code (no dynamic expressions)**

## Configurable (via DB or env)

**Account settings** (per `accounts` row):
- `max_risk_per_trade_pct`
- `max_daily_risk_pct` 
- `leverage`
- `preferred_session`
- `timezone`

**System settings** (env vars):
- Database connection
- JWT secret
- Port

## Postponed (no timeline)

- Multi-symbol support
- Multi-timeframe (Daily/4H)
- ATR indicator
- Configurable EMA periods
- Configurable proximity threshold θ
- Head & shoulders pattern detection
- Win rate correlation analytics (beyond basic SQL)
- Export to CSV/PDF
- Screenshot storage
- Mobile API endpoints
- Live candle updates
- Rule expression parsing/DSL

## MVP boundaries (enforced)

**Data ingestion**: Manual CSV upload or single API endpoint for weekly EURUSD candles only.

**Rule execution**: Runs deterministically on-demand for a specific candle timestamp. No auto-triggering. **Results stored per candle.**

**Trade creation**: Manual entry only. References specific `candle_id`. **No auto-generation.** Risk calculations enforced at planned trade stage.

**Analytics**: SQL-only. No charts, no ML. Win rate, avg R:R, streaks by rule/session/emotion via JOINs on `trades.candle_id → rule_results.candle_id`.

**API surface**: **7 endpoints only**:
1. POST /candles/weekly (ingest)
2. POST /rules/evaluate/{candle_id}
3. POST /trades (create setup)
4. PATCH /trades/{id}/execute
5. PATCH /trades/{id}/close
6. POST /trades/{id}/feedback
7. GET /analytics/{user_id}?filter=...

**Success = these 7 endpoints work end-to-end with sample data.**
