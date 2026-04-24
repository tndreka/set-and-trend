package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// dukaBar matches the JSON shape emitted by `dukascopy-node -f json`.
// Timestamp is epoch millis (UTC).
type dukaBar struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume,omitempty"`
}

// barsPerDay maps dukascopy-node timeframe strings to the approximate number
// of bars per calendar day, used to compute the date range for fetching.
var barsPerDay = map[string]int{
	"h1":  24,
	"h4":  6,
	"d1":  1,
	"mn1": 0, // weekly/monthly use week-based calculation
}

// fetchDukascopy shells out to `dukascopy-node` (via npx by default), parses
// the JSON file it drops into a temp dir, and returns bars sorted ascending.
//
// The CLI only accepts whole-day ranges (yyyy-mm-dd); we pad `from` back so
// that enough bars are captured even across weekends. `timeframe` is the
// dukascopy-node timeframe string: "h1", "h4", "d1", or "mn1".
func fetchDukascopy(ctx context.Context, symbol string, size int, timeframe string) ([]dukaBar, error) {
	tmp, err := os.MkdirTemp("", "duka-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	now := time.Now().UTC()
	var days int
	switch timeframe {
	case "d1":
		days = size + 4 // +4 for weekends
	case "mn1":
		days = size*7 + 4 // weekly bars
	default:
		bpd := barsPerDay[timeframe]
		if bpd == 0 {
			bpd = 6 // default to h4
		}
		days = (size / bpd) + 2
	}
	from := now.AddDate(0, 0, -days).Format("2006-01-02")
	to := now.AddDate(0, 0, 1).Format("2006-01-02")

	bin := os.Getenv("DUKASCOPY_NODE_BIN")
	var cmd *exec.Cmd
	if bin != "" {
		cmd = exec.CommandContext(ctx, bin,
			"-i", strings.ToLower(symbol),
			"-from", from, "-to", to,
			"-t", timeframe, "-f", "json", "-s",
			"-dir", tmp)
	} else {
		cmd = exec.CommandContext(ctx, "npx", "--yes", "dukascopy-node",
			"-i", strings.ToLower(symbol),
			"-from", from, "-to", to,
			"-t", timeframe, "-f", "json", "-s",
			"-dir", tmp)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dukascopy-node: %w (output: %s)", err, truncate(string(out), 300))
	}

	matches, err := filepath.Glob(filepath.Join(tmp, "*.json"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("dukascopy-node: no JSON output (stdout: %s)", truncate(string(out), 300))
	}

	// Merge all emitted files — dukascopy-node can split output across multiple
	// JSON artifacts when the range crosses year boundaries or cache chunks.
	var bars []dukaBar
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		var part []dukaBar
		if err := json.Unmarshal(data, &part); err != nil {
			return nil, fmt.Errorf("parse duka json %s: %w", filepath.Base(m), err)
		}
		bars = append(bars, part...)
	}
	// Dedupe by timestamp + sort ascending.
	seen := make(map[int64]struct{}, len(bars))
	out2 := bars[:0]
	for _, b := range bars {
		if _, dup := seen[b.Timestamp]; dup {
			continue
		}
		seen[b.Timestamp] = struct{}{}
		out2 = append(out2, b)
	}
	bars = out2
	sort.Slice(bars, func(i, j int) bool { return bars[i].Timestamp < bars[j].Timestamp })
	// CLI returns ascending already; trim to last `size` bars if too many.
	if len(bars) > size {
		bars = bars[len(bars)-size:]
	}
	return bars, nil
}

// floatStr formats a price float with 5 decimals so it round-trips cleanly
// into Postgres NUMERIC(12,5) without scientific notation.
func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 5, 64)
}
