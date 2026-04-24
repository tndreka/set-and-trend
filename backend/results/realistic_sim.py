#!/usr/bin/env python3
"""
Realistic Broker Simulation — SAF Strategy
Accounts for: spreads, commissions, swap fees, slippage, position sizing, pip values.
Runs on trade_analysis.csv over 22 years of data.
"""

import csv
import random
import math
from collections import defaultdict
from datetime import datetime

random.seed(42)

# ══════════════════════════════════════════════════════════════════
#  BROKER COST MODEL (typical ECN broker, e.g. IC Markets, Pepperstone)
# ══════════════════════════════════════════════════════════════════

# Spreads in pips (average, not minimum — includes volatile sessions)
SPREADS = {
    "EURUSD": 1.0,   # tight major
    "GBPUSD": 1.3,   # slightly wider
    "AUDUSD": 1.2,
    "NZDUSD": 1.5,
    "USDCAD": 1.5,
    "USDCHF": 1.3,
    "USDJPY": 1.0,
}

# Commission per round-trip per standard lot (100K units)
COMMISSION_PER_LOT = 7.00  # $7 round-trip (typical ECN: $3.50 each way)

# Slippage in pips (average — entry + exit combined)
SLIPPAGE = {
    "EURUSD": 0.3,
    "GBPUSD": 0.5,
    "AUDUSD": 0.4,
    "NZDUSD": 0.5,
    "USDCAD": 0.5,
    "USDCHF": 0.4,
    "USDJPY": 0.3,
}

# Swap cost per night per standard lot (approximate average, both directions)
# These vary wildly by year/broker, this is a conservative average
SWAP_PER_NIGHT_PER_LOT = {
    "EURUSD": -8.0,   # ~$8/night/lot average cost
    "GBPUSD": -9.0,
    "AUDUSD": -7.0,
    "NZDUSD": -7.0,
    "USDCAD": -6.0,
    "USDCHF": -5.0,
    "USDJPY": -7.0,
}

# Pip sizes
PIP_SIZE = {
    "EURUSD": 0.0001,
    "GBPUSD": 0.0001,
    "AUDUSD": 0.0001,
    "NZDUSD": 0.0001,
    "USDCAD": 0.0001,
    "USDCHF": 0.0001,
    "USDJPY": 0.01,
}

# Pip value per standard lot (in USD, approximate)
# For XXX/USD pairs: $10 per pip per lot
# For USD/XXX pairs: $10 / rate (varies, we use ~$8-10)
# For simplicity, use $10 for all (close enough for estimation)
PIP_VALUE_PER_LOT = 10.0  # $10 per pip per standard lot

# Average holding time estimates (in trading days)
# H4 trades: entry to resolution
HOLD_DAYS = {
    "win": 4,      # winners tend to run 3-5 days
    "loss": 2,     # losers hit SL faster, 1-3 days
    "timeout": 10, # 60 H4 bars ≈ 10 trading days
}

# Wednesday triple swap (swap charged 3x on Wed night for weekend)
# Effectively: 5 swap days per 5 trading days
SWAP_MULTIPLIER = 7 / 5  # 7 calendar days of swap per 5 trading days


# ══════════════════════════════════════════════════════════════════
#  ITEM CLASSIFICATIONS (for quality scoring)
# ══════════════════════════════════════════════════════════════════

STRONG = ["d1_in_favor", "w1_pattern", "d1_touching_ema", "h4_in_favor", "d1_aoi"]
USEFUL = ["w1_candle_rejection", "d1_structure_rejection", "d1_pattern"]
ALL_16 = STRONG + USEFUL + [
    "w1_in_favor", "psych_level", "h4_candle_rejection", "w1_touching_ema",
    "h4_ema_touch", "engulfing",
]


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


# ══════════════════════════════════════════════════════════════════
#  COST CALCULATOR
# ══════════════════════════════════════════════════════════════════

