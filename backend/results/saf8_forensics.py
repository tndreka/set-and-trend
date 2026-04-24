#!/usr/bin/env python3
"""
SAF >= 8 Forensic Analysis.

Goal: keep frequency (~2.6 trades/week), raise win rate.

Pipeline:
  1. Baseline: W/L/T distribution, WR by score
  2. Single-item lift: WR|item=1 vs WR|item=0
  3. Anti-signals (items that DECREASE WR when present)
  4. Strong signals (items that INCREASE WR when present)
  5. Item-pair combos (both=1): find filter-out and boost-in candidates
  6. Item-TRIPLE combos for highest-confidence setups
  7. Timeout forensics: do timeouts have a signature?
  8. Synthesize candidate filters and simulate impact
"""

import csv
from collections import defaultdict
from datetime import datetime
import math

CSV = "trade_analysis.csv"
ITEMS = [
    "w1_in_favor","w1_touching_ema","w1_candle_rejection","w1_pattern",
    "d1_in_favor","d1_aoi","d1_touching_ema","d1_candle_rejection",
    "d1_structure_rejection","d1_pattern",
    "h4_in_favor","h4_ema_touch","h4_candle_rejection","h4_pattern",
]

def load():
    rows=[]
    with open(CSV) as f:
        for r in csv.DictReader(f):
            if r["strategy"]!="SAF_CHECKLIST": continue
            s=int(r["score"])
            if s<8: continue
            r["score"]=s
            r["rMult"]=float(r["rMult"])
            r["rr"]=float(r["rr"])
            for it in ITEMS:
                r[it]=int(r.get(it,0))
            rows.append(r)
    return rows

def z_score(k, n, p):
    """Two-proportion z-score test: is observed WR k/n different from baseline p?"""
    if n<5: return 0
    se=math.sqrt(p*(1-p)/n) if p>0 and p<1 else 0
    if se==0: return 0
    return (k/n - p)/se

