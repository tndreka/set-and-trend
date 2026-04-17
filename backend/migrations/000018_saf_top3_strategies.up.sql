-- Seed the top-3 backtested confluences as separate strategies.
-- These fire on exactly 2 specific items (high-confidence, fewer but better signals).
-- SAF_CHECKLIST remains enabled for broader score-based detection.

INSERT INTO strategies (code, name, symbol, timeframe, enabled, description) VALUES
  -- W1 EMA + Rejection (54.4% WR, +1.36R, 2.4 trades/yr)
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (AUDUSD)', 'AUDUSD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURUSD)', 'EURUSD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (GBPUSD)', 'GBPUSD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (NZDUSD)', 'NZDUSD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (USDCAD)', 'USDCAD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (USDCHF)', 'USDCHF', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (USDJPY)', 'USDJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),

  -- D1 Rejection + Structure (44.1% WR, +1.07R, 3.9 trades/yr)
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (AUDUSD)', 'AUDUSD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURUSD)', 'EURUSD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (GBPUSD)', 'GBPUSD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (NZDUSD)', 'NZDUSD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (USDCAD)', 'USDCAD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (USDCHF)', 'USDCHF', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (USDJPY)', 'USDJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),

  -- W1 Rejection + D1 AOI (36.6% WR, +0.68R, 7.6 trades/yr)
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (AUDUSD)', 'AUDUSD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURUSD)', 'EURUSD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (GBPUSD)', 'GBPUSD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (NZDUSD)', 'NZDUSD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (USDCAD)', 'USDCAD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (USDCHF)', 'USDCHF', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (USDJPY)', 'USDJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi')
ON CONFLICT DO NOTHING;
