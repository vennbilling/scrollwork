package llm

import "strings"

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type (
	Message struct {
		Role    MessageRole
		Name    string
		Content string
	}

	MessageRole string
)

func IsAnthropicModel(model string) bool {
	return strings.Contains(model, "claude-")
}

func IsOpenAIModel(model string) bool {
	return strings.Contains(model, "gpt-") || strings.Contains(model, "text-")
}
