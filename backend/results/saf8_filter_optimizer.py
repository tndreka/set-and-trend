#!/usr/bin/env python3
"""
SAF >= 8 Filter Optimizer.

Combines the forensic findings into layered filter recipes and finds the
sweet spot: max WR while staying above 2/week target (~100 trades/yr).

Key findings from saf8_forensics.py:
  - h4_ema_touch is in 99% of trades; when OFF (rare), WR jumps to 52.8%
  - h4_in_favor is an anti-signal (-9.6pp lift)
  - d1_structure_rejection + h4_in_favor combo (2,677 trades) underperforms
  - d1_in_favor, d1_aoi, w1_candle_rejection, d1_candle_rejection are strong
  - Wed/Thu outperform Mon/Tue by ~6pp WR
  - h4_pattern (rare 13.7%) reduces timeout rate 4pp
  - Top triples hit 44-50% WR
"""

import csv
from collections import defaultdict
from datetime import datetime

CSV = "trade_analysis.csv"
ITEMS = ["w1_in_favor","w1_touching_ema","w1_candle_rejection","w1_pattern",
         "d1_in_favor","d1_aoi","d1_touching_ema","d1_candle_rejection",
         "d1_structure_rejection","d1_pattern",
         "h4_in_favor","h4_ema_touch","h4_candle_rejection","h4_pattern"]

def load():
    rows=[]
    with open(CSV) as f:
        for r in csv.DictReader(f):
            if r["strategy"]!="SAF_CHECKLIST": continue
            s=int(r["score"])
            if s<8: continue
            r["score"]=s; r["rMult"]=float(r["rMult"])
            r["dow"]=datetime.strptime(r["date"],"%Y-%m-%d").strftime("%a")
            for it in ITEMS: r[it]=int(r.get(it,0))
            rows.append(r)
    return rows

# --- Helper predicates ---
def has_top_pair(t):
    pairs=[("w1_touching_ema","w1_pattern"),
           ("w1_touching_ema","w1_candle_rejection"),
           ("d1_aoi","h4_pattern"),
           ("w1_candle_rejection","d1_aoi"),
           ("d1_aoi","d1_candle_rejection")]
    return any(t[a]==1 and t[b]==1 for a,b in pairs)

def has_bad_pair(t):
    return t["d1_structure_rejection"]==1 and t["h4_in_favor"]==1

def has_bad_triple(t):
    bad=[("w1_candle_rejection","w1_pattern","d1_candle_rejection"),
         ("w1_touching_ema","d1_candle_rejection","h4_pattern"),
         ("w1_touching_ema","h4_candle_rejection","h4_pattern"),
         ("w1_touching_ema","d1_pattern","h4_pattern"),
         ("w1_candle_rejection","h4_candle_rejection","h4_pattern"),
         ("w1_candle_rejection","w1_pattern","d1_structure_rejection"),
         ("w1_in_favor","w1_pattern","h4_pattern"),
         ("w1_touching_ema","h4_in_favor","h4_pattern"),
         ("w1_touching_ema","d1_touching_ema","h4_pattern"),
         ("w1_candle_rejection","d1_candle_rejection","d1_pattern")]
    return any(t[a]==1 and t[b]==1 and t[c]==1 for a,b,c in bad)

def evaluate(name, ts, years):
    if not ts: return
    n=len(ts)
    wins=sum(1 for t in ts if t["result"]=="win")
    losses=sum(1 for t in ts if t["result"]=="loss")
    tos=sum(1 for t in ts if t["result"]=="timeout")
    wr=wins/n
    exp=sum(t["rMult"] for t in ts)/n
    total_r=sum(t["rMult"] for t in ts)
    tpw=n/(years*52)
    print(f"  {name:<55} {n:>5} tr  {n/years:>5.1f}/yr  {tpw:>4.1f}/wk  "
          f"WR {wr*100:>5.1f}%  Exp {exp:+.3f}R  Total {total_r:+.0f}R  "
          f"Tmo {tos/n*100:.1f}%")

