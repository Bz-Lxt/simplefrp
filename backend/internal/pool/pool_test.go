package pool

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func startEcho(t *testing.T) (addr string, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(c)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close(); <-done }
}

func TestPoolGetPutReuseAndCap(t *testing.T) {
	addr, stop := startEcho(t)
	defer stop()
	factory := func(ctx context.Context) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "tcp", addr)
	}
	p := New(factory, Config{MaxIdle: 2, MaxActive: 2, IdleTTL: time.Second, Wait: 200 * time.Millisecond, NoWarm: true})
	defer p.Close()

	ctx := context.Background()
	c1, err := p.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := p.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Get(ctx)
	if err != ErrTimeout && err != ErrRejected {
		if err == nil {
			t.Fatal("expected rejection at cap")
		}
		if err != ErrTimeout {
			t.Fatalf("got %v", err)
		}
	}
	st := p.Stats()
	if st.Rejected == 0 {
		t.Fatal("expected rejected counter")
	}
	p.Put(c1)
	p.Discard(c2)
	c3, err := p.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.Discard(c3)
	if p.Stats().Reused == 0 {
		t.Fatal("expected reuse from idle")
	}
}

func TestPoolIdleTTLExpires(t *testing.T) {
	addr, stop := startEcho(t)
	defer stop()
	factory := func(ctx context.Context) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "tcp", addr)
	}
	p := New(factory, Config{MaxIdle: 1, MaxActive: 4, IdleTTL: 50 * time.Millisecond, Wait: time.Second, NoWarm: true})
	defer p.Close()
	ctx := context.Background()
	c, err := p.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.Put(c)
	time.Sleep(80 * time.Millisecond)
	c2, err := p.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.Discard(c2)
	if p.Stats().Expired == 0 {
		t.Fatal("expected expired idle conn")
	}
}
