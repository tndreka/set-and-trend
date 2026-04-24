#!/usr/bin/env python3
"""
68-Day Window Simulation — $100 Start, Apr 24 → July 1

Answers two practical questions:
  1. What leverage / position sizing is actually feasible on a $100 account?
  2. What does a 68-day window historically produce with this edge?

Approach:
  - Filter the 3-leg deduped trades (same logic as portfolio_sim.py).
  - Slide a 68-day window across all historical data; record final equity
    for each starting day that has ≥1 trade in the window.
  - Report distribution (min / 10th / median / 90th / max percentiles).
"""

import csv
from collections import defaultdict
from datetime import datetime, timedelta
from statistics import median

CSV_PATH = "trade_analysis.csv"

STRATEGIES = ["SAF_W1_EMA_REJECTION", "SAF_D1_REJECTION_STRUCTURE", "SAF_W1_REJECTION_D1_AOI"]

# ── Costs ─────────────────────────────────────────────────
SPREAD_PIPS = {
    "EURUSD": 1.0, "GBPUSD": 1.3, "AUDUSD": 1.2, "NZDUSD": 1.5,
    "USDCAD": 1.5, "USDCHF": 1.3, "USDJPY": 1.0,
    "EURJPY": 1.5, "GBPJPY": 2.0, "AUDJPY": 1.8, "CADJPY": 2.5,
    "CHFJPY": 2.5, "NZDJPY": 2.5, "EURGBP": 1.2, "EURAUD": 2.0,
    "EURCAD": 2.0, "EURCHF": 1.8, "GBPCHF": 2.5, "GBPAUD": 2.5,
    "AUDCHF": 2.5, "AUDNZD": 2.5, "XAUUSD": 2.5,
}
SLIPPAGE_PIPS = {k: max(0.3, v * 0.35) for k, v in SPREAD_PIPS.items()}
SWAP_USD = {
    "EURUSD": 8.0, "GBPUSD": 9.0, "AUDUSD": 7.0, "NZDUSD": 7.0,
    "USDCAD": 6.0, "USDCHF": 5.0, "USDJPY": 7.0,
    "EURJPY": 8.0, "GBPJPY": 9.0, "AUDJPY": 7.0, "CADJPY": 7.0,
    "CHFJPY": 7.0, "NZDJPY": 8.0, "EURGBP": 6.0, "EURAUD": 8.0,
    "EURCAD": 7.0, "EURCHF": 5.0, "GBPCHF": 7.0, "GBPAUD": 8.0,
    "AUDCHF": 7.0, "AUDNZD": 7.0, "XAUUSD": 12.0,
}
COMMISSION = 7.0        # round-trip per standard lot
PIP_VALUE = 10.0        # $ per pip per standard lot (simplified)
PIP_SIZE = {sym: (0.01 if "JPY" in sym else 0.0001) for sym in SPREAD_PIPS}
PIP_SIZE["XAUUSD"] = 0.01
HOLD_DAYS = {"win": 4, "loss": 2, "timeout": 10}
SWAP_MULT = 7 / 5


def load_deduped():
    rows = []
    with open(CSV_PATH) as f:
        for r in csv.DictReader(f):
            if r["strategy"] not in STRATEGIES:
                continue
            r["rMult"] = float(r["rMult"])
            r["entry"] = float(r["entry"])
            r["sl"] = float(r["sl"])
            r["date_obj"] = datetime.strptime(r["date"], "%Y-%m-%d")
            rows.append(r)
    rows.sort(key=lambda t: t["date_obj"])
    # dedup by (symbol, date, direction), priority W1_EMA > D1_STRUCT > W1_D1_AOI
    pri = {s: i for i, s in enumerate(STRATEGIES)}
    dedup = {}
    for t in rows:
        k = (t["symbol"], t["date"], t["direction"])
        if k not in dedup or pri[t["strategy"]] < pri[dedup[k]["strategy"]]:
            dedup[k] = t
    out = sorted(dedup.values(), key=lambda t: t["date_obj"])
    return out


def trade_net(t, equity, risk_pct, min_lot):
    """Return $ P&L for one trade using realistic costs and broker lot minimum."""
    sym = t["symbol"]
    pip = PIP_SIZE.get(sym, 0.0001)
    risk_pips = abs(t["entry"] - t["sl"]) / pip
    if risk_pips <= 0:
        return 0.0, 0.0

    target_lots = (equity * risk_pct) / (risk_pips * PIP_VALUE)
    lots = max(target_lots, min_lot)       # broker forces min lot
    effective_risk_usd = lots * risk_pips * PIP_VALUE

    spread_usd = SPREAD_PIPS.get(sym, 2.0) * lots * PIP_VALUE
    slip_usd = SLIPPAGE_PIPS.get(sym, 0.7) * lots * PIP_VALUE
    commission = COMMISSION * lots
    hold_n = HOLD_DAYS.get(t["result"], 4) * SWAP_MULT
    swap = SWAP_USD.get(sym, 7.0) * hold_n * lots

    gross = t["rMult"] * effective_risk_usd
    costs = spread_usd + slip_usd + commission + swap
    return gross - costs, effective_risk_usd


def simulate_window(trades, start_capital, risk_pct, min_lot, max_drawdown_cut=0.5):
    eq = start_capital
    peak = eq
    dd = 0.0
    wins = 0
    for t in trades:
        pnl, eff_risk = trade_net(t, eq, risk_pct, min_lot)
        # Catastrophic: if forced size exceeds remaining equity, account blown
        if eff_risk > eq:
            return 0.01, len(trades), wins, 1.0
        eq = max(eq + pnl, 0.01)
        if eq > peak:
            peak = eq
        cur_dd = (peak - eq) / peak
        if cur_dd > dd:
            dd = cur_dd
        if t["result"] == "win":
            wins += 1
        if dd > max_drawdown_cut:
            break
    return eq, len(trades), wins, dd


