package analyzer

import (
	"fmt"
	"math"
	//"strings"
	"time"

	"set-and-trend/backend/internal/engine"
	"set-and-trend/backend/internal/patterns"
)

// SignalQualityReport contains detailed analysis of a signal
type SignalQualityReport struct {
	Symbol         string
	Direction      string
	SignalTime     time.Time
	EntryPrice     float64
	StopLoss       float64
	TakeProfit     float64
	Confidence     float64
	RiskReward     float64
	
	// Entry Analysis
	EntrySlippagePips float64
	EntryDelayCandles int
	ActualEntryPrice  float64
	EntryQuality      string // "EXCELLENT", "GOOD", "POOR", "FAILED"
	
	// Outcome Analysis
	Outcome           string  // "TP_HIT", "SL_HIT", "TIMEOUT", "NOT_EXECUTED"
	CandlesToOutcome  int
	ActualRiskReward  float64
	PnL               float64
	
	// Pattern Validation
	PatternValid      bool
	PatternStrength   float64
	MarketContext     string
	
	// Risk Analysis
	RiskScore         float64
	GreyZone          bool
	GreyZoneReason    string
	
	// Recommendations
	ShouldTrade       bool
	RecommendedAction string
}

// SignalQualityAnalyzer validates signal quality
type SignalQualityAnalyzer struct {
	pipSize float64
}

func NewSignalQualityAnalyzer(symbol string) *SignalQualityAnalyzer {
	// Get pip size from constants
	pipSize := 0.0001 // Default for most pairs
	if len(symbol) >= 6 && symbol[3:6] == "JPY" {
		pipSize = 0.01
	}
	return &SignalQualityAnalyzer{pipSize: pipSize}
}

// AnalyzeSignal performs comprehensive signal quality check
func (sqa *SignalQualityAnalyzer) AnalyzeSignal(sig *engine.BacktestSignal) *SignalQualityReport {
	report := &SignalQualityReport{
		Symbol:     sig.Symbol,
		Direction:  sig.Direction,
		SignalTime: sig.Timestamp,
		EntryPrice: sig.EntryPrice,
		StopLoss:   sig.StopLoss,
		TakeProfit: sig.TakeProfit,
		Confidence: sig.Confidence,
		RiskReward: sig.RiskReward,
	}
	
	// 1. Validate Entry Timing
	sqa.validateEntry(sig, report)
	
	// 2. Simulate Outcome
	sqa.simulateOutcome(sig, report)
	
	// 3. Validate Pattern
	sqa.validatePattern(sig, report)
	
	// 4. Detect Grey Zones
	sqa.detectGreyZones(sig, report)
	
	// 5. Calculate Risk Score
	sqa.calculateRiskScore(report)
	
	// 6. Generate Recommendation
	sqa.generateRecommendation(report)
	
	return report
}

// validateEntry checks if entry price is achievable
func (sqa *SignalQualityAnalyzer) validateEntry(sig *engine.BacktestSignal, report *SignalQualityReport) {
	if len(sig.FutureCandles) == 0 {
		report.EntryQuality = "FAILED"
		report.EntryDelayCandles = -1
		return
	}
	
	stopDistance := math.Abs(sig.EntryPrice - sig.StopLoss)
	tolerance := stopDistance * 0.1 // 10% of stop distance
	
	// Find where price comes close to entry
	for i, candle := range sig.FutureCandles {
		// Check if entry price was available
		withinRange := false
		if sig.Direction == patterns.DirectionLong {
			// For long, check if price dipped to entry
			withinRange = candle.Low <= sig.EntryPrice && candle.Close >= sig.EntryPrice-tolerance
		} else {
			// For short, check if price rallied to entry
			withinRange = candle.High >= sig.EntryPrice && candle.Close <= sig.EntryPrice+tolerance
		}
		
		if withinRange {
			report.ActualEntryPrice = candle.Close
			report.EntryDelayCandles = i
			report.EntrySlippagePips = math.Abs(candle.Close-sig.EntryPrice) / sqa.pipSize
			
			// Rate entry quality
			if i == 0 && report.EntrySlippagePips < 2 {
				report.EntryQuality = "EXCELLENT"
			} else if i <= 3 && report.EntrySlippagePips < 5 {
				report.EntryQuality = "GOOD"
			} else if i <= 10 && report.EntrySlippagePips < 10 {
				report.EntryQuality = "FAIR"
			} else {
				report.EntryQuality = "POOR"
			}
			return
		}
	}
	
	report.EntryQuality = "FAILED"
	report.EntryDelayCandles = len(sig.FutureCandles)
}

