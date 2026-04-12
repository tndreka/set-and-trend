package rules

import (
	"fmt"
	"math"

	"set-and-trend/backend/internal/patterns"
)

func init() {
	SetupBuilders[H4SupplyDemand] = buildH4SupplyDemandSetup
}

// buildH4SupplyDemandSetup fires when:
//
//  1. There is an active (unbroken, non-stale) demand zone
//  2. The latest bar's low has tagged that zone (price reached the proximal edge)
//  3. The latest bar is a bullish rejection (close > open, body >= 50% of range)
//
// Setup geometry:
//
//	Entry  = latest close
//	SL     = zone distal price (the further edge below)
//	TP     = nearest active supply zone proximal, or entry + 2R if none
func buildH4SupplyDemandSetup(ctx EvalContext) (*Setup, error) {
	if len(ctx.Candles) < 50 {
		return nil, nil
	}
	candle, _, ok := ctx.Latest()
	if !ok {
		return nil, fmt.Errorf("h4_supply_demand: empty eval context")
	}

	patternCandles := toPatternCandles(ctx.Candles)
	cfg := patterns.DefaultZoneConfig(ctx.Symbol)
	zones := patterns.DetectSupplyDemandZones(patternCandles, cfg)
	active := patterns.GetActiveZones(patternCandles, zones, cfg.ZoneFreshnessBars)

	var demandHit *patterns.SupplyDemandZone
	for i := range active {
		z := &active[i]
		if z.Type != patterns.ZoneDemand {
			continue
		}
		// "Tagged" = the bar reached into the zone but didn't close through it.
		if candle.Low <= z.ProximalPrice && candle.Close > z.DistalPrice {
			demandHit = z
			break
		}
	}
	if demandHit == nil {
		return nil, nil
	}

	if !isBullishRejection(candle) {
		return nil, nil
	}

	entry := candle.Close
	// SL just below the distal edge (a few pips of buffer would normally come
	// from the symbol config, but the lab is EURUSD-only for Phase 1 so we
	// use a fixed 2-pip cushion).
	sl := demandHit.DistalPrice - 0.0002
	risk := entry - sl
	if risk <= 0 {
		return nil, nil
	}

	// TP = nearest active supply proximal above entry, else 2R fallback.
	tp := entry + 2*risk
	bestSupply := math.MaxFloat64
	for _, z := range active {
		if z.Type != patterns.ZoneSupply {
			continue
		}
		if z.ProximalPrice > entry && z.ProximalPrice < bestSupply {
			bestSupply = z.ProximalPrice
		}
	}
	if bestSupply != math.MaxFloat64 {
		tp = bestSupply
	}

	rr := computeRR("LONG", entry, sl, tp)
	if rr <= 0 {
		return nil, nil
	}

	return &Setup{
		Direction:  "LONG",
		Entry:      entry,
		StopLoss:   sl,
		TakeProfit: tp,
		RR:         rr,
		Confidence: demandHit.Confidence,
		Reason:     "Bullish rejection at active H4 demand zone",
	}, nil
}

// isBullishRejection: close above open, body at least half the range (a soft
// version of patterns.isStrongBullishCandle which is unexported).
func isBullishRejection(c Candle) bool {
	rng := c.High - c.Low
	if rng <= 0 {
		return false
	}
	body := c.Close - c.Open
	if body <= 0 {
		return false
	}
	return body/rng >= 0.5
}
