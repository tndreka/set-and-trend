-- Reverses 000013_restore_journal_schema.
-- Drops the trade-journal tables only. Candle tables are NEVER touched.
-- Dropped in reverse dependency order.

DROP TABLE IF EXISTS signals              CASCADE;
DROP TABLE IF EXISTS pattern_detections   CASCADE;
DROP TABLE IF EXISTS trade_feedback       CASCADE;
DROP TABLE IF EXISTS trade_intents        CASCADE;
DROP TABLE IF EXISTS trade_executions     CASCADE;
DROP TABLE IF EXISTS trades               CASCADE;
DROP TABLE IF EXISTS rule_results         CASCADE;
DROP TABLE IF EXISTS rules                CASCADE;
DROP TABLE IF EXISTS indicators_h1        CASCADE;
DROP TABLE IF EXISTS indicators_h4        CASCADE;
DROP TABLE IF EXISTS indicators_d1        CASCADE;
DROP TABLE IF EXISTS indicators_weekly    CASCADE;
DROP TABLE IF EXISTS accounts             CASCADE;
DROP TABLE IF EXISTS users                CASCADE;

DROP TYPE IF EXISTS emotion_type;
DROP TYPE IF EXISTS rule_timeframe;
DROP TYPE IF EXISTS rule_result_type;
DROP TYPE IF EXISTS execution_type;
DROP TYPE IF EXISTS trade_direction;
DROP TYPE IF EXISTS session_type;
DROP TYPE IF EXISTS account_type;