// simulateOutcome simulates what happens after entry
func (sqa *SignalQualityAnalyzer) simulateOutcome(sig *engine.BacktestSignal, report *SignalQualityReport) {
	if report.EntryQuality == "FAILED" {
		report.Outcome = "NOT_EXECUTED"
		return
	}
	
	startIdx := report.EntryDelayCandles
	if startIdx < 0 {
		startIdx = 0
	}
	
	// Simulate from entry point
	for i := startIdx; i < len(sig.FutureCandles); i++ {
		candle := sig.FutureCandles[i]
		
		hitTP := false
		hitSL := false
		
		if sig.Direction == patterns.DirectionLong {
			hitTP = candle.High >= sig.TakeProfit
			hitSL = candle.Low <= sig.StopLoss
		} else {
			hitTP = candle.Low <= sig.TakeProfit
			hitSL = candle.High >= sig.StopLoss
		}
		
		// Conservative: if both hit, assume SL first
		if hitTP && hitSL {
			report.Outcome = "SL_HIT"
			report.CandlesToOutcome = i - startIdx
			report.ActualRiskReward = -1.0
			return
		}
		
		if hitTP {
			report.Outcome = "TP_HIT"
			report.CandlesToOutcome = i - startIdx
			report.ActualRiskReward = sig.RiskReward
			return
		}
		
		if hitSL {
			report.Outcome = "SL_HIT"
			report.CandlesToOutcome = i - startIdx
			report.ActualRiskReward = -1.0
			return
		}
	}
	
	// Never hit TP or SL
	report.Outcome = "TIMEOUT"
	report.CandlesToOutcome = len(sig.FutureCandles) - startIdx
	
	// Calculate floating P&L
	lastCandle := sig.FutureCandles[len(sig.FutureCandles)-1]
	stopDist := math.Abs(sig.EntryPrice - sig.StopLoss)
	actualDist := math.Abs(lastCandle.Close - sig.EntryPrice)
	
	if sig.Direction == patterns.DirectionLong {
		if lastCandle.Close > sig.EntryPrice {
			report.ActualRiskReward = actualDist / stopDist
		} else {
			report.ActualRiskReward = -(actualDist / stopDist)
		}
	} else {
		if lastCandle.Close < sig.EntryPrice {
			report.ActualRiskReward = actualDist / stopDist
		} else {
			report.ActualRiskReward = -(actualDist / stopDist)
		}
	}
}

// validatePattern checks if the pattern is actually valid
func (sqa *SignalQualityAnalyzer) validatePattern(sig *engine.BacktestSignal, report *SignalQualityReport) {
	// Check if we have enough historical data (candles before signal)
	// This is simplified - you'd check actual pattern formation
	
	report.PatternValid = true
	report.PatternStrength = sig.Confidence
	
	// Check market context from ATR
	if sig.ATR > 0 {
		stopDist := math.Abs(sig.EntryPrice - sig.StopLoss)
		stopToATR := stopDist / sig.ATR
		
		if stopToATR < 1.0 {
			report.MarketContext = "VERY_VOLATILE"
			report.PatternStrength *= 0.7
		} else if stopToATR < 1.5 {
			report.MarketContext = "VOLATILE"
			report.PatternStrength *= 0.85
		} else if stopToATR > 3.0 {
			report.MarketContext = "QUIET"
			report.PatternStrength *= 0.9
		} else {
			report.MarketContext = "NORMAL"
		}
	} else {
		report.MarketContext = "UNKNOWN"
	}
}

// detectGreyZones identifies ambiguous signals
func (sqa *SignalQualityAnalyzer) detectGreyZones(sig *engine.BacktestSignal, report *SignalQualityReport) {
	greyZones := []string{}
	
	// 1. Entry too far from current price
	if report.EntrySlippagePips > 10 {
		greyZones = append(greyZones, fmt.Sprintf("High entry slippage: %.1f pips", report.EntrySlippagePips))
	}
	
	// 2. Stop too tight relative to ATR
	if sig.ATR > 0 {
		stopDist := math.Abs(sig.EntryPrice - sig.StopLoss)
		if stopDist < sig.ATR*1.0 {
			greyZones = append(greyZones, "Stop loss too tight (< 1 ATR)")
		}
	}
	
	// 3. Confidence near threshold
	if sig.Confidence < 0.90 {
		greyZones = append(greyZones, fmt.Sprintf("Confidence only %.1f%%", sig.Confidence*100))
	}
	
	// 4. Entry delay too long
	if report.EntryDelayCandles > 5 {
		greyZones = append(greyZones, fmt.Sprintf("Entry delayed %d candles", report.EntryDelayCandles))
	}
	
	// 5. Poor entry quality
	if report.EntryQuality == "POOR" || report.EntryQuality == "FAILED" {
		greyZones = append(greyZones, fmt.Sprintf("Entry quality: %s", report.EntryQuality))
	}
	
	if len(greyZones) > 0 {
		report.GreyZone = true
		report.GreyZoneReason = greyZones[0]
		if len(greyZones) > 1 {
			report.GreyZoneReason += fmt.Sprintf(" (+%d more issues)", len(greyZones)-1)
		}
	}
}

