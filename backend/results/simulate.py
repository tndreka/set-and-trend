#!/usr/bin/env python3
"""
SAF Backtest Equity Simulator
- Loads trade_analysis.csv
- Filters by various criteria
- Simulates equity curves at different risk levels
- Monte Carlo analysis (1000 shuffles)
- Reports frequency/WR/edge tradeoffs
"""

import csv
import random
import math
import sys
from collections import defaultdict
from datetime import datetime

# ── Load trades ──────────────────────────────────────────────────

def load_trades(path="trade_analysis.csv"):
    trades = []
    with open(path) as f:
        for row in csv.DictReader(f):
            row["score"] = int(row["score"])
            row["rMult"] = float(row["rMult"])
            row["rr"] = float(row["rr"])
            row["engulfing"] = int(row["engulfing"])
            row["date_obj"] = datetime.strptime(row["date"], "%Y-%m-%d")
            trades.append(row)
    trades.sort(key=lambda t: t["date_obj"])
    return trades


# ── Filtering ────────────────────────────────────────────────────

def apply_filter(trades, name):
    """Return (filtered_trades, description) for a named config."""
    configs = {
        "all": (lambda t: True, "All trades (no filter)"),
        "no_engulfing": (lambda t: t["engulfing"] == 0, "No engulfing"),
        "dedicated_only": (lambda t: t["score"] == 0, "Dedicated strategies only (score=0)"),
        "checklist_9+": (lambda t: t["score"] >= 9 and t["engulfing"] == 0, "Checklist ≥9, no engulfing"),
        "checklist_10+": (lambda t: t["score"] >= 10 and t["engulfing"] == 0, "Checklist ≥10, no engulfing"),
        "checklist_11+": (lambda t: t["score"] >= 11 and t["engulfing"] == 0, "Checklist ≥11, no engulfing"),
        "dedicated+10": (lambda t: (t["score"] == 0 or t["score"] >= 10) and t["engulfing"] == 0,
                         "Dedicated + checklist ≥10, no engulfing"),
        "dedicated+11": (lambda t: (t["score"] == 0 or t["score"] >= 11) and t["engulfing"] == 0,
                         "Dedicated + checklist ≥11, no engulfing"),
        "high_rr": (lambda t: t["rr"] >= 3.0 and t["engulfing"] == 0, "RR≥3.0, no engulfing"),
        "best_items": (lambda t: t["engulfing"] == 0 and (
            int(t.get("w1_pattern", 0)) == 1 or int(t.get("d1_structure_rejection", 0)) == 1
        ), "w1_pattern OR d1_structure_rejection, no engulfing"),
    }
    fn, desc = configs[name]
    return [t for t in trades if fn(t)], desc


# ── Stats ────────────────────────────────────────────────────────

def calc_stats(trades):
    if not trades:
        return None
    n = len(trades)
    wins = sum(1 for t in trades if t["result"] == "win")
    losses = sum(1 for t in trades if t["result"] == "loss")
    timeouts = sum(1 for t in trades if t["result"] == "timeout")
    wr = wins / n
    rmults = [t["rMult"] for t in trades]
    exp = sum(rmults) / n
    total_r = sum(rmults)

    # Date span
    d0 = trades[0]["date_obj"]
    d1 = trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)
    per_month = n / (years * 12)
    per_year = n / years

    # Max drawdown in R
    peak = 0
    cum = 0
    max_dd = 0
    for r in rmults:
        cum += r
        if cum > peak:
            peak = cum
        dd = peak - cum
        if dd > max_dd:
            max_dd = dd

    return {
        "trades": n, "wins": wins, "losses": losses, "timeouts": timeouts,
        "wr": wr, "exp": exp, "total_r": total_r, "max_dd_r": max_dd,
        "years": years, "per_month": per_month, "per_year": per_year,
        "avg_rr": sum(t["rr"] for t in trades) / n,
    }


# ── Equity simulation (compounding) ─────────────────────────────

