package llm

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type (
	OpenAIClient struct {
		model     string
		apiClient *openai.Client
	}
)

const (
	organizationUsageCompletionsPath = "/v1/organizations/usage/completions"
)

// NewOpenAIClient returns a new OpenAI client to talk to the OpenAI API.
func NewOpenAIClient(apiKey string, model string) *OpenAIClient {
	apiClient := openai.NewClient(option.WithAPIKey(apiKey))

	return &OpenAIClient{
		model:     model,
		apiClient: &apiClient,
	}
}

func (o *OpenAIClient) GetOrganizationCompletionsUsage(ctx context.Context) (int, error) {
	inputTokens := 0

	if o.apiClient == nil {
		return inputTokens, fmt.Errorf("GetOrganizationCompletionsUsage failed: openai client is nil")
	}

	startTime := string(time.Now().Truncate(24 * time.Hour).Unix())
	endTime := string(time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Unix())

	q := url.Values{}
	q.Add("start_time", startTime)
	q.Add("end_time", endTime)
	qs := q.Encode()

	// TODO: This data is assuming an anthropic shape but OpenAI is slightly different
	// https://platform.openai.com/docs/api-reference/usage/completions
	// UsageData should be our shape of data that is built using specific responses from Anthropic or OpenAI
	d := struct {
		Data []usageData `json:"data"`
	}{}

	path := organizationUsageCompletionsPath + "?" + qs

	err := o.apiClient.Get(ctx, path, nil, &d)
	if err != nil {
		return inputTokens, err
	}

	if len(d.Data) == 0 {
		return inputTokens, nil
	}

	for _, d := range d.Data {
		for _, result := range d.Results {
			if result.Model == o.model {
				inputTokens += result.UncachedInputTokens
			}
		}
	}

	return inputTokens, nil
}

// CountTokens counts the number of tokens in a slice of messages.
func (o *OpenAIClient) CountTokens(ctx context.Context, messages []Message) int {
	return 0
}
