#!/usr/bin/env python3
"""
SAF Strategy — Realistic Growth Reports
Monthly balance, per-trade log, annual projections, multiple scenarios.
Uses real broker costs. Focuses on practical "what does my account look like" view.
"""

import csv
import random
import math
from collections import defaultdict, OrderedDict
from datetime import datetime

random.seed(42)

# ══════════════════════════════════════════════════════════════════
#  BROKER COSTS (ECN tier)
# ════════���═══════════════════════════��═════════════════════════════

SPREADS = {"EURUSD":1.0,"GBPUSD":1.3,"AUDUSD":1.2,"NZDUSD":1.5,"USDCAD":1.5,"USDCHF":1.3,"USDJPY":1.0}
SLIPPAGE = {"EURUSD":0.3,"GBPUSD":0.5,"AUDUSD":0.4,"NZDUSD":0.5,"USDCAD":0.5,"USDCHF":0.4,"USDJPY":0.3}
SWAP_NIGHT = {"EURUSD":-8,"GBPUSD":-9,"AUDUSD":-7,"NZDUSD":-7,"USDCAD":-6,"USDCHF":-5,"USDJPY":-7}
PIP_SIZE = {"EURUSD":0.0001,"GBPUSD":0.0001,"AUDUSD":0.0001,"NZDUSD":0.0001,"USDCAD":0.0001,"USDCHF":0.0001,"USDJPY":0.01}
PIP_VALUE = 10.0
COMMISSION = 7.0
HOLD_DAYS = {"win": 4, "loss": 2, "timeout": 10}
SWAP_MULT = 7 / 5

STRONG = ["d1_in_favor", "w1_pattern", "d1_touching_ema", "h4_in_favor", "d1_aoi"]
USEFUL = ["w1_candle_rejection", "d1_structure_rejection", "d1_pattern"]
ALL_16 = STRONG + USEFUL + ["w1_in_favor","psych_level","h4_candle_rejection","w1_touching_ema","h4_ema_touch","engulfing"]


# ═══════════���═════════════════════════��════════════════════════════
#  DATA
# ═════════════════════════════════════��════════════════════════════

def load_trades(path="trade_analysis.csv"):
    trades = []
    with open(path) as f:
        for row in csv.DictReader(f):
            row["score"] = int(row["score"])
            row["rMult"] = float(row["rMult"])
            row["rr"] = float(row["rr"])
            row["entry_price"] = float(row["entry"])
            row["sl_price"] = float(row["sl"])
            row["tp_price"] = float(row["tp"])
            row["engulfing"] = int(row["engulfing"])
            row["date_obj"] = datetime.strptime(row["date"], "%Y-%m-%d")
            for item in ALL_16:
                row[item] = int(row.get(item, 0))
            trades.append(row)
    trades.sort(key=lambda t: t["date_obj"])
    return trades


def quality_score(t):
    return sum(2 * t[i] for i in STRONG) + sum(t[i] for i in USEFUL)


# ═══════��════════════════════════��═════════════════════════════════
#  CONFIGS TO TEST
# ════��══════════════════════��══════════════════════════════════════

CONFIGS = OrderedDict([
    ("A: ALL no eng", lambda t: t["engulfing"] == 0),
    ("B: TIERED ≥10+ded no eng", lambda t: t["engulfing"] == 0 and (t["score"] == 0 or quality_score(t) >= 10)),
    ("C: STRONG≥4+ded no eng", lambda t: t["engulfing"] == 0 and (t["score"] == 0 or quality_score(t) >= 4)),
    ("D: SNIPER ≥11+ded", lambda t: t["score"] == 0 or quality_score(t) >= 11),
    ("E: DEDICATED only", lambda t: t["score"] == 0),
])


# ══════════���═════════════════════════��═════════════════════════════
#  TRADE COST CALC
# ══════════════��═══════════════════════════════════════════════════

