package scoring

import (
	"set-and-trend/backend/internal/mtf"
)

// ConfluenceScore holds the scoring breakdown
type ConfluenceScore struct {
	// Timeframe scores (0-6 points each)
	WeeklyScore  int
	DailyScore   int
	H4Score      int
	
	// Bonus points
	BonusPoints  int
	
	// Breakdown details
	Details      ScoreDetails
	
	// Summary
	Total        int
	MaxPossible  int
	Percentage   float64
	Grade        string // "A", "B", "C", "F"
}

// ScoreDetails provides breakdown of what contributed to the score
type ScoreDetails struct {
	// Weekly factors
	WeeklyTrendAligned     bool
	WeeklyAtOB             bool
	WeeklyAtFVG            bool
	WeeklyPinBar           bool
	WeeklyStructureReject  bool
	WeeklyPattern          bool
	
	// Daily factors
	DailyTrendAligned      bool
	DailyAtOB              bool
	DailyAtFVG             bool
	DailyPinBar            bool
	DailyStructureReject   bool
	DailyPattern           bool
	
	// H4 factors
	H4TrendAligned         bool
	H4AtOB                 bool
	H4AtFVG                bool
	H4FVGInFavor           bool
	H4PinBar               bool
	H4StructureReject      bool
	H4Pattern              bool
	
	// Bonus factors
	NearPsychLevel         bool
	EngulfingConfirmation  bool
	LiquiditySweep         bool
	RSIDivergence          bool
	CHoCHSignal            bool
}

// Signal represents the trade signal to score
type Signal struct {
	PatternType string  // "H&S" or "IHS"
	Direction   int     // -1=short, 1=long
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	BarIndex    int
}

// CalculateConfluence calculates the confluence score for a signal
func CalculateConfluence(ctx *mtf.MTFContext, signal *Signal) *ConfluenceScore {
	if ctx == nil || signal == nil {
		return &ConfluenceScore{Grade: "F"}
	}

	score := &ConfluenceScore{
		MaxPossible: 23, // 6 + 6 + 6 + 5 bonus
	}

	// Score each timeframe
	score.WeeklyScore = scoreTimeframe(ctx.Weekly, signal, &score.Details, "weekly")
	score.DailyScore = scoreTimeframe(ctx.Daily, signal, &score.Details, "daily")
	score.H4Score = scoreTimeframe(ctx.H4, signal, &score.Details, "h4")

	// Calculate bonus points
	score.BonusPoints = calculateBonusPoints(ctx, signal, &score.Details)

	// Calculate total
	score.Total = score.WeeklyScore + score.DailyScore + score.H4Score + score.BonusPoints
	score.Percentage = float64(score.Total) / float64(score.MaxPossible) * 100

	// Assign grade
	score.Grade = assignGrade(score.Total)

	return score
}

// scoreTimeframe scores a single timeframe context
func scoreTimeframe(tfCtx *mtf.TimeframeContext, signal *Signal, details *ScoreDetails, tfName string) int {
	if tfCtx == nil {
		return 0
	}

	points := 0

	// Factor 1: Trend alignment (EMA50 vs EMA200)
	trendAligned := checkTrendAlignment(tfCtx, signal.Direction)
	if trendAligned {
		points++
		setDetailFlag(details, tfName, "trend", true)
	}

	// Factor 2: At/Near Order Block
	atOB := checkAtOrderBlock(tfCtx, signal)
	if atOB {
		points++
		setDetailFlag(details, tfName, "ob", true)
	}

	// Factor 3: At/Near FVG
	atFVG := checkAtFVG(tfCtx, signal)
	if atFVG {
		points++
		setDetailFlag(details, tfName, "fvg", true)
	}

	// Factor 4: Pin Bar rejection
	hasPinBar := checkPinBarRejection(tfCtx, signal)
	if hasPinBar {
		points++
		setDetailFlag(details, tfName, "pinbar", true)
	}

	// Factor 5: Rejection from structure
	structureReject := checkStructureRejection(tfCtx, signal)
	if structureReject {
		points++
		setDetailFlag(details, tfName, "structure", true)
	}

	// Factor 6: Pattern on this timeframe (only for H4 since that's where we detect H&S)
	if tfName == "h4" && signal.PatternType != "" {
		points++
		details.H4Pattern = true
	}

	return points
}

