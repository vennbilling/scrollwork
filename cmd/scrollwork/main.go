package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "embed"
)

//go:embed banner.txt
var banner []byte

type ScrollworkAgent struct {
	listener *net.UnixListener

	done   chan os.Signal
	cancel context.CancelFunc
}

func main() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n")

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	addr := net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"}

	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		log.Fatal("Scrollwork failed to start:", err)
	}

	sw := ScrollworkAgent{
		listener: listener,

		done:   sigChan,
		cancel: cancel,
	}

	sw.Start(ctx)
}

func (sw *ScrollworkAgent) Start(ctx context.Context) {
	go func() {
		sw.Shutdown(<-sw.done)
	}()

	// TODO: Based on the AI Client configured:
	// 1. Fetch the current quota
	// 2. Spin up a worker that fetches the current quota after X minutes
	// 3. Update the current quota
	// 4. Use current quota when doing count tokens logic

	socketName := sw.listener.Addr().String()

	log.Printf("Scrollwork socket listening at %s", socketName)
	log.Printf("Waiting for connnections...")
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

func (sw *ScrollworkAgent) Shutdown(sig os.Signal) {
	log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)

	sw.cancel()
	sw.listener.Close()
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
