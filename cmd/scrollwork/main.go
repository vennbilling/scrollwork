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

func main() {
	fmt.Println(string(banner))
	fmt.Println("Get your AI limits in real time. Built by Venn Billing.")
	fmt.Println("https://github.com/vennbilling/scrollwork")
	fmt.Println("\n\n\n\n")

	socketName := "/tmp/scrollwork.sock"
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	addr := net.UnixAddr{Name: socketName, Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		log.Fatal("Scrollwork failed to start:", err)
	}

	go func() {
		sig := <-sigChan
		log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)
		cancel()
		listener.Close()
	}()

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
}
