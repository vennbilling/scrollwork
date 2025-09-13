package scrollwork

import (
	"context"
	"log"
	"time"
)

type usageWorker struct {
	notifyChan chan int
	ticker     *time.Ticker
}

// newUsageWorker creates a new [usageWorker].
//
// The usageWorker is responsible for fetching and storing the current token usage for a given organization.
func newUsageWorker(notiferChan chan int) *usageWorker {
	return &usageWorker{
		notifyChan: notiferChan,
	}
}

func (w *usageWorker) start(ctx context.Context, tickRate int) {
	log.Printf("Scrollwork Usage Worker has started.")

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork Usage Worker will be shutting down...")
			return
		}
	}
}

func (w *usageWorker) stop() {
	if w.ticker != nil {
		log.Printf("Stopping usage worker ticker...")
		w.ticker.Stop()
	}

	log.Printf("Scrollwork Usage Worker has shutdown.")
}
