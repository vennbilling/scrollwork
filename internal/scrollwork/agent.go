package scrollwork

import (
	"context"
	"fmt"
	"log"
	"net"
	"scrollwork/internal/llm"
	"scrollwork/internal/usage"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed banner.txt
var banner []byte

type (
	AgentConfig struct {
		Models                      []string
		APIKey                      string
		AdminKey                    string
		RefreshUsageIntervalMinutes int

		APIKeys *llm.APIKeys

		LowRiskThreshold    float32
		MediumRiskThreshold float32
		HigthRiskThreshold  float32
	}

	Agent struct {
		config *AgentConfig

		listener *net.UnixListener
		worker   *UsageWorker

		llmClient       *llm.APIClient
		anthropicClient *llm.AnthropicClient
		openAIClient    struct{}

		usageReceived chan int
		workerReady   chan bool

		currentUsageTokens map[string]int
		usageMu            sync.Mutex
		riskThresholds     usage.RiskThresholds

		wg *sync.WaitGroup
	}
)

// NewAgent returns an Agent.
// A Scrollwork Agent is responsible for handling requests to check the billing risk level of an AI Prompt.
// It also spins up a worker that periodically checks and syncs an organization's current usage.
// This usage is used when calculating the risk of a AI Prompt.
func NewAgent(config *AgentConfig) (*Agent, error) {
	if len(config.Models) == 0 {
		return nil, fmt.Errorf("NewAgent failed: missing LLM models")
	}

	var wg sync.WaitGroup
	usageReceived := make(chan int, 1)
	workerReady := make(chan bool, 1)

	c := llm.ClientConfig{
		Models:  config.Models,
		APIKeys: config.APIKeys,
	}
	llmClient := llm.NewAPIClient(c)

	workerConfig := &UsageWorkerConfig{
		Model:         config.Models[0], // Use first model for now (single model support)
		UsageReceived: usageReceived,
		WorkerReady:   workerReady,
		TickRate:      config.RefreshUsageIntervalMinutes,
		Client:        llmClient,
	}

	// An agent always has a usage worker
	worker := newUsageWorker(workerConfig)

	riskThresholds := usage.NewRiskThresholds(config.LowRiskThreshold, config.MediumRiskThreshold, config.HigthRiskThreshold)

	return &Agent{
		config: config,

		worker: worker,

		llmClient: llmClient,

		usageReceived:  usageReceived,
		workerReady:    workerReady,
		wg:             &wg,
		currentUsageTokens: make(map[string]int),
		riskThresholds: riskThresholds,
	}, nil
}

// Start starts the Scrollwork Agent.
func (a *Agent) Start(ctx context.Context) error {
	for _, model := range a.config.Models {
		supportedLLMModel := llm.IsAnthropicModel(model) || llm.IsOpenAIModel(model)
		if !supportedLLMModel {
			return fmt.Errorf("failed to Start: LLM model %s must either be an OpenAI model or Anthropic model", model)
		}

		// TODO: Remove this check once we have OpenAI integrated
		if llm.IsOpenAIModel(model) {
			return fmt.Errorf("failed to Start: OpenAI model %s is not supported at this time", model)
		}

		// TODO: We should do something like llm.NewAPIClient and obfuscate the Anthropic and OpenAI clients. Scrollwork package shouldn't really care
		// or have any logic based on the model we are using.
		if llm.IsAnthropicModel(model) {
			if a.anthropicClient == nil {
				anthropicClient := llm.NewAnthropicClient(a.config.APIKey, a.config.AdminKey, model)
				a.anthropicClient = anthropicClient
				a.worker.AnthropicClient = anthropicClient
			}
		}
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

	fmt.Println("Using LLM Models:", strings.Join(a.config.Models, ", "))
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

	messages := []llm.Message{{Role: llm.MessageRoleUser, Content: "Hello world"}}
	r, err := a.assesPrompt(ctx, messages)
	if err != nil {
		log.Printf("assesPrompt failed: %v", err)
		return
	}

	conn.Write([]byte(fmt.Sprintf("You are currently using %d tokens. Your prompt has a risk level of %s", a.getTotalUsage(), r)))
}

func (a *Agent) processUsageUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tokens := <-a.usageReceived:
			// For now, update the first model until we have multi-model worker support
			if len(a.config.Models) > 0 {
				a.updateUsage(a.config.Models[0], tokens)
			}
			log.Printf("Current Usage: %d tokens", a.getTotalUsage())
			break
		}
	}
}

// updateUsage updates the token usage for a specific model in a thread-safe manner.
func (a *Agent) updateUsage(model string, tokens int) {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()

	a.currentUsageTokens[model] = tokens
}

// getUsage returns the current token usage for a specific model in a thread-safe manner.
func (a *Agent) getUsage(model string) int {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()

	return a.currentUsageTokens[model]
}

// getTotalUsage returns the total token usage across all models in a thread-safe manner.
func (a *Agent) getTotalUsage() int {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()

	total := 0
	for _, tokens := range a.currentUsageTokens {
		total += tokens
	}
	return total
}

// Asses determines the risk level of a given prompt.
func (a *Agent) assesPrompt(ctx context.Context, messages []llm.Message) (usage.RiskLevel, error) {
	for _, model := range a.config.Models {
		switch {
		case llm.IsAnthropicModel(model):
			tokens, err := a.anthropicClient.CountTokens(ctx, messages)
			if err != nil {
				return usage.RiskLevelUnknown, err
			}

			level := a.riskThresholds.Asses(tokens)

			return level, nil
		case llm.IsOpenAIModel(model):
			return usage.RiskLevelUnknown, fmt.Errorf("OpenAI model %s is not supported at this time", model)
		default:
			return usage.RiskLevelUnknown, fmt.Errorf("unknown model: %s", model)
		}
	}

	return usage.RiskLevelUnknown, fmt.Errorf("no models configured")
}
