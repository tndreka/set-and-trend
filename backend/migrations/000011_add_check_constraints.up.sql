-- ============================================
-- CHECK CONSTRAINTS (Issue #10 Fix)
-- ============================================

-- Accounts table constraints
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_leverage_sane 
    CHECK (leverage > 0 AND leverage <= 1000);
    
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_max_risk_sane 
    CHECK (max_risk_per_trade_pct > 0 AND max_risk_per_trade_pct <= 100);
    
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_daily_risk_sane 
    CHECK (max_daily_risk_pct > 0 AND max_daily_risk_pct <= 100);

-- Candles table constraints (ensure OHLC logic)
ALTER TABLE candles_weekly ADD CONSTRAINT chk_candles_ohlc_logic 
    CHECK (high >= low AND high >= open AND high >= close AND low <= open AND low <= close);

-- Indicators table constraints (EMAs must be positive when set)
ALTER TABLE indicators_weekly ADD CONSTRAINT chk_indicators_ema20_positive 
    CHECK (ema20 > 0);
    
ALTER TABLE indicators_weekly ADD CONSTRAINT chk_indicators_ema50_positive 
    CHECK (ema50 > 0);
    
ALTER TABLE indicators_weekly ADD CONSTRAINT chk_indicators_ema200_positive 
    CHECK (ema200 > 0);

-- Trades table constraints
ALTER TABLE trades ADD CONSTRAINT chk_trades_entry_positive 
    CHECK (planned_entry > 0);
    
ALTER TABLE trades ADD CONSTRAINT chk_trades_sl_positive 
    CHECK (stop_loss > 0);
    
ALTER TABLE trades ADD CONSTRAINT chk_trades_tp_positive 
    CHECK (take_profit > 0);
    
ALTER TABLE trades ADD CONSTRAINT chk_trades_risk_sane 
    CHECK (risk_percent > 0 AND risk_percent <= 100);
    
ALTER TABLE trades ADD CONSTRAINT chk_trades_sl_not_tp 
    CHECK (stop_loss != take_profit);

-- Trade executions constraints
ALTER TABLE trade_executions ADD CONSTRAINT chk_executions_price_positive 
    CHECK (price > 0);
    
ALTER TABLE trade_executions ADD CONSTRAINT chk_executions_quantity_positive 
    CHECK (quantity > 0);
    
ALTER TABLE trade_executions ADD CONSTRAINT chk_executions_not_future 
    CHECK (executed_at <= NOW() + INTERVAL '1 minute');

-- Rule results constraints
ALTER TABLE rule_results ADD CONSTRAINT chk_rule_results_confidence_sane 
    CHECK (confidence_score >= 0 AND confidence_score <= 1);