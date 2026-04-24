#!/usr/bin/env python3
"""
Blind-Spot Analysis — what is the backtest missing?

Checks the deduped 3-leg portfolio for:
  1. Per-pair edge (are some pairs carrying all the alpha?)
  2. Per-strategy RR distribution (is the 2:1 RR actually optimal?)
  3. Direction bias (LONG vs SHORT asymmetry)
  4. Day-of-week / month seasonality
  5. Consecutive-loss runs (can you tolerate them?)
  6. Timeout trades (wasted capital / opportunity cost)
  7. Duration in trade (is the 10-day timeout cap killing profit?)
"""

import csv
from collections import defaultdict
from datetime import datetime

CSV = "trade_analysis.csv"
STRATEGIES = ["SAF_W1_EMA_REJECTION", "SAF_D1_REJECTION_STRUCTURE", "SAF_W1_REJECTION_D1_AOI"]


def load_deduped():
    rows = []
    with open(CSV) as f:
        for r in csv.DictReader(f):
            if r["strategy"] not in STRATEGIES:
                continue
            r["rMult"] = float(r["rMult"])
            r["rr"] = float(r["rr"])
            r["date_obj"] = datetime.strptime(r["date"], "%Y-%m-%d")
            rows.append(r)
    rows.sort(key=lambda t: t["date_obj"])
    pri = {s: i for i, s in enumerate(STRATEGIES)}
    dedup = {}
    for t in rows:
        k = (t["symbol"], t["date"], t["direction"])
        if k not in dedup or pri[t["strategy"]] < pri[dedup[k]["strategy"]]:
            dedup[k] = t
    return sorted(dedup.values(), key=lambda t: t["date_obj"])


def summary(trades):
    if not trades:
        return None
    n = len(trades)
    wins = sum(1 for t in trades if t["result"] == "win")
    losses = sum(1 for t in trades if t["result"] == "loss")
    timeouts = sum(1 for t in trades if t["result"] == "timeout")
    total_r = sum(t["rMult"] for t in trades)
    return {
        "n": n, "wins": wins, "losses": losses, "timeouts": timeouts,
        "wr": wins / n, "exp": total_r / n, "total_r": total_r,
    }


