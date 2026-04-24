#!/usr/bin/env python3
"""
SAF Checklist Parameter Sweep
Exhaustive test of item sets, weights, thresholds, RR gates, and strategy combos.
Goal: find configs closest to 8 trades/month at 50% WR (4+ wins/month).
"""

import csv
import random
import math
import sys
from collections import defaultdict
from datetime import datetime
from itertools import product

random.seed(42)

# ══════════════════════════════════════════════════════════════════
#  ITEM CLASSIFICATIONS (from today's per-item analysis)
# ══════════════════════════════════════════════════════════════════

STRONG = ["d1_in_favor", "w1_pattern", "d1_touching_ema", "h4_in_favor", "d1_aoi"]
USEFUL = ["w1_candle_rejection", "d1_structure_rejection", "d1_pattern"]
WEAK   = ["w1_in_favor", "psych_level", "h4_candle_rejection", "w1_touching_ema"]
ANTI   = ["h4_ema_touch", "engulfing"]
ALL_16 = STRONG + USEFUL + WEAK + ANTI

# ══════════════════════════════════════════════════════════════════
#  ITEM SET PRESETS
# ══════════════════════════════════════════════════════════════════

ITEM_SETS = {
    "all_16_equal": {
        "items": ALL_16,
        "weights": {i: 1 for i in ALL_16},
        "desc": "Current 16 items, equal weight",
    },
    "strong_only": {
        "items": STRONG,
        "weights": {i: 1 for i in STRONG},
        "desc": "5 strong items only",
    },
    "strong_useful": {
        "items": STRONG + USEFUL,
        "weights": {i: 1 for i in STRONG + USEFUL},
        "desc": "8 items (strong + useful), equal weight",
    },
    "tiered_2_1": {
        "items": STRONG + USEFUL,
        "weights": {**{i: 2 for i in STRONG}, **{i: 1 for i in USEFUL}},
        "desc": "8 items, strong=2x useful=1x",
    },
    "heavy_3_1": {
        "items": STRONG + USEFUL,
        "weights": {**{i: 3 for i in STRONG}, **{i: 1 for i in USEFUL}},
        "desc": "8 items, strong=3x useful=1x",
    },
    "d1_focused": {
        "items": ["d1_in_favor", "d1_aoi", "d1_touching_ema", "d1_structure_rejection", "d1_pattern"],
        "weights": {i: 1 for i in ["d1_in_favor", "d1_aoi", "d1_touching_ema", "d1_structure_rejection", "d1_pattern"]},
        "desc": "D1 items only (5 items)",
    },
    "cross_tf_core": {
        "items": ["w1_pattern", "d1_in_favor", "d1_aoi", "d1_touching_ema", "h4_in_favor"],
        "weights": {"w1_pattern": 2, "d1_in_favor": 2, "d1_aoi": 1, "d1_touching_ema": 1, "h4_in_favor": 1},
        "desc": "Cross-TF core (w1_pattern+d1_in_favor heavy, d1_aoi+d1_ema+h4_in_favor)",
    },
    "top3_pairs": {
        "items": ["w1_pattern", "d1_structure_rejection", "d1_in_favor"],
        "weights": {"w1_pattern": 2, "d1_structure_rejection": 1, "d1_in_favor": 1},
        "desc": "Top 3 confluence pairs (w1_pattern, d1_struct_rej, d1_in_favor)",
    },
}

# ══════════════════════════════════════════════════════════════════
#  DATA LOADING
# ══════════════════════════════════════════════════════════════════

def load_trades(path="trade_analysis.csv"):
    trades = []
    with open(path) as f:
        for row in csv.DictReader(f):
            row["score"] = int(row["score"])
            row["rMult"] = float(row["rMult"])
            row["rr"] = float(row["rr"])
            row["engulfing"] = int(row["engulfing"])
            row["date_obj"] = datetime.strptime(row["date"], "%Y-%m-%d")
            # Pre-parse all item booleans
            for item in ALL_16:
                row[item] = int(row.get(item, 0))
            trades.append(row)
    trades.sort(key=lambda t: t["date_obj"])
    return trades


# ══════════════════════════════════════════════════════════════════
#  SCORING & FILTERING
# ══════════════════════════════════════════════════════════════════

