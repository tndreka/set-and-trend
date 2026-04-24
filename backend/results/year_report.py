#!/usr/bin/env python3
"""
SAF Strategy — Year-by-Year Account Growth Report
Shows every trade, monthly balance, and account growth at realistic capitals.
Focus: "What does my account look like each month?"
"""

import csv
import math
from collections import OrderedDict
from datetime import datetime

# ── Broker costs ──
SPREADS = {"EURUSD":1.0,"GBPUSD":1.3,"AUDUSD":1.2,"NZDUSD":1.5,"USDCAD":1.5,"USDCHF":1.3,"USDJPY":1.0}
SLIPPAGE = {"EURUSD":0.3,"GBPUSD":0.5,"AUDUSD":0.4,"NZDUSD":0.5,"USDCAD":0.5,"USDCHF":0.4,"USDJPY":0.3}
SWAP_NIGHT = {"EURUSD":8,"GBPUSD":9,"AUDUSD":7,"NZDUSD":7,"USDCAD":6,"USDCHF":5,"USDJPY":7}
PIP_SIZE = {"EURUSD":0.0001,"GBPUSD":0.0001,"AUDUSD":0.0001,"NZDUSD":0.0001,"USDCAD":0.0001,"USDCHF":0.0001,"USDJPY":0.01}
COMM = 7.0; PIPV = 10.0; SWAP_M = 7/5
HOLD = {"win":4,"loss":2,"timeout":10}

STRONG = ["d1_in_favor","w1_pattern","d1_touching_ema","h4_in_favor","d1_aoi"]
USEFUL = ["w1_candle_rejection","d1_structure_rejection","d1_pattern"]
ALL_ITEMS = STRONG + USEFUL + ["w1_in_favor","psych_level","h4_candle_rejection","w1_touching_ema","h4_ema_touch","engulfing"]

MONTH_NAMES = {"01":"Jan","02":"Feb","03":"Mar","04":"Apr","05":"May","06":"Jun",
               "07":"Jul","08":"Aug","09":"Sep","10":"Oct","11":"Nov","12":"Dec"}


def load_trades(path="trade_analysis.csv"):
    trades = []
    with open(path) as f:
        for row in csv.DictReader(f):
            row["score"] = int(row["score"])
            row["rMult"] = float(row["rMult"])
            row["rr"] = float(row["rr"])
            row["entry_price"] = float(row["entry"])
            row["sl_price"] = float(row["sl"])
            row["engulfing"] = int(row["engulfing"])
            row["date_obj"] = datetime.strptime(row["date"], "%Y-%m-%d")
            for it in ALL_ITEMS:
                row[it] = int(row.get(it, 0))
            trades.append(row)
    trades.sort(key=lambda t: t["date_obj"])
    return trades


def qs(t):
    return sum(2*t[i] for i in STRONG) + sum(t[i] for i in USEFUL)


def tiered_filter(t):
    return t["engulfing"] == 0 and (t["score"] == 0 or qs(t) >= 10)


def cost_R(t):
    sym = t["symbol"]
    risk_pips = abs(t["entry_price"] - t["sl_price"]) / PIP_SIZE[sym]
    if risk_pips <= 0: return 0
    s = SPREADS[sym] + SLIPPAGE[sym] + COMM/PIPV
    sw = SWAP_NIGHT[sym] * HOLD.get(t["result"],4) * SWAP_M / PIPV
    return (s + sw) / risk_pips


# ══════════════════════════════════════════════════════════════════
#  SIMULATE ONE YEAR
# ══════════════════════════════════════════════════════════════════

def sim_year(trades, year, start_eq, risk_pct):
    """Simulate a single year, return trade log and stats."""
    yt = [t for t in trades if t["date"][:4] == year]
    eq = start_eq
    peak = start_eq
    log = []

    for t in yt:
        cr = cost_R(t)
        adj = t["rMult"] - cr
        risk_amt = eq * risk_pct
        pnl_g = t["rMult"] * risk_amt
        pnl_c = cr * risk_amt
        pnl_n = adj * risk_amt
        eq += pnl_n
        eq = max(eq, 0.01)
        if eq > peak: peak = eq
        dd = (peak - eq) / peak

        log.append({
            "date": t["date"], "sym": t["symbol"], "dir": t["direction"],
            "res": t["result"], "rr": t["rr"],
            "orig_R": t["rMult"], "cost_R": cr, "adj_R": adj,
            "risk": risk_amt, "gross": pnl_g, "cost": pnl_c, "net": pnl_n,
            "eq": eq, "dd": dd,
        })

    return log


