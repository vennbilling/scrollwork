package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"scrollwork/internal/scrollwork"
	"sync"
	"syscall"
	"time"

	_ "embed"
)

var (
	model              string
	apiKey             string
	adminKey           string
	refreshRateMinutes int
)

type usageWorker struct {
	Interval time.Duration
	Period   time.Duration
	wg       *sync.WaitGroup
}

func init() {
	flag.StringVar(&model, "model", "", "AI Model")

	// TODO: These are assuming Anthropic keys. We should handle OpenAPI differently
	flag.StringVar(&apiKey, "apiKey", "", "API Key")
	flag.StringVar(&adminKey, "adminKey", "", "Admin Key")
	flag.IntVar(&refreshRateMinutes, "refreshRate", 1, "Refresh rate in minutes for fetching organization usage")
}

func main() {
	flag.Parse()

	if model == "" {
		log.Fatal("AI Model is required. Use --model to set it.")
	}

	if apiKey == "" {
		log.Fatal("API Key is required. Use --apiKey to set it.")
	}

	if adminKey == "" {
		log.Fatal("Admin Key is required. Use --adminkey to set it.")
	}

	if refreshRateMinutes <= 0 {
		log.Fatal("Refresh rate must be a positive.")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := &scrollwork.AgentConfig{
		Model:                       model,
		APIKey:                      apiKey,
		AdminKey:                    adminKey,
		RefreshUsageIntervalMinutes: refreshRateMinutes,
	}
	agent, err := scrollwork.NewAgent(config)
	if err != nil {
		log.Fatalf("Scrollwork Agent could be initialized: %v", err)
	}

	if err := agent.Start(ctx); err != nil {
		log.Fatalf("Scrollwork Agent could not start: %v", err)
	}

	if err := agent.Run(ctx); err != nil {
		log.Fatalf("Scrollwork Agent failed to run: %v", err)
	}

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received, Scrollwork is shutting down...")
	}

	if err := agent.Stop(); err != nil {
		log.Fatalf("Scrollwork Agent failed to shut down: %v", err)
	}
}
