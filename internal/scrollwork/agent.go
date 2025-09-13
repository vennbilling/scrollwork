package scrollwork

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	_ "embed"
)

//go:embed banner.txt
var banner []byte

type AgentConfig struct {
	Model                       string
	APIKey                      string
	RefreshUsageIntervalMinutes int
}

type Agent struct {
	config *AgentConfig

	listener *net.UnixListener
	worker   *UsageWorker

	usageReceived chan int
	cancel        context.CancelFunc

	wg *sync.WaitGroup
}

// NewAgent returns an Agent.
// A Scrollwork Agent is responsible for handling requests to check the billing risk level of an AI Prompt.
// It also spins up a worker that periodically checks and syncs an organization's current usage.
// This usage is used when calculating the risk of a AI Prompt.
func NewAgent(config *AgentConfig) *Agent {
	var wg sync.WaitGroup

	usage := make(chan int, 1)

	return &Agent{
		config: config,

		usageReceived: usage,
		wg:            &wg,
	}
}

// Start starts the Scrollwork Agent.
func (a *Agent) Start(ctx context.Context) error {
	if !a.isUsingAnthropic() && !a.isUsingOpenAI() {
		return fmt.Errorf("invalid AI Model: only OpenAI and Anthropic models are supported.")
	}

	ctx, cancel := context.WithCancel(ctx)

	a.cancel = cancel

	// Configure unix socket listener
	addr := net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		return err
	}

	a.listener = listener
	a.wg.Add(1)

	a.worker = newUsageWorker(a.usageReceived)
	a.wg.Add(1)

	a.startupMessage()

	go func() {
		defer a.wg.Done()
		a.worker.Start(ctx, a.config.RefreshUsageIntervalMinutes)
	}()

	// Wait until worker is ready. When a worker starts, it will also make a request to get the latest usage
	<-a.usageReceived

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
		case <-a.usageReceived:
			log.Printf("Organization usage updated")
			// TODO: Should sync this with a mutex
			break
		default:
			conn, err := a.listener.AcceptUnix()
			if err != nil {
				log.Printf("Connections can no longer be accepted: %v", err)
				break
			}

			log.Printf("Connection accepted")
			go handleConnection(conn)
		}
	}
}

func (a *Agent) isUsingAnthropic() bool {
	return strings.Contains(a.config.Model, "claude-")
}

func (a *Agent) isUsingOpenAI() bool {
	return strings.Contains(a.config.Model, "gpt-") || strings.Contains(a.config.Model, "text-")
}

func (a *Agent) startupMessage() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n")

	log.Printf("Using AI Model: %s.", a.config.Model)
}

func handleConnection(conn net.Conn) {
	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 3. Count the tokens using model's APIs
	// 4. Return tokens used and percentage of total quota used
	// 5. Return a risk level of the request. Low = Low cost, Medium = Medium cost, High = High / Unknown cost. Costs are configurable
	conn.Write([]byte(fmt.Sprintf("Hello. You currently have %d UncachedInputTokens left\n", 1)))
	conn.Close()
	log.Printf("Connection closed")
}
