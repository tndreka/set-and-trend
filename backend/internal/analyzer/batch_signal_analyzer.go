package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"set-and-trend/backend/internal/engine"
)

// BatchAnalysisResult contains aggregate statistics
type BatchAnalysisResult struct {
	TotalSignals      int
	GoodSignals       int
	PoorSignals       int
	FailedSignals     int
	
	AvgRiskScore      float64
	AvgEntrySlippage  float64
	AvgEntryDelay     float64
	
	OutcomeCounts     map[string]int
	EntryQualityCounts map[string]int
	
	Reports           []*SignalQualityReport
	
	// Win rate by quality
	ExcellentWinRate  float64
	GoodWinRate       float64
	FairWinRate       float64
	PoorWinRate       float64
	
	// Should vs Shouldn't trade comparison
	ShouldTradeWinRate    float64
	ShouldntTradeWinRate  float64
}

// BatchAnalyzer analyzes multiple signals
type BatchAnalyzer struct {
	analyzers map[string]*SignalQualityAnalyzer
}

func NewBatchAnalyzer() *BatchAnalyzer {
	return &BatchAnalyzer{
		analyzers: make(map[string]*SignalQualityAnalyzer),
	}
}

// AnalyzeAll analyzes all signals and generates statistics
func (ba *BatchAnalyzer) AnalyzeAll(signals []*engine.BacktestSignal) *BatchAnalysisResult {
	result := &BatchAnalysisResult{
		TotalSignals:       len(signals),
		OutcomeCounts:      make(map[string]int),
		EntryQualityCounts: make(map[string]int),
		Reports:            make([]*SignalQualityReport, 0, len(signals)),
	}
	
	totalRiskScore := 0.0
	totalSlippage := 0.0
	totalDelay := 0.0
	validSlippageCount := 0
	validDelayCount := 0
	
	// Track wins by quality
	qualityWins := make(map[string]int)
	qualityTotal := make(map[string]int)
	shouldTradeWins := 0
	shouldTradeTotal := 0
	shouldntTradeWins := 0
	shouldntTradeTotal := 0
	
	for _, sig := range signals {
		// Get or create analyzer for this symbol
		analyzer, exists := ba.analyzers[sig.Symbol]
		if !exists {
			analyzer = NewSignalQualityAnalyzer(sig.Symbol)
			ba.analyzers[sig.Symbol] = analyzer
		}
		
		// Analyze signal
		report := analyzer.AnalyzeSignal(sig)
		result.Reports = append(result.Reports, report)
		
		// Aggregate stats
		totalRiskScore += report.RiskScore
		
		if report.EntryQuality != "FAILED" {
			totalSlippage += report.EntrySlippagePips
			validSlippageCount++
			
			if report.EntryDelayCandles >= 0 {
				totalDelay += float64(report.EntryDelayCandles)
				validDelayCount++
			}
		}
		
		// Count outcomes
		result.OutcomeCounts[report.Outcome]++
		result.EntryQualityCounts[report.EntryQuality]++
		
		// Count quality categories
		if report.RiskScore >= 70 && report.EntryQuality != "FAILED" {
			result.GoodSignals++
		} else if report.RiskScore >= 50 {
			result.PoorSignals++
		} else {
			result.FailedSignals++
		}
		
		// Track wins by entry quality
		qualityTotal[report.EntryQuality]++
		if report.Outcome == "TP_HIT" {
			qualityWins[report.EntryQuality]++
		}
		
		// Track should vs shouldn't trade
		if report.ShouldTrade {
			shouldTradeTotal++
			if report.Outcome == "TP_HIT" {
				shouldTradeWins++
			}
		} else {
			shouldntTradeTotal++
			if report.Outcome == "TP_HIT" {
				shouldntTradeWins++
			}
		}
	}
	
	// Calculate averages
	if result.TotalSignals > 0 {
		result.AvgRiskScore = totalRiskScore / float64(result.TotalSignals)
	}
	if validSlippageCount > 0 {
		result.AvgEntrySlippage = totalSlippage / float64(validSlippageCount)
	}
	if validDelayCount > 0 {
		result.AvgEntryDelay = totalDelay / float64(validDelayCount)
	}
	
	// Calculate win rates by quality
	if qualityTotal["EXCELLENT"] > 0 {
		result.ExcellentWinRate = float64(qualityWins["EXCELLENT"]) / float64(qualityTotal["EXCELLENT"]) * 100
	}
	if qualityTotal["GOOD"] > 0 {
		result.GoodWinRate = float64(qualityWins["GOOD"]) / float64(qualityTotal["GOOD"]) * 100
	}
	if qualityTotal["FAIR"] > 0 {
		result.FairWinRate = float64(qualityWins["FAIR"]) / float64(qualityTotal["FAIR"]) * 100
	}
	if qualityTotal["POOR"] > 0 {
		result.PoorWinRate = float64(qualityWins["POOR"]) / float64(qualityTotal["POOR"]) * 100
	}
	
	// Calculate should vs shouldn't trade win rates
	if shouldTradeTotal > 0 {
		result.ShouldTradeWinRate = float64(shouldTradeWins) / float64(shouldTradeTotal) * 100
	}
	if shouldntTradeTotal > 0 {
		result.ShouldntTradeWinRate = float64(shouldntTradeWins) / float64(shouldntTradeTotal) * 100
	}
	
	return result
}

