# Timestamp Field Audit - COMPLETED ✅

## Decision: Postgres Owns System Time

Date: January 15, 2026

All system-generated timestamps use `DEFAULT NOW()` in Postgres.

| Table | Column | Owner | Type | Status |
|-------|--------|-------|------|--------|
| users | created_at | Postgres | System | ✅ |
| accounts | updated_at | Postgres | System | ✅ |
| candles_weekly | timestamp_utc | App | Market | ✅ |
| candles_weekly | created_at | Postgres | System | ✅ |
| indicators_weekly | computed_at | Postgres | System | ✅ |
| rules | created_at | Postgres | System | ✅ |
| rule_results | evaluated_at | Postgres | System | ✅ |
| trades | created_at | Postgres | System | ✅ |
| trade_executions | executed_at | Postgres | System | ✅ FIXED TODAY |

## Changes Made Today (Jan 15, 2026)

### Problem
`trade_executions.executed_at` had no database DEFAULT, forcing application code to pass `time.Now()`. This creates:
- Clock skew in distributed systems
- Zero-value timestamp bugs (0001-01-01)
- Unclear timestamp authority
- Non-deterministic event ordering

### Solution
Enforced architectural rule: **Postgres owns all system timestamps**

### Files Changed
1. `migrations/000001_init_schema.up.sql` - Added `DEFAULT NOW()` to executed_at
2. `internal/db/queries/trades.sql` - Removed executed_at parameter
3. `internal/repositories/execution_repository.go` - Removed ExecutedAt from params
4. `internal/services/execution_service.go` - Removed time.Now() calls
5. `internal/config/database.go` - Fixed password URL encoding with url.QueryEscape()

## Verification Query
```sql
SELECT
    c.timestamp_utc as candle_time,
    c.created_at as candle_inserted,
    i.computed_at as indicator_computed,
    rr.evaluated_at as rule_evaluated
FROM candles_weekly c
JOIN indicators_weekly i ON i.candle_id = c.id
LEFT JOIN rule_results rr ON rr.candle_id = c.id
ORDER BY c.timestamp_utc DESC
LIMIT 5;
```

**Results (Jan 15, 2026):**
- Market time: 2025-08-10 (historical) ✅
- System time: 2026-01-15 (today) ✅
- All timestamps logical and ordered ✅

## Interview Talking Point

"I enforce strict boundaries: the database owns system time, the application owns business logic. This eliminates clock skew in distributed systems, provides a single source of truth for event ordering, and makes the system horizontally scalable. The only exception is market data timestamps, which represent external event times and must be explicitly set from the data source."

## Why This Matters

1. **Distributed Systems** - No clock skew between app servers
2. **Data Integrity** - Timestamps are authoritative and immutable
3. **Horizontal Scaling** - App servers are stateless
4. **Audit Trail** - Complete temporal ordering guaranteed
5. **Bug Prevention** - No zero-value timestamps possible