def print_year(year, log, start_eq, risk_pct, cap_label):
    """Print full year report."""
    if not log:
        return

    end_eq = log[-1]["eq"]
    ret = (end_eq / start_eq - 1) * 100
    wins = sum(1 for e in log if e["res"] == "win")
    losses = sum(1 for e in log if e["res"] == "loss")
    timeouts = sum(1 for e in log if e["res"] == "timeout")
    total_g = sum(e["gross"] for e in log)
    total_c = sum(e["cost"] for e in log)
    total_n = sum(e["net"] for e in log)
    max_dd = max(e["dd"] for e in log) * 100

    print()
    print(f"  ╔══════════════════════════════════════════════════════════════════════════════════════╗")
    print(f"  ║  YEAR {year}  |  Start: ${start_eq:,.0f}  |  Risk: {risk_pct*100:.0f}%  |  {cap_label:<20}        ║")
    print(f"  ╚══════════════════════════════════════════════════════════════════════════════════════╝")
    print()

    # Trade-by-trade with monthly groups
    current_month = None
    month_trades = []

    for i, e in enumerate(log):
        m = e["date"][5:7]

        # Print monthly summary when month changes
        if m != current_month and current_month is not None:
            _print_month_summary(current_month, month_trades)
            month_trades = []

        if m != current_month:
            mname = MONTH_NAMES.get(m, m)
            print(f"  ┌── {mname} {year} ──────────────────────────────────────────────────────────────────┐")
            print(f"  │  {'#':>3} {'Date':>6} {'Pair':<7} {'Dir':<5} {'Result':<5} {'RR':>4} "
                  f"{'Gross':>9} {'Costs':>8} {'Net P&L':>9} {'Balance':>11} {'DD':>5} │")
            print(f"  │  {'─'*3} {'─'*6} {'─'*7} {'─'*5} {'─'*5} {'─'*4} "
                  f"{'─'*9} {'─'*8} {'─'*9} {'─'*11} {'─'*5} │")
            current_month = m

        res_sym = {"win":"✓ W","loss":"✗ L","timeout":"~ T"}[e["res"]]
        dd_str = f"{e['dd']*100:.1f}%" if e["dd"] > 0.01 else "  —"

        print(f"  │  {i+1:>3} {e['date'][5:]:>6} {e['sym']:<7} {e['dir']:<5} {res_sym:<5} "
              f"{e['rr']:>3.1f} ${e['gross']:>8,.0f} ${e['cost']:>7,.0f} "
              f"${e['net']:>8,.0f} ${e['eq']:>10,.0f} {dd_str:>5} │")

        month_trades.append(e)

    # Final month summary
    if month_trades:
        _print_month_summary(current_month, month_trades)

    # Year totals
    print()
    print(f"  ╔══════════════════════════════════════════════════════════════════════════════════════╗")
    print(f"  ║  {year} SUMMARY                                                                      ║")
    print(f"  ╠══════════════════════════════════════════════════════════════════════════════════════╣")
    print(f"  ║  Trades:    {len(log):>4}   (Wins: {wins}, Losses: {losses}, Timeouts: {timeouts})                        ║")
    print(f"  ║  Win Rate:  {wins/len(log)*100:>5.1f}%                                                               ║")
    print(f"  ║                                                                                    ║")
    print(f"  ║  Gross P&L:     ${total_g:>+11,.0f}                                                      ║")
    print(f"  ║  Broker Costs:  ${total_c:>11,.0f}  ({total_c/total_g*100:.0f}% of gross)                               ║" if total_g > 0 else
          f"  ║  Broker Costs:  ${total_c:>11,.0f}                                                       ║")
    print(f"  ║  Net Profit:    ${total_n:>+11,.0f}                                                      ║")
    print(f"  ║                                                                                    ║")
    print(f"  ║  Start Balance: ${start_eq:>11,.0f}                                                      ║")
    print(f"  ║  End Balance:   ${end_eq:>11,.0f}                                                      ║")
    print(f"  ║  Year Return:   {ret:>+10.1f}%                                                         ║")
    print(f"  ║  Max Drawdown:  {max_dd:>10.1f}%                                                         ║")
    print(f"  ╚══════════════════════════════════════════════════════════════════════════════════════╝")

    return end_eq