// calculateBonusPoints calculates additional confluence factors
func calculateBonusPoints(ctx *mtf.MTFContext, signal *Signal, details *ScoreDetails) int {
	bonus := 0

	// Bonus 1: Near psychological level (+1)
	if ctx.H4 != nil && ctx.H4.NearPsychLevel != nil {
		bonus++
		details.NearPsychLevel = true
	}

	// Bonus 2: Engulfing confirmation (+1)
	if ctx.H4 != nil && hasEngulfingConfirmation(ctx.H4, signal) {
		bonus++
		details.EngulfingConfirmation = true
	}

	// Bonus 3: Liquidity sweep before entry (+2)
	if ctx.H4 != nil && hasLiquiditySweep(ctx.H4, signal) {
		bonus += 2
		details.LiquiditySweep = true
	}

	// Bonus 4: CHoCH signal in favor (+1)
	if ctx.HasCHoCHSignal(signal.Direction) {
		bonus++
		details.CHoCHSignal = true
	}

	return bonus
}

// checkTrendAlignment verifies EMA trend matches signal direction
func checkTrendAlignment(tfCtx *mtf.TimeframeContext, direction int) bool {
	if tfCtx == nil {
		return false
	}

	// For short signals (H&S), we want price to be above EMAs (selling into strength)
	// For long signals (IHS), we want price to be below EMAs (buying into weakness)
	if direction == -1 { // Short
		// Ideally selling after a rally - trend could be bullish
		return tfCtx.Trend == 1 || tfCtx.LastClose > tfCtx.EMA50
	}
	// Long
	return tfCtx.Trend == -1 || tfCtx.LastClose < tfCtx.EMA50
}

// checkAtOrderBlock verifies price is at an order block
func checkAtOrderBlock(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	if tfCtx == nil || tfCtx.AtOrderBlock == nil {
		return false
	}

	ob := tfCtx.AtOrderBlock
	
	// For short signals, we want to be at a supply zone
	if signal.Direction == -1 && ob.Direction == -1 {
		return true
	}
	
	// For long signals, we want to be at a demand zone
	if signal.Direction == 1 && ob.Direction == 1 {
		return true
	}

	return false
}

// checkAtFVG verifies price is at a Fair Value Gap
func checkAtFVG(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	if tfCtx == nil || tfCtx.AtFVG == nil {
		return false
	}

	fvg := tfCtx.AtFVG
	
	// For short signals, bearish FVG (gap down) supports continuation
	if signal.Direction == -1 && fvg.Direction == -1 {
		return true
	}
	
	// For long signals, bullish FVG (gap up) supports continuation
	if signal.Direction == 1 && fvg.Direction == 1 {
		return true
	}

	// Also count if we're in any FVG (institutional interest)
	return true
}

// checkPinBarRejection looks for recent pin bars supporting the trade
func checkPinBarRejection(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	if tfCtx == nil || len(tfCtx.RecentPinBars) == 0 {
		return false
	}

	// Look for pin bars in the last 5 bars
	for _, pb := range tfCtx.RecentPinBars {
		// For short signals, look for bearish pin bars (shooting stars)
		if signal.Direction == -1 && pb.Direction == -1 {
			return true
		}
		// For long signals, look for bullish pin bars (hammers)
		if signal.Direction == 1 && pb.Direction == 1 {
			return true
		}
	}

	return false
}

