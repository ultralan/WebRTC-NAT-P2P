package relay

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"

	"relay-server/internal/config"
)

type RelayServer struct {
	host   host.Host
	config *config.Config
}

func New(cfg *config.Config) (*RelayServer, error) {
	return &RelayServer{config: cfg}, nil
}

const keyFile = "relay.key"

func loadOrCreateKey() (crypto.PrivKey, error) {
	data, err := os.ReadFile(keyFile)
	if err == nil {
		return crypto.UnmarshalPrivateKey(data)
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	data, err = crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	os.WriteFile(keyFile, data, 0600)
	return priv, nil
}

func (r *RelayServer) Start(ctx context.Context) error {
	priv, err := loadOrCreateKey()
	if err != nil {
		return fmt.Errorf("load key: %w", err)
	}

	listenAddrs := []string{
		fmt.Sprintf("/ip4/%s/tcp/%d", r.config.ListenHost, r.config.ListenPort),
	}

	if r.config.EnableQUIC {
		listenAddrs = append(listenAddrs,
			fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", r.config.ListenHost, r.config.ListenPort),
		)
	}

	if r.config.EnableWS {
		listenAddrs = append(listenAddrs,
			fmt.Sprintf("/ip4/%s/tcp/%d/ws", r.config.ListenHost, r.config.ListenPort+1),
		)
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(listenAddrs...),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(websocket.New),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.EnableHolePunching(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	r.host = h

	// 手动启动 relay v2 服务
	_, err = relayv2.New(h)
	if err != nil {
		return fmt.Errorf("failed to create relay service: %w", err)
	}

	fmt.Println("Relay Server started!")
	fmt.Printf("PeerID: %s\n", h.ID())
	fmt.Println("Listening on:")
	for _, addr := range h.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, h.ID())
	}

	return nil
}

func (r *RelayServer) Stop() error {
	if r.host != nil {
		return r.host.Close()
	}
	return nil
}

func (r *RelayServer) Host() host.Host {
	return r.host
}
