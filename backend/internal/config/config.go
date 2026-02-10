package config

import (
	"os"
	"strconv"
)

type Config struct {
	RelayAddr   string
	ListenPort  int
	APIPrefix   string
}

func Load() *Config {
	port := 8081
	if p := os.Getenv("LISTEN_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	relayAddr := os.Getenv("RELAY_ADDR")
	if relayAddr == "" {
		relayAddr = "/ip4/127.0.0.1/udp/4002/quic-v1/p2p/12D3KooWG53rJbdyC1yqdNuMgVTeE1s4QdFZbtCbkHRraNPbCWLh"
	}

	return &Config{
		RelayAddr:  relayAddr,
		ListenPort: port,
		APIPrefix:  "/api",
	}
}
