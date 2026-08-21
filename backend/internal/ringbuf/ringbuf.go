package ringbuf

import (
	"errors"
	"io"
)

var ErrFull = errors.New("ring buffer full")

type Buffer struct {
	buf  []byte
	r    int
	w    int
	size int
	full bool
}

func New(n int) *Buffer {
	if n <= 0 {
		n = 4096
	}
	return &Buffer{buf: make([]byte, n)}
}

func (b *Buffer) Cap() int { return len(b.buf) }

func (b *Buffer) Len() int {
	if b.full {
		return len(b.buf)
	}
	if b.w >= b.r {
		return b.w - b.r
	}
	return len(b.buf) - b.r + b.w
}

func (b *Buffer) Free() int { return len(b.buf) - b.Len() }

func (b *Buffer) Empty() bool { return !b.full && b.r == b.w }

func (b *Buffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.Free() == 0 {
		return 0, ErrFull
	}
	n := 0
	for n < len(p) && b.Free() > 0 {
		b.buf[b.w] = p[n]
		b.w++
		if b.w == len(b.buf) {
			b.w = 0
		}
		n++
		if b.w == b.r {
			b.full = true
			break
		}
	}
	b.size = b.Len()
	if n < len(p) {
		return n, ErrFull
	}
	return n, nil
}

func (b *Buffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.Empty() {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && !b.Empty() {
		p[n] = b.buf[b.r]
		b.r++
		if b.r == len(b.buf) {
			b.r = 0
		}
		n++
		b.full = false
		if b.r == b.w {
			break
		}
	}
	b.size = b.Len()
	return n, nil
}

func (b *Buffer) Peek(n int) []byte {
	if n <= 0 || b.Empty() {
		return nil
	}
	if n > b.Len() {
		n = b.Len()
	}
	out := make([]byte, n)
	r := b.r
	for i := 0; i < n; i++ {
		out[i] = b.buf[r]
		r++
		if r == len(b.buf) {
			r = 0
		}
	}
	return out
}

func (b *Buffer) Discard(n int) int {
	if n <= 0 || b.Empty() {
		return 0
	}
	if n > b.Len() {
		n = b.Len()
	}
	b.r = (b.r + n) % len(b.buf)
	b.full = false
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	b.size = b.Len()
	return n
}

func (b *Buffer) Reset() {
	b.r = 0
	b.w = 0
	b.full = false
	b.size = 0
}

func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	var total int64
	tmp := make([]byte, 512)
	for !b.Empty() {
		n, err := b.Read(tmp)
		if n > 0 {
			wn, werr := w.Write(tmp[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	var total int64
	tmp := make([]byte, 512)
	for b.Free() > 0 {
		n, err := r.Read(tmp[:min(len(tmp), b.Free())])
		if n > 0 {
			wn, werr := b.Write(tmp[:n])
			total += int64(wn)
			if werr != nil && !errors.Is(werr, ErrFull) {
				return total, werr
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