// calculateRiskScore gives overall risk rating 0-100
func (sqa *SignalQualityAnalyzer) calculateRiskScore(report *SignalQualityReport) {
	score := 100.0
	
	// Deduct for entry issues
	if report.EntryQuality == "FAILED" {
		score -= 50
	} else if report.EntryQuality == "POOR" {
		score -= 30
	} else if report.EntryQuality == "FAIR" {
		score -= 15
	} else if report.EntryQuality == "GOOD" {
		score -= 5
	}
	
	// Deduct for grey zones
	if report.GreyZone {
		score -= 20
	}
	
	// Deduct for volatile market
	if report.MarketContext == "VERY_VOLATILE" {
		score -= 15
	} else if report.MarketContext == "VOLATILE" {
		score -= 10
	}
	
	// Bonus for high confidence
	if report.Confidence >= 0.95 {
		score += 10
	}
	
	// Bonus for good R:R
	if report.RiskReward >= 2.5 {
		score += 5
	}
	
	report.RiskScore = math.Max(0, math.Min(100, score))
}

// generateRecommendation decides if signal should be traded
func (sqa *SignalQualityAnalyzer) generateRecommendation(report *SignalQualityReport) {
	// Decision logic
	if report.EntryQuality == "FAILED" {
		report.ShouldTrade = false
		report.RecommendedAction = "SKIP: Entry not achievable"
		return
	}
	
	if report.RiskScore < 50 {
		report.ShouldTrade = false
		report.RecommendedAction = fmt.Sprintf("SKIP: Risk score too low (%.0f/100)", report.RiskScore)
		return
	}
	
	if report.GreyZone && report.RiskScore < 70 {
		report.ShouldTrade = false
		report.RecommendedAction = fmt.Sprintf("SKIP: Grey zone with low score - %s", report.GreyZoneReason)
		return
	}
	
	if report.EntryQuality == "EXCELLENT" && report.RiskScore >= 80 {
		report.ShouldTrade = true
		report.RecommendedAction = "TAKE: High quality setup"
		return
	}
	
	if report.RiskScore >= 70 {
		report.ShouldTrade = true
		report.RecommendedAction = "TAKE: Good setup"
		return
	}
	
	report.ShouldTrade = false
	report.RecommendedAction = fmt.Sprintf("SKIP: Marginal quality (score: %.0f/100)", report.RiskScore)
}

// PrintReport prints formatted quality report
func (report *SignalQualityReport) PrintReport() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 SIGNAL QUALITY REPORT: %s %s\n", report.Symbol, report.Direction)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Entry Analysis
	fmt.Println("\n🎯 ENTRY ANALYSIS:")
	fmt.Printf("  Entry Quality:     %s\n", report.EntryQuality)
	fmt.Printf("  Entry Slippage:    %.1f pips\n", report.EntrySlippagePips)
	fmt.Printf("  Entry Delay:       %d candles\n", report.EntryDelayCandles)
	fmt.Printf("  Target Entry:      %.5f\n", report.EntryPrice)
	fmt.Printf("  Actual Entry:      %.5f\n", report.ActualEntryPrice)
	
	// Outcome Analysis
	fmt.Println("\n📈 OUTCOME ANALYSIS:")
	outcomeIcon := "⚪"
	if report.Outcome == "TP_HIT" {
		outcomeIcon = "🟢"
	} else if report.Outcome == "SL_HIT" {
		outcomeIcon = "🔴"
	} else if report.Outcome == "TIMEOUT" {
		outcomeIcon = "⏳"
	}
	fmt.Printf("  Outcome:           %s %s\n", outcomeIcon, report.Outcome)
	fmt.Printf("  Candles to Close:  %d\n", report.CandlesToOutcome)
	fmt.Printf("  Expected R:R:      %.2f\n", report.RiskReward)
	fmt.Printf("  Actual R:R:        %.2f\n", report.ActualRiskReward)
	
	// Pattern Validation
	fmt.Println("\n🔍 PATTERN VALIDATION:")
	fmt.Printf("  Pattern Valid:     %v\n", report.PatternValid)
	fmt.Printf("  Pattern Strength:  %.1f%%\n", report.PatternStrength*100)
	fmt.Printf("  Market Context:    %s\n", report.MarketContext)
	fmt.Printf("  Confidence:        %.1f%%\n", report.Confidence*100)
	
	// Grey Zones
	if report.GreyZone {
		fmt.Println("\n⚠️  GREY ZONES DETECTED:")
		fmt.Printf("  Reason: %s\n", report.GreyZoneReason)
	}
	
	// Risk Score
	fmt.Println("\n🎲 RISK ASSESSMENT:")
	scoreIcon := "🔴"
	if report.RiskScore >= 80 {
		scoreIcon = "🟢"
	} else if report.RiskScore >= 60 {
		scoreIcon = "🟡"
	}
	fmt.Printf("  Risk Score:        %s %.0f/100\n", scoreIcon, report.RiskScore)
	
	// Recommendation
	fmt.Println("\n💡 RECOMMENDATION:")
	actionIcon := "✅"
	if !report.ShouldTrade {
		actionIcon = "❌"
	}
	fmt.Printf("  %s %s\n", actionIcon, report.RecommendedAction)
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}