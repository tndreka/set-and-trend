# Set The Trend

**A deterministic trading journal and rule engine for disciplined discretionary traders.**

---

## Overview

Set The Trend is a backend system that transforms subjective trading decisions into objective, queryable data. Built for traders who execute rule-based strategies on weekly timeframes and need a deterministic audit trail of their decision-making process.

### Core Capabilities

- **Rule Engine:** Deterministic PASS/FAIL evaluation of trading rules on weekly candles
- **Trade Lifecycle:** Immutable trade plans with append-only execution log
- **Indicator Pipeline:** Automated EMA computation on historical and new price data
- **Analytics Foundation:** Pure SQL analytics for win rate, R:R, and performance metrics

---

## System Status

**Last Updated:** January 20, 2026

| Component | Status |
|-----------|--------|
| Backend API | ✅ Stable |
| Database | ✅ Seeded (554 candles) |
| Indicators | ✅ Computed (EMA 20/50/200) |
| Rules | ✅ Evaluated (W1_TREND_BULLISH) |
| Authentication | ✅ Complete (JWT + bcrypt) |
| Frontend - Auth | ✅ Complete (signup/login) |
| Frontend - Dashboard | ✅ Complete (UI only) |
| Frontend - Journal | 🚧 Placeholder |
| Frontend - Profile | 🚧 Placeholder |
| Frontend - Settings | 🚧 Placeholder |

---

## Architecture Highlights

### 1. Database-Owned Temporal Authority

All system event timestamps use PostgreSQL's `DEFAULT NOW()` constraint.
```sql
CREATE TABLE trade_executions (
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Benefits:**
- Eliminates clock skew in distributed deployments
- Single source of temporal truth
- Horizontally scalable without coordination
- Structurally prevents zero-value timestamp bugs

---

### 2. Immutable Intent Pattern

Trade plans are stored once and never updated. Actual outcomes are recorded in an append-only execution log.
```
trades table           → What was planned (immutable)
trade_executions table → What actually happened (append-only)
```

**Benefits:**
- Complete audit trail for compliance
- Supports historical replay and backtesting
- No data loss from UPDATE operations
- Provable intent vs. outcome analysis

---

### 3. Deterministic Rule Evaluation

Rules produce identical results for identical inputs, regardless of evaluation time.
```
Same candle data + Same indicators = Same rule result
```

**Benefits:**
- Historical analytics remain valid
- System behavior is replayable
- Backtesting is reliable
- Debugging is straightforward

---

## Tech Stack

### Backend
- **Language:** Go 1.23
- **Web Framework:** Gin
- **Database:** PostgreSQL 16
- **Query Layer:** SQLC (compile-time SQL validation)
- **Connection Pool:** pgx/v5

### Frontend (In Development)
- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript
- **Styling:** TailwindCSS
- **State Management:** React Query
- **Charts:** Recharts

---

## Project Structure
```
set-and-trend/
├── backend/
│   ├── cmd/api/              # HTTP server entrypoint
│   ├── internal/
│   │   ├── config/           # Environment configuration
│   │   ├── db/               # SQLC-generated queries
│   │   ├── domain/           # Business entities (Candle, Trade, Rule)
│   │   ├── handlers/         # HTTP request handlers
│   │   ├── repositories/     # Data access layer
│   │   ├── rules/            # Rule evaluation engine
│   │   └── services/         # Business logic
│   ├── migrations/           # SQL schema definitions
│   └── scripts/              # Data import/processing utilities
├── docs/
│   ├── Vision.md             # Product vision and problem statement
│   ├── MVP-scope.md          # MVP boundaries and constraints
│   ├── backend_architecture.md
│   └── database-schema.sql
├── TIMESTAMP_AUDIT.md        # Timestamp architecture decisions
├── BACKEND_FROZEN.md         # API stability contract
└── README.md                 # This file
```

---

## Getting Started

### Prerequisites

- Go 1.23 or higher
- PostgreSQL 16 or higher
- SQLC 1.30+ (for query generation)
- Node.js 20+ (for frontend)

### Backend Setup

#### 1. Database Initialization
```bash
# Create database and user
psql -U postgres << 'EOF'
CREATE DATABASE set_the_trend;
CREATE USER stt_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE set_the_trend TO stt_user;
\q
EOF
```

#### 2. Environment Configuration
```bash
cd backend
cat > .env << 'EOF'
DB_HOST=localhost
DB_PORT=5432
DB_USER=stt_user
DB_PASSWORD=secure_password
DB_NAME=set_the_trend
DB_SSLMODE=disable
PORT=8080
EOF
```

#### 3. Schema Migration
```bash
psql -h localhost -U stt_user -d set_the_trend < migrations/000001_init_schema.up.sql
```

#### 4. Data Pipeline
```bash
# Import historical candles
cd scripts/import_csv
go run main.go