def _print_month_summary(month, trades):
    mname = MONTH_NAMES.get(month, month)
    wins = sum(1 for e in trades if e["res"] == "win")
    total_n = sum(e["net"] for e in trades)
    total_c = sum(e["cost"] for e in trades)
    sign = "+" if total_n >= 0 else ""
    print(f"  │  {'':>3} {'':>6}                                                                     │")
    print(f"  │  {mname} total: {len(trades)} trades, {wins} wins  │  "
          f"Costs: ${total_c:,.0f}  │  Net: ${sign}{total_n:,.0f}  │  "
          f"Bal: ${trades[-1]['eq']:,.0f}   │")
    print(f"  └─────────────────────────────────────────────────────────────────────────────────────┘")
    print()


# ══════════════════════════════════════════════════════════════════
#  MULTI-YEAR GROWTH VIEW
# ══════════════════════════════════════════════════════════════════

def print_multi_year_growth(trades, start_capital, risk_pct, years, label):
    """Simulate multiple years showing account growth."""
    print()
    print(f"  ╔══════════════════════════════════════════════════════════════════════════════════════╗")
    print(f"  ║  ACCOUNT GROWTH  |  ${start_capital:,} start  |  {risk_pct*100:.0f}% risk  |  {label:<25}    ║")
    print(f"  ╠══════════════════════════════════════════════════════════════════════════════════════╣")
    print(f"  ║  {'Year':>6} {'Trades':>7} {'W/L/T':>8} {'WR':>6} {'Gross':>10} {'Costs':>9} "
          f"{'Net':>10} {'Balance':>11} {'Return':>8} ║")
    print(f"  ║  {'─'*6} {'─'*7} {'─'*8} {'─'*6} {'─'*10} {'─'*9} {'─'*10} {'─'*11} {'─'*8} ║")

    eq = start_capital
    total_costs = 0
    total_net = 0

    for year in years:
        log = sim_year(trades, year, eq, risk_pct)
        if not log:
            continue
        wins = sum(1 for e in log if e["res"] == "win")
        losses = sum(1 for e in log if e["res"] == "loss")
        timeouts = sum(1 for e in log if e["res"] == "timeout")
        gross = sum(e["gross"] for e in log)
        costs = sum(e["cost"] for e in log)
        net = sum(e["net"] for e in log)
        end_eq = log[-1]["eq"]
        ret = (end_eq / eq - 1) * 100

        wlt = f"{wins}/{losses}/{timeouts}"
        wr = wins / len(log) * 100
        total_costs += costs
        total_net += net

        print(f"  ║  {year:>6} {len(log):>7} {wlt:>8} {wr:5.1f}% ${gross:>9,.0f} ${costs:>8,.0f} "
              f"${net:>9,.0f} ${end_eq:>10,.0f} {ret:>+7.1f}% ║")
        eq = end_eq

    cagr = (eq / start_capital) ** (1 / len(years)) - 1 if eq > start_capital else 0
    print(f"  ╠══════════════════════════════════════════════════════════════════════════════════════╣")
    print(f"  ║  TOTAL   {'':>7} {'':>8} {'':>6} {'':>10} ${total_costs:>8,.0f} "
          f"${total_net:>9,.0f} ${eq:>10,.0f} {cagr*100:>+7.1f}% ║")
    print(f"  ║  {'':>6} {'':>7} {'':>8} {'':>6} {'':>10} {'':>9} {'':>10} {'':>11} CAGR     ║")
    print(f"  ╚══════════════════════════════════════════════════════════════════════════════════════╝")
    return eq


# ══════════════════════════════════════════════════════════════════
#  MONTHLY OVERVIEW (condensed)
# ══════════════════════════════════════════════════════════════════