def calc_trade_costs(trade, account_equity, risk_pct):
    """
    Calculate all broker costs for a single trade.
    Returns: adjusted_rMult, cost_breakdown dict
    """
    sym = trade["symbol"]
    entry = trade["entry_price"]
    sl = trade["sl_price"]
    result = trade["result"]
    original_rmult = trade["rMult"]
    rr = trade["rr"]

    pip_size = PIP_SIZE[sym]
    spread_pips = SPREADS[sym]
    slip_pips = SLIPPAGE[sym]

    # Risk in pips
    risk_pips = abs(entry - sl) / pip_size

    # Position size calculation
    risk_amount = account_equity * risk_pct  # $ risked
    # lot_size = risk_amount / (risk_pips * pip_value_per_lot)
    pip_value = PIP_VALUE_PER_LOT
    lot_size = risk_amount / (risk_pips * pip_value) if (risk_pips * pip_value) > 0 else 0

    # Minimum lot size check (0.01 lot = micro lot)
    if lot_size < 0.01:
        lot_size = 0.01

    # ── Cost 1: Spread (charged on entry, effectively widens SL) ──
    spread_cost_pips = spread_pips  # paid once on entry

    # ── Cost 2: Slippage (entry + exit) ──
    slippage_cost_pips = slip_pips

    # ── Cost 3: Commission ──
    commission_dollars = COMMISSION_PER_LOT * lot_size
    commission_pips = commission_dollars / (lot_size * pip_value) if lot_size > 0 else 0

    # ── Cost 4: Swap (overnight fees) ──
    hold_days = HOLD_DAYS.get(result, 4)
    swap_per_night = abs(SWAP_PER_NIGHT_PER_LOT[sym])
    # Adjusted for weekend (7 calendar days per 5 trading days)
    swap_nights = hold_days * SWAP_MULTIPLIER
    swap_dollars = swap_per_night * swap_nights * lot_size
    swap_pips = swap_dollars / (lot_size * pip_value) if lot_size > 0 else 0

    # ── Total cost in pips ──
    total_cost_pips = spread_cost_pips + slippage_cost_pips + commission_pips + swap_pips

    # ── Cost as fraction of risk (in R) ──
    cost_in_R = total_cost_pips / risk_pips if risk_pips > 0 else 0

    # ── Adjusted R-multiple ──
    # Spread + slippage worsen entry price, so:
    #   Win:     you get rr*R but lost cost_in_R on the way in
    #   Loss:    you lose 1R but also lost cost_in_R on the way in
    #   Timeout: whatever partial R, minus cost_in_R
    adjusted_rmult = original_rmult - cost_in_R

    # ── Dollar P&L ──
    gross_pnl = original_rmult * risk_amount
    total_cost_dollars = cost_in_R * risk_amount
    net_pnl = gross_pnl - total_cost_dollars

    return adjusted_rmult, {
        "risk_pips": risk_pips,
        "lot_size": lot_size,
        "spread_pips": spread_cost_pips,
        "slippage_pips": slippage_cost_pips,
        "commission_pips": commission_pips,
        "swap_pips": swap_pips,
        "total_cost_pips": total_cost_pips,
        "cost_in_R": cost_in_R,
        "hold_days": hold_days,
        "risk_amount": risk_amount,
        "gross_pnl": gross_pnl,
        "total_cost_dollars": total_cost_dollars,
        "net_pnl": net_pnl,
    }


# ══════════════════════════════════════════════════════════════════
#  EQUITY SIMULATION WITH COSTS
# ══════════════════════════════════════════════════════════════════

