package llm

import (
	"context"
	"fmt"
	"strings"
)

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

	ClientConfig struct {
		Models  []string
		APIKeys *APIKeys
	}

	APIKeys struct {
		Anthropic AnthropicAPIKeys
		OpenAI    OpenAIAPIKeys
	}

	AnthropicAPIKeys struct {
		AdminAPIKey    string
		MessagesAPIKey string
	}

	OpenAIAPIKeys struct {
		APIKey string
	}

	APIClient struct {
		config    ClientConfig
		anthropic AnthropicClient
		openai    OpenAIClient
	}

	InputTokenUsage struct {
		UncachedTotal int
		CachedTotal   int
	}

	MessageRole string
)

func NewAPIClient(config ClientConfig) *APIClient {
	c := &APIClient{}

	if config.APIKeys.Anthropic.MessagesAPIKey != "" && config.APIKeys.Anthropic.AdminAPIKey != "" {
		// TODO: Initialize Anthropic Client
	}

	if config.APIKeys.OpenAI.APIKey != "" {
		// TODO: Initialize OpenAI Client
	}
	return c
}

// GetCurrentOrganizationUsage fetchs the current input token usage for all the configured models
func (c *APIClient) GetCurrentOrganizationUsage(ctx context.Context) (map[string]int, error) {
	u := map[string]int{}
	for _, model := range c.config.Models {
		switch {
		case IsAnthropicModel(model):
			usage, err := c.anthropic.GetOrganizationMessageUsageReport(ctx)
			if err != nil {
				return u, err
			}
			u[model] = usage
		case IsOpenAIModel(model):
			usage, err := c.openai.GetOrganizationCompletionsUsage(ctx)
			if err != nil {
				return u, err
			}
			u[model] = usage
		default:
			return nil, fmt.Errorf("GetOrganizationUsage failed: unsupported model &s", model)
		}
	}

	return u, nil
}

func IsAnthropicModel(model string) bool {
	return strings.Contains(model, "claude-")
}

func IsOpenAIModel(model string) bool {
	return strings.Contains(model, "gpt-") || strings.Contains(model, "text-")
}
