package llm

import (
	"context"
	"fmt"
	"net/url"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicClient struct {
	messagesClient *anthropic.Client
	adminClient    *anthropic.Client
	version        string
}

const (
	organizationMessagsUsageReportPath = "/v1/organizations/usage_report/messages"
)

func NewAnthropicClient(apiKey string, adminKey string) *AnthropicClient {
	messagesClient := anthropic.NewClient(option.WithAPIKey(apiKey))
	adminClient := anthropic.NewClient(option.WithAPIKey(adminKey))

	return &AnthropicClient{
		messagesClient: &messagesClient,
		adminClient:    &adminClient,
		version:        "2023-06-01",
	}
}

// GetOrganizationMessageUsageReport fetches the current usage for all messages
// FIXME: Usage limit ins't some flat number based on the organization. Usage is based on a set of messages, specifically inputs.
// For each message provided, we'll need to compare against the current organization input usage for the model and asses a risk
// ::ORGANIZATION USAGE::
// 1. Fetch the usage report https://docs.anthropic.com/en/api/admin-api/claude-code/get-claude-code-usage-report and find the model we are using
// 2. Determine how many input MToks we have currently used.
// 3. Store current cost on a struct. This will be used for assesments i.e. a +5% increase could be considered low but a +75% increase could be considered high
func (a *AnthropicClient) GetOrganizationMessageUsageReport(ctx context.Context) (int, error) {
	if a.adminClient == nil {
		return 0, fmt.Errorf("GetOrganizationUsage failed: anthropic admin client is nil")
	}

	// TODO: Unmarshall this properly
	u := struct {
		Data []struct{} `json:"data"`
	}{}

	q := url.Values{}
	q.Add("starting_at", "2025-08-01T00:00:00Z")
	q.Add("group_by[]", "model")
	qs := q.Encode()

	path := organizationMessagsUsageReportPath + "?" + qs

	err := a.adminClient.Get(ctx, path, nil, &u)
	if err != nil {
		return 0, err
	}

	fmt.Println(u)

	return 0, nil
}

func (a *AnthropicClient) CountTokens(ctx context.Context, prompt string) (int, error) {
	if a.messagesClient == nil {
		return 0, fmt.Errorf("CountTokens failed: anthropic messages client is nil")
	}

	return 0, nil
}
