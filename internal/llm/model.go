package llm

import "strings"

func IsAnthropicModel(model string) bool {
	return strings.Contains(model, "claude-")
}

func IsOpenAIModel(model string) bool {
	return strings.Contains(model, "gpt-") || strings.Contains(model, "text-")
}
