# Set The Trend - Development Log

## 2025-12-23 - Day 1: Foundation

### Environment Setup

Git repo created set_the_trend

created docs/
docs/
├── 00-vision.md → Problem + success criteria
├── 01-mvp-scope.md → 8 tables, 7 endpoints only
├── 02-backend-architecture.md → Go structure + ownership
└── 03-database-schema.sql → Postgres DDL

### Postgress + Migrations

Install Postgres (Ubuntu/Debian)
	-sudo apt update
	-sudo apt install postgresql postgresql-contrib

Start service
	-sudo systemctl start postgresql
	-sudo systemctl enable postgresql

Connect as superuser
	-sudo -u postgres psql

Create app user + DB
Commands:
	CREATE USER stt_user WITH PASSWORD 'yourpass';
	CREATE DATABASE set_the_trend;
	GRANT ALL PRIVILEGES ON DATABASE set_the_trend TO stt_user;
	GRANT ALL ON SCHEMA public TO stt_user;
	ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO stt_user;
	\q

Migrations

	cd backend/migrations
	migrate create -ext sql -dir . -seq init_schema
Pasted docs/03-database-schema.sql → 000001_init_schema.up.sql

Run migration
	-migrate -path . -database "postgres://stt_user:yourpass@localhost:5432/set_the_trend?sslmode=disable" up


### Next Steps : TO DO
	
1) internal/domain/enums.go → Go types matching Postgres enums

2) internal/constants/forex.go → EURUSD pip value, θ=0.3

3) services/marketdata.go → EMA + swing calculations

4) sqlc generate → DB queries → Go structs

5) cmd/api/main.go → Gin server



### Day 2 & 3 [services + test]

 internal/domain/enums.go → package domain (was "domanin")
 internal/services/marketdata.go → Removed unused imports
 Deleted marketdata.gox (backup file)
 go test ./... → PASS (0.002s)


### Domain Layer ✅ Fixed
internal/domain/
├── candle.go (253B) ← Candle struct
└── enums.go (1191B) ← TradeBias, TradeResult, Session, Emotion


### Constants Layer ✅ Live
internal/constants/
└── forex.go (272B) ← EURUSD pip value, timeframes, risk guards


---

## 🎯 CURRENT STATUS (All Green)

✅ SQLC layer (Day 2) → 5 generated files, type-safe
✅ Services layer (Day 3) → marketdata.go + tests PASSING
✅ Domain layer → enums + candle structs
✅ Constants → EURUSD pip math
✅ go test ./... → All packages compile + tests pass
✅ Module: set-and-trend/backend (go1.23.4 toolchain)


### File Structure
backend/
├── cmd/api/main.go ← Server entry (minimal)
├── internal/
│ ├── constants/forex.go  EURUSD pip value
│ ├── db/  SQLC generated (5 files)
│ ├── domain/  Candle + enums
│ ├── services/marketdata.go  Indicators + tests ✅
│ ├── repositories/ → Day 4
│ └── handlers/ → Day 4
├── migrations/schema.sql 3 tables ready
└── sqlc.yaml  SQLC config


## 2025-12-24 - Day 4: PRODUCTION SQLC API LIVE 

### SQLC + Gin API Deployed to VPS
✅ VPS: 164.92.229.200:8080 ← LIVE WORLDWIDE
✅ .env → ?? → PostgreSQL 
✅ /api/users POST → SQLC → New DB rows 
✅ curl /health → {"db":"connected"}
✅ psql → COUNT(*) = 2 (sample + API user)
✅ 8-table schema fully integrated


### Key Files Created
internal/config/
├── config.go (Load .env → DB_PASSWORD=lantidhe42@$)
└── database.go (pgxpool → SQLC Queries)

internal/repositories/
└── user_repository.go (pgtype.Timestamptz → time.Time)

internal/handlers/
└── users.go (Gin → Repository → SQLC)

cmd/api/main.go (Gin server + .env config)


### SQLC Generation (6 files, 22KB total)
internal/db/
├── accounts.sql.go (3391B) ← Matches account_type enum
├── candles.sql.go (2236B) ← candles_weekly table
├── db.go (564B)
├── models.go (12853B) ← 8-table structs
├── querier.go (743B)
└── users.sql.go (1238B)


### Production Tests PASSED
curl http://localhost:8080/health → {"status":"ok","db":"connected"}
curl POST /api/users → {"id":"50a69af6-d69f-4dbc-a556-62a352d6dd1e"}
psql → SELECT COUNT(*) FROM users; → 2 rows

🏆 Day 4 COMPLETE: First SQLC endpoint LIVE on VPS
🏆 164.92.229.200:8080 → Accessible worldwide
🏆 .env → PostgreSQL → SQLC → Gin → JSON response
🏆 2 rows verified in production DB