def main():
    trades=load()
    n=len(trades)
    wins=sum(1 for t in trades if t["result"]=="win")
    losses=sum(1 for t in trades if t["result"]=="loss")
    timeouts=sum(1 for t in trades if t["result"]=="timeout")
    baseline_wr=wins/n

    # ────────────────────────────────────────────────────────────
    print("="*95)
    print(f"  PHASE 1 — BASELINE (SAF_CHECKLIST, score >= 8)")
    print("="*95)
    print(f"  Total trades: {n}")
    print(f"  Wins: {wins} ({wins/n*100:.1f}%)")
    print(f"  Losses: {losses} ({losses/n*100:.1f}%)")
    print(f"  Timeouts: {timeouts} ({timeouts/n*100:.1f}%)")
    total_r=sum(t["rMult"] for t in trades)
    print(f"  Expectancy: {total_r/n:+.3f}R  |  Total: {total_r:+.1f}R")

    print(f"\n  By score:")
    print(f"  {'Score':>5} {'Tr':>5} {'WR':>6} {'Exp':>7} {'Tmo%':>6}")
    for s in [8,9,10,11]:
        ts=[t for t in trades if t["score"]==s]
        if not ts: continue
        w=sum(1 for t in ts if t["result"]=="win")
        to=sum(1 for t in ts if t["result"]=="timeout")
        print(f"  {s:>5} {len(ts):>5} {w/len(ts)*100:>5.1f}% {sum(t['rMult'] for t in ts)/len(ts):+7.3f} {to/len(ts)*100:>5.1f}%")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 2 — SINGLE-ITEM LIFT (WR if item=1 vs item=0)")
    print("="*95)
    print(f"  {'Item':<24} {'Hit%':>6} {'WR|1':>7} {'WR|0':>7} {'Lift':>7} {'z':>6} {'Signal'}")
    print(f"  {'-'*24} {'-'*6} {'-'*7} {'-'*7} {'-'*7} {'-'*6} {'-'*6}")
    single_stats = []
    for item in ITEMS:
        with_it=[t for t in trades if t[item]==1]
        without=[t for t in trades if t[item]==0]
        if not with_it or not without: continue
        w1=sum(1 for t in with_it if t["result"]=="win")
        w0=sum(1 for t in without if t["result"]=="win")
        wr1=w1/len(with_it); wr0=w0/len(without)
        lift=wr1-wr0
        z=z_score(w1, len(with_it), wr0)
        single_stats.append((item, len(with_it), wr1, wr0, lift, z))
        flag=""
        if z < -2: flag="⚠ ANTI"
        elif z > 2: flag="✓ STRONG"
        print(f"  {item:<24} {len(with_it)/n*100:>5.1f}% {wr1*100:>6.1f}% {wr0*100:>6.1f}% "
              f"{lift*100:>+6.1f}% {z:>+6.2f} {flag}")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 3 — ITEM-PAIR COMBOS (both=1, min 50 trades)")
    print("="*95)
    pair_stats=[]
    for i in range(len(ITEMS)):
        for j in range(i+1, len(ITEMS)):
            a, b = ITEMS[i], ITEMS[j]
            ts=[t for t in trades if t[a]==1 and t[b]==1]
            if len(ts)<50: continue
            w=sum(1 for t in ts if t["result"]=="win")
            to=sum(1 for t in ts if t["result"]=="timeout")
            wr=w/len(ts)
            exp=sum(t["rMult"] for t in ts)/len(ts)
            z=z_score(w, len(ts), baseline_wr)
            pair_stats.append((a, b, len(ts), wr, exp, z, to/len(ts)))

    # Top positive pairs
    pair_stats.sort(key=lambda x: -x[3])   # sort by WR
    print(f"\n  ✅ TOP 15 pairs by WR (filter-IN candidates):")
    print(f"  {'Pair':<50} {'Tr':>5} {'WR':>6} {'Exp':>7} {'z':>6}")
    for a,b,n_,wr,exp,z,_ in pair_stats[:15]:
        tag="★" if z>2 else " "
        print(f"  {a}+{b:<30} {n_:>5} {wr*100:>5.1f}% {exp:+7.3f} {z:+6.2f} {tag}")

    # Worst pairs
    pair_stats.sort(key=lambda x: x[3])
    print(f"\n  ⚠️  BOTTOM 15 pairs by WR (filter-OUT candidates):")
    print(f"  {'Pair':<50} {'Tr':>5} {'WR':>6} {'Exp':>7} {'z':>6}")
    for a,b,n_,wr,exp,z,_ in pair_stats[:15]:
        tag="⚠" if z<-2 else " "
        print(f"  {a}+{b:<30} {n_:>5} {wr*100:>5.1f}% {exp:+7.3f} {z:+6.2f} {tag}")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 4 — ITEM TRIPLES (all three=1, min 30 trades)")
    print("="*95)
    triple_stats=[]
    for i in range(len(ITEMS)):
        for j in range(i+1, len(ITEMS)):
            for k in range(j+1, len(ITEMS)):
                a,b,c=ITEMS[i],ITEMS[j],ITEMS[k]
                ts=[t for t in trades if t[a]==1 and t[b]==1 and t[c]==1]
                if len(ts)<30: continue
                w=sum(1 for t in ts if t["result"]=="win")
                exp=sum(t["rMult"] for t in ts)/len(ts)
                z=z_score(w, len(ts), baseline_wr)
                triple_stats.append((a,b,c,len(ts),w/len(ts),exp,z))

    triple_stats.sort(key=lambda x: -x[4])
    print(f"\n  ✅ TOP 10 triples by WR:")
    for a,b,c,n_,wr,exp,z in triple_stats[:10]:
        tag="★" if z>2 else " "
        print(f"  {tag} {a[:18]:<18} + {b[:18]:<18} + {c[:18]:<18} | {n_:>4} tr, WR {wr*100:>5.1f}%, Exp {exp:+.3f}R")

    triple_stats.sort(key=lambda x: x[4])
    print(f"\n  ⚠️  BOTTOM 10 triples by WR:")
    for a,b,c,n_,wr,exp,z in triple_stats[:10]:
        tag="⚠" if z<-2 else " "
        print(f"  {tag} {a[:18]:<18} + {b[:18]:<18} + {c[:18]:<18} | {n_:>4} tr, WR {wr*100:>5.1f}%, Exp {exp:+.3f}R")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 5 — TIMEOUT FORENSICS (what fires into timeout?)")
    print("="*95)
    timeout_trades=[t for t in trades if t["result"]=="timeout"]
    if timeout_trades:
        print(f"  {'Item':<24} {'Tmo rate':>9} {'Baseline':>9} {'Delta':>7}")
        print(f"  {'-'*24} {'-'*9} {'-'*9} {'-'*7}")
        baseline_tmo=timeouts/n
        for item in ITEMS:
            with_it=[t for t in trades if t[item]==1]
            if not with_it: continue
            tmo_rate=sum(1 for t in with_it if t["result"]=="timeout")/len(with_it)
            delta=tmo_rate - baseline_tmo
            mark=""
            if delta > 0.03: mark="⚠ tmo prone"
            elif delta < -0.03: mark="✓ resolves"
            print(f"  {item:<24} {tmo_rate*100:>8.1f}% {baseline_tmo*100:>8.1f}% {delta*100:>+6.1f}% {mark}")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 6 — DIRECTION / DAY-OF-WEEK SUB-PATTERNS")
    print("="*95)
    print(f"\n  By direction:")
    for d in ["LONG","SHORT"]:
        ts=[t for t in trades if t["direction"]==d]
        if not ts: continue
        w=sum(1 for t in ts if t["result"]=="win")
        print(f"    {d:<7} {len(ts):>5} tr, WR {w/len(ts)*100:>5.1f}%, Exp {sum(t['rMult'] for t in ts)/len(ts):+.3f}R")

    print(f"\n  By day-of-week:")
    for day in ["Mon","Tue","Wed","Thu","Fri"]:
        ts=[t for t in trades if datetime.strptime(t["date"],"%Y-%m-%d").strftime("%a")==day]
        if not ts: continue
        w=sum(1 for t in ts if t["result"]=="win")
        print(f"    {day:<4} {len(ts):>5} tr, WR {w/len(ts)*100:>5.1f}%, Exp {sum(t['rMult'] for t in ts)/len(ts):+.3f}R")

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*95)
    print(f"  PHASE 7 — FILTER SIMULATION (drop anti-signals, require strong)")
    print("="*95)

    # Extract anti-signals and strong signals from Phase 2
    anti = [it for it, n_, wr1, wr0, lift, z in single_stats if z<-2 and wr1<baseline_wr]
    strong = [it for it, n_, wr1, wr0, lift, z in single_stats if z>2 and wr1>baseline_wr]

    print(f"\n  Identified anti-signals (items that reduce WR when present): {anti}")
    print(f"  Identified strong signals (items that raise WR when present): {strong}")

    # Extract worst pairs (both must be true) for exclusion rule
    pair_stats.sort(key=lambda x: x[3])
    worst_pairs = [(a,b) for a,b,n_,wr,exp,z,_ in pair_stats[:10] if z<-2]
    print(f"\n  Worst pairs (both=1 → significantly below baseline): {worst_pairs[:5]}")

    # Simulate filters
    def apply(name, fn, baseline_n, baseline_wr):
        ts=[t for t in trades if fn(t)]
        if not ts:
            print(f"  {name:<55} {'EMPTY':>10}"); return
        w=sum(1 for t in ts if t["result"]=="win")
        exp=sum(t["rMult"] for t in ts)/len(ts)
        wr=w/len(ts)
        years=22.1
        print(f"  {name:<55} {len(ts):>5} tr ({len(ts)/years:>4.1f}/yr), "
              f"WR {wr*100:>5.1f}%, Exp {exp:+.3f}R  "
              f"({(len(ts)-baseline_n)/baseline_n*100:+.1f}% tr, {(wr-baseline_wr)*100:+.1f}pp WR)")

    print(f"\n  {'Filter':<55} {'Impact':<40}")
    print(f"  {'-'*55} {'-'*40}")
    apply("Baseline (no filter)", lambda t: True, n, baseline_wr)
    if anti:
        apply(f"Drop if any anti-signal fires", lambda t: all(t[it]==0 for it in anti), n, baseline_wr)
    if strong:
        apply(f"Require ≥1 strong signal", lambda t: any(t[it]==1 for it in strong), n, baseline_wr)
    if anti and strong:
        apply(f"Drop anti + require strong",
              lambda t: all(t[it]==0 for it in anti) and any(t[it]==1 for it in strong),
              n, baseline_wr)
    if worst_pairs:
        apply(f"Drop if any worst-pair both fires",
              lambda t: not any(t[a]==1 and t[b]==1 for a,b in worst_pairs),
              n, baseline_wr)

    # Day-of-week + direction filters
    apply("Wed/Thu only",
          lambda t: datetime.strptime(t["date"],"%Y-%m-%d").strftime("%a") in ("Wed","Thu"),
          n, baseline_wr)

    # Combined multi-filter
    if anti:
        apply(f"Wed/Thu + drop anti-signals",
              lambda t: (datetime.strptime(t["date"],"%Y-%m-%d").strftime("%a") in ("Wed","Thu")
                         and all(t[it]==0 for it in anti)),
              n, baseline_wr)

    # Top pair must be present
    pair_stats.sort(key=lambda x:-x[3])
    top_pairs = [(a,b) for a,b,n_,wr,exp,z,_ in pair_stats[:5] if z>2]
    if top_pairs:
        print(f"\n  Top pairs as mandatory-inclusion: {top_pairs}")
        apply(f"Require ≥1 top-pair",
              lambda t: any(t[a]==1 and t[b]==1 for a,b in top_pairs),
              n, baseline_wr)

if __name__=="__main__":
    main()
