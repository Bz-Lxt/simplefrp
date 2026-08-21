package forward

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

type Counters struct {
	Up   atomic.Int64
	Down atomic.Int64
}

func Pipe(ctx context.Context, a, b net.Conn, bytesAtoB, bytesBtoA *atomic.Int64) error {
	bufp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufp)

	errCh := make(chan error, 2)

	go func() {
		n, err := io.CopyBuffer(a, b, *bufp)
		if bytesBtoA != nil {
			bytesBtoA.Add(n)
		}
		_ = a.Close()
		errCh <- err
	}()
	go func() {
		n, err := io.CopyBuffer(b, a, *bufp)
		if bytesAtoB != nil {
			bytesAtoB.Add(n)
		}
		_ = b.Close()
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		_ = a.Close()
		_ = b.Close()
		<-errCh
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		_ = a.Close()
		_ = b.Close()
		<-errCh
		if err == io.EOF {
			return nil
		}
		return err
	}
}
