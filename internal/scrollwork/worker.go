package scrollwork

import (
	"context"
	"errors"
	"fmt"
	"log"
	"scrollwork/internal/llm"
	"time"
)

type (
	UsageWorkerConfig struct {
		Models        []string
		UsageReceived chan map[string]int
		WorkerReady   chan bool
		TickRate      int

		Client *llm.APIClient
	}

	UsageWorker struct {
		config *UsageWorkerConfig

		ticker          *time.Ticker
		AnthropicClient *llm.AnthropicClient
	}
)

// newUsageWorker creates a new [usageWorker].
//
// The usageWorker is responsible for fetching and storing the current token usage for a given organization.
func newUsageWorker(config *UsageWorkerConfig) *UsageWorker {
	return &UsageWorker{
		config: config,
	}
}

func (w *UsageWorker) Start(ctx context.Context) error {
	if err := w.healthCheck(ctx); err != nil {
		return err
	}

	// Fetch the latest usage snapshot for the organization
	usage, err := w.fetchOrganizationUsage(ctx)
	if err != nil {
		return err
	}

	w.config.UsageReceived <- usage
	w.config.WorkerReady <- true

	return nil
}

func (w *UsageWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.config.TickRate) * time.Minute)
	w.ticker = ticker

	for {
		select {
		case <-w.ticker.C:
			log.Printf("Scrollwork Usage Worker is fetching latest usage...")
			usage, err := w.fetchOrganizationUsage(ctx)
			if err != nil {
				log.Printf("Scrollwork Usager Worker failed to fetch latest usage: %v", err)
				break
			}

			w.config.UsageReceived <- usage
			log.Printf("Scrollwork Usage Worker has received the latest usage")
		case <-ctx.Done():
			return
		}
	}
}

func (w *UsageWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}

	w.config.WorkerReady <- false
}

func (w *UsageWorker) fetchOrganizationUsage(ctx context.Context) (map[string]int, error) {
	usage := make(map[string]int)

	// Check if we have any Anthropic models
	hasAnthropicModel := false
	for _, model := range w.config.Models {
		if llm.IsAnthropicModel(model) {
			hasAnthropicModel = true
			break
		}
	}

	// Fetch Anthropic usage once for all Anthropic models
	if hasAnthropicModel {
		if w.AnthropicClient == nil {
			return usage, fmt.Errorf("fetchOrganizationUsage failed: AnthropicClient not configured")
		}

		anthropicUsage, err := w.AnthropicClient.GetOrganizationMessageUsageReport(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return make(map[string]int), nil
			}

			return make(map[string]int), fmt.Errorf("Failed to fetchOrganizationUsage: %v", err)
		}

		// Copy Anthropic usage for configured models
		for _, model := range w.config.Models {
			if llm.IsAnthropicModel(model) {
				if tokens, ok := anthropicUsage[model]; ok {
					usage[model] = tokens
				}
			}
		}
	}

	// TODO: Add OpenAI support
	for _, model := range w.config.Models {
		if llm.IsOpenAIModel(model) {
			return usage, fmt.Errorf("fetchOrganizationUsage failed: OpenAI model %s is not supported", model)
		}
	}

	return usage, nil
}

func (w *UsageWorker) healthCheck(ctx context.Context) error {
	// Check if we have any Anthropic models
	hasAnthropicModel := false
	for _, model := range w.config.Models {
		if llm.IsAnthropicModel(model) {
			hasAnthropicModel = true
			break
		}
	}

	// Health check Anthropic client if needed
	if hasAnthropicModel {
		if w.AnthropicClient == nil {
			return fmt.Errorf("healthCheck failed: AnthropicClient was not configured")
		}

		if err := w.AnthropicClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("healthCheck failed: %v", err)
		}
	}

	// Check for unsupported models
	for _, model := range w.config.Models {
		if llm.IsOpenAIModel(model) {
			return fmt.Errorf("healthCheck failed: LLM model %s is not supported", model)
		}
	}

	return nil
}