def trade_cost_R(trade):
    """Return cost of this trade in R (fraction of risk)."""
    sym = trade["symbol"]
    risk_pips = abs(trade["entry_price"] - trade["sl_price"]) / PIP_SIZE[sym]
    if risk_pips <= 0:
        return 0
    spread = SPREADS[sym]
    slip = SLIPPAGE[sym]
    comm_pips = COMMISSION / PIP_VALUE  # ~0.7 pips
    hold = HOLD_DAYS.get(trade["result"], 4)
    swap_pips = abs(SWAP_NIGHT[sym]) * hold * SWAP_MULT / PIP_VALUE
    total_pips = spread + slip + comm_pips + swap_pips
    return total_pips / risk_pips


# ═════════════════════════════════════════════��════════════════════
#  SIMULATION ENGINE
# ════════════���═══════════════════��═════════════════════════════════

def simulate(trades, capital, risk_pct):
    """Walk trades chronologically, return detailed log."""
    equity = capital
    peak = capital
    max_dd_pct = 0
    log = []

    for t in trades:
        cost_r = trade_cost_R(t)
        adj_r = t["rMult"] - cost_r
        risk_amt = equity * risk_pct
        pnl_gross = t["rMult"] * risk_amt
        pnl_cost = cost_r * risk_amt
        pnl_net = adj_r * risk_amt
        equity += pnl_net
        equity = max(equity, 0.01)
        if equity > peak:
            peak = equity
        dd = (peak - equity) / peak

        # Lot size for reference
        risk_pips = abs(t["entry_price"] - t["sl_price"]) / PIP_SIZE[t["symbol"]]
        lots = risk_amt / (risk_pips * PIP_VALUE) if risk_pips > 0 else 0

        log.append({
            "date": t["date"],
            "symbol": t["symbol"],
            "dir": t["direction"],
            "result": t["result"],
            "rr": t["rr"],
            "orig_R": t["rMult"],
            "cost_R": cost_r,
            "adj_R": adj_r,
            "risk_amt": risk_amt,
            "pnl_gross": pnl_gross,
            "pnl_cost": pnl_cost,
            "pnl_net": pnl_net,
            "lots": lots,
            "equity": equity,
            "dd_pct": dd,
        })

    return log


# ══════��═════════════════════════════════════��═════════════════════
#  REPORTS
# ════════��══════════════════════��══════════════════════════════════

def h(text):
    print(f"\n{'='*100}")
    print(f"  {text}")
    print(f"{'='*100}")


def report_trade_log(log, title, last_n=50):
    """Per-trade log, last N trades."""
    h(f"TRADE LOG — {title} (last {last_n} trades)")
    print()
    print(f"  {'#':>4} {'Date':>11} {'Pair':<7} {'Dir':<6} {'Result':<8} {'RR':>5} "
          f"{'Gross R':>8} {'Cost R':>7} {'Net R':>7} {'Risk$':>8} {'P&L$':>10} {'Lots':>6} {'Equity':>12} {'DD%':>6}")
    print(f"  {'─'*4} {'─'*11} {'─'*7} {'─'*6} {'─'*8} {'─'*5} "
          f"{'─'*8} {'─'*7} {'─'*7} {'─'*8} {'─'*10} {'─'*6} {'─'*12} {'─'*6}")

    start = max(0, len(log) - last_n)
    for i, e in enumerate(log[start:], start + 1):
        res_mark = {"win": "WIN ", "loss": "LOSS", "timeout": "T/O "}[e["result"]]
        print(f"  {i:>4} {e['date']:>11} {e['symbol']:<7} {e['dir']:<6} {res_mark:<8} "
              f"{e['rr']:>4.1f}x {e['orig_R']:>+7.2f} {e['cost_R']:>6.3f} {e['adj_R']:>+6.2f} "
              f"${e['risk_amt']:>7,.0f} ${e['pnl_net']:>9,.0f} {e['lots']:>5.2f} "
              f"${e['equity']:>11,.0f} {e['dd_pct']*100:>5.1f}%")


