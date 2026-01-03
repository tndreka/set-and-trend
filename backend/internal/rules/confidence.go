package rules

// ComputeConfidence calculates confidence as ratio of met conditions
// This function is FROZEN - changing it corrupts all historical data
//
// LOCKED SEMANTICS:
// - If rule FAILS: confidence = 0 (regardless of partial matches)
// - If rule PASSES: confidence = met/total (always 1.0 for all-or-nothing rules)
func ComputeConfidence(met int, total int, passed bool) float64 {
	if total == 0 {
		return 0.0
	}
	
	// ✅ FIXED: FAIL always means confidence = 0
	if !passed {
		return 0.0
	}
	
	// PASS: return ratio (for all-or-nothing rules, this is always 1.0)
	return float64(met) / float64(total)
}

// ConfidenceThreshold defines when a rule passes
const ConfidenceThreshold = 1.0 // All conditions must be met for PASS

// ShouldPass determines if a rule should pass based on conditions
func ShouldPass(metCount int, totalCount int) bool {
	if totalCount == 0 {
		return false
	}
	return metCount == totalCount // All conditions must be met
}
