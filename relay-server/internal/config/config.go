package config

import (
	"os"
	"strconv"
)

type Config struct {
	ListenPort    int
	ListenHost    string
	EnableQUIC    bool
	EnableWS      bool
}

func Load() *Config {
	port := 4002
	if p := os.Getenv("RELAY_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	host := "0.0.0.0"
	if h := os.Getenv("RELAY_HOST"); h != "" {
		host = h
	}

	return &Config{
		ListenPort:    port,
		ListenHost:    host,
		EnableQUIC:    os.Getenv("ENABLE_QUIC") != "false",
		EnableWS:      os.Getenv("ENABLE_WS") != "false",
	}
}