// PrintSummary prints aggregate statistics
func (result *BatchAnalysisResult) PrintSummary() {
	fmt.Println("\n" + strings.Repeat("═", 70))
	// fmt.Println("📊 SIGNAL QUALITY SUMMARY - ALL 13 SIGNALS")
	fmt.Printf("📊 SIGNAL QUALITY SUMMARY - ALL %d SIGNALS\n", result.TotalSignals)
	fmt.Println(strings.Repeat("═", 70))
	
	// Overall Quality Distribution
	fmt.Println("\n🎯 QUALITY DISTRIBUTION:")
	fmt.Printf("  Good Signals:      %d/%d (%.1f%%)\n", 
		result.GoodSignals, result.TotalSignals, 
		float64(result.GoodSignals)/float64(result.TotalSignals)*100)
	fmt.Printf("  Poor Signals:      %d/%d (%.1f%%)\n", 
		result.PoorSignals, result.TotalSignals,
		float64(result.PoorSignals)/float64(result.TotalSignals)*100)
	fmt.Printf("  Failed Signals:    %d/%d (%.1f%%)\n", 
		result.FailedSignals, result.TotalSignals,
		float64(result.FailedSignals)/float64(result.TotalSignals)*100)
	
	// Entry Quality
	fmt.Println("\n🎯 ENTRY QUALITY BREAKDOWN:")
	for _, quality := range []string{"EXCELLENT", "GOOD", "FAIR", "POOR", "FAILED"} {
		count := result.EntryQualityCounts[quality]
		if count > 0 {
			icon := "🟢"
			if quality == "POOR" {
				icon = "🟡"
			} else if quality == "FAILED" {
				icon = "🔴"
			}
			fmt.Printf("  %s %-10s: %d signals\n", icon, quality, count)
		}
	}
	
	// Averages
	fmt.Println("\n📊 AVERAGE METRICS:")
	fmt.Printf("  Avg Risk Score:    %.1f/100\n", result.AvgRiskScore)
	fmt.Printf("  Avg Entry Slip:    %.1f pips\n", result.AvgEntrySlippage)
	fmt.Printf("  Avg Entry Delay:   %.1f candles\n", result.AvgEntryDelay)
	
	// Outcomes
	fmt.Println("\n📈 OUTCOMES:")
	totalOutcomes := 0
	for _, count := range result.OutcomeCounts {
		totalOutcomes += count
	}
	
	for _, outcome := range []string{"TP_HIT", "SL_HIT", "TIMEOUT", "NOT_EXECUTED"} {
		count := result.OutcomeCounts[outcome]
		if count > 0 {
			icon := "🟢"
			if outcome == "SL_HIT" {
				icon = "🔴"
			} else if outcome == "TIMEOUT" {
				icon = "⏳"
			} else if outcome == "NOT_EXECUTED" {
				icon = "❌"
			}
			pct := float64(count) / float64(totalOutcomes) * 100
			fmt.Printf("  %s %-15s: %d signals (%.1f%%)\n", icon, outcome, count, pct)
		}
	}
	
	// Win Rate by Entry Quality
	fmt.Println("\n🏆 WIN RATE BY ENTRY QUALITY:")
	if result.EntryQualityCounts["EXCELLENT"] > 0 {
		fmt.Printf("  EXCELLENT: %.1f%%\n", result.ExcellentWinRate)
	}
	if result.EntryQualityCounts["GOOD"] > 0 {
		fmt.Printf("  GOOD:      %.1f%%\n", result.GoodWinRate)
	}
	if result.EntryQualityCounts["FAIR"] > 0 {
		fmt.Printf("  FAIR:      %.1f%%\n", result.FairWinRate)
	}
	if result.EntryQualityCounts["POOR"] > 0 {
		fmt.Printf("  POOR:      %.1f%%\n", result.PoorWinRate)
	}
	
	// Should vs Shouldn't Trade
	fmt.Println("\n💡 RECOMMENDATION VALIDATION:")
	fmt.Printf("  Signals we SHOULD trade:   %.1f%% win rate\n", result.ShouldTradeWinRate)
	fmt.Printf("  Signals we SHOULDN'T trade: %.1f%% win rate\n", result.ShouldntTradeWinRate)
	
	fmt.Println("\n" + strings.Repeat("═", 70))
}