def simulate_realistic(trades, start_capital, risk_pct):
    """Walk through trades chronologically with real costs at each step."""
    equity = start_capital
    peak = start_capital
    max_dd_pct = 0
    max_dd_dollar = 0

    total_gross = 0
    total_spread = 0
    total_slippage = 0
    total_commission = 0
    total_swap = 0
    total_cost = 0

    yearly = defaultdict(lambda: {
        "trades": 0, "wins": 0, "gross_pnl": 0, "costs": 0,
        "net_pnl": 0, "equity_start": 0, "equity_end": 0,
    })

    trade_log = []
    current_year = None

    for t in trades:
        year = t["date"][:4]
        if year != current_year:
            yearly[year]["equity_start"] = equity
            current_year = year

        adj_rmult, costs = calc_trade_costs(t, equity, risk_pct)

        # Apply to equity
        pnl_net = adj_rmult * (equity * risk_pct)
        equity += pnl_net
        equity = max(equity, 0.01)

        if equity > peak:
            peak = equity
        dd_pct = (peak - equity) / peak
        dd_dollar = peak - equity
        if dd_pct > max_dd_pct:
            max_dd_pct = dd_pct
        if dd_dollar > max_dd_dollar:
            max_dd_dollar = dd_dollar

        # Accumulate costs
        total_gross += costs["gross_pnl"]
        total_spread += costs["spread_pips"] * costs["lot_size"] * PIP_VALUE_PER_LOT
        total_slippage += costs["slippage_pips"] * costs["lot_size"] * PIP_VALUE_PER_LOT
        total_commission += COMMISSION_PER_LOT * costs["lot_size"]
        swap_dollars = costs["swap_pips"] * costs["lot_size"] * PIP_VALUE_PER_LOT
        total_swap += swap_dollars
        total_cost += costs["total_cost_dollars"]

        # Yearly tracking
        yearly[year]["trades"] += 1
        if t["result"] == "win":
            yearly[year]["wins"] += 1
        yearly[year]["gross_pnl"] += costs["gross_pnl"]
        yearly[year]["costs"] += costs["total_cost_dollars"]
        yearly[year]["net_pnl"] += costs["net_pnl"]
        yearly[year]["equity_end"] = equity

        trade_log.append({
            "date": t["date"],
            "symbol": t["symbol"],
            "result": t["result"],
            "original_R": t["rMult"],
            "adjusted_R": adj_rmult,
            "cost_R": costs["cost_in_R"],
            "risk_pips": costs["risk_pips"],
            "lot_size": costs["lot_size"],
            "equity_after": equity,
        })

    return {
        "final_equity": equity,
        "max_dd_pct": max_dd_pct,
        "max_dd_dollar": max_dd_dollar,
        "total_gross": total_gross,
        "total_spread": total_spread,
        "total_slippage": total_slippage,
        "total_commission": total_commission,
        "total_swap": total_swap,
        "total_cost": total_cost,
        "net_profit": equity - start_capital,
        "yearly": yearly,
        "trade_log": trade_log,
    }


# ══════════════════════════════════════════════════════════════════
#  FILTERING (reused from sweep.py)
# ══════════════════════════════════════════════════════════════════

def quality_score(trade):
    return sum(2 * trade[i] for i in STRONG) + sum(trade[i] for i in USEFUL)


CONFIGS = {
    "ALL_no_engulfing": lambda t: t["engulfing"] == 0,
    "BEST_BALANCED (strong≥4+ded, no eng)": lambda t: (
        t["engulfing"] == 0 and (t["score"] == 0 or quality_score(t) >= 4)
    ),
    "QUALITY_TIGHT (strong≥4+ded, no eng, score==0 or qs>=8)": lambda t: (
        t["engulfing"] == 0 and (t["score"] == 0 or quality_score(t) >= 8)
    ),
    "TIERED (tiered≥10+ded, no eng)": lambda t: (
        t["engulfing"] == 0 and (t["score"] == 0 or quality_score(t) >= 10)
    ),
    "SNIPER (tiered≥11+ded)": lambda t: (
        t["score"] == 0 or quality_score(t) >= 11
    ),
    "DEDICATED_ONLY (score=0)": lambda t: t["score"] == 0,
}


# ══════════════════════════════════════════════════════════════════
#  OUTPUT
# ══════════════════════════════════════════════════════════════════

def print_header(text):
    print(f"\n{'='*95}")
    print(f"  {text}")
    print(f"{'='*95}")


def print_cost_model():
    print_header("BROKER COST MODEL (ECN Broker — IC Markets / Pepperstone tier)")
    print()
    print(f"  {'Pair':<10} {'Spread':>8} {'Slippage':>10} {'Commission':>12} {'Swap/night':>12}")
    print(f"  {'─'*10} {'─'*8} {'─'*10} {'─'*12} {'─'*12}")
    for sym in sorted(SPREADS.keys()):
        print(f"  {sym:<10} {SPREADS[sym]:>6.1f} pip  {SLIPPAGE[sym]:>7.1f} pip  "
              f"${COMMISSION_PER_LOT:>8.2f}/lot  ${abs(SWAP_PER_NIGHT_PER_LOT[sym]):>8.2f}/lot")
    print()
    print(f"  Commission: ${COMMISSION_PER_LOT:.2f} round-trip per standard lot (100K units)")
    print(f"  Swap: charged 7 nights per 5 trading days (Wed triple swap)")
    print(f"  Slippage: average entry+exit combined")
    print(f"  Position sizing: risk% of equity / (SL distance in pips × $10/pip/lot)")


