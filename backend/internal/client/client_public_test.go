package client_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/yamux"

	"simplefrp/internal/client"
	"simplefrp/internal/config"
	"simplefrp/internal/protocol"
	"simplefrp/internal/tunnel"
)

var echoPayload = []byte("ping")

func TestAgentReleasesPoolSlotAfterVisitorEOF(t *testing.T) {
	localAddr, stopLocal := startSingleRequestEcho(t)
	t.Cleanup(stopLocal)

	ctrlLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ctrlLn.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	agent := client.New(config.ClientConfig{
		ServerAddr: ctrlLn.Addr().String(),
		LocalAddr:  localAddr,
		Token:      "test-token",
		ClientID:   "edge-test",
		BindHealth: "127.0.0.1:0",
		MaxIdle:    1,
		MaxActive:  1,
		IdleTTL:    time.Minute,
		PoolWait:   time.Second,
	}, log)

	runDone := make(chan error, 1)
	go func() {
		runDone <- agent.Run(ctx)
	}()

	var ctrlConn net.Conn
	var opening *yamux.Session
	t.Cleanup(func() {
		cancel()
		if opening != nil {
			_ = opening.Close()
		}
		if ctrlConn != nil {
			_ = ctrlConn.Close()
		}

		select {
		case err := <-runDone:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Agent.Run returned %v, want context cancellation", err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Agent.Run did not stop")
		}
	})

	tcpLn := ctrlLn.(*net.TCPListener)
	if err := tcpLn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	ctrlConn, err = ctrlLn.Accept()
	if err != nil {
		t.Fatal(err)
	}
	_ = tcpLn.SetDeadline(time.Time{})

	if err := ctrlConn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var req protocol.AuthRequest
	if err := protocol.ReadFrame(ctrlConn, &req); err != nil {
		t.Fatal(err)
	}
	if req.Type != protocol.TypeAuth ||
		req.Token != "test-token" ||
		req.ClientID != "edge-test" {
		t.Fatalf("unexpected auth request: %+v", req)
	}
	if err := protocol.WriteFrame(ctrlConn, protocol.AuthResponse{
		Type:      protocol.TypeAuthOK,
		SessionID: "session-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := ctrlConn.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}

	opening, err = tunnel.NewOpening(ctrlConn)
	if err != nil {
		t.Fatal(err)
	}

	if err := visitorRoundTrip(opening); err != nil {
		t.Fatalf("first visitor: %v", err)
	}
	if err := visitorRoundTrip(opening); err != nil {
		t.Fatalf("second visitor after first completed: %v", err)
	}
}

func visitorRoundTrip(sess *yamux.Session) error {
	stream, err := sess.OpenStream()
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	if err := stream.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return fmt.Errorf("set stream deadline: %w", err)
	}
	if _, err := io.Copy(stream, bytes.NewReader(echoPayload)); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	got := make([]byte, len(echoPayload))
	if _, err := io.ReadFull(stream, got); err != nil {
		return fmt.Errorf("read echo: %w", err)
	}
	if !bytes.Equal(got, echoPayload) {
		return fmt.Errorf("echo = %q, want %q", got, echoPayload)
	}

	var extra [1]byte
	n, err := stream.Read(extra[:])
	if n != 0 || err == nil {
		return fmt.Errorf("stream remained open after response: n=%d err=%v", n, err)
	}
	return nil
}

func startSingleRequestEcho(t *testing.T) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu      sync.Mutex
		conns   = make(map[net.Conn]struct{})
		workers sync.WaitGroup
		once    sync.Once
	)
	acceptDone := make(chan struct{})

	go func() {
		defer close(acceptDone)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			mu.Lock()
			conns[conn] = struct{}{}
			mu.Unlock()

			workers.Add(1)
			go func(conn net.Conn) {
				defer workers.Done()
				defer func() {
					_ = conn.Close()
					mu.Lock()
					delete(conns, conn)
					mu.Unlock()
				}()

				buf := make([]byte, len(echoPayload))
				if _, err := io.ReadFull(conn, buf); err != nil {
					return
				}
				_, _ = io.Copy(conn, bytes.NewReader(buf))
			}(conn)
		}
	}()

	stop := func() {
		once.Do(func() {
			_ = ln.Close()
			<-acceptDone

			mu.Lock()
			for conn := range conns {
				_ = conn.Close()
			}
			mu.Unlock()

			workers.Wait()
		})
	}
	return ln.Addr().String(), stop
}
