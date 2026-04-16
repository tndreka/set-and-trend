#!/usr/bin/env bash
# backtest_iter.sh — run one SaF backtest iteration and diff against previous.
#
# Usage: ./scripts/backtest_iter.sh <iter_number> <iter_name>
#   ./scripts/backtest_iter.sh 0 baseline
#   ./scripts/backtest_iter.sh 1 session-killzone
#
# Writes results/iter_NN_<name>.json and prints a side-by-side diff vs. the
# previous iteration file (if any).

set -euo pipefail

ITER="${1:?usage: $0 <iter> <name>}"
NAME="${2:?usage: $0 <iter> <name>}"

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULTS_DIR="${HERE}/results"
mkdir -p "$RESULTS_DIR"

PADDED=$(printf "%02d" "$ITER")
OUT="${RESULTS_DIR}/iter_${PADDED}_${NAME}.json"

echo "▶️  Running iter ${PADDED} (${NAME}) → ${OUT}"
(cd "$HERE" && go run ./cmd/backtest --engine=saf --iter="$ITER" --name="$NAME" --out="$OUT")

# Find previous iteration file by padded prefix.
PREV_ITER=$((ITER - 1))
PREV_PADDED=$(printf "%02d" "$PREV_ITER")
PREV=$(ls "${RESULTS_DIR}/iter_${PREV_PADDED}_"*.json 2>/dev/null | head -1 || true)

if [ -z "$PREV" ] || [ "$ITER" -eq 0 ]; then
  echo ""
  echo "📊 iter ${PADDED} absolute numbers:"
  jq '.aggregate' "$OUT"
  exit 0
fi

echo ""
echo "📈 Diff: ${PREV##*/}  →  ${OUT##*/}"
jq -s '
  .[0].aggregate as $p
  | .[1].aggregate as $c
  | {
      iter:       [.[0].iter,  .[1].iter],
      name:       [.[0].iter_name, .[1].iter_name],
      trades:     {prev: $p.trades, curr: $c.trades, delta: ($c.trades - $p.trades)},
      win_rate:   {prev: ($p.win_rate*100 | .*10 | round / 10), curr: ($c.win_rate*100 | .*10 | round / 10), delta_pp: (($c.win_rate - $p.win_rate)*100 | .*10 | round / 10)},
      expectancy: {prev: $p.expectancy_r, curr: $c.expectancy_r, delta: ($c.expectancy_r - $p.expectancy_r)},
      total_pnl_r:{prev: $p.total_pnl_r, curr: $c.total_pnl_r, delta: ($c.total_pnl_r - $p.total_pnl_r)},
      max_dd_r:   {prev: $p.max_drawdown_r, curr: $c.max_drawdown_r, delta: ($c.max_drawdown_r - $p.max_drawdown_r)}
    }' "$PREV" "$OUT"
