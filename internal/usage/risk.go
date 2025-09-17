package usage

type (
	RiskThresholds struct {
		lowThreshold    float32
		mediumThreshold float32
		highThreshold   float32
	}

	RiskLevel string
)

const (
	RiskLevelUnknown RiskLevel = "unknown"
	RiskLevelLow     RiskLevel = "low"
	RiskLevelMedium  RiskLevel = "medium"
	RiskLevelHigh    RiskLevel = "high"
)

func NewRiskThresholds(low float32, medium float32, high float32) RiskThresholds {
	return RiskThresholds{
		lowThreshold:    low,
		mediumThreshold: medium,
		highThreshold:   high,
	}
}

func (t *RiskThresholds) Asses(tokens int) RiskLevel {
	if t.lowThreshold == 0 && t.mediumThreshold == 0 && t.highThreshold == 0 {
		return RiskLevelLow
	}

	if t.lowThreshold == t.mediumThreshold && t.mediumThreshold == t.highThreshold {
		return RiskLevelUnknown
	}

	if tokens == 0 {
		return RiskLevelLow
	}

	return RiskLevelUnknown
}
