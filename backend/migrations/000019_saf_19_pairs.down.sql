-- Revert: remove the 12 cross/metal pairs added in 000019.
-- Keeps the original 7 majors seeded by migrations 000017 and 000018.

DELETE FROM strategies
WHERE code IN ('SAF_CHECKLIST', 'SAF_W1_EMA_REJECTION', 'SAF_D1_REJECTION_STRUCTURE', 'SAF_W1_REJECTION_D1_AOI')
  AND symbol IN ('AUDCHF', 'CADJPY', 'CHFJPY', 'EURAUD', 'EURCAD', 'EURCHF', 'EURGBP', 'EURJPY', 'GBPCHF', 'GBPJPY', 'NZDJPY', 'XAUUSD');
