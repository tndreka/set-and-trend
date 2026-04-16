BEGIN;

-- Disable the old 3 strategies (keep rows for reference).
UPDATE strategies SET enabled = FALSE
WHERE code IN ('H4_TREND_FOLLOWING', 'H4_SUPPLY_DEMAND', 'H4_LONDON_BREAKOUT');

-- Insert SAF_CHECKLIST for all 7 traded pairs.
INSERT INTO strategies (code, name, symbol, timeframe, enabled, description)
VALUES
  ('SAF_CHECKLIST', 'Set & Forget Checklist (AUDUSD)', 'AUDUSD', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (EURUSD)', 'EURUSD', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (GBPUSD)', 'GBPUSD', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (NZDUSD)', 'NZDUSD', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (USDCAD)', 'USDCAD', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (USDCHF)', 'USDCHF', 'H4', TRUE, '20-item composite checklist across W1/D1/H4'),
  ('SAF_CHECKLIST', 'Set & Forget Checklist (USDJPY)', 'USDJPY', 'H4', TRUE, '20-item composite checklist across W1/D1/H4')
ON CONFLICT DO NOTHING;

COMMIT;
