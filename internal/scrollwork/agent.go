package scrollwork

import (
	"context"
	"fmt"
	"log"
	"net"
	"scrollwork/internal/llm"
	"scrollwork/internal/usage"
	"sync"
	"time"

	_ "embed"
)

//go:embed banner.txt
var banner []byte

type (
	AgentConfig struct {
		Model                       string
		APIKey                      string
		AdminKey                    string
		RefreshUsageIntervalMinutes int
	}

	Agent struct {
		config *AgentConfig

		listener *net.UnixListener
		worker   *UsageWorker

		anthropicClient *llm.AnthropicClient
		openAIClient    struct{}

		usageReceived chan int
		workerReady   chan bool

		currentUsage usage.Usage

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
func NewAgent(config *AgentConfig) (*Agent, error) {
	if config.Model == "" {
		return nil, fmt.Errorf("NewAgent failed: missing LLM model")
	}

	var wg sync.WaitGroup
	usageReceived := make(chan int, 1)
	workerReady := make(chan bool, 1)

	workerConfig := &UsageWorkerConfig{
		Model:         config.Model,
		UsageReceived: usageReceived,
		WorkerReady:   workerReady,
		TickRate:      config.RefreshUsageIntervalMinutes,
	}

	// An agent always has a usage worker
	worker := newUsageWorker(workerConfig)

	return &Agent{
		config: config,

		worker: worker,

		usageReceived: usageReceived,
		workerReady:   workerReady,
		wg:            &wg,
		currentUsage:  usage.Usage{},
	}, nil
}

// Start starts the Scrollwork Agent.
func (a *Agent) Start(ctx context.Context) error {
	supportedLLMModel := llm.IsAnthropicModel(a.config.Model) || llm.IsOpenAIModel(a.config.Model)
	if !supportedLLMModel {
		return fmt.Errorf("failed to Start: LLM model must either be an OpenAI model or Anthropic model")
	}

	// TODO: Remove this check once we have OpenAI integrated
	if llm.IsOpenAIModel(a.config.Model) {
		return fmt.Errorf("failed to Start: OpenAI is not supported at this time.")
	}

	if llm.IsAnthropicModel(a.config.Model) {
		anthropicClient := llm.NewAnthropicClient(a.config.APIKey, a.config.AdminKey, a.config.Model)
		a.anthropicClient = anthropicClient
		a.worker.AnthropicClient = anthropicClient
	}

	a.startupMessage()

	// Startup the Usage Worker
	log.Printf("Scrollwork Usage Worker starting up...")
	workerStartCtx, workerStartCancel := context.WithTimeout(ctx, 5*time.Second)
	defer workerStartCancel()
	a.worker.Start(workerStartCtx)

	// Wait until worker is ready to run before we start the UNIX listener
	select {
	case <-workerStartCtx.Done():
		if workerStartCtx.Err() != nil {
			return fmt.Errorf("Scrollwork Usage Worker startup aborted")
		}

		return nil

	case <-a.workerReady:
		log.Printf("Scrollwork Usage Worker is ready")
		workerStartCancel()
	}

	return nil
}

func (a *Agent) Run(ctx context.Context) error {
	if a.worker == nil {
		return fmt.Errorf("Scrollwork Agent failed to start: Usage Worker not configured")
	}

	// Run usage Worker
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.worker.Run(ctx)
	}()

	// Handle updates to current usage
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.processUsageUpdates(ctx)
	}()

	log.Printf("Scrollwork Usage Worker is now running")

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

	log.Printf("Scrollwork Agent is now running and ready to accept connections")
	return nil
}

// Stop stops the Scrollwork Agent.
func (a *Agent) Stop() error {
	// Shut down the usage worker
	go func() {
		a.worker.Stop()
	}()

	// TODO: We should probably have a context.Deadline here in case worker shutdownm fails
	select {
	case <-a.worker.config.WorkerReady:
		log.Printf("Scrollwork Usage Worker has shut down")
	}

	// Shut down the UNIX socket
	if a.listener != nil {
		a.listener.Close()
	}

	// Wait for everything else to clean up
	a.wg.Wait()

	return nil
}

func (a *Agent) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork Agent socket has been terminated")
			return
		default:
			conn, err := a.listener.AcceptUnix()
			if err != nil {
				log.Printf("Scrollwork Agent connections can no longer be accepted: %v", err)
				break
			}

			log.Printf("Scrollwork Agent connection accepted")
			go a.handleConnection(ctx, conn)
		}
	}
}

func (a *Agent) startupMessage() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("")
	fmt.Println("")

	fmt.Println("Using LLM Model:", a.config.Model)
	fmt.Println("")
}

func (a *Agent) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	defer log.Printf("Connection closed")

	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 3. Count the tokens using model's APIs
	// 4. Return tokens used and percentage of total quota used
	// 5. Return a risk level of the request. Low = Low cost, Medium = Medium cost, High = High / Unknown cost. Costs are configurable

	r, err := a.assesPrompt(ctx, "Hello world")
	if err != nil {
		log.Printf("assesPrompt failed: %v", err)
		return
	}

	conn.Write([]byte(fmt.Sprintf("Your prompt has a risk level of %s", r)))
}

func (a *Agent) processUsageUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tokens := <-a.usageReceived:
			a.currentUsage.Update(tokens)
			log.Printf("Current Usage: %d tokens", a.currentUsage.Tokens())
			break
		}
	}
}

// Asses determines the risk level of a given prompt.
func (a *Agent) assesPrompt(ctx context.Context, prompt string) (RiskLevel, error) {
	switch {
	case llm.IsAnthropicModel(a.config.Model):
		log.Printf("Counting tokens for prompt %s", prompt)
		_, err := a.anthropicClient.CountTokens(ctx, prompt)
		if err != nil {
			return RiskLevelUnknown, err
		}

		return RiskLevelLow, nil
	case llm.IsOpenAIModel(a.config.Model):
		return RiskLevelUnknown, fmt.Errorf("OpenAI is not supported at this time")
	default:
		return RiskLevelUnknown, fmt.Errorf("unknown model")
	}
}