def percentile(sorted_list, p):
    if not sorted_list:
        return 0
    i = max(0, min(len(sorted_list) - 1, int(p * (len(sorted_list) - 1))))
    return sorted_list[i]


def print_distribution(label, results):
    finals = sorted(r[0] for r in results)
    trade_counts = [r[1] for r in results]
    wrs = [r[2] / r[1] if r[1] > 0 else 0 for r in results]
    dds = sorted(r[3] for r in results)
    blown = sum(1 for r in results if r[0] <= 10)   # < $10 = effectively wiped

    print(f"\n  {label}")
    print(f"  ─────────────────────────────────────────────")
    print(f"  Windows tested:       {len(results)}")
    print(f"  Avg trades/window:    {sum(trade_counts)/len(trade_counts):.1f}")
    print(f"  Avg WR:               {sum(wrs)/len(wrs)*100:.1f}%")
    print(f"  Blown accounts (<$10): {blown}  ({blown/len(results)*100:.1f}%)")
    print(f"")
    print(f"  Final equity distribution:")
    print(f"    Worst:     ${finals[0]:>8,.2f}")
    print(f"    10th %ile: ${percentile(finals, 0.10):>8,.2f}")
    print(f"    Median:    ${percentile(finals, 0.50):>8,.2f}")
    print(f"    Average:   ${sum(finals)/len(finals):>8,.2f}")
    print(f"    90th %ile: ${percentile(finals, 0.90):>8,.2f}")
    print(f"    Best:      ${finals[-1]:>8,.2f}")
    print(f"  Max drawdown distribution:")
    print(f"    Median:    {percentile(dds, 0.50)*100:>5.1f}%")
    print(f"    90th %ile: {percentile(dds, 0.90)*100:>5.1f}%")


def main():
    trades = load_deduped()
    if not trades:
        print("No trades found"); return

    print(f"Loaded {len(trades)} deduped 3-leg trades")
    print(f"Data spans {trades[0]['date']} → {trades[-1]['date']}")
    print(f"Expected trades per year: {len(trades)/((trades[-1]['date_obj'] - trades[0]['date_obj']).days/365.25):.1f}")

    window_days = 68   # Apr 24 → Jul 1 = 68 calendar days

    # Slide window across every start date that has ≥1 trade in the next 68 days.
    # Start date = every day from first trade to last-trade - 68 days.
    first_day = trades[0]["date_obj"]
    last_day = trades[-1]["date_obj"] - timedelta(days=window_days)

    # Generate candidate start dates: one per month to keep tractable but representative.
    start_dates = []
    cursor = first_day
    while cursor <= last_day:
        start_dates.append(cursor)
        cursor += timedelta(days=7)   # weekly start dates
    print(f"Sliding {window_days}-day windows, weekly step → {len(start_dates)} windows\n")

    print("=" * 90)
    print(f"  LEVERAGE CONSTRAINT CHECK — $100 account")
    print("=" * 90)
    print(f"""
  Real brokers quote "leverage" (e.g. 1:30 EU, 1:500 offshore), but what matters
  for this strategy is POSITION SIZING.  The SLs are typically 20-50 pips.

  Minimum position sizes:
    - Standard lot (1.00)  = 100,000 units   → $10/pip   (needs $2,000+ account)
    - Mini lot    (0.10)   = 10,000  units   → $1/pip    (needs $200+ account)
    - Micro lot   (0.01)   = 1,000   units   → $0.10/pip ({'✅' if 100 >= 100 else '❌'} fits $100)
    - Nano lot    (0.001)  = 100     units   → $0.01/pip (rare — XM, Exness, FBS)

  With $100 and 0.01 min lot on a 30-pip SL trade:
    risk_usd = 0.01 * 30 * $10 = $3.00  →  3% of account per trade
    Intended 1%% risk is UNREACHABLE without a nano-lot broker.

  Leverage needed:
    - 0.01 lot on EURUSD = $1,000 notional
    - $100 margin → 10:1 leverage needed → well within EU 1:30 or offshore 1:500
    - So: leverage is NOT the constraint. Lot minimum is.
""")

    print("=" * 90)
    print(f"  HISTORICAL {window_days}-DAY WINDOW OUTCOMES — $100 start")
    print("=" * 90)
    print(f"""
  This shows what happened historically if you had started a $100 account on
  any given week and stopped 68 days later. Read the distribution — not the
  median — because 2 months of trading is HEAVY variance with ~6 trades.
""")

    start_capital = 100

    for label, risk_pct, min_lot in [
        ("NANO-LOT broker (0.001 min), 1% risk — ideal",      0.01, 0.001),
        ("MICRO-LOT broker (0.01 min), 1% intended — forced to ~3%", 0.01, 0.01),
        ("MICRO-LOT broker (0.01 min), 0.5% intended — forced to ~3%", 0.005, 0.01),
        ("MICRO-LOT + accept 3% actual risk",                  0.03, 0.01),
    ]:
        results = []
        for start in start_dates:
            win_end = start + timedelta(days=window_days)
            window_trades = [t for t in trades if start <= t["date_obj"] < win_end]
            if not window_trades:
                continue
            result = simulate_window(window_trades, start_capital, risk_pct, min_lot)
            results.append(result)
        if results:
            print_distribution(label, results)


if __name__ == "__main__":
    main()
