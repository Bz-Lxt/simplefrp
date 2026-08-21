package server_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"simplefrp/internal/config"
	"simplefrp/internal/protocol"
	"simplefrp/internal/server"
)

func TestServerRejectsTruncatedFrame(t *testing.T) {
	ctrlAddr := availableTCPAddr(t)

	cfg := config.ServerConfig{
		BindCtrl:    ctrlAddr,
		BindVisitor: "127.0.0.1:0",
		BindAPI:     "127.0.0.1:0",
		Token:       "test-control-token",
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = server.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil))).Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
			if runErr != nil {
				t.Errorf("server stopped with error: %v", runErr)
			}
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	})

	conn := dialTCP(t, ctrlAddr, done, &runErr)
	defer conn.Close()

	body := []byte(`{"type":"auth","token":"test-control-token","client_id":"edge-test"}`)
	frame := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(body)+32))
	copy(frame[4:], body)
	if n, err := io.Copy(conn, bytes.NewReader(frame)); err != nil || n != int64(len(frame)) {
		t.Fatalf("write malformed frame: wrote %d bytes: %v", n, err)
	}
	halfCloser, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		t.Fatal("TCP connection does not support CloseWrite")
	}
	if err := halfCloser.CloseWrite(); err != nil {
		t.Fatalf("close frame writer: %v", err)
	}

	var response protocol.AuthResponse
	err := protocol.ReadFrameDeadline(conn, &response, 2*time.Second)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			t.Fatalf("timed out waiting for the server to reject the truncated frame: %v", err)
		}
		return
	}
	if response.Type == protocol.TypeAuthOK {
		t.Fatalf("truncated frame was authenticated: %+v", response)
	}
}

func availableTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func dialTCP(t *testing.T, addr string, done <-chan struct{}, runErr *error) net.Conn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			return conn
		}
		select {
		case <-done:
			t.Fatalf("server stopped before accepting control connections: %v", *runErr)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("dial control listener: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
