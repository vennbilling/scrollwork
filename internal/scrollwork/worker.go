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
		Model         string
		UsageReceived chan int
		WorkerReady   chan bool
		TickRate      int
	}

	UsageWorker struct {
		config *UsageWorkerConfig

		ticker          *time.Ticker
		AnthropicClient *llm.AnthropicClient
	}

	UsageData struct {
		Tokens int
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

	w.config.WorkerReady <- true

	return nil
}

func (w *UsageWorker) Run(ctx context.Context) error {
	// Fetch the latest usage snapshot for the organization
	usage, err := w.fetchOrganizationUsage(ctx)
	if err != nil {
		return fmt.Errorf("Scrollwork Usage Worker failed to run: %v", err)
	}

	w.config.UsageReceived <- usage.Tokens

	ticker := time.NewTicker(time.Duration(w.config.TickRate) * time.Minute)
	w.ticker = ticker

	for {
		select {
		case <-w.ticker.C:
			log.Printf("Scrollwork Usage Worker is fetching latest usage...")
			usage, err := w.fetchOrganizationUsage(ctx)
			if err != nil {
				return fmt.Errorf("fetchOrganizationUsage failed: %v", err)
			}

			w.config.UsageReceived <- usage.Tokens
		case <-ctx.Done():
			log.Printf("Scrollwork Usage Worker will be shutting down...")
			return ctx.Err()
		}
	}
}

func (w *UsageWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
}

func (w *UsageWorker) fetchOrganizationUsage(ctx context.Context) (UsageData, error) {
	if llm.IsOpenAIModel(w.config.Model) {
		return UsageData{}, fmt.Errorf("fetchOrganizationUsage failed: OpenAI is not supported")
	}

	tokens, err := w.AnthropicClient.GetOrganizationMessageUsageReport(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return UsageData{}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return UsageData{}, nil
		}

		return UsageData{}, fmt.Errorf("Failed to fetchOrganizationUsage: %v", err)
	}

	return UsageData{Tokens: tokens}, nil
}

func (w *UsageWorker) healthCheck(ctx context.Context) error {
	switch {
	case llm.IsAnthropicModel(w.config.Model):
		if w.AnthropicClient == nil {
			return fmt.Errorf("healthCheck failed: AnthropicClient was not configured")
		}

		if err := w.AnthropicClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("healthCheck failed: %v", err)
		}
	case llm.IsOpenAIModel(w.config.Model):
		return fmt.Errorf("healthCheck failed: LLM model %s is not supported", w.config.Model)
	}

	return nil
}
