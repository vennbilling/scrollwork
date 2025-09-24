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
		Client          *llm.Client
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

	// Fetch the latest usage snapshot for the organization
	usage, err := w.fetchOrganizationUsage(ctx)
	if err != nil {
		return err
	}

	w.config.UsageReceived <- usage.Tokens
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

			w.config.UsageReceived <- usage.Tokens
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
