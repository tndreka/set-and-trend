# Backend Architecture

**Project:** Set The Trend  
**Last Updated:** January 15, 2026  
**Status:** Production Stable

---

## System Overview

Set The Trend is a Go-based backend system for deterministic trading journal and rule evaluation. The architecture follows clean separation of concerns with distinct layers for HTTP handling, business logic, data access, and persistence.

---

## Project Structure
```
backend/
├── cmd/
│   └── api/
│       └── main.go              # HTTP server entrypoint
├── internal/
│   ├── config/
│   │   ├── config.go            # Environment variable loading
│   │   └── database.go          # Database connection pooling
│   ├── constants/
│   │   └── forex.go             # EURUSD pip values, risk guards
│   ├── db/                      # SQLC generated queries (type-safe)
│   │   ├── accounts.sql.go
│   │   ├── candles.sql.go
│   │   ├── indicators.sql.go
│   │   ├── models.go
│   │   ├── querier.go
│   │   ├── rule_results.sql.go
│   │   ├── trades.sql.go
│   │   └── users.sql.go
│   ├── domain/                  # Business entities
│   │   ├── candle.go
│   │   ├── enums.go             # TradeBias, TradeResult, Session, Emotion
│   │   └── execution_events.go
│   ├── handlers/                # HTTP request handlers (Gin)
│   │   ├── accounts.go
│   │   ├── candles.go
│   │   ├── execution_handler.go
│   │   ├── indicators.go
│   │   ├── trade_handler.go
│   │   └── users.go
│   ├── repositories/            # Data access layer
│   │   ├── account_repository.go
│   │   ├── candle_repository.go
│   │   ├── execution_repository.go
│   │   ├── indicator_repository.go
│   │   ├── intent_repository.go
│   │   ├── rule_result_repository.go
│   │   ├── trade_repository.go
│   │   └── user_repository.go
│   ├── rules/                   # Rule evaluation engine
│   │   ├── conditions.go
│   │   ├── confidence.go
│   │   ├── evaluator.go
│   │   ├── session.go
│   │   └── spec.go
│   └── services/                # Business logic (pure functions)
│       ├── execution_mappers.go
│       ├── execution_service.go
│       ├── interfaces.go
│       ├── marketdata.go
│       ├── risk_calculator.go
│       ├── rule_evaluation.go
│       ├── trade_lifecycle.go
│       └── trade_service.go
├── migrations/
│   └── 000001_init_schema.up.sql
├── scripts/                     # Data processing utilities
│   ├── compute_all_emas/
│   ├── evaluate_all_rules/
│   └── import_csv/
├── sqlc.yaml                    # SQLC configuration
├── go.mod
└── go.sum
```

---

## Layer Responsibilities

### 1. HTTP Layer (handlers/)

**Responsibility:** Request validation, response formatting, HTTP-specific concerns.

**Rules:**
- No business logic
- No direct database access
- Validates request parameters
- Calls service layer for operations
- Returns JSON responses

**Example:**
```go
func (h *TradeHandler) CreateTrade(c *gin.Context) {
    var req CreateTradeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    trade, err := h.tradeService.CreateTrade(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{"status": "success", "data": trade})
}
```

---

### 2. Service Layer (services/)

**Responsibility:** Business logic, orchestration, validation.

**Rules:**
- No HTTP concerns
- No SQL (uses repositories)
- Pure functions where possible
- Deterministic behavior
- Comprehensive validation

**Example:**
```go
func (s *TradeService) CreateTrade(ctx context.Context, input CreateTradeInput) (*Trade, error) {
    // 1. Load dependencies
    account, err := s.accountRepo.GetAccountByID(ctx, input.AccountID)
    
    // 2. Validate business rules
    if err := ValidateTradeGeometry(input.PlannedEntry, input.PlannedSL, input.PlannedTP, input.Direction); err != nil {
        return nil, err
    }
    
    // 3. Compute derived values
    riskAmount, _ := ComputeRiskAmount(account.Balance, input.PlannedRiskPct)
    
    // 4. Persist
    return s.tradeRepo.CreateTrade(ctx, params)
}
```

---

