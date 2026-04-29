-- Extend SAF strategy coverage from 7 to 19 pairs.
-- Adds 12 pairs × 4 strategy codes = 48 rows.
-- New pairs: AUDCHF, CADJPY, CHFJPY, EURAUD, EURCAD, EURCHF, EURGBP, EURJPY, GBPCHF, GBPJPY, NZDJPY, XAUUSD.
-- Existing 7 (AUDUSD, EURUSD, GBPUSD, NZDUSD, USDCAD, USDCHF, USDJPY) untouched.
--
-- Pair selection sourced from results/trade_analysis.csv (the 19-pair backtest universe).
-- Requires SymbolRegistry in backend/internal/constants/forex.go to define each symbol;
-- otherwise live evaluation panics via MustGetSymbolConfig.

INSERT INTO strategies (code, name, symbol, timeframe, enabled, description) VALUES
  -- SAF_CHECKLIST (14-item composite across W1/D1/H4)
  ('SAF_CHECKLIST', 'Set & Forget Checklist (AUDCHF)', 'AUDCHF', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (CADJPY)', 'CADJPY', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (CHFJPY)', 'CHFJPY', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURAUD)', 'EURAUD', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURCAD)', 'EURCAD', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURCHF)', 'EURCHF', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURGBP)', 'EURGBP', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURJPY)', 'EURJPY', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (GBPCHF)', 'GBPCHF', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (GBPJPY)', 'GBPJPY', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (NZDJPY)', 'NZDJPY', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (XAUUSD)', 'XAUUSD', 'H4', TRUE, '14-item composite checklist across W1/D1/H4'),

  -- SAF_W1_EMA_REJECTION
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (AUDCHF)', 'AUDCHF', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (CADJPY)', 'CADJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (CHFJPY)', 'CHFJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURAUD)', 'EURAUD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURCAD)', 'EURCAD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURCHF)', 'EURCHF', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURGBP)', 'EURGBP', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (EURJPY)', 'EURJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (GBPCHF)', 'GBPCHF', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (GBPJPY)', 'GBPJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (NZDJPY)', 'NZDJPY', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),
  ('SAF_W1_EMA_REJECTION', 'W1 EMA Touch + Rejection (XAUUSD)', 'XAUUSD', 'H4', TRUE, 'w1_touching_ema + w1_candle_rejection'),

  -- SAF_D1_REJECTION_STRUCTURE
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (AUDCHF)', 'AUDCHF', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (CADJPY)', 'CADJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (CHFJPY)', 'CHFJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURAUD)', 'EURAUD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURCAD)', 'EURCAD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURCHF)', 'EURCHF', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURGBP)', 'EURGBP', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (EURJPY)', 'EURJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (GBPCHF)', 'GBPCHF', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (GBPJPY)', 'GBPJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (NZDJPY)', 'NZDJPY', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),
  ('SAF_D1_REJECTION_STRUCTURE', 'D1 Rejection + Structure (XAUUSD)', 'XAUUSD', 'H4', TRUE, 'd1_candle_rejection + d1_structure_rejection'),

  -- SAF_W1_REJECTION_D1_AOI
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (AUDCHF)', 'AUDCHF', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (CADJPY)', 'CADJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (CHFJPY)', 'CHFJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURAUD)', 'EURAUD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURCAD)', 'EURCAD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURCHF)', 'EURCHF', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURGBP)', 'EURGBP', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (EURJPY)', 'EURJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (GBPCHF)', 'GBPCHF', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (GBPJPY)', 'GBPJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (NZDJPY)', 'NZDJPY', 'H4', TRUE, 'w1_candle_rejection + d1_aoi'),
  ('SAF_W1_REJECTION_D1_AOI', 'W1 Rejection + D1 AOI (XAUUSD)', 'XAUUSD', 'H4', TRUE, 'w1_candle_rejection + d1_aoi')
ON CONFLICT DO NOTHING;
