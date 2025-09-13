package llm_test

import (
	"scrollwork/internal/llm"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAnthropicModel(t *testing.T) {
	t.Parallel()
	tc := []string{
		"claude-opus-4-1-20250805",
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-3-7-sonnet-20250219",
		"claude-3-7-sonnet-latest",
		"claude-3-5-haiku-20241022",
		"claude-3-haiku-20240307",
	}

	for _, td := range tc {
		require.True(t, llm.IsAnthropicModel(td))
	}
}

func TestIsAnthropicModel_AWSBedrock(t *testing.T) {
	t.Parallel()
	tc := []string{
		"anthropic.claude-opus-4-1-20250805-v1:0",
		"anthropic.claude-opus-4-20250514-v1:0",
		"anthropic.claude-sonnet-4-20250514-v1:0",
		"anthropic.claude-3-7-sonnet-20250219-v1:0",
		"anthropic.claude-3-7-sonnet-latest-v1:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0",
		"anthropic.claude-3-haiku-20240307-v1:0",
	}

	for _, td := range tc {
		require.True(t, llm.IsAnthropicModel(td))
	}
}

func TestIsAnthropicModel_GCPVertex(t *testing.T) {
	t.Parallel()
	tc := []string{
		"claude-opus-4-1@20250805",
		"claude-opus-4@20250514",
		"claude-sonnet-4@20250514",
		"claude-3-7-sonnet@20250219",
		"claude-3-7-sonnet-latest",
		"claude-3-5-haiku@20241022",
		"claude-3-haiku@20240307",
	}

	for _, td := range tc {
		require.True(t, llm.IsAnthropicModel(td))
	}
}

func TestIsAnthropicModel_Error(t *testing.T) {
	t.Parallel()

	tc := []string{
		"gpt-test",
		"garbage-garbage-garbage",
	}

	for _, td := range tc {
		require.False(t, llm.IsAnthropicModel(td))
	}
}
