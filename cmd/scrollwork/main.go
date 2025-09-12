package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "embed"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

//go:embed banner.txt
var banner []byte

var (
	model              string
	apiKey             string
	refreshRateMinutes int
)

type organizationUsage struct {
	UncachedInputTokens  int
	CacheReadInputTokens int
	OutputTokens         int
}

type worker struct {
	Interval time.Duration
	Period   time.Duration
	wg       *sync.WaitGroup

	usageChan chan organizationUsage
}

type ScrollworkAgent struct {
	listener           *net.UnixListener
	model              string
	refreshRateMinutes int

	AnthropicClient anthropic.Client

	OrganizationUsage organizationUsage

	usageChan chan organizationUsage
	done      chan os.Signal
	cancel    context.CancelFunc

	wg *sync.WaitGroup
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

	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n")

	// Configure lifecycle signals
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Usage Channel
	usageChan := make(chan organizationUsage, 1)

	var wg sync.WaitGroup

	sw := ScrollworkAgent{
		model:              model,
		refreshRateMinutes: refreshRateMinutes,

		usageChan: usageChan,
		done:      sigChan,
		cancel:    cancel,
		wg:        &wg,
	}

	// Configure UNIX socket and lisenter
	addr := net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		log.Fatal("failed to listen on unix socket:", err)
	}

	sw.listener = listener

	// Configure AI Model and Client
	if !sw.IsUsingAnthropic() && !sw.IsUsingOpenAI() {
		listener.Close()
		log.Fatal("Only OpenAI and Anthropic models are supported.")
	}

	if sw.IsUsingAnthropic() {
		sw.AnthropicClient = anthropic.NewClient(option.WithAPIKey(apiKey))
	}

	w := NewWorker(time.Duration(sw.refreshRateMinutes)*time.Minute, usageChan, &wg)

	wg.Add(1)
	go w.Start(ctx)

	sw.Start(ctx)

	// TODO: Shutdown isn't working right. Sometimes we see the logging for the worker shutting down. Sometimes we dont...
	w.wg.Done()
}

func (sw *ScrollworkAgent) Start(ctx context.Context) {
	// TODO: Hard crashes aren't calling shutdown. Leverage recover()
	go func() {
		sig := <-sw.done

		log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)
		sw.Shutdown()
	}()

	// TODO: Based on the AI Client configured:
	// 2. Spin up a worker that fetches the current quota after X minutes
	// 3. Update the current quota
	// 4. Use current quota when doing count tokens logic

	if err := sw.fetchOrganizationUsage(ctx); err != nil {
		// TODO: Either retry or fail fast if we can't get this info...
		log.Printf("Scrollwork failed FetchOrganizationInfo. Quotas not be enforced")
	}

	socketName := sw.listener.Addr().String()

	log.Printf("Scrollwork has started and can receive connections")
	log.Printf("Current AI Model: %s.", sw.model)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork socket at %s closed", socketName)
			return
		case orgUsage := <-sw.usageChan:
			log.Printf("Organization usage updated: %+v", sw.OrganizationUsage)
			// TODO: Should sync this with a mutex
			sw.OrganizationUsage = orgUsage
			break
		default:
			conn, err := sw.listener.AcceptUnix()
			if err != nil {
				log.Printf("Connections can no longer be accepted: %v", err)
				break
			}

			log.Printf("Connection accepted")
			go handleConnection(conn, sw.OrganizationUsage)
		}
	}
}

func (sw *ScrollworkAgent) Shutdown() {
	sw.cancel()
	sw.listener.Close()
	sw.wg.Wait()
}

func (sw *ScrollworkAgent) IsUsingAnthropic() bool {
	return strings.Contains(sw.model, "claude-")
}

func (sw *ScrollworkAgent) IsUsingOpenAI() bool {
	return strings.Contains(sw.model, "gpt-") || strings.Contains(sw.model, "text-")
}

func (sw *ScrollworkAgent) fetchOrganizationUsage(ctx context.Context) error {
	if sw.IsUsingOpenAI() {
		return fmt.Errorf("FetchOrganizationInfo failed: OpenAI is not supported at this time")
	}

	log.Printf("Fetching current organization usage...")

	// TODO: Fetch from /v1/organizations/usage_report/messages
	// TODO: Refresh on a cadenence that is configured on app start
	resp := organizationUsage{
		UncachedInputTokens:  12345,
		CacheReadInputTokens: 67890,
		OutputTokens:         13579,
	}

	sw.OrganizationUsage = resp

	return nil
}

func handleConnection(conn net.Conn, usage organizationUsage) {
	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 3. Count the tokens using model's APIs
	// 4. Return tokens used and percentage of total quota used
	// 5. Return a risk level of the request. Low = Low cost, Medium = Medium cost, High = High / Unknown cost. Costs are configurable
	conn.Write([]byte(fmt.Sprintf("Hello. You currently have %d UncachedInputTokens left\n", usage.UncachedInputTokens)))
	conn.Close()
	log.Printf("Connection closed")
}

func NewWorker(interval time.Duration, usageChan chan organizationUsage, wg *sync.WaitGroup) *worker {
	return &worker{
		Interval: interval,
		Period:   interval,
		wg:       wg,

		usageChan: usageChan,
	}
}

func (w *worker) Start(ctx context.Context) {
	log.Printf("Refresh Usage Worker started with interval %v and period %v", w.Interval, w.Period)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Refresh Usage Worker is shutting down...")
			return
		case <-time.After(w.Period):
			break
		}

		started := time.Now()

		w.usageChan <- organizationUsage{
			UncachedInputTokens:  12345,
			CacheReadInputTokens: 67890,
			OutputTokens:         13580,
		}

		finished := time.Now()

		duration := finished.Sub(started)
		w.Period = w.Interval - duration
	}
}
