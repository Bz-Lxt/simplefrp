package client_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"simplefrp/internal/client"
	"simplefrp/internal/config"
	"simplefrp/internal/logger"
)

func TestAgentRetriesAfterInitialHandshakeFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	accepted := make(chan struct{}, 4)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
			select {
			case accepted <- struct{}{}:
			default:
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	agent := client.New(config.ClientConfig{
		ServerAddr: ln.Addr().String(),
		LocalAddr:  "127.0.0.1:1",
		Token:      "test-token",
		ClientID:   "retry-edge",
		BindHealth: "127.0.0.1:0",
		MaxIdle:    1,
		MaxActive:  1,
		IdleTTL:    time.Minute,
		PoolWait:   time.Second,
	}, logger.New("error", io.Discard))

	var runErr error
	runDone := make(chan struct{})
	go func() {
		runErr = agent.Run(ctx)
		close(runDone)
	}()

	defer func() {
		cancel()
		_ = ln.Close()
		select {
		case <-runDone:
		case <-time.After(3 * time.Second):
			t.Errorf("Agent.Run did not stop during cleanup")
		}
		select {
		case <-acceptDone:
		case <-time.After(time.Second):
			t.Errorf("control listener did not stop during cleanup")
		}
	}()

	select {
	case <-accepted:
	case <-runDone:
		t.Fatalf("Agent.Run returned before the first control connection: %v", runErr)
	case <-time.After(2 * time.Second):
		t.Fatal("client did not make the first control connection")
	}

	select {
	case <-accepted:
	case <-runDone:
		t.Fatalf("Agent.Run stopped after a transient handshake failure: %v", runErr)
	case <-time.After(3 * time.Second):
		t.Fatal("client did not retry the control connection")
	}

	cancel()
	select {
	case <-runDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Agent.Run did not stop after cancellation")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Agent.Run returned %v after cancellation, want context.Canceled", runErr)
	}
}