def simulate_equity(rmults, start_capital, risk_pct):
    """Simulate compounding equity curve. Returns (equity_curve, stats)."""
    equity = start_capital
    peak = start_capital
    max_dd_pct = 0
    max_dd_dollar = 0
    curve = [equity]

    for r in rmults:
        risk_amount = equity * risk_pct
        pnl = risk_amount * r
        equity += pnl
        if equity <= 0:
            equity = 0
            curve.append(0)
            return curve, {"final": 0, "max_dd_pct": 1.0, "max_dd_dollar": start_capital, "busted": True}
        curve.append(equity)
        if equity > peak:
            peak = equity
        dd_pct = (peak - equity) / peak
        dd_dollar = peak - equity
        if dd_pct > max_dd_pct:
            max_dd_pct = dd_pct
        if dd_dollar > max_dd_dollar:
            max_dd_dollar = dd_dollar

    return curve, {
        "final": equity,
        "max_dd_pct": max_dd_pct,
        "max_dd_dollar": max_dd_dollar,
        "busted": False,
    }


# ── Monte Carlo ──────────────────────────────────────────────────
# NOTE: With percentage-based compounding, final equity is PATH-INDEPENDENT
# (multiplication is commutative). The Monte Carlo value is in DRAWDOWN
# distribution — same final balance but very different max drawdowns and
# losing streak experiences depending on trade order.