### 3. Repository Layer (repositories/)

**Responsibility:** Data access, type conversion, query execution.

**Rules:**
- No business logic
- CRUD operations only
- Converts between DB types and domain types
- Wraps SQLC-generated queries

**Example:**
```go
func (r *CandleRepository) CreateCandle(ctx context.Context, params CandleCreateParams) (*Candle, error) {
    // Convert domain types to DB types
    openDec, _ := decimal.NewFromString(params.Open)
    
    // Execute SQLC query
    dbCandle, err := r.q.CreateCandle(ctx, db.CreateCandleParams{
        Open: openDec,
        // ...
    })
    
    // Convert DB types to domain types
    return &Candle{
        ID:   dbCandle.ID,
        Open: dbCandle.Open.String(),
        // ...
    }, nil
}
```

---

### 4. Database Layer (internal/db/)

**Responsibility:** Type-safe SQL query execution.

**Rules:**
- Generated by SQLC (DO NOT EDIT)
- Compile-time SQL validation
- Type-safe parameters
- No runtime SQL errors

**Configuration (sqlc.yaml):**
```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations/000001_init_schema.up.sql"
    queries: "internal/db/queries/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
```

---

## Ownership Matrix

| Responsibility | Owner | Input | Output | Trigger |
|----------------|-------|-------|--------|---------|
| **User Registration** | `services.AuthService` | Username, email, password | `users` row + JWT token | On-demand (POST /api/auth/signup) |
| **User Login** | `services.AuthService` | Username/email + password | JWT token | On-demand (POST /api/auth/login) |
| **Password Hashing** | `services.AuthService` | Plain password | bcrypt hash | Internal (during signup) |
| **Candle Ingestion** | `repositories.CandleRepository` | CSV/JSON OHLC data | `candles_weekly` row | On-demand (POST /api/candles) |
| **Indicator Computation** | `services.MarketData` | `candles_weekly` row | `indicators_weekly` row | On-demand (sync) |
| **Swing Detection** | `services.MarketData` | Recent candles array | Swing high/low prices | On-demand (internal) |
| **Rule Evaluation** | `services.RuleEvaluation` | Candle + Indicators | `rule_results` rows | On-demand (POST /api/rules/evaluate) |
| **Trade Creation** | `services.TradeService` | Trade parameters + account | `trades` row | On-demand (POST /api/trades) |
| **Trade Execution** | `services.ExecutionService` | Trade ID + actual prices | `trade_executions` row | On-demand (POST /api/trades/:id/execute) |
| **Trade Close** | `services.ExecutionService` | Trade ID + close price | `trade_executions` row | On-demand (POST /api/trades/:id/close) |
| **Analytics** | Direct SQL | user_id + filters | Aggregated metrics | On-demand (GET /api/analytics) |

---

## Execution Flow

### Flow 1: Candle Ingestion + Indicator Computation
```
POST /api/candles
    ↓
CandleHandler.CreateCandle()
    ↓
CandleRepository.CreateCandle()
    ↓
SQLC: INSERT INTO candles_weekly
    ↓
IndicatorService.ComputeIndicator()
    ↓
IndicatorRepository.CreateIndicator()
    ↓
SQLC: INSERT INTO indicators_weekly
```

**Characteristics:**
- Synchronous (no background jobs)
- Single database transaction
- Idempotent (can be re-run safely)

---

### Flow 2: Rule Evaluation
```
POST /api/rules/evaluate/{candle_id}
    ↓
RuleEvaluationService.EvaluateCandle()
    ↓
CandleRepository.GetCandleByID()
IndicatorRepository.GetIndicatorByCandleID()
    ↓
rules.EvaluateAllRules() [pure function]
    ↓
RuleResultRepository.CreateRuleResult() [for each rule]
    ↓
SQLC: INSERT INTO rule_results (ON CONFLICT DO NOTHING)
```

**Characteristics:**
- Deterministic (same input → same output)
- Idempotent (ON CONFLICT DO NOTHING)
- Results are immutable

---