def print_sim_result(name, result, config_trades, start_capital, risk_pct, years):
    n = len(config_trades)
    wins = sum(1 for t in config_trades if t["result"] == "win")
    wr = wins / n * 100 if n > 0 else 0
    per_month = n / (years * 12)

    print(f"\n  ┌─ {name}")
    print(f"  │  {n} trades | {per_month:.1f}/mo | WR {wr:.1f}% | Risk {risk_pct*100:.0f}% per trade")
    print(f"  │")

    # Cost breakdown
    r = result
    print(f"  │  ── COST BREAKDOWN (${start_capital:,.0f} start) ──")
    print(f"  │")
    print(f"  │  Gross profit:     ${r['total_gross']:>12,.2f}")
    print(f"  │  - Spread cost:    ${r['total_spread']:>12,.2f}")
    print(f"  │  - Slippage cost:  ${r['total_slippage']:>12,.2f}")
    print(f"  │  - Commission:     ${r['total_commission']:>12,.2f}")
    print(f"  │  - Swap/overnight: ${r['total_swap']:>12,.2f}")
    print(f"  │  ────────────────────────────────")
    print(f"  │  Total costs:      ${r['total_cost']:>12,.2f}")
    print(f"  │  Net profit:       ${r['net_profit']:>12,.2f}")
    print(f"  │  Final equity:     ${r['final_equity']:>12,.2f}")
    print(f"  │")

    cost_pct = r["total_cost"] / r["total_gross"] * 100 if r["total_gross"] > 0 else 0
    print(f"  │  Costs ate {cost_pct:.1f}% of gross profit")
    print(f"  │  Max drawdown: {r['max_dd_pct']*100:.1f}% (${r['max_dd_dollar']:,.0f})")
    print(f"  │")

    # Average cost per trade in R
    avg_cost_R = sum(tl["cost_R"] for tl in r["trade_log"]) / n if n > 0 else 0
    avg_orig_R = sum(tl["original_R"] for tl in r["trade_log"]) / n if n > 0 else 0
    avg_adj_R = sum(tl["adjusted_R"] for tl in r["trade_log"]) / n if n > 0 else 0
    print(f"  │  Avg cost per trade: {avg_cost_R:.3f}R")
    print(f"  │  Avg R before costs: {avg_orig_R:+.3f}R  |  After costs: {avg_adj_R:+.3f}R")
    print(f"  │")

    # Year-by-year
    print(f"  │  ── YEAR-BY-YEAR ──")
    print(f"  │  {'Year':>6} {'Trades':>7} {'WR':>6} {'Gross$':>10} {'Costs$':>9} {'Net$':>10} {'Equity':>12} {'DD%':>6}")
    print(f"  │  {'─'*6} {'─'*7} {'─'*6} {'─'*10} {'─'*9} {'─'*10} {'─'*12} {'─'*6}")

    for year in sorted(r["yearly"].keys()):
        y = r["yearly"][year]
        if y["trades"] == 0:
            continue
        wr_y = y["wins"] / y["trades"] * 100
        eq = y["equity_end"]
        # Calc DD from peak
        dd = 0
        if eq < r.get("_peak", eq):
            dd = (r.get("_peak", eq) - eq) / r.get("_peak", eq) * 100
        print(f"  │  {year:>6} {y['trades']:>7} {wr_y:5.1f}% ${y['gross_pnl']:>9,.0f} "
              f"${y['costs']:>8,.0f} ${y['net_pnl']:>9,.0f} ${eq:>11,.0f} {dd:5.1f}%")

    print(f"  └{'─'*80}")


