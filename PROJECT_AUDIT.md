# Set The Trend - Complete Project Audit & Context File

**Last Updated:** February 1, 2026  
**Audit Purpose:** Provide comprehensive context for ongoing development

---

## Table of Contents
1. [Project Overview](#1-project-overview)
2. [Architecture Summary](#2-architecture-summary)
3. [Database Schema (Deep Dive)](#3-database-schema-deep-dive)
4. [Pattern Detection System](#4-pattern-detection-system)
5. [Confluence Scoring System](#5-confluence-scoring-system)
6. [Rules Engine](#6-rules-engine)
7. [Services & API Layer](#7-services--api-layer)
8. [Frontend Status](#8-frontend-status)
9. [Current Completed Features](#9-current-completed-features)
10. [Known Gaps & TODOs](#10-known-gaps--todos)
11. [Next Steps Placeholder](#11-next-steps-placeholder)

---

## 1. Project Overview

### What is Set The Trend?

A **deterministic trading journal and rule engine** for disciplined discretionary traders. It turns subjective trading decisions into objective, queryable data.

### Core Philosophy
- **No execution automation** - does NOT connect to brokers or execute trades
- **No predictions** - does NOT use ML or "AI auto-trading"
- **Deterministic rules** - same inputs always produce same outputs
- **Single trader focus** - designed for one user with honest trade logging
- **Higher timeframe focus** - Weekly (W1), Daily (D1), 4-Hour (H4) timeframes

### One-Sentence Definition
> Set The Trend is a focused trading journal and rule engine turning clear numeric rules and structured post-trade feedback into real, testable edge.

---

## 2. Architecture Summary

### Tech Stack

| Layer | Technology |
|-------|------------|
| Backend Language | Go 1.23 |
| Web Framework | Gin |
| Database | PostgreSQL 16 |
| Query Layer | SQLC (compile-time SQL validation) |
| Connection Pool | pgx/v5 |
| Frontend | Next.js + TypeScript + Tailwind |
| Auth | JWT + bcrypt |

### Backend Structure

```
backend/
├── cmd/                         # Entry points (executables)
│   ├── api/main.go             # REST API server
│   ├── backtest/main.go        # Backtesting runner
│   ├── confluence/main.go      # Confluence simulation
│   ├── live_scanner/main.go    # Real-time pattern scanner
│   ├── paper_sim/main.go       # Paper trading simulation
│   ├── paper_trade/main.go     # Paper trade management
│   └── simulation/main.go      # Historical simulation
│
├── internal/
│   ├── config/                 # Config loading & DB connection
│   ├── confluence/             # Multi-timeframe confluence scoring
│   ├── constants/              # Forex constants (pip values, etc.)
│   ├── db/                     # SQLC generated code (DO NOT EDIT)
│   ├── domain/                 # Business enums & domain types
│   ├── execution/              # Trade execution cost models
│   ├── handlers/               # HTTP request handlers
│   ├── marketdata/             # Market data fetching
│   ├── middleware/             # Auth middleware
│   ├── patterns/               # ⭐ CRITICAL: Pattern detection algorithms
│   ├── repositories/           # Data access layer
│   ├── rules/                  # ⭐ CRITICAL: Rule evaluation engine
│   ├── services/               # Business logic layer
│   └── simulation/             # Simulation utilities
│
└── migrations/                  # SQL migration files
```

### Key Design Principles

1. **Database-Owned Temporal Authority**: All timestamps use PostgreSQL's `DEFAULT NOW()` - no `time.Now()` in Go for DB operations
2. **Immutable Intent Pattern**: Trade plans are stored once, never updated. Outcomes go in append-only execution log
3. **Deterministic Rule Evaluation**: Same candle + Same indicators = Same rule result

---

## 3. Database Schema (Deep Dive)

### Core Tables

#### Users & Accounts
```sql
-- Users table (simple auth)
users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
)

-- Trading accounts (with snapshot fields)
accounts (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    type account_type ('demo', 'live'),
    broker_name TEXT,
    currency TEXT,
    balance NUMERIC(12,2),
    leverage INT,
    max_risk_per_trade_pct NUMERIC(5,2),
    max_daily_risk_pct NUMERIC(5,2),
    timezone TEXT,
    preferred_session session_type
)
```

#### Multi-Timeframe Candle Tables

| Table | Purpose | Unique Key |
|-------|---------|------------|
| `candles_weekly` | W1 candles | `timestamp_utc` |
| `candles_d1` | D1 candles | `timestamp_utc` |
| `candles_h4` | H4 candles | `timestamp_utc` |
| `candles_h1` | H1 candles | `timestamp_utc` |

**Schema (all tables):**
```sql
(
    id UUID PRIMARY KEY,
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,
    open NUMERIC(12,5),
    high NUMERIC(12,5),
    low NUMERIC(12,5),
    close NUMERIC(12,5),
    volume BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
)
```

#### Multi-Timeframe Indicator Tables

| Table | Linked To |
|-------|-----------|
| `indicators_weekly` | `candles_weekly` |
| `indicators_d1` | `candles_d1` |
| `indicators_h4` | `candles_h4` |
| `indicators_h1` | `candles_h1` |

**Schema (all tables):**
```sql
(
    id UUID PRIMARY KEY,
    candle_id UUID REFERENCES candles_xxx(id),
    ema20 NUMERIC(12,5),
    ema50 NUMERIC(12,5),
    ema200 NUMERIC(12,5),
    range_size NUMERIC(12,5),
    body_size NUMERIC(12,5),
    upper_wick NUMERIC(12,5),
    lower_wick NUMERIC(12,5),
    mid_price NUMERIC(12,5),
    last_swing_high_price NUMERIC(12,5),
    last_swing_low_price NUMERIC(12,5),
    computed_at TIMESTAMPTZ DEFAULT NOW()
)
```

#### Rules System

```sql
-- Rule definitions
rules (
    id UUID PRIMARY KEY,
    code TEXT UNIQUE,           -- e.g., 'W1_TREND_BULLISH'
    name TEXT,
    timeframe rule_timeframe,   -- ENUM: W1, D1, H4, H1
    description TEXT
)

-- Rule evaluation results (per candle)
rule_results (
    id UUID PRIMARY KEY,
    rule_id UUID REFERENCES rules(id),
    candle_id UUID REFERENCES candles_weekly(id),
    result rule_result_type,    -- ENUM: PASS, FAIL
    confidence_score NUMERIC(5,4),
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(rule_id, candle_id)
)
```

#### Pattern Detection Tables (H&S System)

```sql
-- Detected patterns
pattern_detections (
    id UUID PRIMARY KEY,
    symbol TEXT,
    timeframe rule_timeframe,
    detected_candle_idx INT,
    
    -- Structure points
    left_shoulder_price NUMERIC(12,5),
    left_shoulder_idx INT,
    head_price NUMERIC(12,5),
    head_idx INT,
    right_shoulder_price NUMERIC(12,5),
    right_shoulder_idx INT,
    neckline_price NUMERIC(12,5),
    neckline_idx INT,
    
    -- Component scores (0.0-1.0)
    shoulder_symmetry NUMERIC(5,4),
    head_prominence NUMERIC(5,4),
    time_symmetry NUMERIC(5,4),
    volume_profile NUMERIC(5,4),
    neckline_quality NUMERIC(5,4),
    
    -- Context
    context_trend VARCHAR(20),
    volatility_regime VARCHAR(20),
    context_dist_ema200 NUMERIC(8,5),
    recent_swings INT,
    
    -- Confidence
    overall_confidence NUMERIC(6,4),
    final_confidence NUMERIC(6,4),
    pattern_type VARCHAR(10)      -- 'H&S', 'IHS', 'DOUBLE_TOP', 'DOUBLE_BOTTOM'
)

-- Trade signals generated from patterns
signals (
    id UUID PRIMARY KEY,
    pattern_detection_id UUID REFERENCES pattern_detections(id),
    symbol TEXT,
    timeframe rule_timeframe,
    direction trade_direction,    -- LONG, SHORT
    detected_price NUMERIC(12,5),
    theoretical_sl NUMERIC(12,5),
    theoretical_tp NUMERIC(12,5),
    projected_rr NUMERIC(5,2),
    confidence NUMERIC(6,4),
    status TEXT DEFAULT 'BACKTEST_ONLY'
)

-- Backtest results
backtest_trades (
    id UUID PRIMARY KEY,
    backtest_run_id UUID,
    pattern_detection_id UUID,
    signal_id UUID,
    entry_price, exit_price, stop_loss, take_profit,
    direction, result, reason,
    pnl, pnl_pips, pnl_r,
    confidence, risk_reward,
    entry_time, exit_time, bar_index, exit_bar
)
```

#### Trade Lifecycle Tables

```sql
-- Trade intent (IMMUTABLE after creation)
trades (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    account_id UUID REFERENCES accounts(id),
    candle_id UUID REFERENCES candles_weekly(id),
    symbol TEXT,
    timeframe TEXT,
    direction trade_direction,     -- LONG, SHORT
    planned_entry NUMERIC(12,5),
    stop_loss NUMERIC(12,5),
    take_profit NUMERIC(12,5),
    risk_percent NUMERIC(5,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
)

-- Trade execution events (APPEND-ONLY)
trade_executions (
    id UUID PRIMARY KEY,
    trade_id UUID REFERENCES trades(id),
    execution_type execution_type,  -- ENTRY_FILLED, PARTIAL_EXIT, TP_HIT, SL_HIT, MANUAL_CLOSE
    price NUMERIC(12,5),
    quantity NUMERIC(12,4),
    executed_at TIMESTAMPTZ DEFAULT NOW()
)

-- Trade feedback (behavioral tracking)
trade_feedback (
    id UUID PRIMARY KEY,
    trade_id UUID REFERENCES trades(id),
    emotion_before emotion_type,
    emotion_after emotion_type,
    plan_adherence NUMERIC(3,2),    -- 0.0 to 1.0
    notes TEXT
)
```

---

## 4. Pattern Detection System

**Location:** `backend/internal/patterns/`

### Supported Patterns

| Pattern | File | Direction |
|---------|------|-----------|
| Head & Shoulders (H&S) | `structure.go` | Bearish (SHORT) |
| Inverse H&S (IHS) | `structure.go` | Bullish (LONG) |
| Double Top | `double_patterns.go` | Bearish (SHORT) |
| Double Bottom | `double_patterns.go` | Bullish (LONG) |

### Core Files

| File | Purpose |
|------|---------|
| `types.go` | Data structures: `Candle`, `SwingPoint`, `DetectedStructure`, `TradeSignal`, `MarketContext` |
| `swing.go` | Swing detection: `FindSwingHighs()`, `FindSwingLows()`, `FindMinBetweenPeaks()`, `FindMaxBetweenTroughs()` |
| `structure.go` | H&S/IHS detection: `DetectHeadAndShoulders()`, `DetectInverseHeadAndShoulders()` |
| `double_patterns.go` | Double Top/Bottom: `DetectDoubleTop()`, `DetectDoubleBottom()` |
| `confidence.go` | Confidence scoring: `CalculateStructureConfidence()`, weighted component scoring |
| `context.go` | Market context: `DetermineMarketContext()`, `CalculateContextBonus()` |
| `signals.go` | Signal generation: `GenerateTradeSignal()`, entry/SL/TP calculation |
| `backtest.go` | Backtesting: `RunBacktest()`, `SimulateExit()`, `CalculateBacktestMetrics()` |

### Detection Algorithm (H&S Example)

```go
// 1. Find swing highs (pivot points)
highs := FindSwingHighs(candles, lookback)  // lookback varies by TF

// 2. Validate 3-peak structure
for each combination of (LS, Head, RS):
    - Head must be highest
    - Shoulder symmetry >= 80%
    - Find troughs between peaks (neckline points)
    - Pattern height >= 2% (meaningful size)

// 3. Calculate component scores
structure := DetectedStructure{
    ShoulderSymmetry: 1.0 - (|LS - RS| / max(LS, RS))
    HeadProminence:   (head - neckline) / head
    TimeSymmetry:     1.0 - (|distLS - distRS| / max)
    VolumeProfile:    volume analysis score
    NecklineQuality:  1.0 - (|trough1 - trough2| / neckline)
}

// 4. Calculate overall confidence (weighted average)
Weights: Symmetry=0.25, Head=0.20, Time=0.15, Volume=0.20, Neckline=0.20
```

### Market Context Adjustments

```go
type MarketContext struct {
    Trend              string   // STRONG_UP, WEAK_UP, SIDEWAYS, WEAK_DOWN, STRONG_DOWN
    VolatilityRegime   string   // COMPRESSION, EXPANSION, NORMAL
    DistanceFromEMA200 float64
    RecentSwings       int
}

// Context bonuses/penalties applied to confidence:
// - H&S in STRONG_UP trend: bonus (reversal more likely)
// - H&S in COMPRESSION: bonus (breakout setup)
// - IHS in STRONG_DOWN trend: bonus
```

### Signal Generation

```go
// H&S Signal (SHORT)
Entry:     Neckline - 10 pips (buffer)
StopLoss:  RightShoulder + 20 pips
TakeProfit: Neckline - PatternHeight

// IHS Signal (LONG)
Entry:     Neckline + 10 pips
StopLoss:  RightShoulder - 20 pips  
TakeProfit: Neckline + PatternHeight
```

### Backtesting System

```go
type BacktestConfig struct {
    WindowSize          int     // 30 candles
    ConfidenceThreshold float64 // 0.60 (60%)
    MinRR               float64 // 1.5:1
    MaxBarsToExit       int     // 20 bars
    CooldownBars        int     // 10 bars between trades
    SpreadPips          float64 // 0.2 (EURUSD)
    SlippagePips        float64 // 0.3
}

type BacktestMetrics struct {
    TotalPatterns, TradedPatterns, FilteredPatterns int
    WinRate, ProfitFactor, Expectancy float64
    AvgWin, AvgLoss, MaxDrawdown float64
    TotalPnL, TotalPnLPips, TotalPnLR float64
}
```

---

## 5. Confluence Scoring System

**Location:** `backend/internal/confluence/`

### Purpose
Multi-timeframe confluence scoring to filter pattern signals by overall market agreement.

### Confluence Checklist (Weighted)

| Timeframe | Component | Weight |
|-----------|-----------|--------|
| **Weekly (30.6%)** | Trend in Favor | 5.6% |
| | At/Rejected AOI | 5.6% |
| | Touching EMA200 | 2.8% |
| | Candlestick Rejection | 5.6% |
| | Structure Rejection | 5.6% |
| | H&S/IHS Pattern | 5.6% |
| **Daily (30.6%)** | Same components | Same weights |
| **4H (25%)** | Favor, EMA, Reject, Struct, Pattern | Proportional |
| **Lower TFs (13.9%)** | 2H Psychological Levels | 2.8% |
| | 1H Shift of Structure | 5.6% |
| | M30 Engulfing | 5.6% |

**Minimum Threshold:** 70% confluence required for live signals (50% for scanner alerts)

### Core Files

| File | Purpose |
|------|---------|
| `types.go` | `ChecklistScore`, `ConfluenceStream`, `ConfluenceUpdate` |
| `scorer.go` | `CalculateConfluence()`, component score calculations |
| `buffer.go` | Thread-safe `CandleBuffer` for live data |
| `integration.go` | `ConfluenceBacktester` for historical testing |

### Usage Flow

```go
scorer := NewConfluenceScorer(config)

score, signal := scorer.CalculateConfluence(
    weeklyCandles,
    dailyCandles,
    h4Candles,
    h2Candles,
    h1Candles,
    m30Candles,
    symbol,
)

if score.TotalPercent >= 70.0 && signal != nil {
    // Valid high-confluence signal
}
```

---

## 6. Rules Engine

**Location:** `backend/internal/rules/`

### Purpose
Deterministic PASS/FAIL evaluation of trading rules on candles.

### Core Principle
```
PURE FUNCTIONS. NO DATABASE. NO SIDE EFFECTS. DETERMINISTIC.
```

### Files

| File | Purpose |
|------|---------|
| `spec.go` | Rule registry and definitions |
| `conditions.go` | Individual condition evaluators |
| `evaluator.go` | Main `EvaluateRule()` function |
| `confidence.go` | Confidence scoring for rule results |
| `session.go` | Trading session utilities |

### Rule Structure

```go
type RuleSpec struct {
    Code        RuleCode
    Name        string
    Timeframe   string
    Conditions  []ConditionCode
    Description string
}

// Example: W1_TREND_BULLISH
// Conditions: EMA20 > EMA50, EMA50 > EMA200, Close > EMA20
```

### Evaluation Flow

```go
result, err := EvaluateRule("W1_TREND_BULLISH", candle, indicators)
// Returns: RuleResult{Result: "PASS", Confidence: 0.85, ConditionsMet: [...]}

results := EvaluateAllRules(candle, indicators)
// Returns map of all registered rules evaluated
```

---

## 7. Services & API Layer

### API Endpoints (Protected Routes)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/auth/signup` | POST | User registration |
| `/api/auth/login` | POST | User login (returns JWT) |
| `/api/accounts` | POST | Create trading account |
| `/api/candles/latest` | GET | Get latest candles |
| `/api/indicators/latest` | GET | Get latest indicators |
| `/api/indicators/compute` | POST | Compute indicators |
| `/api/trades` | POST | Create trade intent |
| `/api/trades/:id/execute` | POST | Record trade execution |
| `/api/trades/:id/close` | POST | Close trade |
| `/api/trades/:id/cancel` | POST | Cancel trade |
| `/api/trades/:id/state` | GET | Get trade state |
| `/api/trades/:id/feedback` | POST | Add trade feedback |
| `/api/analytics/summary` | GET | Performance summary |
| `/api/analytics/by-rule` | GET | Stats by rule |
| `/api/analytics/by-session` | GET | Stats by session |

### Service Layer Files

| File | Purpose |
|------|---------|
| `trade_service.go` | Trade creation & retrieval |
| `execution_service.go` | Trade execution lifecycle |
| `pattern_service.go` | Pattern detection orchestration |
| `risk_calculator.go` | Position sizing calculations |
| `rule_evaluation.go` | Rule evaluation orchestration |
| `auth/` | Authentication service |

---

## 8. Frontend Status

**Framework:** Next.js + TypeScript + Tailwind CSS

### Completed Pages

| Page | Status |
|------|--------|
| `/login` | ✅ Complete |
| `/signup` | ✅ Complete |
| `/dashboard` | ✅ UI Complete |

### Placeholder Pages

| Page | Status |
|------|--------|
| `/journal` | 🚧 Placeholder |
| `/profile` | 🚧 Placeholder |
| `/settings` | 🚧 Placeholder |

---

## 9. Current Completed Features

### ✅ Database & Data Pipeline
- [x] PostgreSQL schema with all migrations
- [x] Multi-timeframe candle storage (W1, D1, H4, H1)
- [x] Indicator computation pipeline (EMA 20/50/200)
- [x] SQLC code generation

### ✅ Pattern Detection
- [x] Swing high/low detection algorithm
- [x] Head & Shoulders detection
- [x] Inverse Head & Shoulders detection
- [x] Double Top detection
- [x] Double Bottom detection
- [x] Confidence scoring (geometric + context)
- [x] Signal generation (entry/SL/TP)
- [x] Backtesting framework with realistic costs

### ✅ Confluence System
- [x] Multi-timeframe confluence scorer
- [x] Weighted checklist calculation
- [x] Thread-safe candle buffers
- [x] Confluence backtester

### ✅ Rules Engine
- [x] Deterministic rule evaluation
- [x] Condition-based rule specs
- [x] Confidence scoring

### ✅ API & Auth
- [x] JWT authentication
- [x] Protected route middleware
- [x] CORS configuration
- [x] Trade lifecycle endpoints
- [x] Analytics endpoints

### ✅ Live Scanner
- [x] Real-time pattern scanner (`cmd/live_scanner/`)
- [x] Telegram integration for alerts
- [x] Multi-pair scanning (EURUSD, GBPUSD, USDJPY, USDCHF, AUDUSD)

### ✅ Other Tools
- [x] Paper trading simulation
- [x] Historical backtesting runner
- [x] Confluence simulation

---

## 10. Known Gaps & TODOs

### 🔴 Critical
- [ ] No symbol column in candle tables (currently EURUSD-only by design, but limits expansion)
- [ ] No live data feed integration (manual import only)
- [ ] Frontend trade journal not implemented

### 🟡 Important
- [ ] Pattern detection only in-memory (no automatic DB persistence)
- [ ] No MT5 execution integration (design decision - journal only)
- [ ] No multi-user support (single trader by design)
- [ ] No real-time WebSocket for frontend

### 🟢 Nice to Have
- [ ] Pattern visualization in frontend
- [ ] Confluence dashboard
- [ ] Historical pattern browser
- [ ] Export to CSV/Excel

---

## 11. Next Steps Placeholder

> **Add your next development priorities here:**

1. _[Your next feature]_
2. _[Your next feature]_
3. _[Your next feature]_

---

## Quick Reference Commands

### Start API Server
```bash
cd backend && go run cmd/api/main.go
```

### Run Live Scanner
```bash
cd backend && go run cmd/live_scanner/main.go
```

### Run Backtest
```bash
cd backend && go run cmd/backtest/main.go
```

### Import Candles
```bash
cd backend/scripts && python import_candles_multi_tf.py
```

### Start Frontend
```bash
cd frontend && npm run dev
```

---

## File Quick Reference

| Need | Go To |
|------|-------|
| Add new pattern | `backend/internal/patterns/` |
| Add new confluence check | `backend/internal/confluence/scorer.go` |
| Add new rule | `backend/internal/rules/spec.go` |
| Add new API endpoint | `backend/cmd/api/main.go` + `handlers/` |
| Modify DB schema | `backend/migrations/` + regenerate SQLC |
| Add new symbol | `backend/internal/constants/forex.go` |
