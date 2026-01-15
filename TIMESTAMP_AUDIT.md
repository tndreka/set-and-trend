# Timestamp Field Audit

**Last Updated:** January 15, 2026  
**Status:** COMPLETE ✅

## Design Principle: Database-Owned System Time

All system-generated timestamps use `DEFAULT NOW()` in PostgreSQL to ensure temporal consistency across distributed deployments.

## Timestamp Ownership Matrix

| Table | Column | Owner | Type | Status |
|-------|--------|-------|------|--------|
| users | created_at | Database | System | ✅ |
| accounts | updated_at | Database | System | ✅ |
| candles_weekly | timestamp_utc | Application | Market Data | ✅ |
| candles_weekly | created_at | Database | System | ✅ |
| indicators_weekly | computed_at | Database | System | ✅ |
| rules | created_at | Database | System | ✅ |
| rule_results | evaluated_at | Database | System | ✅ |
| trades | created_at | Database | System | ✅ |
| trade_executions | executed_at | Database | System | ✅ FIXED |

## Recent Changes (January 15, 2026)

### Problem Identified
The `trade_executions.executed_at` field lacked a database-level DEFAULT constraint, requiring application code to manually pass timestamps via `time.Now()`. This pattern introduces several risks:

- **Clock Skew:** Application server clocks may drift from database server time
- **Zero-Value Bugs:** Uninitialized Go `time.Time` values default to `0001-01-01`
- **Ambiguous Authority:** Unclear whether app or database is the source of truth
- **Non-Deterministic Ordering:** Events may appear out-of-order if app servers have different times

### Solution Implemented
Enforced database-level timestamp generation using PostgreSQL's `DEFAULT NOW()` constraint.

### Technical Implementation

#### 1. Schema Layer
```sql
-- migrations/000001_init_schema.up.sql
CREATE TABLE trade_executions (
    -- ... other fields
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 2. Query Layer
```sql
-- internal/db/queries/trades.sql
-- name: CreateTradeExecution :one
INSERT INTO trade_executions (
    trade_id,
    execution_type,
    price,
    quantity
    -- executed_at removed - handled by DB
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;
```

#### 3. Repository Layer
```go
// internal/repositories/execution_repository.go
type CreateExecutionParams struct {
    TradeID       uuid.UUID
    ExecutionType string
    Price         *float64
    Quantity      *float64
    // ExecutedAt removed - database generates timestamp
}
```

#### 4. Service Layer
```go
// internal/services/execution_service.go
execution, err := s.executionRepo.CreateExecutionTx(
    ctx,
    tx,
    repositories.CreateExecutionParams{
        TradeID:       tradeID,
        ExecutionType: eventType,
        Price:         &price,
        Quantity:      &positionSize,
        // No ExecutedAt field - database handles it
    },
)
```

#### 5. Configuration Layer
```go
// internal/config/database.go
import "net/url"

dsn := fmt.Sprintf(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    cfg.DBUser,
    url.QueryEscape(cfg.DBPassword), // URL-encode special characters
    cfg.DBHost,
    cfg.DBPort,
    cfg.DBName,
    cfg.DBSSLMode,
)
```

## Verification Process

### 1. Data Reconstruction
- Dropped and recreated schema with updated constraints
- Re-imported 554 weekly candles (2015-2025 dataset)
- Re-computed all EMAs (20/50/200 periods)
- Re-evaluated rule results (W1_TREND_BULLISH)

### 2. Timestamp Validation Query
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

### 3. Verification Results
```
market_time:          2025-08-10 (historical market data)
candle_inserted:      2026-01-15 (system ingestion time)
indicator_computed:   2026-01-15 (system computation time)
rule_evaluated:       2026-01-15 (system evaluation time)
```

**Outcome:** ✅ Market timestamps correctly represent historical events. System timestamps correctly represent current processing time.

## Architectural Benefits

### 1. Distributed Systems Consistency
- Database server clock is the single source of truth
- No clock synchronization needed between application servers
- Horizontally scalable without timestamp coordination

### 2. Data Integrity Guarantees
- Zero-value timestamps are structurally impossible (database constraint)
- Audit trail has guaranteed temporal ordering
- No manual timestamp bugs from application logic

### 3. Operational Simplicity
- Application servers remain stateless regarding time
- No NTP configuration required for app servers
- Works seamlessly with load balancers and auto-scaling

### 4. Debugging & Forensics
- All events have authoritative timestamps
- Temporal ordering is deterministic
- Historical replay is reliable

## Exception: Market Data Timestamps

The `candles_weekly.timestamp_utc` field is intentionally **application-controlled** because it represents external market event times, not system processing times.
```sql
-- Market data timestamp is explicit
INSERT INTO candles_weekly (timestamp_utc, ...) 
VALUES ('2025-08-10 00:00:00+00', ...);
```

This distinction is critical for analytics that correlate system events with market conditions.

## Future Considerations

### Timezone Handling
All timestamps are stored in UTC. User-specific timezone preferences are:
- Stored in `accounts.timezone` (IANA format)
- Applied only at presentation layer (frontend)
- Never stored in event data

### Historical Data Imports
For importing historical execution data with known timestamps:
1. Create a separate `CreateExecutionWithTimestamp` query
2. Restrict usage to admin/import scripts only
3. Document that this bypasses normal timestamp authority

### Monitoring
Consider adding database-level monitoring for:
- Timestamp gaps (e.g., no events for >24 hours)
- Future timestamps (clock skew detection)
- Timezone inconsistencies