// GetWorstSignals returns bottom N signals by risk score
func (result *BatchAnalysisResult) GetWorstSignals(n int) []*SignalQualityReport {
	sorted := make([]*SignalQualityReport, len(result.Reports))
	copy(sorted, result.Reports)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RiskScore < sorted[j].RiskScore
	})
	
	if n > len(sorted) {
		n = len(sorted)
	}
	
	return sorted[:n]
}

// GetBestSignals returns top N signals by risk score
func (result *BatchAnalysisResult) GetBestSignals(n int) []*SignalQualityReport {
	sorted := make([]*SignalQualityReport, len(result.Reports))
	copy(sorted, result.Reports)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RiskScore > sorted[j].RiskScore
	})
	
	if n > len(sorted) {
		n = len(sorted)
	}
	
	return sorted[:n]
}

// PrintDetailedAnalysis prints full reports for all signals
func (result *BatchAnalysisResult) PrintDetailedAnalysis() {
	fmt.Println("\n\n" + strings.Repeat("═", 70))
	fmt.Println("📋 DETAILED ANALYSIS - ALL SIGNALS")
	fmt.Println(strings.Repeat("═", 70))
	
	for i, report := range result.Reports {
		fmt.Printf("\n[%d/%d] ", i+1, result.TotalSignals)
		report.PrintReport()
	}
}

// PrintWorstSignals prints the worst performing signals
func (result *BatchAnalysisResult) PrintWorstSignals(n int) {
	worst := result.GetWorstSignals(n)
	
	fmt.Println("\n\n" + strings.Repeat("═", 70))
	fmt.Printf("🔴 TOP %d WORST SIGNALS (Lowest Risk Score)\n", n)
	fmt.Println(strings.Repeat("═", 70))
	
	for i, report := range worst {
		fmt.Printf("\n[%d] ", i+1)
		report.PrintReport()
	}
}

// PrintBestSignals prints the best performing signals
func (result *BatchAnalysisResult) PrintBestSignals(n int) {
	best := result.GetBestSignals(n)
	
	fmt.Println("\n\n" + strings.Repeat("═", 70))
	fmt.Printf("🟢 TOP %d BEST SIGNALS (Highest Risk Score)\n", n)
	fmt.Println(strings.Repeat("═", 70))
	
	for i, report := range best {
		fmt.Printf("\n[%d] ", i+1)
		report.PrintReport()
	}
}