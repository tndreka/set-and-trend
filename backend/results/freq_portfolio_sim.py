#!/usr/bin/env python3
"""
Portfolio sim across SAF_CHECKLIST thresholds — find the sweet spot for
user target (2 trades/week = ~100/yr) given realistic broker costs.
"""

import csv
from datetime import datetime
from collections import defaultdict

CSV = "trade_analysis.csv"
FOCUSED = ["SAF_W1_EMA_REJECTION", "SAF_D1_REJECTION_STRUCTURE", "SAF_W1_REJECTION_D1_AOI"]
ALL_4 = FOCUSED + ["SAF_CHECKLIST"]

SPREAD = {"EURUSD":1.0,"GBPUSD":1.3,"AUDUSD":1.2,"NZDUSD":1.5,"USDCAD":1.5,"USDCHF":1.3,
          "USDJPY":1.0,"EURJPY":1.5,"GBPJPY":2.0,"AUDJPY":1.8,"CADJPY":2.5,"CHFJPY":2.5,
          "NZDJPY":2.5,"EURGBP":1.2,"EURAUD":2.0,"EURCAD":2.0,"EURCHF":1.8,"GBPCHF":2.5,
          "GBPAUD":2.5,"AUDCHF":2.5,"AUDNZD":2.5,"XAUUSD":2.5}
SWAP = {k: max(5, v*6) for k, v in SPREAD.items()}
PIP = {sym:(0.01 if "JPY" in sym else 0.0001) for sym in SPREAD}
PIP["XAUUSD"]=0.01
PIP_VAL=10.0; COMM=7.0
HOLD={"win":4,"loss":2,"timeout":10}

def load():
    rows=[]
    with open(CSV) as f:
        for r in csv.DictReader(f):
            if r["strategy"] not in ALL_4: continue
            r["score"]=int(r["score"]); r["rMult"]=float(r["rMult"])
            r["entry"]=float(r["entry"]); r["sl"]=float(r["sl"])
            r["date_obj"]=datetime.strptime(r["date"],"%Y-%m-%d")
            rows.append(r)
    return sorted(rows,key=lambda t:t["date_obj"])

def dedup(ts):
    pri={s:i for i,s in enumerate(ALL_4)}
    dd={}
    for t in ts:
        k=(t["symbol"],t["date"],t["direction"])
        if k not in dd or pri[t["strategy"]]<pri[dd[k]["strategy"]]: dd[k]=t
    return sorted(dd.values(),key=lambda t:t["date_obj"])

def sim(trades, start, risk, min_lot=0.01):
    eq=start; peak=eq; maxdd=0; wins=0; blown=False
    for t in trades:
        pip=PIP.get(t["symbol"],0.0001)
        risk_pips=abs(t["entry"]-t["sl"])/pip
        if risk_pips<=0: continue
        target_lots=(eq*risk)/(risk_pips*PIP_VAL)
        lots=max(target_lots,min_lot)
        eff_risk=lots*risk_pips*PIP_VAL
        if eff_risk>eq*0.9:
            blown=True; break
        commission=COMM*lots
        swap=SWAP.get(t["symbol"],7)*HOLD.get(t["result"],4)*7/5*lots
        spread=SPREAD.get(t["symbol"],2)*lots*PIP_VAL
        slip=max(0.3,SPREAD.get(t["symbol"],2)*0.35)*lots*PIP_VAL
        costs=commission+swap+spread+slip
        gross=t["rMult"]*eff_risk
        eq=max(eq+gross-costs,0.01)
        if eq<=1: blown=True; break
        peak=max(peak,eq)
        dd=(peak-eq)/peak
        maxdd=max(maxdd,dd)
        if t["result"]=="win": wins+=1
    return eq, maxdd, wins, blown

def main():
    trades=load()
    years=(trades[-1]["date_obj"]-trades[0]["date_obj"]).days/365.25
    focused_trades=[t for t in trades if t["strategy"] in FOCUSED]
    saf_trades=[t for t in trades if t["strategy"]=="SAF_CHECKLIST"]
    print(f"Loaded {len(trades)} trades across {years:.1f} years")
    print(f"  Focused 3-leg: {len(focused_trades)}")
    print(f"  SAF_CHECKLIST: {len(saf_trades)}\n")

    print("="*105)
    print("  4-LEG DEDUPED PORTFOLIO — EQUITY SIM at varying SAF threshold")
    print("  $100 start, 1% risk intended (micro-lot may force higher on SL-small trades)")
    print("="*105)
    print(f"  {'SAF≥':<5} {'Tr':>6} {'Tr/yr':>6} {'Tr/wk':>6} {'WR':>6} {'Exp/R':>7}  "
          f"{'$100→':>9} {'$500→':>11} {'CAGR':>7} {'MaxDD':>7} {'Blow':>5}")
    print(f"  {'-'*5} {'-'*6} {'-'*6} {'-'*6} {'-'*6} {'-'*7}  {'-'*9} {'-'*11} {'-'*7} {'-'*7} {'-'*5}")

    for thresh in [5,6,7,8,9,10,11,12]:
        saf_filt=[t for t in saf_trades if t["score"]>=thresh]
        combined=focused_trades+saf_filt
        deduped=dedup(combined)
        if not deduped: continue
        n=len(deduped); wins=sum(1 for t in deduped if t["result"]=="win")
        wr=wins/n; exp=sum(t["rMult"] for t in deduped)/n
        tr_yr=n/years; tr_wk=tr_yr/52

        eq100,dd100,_,blown100=sim(deduped,100,0.01)
        eq500,dd500,_,blown500=sim(deduped,500,0.01)
        cagr100=(eq100/100)**(1/years)-1 if eq100>1 else -1
        print(f"  {thresh:<5} {n:>6} {tr_yr:>6.1f} {tr_wk:>6.1f} {wr*100:>5.1f}% {exp:>+7.3f}  "
              f"${eq100:>7,.0f} ${eq500:>9,.0f} {cagr100*100:>6.1f}% {dd100*100:>6.1f}% "
              f"{'YES' if blown100 else 'no':>5}")

    print("\n"+"="*105)
    print("  COMPARISON — 3-leg focused only (status quo) vs. user-target frequency")
    print("="*105)
    for label, ts in [("3-leg focused (current)", focused_trades), ("SAF≥8 added (≈2.6/wk)",
            focused_trades + [t for t in saf_trades if t["score"]>=8])]:
        deduped=dedup(ts)
        n=len(deduped); wins=sum(1 for t in deduped if t["result"]=="win")
        print(f"\n  {label}: {n} trades, {n/years:.1f}/yr, {n/years/52:.1f}/wk, "
              f"{wins/n*100:.1f}% WR, {sum(t['rMult'] for t in deduped)/n:+.3f}R exp")
        for risk_label, risk in [("1% risk", 0.01), ("2% risk", 0.02), ("3% risk", 0.03)]:
            eq100,dd100,_,blown=sim(deduped,100,risk)
            eq500,dd500,_,_=sim(deduped,500,risk)
            cagr=(eq500/500)**(1/years)-1 if eq500>1 else -1
            print(f"    {risk_label:<10}  $100→${eq100:>10,.0f} (DD {dd100*100:>4.1f}%{', BLOWN' if blown else ''})  "
                  f"$500→${eq500:>12,.0f} (DD {dd500*100:>4.1f}%, CAGR {cagr*100:>4.1f}%)")

if __name__=="__main__":
    main()
