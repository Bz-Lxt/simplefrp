package tunnel

import (
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

func Config() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 8 * time.Second
	cfg.ConnectionWriteTimeout = 10 * time.Second
	cfg.MaxStreamWindowSize = 256 * 1024
	cfg.LogOutput = io.Discard
	return cfg
}

// NewOpening is used by the public server: it opens a logical stream per visitor.
func NewOpening(conn net.Conn) (*yamux.Session, error) {
	return yamux.Client(conn, Config())
}

// NewAccepting is used by the edge client: it accepts logical streams from the server.
func NewAccepting(conn net.Conn) (*yamux.Session, error) {
	return yamux.Server(conn, Config())
}