def compute_quality_score(trade, items, weights):
    """Compute weighted score for a trade given item set and weights."""
    return sum(weights.get(item, 0) * trade[item] for item in items)


def filter_trades(trades, config):
    """Apply a config filter and return matching trades."""
    item_set = config["items"]
    weights = config["weights"]
    min_score = config["min_score"]
    min_rr = config["min_rr"]
    exclude_eng = config["exclude_engulfing"]
    strategy = config["strategy"]  # "checklist", "dedicated", "combined"

    result = []
    for t in trades:
        is_checklist = t["score"] > 0
        is_dedicated = t["score"] == 0

        # Strategy filter
        if strategy == "checklist" and not is_checklist:
            continue
        if strategy == "dedicated" and not is_dedicated:
            continue

        # Engulfing filter
        if exclude_eng and t["engulfing"] == 1:
            continue

        # RR gate
        if t["rr"] < min_rr:
            continue

        # Quality score gate (only for checklist trades; dedicated always pass)
        if is_checklist:
            qs = compute_quality_score(t, item_set, weights)
            if qs < min_score:
                continue

        result.append(t)
    return result


# ══════════════════════════════════════════════════════════════════
#  STATS
# ══════════════════════════════════════════════════════════════════

def calc_stats(trades):
    if not trades:
        return None
    n = len(trades)
    wins = sum(1 for t in trades if t["result"] == "win")
    wr = wins / n
    rmults = [t["rMult"] for t in trades]
    exp = sum(rmults) / n
    total_r = sum(rmults)

    d0 = trades[0]["date_obj"]
    d1 = trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)
    per_month = n / (years * 12)
    wins_per_month = wins / (years * 12)

    # Max drawdown in R
    peak = cum = max_dd = 0
    for r in rmults:
        cum += r
        if cum > peak:
            peak = cum
        dd = peak - cum
        if dd > max_dd:
            max_dd = dd

    return {
        "trades": n, "wins": wins, "wr": wr, "exp": exp,
        "total_r": total_r, "max_dd_r": max_dd,
        "years": years, "per_month": per_month,
        "wins_per_month": wins_per_month,
    }


# ══════════════════════════════════════════════════════════════════
#  EQUITY & MONTE CARLO (from simulate.py)
# ══════════════════════════════════════════════════════════════════

def simulate_equity(rmults, start_capital, risk_pct):
    equity = start_capital
    peak = start_capital
    max_dd_pct = 0
    for r in rmults:
        equity *= (1 + risk_pct * r)
        equity = max(equity, 0.01)
        if equity > peak:
            peak = equity
        dd = (peak - equity) / peak
        if dd > max_dd_pct:
            max_dd_pct = dd
    return equity, max_dd_pct