def report_monthly(log, title):
    """Monthly balance sheet."""
    h(f"MONTHLY BALANCE — {title}")
    print()

    monthly = OrderedDict()
    for e in log:
        ym = e["date"][:7]
        if ym not in monthly:
            monthly[ym] = {"trades":0,"wins":0,"losses":0,"timeouts":0,
                           "gross":0,"costs":0,"net":0,"equity_end":0,"peak_eq":0}
        m = monthly[ym]
        m["trades"] += 1
        if e["result"] == "win": m["wins"] += 1
        elif e["result"] == "loss": m["losses"] += 1
        else: m["timeouts"] += 1
        m["gross"] += e["pnl_gross"]
        m["costs"] += e["pnl_cost"]
        m["net"] += e["pnl_net"]
        m["equity_end"] = e["equity"]

    print(f"  {'Month':>8} {'Trades':>7} {'W/L/T':>8} {'WR':>6} {'Gross$':>10} {'Costs$':>9} "
          f"{'Net$':>10} {'Balance':>12} {'MoM%':>7}")
    print(f"  {'─'*8} {'─'*7} {'─'*8} {'─'*6} {'─'*10} {'─'*9} {'─'*10} {'─'*12} {'─'*7}")

    prev_eq = None
    for ym, m in monthly.items():
        wr = m["wins"] / m["trades"] * 100 if m["trades"] > 0 else 0
        wlt = f"{m['wins']}/{m['losses']}/{m['timeouts']}"
        mom = ((m["equity_end"] / prev_eq) - 1) * 100 if prev_eq and prev_eq > 0 else 0
        print(f"  {ym:>8} {m['trades']:>7} {wlt:>8} {wr:5.1f}% ${m['gross']:>9,.0f} "
              f"${m['costs']:>8,.0f} ${m['net']:>9,.0f} ${m['equity_end']:>11,.0f} {mom:>+6.1f}%")
        prev_eq = m["equity_end"]

    # Summary stats
    all_months = list(monthly.values())
    profitable = sum(1 for m in all_months if m["net"] > 0)
    total_months = len(all_months)
    avg_net = sum(m["net"] for m in all_months) / total_months if total_months > 0 else 0
    best_month = max(all_months, key=lambda m: m["net"])
    worst_month = min(all_months, key=lambda m: m["net"])

    print(f"\n  Summary: {profitable}/{total_months} months profitable ({profitable/total_months*100:.0f}%)")
    print(f"  Avg monthly P&L: ${avg_net:,.0f}")
    print(f"  Best month:  ${best_month['net']:>+10,.0f} ({best_month['wins']}/{best_month['trades']} wins)")
    print(f"  Worst month: ${worst_month['net']:>+10,.0f} ({worst_month['wins']}/{worst_month['trades']} wins)")


def report_quarterly(log, title):
    """Quarterly summary."""
    h(f"QUARTERLY SUMMARY — {title}")
    print()

    quarterly = OrderedDict()
    for e in log:
        y = int(e["date"][:4])
        m = int(e["date"][5:7])
        q = f"{y}-Q{(m-1)//3+1}"
        if q not in quarterly:
            quarterly[q] = {"trades":0,"wins":0,"gross":0,"costs":0,"net":0,"equity_end":0}
        qr = quarterly[q]
        qr["trades"] += 1
        if e["result"] == "win": qr["wins"] += 1
        qr["gross"] += e["pnl_gross"]
        qr["costs"] += e["pnl_cost"]
        qr["net"] += e["pnl_net"]
        qr["equity_end"] = e["equity"]

    print(f"  {'Quarter':>8} {'Trades':>7} {'Wins':>5} {'WR':>6} {'Gross$':>10} {'Costs$':>9} "
          f"{'Net$':>10} {'Balance':>12} {'QoQ%':>7}")
    print(f"  {'���'*8} {'─'*7} {'─'*5} {'─'*6} {'─'*10} {'─'*9} {'─'*10} {'─'*12} {'─'*7}")

    prev = None
    for q, qr in quarterly.items():
        wr = qr["wins"] / qr["trades"] * 100 if qr["trades"] > 0 else 0
        qoq = ((qr["equity_end"] / prev) - 1) * 100 if prev and prev > 0 else 0
        print(f"  {q:>8} {qr['trades']:>7} {qr['wins']:>5} {wr:5.1f}% ${qr['gross']:>9,.0f} "
              f"${qr['costs']:>8,.0f} ${qr['net']:>9,.0f} ${qr['equity_end']:>11,.0f} {qoq:>+6.1f}%")
        prev = qr["equity_end"]


