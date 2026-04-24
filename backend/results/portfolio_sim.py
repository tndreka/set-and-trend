#!/usr/bin/env python3
"""
Portfolio Simulation — 3 SAF Confluence Strategies

Loads trade_analysis.csv (with strategy column), filters to the 3 top
confluence strategies from SAF_CONFLUENCE_ANALYSIS.md, applies a realistic
broker cost model, and projects an equity curve starting from $1,000.

Strategies:
  - SAF_W1_EMA_REJECTION        (best quality, ~2.4/yr, ~45% WR)
  - SAF_D1_REJECTION_STRUCTURE  (good balance, ~4/yr, ~43% WR)
  - SAF_W1_REJECTION_D1_AOI     (best volume, ~7.6/yr, ~38% WR)

SAF_CHECKLIST (the 14-item aggregate) is shown as a baseline for comparison
and is expected to bleed.
"""

import csv
import sys
from collections import defaultdict
from datetime import datetime

CSV_PATH = "trade_analysis.csv"

STRATEGIES = {
    "SAF_W1_EMA_REJECTION": "W1 EMA rejection",
    "SAF_D1_REJECTION_STRUCTURE": "D1 rejection + structure",
    "SAF_W1_REJECTION_D1_AOI": "W1 rejection + D1 AOI",
}
FOURTH_LEG = "SAF_CHECKLIST"
BASELINE_STRATEGY = "SAF_CHECKLIST"


# ──────────────────────────────────────────────────────────────
#  BROKER COST MODEL — ECN tier (IC Markets / Pepperstone style)
# ──────────────────────────────────────────────────────────────
# Per-pair spread (average). Crosses/exotics defaulted conservatively.
SPREAD_PIPS = {
    "EURUSD": 1.0, "GBPUSD": 1.3, "AUDUSD": 1.2, "NZDUSD": 1.5,
    "USDCAD": 1.5, "USDCHF": 1.3, "USDJPY": 1.0,
    # crosses — wider
    "EURJPY": 1.5, "GBPJPY": 2.0, "AUDJPY": 1.8, "CADJPY": 2.5,
    "CHFJPY": 2.5, "NZDJPY": 2.5, "EURGBP": 1.2, "EURAUD": 2.0,
    "EURCAD": 2.0, "EURCHF": 1.8, "GBPCHF": 2.5, "GBPAUD": 2.5,
    "AUDCHF": 2.5, "AUDNZD": 2.5,
    # metals
    "XAUUSD": 2.5,
}
SLIPPAGE_PIPS = {k: max(0.3, v * 0.35) for k, v in SPREAD_PIPS.items()}
# Approx swap/night/lot (USD, both directions averaged, conservative).
SWAP_USD_PER_NIGHT_PER_LOT = {
    "EURUSD": 8.0, "GBPUSD": 9.0, "AUDUSD": 7.0, "NZDUSD": 7.0,
    "USDCAD": 6.0, "USDCHF": 5.0, "USDJPY": 7.0,
    "EURJPY": 8.0, "GBPJPY": 9.0, "AUDJPY": 7.0, "CADJPY": 7.0,
    "CHFJPY": 7.0, "NZDJPY": 8.0, "EURGBP": 6.0, "EURAUD": 8.0,
    "EURCAD": 7.0, "EURCHF": 5.0, "GBPCHF": 7.0, "GBPAUD": 8.0,
    "AUDCHF": 7.0, "AUDNZD": 7.0,
    "XAUUSD": 12.0,
}
COMMISSION_RT_PER_LOT = 7.0
PIP_VALUE_USD = 10.0  # simplified — per standard lot per pip
PIP_SIZE = {sym: (0.01 if "JPY" in sym else 0.0001) for sym in SPREAD_PIPS}
PIP_SIZE["XAUUSD"] = 0.01

HOLD_DAYS = {"win": 4, "loss": 2, "timeout": 10}
SWAP_MULT = 7 / 5   # weekend 3x swap on Wed


def default_or_warn(d, sym, fallback):
    if sym not in d:
        return fallback
    return d[sym]


def load_trades(path):
    rows = []
    with open(path) as f:
        reader = csv.DictReader(f)
        for r in reader:
            try:
                r["score"] = int(r["score"])
                r["rMult"] = float(r["rMult"])
                r["entry"] = float(r["entry"])
                r["sl"] = float(r["sl"])
                r["date_obj"] = datetime.strptime(r["date"], "%Y-%m-%d")
            except (ValueError, KeyError):
                continue
            rows.append(r)
    rows.sort(key=lambda t: t["date_obj"])
    return rows


