-- =====================================================================
-- 000015_seed_h4_rules_and_strategy_links
-- =====================================================================
-- Seeds the three H4 rule rows that the multi-strategy lab evaluates,
-- and links each strategy to its rule via the strategy_rules M2M from
-- migration 000014. Idempotent via ON CONFLICT.
-- =====================================================================

INSERT INTO rules (id, code, name, timeframe, description) VALUES
    (gen_random_uuid(), 'H4_TREND_FOLLOWING', 'H4 Trend Following', 'H4',
     'EMA50>EMA200, close above EMA50, EMA50 rising. Long on prior swing low.'),
    (gen_random_uuid(), 'H4_SUPPLY_DEMAND', 'H4 Supply & Demand', 'H4',
     'Bullish rejection at active H4 demand zone. SL below zone, TP at nearest supply.'),
    (gen_random_uuid(), 'H4_LONDON_BREAKOUT', 'H4 London Breakout', 'H4',
     'London-open H4 bar breaks Asian session range. SL on opposite boundary, TP 1.5x range.')
ON CONFLICT (code) DO NOTHING;

-- Link each strategy to its rule. Resolved by code → uuid lookups so this
-- migration is safe regardless of the seed order on a fresh DB.
INSERT INTO strategy_rules (strategy_id, rule_id)
SELECT s.id, r.id
FROM strategies s
JOIN rules r ON r.code = s.code
WHERE s.code IN ('H4_TREND_FOLLOWING', 'H4_SUPPLY_DEMAND', 'H4_LONDON_BREAKOUT')
ON CONFLICT DO NOTHING;