def report_annual(log, title, start_capital):
    """Annual P&L statement."""
    h(f"ANNUAL P&L — {title}")
    print()

    yearly = OrderedDict()
    for e in log:
        y = e["date"][:4]
        if y not in yearly:
            yearly[y] = {"trades":0,"wins":0,"losses":0,"gross":0,"costs":0,"net":0,
                         "equity_start":0,"equity_end":0,"max_dd":0}
        yr = yearly[y]
        yr["trades"] += 1
        if e["result"] == "win": yr["wins"] += 1
        elif e["result"] == "loss": yr["losses"] += 1
        yr["gross"] += e["pnl_gross"]
        yr["costs"] += e["pnl_cost"]
        yr["net"] += e["pnl_net"]
        yr["equity_end"] = e["equity"]
        if e["dd_pct"] > yr["max_dd"]:
            yr["max_dd"] = e["dd_pct"]

    # Set equity_start
    prev_end = start_capital
    for y in yearly:
        yearly[y]["equity_start"] = prev_end
        prev_end = yearly[y]["equity_end"]

    print(f"  {'Year':>6} {'Trades':>7} {'W/L':>6} {'WR':>6} {'Gross$':>11} {'Costs$':>10} "
          f"{'Net$':>11} {'Start$':>11} {'End$':>12} {'Return':>8} {'MaxDD':>7}")
    print(f"  {'─'*6} {'─'*7} {'─'*6} {'─'*6} {'─'*11} {'─'*10} {'─'*11} {'─'*11} {'─'*12} {'─'*8} {'─'*7}")

    total_costs = 0
    total_net = 0
    for y, yr in yearly.items():
        wr = yr["wins"] / yr["trades"] * 100 if yr["trades"] > 0 else 0
        ret = (yr["equity_end"] / yr["equity_start"] - 1) * 100 if yr["equity_start"] > 0 else 0
        wl = f"{yr['wins']}/{yr['losses']}"
        total_costs += yr["costs"]
        total_net += yr["net"]
        print(f"  {y:>6} {yr['trades']:>7} {wl:>6} {wr:5.1f}% ${yr['gross']:>10,.0f} ${yr['costs']:>9,.0f} "
              f"${yr['net']:>10,.0f} ${yr['equity_start']:>10,.0f} ${yr['equity_end']:>11,.0f} "
              f"{ret:>+7.1f}% {yr['max_dd']*100:>5.1f}%")

    print(f"\n  Total costs paid to broker over {len(yearly)} years: ${total_costs:,.0f}")
    print(f"  Total net profit: ${total_net:,.0f}")
    print(f"  CAGR: {((log[-1]['equity']/start_capital)**(1/max(len(yearly),1))-1)*100:.1f}%")


