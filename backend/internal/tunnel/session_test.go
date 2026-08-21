package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestYamuxMultiplexStreams(t *testing.T) {
	a, b := net.Pipe()
	opening, err := NewOpening(a)
	if err != nil {
		t.Fatal(err)
	}
	accepting, err := NewAccepting(b)
	if err != nil {
		t.Fatal(err)
	}
	defer opening.Close()
	defer accepting.Close()

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)

	go func() {
		for i := 0; i < n; i++ {
			stream, err := accepting.AcceptStream()
			if err != nil {
				errCh <- err
				return
			}
			go func(s io.ReadWriteCloser) {
				defer wg.Done()
				buf := make([]byte, 64)
				nr, rerr := io.ReadFull(s, buf[:5])
				if rerr != nil {
					errCh <- rerr
					_ = s.Close()
					return
				}
				if _, werr := s.Write(buf[:nr]); werr != nil {
					errCh <- werr
				}
				_ = s.Close()
			}(stream)
		}
	}()

	for i := 0; i < n; i++ {
		s, err := opening.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		payload := []byte("ping!")
		if _, err := s.Write(payload); err != nil {
			t.Fatal(err)
		}
		got := make([]byte, 5)
		if _, err := io.ReadFull(s, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("stream %d mixed: %q", i, got)
		}
		_ = s.Close()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting echo workers")
	}
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}
