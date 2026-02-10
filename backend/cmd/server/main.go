package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/api"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/p2p"
)

func main() {
	cfg := config.Load()

	// 初始化数据库
	if err := db.Init("books.db"); err != nil {
		fmt.Printf("Failed to init database: %v\n", err)
		os.Exit(1)
	}

	host, err := p2p.NewP2PHost(cfg.RelayAddr)
	if err != nil {
		fmt.Printf("Failed to create P2P host: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		fmt.Printf("Failed to start P2P: %v\n", err)
		os.Exit(1)
	}

	p2p.RegisterPingProtocol(host.Host())

	router := api.NewRouter()

	// 启动普通 HTTP 服务器
	go func() {
		addr := fmt.Sprintf(":%d", cfg.ListenPort)
		fmt.Printf("HTTP server listening on %s\n", addr)
		if err := router.Run(addr); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	transport, err := p2p.NewHTTPTransport(host.Host(), router)
	if err != nil {
		fmt.Printf("Failed to create transport: %v\n", err)
		os.Exit(1)
	}

	go transport.Serve()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	transport.Close()
	host.Stop()
}
