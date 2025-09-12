package scrollwork

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type AgentConfig struct {
	Model                       string
	APIKey                      string
	RefreshUsageIntervalMinutes int
}

type Agent struct {
	config *AgentConfig

	listener *net.UnixListener

	usage  chan int
	done   chan os.Signal
	cancel context.CancelFunc

	wg *sync.WaitGroup
}

func NewAgent(config *AgentConfig) *Agent {
	var wg sync.WaitGroup
	return &Agent{
		config: config,

		wg: &wg,
	}
}

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

	// TODO: Configure usage worker and block while we wait for initial usage

	go func() {
		defer a.wg.Done()
		a.listen(ctx)
	}()

	return nil
}

func (a *Agent) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}

	// Shut down the UNIX socket
	if a.listener != nil {
		a.listener.Close()
	}

	// Shut down the usage worker

	a.wg.Wait()

	return nil
}

func (a *Agent) listen(ctx context.Context) {
	log.Printf("Scrollwork Agent socket has started and is now listening for connections.")
	log.Printf("Current AI Model: %s.", a.config.Model)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork Agent socket closed")
			return
		case <-a.usage:
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