def monte_carlo(rmults, start_capital, risk_pct, n_sims=2000):
    """Shuffle trade order N times. Focus: drawdown distribution + losing streaks."""
    max_dds = []
    max_consec_losses_list = []
    underwater_periods = []  # longest streak below peak (in trades)
    half_equity_hits = 0  # how often equity drops below 50% of peak

    for _ in range(n_sims):
        shuffled = rmults[:]
        random.shuffle(shuffled)

        equity = start_capital
        peak = start_capital
        max_dd_pct = 0
        consec_loss = 0
        max_consec_loss = 0
        underwater = 0
        max_underwater = 0

        for r in shuffled:
            equity += equity * risk_pct * r
            equity = max(equity, 0.01)  # floor to avoid zero

            if equity > peak:
                peak = equity
                if underwater > max_underwater:
                    max_underwater = underwater
                underwater = 0
            else:
                underwater += 1

            dd_pct = (peak - equity) / peak
            if dd_pct > max_dd_pct:
                max_dd_pct = dd_pct

            if r < 0:
                consec_loss += 1
                if consec_loss > max_consec_loss:
                    max_consec_loss = consec_loss
            else:
                consec_loss = 0

        if underwater > max_underwater:
            max_underwater = underwater

        max_dds.append(max_dd_pct)
        max_consec_losses_list.append(max_consec_loss)
        underwater_periods.append(max_underwater)
        if max_dd_pct >= 0.50:
            half_equity_hits += 1

    max_dds.sort()
    max_consec_losses_list.sort()
    underwater_periods.sort()

    # Final equity (same for all shuffles — commutative)
    final_equity = start_capital
    for r in rmults:
        final_equity *= (1 + risk_pct * r)

    return {
        "final_equity": final_equity,
        "median_dd": max_dds[len(max_dds) // 2],
        "p75_dd": max_dds[int(len(max_dds) * 0.75)],
        "p90_dd": max_dds[int(len(max_dds) * 0.90)],
        "p95_dd": max_dds[int(len(max_dds) * 0.95)],
        "worst_dd": max_dds[-1],
        "median_consec_loss": max_consec_losses_list[len(max_consec_losses_list) // 2],
        "p95_consec_loss": max_consec_losses_list[int(len(max_consec_losses_list) * 0.95)],
        "worst_consec_loss": max_consec_losses_list[-1],
        "median_underwater": underwater_periods[len(underwater_periods) // 2],
        "p95_underwater": underwater_periods[int(len(underwater_periods) * 0.95)],
        "half_equity_pct": half_equity_hits / n_sims * 100,
    }


# ── Reporting ────────────────────────────────────────────────────

def print_header(text):
    print(f"\n{'='*70}")
    print(f"  {text}")
    print(f"{'='*70}")


def print_filter_stats(trades, desc):
    stats = calc_stats(trades)
    if not stats:
        print(f"  {desc}: NO TRADES")
        return stats
    print(f"\n  {desc}")
    print(f"  {'─'*50}")
    print(f"  Trades: {stats['trades']}  |  {stats['per_month']:.1f}/mo  |  {stats['per_year']:.0f}/yr  |  {stats['years']:.1f} years")
    print(f"  W/L/T:  {stats['wins']}/{stats['losses']}/{stats['timeouts']}  |  WR: {stats['wr']*100:.1f}%")
    print(f"  Exp: {stats['exp']:+.2f}R  |  Total PnL: {stats['total_r']:+.1f}R  |  MaxDD: {stats['max_dd_r']:.1f}R")
    print(f"  Avg RR: {stats['avg_rr']:.2f}")
    return stats


def print_equity_table(rmults, start_capital, risk_levels):
    print(f"\n  Risk Sizing Comparison (${start_capital:,.0f} start, compounding):")
    print(f"  {'Risk':>6}  {'Final':>12}  {'MaxDD%':>8}  {'MaxDD$':>10}  {'CAGR':>8}  {'Busted':>7}")
    print(f"  {'─'*6}  {'─'*12}  {'─'*8}  {'─'*10}  {'─'*8}  {'─'*7}")

    for risk in risk_levels:
        _, stats = simulate_equity(rmults, start_capital, risk)
        if stats["busted"]:
            print(f"  {risk*100:5.1f}%  {'BUSTED':>12}  {stats['max_dd_pct']*100:7.1f}%  {'—':>10}  {'—':>8}  {'YES':>7}")
        else:
            years = max(len(rmults) / 50, 1)  # rough estimate
            cagr = (stats["final"] / start_capital) ** (1 / years) - 1 if stats["final"] > 0 else 0
            print(f"  {risk*100:5.1f}%  ${stats['final']:>10,.0f}  {stats['max_dd_pct']*100:7.1f}%  ${stats['max_dd_dollar']:>8,.0f}  {cagr*100:7.1f}%  {'No':>7}")


def print_monte_carlo(rmults, start_capital, risk_levels, n_sims=2000):
    print(f"\n  Monte Carlo ({n_sims} sims) — Drawdown & Streak Risk by Trade Order:")
    print(f"  (Final equity is always the same — multiplication is commutative)")
    print(f"  {'Risk':>6}  {'Final$':>12}  {'MedDD%':>8}  {'P90DD%':>8}  {'P95DD%':>8}  {'WorstDD':>8}  {'≥50%DD':>7}  {'MaxLoss':>8}  {'UW trades':>10}")
    print(f"  {'─'*6}  {'─'*12}  {'─'*8}  {'─'*8}  {'─'*8}  {'─'*8}  {'─'*7}  {'─'*8}  {'─'*10}")

    for risk in risk_levels:
        mc = monte_carlo(rmults, start_capital, risk, n_sims)
        print(f"  {risk*100:5.1f}%  ${mc['final_equity']:>10,.0f}  "
              f"{mc['median_dd']*100:7.1f}%  {mc['p90_dd']*100:7.1f}%  {mc['p95_dd']*100:7.1f}%  "
              f"{mc['worst_dd']*100:7.1f}%  {mc['half_equity_pct']:5.1f}%  "
              f"{mc['p95_consec_loss']:>6}L  {mc['p95_underwater']:>8} tr")
    print(f"  Legend: MaxLoss=P95 consecutive losses, UW=P95 underwater period (trades below peak)")


def print_monthly_frequency_vs_wr(trades):
    """Show the tradeoff: as you demand higher frequency, what WR do you get?"""
    print(f"\n  Frequency vs Win Rate Tradeoff:")
    print(f"  {'Target':>12}  {'Months Hit':>12}  {'Avg WR':>8}  {'Avg Exp':>8}  {'Feasible':>10}")
    print(f"  {'─'*12}  {'─'*12}  {'─'*8}  {'─'*8}  {'─'*10}")

    monthly = defaultdict(list)
    for t in trades:
        ym = t["date"][:7]
        monthly[ym].append(t)

    for target in [2, 4, 6, 8, 10, 12]:
        qualifying = {ym: ts for ym, ts in monthly.items() if len(ts) >= target}
        if not qualifying:
            print(f"  {target:>8}/mo   {'0':>12}  {'—':>8}  {'—':>8}  {'No':>10}")
            continue
        all_trades_in_qual = [t for ts in qualifying.values() for t in ts]
        wr = sum(1 for t in all_trades_in_qual if t["result"] == "win") / len(all_trades_in_qual)
        exp = sum(t["rMult"] for t in all_trades_in_qual) / len(all_trades_in_qual)
        pct = len(qualifying) / len(monthly) * 100
        feasible = "Yes" if wr >= 0.50 else "Close" if wr >= 0.40 else "No"
        print(f"  {target:>8}/mo   {len(qualifying):>5} ({pct:4.0f}%)  {wr*100:6.1f}%  {exp:+6.2f}R  {feasible:>10}")


def print_what_if_section(trades):
    """Hypothetical: what if we could boost WR to 50%+ and maintain frequency?"""
    print_header("WHAT-IF: Target 8-12 trades/mo at 50% WR")

    no_eng = [t for t in trades if t["engulfing"] == 0]
    stats = calc_stats(no_eng)

    print(f"\n  Current reality (no engulfing): {stats['per_month']:.1f}/mo, {stats['wr']*100:.1f}% WR, {stats['exp']:+.2f}R exp")
    print(f"\n  To reach 8-12/mo at 50% WR, you need to ADD trades.")
    print(f"  Possible paths:")
    print(f"    1. Add 5-10 more FX pairs (GBPJPY, EURJPY, AUDJPY, GBPAUD, EURGBP...)")
    print(f"    2. Add H1 entry timing (scan H4 for context, enter on H1)")
    print(f"    3. Add breakout/pullback strategies (currently only rejection-based)")
    print(f"    4. Remove London/NY session gate (allows Asian session entries)")
    print()

    # Realistic scenarios with Monte Carlo
    print(f"  ── Scenario Comparison (2000 MC sims each, $1,000 start) ──")
    print()

    scenarios = [
        ("CURRENT (3.4/mo, 37.7% WR, 2.29R win)", 0.377, 2.29, 3.4, 22),
        ("PATH A: +5 pairs (6/mo, 37.7% WR)", 0.377, 2.29, 6, 22),
        ("PATH B: +5 pairs + w1 filter (8/mo, 46% WR)", 0.46, 2.29, 8, 22),
        ("PATH C: +pairs + H1 entry (10/mo, 42% WR)", 0.42, 2.29, 10, 22),
        ("TARGET: 10/mo, 50% WR, 2.5R", 0.50, 2.50, 10, 22),
        ("DREAM: 12/mo, 55% WR, 2.5R", 0.55, 2.50, 12, 22),
    ]

    for desc, wr, avg_win, tpm, years in scenarios:
        n_trades = int(tpm * 12 * years)
        print(f"  {desc}")
        print(f"  {'Risk':>6}  {'Final$':>12}  {'MedDD%':>8}  {'P95DD%':>8}  {'≥50%DD':>7}  {'Streak':>7}  {'$/yr':>10}")
        print(f"  {'─'*6}  {'─'*12}  {'─'*8}  {'─'*8}  {'─'*7}  {'─'*7}  {'─'*10}")

        for risk in [0.01, 0.02, 0.03, 0.05]:
            # Generate synthetic trades matching the scenario
            synthetic = []
            for _ in range(n_trades):
                if random.random() < wr:
                    synthetic.append(avg_win)
                else:
                    synthetic.append(-1.0)
            mc = monte_carlo(synthetic, 1000, risk, 2000)
            annual = (mc["final_equity"] - 1000) / years if mc["final_equity"] > 1000 else 0
            print(f"  {risk*100:5.1f}%  ${mc['final_equity']:>10,.0f}  "
                  f"{mc['median_dd']*100:7.1f}%  {mc['p95_dd']*100:7.1f}%  "
                  f"{mc['half_equity_pct']:5.1f}%  "
                  f"{mc['p95_consec_loss']:>5}L  ${annual:>9,.0f}")
        print()


# ── Year-by-year breakdown ───────────────────────────────────────

def print_yearly_breakdown(trades, risk_pct, start_capital):
    print(f"\n  Year-by-Year at {risk_pct*100:.0f}% risk, ${start_capital:,.0f} start:")
    print(f"  {'Year':>6}  {'Trades':>7}  {'WR':>6}  {'PnL(R)':>8}  {'Equity':>10}  {'DD%':>7}")
    print(f"  {'─'*6}  {'─'*7}  {'─'*6}  {'─'*8}  {'─'*10}  {'─'*7}")

    yearly = defaultdict(list)
    for t in trades:
        yearly[t["date"][:4]].append(t)

    equity = start_capital
    peak = start_capital
    for year in sorted(yearly.keys()):
        yts = yearly[year]
        rmults = [t["rMult"] for t in yts]
        wr = sum(1 for t in yts if t["result"] == "win") / len(yts) * 100
        pnl_r = sum(rmults)
        for r in rmults:
            equity += equity * risk_pct * r
            equity = max(equity, 0)
        if equity > peak:
            peak = equity
        dd = (peak - equity) / peak * 100 if peak > 0 else 0
        print(f"  {year:>6}  {len(yts):>7}  {wr:5.1f}%  {pnl_r:>+7.1f}R  ${equity:>9,.0f}  {dd:5.1f}%")


# ── Main ─────────────────────────────────────────────────────────

def main():
    random.seed(42)
    trades = load_trades()
    start_capital = 1000
    risk_levels = [0.01, 0.02, 0.03, 0.05]

    # ── Section 1: Filter comparison ──
    print_header("FILTER COMPARISON — Which config gets closest to 8-12/mo + 50% WR?")

    filter_names = [
        "all", "no_engulfing", "dedicated_only", "checklist_9+",
        "checklist_10+", "dedicated+10", "dedicated+11", "best_items",
    ]
    best_config = None
    best_score = -999

    for name in filter_names:
        filtered, desc = apply_filter(trades, name)
        stats = print_filter_stats(filtered, f"[{name}] {desc}")
        if stats:
            # Score: reward WR closeness to 50% and frequency closeness to 10/mo
            wr_score = -abs(stats["wr"] - 0.50) * 100  # penalty for distance from 50%
            freq_score = -abs(stats["per_month"] - 10) * 2  # penalty for distance from 10/mo
            edge_score = stats["exp"] * 10  # reward positive edge
            total = wr_score + freq_score + edge_score
            if total > best_score and stats["exp"] > 0:
                best_score = total
                best_config = name

    if best_config:
        print(f"\n  >>> Closest to target: [{best_config}]")

    # ── Section 2: Frequency vs WR tradeoff ──
    print_header("FREQUENCY vs WIN RATE TRADEOFF (no engulfing)")
    no_eng = [t for t in trades if t["engulfing"] == 0]
    print_monthly_frequency_vs_wr(no_eng)

    # ── Section 3: Equity simulation on best configs ──
    for config_name in ["no_engulfing", "dedicated_only", "dedicated+10"]:
        filtered, desc = apply_filter(trades, config_name)
        if not filtered:
            continue
        rmults = [t["rMult"] for t in filtered]
        print_header(f"EQUITY SIM: [{config_name}] {desc}")
        stats = calc_stats(filtered)
        print(f"  {stats['trades']} trades | {stats['per_month']:.1f}/mo | WR {stats['wr']*100:.1f}% | Exp {stats['exp']:+.2f}R")
        print_equity_table(rmults, start_capital, risk_levels)
        print_monte_carlo(rmults, start_capital, risk_levels, n_sims=1000)
        print_yearly_breakdown(filtered, 0.02, start_capital)

    # ── Section 4: What-if analysis ──
    print_what_if_section(trades)

    # ── Section 5: The gap analysis ──
    print_header("GAP ANALYSIS — What's needed to hit 8-12/mo at 50% WR")

    no_eng = [t for t in trades if t["engulfing"] == 0]
    stats = calc_stats(no_eng)

    current_tpm = stats["per_month"]
    target_tpm = 10
    gap = target_tpm - current_tpm

    print(f"\n  Current: {current_tpm:.1f} trades/month, {stats['wr']*100:.1f}% WR")
    print(f"  Target:  {target_tpm} trades/month, 50% WR")
    print(f"  Gap:     +{gap:.1f} trades/month needed, +{50-stats['wr']*100:.1f}pp WR needed")
    print()
    print(f"  Frequency gap solutions (each adds ~2-4 trades/month):")
    print(f"    [ ] Add 5+ more FX pairs (GBPJPY, EURJPY, AUDJPY, GBPAUD, EURGBP)")
    print(f"    [ ] Add H1 entry strategies (scan H4 for context, enter on H1)")
    print(f"    [ ] Add breakout/pullback strategies (currently only rejection-based)")
    print(f"    [ ] Remove London/NY session gate (allows Asian session entries)")
    print()
    print(f"  Win rate gap solutions (each adds ~5-10pp WR):")
    print(f"    [ ] Only trade w1_pattern=1 signals (current best confluence item)")
    print(f"    [ ] Add higher-TF trend filter (monthly trend alignment)")
    print(f"    [ ] Tighter SL (0.3×ATR instead of 0.5×ATR) — improves WR but lowers RR")
    print(f"    [ ] Require d1_structure_rejection for all entries")
    print()

    # Quick sim: what if we ONLY take w1_pattern trades?
    w1p = [t for t in no_eng if int(t.get("w1_pattern", 0)) == 1]
    if w1p:
        ws = calc_stats(w1p)
        print(f"  Quick check — w1_pattern=1 only:")
        print(f"    {ws['trades']} trades, {ws['per_month']:.1f}/mo, WR={ws['wr']*100:.1f}%, Exp={ws['exp']:+.2f}R")

    d1sr = [t for t in no_eng if int(t.get("d1_structure_rejection", 0)) == 1]
    if d1sr:
        ds = calc_stats(d1sr)
        print(f"  Quick check — d1_structure_rejection=1 only:")
        print(f"    {ds['trades']} trades, {ds['per_month']:.1f}/mo, WR={ds['wr']*100:.1f}%, Exp={ds['exp']:+.2f}R")

    both = [t for t in no_eng if int(t.get("w1_pattern", 0)) == 1 and int(t.get("d1_structure_rejection", 0)) == 1]
    if both:
        bs = calc_stats(both)
        print(f"  Quick check — w1_pattern + d1_structure_rejection:")
        print(f"    {bs['trades']} trades, {bs['per_month']:.1f}/mo, WR={bs['wr']*100:.1f}%, Exp={bs['exp']:+.2f}R")

    print(f"\n{'='*70}")
    print(f"  BOTTOM LINE")
    print(f"{'='*70}")
    print(f"  Your current system is profitable (+{stats['total_r']:.0f}R over {stats['years']:.0f} years)")
    print(f"  but generates {current_tpm:.1f}/mo at {stats['wr']*100:.1f}% WR — not 8-12/mo at 50%.")
    print(f"  The edge is REAL. To scale frequency, add more pairs or strategies.")
    print(f"  To scale WR, filter harder on w1_pattern — but that cuts frequency.")
    print(f"  The optimal path is likely: more pairs + w1_pattern filter.")
    print(f"{'='*70}\n")


if __name__ == "__main__":
    main()
