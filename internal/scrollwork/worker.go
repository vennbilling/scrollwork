package scrollwork

import (
	"context"
	"log"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type (
	UsageWorker struct {
		usageReceived chan int
		workerReady   chan bool
		ticker        *time.Ticker

		anthropicClient anthropic.Client
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
	usage := w.fetchOrganizationUsage()
	w.usageReceived <- usage.Tokens

	w.workerReady <- true

	ticker := time.NewTicker(time.Duration(tickRate) * time.Minute)
	w.ticker = ticker

	log.Printf("Scrollwork Usage Worker has started")

	for {
		select {
		case <-ticker.C:
			usage := w.fetchOrganizationUsage()
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

func (w *UsageWorker) fetchOrganizationUsage() UsageData {
	log.Printf("Fetching latest usage")
	return UsageData{Tokens: 0}
}