def trade_costs(trade, equity, risk_pct):
    sym = trade["symbol"]
    pip = PIP_SIZE.get(sym, 0.0001)
    spread = default_or_warn(SPREAD_PIPS, sym, 2.0)
    slip = default_or_warn(SLIPPAGE_PIPS, sym, 0.7)
    swap_night = default_or_warn(SWAP_USD_PER_NIGHT_PER_LOT, sym, 7.0)

    risk_pips = abs(trade["entry"] - trade["sl"]) / pip
    if risk_pips <= 0:
        return 0.0, 0.0
    risk_usd = equity * risk_pct
    lots = risk_usd / (risk_pips * PIP_VALUE_USD)
    if lots < 0.01:
        lots = 0.01

    commission_usd = COMMISSION_RT_PER_LOT * lots
    hold_nights = HOLD_DAYS.get(trade["result"], 4) * SWAP_MULT
    swap_usd = swap_night * hold_nights * lots
    spread_usd = spread * lots * PIP_VALUE_USD
    slip_usd = slip * lots * PIP_VALUE_USD

    total_cost_usd = commission_usd + swap_usd + spread_usd + slip_usd
    cost_r = total_cost_usd / risk_usd if risk_usd > 0 else 0
    return cost_r, total_cost_usd


def simulate(trades, start_capital, risk_pct):
    equity = start_capital
    peak = equity
    max_dd_pct = 0.0
    yearly = defaultdict(lambda: {"trades": 0, "wins": 0, "net": 0.0, "eq_end": 0.0})
    cost_sum = 0.0
    gross_sum = 0.0

    for t in trades:
        cost_r, cost_usd = trade_costs(t, equity, risk_pct)
        risk_usd = equity * risk_pct
        gross = t["rMult"] * risk_usd
        net = gross - cost_usd
        equity = max(equity + net, 0.01)
        peak = max(peak, equity)
        dd = (peak - equity) / peak
        if dd > max_dd_pct:
            max_dd_pct = dd

        y = t["date"][:4]
        yearly[y]["trades"] += 1
        if t["result"] == "win":
            yearly[y]["wins"] += 1
        yearly[y]["net"] += net
        yearly[y]["eq_end"] = equity

        gross_sum += gross
        cost_sum += cost_usd

    return {
        "final_equity": equity,
        "peak": peak,
        "max_dd_pct": max_dd_pct,
        "yearly": dict(yearly),
        "gross": gross_sum,
        "cost": cost_sum,
        "net": equity - start_capital,
    }


def summarize_strategy(name, trades):
    if not trades:
        return
    n = len(trades)
    wins = sum(1 for t in trades if t["result"] == "win")
    losses = sum(1 for t in trades if t["result"] == "loss")
    timeouts = sum(1 for t in trades if t["result"] == "timeout")
    wr = wins / n
    avg_r = sum(t["rMult"] for t in trades) / n
    total_r = sum(t["rMult"] for t in trades)
    d0, d1 = trades[0]["date_obj"], trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)
    print(f"  {name:<30}  {n:>5} tr  {n/years:>5.1f}/yr  WR {wr*100:>5.1f}%  "
          f"Exp {avg_r:+.3f}R  Total {total_r:+.0f}R  L/T {losses}/{timeouts}")


def print_equity_curve(name, result, start, risk_pct, years):
    r = result
    print(f"\n  ── {name} | ${start:,.0f} start | {risk_pct*100:.1f}% risk ──")
    print(f"  {'Year':>6} {'Trades':>7} {'WR':>6} {'Net$':>11} {'Equity':>12}")
    print(f"  {'─'*6} {'─'*7} {'─'*6} {'─'*11} {'─'*12}")
    prior_eq = start
    for year in sorted(r["yearly"].keys()):
        y = r["yearly"][year]
        if y["trades"] == 0:
            continue
        wr = y["wins"] / y["trades"] * 100
        ret_pct = (y["eq_end"] / prior_eq - 1) * 100 if prior_eq > 0 else 0
        print(f"  {year:>6} {y['trades']:>7} {wr:5.1f}% ${y['net']:>+10,.0f} ${y['eq_end']:>11,.0f}  ({ret_pct:+.1f}%)")
        prior_eq = y["eq_end"]

    cagr = (r["final_equity"] / start) ** (1 / years) - 1 if r["final_equity"] > start else -1
    print(f"\n  Final equity:   ${r['final_equity']:>12,.2f}")
    print(f"  Peak equity:    ${r['peak']:>12,.2f}")
    print(f"  Max drawdown:   {r['max_dd_pct']*100:>11.1f}%")
    print(f"  Gross profit:   ${r['gross']:>12,.0f}")
    print(f"  Total costs:    ${r['cost']:>12,.0f}  ({r['cost']/r['gross']*100 if r['gross']>0 else 0:.1f}% of gross)")
    print(f"  Net profit:     ${r['net']:>+12,.2f}")
    print(f"  CAGR:           {cagr*100:>11.1f}%")


