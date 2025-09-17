package usage_test

import (
	"scrollwork/internal/usage"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsses(t *testing.T) {
	t.Parallel()

	tt := []struct {
		LowThreshold    float32
		MediumThreshold float32
		HighThreshold   float32
		Tokens          int
		Expected        usage.RiskLevel
	}{
		{
			LowThreshold:    0,
			MediumThreshold: 0,
			HighThreshold:   0,
			Tokens:          0,
			Expected:        usage.RiskLevelLow,
		},
		{
			LowThreshold:    1,
			MediumThreshold: 1,
			HighThreshold:   1,
			Tokens:          0,
			Expected:        usage.RiskLevelUnknown,
		},
		{
			LowThreshold:    1,
			MediumThreshold: 2,
			HighThreshold:   3,
			Tokens:          0,
			Expected:        usage.RiskLevelLow,
		},
		{
			LowThreshold:    1,
			MediumThreshold: 2,
			HighThreshold:   3,
			Tokens:          100,
			Expected:        usage.RiskLevelUnknown,
		},
	}

	for _, td := range tt {
		rt := usage.NewRiskThresholds(td.LowThreshold, td.MediumThreshold, td.HighThreshold)
		risk := rt.Asses(td.Tokens)
		require.Equal(t, td.Expected, risk)
	}
}
