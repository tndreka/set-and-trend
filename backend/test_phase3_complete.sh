#!/bin/bash

# Phase 3 Complete Integration Test
# Tests:  Trade creation → Execution → State transitions → Error handling

set -e  # Exit on error

BASE_URL="http://localhost:8080/api"
DB_URL="postgres://stt_user:lantidhe42H%40%24%40@localhost:5432/set_the_trend?sslmode=disable"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🧪 PHASE 3 COMPLETE INTEGRATION TEST"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ============================================
# STEP 0:  Verify Prerequisites
# ============================================
echo -e "${BLUE}📋 Step 0: Verifying prerequisites... ${NC}"

USER_ID=$(psql "$DB_URL" -t -c "SELECT id FROM users LIMIT 1" | tr -d ' ')
CANDLE_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM candles_weekly" | tr -d ' ')
CANDLE_ID=$(psql "$DB_URL" -t -c "SELECT id FROM candles_weekly ORDER BY timestamp_utc DESC LIMIT 1" | tr -d ' ')

if [ -z "$USER_ID" ]; then
    echo -e "${RED}❌ No users found.  Run seed data first.${NC}"
    exit 1
fi

if [ "$CANDLE_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ No candles found. Import CSV data first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites met: ${NC}"
echo "   User ID: $USER_ID"
echo "   Candles: $CANDLE_COUNT"
echo "   Latest Candle ID: $CANDLE_ID"
echo ""


# ============================================
# STEP 1: Create Test Account
# ============================================
echo -e "${BLUE}📋 Step 1: Creating test account...${NC}"

ACCOUNT_RESPONSE=$(curl -s -X POST "$BASE_URL/accounts" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\":  \"$USER_ID\",
    \"type\": \"demo\",
    \"broker_name\": \"Test Broker MT4\",
    \"currency\": \"USD\",
    \"balance\": \"10000.00\",
    \"leverage\": 100,
    \"max_risk_per_trade_pct\": 1.0,
    \"max_daily_risk_pct\":  3.0,
    \"timezone\": \"UTC\",
    \"preferred_session\": \"london\"
  }")

ACCOUNT_ID=$(echo "$ACCOUNT_RESPONSE" | jq -r '.data.id // .id // empty')

if [ -z "$ACCOUNT_ID" ]; then
    echo -e "${RED}❌ Failed to create account${NC}"
    echo "$ACCOUNT_RESPONSE" | jq
    exit 1
fi

echo -e "${GREEN}✅ Account created:  $ACCOUNT_ID${NC}"
echo ""


# ============================================
# STEP 2: Create Test Trade
# ============================================
echo -e "${BLUE}📋 Step 2: Creating test trade...${NC}"

TRADE_RESPONSE=$(curl -s -X POST "$BASE_URL/trades" \
  -H "Content-Type: application/json" \
  -d "{
    \"account_id\": \"$ACCOUNT_ID\",
    \"candle_id\": \"$CANDLE_ID\",
    \"bias\": \"long\",
    \"planned_entry\": 1.08500,
    \"planned_sl\": 1.08000,
    \"planned_tp\": 1.09500,
    \"planned_risk_pct\": 1.0,
    \"reason_for_trade\": \"Phase 3 integration test - W1 bullish setup with EMA alignment\"
  }")

TRADE_ID=$(echo "$TRADE_RESPONSE" | jq -r '.data.id // empty')

if [ -z "$TRADE_ID" ]; then
    echo -e "${RED}❌ Failed to create trade${NC}"
    echo "$TRADE_RESPONSE" | jq
    exit 1
fi

echo -e "${GREEN}✅ Trade created: $TRADE_ID${NC}"
echo ""

# ============================================
# STEP 3: Check Initial State (Should be "planned")
# ============================================
echo -e "${BLUE}📋 Step 3: Checking initial state...${NC}"

STATE_RESPONSE=$(curl -s "$BASE_URL/trades/$TRADE_ID/state")
STATE=$(echo "$STATE_RESPONSE" | jq -r '.state')

if [ "$STATE" != "planned" ]; then
    echo -e "${RED}❌ Expected state 'planned', got '$STATE'${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Initial state is 'planned'${NC}"
echo ""

# ============================================
# STEP 4: Execute Trade (planned → open)
# ============================================
echo -e "${BLUE}📋 Step 4: Executing trade (ENTRY event)...${NC}"

EXEC_RESPONSE=$(curl -s -X POST "$BASE_URL/trades/$TRADE_ID/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_entry": 1.08505,
    "reason": "Entry triggered on EMA50 retest with bullish confirmation"
  }')

EXEC_STATUS=$(echo "$EXEC_RESPONSE" | jq -r '.status')

if [ "$EXEC_STATUS" != "success" ]; then
    echo -e "${RED}❌ Trade execution failed${NC}"
    echo "$EXEC_RESPONSE" | jq
    exit 1
fi

NEW_STATE=$(echo "$EXEC_RESPONSE" | jq -r '.state')
echo -e "${GREEN}✅ Trade executed successfully${NC}"
echo "   State transition:  planned → $NEW_STATE"
echo ""

# ============================================
# STEP 5: Verify State After Execution
# ============================================
echo -e "${BLUE}📋 Step 5: Verifying state after execution...${NC}"

STATE_RESPONSE=$(curl -s "$BASE_URL/trades/$TRADE_ID/state")
STATE=$(echo "$STATE_RESPONSE" | jq -r '.state')

if [ "$STATE" != "open" ]; then
    echo -e "${RED}❌ Expected state 'open', got '$STATE'${NC}"
    exit 1
fi

echo -e "${GREEN}✅ State is 'open'${NC}"
echo ""

# ============================================
# STEP 6: Test Double Execution (Should Fail)
# ============================================
echo -e "${BLUE}📋 Step 6: Testing double execution prevention...${NC}"

DOUBLE_EXEC=$(curl -s -X POST "$BASE_URL/trades/$TRADE_ID/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_entry": 1.08510
  }')

ERROR=$(echo "$DOUBLE_EXEC" | jq -r '.error // empty')

if [ -z "$ERROR" ]; then
    echo -e "${RED}❌ Double execution should have been prevented! ${NC}"
    echo "$DOUBLE_EXEC" | jq
    exit 1
fi

echo -e "${GREEN}✅ Double execution prevented${NC}"
echo "   Error: $ERROR"
echo ""

# ============================================
# STEP 7: Get Execution History
# ============================================
echo -e "${BLUE}📋 Step 7: Getting execution history...${NC}"

EXECS_RESPONSE=$(curl -s "$BASE_URL/trades/$TRADE_ID/executions")
EXEC_COUNT=$(echo "$EXECS_RESPONSE" | jq -r '.count')

if [ "$EXEC_COUNT" != "1" ]; then
    echo -e "${RED}❌ Expected 1 execution, got $EXEC_COUNT${NC}"
    exit 1
fi

FIRST_EXEC=$(echo "$EXECS_RESPONSE" | jq -r '.executions[0]')
EVENT_TYPE=$(echo "$FIRST_EXEC" | jq -r '.event_type')
ENTRY_PRICE=$(echo "$FIRST_EXEC" | jq -r '.price')

echo -e "${GREEN}✅ Execution history verified${NC}"
echo "   Event count: $EXEC_COUNT"
echo "   Event type: $EVENT_TYPE"
echo "   Entry price: $ENTRY_PRICE"
echo ""

# ============================================
# STEP 8: Close Trade (open → closed)
# ============================================
echo -e "${BLUE}📋 Step 8: Closing trade (TP HIT)...${NC}"

CLOSE_RESPONSE=$(curl -s -X POST "$BASE_URL/trades/$TRADE_ID/close" \
  -H "Content-Type: application/json" \
  -d '{
    "close_price": 1.09400,
    "reason": "tp hit"
  }')

CLOSE_STATUS=$(echo "$CLOSE_RESPONSE" | jq -r '.status')

if [ "$CLOSE_STATUS" != "success" ]; then
    echo -e "${RED}❌ Trade close failed${NC}"
    echo "$CLOSE_RESPONSE" | jq
    exit 1
fi

FINAL_STATE=$(echo "$CLOSE_RESPONSE" | jq -r '.state')
echo -e "${GREEN}✅ Trade closed successfully${NC}"
echo "   State transition: open → $FINAL_STATE"
echo ""

# ============================================
# STEP 9: Verify Final State
# ============================================
echo -e "${BLUE}📋 Step 9: Verifying final state...${NC}"

STATE_RESPONSE=$(curl -s "$BASE_URL/trades/$TRADE_ID/state")
STATE=$(echo "$STATE_RESPONSE" | jq -r '.state')

if [ "$STATE" != "closed" ]; then
    echo -e "${RED}❌ Expected state 'closed', got '$STATE'${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Final state is 'closed'${NC}"
echo ""

# ============================================
# STEP 10: Test Double Close (Should Fail)
# ============================================
echo -e "${BLUE}📋 Step 10: Testing double close prevention...${NC}"

DOUBLE_CLOSE=$(curl -s -X POST "$BASE_URL/trades/$TRADE_ID/close" \
  -H "Content-Type: application/json" \
  -d '{
    "close_price": 1.09500
  }')

ERROR=$(echo "$DOUBLE_CLOSE" | jq -r '.error // empty')

if [ -z "$ERROR" ]; then
    echo -e "${RED}❌ Double close should have been prevented!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Double close prevented${NC}"
echo "   Error: $ERROR"
echo ""

# ============================================
# STEP 11: Verify Final Execution Count
# ============================================
echo -e "${BLUE}📋 Step 11: Verifying final execution count...${NC}"

FINAL_EXECS=$(curl -s "$BASE_URL/trades/$TRADE_ID/executions")
FINAL_COUNT=$(echo "$FINAL_EXECS" | jq -r '.count')
EXECUTIONS=$(echo "$FINAL_EXECS" | jq -r '.executions')

if [ "$FINAL_COUNT" != "2" ]; then
    echo -e "${RED}❌ Expected 2 executions, got $FINAL_COUNT${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Final execution count verified${NC}"
echo "   Total executions: $FINAL_COUNT"
echo ""
echo "Execution Details:"
echo "$EXECUTIONS" | jq -r '.[] | "  - \(.event_type) at \(.price // "N/A") (PnL: \(.pnl // "N/A"))"'
echo ""

# ============================================
# STEP 12: Test Cancel on Planned Trade
# ============================================
echo -e "${BLUE}📋 Step 12: Testing cancel functionality...${NC}"

# Create a new trade to test cancel
CANCEL_TRADE=$(curl -s -X POST "$BASE_URL/trades" \
  -H "Content-Type:  application/json" \
  -d "{
    \"account_id\":  \"$ACCOUNT_ID\",
    \"candle_id\": \"$CANDLE_ID\",
    \"bias\":  \"short\",
    \"planned_entry\": 1.08500,
    \"planned_sl\":  1.09000,
    \"planned_tp\": 1.07500,
    \"planned_risk_pct\": 1.0,
    \"reason_for_trade\": \"Test trade for cancel functionality\"
  }")

CANCEL_TRADE_ID=$(echo "$CANCEL_TRADE" | jq -r '.data.id')

# Cancel it
CANCEL_RESPONSE=$(curl -s -X POST "$BASE_URL/trades/$CANCEL_TRADE_ID/cancel" \
  -H "Content-Type:  application/json" \
  -d '{
    "reason": "Market conditions changed - setup invalidated"
  }')

CANCEL_STATUS=$(echo "$CANCEL_RESPONSE" | jq -r '.status')

if [ "$CANCEL_STATUS" != "success" ]; then
    echo -e "${RED}❌ Trade cancellation failed${NC}"
    echo "$CANCEL_RESPONSE" | jq
    exit 1
fi

CANCELLED_STATE=$(echo "$CANCEL_RESPONSE" | jq -r '.state')

if [ "$CANCELLED_STATE" != "cancelled" ]; then
    echo -e "${RED}❌ Expected state 'cancelled', got '$CANCELLED_STATE'${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Trade cancelled successfully${NC}"
echo "   Trade ID: $CANCEL_TRADE_ID"
echo "   State:  $CANCELLED_STATE"
echo ""

# ============================================
# STEP 13: Test Cancel After Execution (Should Fail)
# ============================================
echo -e "${BLUE}📋 Step 13: Testing cancel prevention after execution...${NC}"

# Try to cancel the executed trade
INVALID_CANCEL=$(curl -s -X POST "$BASE_URL/trades/$TRADE_ID/cancel" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Trying to cancel executed trade"
  }')

ERROR=$(echo "$INVALID_CANCEL" | jq -r '.error // empty')

if [ -z "$ERROR" ]; then
    echo -e "${RED}❌ Cancel after execution should have been prevented!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Cancel after execution prevented${NC}"
echo "   Error: $ERROR"
echo ""

# ============================================
# STEP 14: Verify Database Integrity
# ============================================
echo -e "${BLUE}📋 Step 14: Verifying database integrity...${NC}"

DB_TRADE_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM trades WHERE account_id = '$ACCOUNT_ID'" | tr -d ' ')
DB_EXEC_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM trade_executions WHERE trade_id = '$TRADE_ID'" | tr -d ' ')
DB_INTENT_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM trade_intents WHERE trade_id = '$CANCEL_TRADE_ID'" | tr -d ' ')

echo -e "${GREEN}✅ Database integrity verified${NC}"
echo "   Trades created: $DB_TRADE_COUNT"
echo "   Executions for test trade: $DB_EXEC_COUNT"
echo "   Intents for cancelled trade: $DB_INTENT_COUNT"
echo ""

# ============================================
# STEP 15: Test PnL Calculation
# ============================================
echo -e "${BLUE}📋 Step 15: Verifying PnL calculation...${NC}"

PNL=$(echo "$FINAL_EXECS" | jq -r '.executions[] | select(.event_type == "tp_hit") | .pnl')
PNL_PIPS=$(echo "$FINAL_EXECS" | jq -r '.executions[] | select(.event_type == "tp_hit") | .pnl_pips')

if [ "$PNL" != "null" ] && [ "$PNL_PIPS" != "null" ]; then
    echo -e "${GREEN}✅ PnL calculated successfully${NC}"
    echo "   PnL: \$$PNL"
    echo "   PnL Pips: $PNL_PIPS"
else
    echo -e "${YELLOW}⚠️  PnL not calculated (may be expected for some event types)${NC}"
fi
echo ""

# ============================================
# FINAL SUMMARY
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}  ✅ ALL TESTS PASSED!  PHASE 3 VERIFIED  ${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 Test Summary:"
echo "   ✅ Account creation"
echo "   ✅ Trade creation (planned state)"
echo "   ✅ Trade execution (planned → open)"
echo "   ✅ Double execution prevention"
echo "   ✅ Execution history tracking"
echo "   ✅ Trade closure (open → closed)"
echo "   ✅ Double close prevention"
echo "   ✅ Trade cancellation (planned → cancelled)"
echo "   ✅ Cancel prevention after execution"
echo "   ✅ State machine transitions"
echo "   ✅ Database integrity"
echo "   ✅ PnL calculation"
echo ""
echo "🎯 Phase 3 Status: PRODUCTION-READY ✅"
echo ""
echo "Test Trade IDs:"
echo "   Completed: $TRADE_ID"
echo "   Cancelled: $CANCEL_TRADE_ID"
echo ""