def report_drawdown(log, title):
    """Drawdown analysis."""
    h(f"DRAWDOWN ANALYSIS — {title}")
    print()

    # Find all drawdown periods
    peak = 0
    dd_start = None
    dd_periods = []
    current_dd = 0

    for i, e in enumerate(log):
        if e["equity"] > peak:
            if dd_start is not None and current_dd > 0.05:
                dd_periods.append({
                    "start": dd_start,
                    "end": e["date"],
                    "depth": current_dd,
                    "trades": i - dd_start_idx,
                })
            peak = e["equity"]
            dd_start = None
            current_dd = 0
        else:
            if dd_start is None:
                dd_start = e["date"]
                dd_start_idx = i
            dd = (peak - e["equity"]) / peak
            if dd > current_dd:
                current_dd = dd

    # Sort by depth
    dd_periods.sort(key=lambda x: x["depth"], reverse=True)

    print(f"  Top drawdown periods (>5%):")
    print(f"  {'#':>3} {'Start':>11} {'End':>11} {'Depth':>7} {'Trades':>7} {'Recovery':>10}")
    print(f"  {'─'*3} {'─'*11} {'─'*11} {'─'*7} {'─'*7} {'─'*10}")
    for i, dd in enumerate(dd_periods[:10], 1):
        print(f"  {i:>3} {dd['start']:>11} {dd['end']:>11} {dd['depth']*100:>5.1f}% {dd['trades']:>5} tr  {'recovered':>10}")

    # Losing streak analysis
    streak = 0
    max_streak = 0
    streaks = []
    for e in log:
        if e["adj_R"] < 0:
            streak += 1
        else:
            if streak > 0:
                streaks.append(streak)
            streak = 0
    if streak > 0:
        streaks.append(streak)

    streaks.sort(reverse=True)
    print(f"\n  Losing streak distribution:")
    print(f"    Longest: {streaks[0] if streaks else 0} consecutive losses")
    print(f"    Top 5: {streaks[:5]}")
    print(f"    Avg streak: {sum(streaks)/len(streaks):.1f}" if streaks else "    No streaks")

    # Months underwater
    monthly_eq = OrderedDict()
    for e in log:
        monthly_eq[e["date"][:7]] = e["equity"]

    peak_m = 0
    underwater = 0
    max_uw = 0
    for ym, eq in monthly_eq.items():
        if eq >= peak_m:
            peak_m = eq
            if underwater > max_uw:
                max_uw = underwater
            underwater = 0
        else:
            underwater += 1

    print(f"\n  Longest underwater period: {max(max_uw, underwater)} months")


def report_scenarios(trades_by_config, all_trades):
    """Compare configs at different starting capitals and risk levels."""
    h("SCENARIO MATRIX — All Configs × Risk × Starting Capital")
    print()

    print(f"  {'Config':<30} {'Risk':>5} {'Start':>8} {'End (1yr)':>10} {'End (3yr)':>11} {'End (5yr)':>11} {'MaxDD':>7}")
    print(f"  {'─'*30} {'─'*5} {'─'*8} {'─'*10} {'─'*11} {'─'*11} {'─'*7}")

    for name, trades in trades_by_config.items():
        if len(trades) < 20:
            continue
        # Use last 5 years of trades for projection (most relevant)
        recent = [t for t in trades if t["date"] >= "2021"]
        if len(recent) < 10:
            recent = trades[-100:]  # fallback

        for risk in [0.01, 0.02, 0.03]:
            for cap in [1000, 5000, 10000]:
                log = simulate(trades, cap, risk)
                # For 1yr/3yr/5yr: approximate from annual returns
                years_data = max((trades[-1]["date_obj"] - trades[0]["date_obj"]).days / 365.25, 1)
                cagr = (log[-1]["equity"] / cap) ** (1 / years_data) - 1 if log[-1]["equity"] > cap else -0.1
                yr1 = cap * (1 + cagr)
                yr3 = cap * (1 + cagr) ** 3
                yr5 = cap * (1 + cagr) ** 5
                max_dd = max(e["dd_pct"] for e in log)

                if cap == 1000 and risk == 0.02:  # highlight default
                    marker = " <<<"
                else:
                    marker = ""

                print(f"  {name:<30} {risk*100:4.0f}% ${cap:>6,} ${yr1:>9,.0f} ${yr3:>10,.0f} ${yr5:>10,.0f} "
                      f"{max_dd*100:>5.1f}%{marker}")
        print()


