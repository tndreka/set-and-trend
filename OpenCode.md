# OpenCode.md

## Build, Lint, Test Commands
```bash
# Build backend
cd backend && go build -o bin/api cmd/api/main.go

# Run all tests
cd backend && go test ./...

# Run single test file
cd backend && go test -v ./internal/rules/ -run TestSession

# Lint Go code
cd backend && go vet ./...
cd backend && golangci-lint run

# Generate SQLC queries
cd backend && sqlc generate

# Format Go code
cd backend && gofmt -w .
```

## Code Style Guidelines

### Go Language
- Follow standard `gofmt` formatting
- Use camelCase for variable names (Go convention)
- All errors must be handled explicitly, never ignored
- Use descriptive variable and function names
- Keep functions short and focused (single responsibility)
- Use pointer receivers for large structs

### Imports
- Group imports by standard library, external libraries, internal packages
- Use goimports to manage import ordering and formatting

### Types
- Use `github.com/shopspring/decimal.Decimal` for monetary values
- Use `github.com/google/uuid.UUID` for UUIDs
- Use `github.com/jackc/pgx/v5/pgtype.Timestamptz` for timestamps

### Naming Conventions
- Use `PascalCase` for exported types and methods
- Use `camelCase` for unexported types and methods
- Use `UPPER_SNAKE_CASE` for constants
- Use `go_snake_case` for database fields

### Error Handling
- All errors must be handled or explicitly logged
- Use `errors.Is()` and `errors.As()` for error type checking
- Prefer returning errors over panicking when possible

### Testing
- Write unit tests for all business logic
- Use table-driven tests for multiple test cases
- Include integration tests for database interactions
- Test edge cases and error conditions

### Code Organization
- Keep packages focused and cohesive
- Use clear directory structure: `internal/services`, `internal/repositories`, etc.
- Follow the dependency rule: inner packages should not depend on outer packages