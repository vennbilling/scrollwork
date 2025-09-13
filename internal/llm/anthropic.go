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

	// FIXME: Usage limit ins't some flat number based on the organization. Usage is based on a set of messages, specifically inputs.
	// For each message provided, we'll need to compare against the current organization input usage for the model and asses a risk
	// ::ORGANIZATION USAGE::
	// 1. Fetch the usage report https://docs.anthropic.com/en/api/admin-api/claude-code/get-claude-code-usage-report and find the model we are using
	// 2. Determine how many input MToks we have currently used.
	// 3. Store current cost on a struct. This will be used for assesments i.e. a +5% increase could be considered low but a +75% increase could be considered high
	return 0, nil
}

func (a *AnthropicClient) CountTokens(ctx context.Context, prompt string) (int, error) {
	if a.client == nil {
		return 0, fmt.Errorf("CountTokens failed: anthropic client is nil")
	}

	return 0, nil
}
