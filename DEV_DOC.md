

---

## 2026-01-15 - Timestamp Architecture Refactoring

### Overview
Refactored timestamp generation to enforce database-owned temporal authority for all system events. This eliminates application-level `time.Now()` usage in persistence operations.

### Problem Analysis

**Before:**
- `trade_executions.executed_at` had no DEFAULT constraint
- Application passed timestamps via `time.Now()`
- Potential for clock skew across distributed app servers
- Risk of zero-value timestamps (0001-01-01) from uninitialized structs

**Impact:**
- Non-deterministic event ordering in multi-server deployments
- Debugging complexity (which clock is authoritative?)
- Horizontal scaling concerns (clock coordination needed)

### Solution Design

**After:**
- All system timestamps use PostgreSQL `DEFAULT NOW()`
- Application no longer provides timestamps for system events
- Database server clock is single source of truth
- Zero-value timestamps are structurally impossible

### Implementation Details

#### Schema Changes
```sql
-- File: migrations/000001_init_schema.up.sql
-- Change:
executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

#### Query Changes
```sql
-- File: internal/db/queries/trades.sql
-- Removed executed_at from INSERT column list
-- Reduced from 5 parameters to 4
INSERT INTO trade_executions (
    trade_id, execution_type, price, quantity
) VALUES ($1, $2, $3, $4)
RETURNING *;
```

#### Repository Changes
```go
// File: internal/repositories/execution_repository.go
// Removed ExecutedAt field from params struct
type CreateExecutionParams struct {
    TradeID       uuid.UUID
    ExecutionType string
    Price         *float64
    Quantity      *float64
    // ExecutedAt removed
}
```

#### Service Changes
```go
// File: internal/services/execution_service.go
// Removed time.Now() calls
repositories.CreateExecutionParams{
    TradeID:       tradeID,
    ExecutionType: eventType,
    Price:         &price,
    Quantity:      &positionSize,
    // No ExecutedAt field
}
```

#### Configuration Changes
```go
// File: internal/config/database.go
// Fixed password URL encoding for special characters
import "net/url"

dsn := fmt.Sprintf(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    cfg.DBUser,
    url.QueryEscape(cfg.DBPassword), // Encode @ and $
    cfg.DBHost,
    cfg.DBPort,
    cfg.DBName,
    cfg.DBSSLMode,
)
```

### Verification Process

#### Step 1: Schema Rebuild
```bash
psql -h localhost -U stt_user -d set_the_trend \
  -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

psql -h localhost -U stt_user -d set_the_trend \
  < migrations/000001_init_schema.up.sql
```

**Verification:**
```bash
psql -h localhost -U stt_user -d set_the_trend -c "\d trade_executions"
# Expected: executed_at shows "DEFAULT now()"
```

#### Step 2: Data Pipeline Re-execution
```bash
# Import 554 weekly candles (2015-2025)
cd scripts/import_csv && go run main.go

# Compute EMAs (20/50/200)
cd ../compute_all_emas && go run main.go

# Evaluate rules (W1_TREND_BULLISH)
cd ../evaluate_all_rules && go run main.go
```

**Results:**
- 554 candles imported ✅
- 554 indicators computed ✅
- 554 rule results generated ✅
- 0 compilation errors ✅

#### Step 3: Timestamp Validation
```sql
SELECT
    c.timestamp_utc as market_time,
    c.created_at as candle_inserted,
    i.computed_at as indicator_computed,
    rr.evaluated_at as rule_evaluated
FROM candles_weekly c
JOIN indicators_weekly i ON i.candle_id = c.id
LEFT JOIN rule_results rr ON rr.candle_id = c.id
ORDER BY c.timestamp_utc DESC
LIMIT 5;
```

**Expected Pattern:**
- `market_time`: 2025-08-10 (historical)
- `candle_inserted`: 2026-01-15 (today)
- `indicator_computed`: 2026-01-15 (today)
- `rule_evaluated`: 2026-01-15 (today)

**Result:** ✅ All system timestamps reflect current processing time, market timestamps reflect historical event time

### Technical Metrics

| Metric | Value |
|--------|-------|
| Files Modified | 5 |
| Lines Changed | ~50 |
| Candles Re-imported | 554 |
| Indicators Recomputed | 554 |
| Rules Re-evaluated | 554 |
| Compilation Errors | 0 |
| Test Failures | 0 |

### Lessons Learned

#### 1. Password URL Encoding
PostgreSQL connection strings in URL format require special character encoding:
- `@` symbol → `%40`
- `$` symbol → `%24`
- Solution: `url.QueryEscape()` in Go

#### 2. SQLC Syntax Strictness
SQLC parser enforces strict SQL syntax:
- Trailing commas in column lists cause parse errors
- Must regenerate after every query file change
- Type mismatches are caught at compile time (beneficial)

#### 3. Transaction Method Parity
Repository methods with transaction variants must both be updated:
- `CreateExecution(ctx, params)`
- `CreateExecutionTx(ctx, tx, params)`

#### 4. Full Pipeline Re-verification
Schema changes require complete data pipeline re-run:
1. Import source data (candles)
2. Compute derived data (indicators)
3. Execute business logic (rules)
4. Verify with SQL queries

### Documentation Created

1. **TIMESTAMP_AUDIT.md** - Timestamp ownership matrix and rationale
2. **BACKEND_FROZEN.md** - API stability contract for frontend development
3. **Updated docs/backend_architecture.md** - Added timestamp architecture section
4. **Updated README.md** - Added architectural highlights

### Status

**Backend:** Production Stable ✅  
**API:** Frozen for Frontend Development  
**Data:** 554 candles with indicators and rules  
**Next Phase:** Frontend Dashboard Implementation