def monte_carlo(rmults, start_capital, risk_pct, n_sims=1000):
    max_dds = []
    max_streaks = []
    for _ in range(n_sims):
        shuffled = rmults[:]
        random.shuffle(shuffled)
        eq = start_capital
        peak = start_capital
        max_dd = 0
        streak = max_streak = 0
        for r in shuffled:
            eq *= (1 + risk_pct * r)
            eq = max(eq, 0.01)
            if eq > peak:
                peak = eq
            dd = (peak - eq) / peak
            if dd > max_dd:
                max_dd = dd
            if r < 0:
                streak += 1
                if streak > max_streak:
                    max_streak = streak
            else:
                streak = 0
        max_dds.append(max_dd)
        max_streaks.append(max_streak)

    max_dds.sort()
    max_streaks.sort()
    # Final equity (path-independent)
    final = start_capital
    for r in rmults:
        final *= (1 + risk_pct * r)

    return {
        "final": final,
        "med_dd": max_dds[len(max_dds) // 2],
        "p95_dd": max_dds[int(len(max_dds) * 0.95)],
        "worst_dd": max_dds[-1],
        "med_streak": max_streaks[len(max_streaks) // 2],
        "p95_streak": max_streaks[int(len(max_streaks) * 0.95)],
    }


# ══════════════════════════════════════════════════════════════════
#  SWEEP ENGINE
# ══════════════════════════════════════════════════════════════════

def generate_configs():
    """Generate all parameter combinations."""
    configs = []

    for iset_name, iset in ITEM_SETS.items():
        max_score = sum(iset["weights"].values())

        # Threshold sweep: every integer from 1 to max
        thresholds = list(range(1, max_score + 1))

        for min_score, min_rr, exclude_eng, strategy in product(
            thresholds,
            [1.5, 2.0, 2.5, 3.0],
            [True, False],
            ["checklist", "dedicated", "combined"],
        ):
            # Skip nonsensical: dedicated strategy ignores checklist params
            # but we still want one entry for dedicated per RR/eng combo
            if strategy == "dedicated" and (min_score > 1 or iset_name != "all_16_equal"):
                continue

            name = f"{iset_name}≥{min_score}"
            if strategy == "combined":
                name += "+ded"
            elif strategy == "dedicated":
                name = "dedicated_only"
            if exclude_eng:
                name += "_noEng"
            name += f"_RR{min_rr}"

            configs.append({
                "name": name,
                "iset_name": iset_name,
                "iset_desc": iset["desc"],
                "items": iset["items"],
                "weights": iset["weights"],
                "min_score": min_score,
                "min_rr": min_rr,
                "exclude_engulfing": exclude_eng,
                "strategy": strategy,
            })

    return configs


def run_sweep(trades):
    configs = generate_configs()
    results = []

    for cfg in configs:
        filtered = filter_trades(trades, cfg)
        stats = calc_stats(filtered)
        if not stats or stats["trades"] < 10:  # skip configs with < 10 trades
            continue
        results.append({**cfg, **stats})

    return results


# ══════════════════════════════════════════════════════════════════
#  SCORING: how close to user target (8/mo, 50% WR)
# ══════════════════════════════════════════════════════════════════

def target_score(r):
    """Score how close a config is to the target. Higher = better."""
    # Target: 8/mo, 50% WR, positive expectancy
    wr_score = -abs(r["wr"] - 0.50) * 200         # penalty for WR distance from 50%
    freq_score = -abs(r["per_month"] - 8) * 5      # penalty for frequency distance from 8/mo
    exp_score = r["exp"] * 30                       # reward positive edge
    total_r_score = r["total_r"] * 0.1              # reward total profit
    dd_penalty = -r["max_dd_r"] * 0.5              # penalize drawdown

    # Hard bonus for WR >= 50%
    wr_bonus = 20 if r["wr"] >= 0.50 else 0
    # Hard bonus for >= 4 wins/month
    wins_bonus = 15 if r["wins_per_month"] >= 4 else 0

    return wr_score + freq_score + exp_score + total_r_score + dd_penalty + wr_bonus + wins_bonus


# ══════════════════════════════════════════════════════════════════
#  OUTPUT
# ══════════════════════════════════════════════════════════════════

def print_header(text):
    print(f"\n{'='*90}")
    print(f"  {text}")
    print(f"{'='*90}")


def print_sweep_table(results, title, limit=30):
    print_header(title)
    print(f"  {'Rank':>4}  {'Config':<45} {'Trades':>6} {'/mo':>5} {'WR':>6} {'Exp':>7} {'TotalR':>7} {'MaxDD':>6} {'W/mo':>5}")
    print(f"  {'─'*4}  {'─'*45} {'─'*6} {'─'*5} {'─'*6} {'─'*7} {'─'*7} {'─'*6} {'─'*5}")
    for i, r in enumerate(results[:limit]):
        print(f"  {i+1:>4}  {r['name']:<45} {r['trades']:>6} {r['per_month']:>4.1f} "
              f"{r['wr']*100:>5.1f}% {r['exp']:>+6.2f}R {r['total_r']:>+6.0f}R "
              f"{r['max_dd_r']:>5.1f}R {r['wins_per_month']:>4.1f}")


def print_deep_dive(r, trades):
    """Detailed equity + MC analysis for one config."""
    filtered = filter_trades(trades, r)
    rmults = [t["rMult"] for t in filtered]
    stats = r

    print(f"\n  ┌─ {r['name']}")
    print(f"  │  {r['iset_desc']}")
    print(f"  │  {stats['trades']} trades | {stats['per_month']:.1f}/mo | WR {stats['wr']*100:.1f}% | "
          f"Exp {stats['exp']:+.2f}R | Wins/mo {stats['wins_per_month']:.1f}")
    print(f"  │")
    print(f"  │  {'Risk':>6}  {'Final$':>12}  {'MedDD%':>8}  {'P95DD%':>8}  {'WorstDD':>8}  {'Streak':>7}  {'$/yr':>10}")
    print(f"  │  {'─'*6}  {'─'*12}  {'─'*8}  {'─'*8}  {'─'*8}  {'─'*7}  {'─'*10}")

    for risk in [0.01, 0.02, 0.03, 0.05]:
        mc = monte_carlo(rmults, 1000, risk, 1000)
        annual = (mc["final"] - 1000) / stats["years"] if mc["final"] > 1000 else 0
        print(f"  │  {risk*100:5.1f}%  ${mc['final']:>10,.0f}  {mc['med_dd']*100:7.1f}%  "
              f"{mc['p95_dd']*100:7.1f}%  {mc['worst_dd']*100:7.1f}%  "
              f"{mc['p95_streak']:>5}L  ${annual:>9,.0f}")

    # Year-by-year at 2%
    print(f"  │")
    print(f"  │  Year-by-Year (2% risk):")
    yearly = defaultdict(list)
    for t in filtered:
        yearly[t["date"][:4]].append(t)
    equity = 1000
    peak_eq = 1000
    for year in sorted(yearly.keys()):
        yts = yearly[year]
        wr = sum(1 for t in yts if t["result"] == "win") / len(yts) * 100
        pnl = sum(t["rMult"] for t in yts)
        for t in yts:
            equity *= (1 + 0.02 * t["rMult"])
            equity = max(equity, 0.01)
        if equity > peak_eq:
            peak_eq = equity
        dd = (peak_eq - equity) / peak_eq * 100
        print(f"  │    {year}  {len(yts):>3} trades  WR={wr:5.1f}%  PnL={pnl:>+6.1f}R  Equity=${equity:>10,.0f}  DD={dd:4.1f}%")
    print(f"  └{'─'*70}")


def print_pair_scaling(top_results):
    print_header("PAIR SCALING PROJECTION — What if you add more pairs?")
    print(f"\n  Current: 7 pairs (EURUSD, GBPUSD, AUDUSD, NZDUSD, USDCAD, USDCHF, USDJPY)")
    print(f"  Candidates: GBPJPY, EURJPY, AUDJPY, GBPAUD, EURGBP, NZDJPY, CADJPY, EURAUD")
    print()
    print(f"  {'Config':<40} {'Now/mo':>7} {'10p/mo':>7} {'14p/mo':>7} {'Now W':>6} {'10p W':>6} {'14p W':>6}")
    print(f"  {'─'*40} {'─'*7} {'─'*7} {'─'*7} {'─'*6} {'─'*6} {'─'*6}")

    for r in top_results[:10]:
        # Linear scaling assumption: more pairs = proportionally more trades, same WR
        cur = r["per_month"]
        w = r["wins_per_month"]
        s10 = cur * 10 / 7
        s14 = cur * 14 / 7
        w10 = w * 10 / 7
        w14 = w * 14 / 7
        print(f"  {r['name']:<40} {cur:>6.1f} {s10:>6.1f} {s14:>6.1f} "
              f"{w:>5.1f} {w10:>5.1f} {w14:>5.1f}")

    print()
    print(f"  Target: 8/mo with 4+ wins")
    print(f"  Note: Linear scaling assumes new pairs have similar edge. Must validate with backtest.")


def print_target_analysis(results, trades):
    print_header("TARGET ANALYSIS — Closest to 8/mo, 50% WR, 4+ wins/month")

    # Find configs that hit or come closest to each target dimension
    best_wr = max(results, key=lambda r: r["wr"] if r["per_month"] >= 1.0 else 0)
    best_freq = max(results, key=lambda r: r["per_month"] if r["exp"] > 0 else 0)
    best_exp = max(results, key=lambda r: r["exp"] if r["trades"] >= 20 else 0)
    best_balanced = max(results, key=target_score)
    best_wins_mo = max(results, key=lambda r: r["wins_per_month"] if r["wr"] >= 0.40 else 0)

    # 50%+ WR configs
    wr50 = [r for r in results if r["wr"] >= 0.50]
    # 40%+ WR configs with >= 2/mo
    wr40_freq = [r for r in results if r["wr"] >= 0.40 and r["per_month"] >= 2.0]

    print(f"\n  ── Best by Win Rate (≥1 trade/mo) ──")
    print(f"  {best_wr['name']}: WR={best_wr['wr']*100:.1f}%, {best_wr['per_month']:.1f}/mo, Exp={best_wr['exp']:+.2f}R")

    print(f"\n  ── Best by Frequency (positive edge) ──")
    print(f"  {best_freq['name']}: {best_freq['per_month']:.1f}/mo, WR={best_freq['wr']*100:.1f}%, Exp={best_freq['exp']:+.2f}R")

    print(f"\n  ── Best by Expectancy (≥20 trades) ──")
    print(f"  {best_exp['name']}: Exp={best_exp['exp']:+.2f}R, WR={best_exp['wr']*100:.1f}%, {best_exp['per_month']:.1f}/mo")

    print(f"\n  ── Best Wins/Month (≥40% WR) ──")
    print(f"  {best_wins_mo['name']}: {best_wins_mo['wins_per_month']:.1f} wins/mo, WR={best_wins_mo['wr']*100:.1f}%, {best_wins_mo['per_month']:.1f}/mo")

    print(f"\n  ── Best Balanced (target scorer) ──")
    print(f"  {best_balanced['name']}: {best_balanced['per_month']:.1f}/mo, WR={best_balanced['wr']*100:.1f}%, "
          f"Exp={best_balanced['exp']:+.2f}R, {best_balanced['wins_per_month']:.1f} wins/mo")

    print(f"\n  ── Configs with ≥50% WR ──")
    if wr50:
        wr50.sort(key=lambda r: r["per_month"], reverse=True)
        for r in wr50[:10]:
            print(f"    {r['name']:<45} {r['trades']:>4} trades  {r['per_month']:.1f}/mo  "
                  f"WR={r['wr']*100:.1f}%  Exp={r['exp']:+.2f}R  W/mo={r['wins_per_month']:.1f}")
    else:
        print(f"    NONE found. Closest:")
        close = sorted(results, key=lambda r: abs(r["wr"] - 0.50))[:5]
        for r in close:
            print(f"    {r['name']:<45} WR={r['wr']*100:.1f}%  {r['per_month']:.1f}/mo  Exp={r['exp']:+.2f}R")

    print(f"\n  ── Configs with ≥40% WR AND ≥2/mo ──")
    if wr40_freq:
        wr40_freq.sort(key=lambda r: r["wins_per_month"], reverse=True)
        for r in wr40_freq[:10]:
            print(f"    {r['name']:<45} {r['trades']:>4} trades  {r['per_month']:.1f}/mo  "
                  f"WR={r['wr']*100:.1f}%  Exp={r['exp']:+.2f}R  W/mo={r['wins_per_month']:.1f}")

    # Gap analysis
    print(f"\n  ── GAP TO TARGET ──")
    b = best_balanced
    target_tpm = 8
    target_wr = 0.50
    target_wpm = 4
    gap_freq = max(0, target_tpm - b["per_month"])
    gap_wr = max(0, target_wr - b["wr"]) * 100
    gap_wpm = max(0, target_wpm - b["wins_per_month"])

    print(f"  Best config: {b['name']}")
    print(f"  Currently: {b['per_month']:.1f}/mo, {b['wr']*100:.1f}% WR, {b['wins_per_month']:.1f} wins/mo")
    print(f"  Target:    {target_tpm}/mo, {target_wr*100:.0f}% WR, {target_wpm} wins/mo")
    print(f"  Gap:       {gap_freq:+.1f} trades/mo, {gap_wr:+.1f}pp WR, {gap_wpm:+.1f} wins/mo")

    if gap_freq > 0:
        pairs_needed = math.ceil(7 * (target_tpm / b["per_month"])) if b["per_month"] > 0 else 99
        print(f"\n  To reach {target_tpm}/mo at current edge: need ~{pairs_needed} pairs (currently 7)")
        print(f"  OR add H1 entry timing / new strategy types")

    if gap_wr > 0:
        print(f"\n  To reach {target_wr*100:.0f}% WR: tighten filters further (but kills frequency)")
        print(f"  Best WR config available: {best_wr['name']} at {best_wr['wr']*100:.1f}% ({best_wr['per_month']:.1f}/mo)")


def print_variety_menu(results, trades):
    """Show a menu of diverse configs the user can choose between."""
    print_header("SETUP VARIETY MENU — Different configs for different market conditions")

    # Pick diverse configs: best WR, best freq, best exp, best balanced, best total profit
    categories = {
        "SNIPER (highest WR)": max((r for r in results if r["per_month"] >= 0.5), key=lambda r: r["wr"], default=None),
        "ACTIVE (most trades)": max((r for r in results if r["exp"] > 0 and r["wr"] >= 0.30), key=lambda r: r["per_month"], default=None),
        "EDGE KING (best exp)": max((r for r in results if r["trades"] >= 30), key=lambda r: r["exp"], default=None),
        "PROFIT MACHINE (best total R)": max((r for r in results if r["wr"] >= 0.35), key=lambda r: r["total_r"], default=None),
        "BALANCED (target scorer)": max(results, key=target_score),
        "SAFE (lowest DD)": min((r for r in results if r["exp"] > 0 and r["trades"] >= 30), key=lambda r: r["max_dd_r"], default=None),
    }

    for label, r in categories.items():
        if r is None:
            continue
        print(f"\n  [{label}]")
        print(f"  Config: {r['name']}")
        print(f"  {r['iset_desc']}")
        print(f"  {r['trades']} trades | {r['per_month']:.1f}/mo | WR {r['wr']*100:.1f}% | "
              f"Exp {r['exp']:+.2f}R | Total {r['total_r']:+.0f}R | MaxDD {r['max_dd_r']:.1f}R | "
              f"Wins/mo {r['wins_per_month']:.1f}")

        # Quick equity at 2%
        filtered = filter_trades(trades, r)
        rmults = [t["rMult"] for t in filtered]
        eq, dd = simulate_equity(rmults, 1000, 0.02)
        print(f"  $1,000 at 2% risk → ${eq:,.0f} (max DD {dd*100:.1f}%) over {r['years']:.0f} years")


# ══════════════════════════════════════════════════════════════════
#  MAIN
# ══════════════════════════════════════════════════════════════════

def main():
    trades = load_trades()
    print(f"Loaded {len(trades)} trades from trade_analysis.csv")
    print(f"Date range: {trades[0]['date']} to {trades[-1]['date']}")

    # Baseline verification
    no_eng = [t for t in trades if t["engulfing"] == 0]
    base = calc_stats(no_eng)
    print(f"Baseline (no engulfing): {base['trades']} trades, {base['per_month']:.1f}/mo, "
          f"WR={base['wr']*100:.1f}%, Exp={base['exp']:+.2f}R")

    # Run sweep
    print(f"\nRunning parameter sweep...")
    results = run_sweep(trades)
    print(f"Generated {len(results)} valid configurations")

    # Sort by target score
    results.sort(key=target_score, reverse=True)

    # ── Part 1: Top 30 configs ──
    print_sweep_table(results, "TOP 30 CONFIGS (sorted by target score: 8/mo + 50% WR + positive edge)")

    # ── Also show top by pure WR ──
    by_wr = sorted([r for r in results if r["per_month"] >= 0.5], key=lambda r: r["wr"], reverse=True)
    print_sweep_table(by_wr, "TOP 30 BY WIN RATE (≥0.5 trades/mo)", 30)

    # ── Top by wins/month ──
    by_wpm = sorted([r for r in results if r["wr"] >= 0.35], key=lambda r: r["wins_per_month"], reverse=True)
    print_sweep_table(by_wpm, "TOP 30 BY WINS PER MONTH (≥35% WR)", 30)

    # ── Part 2: Deep dive top 5 ──
    print_header("DEEP DIVE — Top 5 Balanced Configs")
    seen = set()
    count = 0
    for r in results:
        # Deduplicate by similar stats
        key = f"{r['wr']:.2f}_{r['per_month']:.1f}_{r['exp']:.2f}"
        if key in seen:
            continue
        seen.add(key)
        print_deep_dive(r, trades)
        count += 1
        if count >= 5:
            break

    # ── Part 3: Variety menu ──
    print_variety_menu(results, trades)

    # ── Part 4: Pair scaling ──
    print_pair_scaling(results)

    # ── Part 5: Target analysis ──
    print_target_analysis(results, trades)

    print(f"\n{'='*90}")
    print(f"  SWEEP COMPLETE — {len(results)} configs tested")
    print(f"{'='*90}\n")


if __name__ == "__main__":
    main()