def print_comparison_table(results_by_config, start_capital):
    print_header("SIDE-BY-SIDE: Ideal (no costs) vs Realistic (with costs)")
    print()
    print(f"  {'Config':<40} {'Gross':>10} {'Costs':>10} {'Net':>10} {'Cost%':>7} {'DD%':>6} {'Final':>12}")
    print(f"  {'─'*40} {'─'*10} {'─'*10} {'─'*10} {'─'*7} {'─'*6} {'─'*12}")

    for name, r in results_by_config.items():
        cost_pct = r["total_cost"] / r["total_gross"] * 100 if r["total_gross"] > 0 else 0
        print(f"  {name:<40} ${r['total_gross']:>9,.0f} ${r['total_cost']:>9,.0f} "
              f"${r['net_profit']:>9,.0f} {cost_pct:5.1f}% {r['max_dd_pct']*100:5.1f}% ${r['final_equity']:>11,.0f}")


def print_risk_comparison(trades, name, start_capital):
    """Show same config at different risk levels."""
    print_header(f"RISK SIZING: {name}")
    print()
    print(f"  {'Risk':>6} {'Final':>12} {'Net Profit':>12} {'Costs':>10} {'Cost%':>7} {'MaxDD%':>8} {'CAGR':>8}")
    print(f"  {'─'*6} {'─'*12} {'─'*12} {'─'*10} {'─'*7} {'─'*8} {'─'*8}")

    d0 = trades[0]["date_obj"]
    d1 = trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)

    for risk in [0.005, 0.01, 0.015, 0.02, 0.03, 0.05]:
        r = simulate_realistic(trades, start_capital, risk)
        cost_pct = r["total_cost"] / r["total_gross"] * 100 if r["total_gross"] > 0 else 0
        cagr = (r["final_equity"] / start_capital) ** (1 / years) - 1 if r["final_equity"] > start_capital else 0
        print(f"  {risk*100:5.1f}% ${r['final_equity']:>11,.0f} ${r['net_profit']:>11,.0f} "
              f"${r['total_cost']:>9,.0f} {cost_pct:5.1f}% {r['max_dd_pct']*100:7.1f}% {cagr*100:7.1f}%")


def print_cost_impact_on_wr(trades, start_capital, risk_pct):
    """Show how costs change the effective win rate and expectancy."""
    print_header("COST IMPACT ON EDGE — Before vs After Costs")
    print()

    n = len(trades)
    d0 = trades[0]["date_obj"]
    d1 = trades[-1]["date_obj"]
    years = max((d1 - d0).days / 365.25, 1)

    # Original stats
    orig_rmults = [t["rMult"] for t in trades]
    orig_exp = sum(orig_rmults) / n
    orig_wins = sum(1 for t in trades if t["result"] == "win")
    orig_wr = orig_wins / n

    # Adjusted stats
    adj_rmults = []
    adj_wins = 0
    for t in trades:
        adj_r, costs = calc_trade_costs(t, start_capital, risk_pct)
        adj_rmults.append(adj_r)
        if adj_r > 0:
            adj_wins += 1

    adj_exp = sum(adj_rmults) / n
    adj_wr = adj_wins / n

    # Trades that flip from win to loss due to costs
    flipped = sum(1 for i, t in enumerate(trades) if t["rMult"] > 0 and adj_rmults[i] <= 0)

    print(f"  {n} trades over {years:.1f} years ({n/(years*12):.1f}/mo)")
    print()
    print(f"  {'Metric':<25} {'Before Costs':>15} {'After Costs':>15} {'Impact':>12}")
    print(f"  {'─'*25} {'─'*15} {'─'*15} {'─'*12}")
    print(f"  {'Win Rate':<25} {orig_wr*100:>14.1f}% {adj_wr*100:>14.1f}% {(adj_wr-orig_wr)*100:>+11.1f}pp")
    print(f"  {'Expectancy (R)':<25} {orig_exp:>+14.3f}R {adj_exp:>+14.3f}R {adj_exp-orig_exp:>+11.3f}R")
    print(f"  {'Total R':<25} {sum(orig_rmults):>+14.1f}R {sum(adj_rmults):>+14.1f}R {sum(adj_rmults)-sum(orig_rmults):>+11.1f}R")
    print(f"  {'Trades flipped W→L':<25} {'':>15} {flipped:>15} {'':>12}")
    print()
    print(f"  Note: WR comparison uses rMult > 0 as 'win' (costs can make small wins negative)")
    print(f"  Risk level used for position sizing: {risk_pct*100:.1f}%")


