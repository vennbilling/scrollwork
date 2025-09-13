package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicClient struct {
	client *anthropic.Client
}

func NewAnthropicClient(apiKey string) *AnthropicClient {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &AnthropicClient{
		client: &client,
	}
}

func (a *AnthropicClient) GetOrganizationUsage(ctx context.Context) (int, error) {
	if a.client == nil {
		return 0, fmt.Errorf("GetOrganizationUsage failed: anthropic client is nil")
	}

	return 0, nil
}

func (a *AnthropicClient) CountTokens(ctx context.Context, prompt string) (int, error) {
	if a.client == nil {
		return 0, fmt.Errorf("CountTokens failed: anthropic client is nil")
	}

	return 0, nil
}
