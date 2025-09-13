package scrollwork

import (
	"context"
	"fmt"
	"log"
	"net"
	"scrollwork/internal/llm"
	"sync"

	_ "embed"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

//go:embed banner.txt
var banner []byte

type (
	AgentConfig struct {
		Model                       string
		APIKey                      string
		RefreshUsageIntervalMinutes int
	}

	Agent struct {
		config *AgentConfig

		listener *net.UnixListener
		worker   *UsageWorker

		anthropicClient anthropic.Client
		openAIClient    struct{}

		usageReceived chan int
		workerReady   chan bool
		cancel        context.CancelFunc

		wg *sync.WaitGroup
	}

	RiskLevel string
)

const (
	RiskLevelUnknown RiskLevel = "unknown"
	RiskLevelLow     RiskLevel = "low"
)

// NewAgent returns an Agent.
// A Scrollwork Agent is responsible for handling requests to check the billing risk level of an AI Prompt.
// It also spins up a worker that periodically checks and syncs an organization's current usage.
// This usage is used when calculating the risk of a AI Prompt.
func NewAgent(config *AgentConfig) *Agent {
	var wg sync.WaitGroup

	usageReceived := make(chan int, 1)
	workerReady := make(chan bool, 1)

	worker := newUsageWorker(usageReceived, workerReady)

	return &Agent{
		config: config,

		worker: worker,

		usageReceived: usageReceived,
		workerReady:   workerReady,
		wg:            &wg,
	}
}

// Start starts the Scrollwork Agent.
func (a *Agent) Start(ctx context.Context) error {
	supportedLLMModel := llm.IsAnthropicModel(a.config.Model) || llm.IsOpenAIModel(a.config.Model)
	if !supportedLLMModel {
		return fmt.Errorf("failed to Start: LLM model must either be an OpenAI model or Anthropic model")
	}

	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	if llm.IsAnthropicModel(a.config.Model) {
		anthropicClient := anthropic.NewClient(option.WithAPIKey(a.config.APIKey))
		a.anthropicClient = anthropicClient
		a.worker.anthropicClient = anthropicClient
	}

	// TODO: Remove this check once we have OpenAI integrated
	if llm.IsOpenAIModel(a.config.Model) {
		return fmt.Errorf("failed to Start: OpenAI is not supported at this time.")
	}

	a.startupMessage()

	// Configure worker to periodically fetch current usage
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.worker.Start(ctx, a.config.RefreshUsageIntervalMinutes)
	}()

	// Handle updates to current usage
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.processUsageUpdates(ctx)
	}()

	// Wait until worker is ready before we start the UNIX listener
	<-a.workerReady

	// Configure unix socket listener
	addr := net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		return err
	}
	a.listener = listener
	a.wg.Add(1)

	go func() {
		defer a.wg.Done()
		a.listen(ctx)
	}()

	return nil
}

// Stop stops the Scrollwork Agent.
func (a *Agent) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}

	// Shut down the UNIX socket
	if a.listener != nil {
		a.listener.Close()
	}

	// Shut down the usage worker
	a.worker.Stop()

	// Wait for everything else to clean up
	a.wg.Wait()

	return nil
}

func (a *Agent) listen(ctx context.Context) {
	log.Printf("Scrollwork Agent socket has started and is now ready for connections.")
	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork Agent socket closed")
			return
		default:
			conn, err := a.listener.AcceptUnix()
			if err != nil {
				log.Printf("Connections can no longer be accepted: %v", err)
				break
			}

			log.Printf("Connection accepted")
			go a.handleConnection(conn)
		}
	}
}

func (a *Agent) startupMessage() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n")

	log.Printf("Using AI Model: %s.", a.config.Model)
}

func (a *Agent) handleConnection(conn net.Conn) {
	defer conn.Close()
	defer log.Printf("Connection closed")

	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 3. Count the tokens using model's APIs
	// 4. Return tokens used and percentage of total quota used
	// 5. Return a risk level of the request. Low = Low cost, Medium = Medium cost, High = High / Unknown cost. Costs are configurable

	r, err := a.assesPrompt("Hello world")
	if err != nil {
		log.Printf("assesPrompt failed: %v", err)
		return
	}

	conn.Write([]byte(fmt.Sprintf("Hello. You currently have %d UncachedInputTokens left.\nYour prompt has a risk level of %s", 1, r)))
}

func (a *Agent) processUsageUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.usageReceived:
			log.Printf("Received updated usage stats")
		}
	}
}

// Asses determines the risk level of a given prompt.
func (a *Agent) assesPrompt(prompt string) (RiskLevel, error) {
	// TODO: Leverage the appropaite client's token counting functionality
	switch {
	case llm.IsAnthropicModel(a.config.Model):
		log.Printf("Counting tokens for prompt %s", prompt)
		return RiskLevelLow, nil
	case llm.IsOpenAIModel(a.config.Model):
		return RiskLevelUnknown, fmt.Errorf("OpenAI is not supported at this time")
	default:
		return RiskLevelUnknown, fmt.Errorf("unknown model")
	}
}
