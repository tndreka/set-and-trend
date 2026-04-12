-- Reverse the seed: drop the strategy_rules links and the H4 rule rows.
DELETE FROM strategy_rules
WHERE rule_id IN (
    SELECT id FROM rules
    WHERE code IN ('H4_TREND_FOLLOWING', 'H4_SUPPLY_DEMAND', 'H4_LONDON_BREAKOUT')
);

DELETE FROM rules
WHERE code IN ('H4_TREND_FOLLOWING', 'H4_SUPPLY_DEMAND', 'H4_LONDON_BREAKOUT');