def print_monthly_overview(trades, start_capital, risk_pct, years, label):
    """One-line-per-month view across multiple years."""
    print()
    print(f"  ╔══════════════════════════════════════════════════════════════════════════════════════╗")
    print(f"  ║  MONTHLY OVERVIEW  |  ${start_capital:,} start  |  {risk_pct*100:.0f}% risk  |  {label:<22}    ║")
    print(f"  ╠══════════════════════════════════════════════════════════════════════════════════════╣")
    print(f"  ║  {'Month':>8} {'Trades':>7} {'Wins':>5} {'WR':>6} {'Costs':>8} {'Net P&L':>10} "
          f"{'Balance':>11} {'MoM%':>7} {'Chart':>10} ║")
    print(f"  ║  {'─'*8} {'─'*7} {'─'*5} {'─'*6} {'─'*8} {'─'*10} {'─'*11} {'─'*7} {'─'*10} ║")

    eq = start_capital
    prev_eq = start_capital

    for year in years:
        log = sim_year(trades, year, eq, risk_pct)
        if not log:
            continue

        # Group by month
        months = OrderedDict()
        for e in log:
            ym = e["date"][:7]
            if ym not in months:
                months[ym] = []
            months[ym].append(e)

        for ym, month_log in months.items():
            wins = sum(1 for e in month_log if e["res"] == "win")
            wr = wins / len(month_log) * 100
            costs = sum(e["cost"] for e in month_log)
            net = sum(e["net"] for e in month_log)
            eq = month_log[-1]["eq"]
            mom = (eq / prev_eq - 1) * 100 if prev_eq > 0 else 0

            # Mini bar chart
            bars = int(abs(mom) / 5)
            if mom >= 0:
                chart = "█" * min(bars, 8) + "░" * max(0, 8 - bars)
            else:
                chart = "░" * max(0, 8 - bars) + "▓" * min(bars, 8)

            sign = "+" if net >= 0 else ""
            mname = MONTH_NAMES.get(ym[5:7], ym[5:7])
            print(f"  ║  {mname} {ym[:4]} {len(month_log):>5} {wins:>5} {wr:5.1f}% ${costs:>7,.0f} "
                  f"${sign}{net:>8,.0f} ${eq:>10,.0f} {mom:>+6.1f}% {chart:>10} ║")

            prev_eq = eq

        # Year separator
        print(f"  ║  {'─'*86} ║")

    print(f"  ╚══════════════════════════════════════════════════════════════════════════════════════╝")


# ══════════════════════════════════════════════════════════════════
#  CAPITAL SCENARIOS
# ══════════════════════════════════════════════════════════════════

def print_capital_scenarios(trades, years):
    """Compare different starting capitals and risk levels."""
    print()
    print(f"  ╔══════════════════════════════════════════════════════════════════════════════════════╗")
    print(f"  ║  STARTING CAPITAL × RISK SCENARIOS  ({years[0]}–{years[-1]})                              ║")
    print(f"  ╠══════════════════════════════════════════════════════════════════════════════════════╣")
    print(f"  ║  {'Capital':>10} {'Risk':>5} {'End Balance':>13} {'Net Profit':>12} {'Costs':>10} "
          f"{'Cost%':>6} {'CAGR':>7} {'MaxDD':>6} ║")
    print(f"  ║  {'─'*10} {'─'*5} {'─'*13} {'─'*12} {'─'*10} {'─'*6} {'─'*7} {'─'*6} ║")

    for cap in [500, 1000, 2000, 5000, 10000, 25000]:
        for risk in [0.01, 0.02, 0.03]:
            eq = cap
            total_c = 0
            total_g = 0
            max_dd = 0

            for year in years:
                log = sim_year(trades, year, eq, risk)
                if not log: continue
                total_g += sum(e["gross"] for e in log)
                total_c += sum(e["cost"] for e in log)
                dd = max(e["dd"] for e in log)
                if dd > max_dd: max_dd = dd
                eq = log[-1]["eq"]

            net = eq - cap
            cpct = total_c / total_g * 100 if total_g > 0 else 0
            cagr = (eq / cap) ** (1 / len(years)) - 1 if eq > cap else 0
            marker = " ◄" if cap == 1000 and risk == 0.02 else ""

            print(f"  ║  ${cap:>8,} {risk*100:4.0f}% ${eq:>12,.0f} ${net:>11,.0f} ${total_c:>9,.0f} "
                  f"{cpct:5.1f}% {cagr*100:6.1f}% {max_dd*100:5.1f}%{marker} ║")
        print(f"  ║  {'':>10} {'':>5} {'':>13} {'':>12} {'':>10} {'':>6} {'':>7} {'':>6} ║")

    print(f"  ╚══════════════════════════════════════════════════════════════════════════════════════╝")


# ══════════════════════════════════════════════════════════════════
#  MAIN
# ══════════════════════════════════════════════════════════════════