# Compute EMAs
cd ../compute_all_emas
go run main.go

# Evaluate rules
cd ../evaluate_all_rules
go run main.go
```

#### 5. Start Server
```bash
cd ../../
go run cmd/api/main.go
```

Server will be available at `http://localhost:8080`

### Verification
```bash
# Health check
curl http://localhost:8080/health

# Fetch latest candles
curl http://localhost:8080/api/candles/latest?limit=5

# Fetch latest indicators
curl http://localhost:8080/api/indicators/latest?limit=5
```

---

## API Documentation

### Authentication

**POST** `/api/auth/signup`

Creates a new user account.

**Request Body:**
```json
{
  "username": "trader1",
  "email": "trader@example.com",
  "password": "SecurePass123!",
  "name": "John",
  "surname": "Doe"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "trader1",
    "email": "trader@example.com",
    "name": "John",
    "surname": "Doe"
  }
}
```

---

**POST** `/api/auth/login`

Authenticates a user and returns JWT token.

**Request Body:**
```json
{
  "username_or_email": "trader1",
  "password": "SecurePass123!"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "trader1",
    "email": "trader@example.com",
    "name": "John",
    "surname": "Doe"
  }
}
```

---

### Candles

**GET** `/api/candles/latest?limit=N`

Returns the N most recent weekly candles with OHLC data.

**Response:**
```json
{
  "status": "success",
  "data": [
    {
      "id": "uuid",
      "timestamp_utc": "2025-08-10T00:00:00Z",
      "open": "1.09500",
      "high": "1.09850",
      "low": "1.09200",
      "close": "1.09650",
      "volume": 125000
    }
  ]
}
```

---

### Indicators

**GET** `/api/indicators/latest?limit=N`

Returns the N most recent indicator records joined with candle data.

**Response:**
```json
{
  "status": "success",
  "data": [
    {
      "id": "uuid",
      "candle_id": "uuid",
      "ema20": "1.09600",
      "ema50": "1.09400",
      "ema200": "1.08900",
      "timestamp_utc": "2025-08-10T00:00:00Z",
      "close": "1.09650"
    }
  ]
}
```

---

### Trades

**POST** `/api/trades`

Creates a new trade plan.

**Request Body:**
```json
{
  "account_id": "uuid",
  "candle_id": "uuid",
  "direction": "LONG",
  "planned_entry": 1.0950,
  "planned_sl": 1.0900,
  "planned_tp": 1.1050,
  "planned_risk_pct": 1.0
}
```

---

**POST** `/api/trades/:id/execute`

Records trade entry execution.

**Request Body:**
```json
{
  "actual_entry": 1.0952,
  "reason": "Entry filled with 2 pip slippage"
}
```

---

**POST** `/api/trades/:id/close`

Records trade exit execution.

**Request Body:**
```json
{
  "close_price": 1.1048,
  "reason": "Manual close near TP"
}
```

---

**GET** `/api/trades/:id/state`

Returns current trade state (planned/open/partial/closed/cancelled).

---

**GET** `/api/trades/:id/executions`

Returns all execution events for the trade in chronological order.

---

