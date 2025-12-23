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

