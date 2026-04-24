#!/usr/bin/env python3
"""
Frequency Analysis — what lowering SAF_MIN_SCORE does to the 4-leg portfolio.

Goal: find a threshold that gets ~100 trades/year (2/week) while staying
profitable with decent expectancy.
"""

import csv
import sys
from collections import defaultdict
from datetime import datetime

PATH = sys.argv[1] if len(sys.argv) > 1 else "trade_analysis.csv"

STRATEGIES = ["SAF_W1_EMA_REJECTION", "SAF_D1_REJECTION_STRUCTURE",
              "SAF_W1_REJECTION_D1_AOI", "SAF_CHECKLIST"]


def load():
    rows = []
    with open(PATH) as f:
        for r in csv.DictReader(f):
            if r["strategy"] not in STRATEGIES:
                continue
            r["score"] = int(r["score"])
            r["rMult"] = float(r["rMult"])
            r["rr"] = float(r["rr"])
            r["date_obj"] = datetime.strptime(r["date"], "%Y-%m-%d")
            rows.append(r)
    return sorted(rows, key=lambda t: t["date_obj"])


def dedup(trades):
    pri = {s: i for i, s in enumerate(STRATEGIES)}
    dd = {}
    for t in trades:
        k = (t["symbol"], t["date"], t["direction"])
        if k not in dd or pri[t["strategy"]] < pri[dd[k]["strategy"]]:
            dd[k] = t
    return sorted(dd.values(), key=lambda t: t["date_obj"])


def summary(ts, years):
    if not ts:
        return None
    n = len(ts); wins = sum(1 for t in ts if t["result"] == "win")
    tot = sum(t["rMult"] for t in ts)
    return n, wins / n, tot / n, tot, n / years


def main():
    raw = load()
    years = (raw[-1]["date_obj"] - raw[0]["date_obj"]).days / 365.25
    print(f"Loaded {len(raw)} trades across {years:.1f} years\n")

    # Score distribution
    print("=" * 85)
    print("  SAF_CHECKLIST score distribution")
    print("=" * 85)
    saf = [t for t in raw if t["strategy"] == "SAF_CHECKLIST"]
    by_score = defaultdict(list)
    for t in saf:
        by_score[t["score"]].append(t)
    print(f"  {'Score':>5} {'Trades':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}")
    print(f"  {'─'*5} {'─'*7} {'─'*7} {'─'*8} {'─'*8}")
    for s in sorted(by_score.keys()):
        ts = by_score[s]
        summ = summary(ts, years)
        if summ:
            n, wr, exp, tot, tyr = summ
            print(f"  {s:>5} {n:>7} {wr*100:>6.1f}% {exp:>+8.3f} {tot:>+7.1f}R  ({tyr:.1f}/yr)")

    # Cumulative view — at threshold X, how does SAF_CHECKLIST perform?
    print("\n" + "=" * 85)
    print("  SAF_CHECKLIST cumulative (score ≥ X)")
    print("=" * 85)
    print(f"  {'≥':>3} {'Trades':>7} {'Tr/yr':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}")
    print(f"  {'─'*3} {'─'*7} {'─'*7} {'─'*7} {'─'*8} {'─'*8}")
    for thresh in range(5, 15):
        ts = [t for t in saf if t["score"] >= thresh]
        summ = summary(ts, years)
        if summ:
            n, wr, exp, tot, tyr = summ
            print(f"  {thresh:>3} {n:>7} {tyr:>6.1f}  {wr*100:>6.1f}% {exp:>+8.3f} {tot:>+7.1f}R")

    # 4-leg deduped portfolio at each SAF threshold
    print("\n" + "=" * 85)
    print("  4-LEG DEDUPED PORTFOLIO at varying SAF_CHECKLIST thresholds")
    print("=" * 85)
    focused = [t for t in raw if t["strategy"] != "SAF_CHECKLIST"]
    print(f"  Focused 3-leg baseline (raw, undeduped): {len(focused)} trades")
    print()
    print(f"  {'SAF≥':>5} {'Trades':>7} {'Tr/yr':>7} {'WR':>7} {'Exp':>8} {'TotalR':>8}")
    print(f"  {'─'*5} {'─'*7} {'─'*7} {'─'*7} {'─'*8} {'─'*8}")
    for thresh in range(5, 15):
        saf_t = [t for t in saf if t["score"] >= thresh]
        combined = focused + saf_t
        deduped = dedup(combined)
        summ = summary(deduped, years)
        if summ:
            n, wr, exp, tot, tyr = summ
            print(f"  {thresh:>5} {n:>7} {tyr:>6.1f}  {wr*100:>6.1f}% {exp:>+8.3f} {tot:>+7.1f}R")


if __name__ == "__main__":
    main()
