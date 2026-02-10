package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	ma "github.com/multiformats/go-multiaddr"
)

type P2PHost struct {
	host      host.Host
	relayInfo *peer.AddrInfo
}

func NewP2PHost(relayAddr string) (*P2PHost, error) {
	relayMA, err := ma.NewMultiaddr(relayAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid relay addr: %w", err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMA)
	if err != nil {
		return nil, fmt.Errorf("parse relay info: %w", err)
	}

	return &P2PHost{relayInfo: relayInfo}, nil
}

func (p *P2PHost) Start(ctx context.Context) error {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("create host: %w", err)
	}
	p.host = h

	return p.connectRelay(ctx)
}

func (p *P2PHost) connectRelay(ctx context.Context) error {
	if err := p.host.Connect(ctx, *p.relayInfo); err != nil {
		return fmt.Errorf("connect relay: %w", err)
	}

	_, err := client.Reserve(ctx, p.host, *p.relayInfo)
	if err != nil {
		return fmt.Errorf("reserve relay: %w", err)
	}

	relayedAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s",
			p.relayInfo.ID, p.host.ID()))

	fmt.Println("Backend P2P started!")
	fmt.Printf("PeerID: %s\n", p.host.ID())
	fmt.Printf("Relayed addr: %s\n", relayedAddr)

	return nil
}

func (p *P2PHost) Host() host.Host {
	return p.host
}

func (p *P2PHost) Stop() error {
	if p.host != nil {
		return p.host.Close()
	}
	return nil
}

var _ = swarm.ErrDialBackoff