def report_year_deep_dive(log, title, year):
    """Every trade in a specific year with running balance."""
    year_log = [e for e in log if e["date"][:4] == year]
    if not year_log:
        print(f"\n  No trades in {year}")
        return

    h(f"YEAR {year} DEEP DIVE ��� {title} ({len(year_log)} trades)")
    print()

    # Start equity = equity before first trade of this year
    idx = log.index(year_log[0])
    start_eq = log[idx - 1]["equity"] if idx > 0 else log[0]["equity"]

    print(f"  Starting balance: ${start_eq:,.0f}")
    print()
    print(f"  {'#':>3} {'Date':>11} {'Pair':<7} {'Dir':<5} {'Res':<5} {'RR':>4} "
          f"{'Gross':>8} {'Cost':>7} {'Net P&L':>9} {'Balance':>11} {'DD%':>5}")
    print(f"  {'─'*3} {'─'*11} {'─'*7} {'─'*5} {'─'*5} {'─'*4} "
          f"{'─'*8} {'─'*7} {'─'*9} {'─'*11} {'─'*5}")

    for i, e in enumerate(year_log, 1):
        res = {"win":"W","loss":"L","timeout":"T"}[e["result"]]
        print(f"  {i:>3} {e['date']:>11} {e['symbol']:<7} {e['dir']:<5} {res:<5} "
              f"{e['rr']:>3.1f} ${e['pnl_gross']:>7,.0f} ${e['pnl_cost']:>6,.0f} "
              f"${e['pnl_net']:>8,.0f} ${e['equity']:>10,.0f} {e['dd_pct']*100:>4.1f}%")

        # Monthly separator
        if i < len(year_log) and year_log[i]["date"][5:7] != e["date"][5:7]:
            month_trades = [x for x in year_log[:i] if x["date"][5:7] == e["date"][5:7]]
            month_net = sum(x["pnl_net"] for x in month_trades)
            month_wins = sum(1 for x in month_trades if x["result"] == "win")
            print(f"  {'':>3} {'─── Month':>11} {e['date'][5:7]} total: {len(month_trades)} trades, "
                  f"{month_wins} wins, net ${month_net:+,.0f} {'─'*30}")

    end_eq = year_log[-1]["equity"]
    year_ret = (end_eq / start_eq - 1) * 100
    year_net = sum(e["pnl_net"] for e in year_log)
    year_costs = sum(e["pnl_cost"] for e in year_log)
    year_wins = sum(1 for e in year_log if e["result"] == "win")

    print(f"\n  {'─'*90}")
    print(f"  YEAR {year} TOTALS:")
    print(f"    Trades: {len(year_log)} | Wins: {year_wins} | WR: {year_wins/len(year_log)*100:.1f}%")
    print(f"    Gross P&L: ${sum(e['pnl_gross'] for e in year_log):+,.0f}")
    print(f"    Costs:     ${year_costs:,.0f}")
    print(f"    Net P&L:   ${year_net:+,.0f}")
    print(f"    Start: ${start_eq:,.0f} → End: ${end_eq:,.0f} ({year_ret:+.1f}%)")


def report_projection(log, title, start_capital, risk_pct):
    """Forward projection based on historical stats."""
    h(f"FORWARD PROJECTION — {title}")
    print()

    # Calculate historical stats from the log
    n = len(log)
    d0 = datetime.strptime(log[0]["date"], "%Y-%m-%d")
    d1 = datetime.strptime(log[-1]["date"], "%Y-%m-%d")
    years = (d1 - d0).days / 365.25
    trades_per_month = n / (years * 12)
    avg_adj_R = sum(e["adj_R"] for e in log) / n
    wr = sum(1 for e in log if e["result"] == "win") / n
    avg_cost_R = sum(e["cost_R"] for e in log) / n
    cagr = (log[-1]["equity"] / start_capital) ** (1 / years) - 1

    print(f"  Based on {n} trades over {years:.1f} years:")
    print(f"  Trades/month: {trades_per_month:.1f} | WR: {wr*100:.1f}% | Avg net R: {avg_adj_R:+.3f} | Avg cost: {avg_cost_R:.3f}R")
    print(f"  Historical CAGR: {cagr*100:.1f}%")
    print()

    # Project for different scenarios
    scenarios = [
        ("Conservative (50% of edge)", cagr * 0.5),
        ("Realistic (75% of edge)", cagr * 0.75),
        ("Historical (100% of edge)", cagr),
    ]

    for cap in [1000, 5000, 10000, 25000]:
        print(f"  Starting capital: ${cap:,}")
        print(f"  {'Scenario':<30} {'Year 1':>10} {'Year 2':>10} {'Year 3':>10} {'Year 5':>10} {'Year 10':>11}")
        print(f"  {'─'*30} {'─'*10} {'─'*10} {'─'*10} {'─'*10} {'─'*11}")
        for label, rate in scenarios:
            y1 = cap * (1 + rate)
            y2 = cap * (1 + rate) ** 2
            y3 = cap * (1 + rate) ** 3
            y5 = cap * (1 + rate) ** 5
            y10 = cap * (1 + rate) ** 10
            print(f"  {label:<30} ${y1:>9,.0f} ${y2:>9,.0f} ${y3:>9,.0f} ${y5:>9,.0f} ${y10:>10,.0f}")
        print()


