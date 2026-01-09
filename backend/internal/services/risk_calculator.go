package services

import (
	"errors"
	"fmt"
	"math"
)

// ComputeRiskAmount calculates the dollar amount risked based on balance and percentage
// Example: balance=$10,000, riskPct=1.0 → $100
func ComputeRiskAmount(balance float64, riskPct float64) (float64, error) {
	if balance <= 0 {
		return 0, errors.New("balance must be positive")
	}
	if riskPct < 0 || riskPct > 100 {
		return 0, errors.New("risk percentage must be between 0 and 100")
	}
	
	return balance * (riskPct / 100.0), nil
}

// ComputeStopDistance calculates distance from entry to stop loss in price units
// Assumes geometry is already validated via ValidateTradeGeometry
// For LONG: entry - sl (e.g., 1.1050 - 1.1000 = 0.0050)
// For SHORT: sl - entry (e.g., 1.1000 - 1.1050 = 0.0050)
func ComputeStopDistance(entry, sl float64) (float64, error) {
	if entry <= 0 || sl <= 0 {
		return 0, errors.New("entry and sl must be positive")
	}
	
	distance := math.Abs(entry - sl)
	
	if distance == 0 {
		return 0, errors.New("entry and sl cannot be equal")
	}
	
	return distance, nil
}

// ComputeStopDistancePips converts price distance to pips
// NOTE: pipValue is the price increment (e.g., 0.0001 for EURUSD)
// This is NOT pip value per lot ($10) - that's used in ComputePositionSize
// For EURUSD: 0.0050 / 0.0001 = 50 pips
func ComputeStopDistancePips(stopDistance, pipValue float64) (float64, error) {
	if pipValue <= 0 {
		return 0, errors.New("pip value must be positive")
	}
	if stopDistance <= 0 {
		return 0, errors.New("stop distance must be positive")
	}
	
	return stopDistance / pipValue, nil
}

// ComputePositionSize calculates lot size based on risk and stop distance
// NOTE: pipValuePerLot is the dollar value of one pip for one lot (e.g., $10 for EURUSD)
// Formula: position_size = risk_amount / (stop_distance_pips * pip_value_per_lot)
// Example: $100 risk / (50 pips * $10/pip) = 0.2 lots
func ComputePositionSize(
	riskAmount float64,
	stopDistancePips float64,
	pipValuePerLot float64,
) (float64, error) {
	if riskAmount <= 0 {
		return 0, errors.New("risk amount must be positive")
	}
	if stopDistancePips <= 0 {
		return 0, errors.New("stop distance must be positive")
	}
	if pipValuePerLot <= 0 {
		return 0, errors.New("pip value per lot must be positive")
	}
	
	positionSize := riskAmount / (stopDistancePips * pipValuePerLot)
	
	return positionSize, nil
}

// ComputeRR calculates risk-reward ratio
// Assumes geometry is already validated via ValidateTradeGeometry
// For LONG: RR = (tp - entry) / (entry - sl)
// For SHORT: RR = (entry - tp) / (sl - entry)
func ComputeRR(entry, sl, tp float64, bias string) (float64, error) {
	if entry <= 0 || sl <= 0 || tp <= 0 {
		return 0, errors.New("all prices must be positive")
	}
	
	var reward, risk float64
	
	if bias == "long" {
		reward = tp - entry
		risk = entry - sl
	} else if bias == "short" {
		reward = entry - tp
		risk = sl - entry
	} else {
		return 0, errors.New("bias must be 'long' or 'short'")
	}
	
	if risk <= 0 {
		return 0, errors.New("risk must be positive")
	}
	
	rr := reward / risk
	
	if rr <= 0 {
		return 0, errors.New("RR must be positive")
	}
	
	return rr, nil
}

// ValidateTradeGeometry performs all geometric validations
// This is the SINGLE SOURCE OF TRUTH for trade geometry
// All other functions assume input has passed this check
func ValidateTradeGeometry(entry, sl, tp float64, bias string) error {
	if entry <= 0 || sl <= 0 || tp <= 0 {
		return errors.New("all prices must be positive")
	}
	
	if bias == "long" {
		if sl >= entry {
			return errors.New("long trade: sl must be below entry")
		}
		if tp <= entry {
			return errors.New("long trade: tp must be above entry")
		}
	} else if bias == "short" {
		if sl <= entry {
			return errors.New("short trade: sl must be above entry")
		}
		if tp >= entry {
			return errors.New("short trade: tp must be below entry")
		}
	} else {
		return errors.New("bias must be 'long' or 'short'")
	}
	
	return nil
}

// ComputeMaxPositionSize calculates maximum position size based on leverage
// contractSize: units per lot (e.g., 100,000 for EURUSD standard lot)
func ComputeMaxPositionSize(balance float64, leverage int, contractSize float64) (float64, error) {
	if balance <= 0 {
		return 0, errors.New("balance must be positive")
	}
	if leverage <= 0 {
		return 0, errors.New("leverage must be positive")
	}
	if contractSize <= 0 {
		return 0, errors.New("contract size must be positive")
	}

	maxPositionSize := balance * float64(leverage) / contractSize
	return maxPositionSize, nil
}

// ValidateEntryPrice checks if actual entry is within tolerance of planned
func ValidateEntryPrice(
	plannedEntry float64,
	actualEntry float64,
	pipValue float64,
	maxSlippagePips float64,
) error {
	if plannedEntry <= 0 || actualEntry <= 0 {
		return errors.New("prices must be positive")
	}
	
	slippagePips := math.Abs(actualEntry-plannedEntry) / pipValue
	
	if slippagePips > maxSlippagePips {
		return fmt.Errorf(
			"entry slippage %.2f pips exceeds max %.2f pips",
			slippagePips,
			maxSlippagePips,
		)
	}
	
	return nil
}

// ComputeExecutionPnL calculates PnL for an execution event
func ComputeExecutionPnL(
	bias string,
	entryPrice float64,
	exitPrice float64,
	positionSize float64,
	pipValue float64,
) (pnlMoney float64, pnlPips float64, err error) {
	
	if entryPrice <= 0 || exitPrice <= 0 {
		return 0, 0, errors.New("prices must be positive")
	}
	
	var priceMove float64
	if bias == "long" {
		priceMove = exitPrice - entryPrice
	} else if bias == "short" {
		priceMove = entryPrice - exitPrice
	} else {
		return 0, 0, errors.New("bias must be long or short")
	}
	
	// Pips gained/lost
	pnlPips = priceMove / pipValue
	
	// Money gained/lost (for standard lot: 1 pip = $10)
	const pipValuePerLot = 10.0
	pnlMoney = pnlPips * positionSize * pipValuePerLot
	
	return pnlMoney, pnlPips, nil
}