### Flow 3: User Authentication
```
POST /api/auth/signup
    ↓
UserHandler.SignUp()
    ↓
AuthService.Signup()
    ↓
bcrypt.GenerateFromPassword() [hash password]
    ↓
Queries.CreateUser()
    ↓
SQLC: INSERT INTO users (username, email, password_hash, ...)
    ↓
GenerateToken() [create JWT]
    ↓
Return {token, user}

POST /api/auth/login
    ↓
UserHandler.Login()
    ↓
AuthService.Login()
    ↓
UserRepository.GetUserByUsername()
    ↓
SQLC: SELECT * FROM users WHERE username = $1
    ↓
bcrypt.CompareHashAndPassword() [verify password]
    ↓
UpdateUserLastLogin()
    ↓
GenerateToken() [create JWT]
    ↓
Return {token, user}
```

**Characteristics:**
- Password never stored in plaintext
- bcrypt cost factor: 10
- JWT expiration: 24 hours
- Support username OR email login
- Last login timestamp tracked

---

### Flow 4: Trade Lifecycle
```
POST /api/trades
    ↓
TradeService.CreateTrade()
    ↓
ValidateTradeGeometry()
ComputeRiskAmount()
ComputePositionSize()
    ↓
TradeRepository.CreateTrade()
    ↓
SQLC: INSERT INTO trades [immutable]

POST /api/trades/:id/execute
    ↓
ExecutionService.ExecuteTrade()
    ↓
ExecutionRepository.CreateExecution()
    ↓
SQLC: INSERT INTO trade_executions [append-only]

POST /api/trades/:id/close
    ↓
ExecutionService.CloseTrade()
    ↓
ComputePnL()
    ↓
ExecutionRepository.CreateExecution()
    ↓
SQLC: INSERT INTO trade_executions [append-only]
```

**Characteristics:**
- Immutable intent (`trades` table never updated)
- Append-only outcomes (`trade_executions` only INSERT)
- Complete audit trail
- State derived from execution history

---

## Data Flow Patterns

### Pattern 1: Immutable Intent

**Principle:** Trade plans are recorded once and never modified.

**Implementation:**
- `trades` table has no UPDATE operations
- Planned parameters frozen at creation
- Actual parameters recorded in separate execution events

**Benefits:**
- Complete audit trail
- Can prove what was planned vs. what happened
- Supports compliance requirements

---

### Pattern 2: Append-Only Events

**Principle:** Execution events are recorded, never deleted or updated.

**Implementation:**
- `trade_executions` table only allows INSERT
- Each execution creates a new row
- Trade state derived from execution history

**Benefits:**
- Event sourcing pattern
- Full replay capability
- No data loss from updates

---

### Pattern 3: Deterministic Computation

**Principle:** Same input always produces same output.

**Implementation:**
- Rules engine is pure functions
- No random values
- No time-based decisions (except timestamp recording)

**Benefits:**
- Testable
- Replayable
- Predictable

---

## Synchronous Operations

**Design Decision:** MVP uses 100% synchronous operations.

### No Background Jobs
- All operations happen in request context
- No queues, no workers, no cron jobs
- Simplifies debugging and deployment

### No Real-Time Updates
- Weekly candles only
- Manual trigger for rule evaluation
- No WebSocket, no polling

### No Caching
- Data is small (554 candles)
- Recompute on every request
- Database is fast enough

### Benefits
- Simpler architecture
- Easier to reason about
- No distributed systems complexity
- Sufficient for MVP scale

---

## Service Boundaries

### services.MarketData

**Responsibilities:**
- Compute indicators from candles (EMA, range, wicks)
- Detect swing highs/lows
- Candle structure analysis

**Key Functions:**
```go
ComputeBasicIndicators(candle Candle) WeeklyIndicators
```

---

### services.RuleEvaluation

**Responsibilities:**
- Orchestrate rule evaluation pipeline
- Load candle + indicator data
- Call pure rule evaluation functions
- Persist results

**Key Functions:**
```go
EvaluateCandle(ctx, candleID) error
```

---

### services.TradeService

**Responsibilities:**
- Validate trade geometry (entry/SL/TP relationships)
- Compute risk amounts
- Calculate position sizes
- Enforce risk limits
- Create trade plans

