package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"

	"simplefrp/internal/auth"
	"simplefrp/internal/config"
	"simplefrp/internal/forward"
	"simplefrp/internal/httputil"
	"simplefrp/internal/idgen"
	"simplefrp/internal/protocol"
	"simplefrp/internal/tunnel"
)

var ErrNoClient = errors.New("no client session")

type Hub struct {
	log *slog.Logger
	cfg config.ServerConfig

	mu        sync.RWMutex
	session   *yamux.Session
	clientID  string
	sessionID string

	up       atomic.Int64
	down     atomic.Int64
	visitors atomic.Uint64
	reject   atomic.Uint64
	streams  atomic.Int64
}

func New(cfg config.ServerConfig, log *slog.Logger) *Hub {
	return &Hub{cfg: cfg, log: log}
}

func (h *Hub) Run(ctx context.Context) error {
	ctrlLn, err := net.Listen("tcp", h.cfg.BindCtrl)
	if err != nil {
		return err
	}
	visLn, err := net.Listen("tcp", h.cfg.BindVisitor)
	if err != nil {
		_ = ctrlLn.Close()
		return err
	}
	apiLn, err := net.Listen("tcp", h.cfg.BindAPI)
	if err != nil {
		_ = ctrlLn.Close()
		_ = visLn.Close()
		return err
	}

	h.log.Info("server listening",
		"ctrl", h.cfg.BindCtrl,
		"visitor", h.cfg.BindVisitor,
		"api", h.cfg.BindAPI,
	)

	api := &http.Server{
		Handler:           h.apiMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := api.Serve(apiLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			h.log.Error("api server", "err", err)
		}
	}()
	go h.acceptCtrl(ctx, ctrlLn)
	go h.acceptVisitor(ctx, visLn)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = api.Shutdown(shutdownCtx)
	_ = ctrlLn.Close()
	_ = visLn.Close()
	h.dropSession()
	return nil
}

func (h *Hub) acceptCtrl(ctx context.Context, ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				h.log.Warn("ctrl accept", "err", err)
				continue
			}
		}
		go h.handleCtrl(ctx, conn)
	}
}

func (h *Hub) handleCtrl(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	var req protocol.AuthRequest
	if err := protocol.ReadFrame(conn, &req); err != nil {
		h.log.Warn("auth read", "err", err)
		return
	}
	if err := req.Validate(); err != nil {
		_ = protocol.WriteFrame(conn, protocol.AuthResponse{Type: protocol.TypeAuthFail, Reason: err.Error()})
		h.log.Warn("auth rejected", "err", err)
		return
	}
	if err := auth.Equal(req.Token, h.cfg.Token); err != nil {
		_ = protocol.WriteFrame(conn, protocol.AuthResponse{Type: protocol.TypeAuthFail, Reason: "invalid token"})
		h.log.Warn("auth token mismatch", "client_id", req.ClientID)
		return
	}
	sid := idgen.SessionID()
	if err := protocol.WriteFrame(conn, protocol.AuthResponse{Type: protocol.TypeAuthOK, SessionID: sid}); err != nil {
		return
	}
	_ = conn.SetDeadline(time.Time{})

	sess, err := tunnel.NewOpening(conn)
	if err != nil {
		h.log.Error("yamux open", "err", err)
		return
	}
	h.setSession(sess, req.ClientID, sid)
	h.log.Info("client registered", "client_id", req.ClientID, "session_id", sid)

	<-sess.CloseChan()
	h.clearIfCurrent(sess)
	h.log.Info("client disconnected", "client_id", req.ClientID, "session_id", sid)
	_ = ctx.Err()
}

func (h *Hub) acceptVisitor(ctx context.Context, ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				h.log.Warn("visitor accept", "err", err)
				continue
			}
		}
		go h.handleVisitor(ctx, conn)
	}
}

func (h *Hub) handleVisitor(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	h.visitors.Add(1)
	stream, err := h.openStream()
	if err != nil {
		h.reject.Add(1)
		h.log.Warn("visitor rejected", "err", err, "remote", conn.RemoteAddr().String())
		return
	}
	h.streams.Add(1)
	defer h.streams.Add(-1)
	defer stream.Close()
	h.log.Debug("stream opened", "remote", conn.RemoteAddr().String())
	if err := forward.Pipe(ctx, conn, stream, &h.up, &h.down); err != nil && !errors.Is(err, context.Canceled) {
		h.log.Debug("forward ended", "err", err)
	}
}

func (h *Hub) apiMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
			"status": "ok",
			"role":   "server",
		}))
	})
	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		h.mu.RLock()
		connected := h.session != nil && !h.session.IsClosed()
		clientID := h.clientID
		sid := h.sessionID
		h.mu.RUnlock()
		httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
			"role":              "server",
			"client_connected":  connected,
			"client_id":         clientID,
			"session_id":        sid,
			"active_streams":    h.streams.Load(),
			"bytes_up":          h.up.Load(),
			"bytes_down":        h.down.Load(),
			"visitors_total":    h.visitors.Load(),
			"visitors_rejected": h.reject.Load(),
		}))
	})
	mux.HandleFunc("/", httputil.NotFound)
	return mux
}

func (h *Hub) setSession(sess *yamux.Session, clientID, sid string) {
	h.mu.Lock()
	if h.session != nil {
		_ = h.session.Close()
	}
	h.session = sess
	h.clientID = clientID
	h.sessionID = sid
	h.mu.Unlock()
}

func (h *Hub) clearIfCurrent(sess *yamux.Session) {
	h.mu.Lock()
	if h.session == sess {
		h.session = nil
	}
	h.mu.Unlock()
}

func (h *Hub) dropSession() {
	h.mu.Lock()
	if h.session != nil {
		_ = h.session.Close()
		h.session = nil
	}
	h.mu.Unlock()
}

func (h *Hub) openStream() (*yamux.Stream, error) {
	h.mu.RLock()
	sess := h.session
	h.mu.RUnlock()
	if sess == nil || sess.IsClosed() {
		return nil, ErrNoClient
	}
	return sess.OpenStream()
}