def main():
    trades = load_trades(CSV_PATH)
    if not trades:
        print(f"No trades in {CSV_PATH}", file=sys.stderr)
        sys.exit(1)

    d0, d1 = trades[0]["date_obj"], trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)

    by_strategy = defaultdict(list)
    for t in trades:
        by_strategy[t.get("strategy", "?")].append(t)

    print(f"\nLoaded {len(trades)} trades spanning {years:.1f} years ({d0.date()} → {d1.date()})")
    print(f"Strategies present: {sorted(by_strategy.keys())}\n")

    print("=" * 90)
    print("  PER-STRATEGY RAW STATS (no costs)")
    print("=" * 90)
    for code in [BASELINE_STRATEGY] + list(STRATEGIES.keys()):
        summarize_strategy(code, by_strategy.get(code, []))

    start_capital = 1000
    risk_pct = 0.01  # conservative 1%

    print("\n" + "=" * 90)
    print(f"  EQUITY CURVE — 3 Confluence Portfolio (combined chronologically)")
    print("=" * 90)
    combined = []
    for code in STRATEGIES:
        combined.extend(by_strategy.get(code, []))
    combined.sort(key=lambda t: t["date_obj"])

    if combined:
        result = simulate(combined, start_capital, risk_pct)
        print_equity_curve("3-STRATEGY PORTFOLIO (raw union)", result, start_capital, risk_pct, years)

    # Deduped 3-leg — one trade per (symbol, date, direction).
    # Priority: W1_EMA_REJECTION > D1_REJECTION_STRUCTURE > W1_REJECTION_D1_AOI
    # (highest expectancy first, since rMult is identical for same setup).
    priority = list(STRATEGIES.keys())
    dedup_3 = {}
    for t in combined:
        key = (t["symbol"], t["date"], t["direction"])
        existing = dedup_3.get(key)
        if existing is None:
            dedup_3[key] = t
            continue
        if priority.index(t["strategy"]) < priority.index(existing["strategy"]):
            dedup_3[key] = t
    dedup_3_list = sorted(dedup_3.values(), key=lambda t: t["date_obj"])

    if dedup_3_list:
        print("\n" + "=" * 90)
        print("  EQUITY CURVE — 3-Leg Portfolio (DEDUPED, one position per sym/date/dir)")
        print("=" * 90)
        print(f"  Dedup: {len(combined)} raw → {len(dedup_3_list)} unique  "
              f"({len(combined) - len(dedup_3_list)} removed, "
              f"{(len(combined)-len(dedup_3_list))/len(combined)*100:.1f}%)")
        result_dedup3 = simulate(dedup_3_list, start_capital, risk_pct)
        print_equity_curve("3-LEG DEDUPED PORTFOLIO", result_dedup3, start_capital, risk_pct, years)

    print("\n" + "=" * 90)
    print(f"  EQUITY CURVE — Each strategy standalone (${start_capital} each, {risk_pct*100:.0f}% risk)")
    print("=" * 90)
    for code, desc in STRATEGIES.items():
        s_trades = by_strategy.get(code, [])
        if not s_trades:
            continue
        r = simulate(s_trades, start_capital, risk_pct)
        print_equity_curve(f"{code} ({desc})", r, start_capital, risk_pct, years)

    print("\n" + "=" * 90)
    print(f"  EQUITY CURVE — SAF_CHECKLIST baseline (14-item aggregate)")
    print("=" * 90)
    saf = by_strategy.get(BASELINE_STRATEGY, [])
    if saf:
        r = simulate(saf, start_capital, risk_pct)
        print_equity_curve(f"{BASELINE_STRATEGY} (diagnostic)", r, start_capital, risk_pct, years)

    # ── 4-leg portfolio: add SAF_CHECKLIST ──
    print("\n" + "=" * 90)
    print("  EQUITY CURVE — 4-Leg Portfolio (3 confluences + SAF_CHECKLIST)")
    print("=" * 90)
    four_leg = list(combined)
    four_leg.extend(by_strategy.get(FOURTH_LEG, []))
    four_leg.sort(key=lambda t: t["date_obj"])
    result_4 = None
    if four_leg:
        result_4 = simulate(four_leg, start_capital, risk_pct)
        print_equity_curve("4-LEG PORTFOLIO", result_4, start_capital, risk_pct, years)

    # ── Overlap analysis: how many days do multiple legs fire the same trade? ──
    print("\n" + "=" * 90)
    print("  OVERLAP ANALYSIS — same (symbol, date, direction) across legs")
    print("=" * 90)
    all_4 = list(STRATEGIES.keys()) + [FOURTH_LEG]
    trade_keys = defaultdict(list)
    for t in trades:
        if t["strategy"] in all_4:
            key = (t["symbol"], t["date"], t["direction"])
            trade_keys[key].append(t["strategy"])

    dup_count = sum(1 for v in trade_keys.values() if len(v) > 1)
    unique = len(trade_keys)
    total_signals = sum(len(v) for v in trade_keys.values())
    print(f"  Unique (sym,date,dir) keys: {unique}")
    print(f"  Total signals emitted:      {total_signals}")
    print(f"  Keys with ≥2 legs firing:   {dup_count}  ({dup_count/unique*100:.1f}% of unique)")
    print(f"  Redundant signals:          {total_signals - unique}  ({(total_signals-unique)/total_signals*100:.1f}% of all)\n")

    # Pairwise overlap
    pairs = defaultdict(int)
    for strats in trade_keys.values():
        if len(strats) > 1:
            s = sorted(set(strats))
            for i in range(len(s)):
                for j in range(i + 1, len(s)):
                    pairs[(s[i], s[j])] += 1
    if pairs:
        print(f"  {'Leg A':<30} {'Leg B':<30} {'Co-fires':>9}")
        print(f"  {'─'*30} {'─'*30} {'─'*9}")
        for (a, b), n in sorted(pairs.items(), key=lambda x: -x[1]):
            print(f"  {a:<30} {b:<30} {n:>9}")

    # Dedup: if ≥1 leg fired on (sym,date,dir), count it once. We score-dedup
    # by keeping the single trade — since rMult is the same execution, they
    # should be identical. Use the first trade of the key.
    dedup = {}
    for t in trades:
        if t["strategy"] not in all_4:
            continue
        key = (t["symbol"], t["date"], t["direction"])
        # prefer a focused-leg trade over SAF_CHECKLIST if both exist
        if key not in dedup or (dedup[key]["strategy"] == FOURTH_LEG and t["strategy"] != FOURTH_LEG):
            dedup[key] = t
    dedup_list = sorted(dedup.values(), key=lambda t: t["date_obj"])
    print(f"\n  After dedup: {len(dedup_list)} trades (vs {len(four_leg)} raw)")

    if dedup_list:
        result_dedup = simulate(dedup_list, start_capital, risk_pct)
        print_equity_curve("4-LEG DEDUPED PORTFOLIO", result_dedup, start_capital, risk_pct, years)

    # ── Side-by-side comparison ──
    print("\n" + "=" * 90)
    print("  SIDE-BY-SIDE COMPARISON")
    print("=" * 90)
    print(f"  {'Portfolio':<28} {'Trades':>7} {'Tr/yr':>6} {'WR':>6} {'CAGR':>7} {'MaxDD':>7} {'Final':>12}")
    print(f"  {'─'*28} {'─'*7} {'─'*6} {'─'*6} {'─'*7} {'─'*7} {'─'*12}")

    def _row(name, trades_list, r):
        if not r:
            return
        wins = sum(1 for t in trades_list if t["result"] == "win")
        wr = wins / len(trades_list) * 100
        cagr = (r["final_equity"] / start_capital) ** (1 / years) - 1 if r["final_equity"] > start_capital else -1
        print(f"  {name:<28} {len(trades_list):>7} {len(trades_list)/years:>6.1f} {wr:>5.1f}% "
              f"{cagr*100:>6.1f}% {r['max_dd_pct']*100:>6.1f}% ${r['final_equity']:>11,.0f}")

    if combined:
        _row("3-leg (confluences)", combined, simulate(combined, start_capital, risk_pct))
    if four_leg:
        _row("4-leg (raw union)", four_leg, result_4)
    if dedup_list:
        _row("4-leg (deduped)", dedup_list, simulate(dedup_list, start_capital, risk_pct))

    # Risk sensitivity on the portfolio
    if combined:
        print("\n" + "=" * 90)
        print("  RISK SENSITIVITY — 3-strategy portfolio")
        print("=" * 90)
        print(f"  {'Risk':>6} {'Final':>14} {'Net':>14} {'MaxDD':>8} {'CAGR':>8}")
        print(f"  {'─'*6} {'─'*14} {'─'*14} {'─'*8} {'─'*8}")
        for risk in [0.005, 0.01, 0.015, 0.02, 0.03]:
            r = simulate(combined, start_capital, risk)
            cagr = (r["final_equity"] / start_capital) ** (1 / years) - 1 if r["final_equity"] > start_capital else -1
            print(f"  {risk*100:5.1f}% ${r['final_equity']:>13,.0f} ${r['net']:>+13,.0f} "
                  f"{r['max_dd_pct']*100:6.1f}% {cagr*100:6.1f}%")


if __name__ == "__main__":
    main()
