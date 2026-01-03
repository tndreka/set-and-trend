package rules

import (
	"fmt"
	"log"
)

// EvaluateRule is a pure function that evaluates a rule
// NO DATABASE. NO SIDE EFFECTS. DETERMINISTIC.
func EvaluateRule(ruleCode RuleCode, c Candle, ind Indicators) (RuleResult, error) {
	// Get rule spec
	spec, exists := RuleRegistry[ruleCode]
	if !exists {
		return RuleResult{}, fmt.Errorf("rule not found: %s", ruleCode)
	}

	result := RuleResult{
		RuleCode:       ruleCode,
		ConditionsMet:  []ConditionCode{},
		ConditionsFail: []ConditionCode{},
	}

	// Evaluate each condition
	for _, condCode := range spec.Conditions {
		if EvaluateCondition(condCode, c, ind) {
			result.ConditionsMet = append(result.ConditionsMet, condCode)
		} else {
			result.ConditionsFail = append(result.ConditionsFail, condCode)
		}
	}

	// Determine pass/fail
	metCount := len(result.ConditionsMet)
	totalCount := len(spec.Conditions)
	passed := ShouldPass(metCount, totalCount)

	if passed {
		result.Result = "PASS"
	} else {
		result.Result = "FAIL"
	}

	// ✅ FIXED: Calculate confidence with pass/fail awareness
	result.Confidence = ComputeConfidence(metCount, totalCount, passed)

	return result, nil
}

// EvaluateAllRules evaluates all registered rules
func EvaluateAllRules(c Candle, ind Indicators) map[RuleCode]RuleResult {
	results := make(map[RuleCode]RuleResult)

	for code := range RuleRegistry {
		result, err := EvaluateRule(code, c, ind)
		if err != nil {
			// ✅ FIXED: Log instead of silently skipping
			log.Printf("⚠️  Failed to evaluate rule %s: %v", code, err)
			continue
		}
		results[code] = result
	}

	return results
}