# ══════════════════════════════════════════════════════════════════
#  MAIN
# ══════════════════════════════════════════════════════════════════

def main():
    trades = load_trades()
    start_capital = 1000
    risk_pct = 0.02  # 2% default

    d0 = trades[0]["date_obj"]
    d1 = trades[-1]["date_obj"]
    years = (d1 - d0).days / 365.25

    print(f"Loaded {len(trades)} trades ({trades[0]['date']} to {trades[-1]['date']}, {years:.1f} years)")

    # ── Show cost model ──
    print_cost_model()

    # ── Run each config ──
    results_by_config = {}

    for name, filter_fn in CONFIGS.items():
        filtered = [t for t in trades if filter_fn(t)]
        if len(filtered) < 10:
            continue

        result = simulate_realistic(filtered, start_capital, risk_pct)
        results_by_config[name] = result
        print_sim_result(name, result, filtered, start_capital, risk_pct, years)

    # ── Side-by-side comparison ──
    print_comparison_table(results_by_config, start_capital)

    # ── Cost impact on edge ──
    # Use best balanced config
    best_trades = [t for t in trades if t["engulfing"] == 0 and (
        t["score"] == 0 or quality_score(t) >= 4
    )]
    print_cost_impact_on_wr(best_trades, start_capital, risk_pct)

    # ── Risk sizing comparison for best config ──
    print_risk_comparison(best_trades, "BEST BALANCED (strong≥4+ded, no eng)", start_capital)

    # ── Different starting capitals ──
    print_header("STARTING CAPITAL COMPARISON — Best Balanced at 2% risk")
    print()
    print(f"  {'Start$':>10} {'Final$':>14} {'Net$':>14} {'Costs$':>12} {'Cost%':>7} {'MaxDD$':>12} {'CAGR':>8}")
    print(f"  {'─'*10} {'─'*14} {'─'*14} {'─'*12} {'─'*7} {'─'*12} {'─'*8}")

    for cap in [500, 1000, 2000, 5000, 10000, 25000, 50000]:
        r = simulate_realistic(best_trades, cap, 0.02)
        cost_pct = r["total_cost"] / r["total_gross"] * 100 if r["total_gross"] > 0 else 0
        cagr = (r["final_equity"] / cap) ** (1 / years) - 1 if r["final_equity"] > cap else 0
        print(f"  ${cap:>8,} ${r['final_equity']:>13,.0f} ${r['net_profit']:>13,.0f} "
              f"${r['total_cost']:>11,.0f} {cost_pct:5.1f}% ${r['max_dd_dollar']:>11,.0f} {cagr*100:7.1f}%")

    # ── Monthly P&L at $10K starting capital ──
    print_header("MONTHLY CASH P&L — Best Balanced, $10K start, 2% risk")
    print()
    r10k = simulate_realistic(best_trades, 10000, 0.02)
    monthly = defaultdict(lambda: {"net": 0, "count": 0, "wins": 0})
    for tl in r10k["trade_log"]:
        ym = tl["date"][:7]
        adj_pnl = tl["adjusted_R"] * tl["equity_after"] * 0.02 / (1 + 0.02 * tl["adjusted_R"])  # approximate
        monthly[ym]["net"] += adj_pnl
        monthly[ym]["count"] += 1
        if tl["adjusted_R"] > 0:
            monthly[ym]["wins"] += 1

    # Show last 24 months
    months = sorted(monthly.keys())
    recent = months[-24:] if len(months) >= 24 else months
    print(f"  Last {len(recent)} months:")
    print(f"  {'Month':>8} {'Trades':>7} {'Wins':>5} {'Net P&L':>10}")
    print(f"  {'─'*8} {'─'*7} {'─'*5} {'─'*10}")
    for ym in recent:
        m = monthly[ym]
        print(f"  {ym:>8} {m['count']:>7} {m['wins']:>5} ${m['net']:>9,.0f}")

    print(f"\n{'='*95}")
    print(f"  SIMULATION COMPLETE")
    print(f"{'='*95}\n")


if __name__ == "__main__":
    main()