def main():
    trades = load_deduped()
    print(f"Loaded {len(trades)} deduped 3-leg trades\n")

    # 1. Per-pair
    print("=" * 90)
    print("  1. PER-PAIR EDGE — is the alpha concentrated in some pairs?")
    print("=" * 90)
    by_pair = defaultdict(list)
    for t in trades:
        by_pair[t["symbol"]].append(t)

    print(f"  {'Pair':<10} {'Trades':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}  Bar")
    print(f"  {'─'*10} {'─'*7} {'─'*7} {'─'*8} {'─'*8}  {'─'*30}")
    pair_rows = []
    for sym, ts in by_pair.items():
        s = summary(ts)
        pair_rows.append((sym, s))
    pair_rows.sort(key=lambda x: -x[1]["total_r"])
    for sym, s in pair_rows:
        bar = "█" * max(0, int(s["total_r"] / 5))
        sign = "+" if s["total_r"] >= 0 else ""
        print(f"  {sym:<10} {s['n']:>7} {s['wr']*100:>6.1f}% {s['exp']:>+8.3f}  {sign}{s['total_r']:>6.1f}R  {bar}")

    # which pairs to drop?
    losers = [sym for sym, s in pair_rows if s["total_r"] < 0 or s["exp"] < 0]
    if losers:
        print(f"\n  ⚠️  Unprofitable or negative-expectancy pairs: {', '.join(losers)}")
        # What does portfolio look like without them?
        filtered = [t for t in trades if t["symbol"] not in losers]
        s = summary(filtered)
        improv = s["total_r"] - summary(trades)["total_r"]
        print(f"  Without them: {s['n']} trades | WR {s['wr']*100:.1f}% | Exp {s['exp']:+.3f}R | "
              f"Total {s['total_r']:+.1f}R  (delta vs full portfolio: {improv:+.1f}R)")

    # 2. Per-strategy RR distribution
    print("\n" + "=" * 90)
    print("  2. PER-STRATEGY RR DISTRIBUTION — is the 2:1 RR optimal?")
    print("=" * 90)
    by_strat = defaultdict(list)
    for t in trades:
        by_strat[t["strategy"]].append(t)
    print(f"  {'Strategy':<30} {'Trades':>7} {'Avg RR':>7} {'Min RR':>7} {'Max RR':>7} {'WR':>6} {'Exp':>7}")
    print(f"  {'─'*30} {'─'*7} {'─'*7} {'─'*7} {'─'*7} {'─'*6} {'─'*7}")
    for strat, ts in by_strat.items():
        s = summary(ts)
        rrs = [t["rr"] for t in ts]
        print(f"  {strat:<30} {s['n']:>7} {sum(rrs)/len(rrs):>7.2f} {min(rrs):>7.2f} {max(rrs):>7.2f} "
              f"{s['wr']*100:>5.1f}% {s['exp']:>+7.3f}")

    # 3. Direction bias
    print("\n" + "=" * 90)
    print("  3. DIRECTION BIAS — LONG vs SHORT")
    print("=" * 90)
    by_dir = defaultdict(list)
    for t in trades:
        by_dir[t["direction"]].append(t)
    print(f"  {'Direction':<10} {'Trades':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}")
    print(f"  {'─'*10} {'─'*7} {'─'*7} {'─'*8} {'─'*8}")
    for d, ts in by_dir.items():
        s = summary(ts)
        print(f"  {d:<10} {s['n']:>7} {s['wr']*100:>6.1f}% {s['exp']:>+8.3f}  {s['total_r']:>+7.1f}R")

    # 4. Day-of-week / month seasonality
    print("\n" + "=" * 90)
    print("  4. DAY-OF-WEEK SEASONALITY")
    print("=" * 90)
    dow = defaultdict(list)
    for t in trades:
        dow[t["date_obj"].strftime("%a")].append(t)
    print(f"  {'Day':<5} {'Trades':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}")
    print(f"  {'─'*5} {'─'*7} {'─'*7} {'─'*8} {'─'*8}")
    for day in ["Mon", "Tue", "Wed", "Thu", "Fri"]:
        ts = dow.get(day, [])
        if not ts:
            continue
        s = summary(ts)
        print(f"  {day:<5} {s['n']:>7} {s['wr']*100:>6.1f}% {s['exp']:>+8.3f}  {s['total_r']:>+7.1f}R")

    # 5. Consecutive losses
    print("\n" + "=" * 90)
    print("  5. CONSECUTIVE LOSS RUNS — psychology check")
    print("=" * 90)
    runs = []
    cur = 0
    for t in trades:
        if t["result"] == "loss":
            cur += 1
        else:
            if cur > 0:
                runs.append(cur)
            cur = 0
    if cur > 0:
        runs.append(cur)
    max_run = max(runs) if runs else 0
    runs_ge5 = sum(1 for r in runs if r >= 5)
    runs_ge7 = sum(1 for r in runs if r >= 7)
    runs_ge10 = sum(1 for r in runs if r >= 10)
    print(f"  Worst losing streak:        {max_run} losses in a row")
    print(f"  Streaks of ≥5 losses:       {runs_ge5}")
    print(f"  Streaks of ≥7 losses:       {runs_ge7}")
    print(f"  Streaks of ≥10 losses:      {runs_ge10}")
    print(f"  Avg streak length:          {sum(runs)/len(runs):.1f}" if runs else "")

    # 6. Timeouts
    print("\n" + "=" * 90)
    print("  6. TIMEOUT TRADES — capital tied up for no outcome")
    print("=" * 90)
    timeouts = [t for t in trades if t["result"] == "timeout"]
    s = summary(timeouts) if timeouts else None
    if s:
        print(f"  Timeout trades: {s['n']} ({s['n']/len(trades)*100:.1f}% of all)")
        print(f"  Timeout expectancy: {s['exp']:+.3f}R  (vs +0.65R portfolio)")
        print(f"  Timeout total R: {s['total_r']:+.1f}")
    else:
        print("  No timeout trades.")

    # 7. Quick "what if" — remove losing pairs
    print("\n" + "=" * 90)
    print("  7. QUICK EXPERIMENTS — impact of simple filters")
    print("=" * 90)

    experiments = [
        ("All 3-leg trades", lambda t: True),
        ("Exclude unprofitable pairs", lambda t: t["symbol"] not in losers),
        ("LONG only", lambda t: t["direction"] == "LONG"),
        ("SHORT only", lambda t: t["direction"] == "SHORT"),
        ("Top 5 pairs only", lambda t: t["symbol"] in [p[0] for p in pair_rows[:5]]),
        ("Top 8 pairs only", lambda t: t["symbol"] in [p[0] for p in pair_rows[:8]]),
        ("Skip Friday trades", lambda t: t["date_obj"].strftime("%a") != "Fri"),
    ]
    print(f"  {'Filter':<35} {'Trades':>7} {'WR':>6} {'Exp':>7} {'TotalR':>8}")
    print(f"  {'─'*35} {'─'*7} {'─'*6} {'─'*7} {'─'*8}")
    for name, fn in experiments:
        f = [t for t in trades if fn(t)]
        if not f:
            continue
        s = summary(f)
        print(f"  {name:<35} {s['n']:>7} {s['wr']*100:>5.1f}% {s['exp']:>+7.3f} {s['total_r']:>+7.1f}R")


if __name__ == "__main__":
    main()
