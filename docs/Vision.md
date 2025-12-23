Set The Trend — Vision

Problem

Traders follow subjective rules inconsistently and have no deterministic way to verify whether losses come from market conditions or rule violations, especially on higher timeframes like the weekly chart.
There is no single backend source of truth that ties:

    numeric rules and market context

    planned trade parameters

    actual executions and outcomes

    behavioral feedback

Because of this, most “journaling” is shallow, stats are misleading, and traders cannot systematically refine a small rule-based playbook over time.
What this product does NOT do

    It does not execute trades or connect to brokers.

    It does not predict price or “call” the market.

    It does not send signals, alerts, or Telegram messages.

    It does not use machine learning or “AI auto-trading” in the MVP.

    It does not support every market and timeframe at once (starts with EURUSD, Weekly only).

    It is not multi-user (single trader only).

    It is not real-time execution dependent (end-of-candle, end-of-trade data is enough).

Target user

A single discretionary trader (you) trading a narrow, rule-based playbook on higher timeframes, who:

    is willing to log trades honestly (plan, execution, and emotions)

    wants numeric rules instead of screenshots and vague “setups”

    cares more about long-term edge and discipline than about automation or signals

Without honest, structured input from the trader, the system fails and its analytics are meaningless.
One-sentence product definition

Set The Trend is a focused trading journal and rule engine for a single discretionary trader, turning clear numeric rules and structured post-trade feedback into real, testable edge without any automation hype.
Success criteria (MVP, backend-only)

The MVP is “done” only when all of the following are true:

    Weekly  candles can be ingested, stored in Postgres, and queried deterministically for a given week.

    EMA and basic structure rules (trend, pullback, rejection) evaluate deterministically per weekly candle, with stored PASS/FAIL results.

    Each trade stores an immutable snapshot of key account fields at setup time (balance, leverage, risk limits, timezone).

    Each trade stores both planned and actual parameters (entry, SL, TP, risk, size, outcome) in a way that is queryable by SQL without extra processing.

    Basic outcome metrics (win rate, average R:R, max consecutive wins/losses) can be computed directly from the database using pure SQL.

    Behavior fields (plan-following and emotions) are stored per trade and can be filtered alongside rule results (e.g., “win rate when rule X = PASS and emotion_before = calm”)
