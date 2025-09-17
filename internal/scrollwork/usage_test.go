package scrollwork_test

import (
	"scrollwork/internal/scrollwork"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	u := scrollwork.Usage{}

	u.Update(10)

	require.Equal(t, 10, u.Tokens())
}
