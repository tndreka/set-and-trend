package patterns

import "math"

// ConfidenceWeights holds the weights for each component
// These should be tuned based on backtesting results
type ConfidenceWeights struct {
	ShoulderSymmetry float64
	HeadProminence   float64
	TimeSymmetry     float64
	VolumeProfile    float64
	NecklineQuality  float64
}

// DefaultWeights returns the default confidence weights
// These are starting values - should be optimized via backtesting
func DefaultWeights() ConfidenceWeights {
	return ConfidenceWeights{
		ShoulderSymmetry: 0.25, // Equal weights to start
		HeadProminence:   0.20,
		TimeSymmetry:     0.15,
		VolumeProfile:    0.20,
		NecklineQuality:  0.20,
	}
}

// CalculateStructureConfidence calculates geometric confidence
// This is the raw pattern quality score before context adjustments
func CalculateStructureConfidence(s *DetectedStructure) float64 {
	if s == nil {
		return 0.0
	}

	w := DefaultWeights()
	score := 0.0

	// Each component contributes to the score
	score += s.ShoulderSymmetry * w.ShoulderSymmetry
	score += s.HeadProminence * w.HeadProminence
	score += s.TimeSymmetry * w.TimeSymmetry
	score += capValue(s.VolumeProfile+0.2, 0.0, 1.0) * w.VolumeProfile // Normalize volume from [-0.2, 0.8] to [0, 1]
	score += s.NecklineQuality * w.NecklineQuality

	return math.Min(score, 1.0)
}

// CalculateStructureConfidenceWithWeights allows custom weights
func CalculateStructureConfidenceWithWeights(s *DetectedStructure, w ConfidenceWeights) float64 {
	if s == nil {
		return 0.0
	}

	score := 0.0
	score += s.ShoulderSymmetry * w.ShoulderSymmetry
	score += s.HeadProminence * w.HeadProminence
	score += s.TimeSymmetry * w.TimeSymmetry
	score += capValue(s.VolumeProfile+0.2, 0.0, 1.0) * w.VolumeProfile
	score += s.NecklineQuality * w.NecklineQuality

	return math.Min(score, 1.0)
}

// FinalConfidence combines structure confidence with context bonus
func FinalConfidence(s *DetectedStructure, ctx MarketContext) float64 {
	if s == nil {
		return 0.0
	}

	confidence := s.OverallConfidence + CalculateContextBonus(s, ctx)
	return capValue(confidence, 0.0, 1.0)
}

// capValue clamps a value between min and max
func capValue(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
