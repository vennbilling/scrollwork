package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"scrollwork/internal/scrollwork"
	"sync"
	"syscall"
	"time"

	_ "embed"
)

//go:embed banner.txt
var banner []byte

var (
	model              string
	apiKey             string
	refreshRateMinutes int
)

type usageWorker struct {
	Interval time.Duration
	Period   time.Duration
	wg       *sync.WaitGroup
}

func init() {
	flag.StringVar(&model, "model", "", "AI Model")
	flag.StringVar(&apiKey, "apiKey", "", "API Key")
	flag.IntVar(&refreshRateMinutes, "refreshRate", 1, "Refresh rate in minutes for fetching organization usage")
}

func main() {
	flag.Parse()

	if model == "" {
		fmt.Println("AI Model is required. Use --model to set it.")
		os.Exit(1)
	}

	if apiKey == "" {
		fmt.Println("API Key is required. Use --apiKey to set it.")
		os.Exit(1)
	}

	config := &scrollwork.AgentConfig{
		Model:                       model,
		APIKey:                      apiKey,
		RefreshUsageIntervalMinutes: refreshRateMinutes,
	}
	agent := scrollwork.NewAgent(config)

	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n")

	if err := agent.Start(); err != nil {
		log.Fatalf("Scrollwork Agent could not start: %v", err)
	}

	// Configure lifecycle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received, Scrollwork Agent is shutting down...")

	if err := agent.Stop(); err != nil {
		log.Fatalf("Scrollwork Agent failed to shut down: %v", err)
	}

	log.Printf("Scrollwork Agent shut down complete.")
}
