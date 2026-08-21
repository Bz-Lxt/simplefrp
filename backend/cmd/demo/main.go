package main

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"simplefrp/internal/clock"
	"simplefrp/internal/config"
	"simplefrp/internal/httputil"
	"simplefrp/internal/logger"
)

func main() {
	cfg := config.LoadDemo()
	log := logger.New(cfg.LogLevel, os.Stdout)

	hostname, _ := os.Hostname()
	var hits atomic.Uint64

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteData(w, http.StatusOK, httputil.StampMap(map[string]any{
			"status":  "ok",
			"role":    "demo",
			"node_id": cfg.NodeID,
		}))
	})
	mux.HandleFunc("/api/v1/identity", func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		httputil.WriteData(w, http.StatusOK, map[string]any{
			"node_id":      cfg.NodeID,
			"hostname":     hostname,
			"hits":         n,
			"remote_addr":  r.RemoteAddr,
			"time":         clock.Stamp(),
			"time_rfc3339": clock.Now().Format(time.RFC3339),
		})
	})
	mux.Handle("/", spaHandler(cfg.StaticDir))

	hs := &http.Server{
		Addr:              cfg.BindHTTP,
		Handler:           mux,
		ReadHeaderTimeout: 0,
		ReadTimeout:       0,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("demo listening", "addr", cfg.BindHTTP, "node_id", cfg.NodeID, "static", cfg.StaticDir)
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("demo server", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = hs.Shutdown(shutdownCtx)
}

func spaHandler(dir string) http.Handler {
	root := filepath.Clean(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		name := root
		if rel != "" && rel != "." {
			name = filepath.Join(root, filepath.FromSlash(rel))
		}
		if !isInside(root, name) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "invalid path")
			return
		}
		if info, err := os.Stat(name); err == nil && !info.IsDir() {
			http.ServeFile(w, r, name)
			return
		}
		index := filepath.Join(root, "index.html")
		if _, err := os.Stat(index); errors.Is(err, fs.ErrNotExist) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "static site not built")
			return
		}
		http.ServeFile(w, r, index)
	})
}

func isInside(root, name string) bool {
	root = filepath.Clean(root)
	name = filepath.Clean(name)
	if name == root {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(name, root+sep)
}
