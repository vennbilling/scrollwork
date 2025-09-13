package scrollwork

import (
	"context"
	"log"
	"time"
)

type UsageWorker struct {
	usageChan chan int
	ticker    *time.Ticker
}

// newUsageWorker creates a new [usageWorker].
//
// The usageWorker is responsible for fetching and storing the current token usage for a given organization.
func newUsageWorker(usageReceivedChan chan int) *UsageWorker {
	return &UsageWorker{
		usageChan: usageReceivedChan,
	}
}

func (w *UsageWorker) Start(ctx context.Context, tickRate int) {
	log.Printf("Scrollwork Usage Worker has started.")

	// Immediately fetch usage on start
	w.fetchUsage()

	for {
		select {
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

func (w *UsageWorker) fetchUsage() {
	log.Printf("Fetching latest usage")

	w.usageChan <- 1
}
