package client

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"

	"simplefrp/internal/config"
	"simplefrp/internal/forward"
	"simplefrp/internal/httputil"
	"simplefrp/internal/pool"
	"simplefrp/internal/protocol"
	"simplefrp/internal/tunnel"
)

type Agent struct {
	cfg  config.ClientConfig
	log  *slog.Logger
	pool *pool.Pool

	connected  atomic.Bool
	reconnects atomic.Uint64
}

func New(cfg config.ClientConfig, log *slog.Logger) *Agent {
	factory := func(ctx context.Context) (net.Conn, error) {
		d := net.Dialer{Timeout: 3 * time.Second}
		return d.DialContext(ctx, "tcp", cfg.LocalAddr)
	}
	p := pool.New(factory, pool.Config{
		MaxIdle:   cfg.MaxIdle,
		MaxActive: cfg.MaxActive,
		IdleTTL:   cfg.IdleTTL,
		Wait:      cfg.PoolWait,
	})
	return &Agent{cfg: cfg, log: log, pool: p}
}

func (a *Agent) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", a.handleHealth)
	mux.HandleFunc("/api/v1/ready", a.handleReady)
	mux.HandleFunc("/api/v1/status", a.handleStatus)
	mux.HandleFunc("/", httputil.NotFound)
	hs := &http.Server{Addr: a.cfg.BindHealth, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		a.log.Info("client health listening", "addr", a.cfg.BindHealth)
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("health server", "err", err)
		}
	}()

	backoff := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = hs.Shutdown(shutdownCtx)
			cancel()
			a.pool.Close()
			return ctx.Err()
		default:
		}
		err := a.session(ctx)
		was := a.connected.Swap(false)
		if ctx.Err() != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = hs.Shutdown(shutdownCtx)
			cancel()
			a.pool.Close()
			return ctx.Err()
		}
		if was {
			backoff = 0
		}
		a.reconnects.Add(1)
		backoff = config.NextBackoff(backoff)
		a.log.Warn("control disconnected, reconnecting", "err", err, "backoff", backoff.String())
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = hs.Shutdown(shutdownCtx)
			cancel()
			a.pool.Close()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (a *Agent) session(ctx context.Context) error {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", a.cfg.ServerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := protocol.AuthRequest{Type: protocol.TypeAuth, Token: a.cfg.Token, ClientID: a.cfg.ClientID}
	if err := protocol.WriteFrame(conn, req); err != nil {
		return err
	}
	var resp protocol.AuthResponse
	if err := protocol.ReadFrameDeadline(conn, &resp, 8*time.Second); err != nil {
		return err
	}
	if resp.Type != protocol.TypeAuthOK {
		return errors.New(resp.Reason)
	}

	sess, err := tunnel.NewAccepting(conn)
	if err != nil {
		return err
	}
	defer sess.Close()
	a.connected.Store(true)
	a.log.Info("registered with server", "session_id", resp.SessionID, "client_id", a.cfg.ClientID)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.acceptLoops(ctx, sess)
	}()
	select {
	case <-ctx.Done():
		_ = sess.Close()
		return ctx.Err()
	case <-sess.CloseChan():
		return errors.New("yamux session closed")
	case err := <-errCh:
		return err
	}
}

func (a *Agent) acceptLoops(ctx context.Context, sess *yamux.Session) error {
	for {
		stream, err := sess.AcceptStream()
		if err != nil {
			return err
		}
		go a.handleStream(ctx, stream)
	}
}

func (a *Agent) handleStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, a.cfg.PoolWait)
	defer cancel()
	local, err := a.pool.Get(ctx)
	if err != nil {
		a.log.Warn("pool exhausted", "err", err)
		return
	}
	a.log.Debug("stream accepted, local conn acquired")
	if err := forward.Pipe(ctx, stream, local, nil, nil); err != nil && !errors.Is(err, context.Canceled) {
		a.log.Debug("stream forward ended", "err", err)
	}
	a.pool.Discard(local)
}

func (a *Agent) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
		"status":    "ok",
		"role":      "client",
		"connected": a.connected.Load(),
	}))
}

func (a *Agent) handleReady(w http.ResponseWriter, _ *http.Request) {
	if !a.connected.Load() {
		httputil.WriteError(w, http.StatusServiceUnavailable, "not_ready", "control tunnel is not connected")
		return
	}
	httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
		"status":    "ok",
		"role":      "client",
		"connected": true,
	}))
}

func (a *Agent) handleStatus(w http.ResponseWriter, _ *http.Request) {
	st := a.pool.Stats()
	httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
		"role":       "client",
		"connected":  a.connected.Load(),
		"client_id":  a.cfg.ClientID,
		"reconnects": a.reconnects.Load(),
		"pool":       st,
	}))
}
