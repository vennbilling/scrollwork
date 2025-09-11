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

type Scrollwork struct {
	Addr net.UnixAddr
}

func main() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n\n\n")

	sw := Scrollwork{
		Addr: net.UnixAddr{Name: "/tmp/scrollwork.sock", Net: "unix"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	listener, err := net.ListenUnix("unix", &sw.Addr)
	if err != nil {
		log.Fatal("Scrollwork failed to start:", err)
	}

	go func() {
		sig := <-sigChan
		log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)
		cancel()
		listener.Close()
	}()

	// TODO: Based on the AI Client configured:
	// 1. Fetch the current quota
	// 2. Spin up a worker that fetches the current quota after X minutes
	// 3. Update the current quota
	// 4. Use current quota when doing count tokens logic

	log.Printf("Scrollwork socket listening at %s", socketName)
	log.Printf("Waiting for connnections...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scrollwork socket at %s closed", socketName)
			return
		default:
			conn, err := listener.AcceptUnix()
			if err != nil {
				log.Printf("Connections can no longer be accepted: %v", err)
				break
			}

			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn) {
	log.Printf("Connection accepted")
	conn.Write([]byte("Hello\n"))
	conn.Close()

	// TODO:
	// 1. parse the JSON received, split by \n
	// 2. Validate. throw out invalid json
	// 1. parse the message out of the JSON
	// 2. Count the tokens
	// 3. Return tokens used and percentage of total quota used
}