// checkStructureRejection verifies rejection from key structure levels
func checkStructureRejection(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	if tfCtx == nil || tfCtx.MarketStructure == nil {
		return false
	}

	ms := tfCtx.MarketStructure
	currentPrice := tfCtx.LastClose
	tolerance := 0.003 // 0.3%

	// For short signals, check rejection from swing highs
	if signal.Direction == -1 {
		for _, sh := range ms.SwingHighs {
			diff := (currentPrice - sh.Price) / sh.Price
			if diff >= -tolerance && diff <= tolerance*2 {
				return true // Near a swing high
			}
		}
	}

	// For long signals, check rejection from swing lows
	if signal.Direction == 1 {
		for _, sl := range ms.SwingLows {
			diff := (sl.Price - currentPrice) / sl.Price
			if diff >= -tolerance && diff <= tolerance*2 {
				return true // Near a swing low
			}
		}
	}

	return false
}

// hasEngulfingConfirmation checks for recent engulfing patterns
func hasEngulfingConfirmation(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	if len(tfCtx.RecentEngulfing) == 0 {
		return false
	}

	for _, eng := range tfCtx.RecentEngulfing {
		if eng.Direction == signal.Direction {
			return true
		}
	}

	return false
}

// hasLiquiditySweep checks if liquidity was swept before the signal
func hasLiquiditySweep(tfCtx *mtf.TimeframeContext, signal *Signal) bool {
	// Look for swept liquidity zones
	for _, lz := range tfCtx.LiquidityZones {
		// Swept liquidity should be opposite to signal direction
		// Short signal benefits from buyside liquidity sweep (stops above highs taken)
		if signal.Direction == -1 && lz.Direction == 1 && lz.Swept {
			return true
		}
		// Long signal benefits from sellside liquidity sweep (stops below lows taken)
		if signal.Direction == 1 && lz.Direction == -1 && lz.Swept {
			return true
		}
	}

	return false
}

// setDetailFlag sets the appropriate detail flag
func setDetailFlag(details *ScoreDetails, tfName, factor string, value bool) {
	switch tfName {
	case "weekly":
		switch factor {
		case "trend":
			details.WeeklyTrendAligned = value
		case "ob":
			details.WeeklyAtOB = value
		case "fvg":
			details.WeeklyAtFVG = value
		case "pinbar":
			details.WeeklyPinBar = value
		case "structure":
			details.WeeklyStructureReject = value
		}
	case "daily":
		switch factor {
		case "trend":
			details.DailyTrendAligned = value
		case "ob":
			details.DailyAtOB = value
		case "fvg":
			details.DailyAtFVG = value
		case "pinbar":
			details.DailyPinBar = value
		case "structure":
			details.DailyStructureReject = value
		}
	case "h4":
		switch factor {
		case "trend":
			details.H4TrendAligned = value
		case "ob":
			details.H4AtOB = value
		case "fvg":
			details.H4AtFVG = value
		case "pinbar":
			details.H4PinBar = value
		case "structure":
			details.H4StructureReject = value
		}
	}
}

// assignGrade determines the trade grade based on total score
func assignGrade(total int) string {
	// Grading thresholds:
	// A: 15+ points - HIGH CONFIDENCE (65%+ of max)
	// B: 10-14 points - STANDARD TRADE (43-65%)
	// C: 6-9 points - LOW CONFIDENCE (26-43%)
	// F: <6 points - SKIP (<26%)
	
	switch {
	case total >= 15:
		return "A"
	case total >= 10:
		return "B"
	case total >= 6:
		return "C"
	default:
		return "F"
	}
}

// GetPositionSizeMultiplier returns position size multiplier based on grade
func GetPositionSizeMultiplier(grade string) float64 {
	switch grade {
	case "A":
		return 1.0  // Full size
	case "B":
		return 0.75 // 75% size
	case "C":
		return 0.5  // 50% size
	default:
		return 0.0  // Don't trade
	}
}

// ShouldTakeTrade determines if the trade should be taken
func ShouldTakeTrade(score *ConfluenceScore) bool {
	// Only take A and B grade trades
	return score.Grade == "A" || score.Grade == "B"
}
