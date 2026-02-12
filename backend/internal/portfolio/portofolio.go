package portfolio

import (
	"fmt"
	"math"
	"strings"

	"set-and-trend/backend/internal/constants"
	"set-and-trend/backend/internal/engine"
	"set-and-trend/backend/internal/patterns"
)

type Portfolio struct {
	equity      float64
	leverage    float64
	totalPnL    float64
	totalTrades int
	wins        int
	losses      int
	peakEquity  float64
	maxDD       float64
}

func New(capital, leverage float64) *Portfolio {
	return &Portfolio{
		equity:     capital,
		leverage:   leverage,
		peakEquity: capital,
	}
}

func (p *Portfolio) Execute(signals []*engine.BacktestSignal) {
	for _, sig := range signals {
		size := p.calculateSize(sig)
		pnl := p.realisticPnL(sig, size)
		
		p.totalPnL += pnl
		p.equity += pnl
		p.totalTrades++
		if pnl > 0 { 
			p.wins++ 
		} else { 
			p.losses++ 
		}
		
		if p.equity > p.peakEquity {
			p.peakEquity = p.equity
		}
		dd := (p.peakEquity - p.equity) / p.peakEquity
		if dd > p.maxDD { 
			p.maxDD = dd 
		}
		
		fmt.Printf("💰 %-8s %4s @%.5f → $%6.0f (%.1f%%)\n", 
			sig.Symbol, sig.Direction, sig.EntryPrice, pnl, pnl/p.equity*100)
	}
}

func (p *Portfolio) calculateSize(sig *engine.BacktestSignal) float64 {
	riskAmount := p.equity * 0.02
	stopDistance := math.Abs(sig.EntryPrice - sig.StopLoss)
	
	// ✅ FIXED: Use new Symbol API
	symbolConfig := constants.MustGetSymbolConfig(sig.Symbol)
	pipValue := symbolConfig.PipValue
	
	lots := riskAmount / (stopDistance / symbolConfig.PipSize * pipValue)
	
	// Calculate margin requirement
	contractSize := symbolConfig.ContractSize
	margin := lots * sig.EntryPrice * contractSize / p.leverage
	
	if margin > p.equity { 
		return 0 
	}
	
	return math.Max(lots, symbolConfig.MinLot)
}

func (p *Portfolio) realisticPnL(sig *engine.BacktestSignal, size float64) float64 {

	if size == 0 {
		return 0
	}
	
	riskDistance := math.Abs(sig.EntryPrice - sig.StopLoss)
	
	// ✅ FIXED: Use new Symbol API
	symbolConfig := constants.MustGetSymbolConfig(sig.Symbol)
	pipValue := symbolConfig.PipValue
	pipSize := symbolConfig.PipSize
	
	// Find entry candle
	entryIdx := 0
	for i, c := range sig.FutureCandles {
		if math.Abs(c.Close-sig.EntryPrice) < riskDistance*0.1 {
			entryIdx = i + 1
			break
		}
	}
	
	// Simulate trade execution
	for i := entryIdx; i < len(sig.FutureCandles); i++ {
		c := sig.FutureCandles[i]
		hitTP, hitSL := false, false
		
		// ✅ FIXED: Use patterns package constant
		if sig.Direction == patterns.DirectionLong {
			hitTP = c.High >= sig.TakeProfit
			hitSL = c.Low <= sig.StopLoss
		} else {
			hitTP = c.Low <= sig.TakeProfit
			hitSL = c.High >= sig.StopLoss
		}
		
		// Conservative: if both hit in same candle, assume SL hit first
		if hitTP && hitSL { 
			pips := riskDistance / pipSize
			return -pips * size * pipValue 
		}
		
		if hitTP { 
			pips := riskDistance / pipSize * sig.RiskReward
			return pips * size * pipValue 
		}
		
		if hitSL { 
			pips := riskDistance / pipSize
			return -pips * size * pipValue 
		}
	}
	
	// Trade didn't close - calculate floating PnL
	last := sig.FutureCandles[len(sig.FutureCandles)-1].Close
	distance := math.Abs(last - sig.EntryPrice)
	pips := distance / pipSize
	
	if sig.Direction == patterns.DirectionLong {
		if last > sig.EntryPrice {
			return pips * size * pipValue
		}
		return -pips * size * pipValue
	} else {
		if last < sig.EntryPrice {
			return pips * size * pipValue
		}
		return -pips * size * pipValue
	}
}

func (p *Portfolio) Report() {
	if p.totalTrades == 0 {
		fmt.Println("\n⚠️  No trades executed")
		return
	}
	
	winRate := float64(p.wins) / float64(p.totalTrades) * 100
	ret := (p.equity - 2000) / 2000 * 100
	
	fmt.Printf("\n"+strings.Repeat("=", 60)+"\n")
	fmt.Printf("🏆 ELITE PORTFOLIO: $2000 → $%.0f (%.0f%%)\n", p.equity, ret)
	fmt.Printf("📊 Win: %d | Loss: %d | Win Rate: %.1f%%\n", p.wins, p.losses, winRate)
	fmt.Printf("📉 Max DD: %.1f%% | Trades: %d\n", p.maxDD*100, p.totalTrades)
	fmt.Printf("💵 Total PnL: $%.0f | Avg PnL/Trade: $%.0f\n", p.totalPnL, p.totalPnL/float64(p.totalTrades))
	fmt.Println(strings.Repeat("=", 60))
}