**Key Functions:**
```go
CreateTrade(ctx, input) (*Trade, error)
ValidateTradeGeometry(entry, sl, tp, bias) error
ComputeRiskAmount(balance, riskPct) (float64, error)
ComputePositionSize(risk, stopDistance, pipValue) (float64, error)
```

---

### services.ExecutionService

**Responsibilities:**
- Record execution events
- Validate trade state transitions
- Compute PnL
- Manage execution history

**Key Functions:**
```go
ExecuteTrade(ctx, input) error
CloseTrade(ctx, input) error
CancelTrade(ctx, input) error
GetTradeState(ctx, tradeID) (TradeState, error)
```

---

### rules (package)

**Responsibilities:**
- Pure rule evaluation logic
- Condition checking
- Confidence scoring
- Session derivation

**Key Functions:**
```go
EvaluateRule(ruleCode, candle, indicators) (RuleResult, error)
EvaluateAllRules(candle, indicators) map[RuleCode]RuleResult
```

**Characteristics:**
- Pure functions (no side effects)
- Deterministic (no randomness, no time dependency)
- No database access
- No HTTP concerns

---

## Database Patterns

### SQLC Integration

**Philosophy:** Write SQL, get type-safe Go code.

**Query Example:**
```sql
-- name: CreateCandle :one
INSERT INTO candles_weekly (
    id, timestamp_utc, open, high, low, close, volume
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;
```

**Generated Go:**
```go
type CreateCandleParams struct {
    ID           uuid.UUID
    TimestampUtc pgtype.Timestamptz
    Open         decimal.Decimal
    High         decimal.Decimal
    Low          decimal.Decimal
    Close        decimal.Decimal
    Volume       pgtype.Int8
}

func (q *Queries) CreateCandle(ctx context.Context, arg CreateCandleParams) (CandlesWeekly, error)
```

**Benefits:**
- Compile-time SQL validation
- No runtime SQL errors
- Type-safe parameters
- IDE autocomplete for database operations

---

### Connection Pooling

**Configuration (database.go):**
```go
poolConfig.MaxConns = 100
poolConfig.MinConns = 10
poolConfig.MaxConnLifetime = time.Hour
poolConfig.MaxConnIdleTime = 30 * time.Minute
poolConfig.HealthCheckPeriod = time.Minute
poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
```

**Rationale:**
- Max 100 connections supports high concurrency
- Min 10 connections reduces connection overhead
- Hour lifetime prevents stale connections
- 30-minute idle timeout reclaims resources

---

## Timestamp Architecture

**Added:** January 15, 2026  
**Status:** Production Standard

### Design Principle: Database-Owned System Time

All system-generated timestamps use PostgreSQL's `DEFAULT NOW()` constraint to ensure temporal consistency across distributed deployments.

### Implementation

#### System Timestamps (Database-Generated)

The following timestamps are automatically generated by PostgreSQL:

- `users.created_at`
- `accounts.updated_at`
- `candles_weekly.created_at`
- `indicators_weekly.computed_at`
- `rule_results.evaluated_at`
- `trades.created_at`
- `trade_executions.executed_at`

