package scrollwork

import (
	"context"
	"log"
	"time"
)

type UsageWorker struct {
	notifyChan chan int
	ticker     *time.Ticker
}

// newUsageWorker creates a new [usageWorker].
//
// The usageWorker is responsible for fetching and storing the current token usage for a given organization.
func newUsageWorker(notiferChan chan int) *UsageWorker {
	return &UsageWorker{
		notifyChan: notiferChan,
	}
}

func (w *UsageWorker) Start(ctx context.Context, tickRate int) {
	log.Printf("Scrollwork Usage Worker has started.")

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
