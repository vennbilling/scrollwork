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
	"syscall"

	_ "embed"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

//go:embed banner.txt
var banner []byte

var (
	model  string
	apiKey string
)

type ScrollworkAgent struct {
	listener *net.UnixListener
	model    string

	AnthropicClient anthropic.Client

	done   chan os.Signal
	cancel context.CancelFunc
}

func init() {
	flag.StringVar(&model, "model", "", "AI Model")
	flag.StringVar(&apiKey, "apiKey", "", "API Key")
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

	sw := ScrollworkAgent{
		model: model,

		done:   sigChan,
		cancel: cancel,
	}

	// Configure UNIX socket and lisenter
	addr := net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		log.Fatal("failed to listen on unix socket:", err)
	}

	sw.listener = listener

	// Configure AI Model and Client
	if !sw.IsAnthropic() && !sw.IsOpenAI() {
		listener.Close()
		log.Fatal("Only OpenAI and Anthropic models are supported.")
	}

	if sw.IsAnthropic() {
		sw.AnthropicClient = anthropic.NewClient(option.WithAPIKey(apiKey))
	}

	sw.Start(ctx)
}

func (sw *ScrollworkAgent) Start(ctx context.Context) {
	go func() {
		sig := <-sw.done

		log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)
		sw.Shutdown()
	}()

	// TODO: Based on the AI Client configured:
	// 1. Fetch the current quota
	// 2. Spin up a worker that fetches the current quota after X minutes
	// 3. Update the current quota
	// 4. Use current quota when doing count tokens logic

	socketName := sw.listener.Addr().String()

	log.Printf("Scrollwork has started and can receive connections")
	log.Printf("Current AI Model: %s.", sw.model)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork socket at %s closed", socketName)
			return
		default:
			conn, err := sw.listener.AcceptUnix()
			if err != nil {
				log.Printf("Connections can no longer be accepted: %v", err)
				break
			}

			log.Printf("Connection accepted")
			go handleConnection(conn)
		}
	}
}

func (sw *ScrollworkAgent) Shutdown() {
	sw.cancel()
	sw.listener.Close()
}

func (sw *ScrollworkAgent) IsAnthropic() bool {
	return strings.Contains(sw.model, "claude-")
}

func (sw *ScrollworkAgent) IsOpenAI() bool {
	return strings.Contains(sw.model, "gpt-") || strings.Contains(sw.model, "text-")
}

func handleConnection(conn net.Conn) {
	conn.Write([]byte("Hello\n"))
	conn.Close()

	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 1. parse the message out of the JSON
	// 2. Count the tokens
	// 3. Return tokens used and percentage of total quota used
}
