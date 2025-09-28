package usage_test

import (
	"scrollwork/internal/usage"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	m := usage.ModelUsage{}

	m.Update(10)

	require.Equal(t, 10, m.Tokens())
}
