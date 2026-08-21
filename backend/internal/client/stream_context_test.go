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

	"simplefrp/internal/client"
	"simplefrp/internal/config"
	"simplefrp/internal/server"
)

func TestAgentEstablishedStreamOutlivesPoolWait(t *testing.T) {
	const poolWait = 500 * time.Millisecond

	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	echoDone := make(chan struct{})
	go serveEcho(echoLn, echoDone)
	var visitor net.Conn
	var stopAgent, stopHub context.CancelFunc
	var agentDone, hubDone chan error
	t.Cleanup(func() {
		if visitor != nil {
			_ = visitor.Close()
		}
		if stopAgent != nil {
			stopAgent()
		}
		if stopHub != nil {
			stopHub()
		}
		_ = echoLn.Close()
		if agentDone != nil {
			waitForRunExit(t, "agent", agentDone, context.Canceled)
		}
		if hubDone != nil {
			waitForRunExit(t, "server", hubDone, nil)
		}
		select {
		case <-echoDone:
		case <-time.After(2 * time.Second):
			t.Error("echo listener did not stop")
		}
	})

	addrs := availableTCPAddrs(t, 2)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hubCtx, cancelHub := context.WithCancel(context.Background())
	stopHub = cancelHub
	hubDone = make(chan error, 1)
	hub := server.New(config.ServerConfig{
		BindCtrl:    addrs[0],
		BindVisitor: addrs[1],
		BindAPI:     "127.0.0.1:0",
		Token:       "stream-context-token",
	}, log)
	go func() { hubDone <- hub.Run(hubCtx) }()
	waitForListener(t, addrs[0])

	agentCtx, cancelAgent := context.WithCancel(context.Background())
	stopAgent = cancelAgent
	agentDone = make(chan error, 1)
	agent := client.New(config.ClientConfig{
		ServerAddr: addrs[0],
		LocalAddr:  echoLn.Addr().String(),
		Token:      "stream-context-token",
		ClientID:   "edge-context-test",
		BindHealth: "127.0.0.1:0",
		MaxIdle:    1,
		MaxActive:  8,
		IdleTTL:    time.Second,
		PoolWait:   poolWait,
	}, log)
	go func() { agentDone <- agent.Run(agentCtx) }()

	visitor = waitForForwarding(t, addrs[1])
	time.Sleep(2 * poolWait)
	if err := echoRoundTrip(visitor, []byte("still-open"), time.Second); err != nil {
		t.Fatalf("established visitor stream stopped after pool wait elapsed: %v", err)
	}
}

func availableTCPAddrs(t *testing.T, count int) []string {
	t.Helper()
	listeners := make([]net.Listener, 0, count)
	for i := 0; i < count; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			t.Fatal(err)
		}
		listeners = append(listeners, ln)
	}
	addrs := make([]string, len(listeners))
	for i, ln := range listeners {
		addrs[i] = ln.Addr().String()
		_ = ln.Close()
	}
	return addrs
}

func serveEcho(ln net.Listener, done chan<- struct{}) {
	var handlers sync.WaitGroup
	defer func() {
		handlers.Wait()
		close(done)
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		handlers.Add(1)
		go func(conn net.Conn) {
			defer handlers.Done()
			defer conn.Close()
			_, _ = io.Copy(conn, conn)
		}(conn)
	}
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not listen on %s", addr)
}

func waitForForwarding(t *testing.T, addr string) net.Conn {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			err = echoRoundTrip(conn, []byte("connected"), 200*time.Millisecond)
			if err == nil {
				return conn
			}
			_ = conn.Close()
		}
		lastErr = err
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("forwarding did not become ready: %v", lastErr)
	return nil
}

func echoRoundTrip(conn net.Conn, payload []byte, timeout time.Duration) error {
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	if _, err := conn.Write(payload); err != nil {
		return err
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		return err
	}
	if !bytes.Equal(got, payload) {
		return fmt.Errorf("echo mismatch: got %q, want %q", got, payload)
	}
	return nil
}

func waitForRunExit(t *testing.T, name string, done <-chan error, want error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Errorf("%s returned %v, want %v", name, err, want)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("%s did not stop", name)
	}
}
