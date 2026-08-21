package forward

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestPipeBidirectional(t *testing.T) {
	a, b := net.Pipe()
	c, d := net.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var up, down atomic.Int64
	done := make(chan error, 1)
	go func() { done <- Pipe(ctx, b, c, &up, &down) }()

	msg1 := []byte("hello-visitor")
	msg2 := []byte("from-intranet")
	if _, err := a.Write(msg1); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(msg1))
	if _, err := io.ReadFull(d, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg1) {
		t.Fatalf("got %q", got)
	}
	if _, err := d.Write(msg2); err != nil {
		t.Fatal(err)
	}
	got2 := make([]byte, len(msg2))
	if _, err := io.ReadFull(a, got2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got2, msg2) {
		t.Fatalf("got %q", got2)
	}
	_ = a.Close()
	_ = d.Close()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("pipe did not finish")
	}
	if up.Load() == 0 || down.Load() == 0 {
		t.Fatalf("counters up=%d down=%d", up.Load(), down.Load())
	}
}