## Key Architectural Decisions

### Why Postgres-Owned Timestamps?

**Problem:** Application server clocks can skew across a distributed fleet.

**Solution:** Delegate timestamp generation to the database layer.

**Implementation:**
```sql
executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

**Result:**
- Single authoritative time source
- Horizontally scalable without clock coordination
- No zero-value timestamp bugs
- Simplified debugging (all events have DB-generated time)

---

### Why Immutable Trade Plans?

**Problem:** Updating trade records destroys historical intent.

**Solution:** Write-once trade plans + append-only execution log.

**Implementation:**
- `trades` table: Never updated after creation
- `trade_executions` table: Only INSERT operations

**Result:**
- Complete audit trail
- Can prove planned vs. actual behavior
- Supports compliance requirements
- Enables replay for backtesting

---

### Why SQLC Instead of ORM?

**Problem:** ORMs abstract SQL, hide performance issues, and generate runtime errors.

**Solution:** SQLC generates type-safe Go code from SQL queries at compile time.

**Benefits:**
- SQL is first-class (write optimal queries)
- Type safety (compile-time validation)
- No runtime surprises (schema mismatches caught at build)
- Performance visibility (see exact queries executed)

---

## Development Roadmap

### Phase 1: Backend Foundation ✅ COMPLETE
- Database schema with 8 normalized tables
- Rule evaluation engine with deterministic logic
- Trade lifecycle management (plan → execute → close)
- Historical data pipeline (554 weekly candles)
- EMA computation (20/50/200 periods)

### Phase 2: Frontend Dashboard 🚧 IN PROGRESS
- Interactive candlestick chart with EMA overlays
- Rule results visualization (PASS/FAIL indicators)
- Trade planning interface
- Execution tracking
- Basic analytics dashboard

### Phase 3: Live Market Integration 📅 PLANNED
- WebSocket price feeds from market data provider
- Automatic weekly candle aggregation
- Rule evaluation triggers on candle close
- Email/SMS notification system
- Signal generation based on rule results

### Phase 4: Advanced Analytics 📅 PLANNED
- Win rate correlation by rule combination
- R:R distribution analysis
- Session-based performance metrics
- Emotion vs. outcome analysis
- Slippage tracking and statistics

### Phase 5: AI-Powered Insights 📅 FUTURE
- LLM-based trade analysis
- Pattern detection in trade history
- Personalized mistake identification
- Adaptive rule optimization suggestions

---

## Testing

### Unit Tests
```bash
cd backend
go test ./internal/services/... -v
go test ./internal/rules/... -v
```

### Integration Tests
```bash
# Verify full data pipeline
cd scripts/import_csv && go run main.go
cd ../compute_all_emas && go run main.go
cd ../evaluate_all_rules && go run main.go

# Verify API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/candles/latest?limit=1
```

---

## Documentation

- **[Vision & Problem Statement](docs/Vision.md)** - Product vision and target user
- **[MVP Scope](docs/MVP-scope.md)** - Phase 1 boundaries and constraints
- **[Backend Architecture](docs/backend_architecture.md)** - System design and ownership
- **[Database Schema](docs/database-schema.sql)** - SQL DDL with comments
- **[Timestamp Audit](TIMESTAMP_AUDIT.md)** - Timestamp ownership decisions
- **[Backend Freeze Notice](BACKEND_FROZEN.md)** - API stability contract

---

## Contributing

This project is currently in active development for production use. Contributions are welcome after the MVP is complete.

### Code Style
- Go: Follow standard `gofmt` formatting
- SQL: Use lowercase keywords, snake_case identifiers
- Commits: Conventional commits format

### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Update documentation
6. Submit PR with clear description

---

## License

MIT License - See LICENSE file for details.

---

## Contact

For questions, bug reports, or feature requests:
- **Issues:** [GitHub Issues](https://github.com/tndreka/set-and-trend/issues)
- **Documentation:** See `docs/` directory


---

**Built with discipline. Tested with rigor. Deployed with confidence.**

