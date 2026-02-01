package execution

type CostModel struct {
    Symbol string
    FixedSpreadPips float64    // EURUSD: 0.2, GBPJPY: 0.8
    SlippageModel string       // "fixed", "volatility"
    PipSize float64           // 0.0001 for most pairs, 0.01 for JPY
}

func (cm *CostModel) GetAsk(bid float64) float64 {
    return bid + (cm.FixedSpreadPips * cm.PipSize)
}

func (cm *CostModel) GetBid(ask float64) float64 {
    return ask - (cm.FixedSpreadPips * cm.PipSize)
}

// Entry price after spread applied
func (cm *CostModel) ApplyEntrySpread(price float64, direction string) float64 {
    if direction == "LONG" {
        return cm.GetAsk(price) // Buy at ask (worse)
    }
    return price // Sell at bid
}

// Exit price with spread + slippage
func (cm *CostModel) ApplyExitCost(price float64, direction string, isSL bool) float64 {
    slippage := cm.FixedSpreadPips * 0.5 // 50% of spread as slippage
    if isSL {
        slippage *= 2.0 // SL slips 2x worse than TP in volatility
    }
    
    if direction == "SHORT" {
        return price + (cm.FixedSpreadPips * cm.PipSize) + slippage // Cover at ask
    }
    return price - (cm.FixedSpreadPips * cm.PipSize) - slippage // Sell at bid
}