# ══════════════════════════════════════════════════════════════════
#  MAIN
# ══════════���═══════════════════════��═══════════════════════════════

def main():
    trades = load_trades()
    start_capital = 1000
    risk_pct = 0.02

    print(f"SAF Strategy — Realistic Growth Report")
    print(f"Data: {len(trades)} trades, {trades[0]['date']} to {trades[-1]['date']}")
    print(f"Starting capital: ${start_capital:,} | Risk: {risk_pct*100:.0f}% per trade")

    # Apply configs
    trades_by_config = {}
    for name, fn in CONFIGS.items():
        trades_by_config[name] = [t for t in trades if fn(t)]

    # Use TIERED as the main focus (best after costs)
    main_config = "B: TIERED ≥10+ded no eng"
    main_trades = trades_by_config[main_config]
    main_log = simulate(main_trades, start_capital, risk_pct)

    # ── Report 1: Annual P&L for all configs ──
    for name, cfg_trades in trades_by_config.items():
        if len(cfg_trades) < 20:
            continue
        log = simulate(cfg_trades, start_capital, risk_pct)
        report_annual(log, f"{name} | ${start_capital:,} | {risk_pct*100:.0f}%", start_capital)

    # ── Report 2: Monthly balance for main config ──
    report_monthly(main_log, f"{main_config} | ${start_capital:,} | {risk_pct*100:.0f}%")

    # ── Report 3: Quarterly for main config ──
    report_quarterly(main_log, f"{main_config} | ${start_capital:,} | {risk_pct*100:.0f}%")

    # ── Report 4: Deep dive last 3 years ──
    for year in ["2023", "2024", "2025"]:
        report_year_deep_dive(main_log, main_config, year)

    # ── Report 5: Drawdown analysis ─���
    report_drawdown(main_log, main_config)

    # ── Report 6: Trade log (last 50) ──
    report_trade_log(main_log, main_config, 50)

    # ── Report 7: Forward projection ──
    report_projection(main_log, main_config, start_capital, risk_pct)

    # ── Report 8: Scenario matrix ──
    report_scenarios(trades_by_config, trades)

    # ── Report 9: Multiple starting capitals for main config ──
    h(f"STARTING CAPITAL COMPARISON — {main_config} at 2% risk")
    print()
    print(f"  {'Start':>10} {'Final':>12} {'Net':>12} {'Costs':>10} {'Cost%':>7} {'CAGR':>7} {'MaxDD%':>7}")
    print(f"  {'─'*10} {'─'*12} {'─'*12} {'─'*10} {'─'*7} {'─'*7} {'─'*7}")

    for cap in [500, 1000, 2000, 5000, 10000, 25000, 50000, 100000]:
        log = simulate(main_trades, cap, 0.02)
        final = log[-1]["equity"]
        costs = sum(e["pnl_cost"] for e in log)
        gross = sum(e["pnl_gross"] for e in log)
        cost_pct = costs / gross * 100 if gross > 0 else 0
        d0 = datetime.strptime(log[0]["date"], "%Y-%m-%d")
        d1 = datetime.strptime(log[-1]["date"], "%Y-%m-%d")
        years = (d1 - d0).days / 365.25
        cagr = (final / cap) ** (1 / years) - 1 if final > cap else 0
        max_dd = max(e["dd_pct"] for e in log) * 100
        print(f"  ${cap:>8,} ${final:>11,.0f} ${final-cap:>11,.0f} ${costs:>9,.0f} {cost_pct:5.1f}% "
              f"{cagr*100:6.1f}% {max_dd:5.1f}%")

    print(f"\n{'='*100}")
    print(f"  ALL REPORTS COMPLETE")
    print(f"{'='*100}\n")


if __name__ == "__main__":
    main()