def main():
    trades_raw = load_trades()
    trades = [t for t in trades_raw if tiered_filter(t)]

    all_years = sorted(set(t["date"][:4] for t in trades))
    recent_years = [y for y in all_years if y >= "2020"]

    print()
    print("  ████████████████████████████████████████████████████████████████████████████████████████")
    print("  ██                                                                                  ██")
    print("  ██   SAF STRATEGY — REALISTIC ACCOUNT GROWTH REPORT                                 ██")
    print("  ██   Config: TIERED (strong×2 + useful, ≥10 + dedicated, no engulfing)              ██")
    print("  ██   Broker: ECN (spreads + $7/lot commission + swap + slippage)                     ██")
    print("  ██                                                                                  ██")
    print("  ████████████████████████████████████████████████████████████████████████████████████████")

    # ── REPORT 1: Multi-year growth at different capitals ──
    for cap in [1000, 5000, 10000]:
        print_multi_year_growth(trades, cap, 0.02, recent_years, f"TIERED @ ${cap:,}")

    # ── REPORT 2: Monthly overview (last 3 years) ──
    last3 = [y for y in all_years if y >= "2023"]
    for cap in [1000, 10000]:
        # Need to grow capital from 2004 to 2023 first
        eq = cap
        for y in all_years:
            if y >= "2023": break
            log = sim_year(trades, y, eq, 0.02)
            if log: eq = log[-1]["eq"]
        print_monthly_overview(trades, eq, 0.02, last3, f"TIERED (grown from ${cap:,})")

    # ── REPORT 3: Full trade-by-trade for each recent year ──
    # Grow from $1,000 to present
    eq = 1000
    for y in all_years:
        if y >= "2023": break
        log = sim_year(trades, y, eq, 0.02)
        if log: eq = log[-1]["eq"]

    for year in ["2023", "2024", "2025"]:
        log = sim_year(trades, year, eq, 0.02)
        if log:
            eq = print_year(year, log, eq, 0.02, f"grown from $1,000")

    # ── REPORT 4: Fresh start scenarios (Year 1 with no history) ──
    print()
    print()
    print("  ████████████████████████████████████████████████████████████████████████████████████████")
    print("  ██  FRESH START: What if you started trading TODAY with these recent-year stats?     ██")
    print("  ████████████████████████████████████████████████████████████████████████████████████████")

    # Use 2023-2025 average stats to project Year 1
    recent_trades = [t for t in trades if t["date"] >= "2023"]
    n = len(recent_trades)
    wins = sum(1 for t in recent_trades if t["result"] == "win")
    wr = wins / n * 100
    avg_rmult = sum(t["rMult"] for t in recent_trades) / n
    avg_cost = sum(cost_R(t) for t in recent_trades) / n
    avg_adj = avg_rmult - avg_cost
    tpm = n / 36  # 3 years = 36 months

    print(f"\n  Recent stats (2023–2025): {n} trades, {tpm:.1f}/month, WR {wr:.1f}%, "
          f"Exp {avg_adj:+.2f}R (after costs)")
    print()

    for cap in [500, 1000, 2000, 5000, 10000]:
        # Simulate Year 1 using 2023 data, Year 2 using 2024, Year 3 using 2025
        print(f"\n  Starting with ${cap:,}:")
        eq = cap
        for yr_label, yr_data in [("Year 1 (2023 trades)", "2023"),
                                   ("Year 2 (2024 trades)", "2024"),
                                   ("Year 3 (2025 trades)", "2025")]:
            log = sim_year(trades, yr_data, eq, 0.02)
            if not log: continue
            wins_y = sum(1 for e in log if e["res"] == "win")
            net_y = sum(e["net"] for e in log)
            costs_y = sum(e["cost"] for e in log)
            end = log[-1]["eq"]
            ret = (end / eq - 1) * 100
            print(f"    {yr_label}: {len(log)} trades, {wins_y} wins, "
                  f"costs ${costs_y:,.0f}, net ${net_y:+,.0f}, "
                  f"${eq:,.0f} → ${end:,.0f} ({ret:+.1f}%)")
            eq = end
        print(f"    ── 3-year result: ${cap:,} → ${eq:,.0f} ({(eq/cap-1)*100:+.1f}%)")

    # ── REPORT 5: Capital × Risk matrix ──
    print_capital_scenarios(trades, recent_years)

    print()
    print("  ████████████████████████████████████████████████████████████████████████████████████████")
    print("  ██  REPORTS COMPLETE                                                                ██")
    print("  ████████████████████████████████████████████████████████████████████████████████████████")
    print()


if __name__ == "__main__":
    main()
