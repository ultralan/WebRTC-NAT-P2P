package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"relay-server/internal/config"
	"relay-server/internal/relay"
)

func main() {
	cfg := config.Load()

	srv, err := relay.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create relay: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		fmt.Printf("Failed to start relay: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	srv.Stop()
}
