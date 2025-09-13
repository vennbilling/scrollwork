package scrollwork

import (
	"context"
	"log"
	"scrollwork/internal/llm"
	"time"
)

type (
	UsageWorker struct {
		usageReceived chan int
		workerReady   chan bool
		ticker        *time.Ticker

		anthropicClient *llm.AnthropicClient
	}

	UsageData struct {
		Tokens int
	}
)

// newUsageWorker creates a new [usageWorker].
//
// The usageWorker is responsible for fetching and storing the current token usage for a given organization.
func newUsageWorker(usageReceivedChan chan int, workerReadyChan chan bool) *UsageWorker {
	return &UsageWorker{
		usageReceived: usageReceivedChan,
		workerReady:   workerReadyChan,
	}
}

func (w *UsageWorker) Start(ctx context.Context, tickRate int) {
	log.Printf("Scrollwork Usage Worker starting...")
	// Immediately fetch usage on start and notify
	usage := w.fetchOrganizationUsage(ctx)
	w.usageReceived <- usage.Tokens

	w.workerReady <- true

	ticker := time.NewTicker(time.Duration(tickRate) * time.Minute)
	w.ticker = ticker

	log.Printf("Scrollwork Usage Worker has started")

	for {
		select {
		case <-ticker.C:
			usage := w.fetchOrganizationUsage(ctx)
			w.usageReceived <- usage.Tokens
		case <-ctx.Done():
			log.Printf("Scrollwork Usage Worker will be shutting down...")
			return
		}
	}
}

func (w *UsageWorker) Stop() {
	if w.ticker != nil {
		log.Printf("Stopping usage worker ticker...")
		w.ticker.Stop()
	}

	log.Printf("Scrollwork Usage Worker has shutdown.")
}

func (w *UsageWorker) fetchOrganizationUsage(ctx context.Context) UsageData {
	log.Printf("Fetching latest usage")

	tokens, err := w.anthropicClient.GetOrganizationMessageUsageReport(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch Organization Usage: %v", err)
	}

	return UsageData{Tokens: tokens}
}
