package scrollwork

import (
	"context"
	"log"
)

type usageWorker struct {
	notifyChan chan int
}

// newUsageWorker creates a new uageWorker. The usageWorker is
// responsible for fetching and storing the current token usage for a given organization
func newUsageWorker(notiferChan chan int) *usageWorker {
	return &usageWorker{
		notifyChan: notiferChan,
	}
}

func (w *usageWorker) start(ctx context.Context) {
	log.Printf("Scrollwork Usage Worker has started.")
	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork Usage Worker will be shutting down...")
			return
		default:
		}
	}
}

func (w *usageWorker) stop() {
	log.Printf("Scrollwork Usage Worker has shutdown.")
}