def main():
    trades=load()
    years=22.1
    print(f"Baseline SAF>=8: {len(trades)} trades\n")

    # Filter stacks — progressively tighter
    print("="*105)
    print(f"  FILTER LADDER — find the sweet spot for {' '*3}WR vs frequency")
    print("="*105)
    print(f"  Target: ≥2 trades/week (~100+/yr), highest possible WR")
    print()
    print(f"  {'Filter':<55} {'Tr':>5} {'Tr/yr':>7} {'Tr/wk':>6}  "
          f"{'WR':>8} {'Exp':>9} {'Total':>7} {'Tmo':>5}")
    print(f"  {'-'*55} {'-'*5} {'-'*7} {'-'*6}  {'-'*8} {'-'*9} {'-'*7} {'-'*5}")

    evaluate("1. Baseline (SAF>=8 all)", trades, years)
    evaluate("2. + Drop bad pair (d1_struct_rej + h4_in_favor)",
             [t for t in trades if not has_bad_pair(t)], years)
    evaluate("3. + Drop bad triples (10 worst)",
             [t for t in trades if not has_bad_triple(t)], years)
    evaluate("4. + Drop bad pair AND bad triples",
             [t for t in trades if not has_bad_pair(t) and not has_bad_triple(t)], years)
    evaluate("5. + Require >=1 top-pair",
             [t for t in trades if has_top_pair(t)], years)
    evaluate("6. + Require top-pair AND drop bad pair",
             [t for t in trades if has_top_pair(t) and not has_bad_pair(t)], years)
    evaluate("7. + Wed/Thu only",
             [t for t in trades if t["dow"] in ("Wed","Thu")], years)
    evaluate("8. + Wed/Thu + drop bad pair",
             [t for t in trades if t["dow"] in ("Wed","Thu") and not has_bad_pair(t)], years)
    evaluate("9. + Wed/Thu + top-pair required",
             [t for t in trades if t["dow"] in ("Wed","Thu") and has_top_pair(t)], years)
    evaluate("10. + Wed/Thu + top-pair + drop bad pair",
             [t for t in trades if t["dow"] in ("Wed","Thu") and has_top_pair(t)
              and not has_bad_pair(t)], years)
    evaluate("11. Require h4_pattern=1 (rare, resolves clean)",
             [t for t in trades if t["h4_pattern"]==1], years)
    evaluate("12. Require w1_candle_rejection=1",
             [t for t in trades if t["w1_candle_rejection"]==1], years)
    evaluate("13. Require d1_in_favor AND d1_aoi (both strong)",
             [t for t in trades if t["d1_in_favor"]==1 and t["d1_aoi"]==1], years)
    evaluate("14. STRICT: Wed/Thu + d1_in_favor + d1_aoi + no bad pair",
             [t for t in trades if t["dow"] in ("Wed","Thu") and t["d1_in_favor"]==1
              and t["d1_aoi"]==1 and not has_bad_pair(t)], years)
    evaluate("15. STRICT-2: top-pair + d1_in_favor + no bad triple",
             [t for t in trades if has_top_pair(t) and t["d1_in_favor"]==1
              and not has_bad_triple(t)], years)

    # ────────────────────────────────────────────────────────────
    print("\n" + "="*105)
    print(f"  BEST-FIT for target (≥2/wk target)")
    print("="*105)

    candidates = [
        ("4. Drop bad pair + bad triples", lambda t: not has_bad_pair(t) and not has_bad_triple(t)),
        ("5. Require top-pair", has_top_pair),
        ("6. Top-pair AND drop bad pair", lambda t: has_top_pair(t) and not has_bad_pair(t)),
        ("13. d1_in_favor AND d1_aoi", lambda t: t["d1_in_favor"]==1 and t["d1_aoi"]==1),
    ]

    for name, fn in candidates:
        ts=[t for t in trades if fn(t)]
        if not ts: continue
        n=len(ts)
        wins=sum(1 for t in ts if t["result"]=="win")
        wr=wins/n
        exp=sum(t["rMult"] for t in ts)/n
        tpw=n/(years*52)

        if tpw >= 2:
            gate="✅ hits 2/wk"
        elif tpw >= 1.5:
            gate="~ close"
        else:
            gate="❌ too slow"

        # Simulate 22-year equity on $100 and $500
        eq=100
        for t in ts:
            pnl = t["rMult"] * eq * 0.02  # 2% risk (realistic with small account)
            eq = max(eq + pnl, 0.01)
        print(f"\n  {name}")
        print(f"    {n} tr | {tpw:.2f}/wk | WR {wr*100:.1f}% | Exp {exp:+.3f}R | {gate}")
        print(f"    $100 @ 2% risk → ${eq:,.0f}  ({'WIN' if eq>100 else 'LOSS'})")

if __name__=="__main__":
    main()
