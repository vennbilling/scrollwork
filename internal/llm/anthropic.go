package llm

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type (
	AnthropicClient struct {
		model          string
		messagesClient *anthropic.Client
		adminClient    *anthropic.Client
		version        string
	}

	usageData struct {
		StartingAt string         `json:"starting_at"`
		EndingAt   string         `json:"ending_at"`
		Results    []messageUsage `json:"results"`
	}

	messageUsage struct {
		Model               string `json:"model"`
		UncachedInputTokens int    `json:"uncached_input_tokens"`
	}
)

const (
	anthropicVersion                   = "2023-06-01"
	organizationInfoPath               = "/v1/organizations/me"
	organizationMessagsUsageReportPath = "/v1/organizations/usage_report/messages"
)

func NewAnthropicClient(apiKey string, adminKey string, model string) *AnthropicClient {
	messagesClient := anthropic.NewClient(option.WithAPIKey(apiKey))
	adminClient := anthropic.NewClient(option.WithAPIKey(adminKey))

	return &AnthropicClient{
		messagesClient: &messagesClient,
		adminClient:    &adminClient,
		version:        anthropicVersion,
		model:          model,
	}
}

// HealthCheck fetches the current organization. It is used to verify the API Key and AnthropicClient.
func (a *AnthropicClient) HealthCheck(ctx context.Context) error {
	if a.adminClient == nil {
		return fmt.Errorf("HealthCheck failed: anthropic admin client is nil")
	}

	d := struct {
		ID string `json:"id"`
	}{}

	err := a.adminClient.Get(ctx, organizationInfoPath, nil, &d)
	if err != nil {
		return err
	}

	return nil
}

// GetOrganizationMessageUsageReport fetches the current number of uncached input tokens for all messages for a given model.
func (a *AnthropicClient) GetOrganizationMessageUsageReport(ctx context.Context) (int, error) {
	inputTokens := 0

	if a.adminClient == nil {
		return inputTokens, fmt.Errorf("GetOrganizationMessageUsageReport failed: anthropic admin client is nil")
	}

	startingAt := time.Now().Truncate(24 * time.Hour).Format(time.RFC3339)
	endingAt := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339)

	// TODO: This data is assuming an anthropic shape but OpenAI is slightly different
	// https://platform.openai.com/docs/api-reference/usage/completions
	// UsageData should be our shape of data that is built using specific responses from Anthropic or OpenAI
	d := struct {
		Data []usageData `json:"data"`
	}{}

	q := url.Values{}
	q.Add("starting_at", startingAt)
	q.Add("ending_at", endingAt)
	qs := q.Encode()

	path := organizationMessagsUsageReportPath + "?" + qs

	err := a.adminClient.Get(ctx, path, nil, &d)
	if err != nil {
		return inputTokens, err
	}

	if len(d.Data) == 0 {
		return inputTokens, nil
	}

	for _, d := range d.Data {
		for _, result := range d.Results {
			if result.Model == a.model {
				inputTokens += result.UncachedInputTokens
			}
		}
	}

	return inputTokens, nil
}

func (a *AnthropicClient) CountTokens(ctx context.Context, messages []Message) (int, error) {
	if a.messagesClient == nil {
		return 0, fmt.Errorf("CountTokens failed: anthropic messages client is nil")
	}

	return 0, nil
}
