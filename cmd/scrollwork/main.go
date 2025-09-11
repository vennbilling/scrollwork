package main

import (
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

	l, err := net.Listen("unix", socketName)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	log.Printf("Scrollwork socket listening at %s", socketName)
	log.Printf("Waiting for connnections...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received %v signal. Scrollwork is shutting down...\n", sig)

	l.Close()
	log.Printf("Scrollwork socket at %s closed", socketName)
}