**Schema Pattern:**
```sql
CREATE TABLE example (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Application Code:**
```go
// Application does NOT pass timestamp
repo.Create(ctx, CreateParams{
    // No timestamp field needed
})
```

#### Market Timestamps (Application-Provided)

The following timestamp represents external market event time:

- `candles_weekly.timestamp_utc` - Market candle close time

**Rationale:** This timestamp represents when a market event occurred (candle close), not when the system processed it. Must be explicitly provided from external data source.

**Schema Pattern:**
```sql
CREATE TABLE candles_weekly (
    timestamp_utc TIMESTAMPTZ NOT NULL UNIQUE,  -- No DEFAULT
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- System time
);
```

**Application Code:**
```go
// Application explicitly provides market timestamp
repo.CreateCandle(ctx, CandleParams{
    TimestampUTC: marketCloseTime,  // From external source
    // created_at generated by database
})
```

### Architectural Benefits

#### 1. Distributed Systems Consistency
- Database server clock is the single source of truth
- No clock synchronization needed between application servers
- Horizontally scalable without timestamp coordination
- Load balancers can distribute requests to any app server

#### 2. Data Integrity Guarantees
- Zero-value timestamps (0001-01-01) are structurally impossible
- All events have authoritative timestamps
- Audit trail has guaranteed temporal ordering
- No manual timestamp bugs from application logic

#### 3. Operational Simplicity
- Application servers remain stateless regarding time
- No NTP configuration required for app servers
- Simplified debugging (all timestamps from same source)
- Clear ownership: DB = system time, App = domain time

#### 4. Compliance & Forensics
- Complete audit trail with reliable timestamps
- Temporal ordering is deterministic
- Historical replay is reliable
- Meets regulatory requirements for event logging

### Exception Handling

#### Historical Data Imports

For importing historical execution data with known timestamps (e.g., from broker CSV), create a separate query variant:
```sql
-- name: CreateExecutionWithTimestamp :one
INSERT INTO trade_executions (
    trade_id,
    execution_type,
    price,
    quantity,
    executed_at  -- Explicit timestamp
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;
```

**Usage:** Restrict to admin/import scripts only. Normal application flow uses the standard query without timestamp.

### Verification

Run this query to verify timestamp consistency:
```sql
SELECT
    c.timestamp_utc as market_time,
    c.created_at as system_time,
    i.computed_at as indicator_time
FROM candles_weekly c
JOIN indicators_weekly i ON i.candle_id = c.id
ORDER BY c.timestamp_utc DESC
LIMIT 5;
```

**Expected Result:**
- `market_time`: Historical dates (from external market data)
- `system_time`: Recent dates (when system processed the data)
- `indicator_time`: Recent dates (when indicators were computed)

### Timezone Handling

All timestamps are stored in UTC. User-specific timezone preferences:
- Stored in `accounts.timezone` (IANA format, e.g., "America/New_York")
- Applied only at presentation layer (frontend/API responses)
- Never stored in event data
- Never used in business logic calculations

### Migration Notes

When this pattern was enforced (January 15, 2026), the following changes were made:

1. Added `DEFAULT NOW()` to `trade_executions.executed_at`
2. Removed `ExecutedAt` field from `CreateExecutionParams` struct
3. Updated SQLC queries to exclude `executed_at` from INSERT statements
4. Removed all `time.Now()` calls from service layer persistence logic

See `TIMESTAMP_AUDIT.md` in project root for complete details.

---

## Testing Strategy

### Unit Tests
- Service layer functions (risk calculations, validation)
- Rules engine (condition evaluation, confidence scoring)
- Domain models (enum validation)

### Integration Tests
- Full data pipeline (import → indicators → rules)
- API endpoints (request → database → response)
- Transaction boundaries

### Test Data
- 554 weekly candles (2015-2025)
- Known EMA values for verification
- Rule results with expected PASS/FAIL outcomes

---

## Deployment Considerations

### Environment Variables
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=stt_user
DB_PASSWORD=secure_password
DB_NAME=set_the_trend
DB_SSLMODE=disable
PORT=8080
```

### Health Checks
```bash
GET /health
Response: {"status":"ok"}
```

### Logging
- Structured logging with zerolog
- Request ID tracking
- Error context preservation

### Monitoring
- Connection pool metrics
- Query performance
- Error rates
- Response times

---

## Future Enhancements

### Phase 2: Real-Time Integration
- WebSocket price feeds
- Automatic weekly candle aggregation
- Rule evaluation triggers on candle close
- Notification system

### Phase 3: Advanced Analytics
- Win rate correlation analysis
- R:R distribution
- Session-based performance
- Emotion vs. outcome correlation

### Phase 4: Multi-Timeframe
- Daily and 4H candles
- Multi-timeframe rule combinations
- Cross-timeframe confirmation

---

## References

- **Vision Document:** `docs/Vision.md`
- **MVP Scope:** `docs/MVP-scope.md`
- **Database Schema:** `docs/database-schema.sql`
- **Timestamp Audit:** `TIMESTAMP_AUDIT.md` (project root)
- **API Stability:** `BACKEND_FROZEN.md` (project root)
