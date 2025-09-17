package usage_test

import (
	"scrollwork/internal/usage"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	u := usage.Usage{}

	u.Update(10)

	require.Equal(t, 10, u.Tokens())
}
