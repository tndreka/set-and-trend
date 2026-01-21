

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

---

## 2026-01-20 - User Authentication & Frontend Integration

### Overview
Implemented complete user authentication system with JWT tokens and bcrypt password hashing. Integrated authentication with frontend signup/login pages. Redesigned dashboard with vertical navigation and consistent dark theme.

### Backend Implementation

#### Database Migration
**File:** `backend/migrations/000006_add_user_auth_fields.up.sql`

**Added Fields:**
- `username` VARCHAR(50) UNIQUE NOT NULL
- `email` VARCHAR(255) UNIQUE NOT NULL
- `password_hash` VARCHAR(255) NOT NULL
- `name`, `surname` VARCHAR(100)
- `is_email_verified` BOOLEAN DEFAULT false
- `email_verification_token` VARCHAR(255)
- `password_reset_token` VARCHAR(255)
- `password_reset_expires` TIMESTAMPTZ
- `last_login` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ DEFAULT NOW()

**Indexes Created:**
- idx_users_username
- idx_users_email
- idx_users_email_verification_token
- idx_users_password_reset_token

#### Authentication Service
**Files:** `backend/internal/services/auth/auth_service.go`, `jwt.go`

**Features:**
- User signup with validation
- bcrypt password hashing (cost: 10)
- JWT token generation (24h expiration)
- User login with credential verification
- Support for username OR email login

**Key Functions:**
```go
Signup(ctx, username, email, password, name, surname) (*CreateUserRow, error)
Login(ctx, usernameOrEmail, password) (string, error)
GenerateToken(userID, username, secret) (string, error)
ValidateToken(tokenString, secret) (*Claims, error)
```

#### API Endpoints
**File:** `backend/cmd/api/main.go`

**New Routes:**
- POST `/api/auth/signup` - User registration
- POST `/api/auth/login` - User authentication

**Request/Response:**
```json
// Signup Request
{
  "username": "string",
  "email": "string",
  "password": "string",
  "name": "string",
  "surname": "string"
}

// Login Request
{
  "username_or_email": "string",
  "password": "string"
}

// Response (both)
{
  "token": "jwt_token_string",
  "user": {
    "id": "uuid",
    "username": "string",
    "email": "string",
    "name": "string",
    "surname": "string"
  }
}
```

#### Repository Updates
**File:** `backend/internal/repositories/user_repository.go`

**New Methods:**
- `GetUserByUsername()` - Returns user with password_hash
- `GetUserByEmail()` - Returns user with password_hash
- `GetUserByID()` - Returns user profile (no password)

#### Query Layer
**File:** `backend/internal/db/queries/users.sql`

**New Queries:**
- CreateUser with auth fields
- GetUserByUsername
- GetUserByEmail
- GetUserByID
- UpdateUserLastLogin
- SetEmailVerificationToken
- VerifyEmail
- SetPasswordResetToken
- ResetPassword
- UpdateUserProfile

**SQLC Config Update:**
```yaml
schema: 
  - "migrations/000001_init_schema.up.sql"
  - "migrations/000006_add_user_auth_fields.up.sql"
```

### Frontend Implementation

#### Authentication Pages
**Files:** `frontend/app/signup/page.tsx`, `frontend/app/login/page.tsx`

**Features:**
- Form validation for all fields
- API integration with error handling
- Loading states during submission
- JWT token storage in localStorage
- Redirect to dashboard on success
- Error message display to user

**Signup Form Fields:**
- Name, Surname, Username, Email, Password
- Terms & Conditions checkbox

**Login Form Fields:**
- Username or Email
- Password
- Remember me checkbox
- Forgot password link

#### API Client
**File:** `frontend/lib/api/client.ts`

**New Methods:**
```typescript
interface SignupRequest {
  username: string;
  email: string;
  password: string;
  name?: string;
  surname?: string;
}

interface LoginRequest {
  username_or_email: string;
  password: string;
}

apiClient.signup(data: SignupRequest)
apiClient.login(data: LoginRequest)
```

#### Dashboard Redesign
**File:** `frontend/app/dashboard/page.tsx`

**Major Changes:**
- Applied gray-950 background (matching landing page)
- Added background grid pattern and green glow effects
- Implemented fixed vertical sidebar navigation
- Section-based UI (Markets/Journal/Profile/Settings)
- Active state indicators with green accent
- Gradient card styling for all components

**Navigation Structure:**
- Markets (Active - shows chart and indicators)
- Journal (Placeholder)
- Profile (Placeholder)
- Settings (Placeholder)

#### UI Theme Updates
**Files Modified:**
- `frontend/app/page.tsx` - Lightened to gray-950
- `frontend/app/login/page.tsx` - Dark theme applied
- `frontend/app/signup/page.tsx` - Dark theme applied
- `frontend/components/ui/LoadingScreen.tsx` - Themed animation

**Color Palette:**
- Background: `bg-gray-950`
- Accent: `text-green-400` / `bg-green-500`
- Borders: `border-white/10`
- Cards: `from-gray-900 to-black` gradient

### Technical Metrics

| Metric | Value |
|--------|-------|
| Files Created | 17 |
| Files Modified | 15 |
| Lines Added | ~10,958 |
| Lines Removed | ~83 |
| New Dependencies | 3 (bcrypt, JWT, axios) |
| New Migrations | 1 (000006) |
| New API Endpoints | 2 |
| SQL Queries Added | 10 |
| Frontend Pages | 3 |

### Security Implementation

**Password Security:**
- bcrypt hashing with cost factor 10
- Automatic salt generation
- Only hash stored, never plaintext
- Constant-time comparison

**JWT Security:**
- Secret stored in environment variable
- 24-hour expiration
- Claims: userID and username only
- Client-side localStorage storage

**API Security:**
- CORS configured for frontend domain
- Input validation on all fields
- Generic error messages (prevent user enumeration)

### Verification Results

**Test User Creation:**
```bash
curl -X POST http://164.92.229.200:8080/api/auth/signup \
  -d '{"username":"test","email":"test@example.com","password":"pass123"}'
```
✅ User created with bcrypt hash
✅ JWT token returned
✅ User data stored in database

**Test Login:**
```bash
curl -X POST http://164.92.229.200:8080/api/auth/login \
  -d '{"username_or_email":"test","password":"pass123"}'
```
✅ Password validated
✅ JWT token returned
✅ Last login timestamp updated

### Known Limitations

**Implemented:**
- ✅ Signup and login
- ✅ JWT token generation
- ✅ Password hashing
- ✅ Token storage

**Not Implemented (Future):**
- ❌ Logout functionality
- ❌ Token refresh mechanism
- ❌ Email verification flow
- ❌ Password reset flow
- ❌ Protected route middleware
- ❌ Rate limiting

### Lessons Learned

1. **Field Naming:** Backend expects `username_or_email`, frontend must match exactly
2. **URL Encoding:** Special characters in passwords need `url.QueryEscape()`
3. **SQLC Config:** Must include all migrations in schema array
4. **Theme Consistency:** Define color palette once, reuse everywhere

### Status

**Backend:** Authentication Complete ✅  
**Frontend:** Auth UI Complete ✅  
**Integration:** Working End-to-End ✅  
**Next Phase:** Protected Routes & Trade Journal
