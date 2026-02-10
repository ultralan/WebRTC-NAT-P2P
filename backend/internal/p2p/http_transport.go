package p2p

import (
	"io"
	"net"
	"net/http"

	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

const HTTPProtocol = "/http/1.1"
const PingProtocol = "/ping/1.0.0"

type HTTPTransport struct {
	listener net.Listener
	server   *http.Server
}

func NewHTTPTransport(h host.Host, handler http.Handler) (*HTTPTransport, error) {
	listener, err := gostream.Listen(h, HTTPProtocol)
	if err != nil {
		return nil, err
	}

	return &HTTPTransport{
		listener: listener,
		server:   &http.Server{Handler: handler},
	}, nil
}

func (t *HTTPTransport) Serve() error {
	return t.server.Serve(t.listener)
}

func (t *HTTPTransport) Close() error {
	return t.server.Close()
}

func RegisterPingProtocol(h host.Host) {
	h.SetStreamHandler(PingProtocol, handlePing)
}

func handlePing(s network.Stream) {
	defer s.Close()

	buf := make([]byte, 64)
	_, err := s.Read(buf)
	if err != nil && err != io.EOF {
		return
	}

	s.Write([]byte("pong"))
}
