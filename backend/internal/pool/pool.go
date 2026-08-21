package pool

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrClosed   = errors.New("pool: closed")
	ErrRejected = errors.New("pool: max active connections reached")
	ErrTimeout  = errors.New("pool: wait timeout")
)

type Factory func(ctx context.Context) (net.Conn, error)

type Config struct {
	MaxIdle   int
	MaxActive int
	IdleTTL   time.Duration
	Wait      time.Duration
	NoWarm    bool
}

func (c Config) withDefaults() Config {
	if c.MaxIdle <= 0 {
		c.MaxIdle = 4
	}
	if c.MaxActive <= 0 {
		c.MaxActive = 64
	}
	if c.IdleTTL <= 0 {
		c.IdleTTL = 30 * time.Second
	}
	if c.Wait <= 0 {
		c.Wait = 5 * time.Second
	}
	return c
}

type Stats struct {
	Idle     int    `json:"idle"`
	Active   int    `json:"active"`
	Dialed   uint64 `json:"dialed"`
	Reused   uint64 `json:"reused"`
	Rejected uint64 `json:"rejected"`
	Expired  uint64 `json:"expired"`
}

type idleConn struct {
	c       net.Conn
	addedAt time.Time
}

type Pool struct {
	factory Factory
	cfg     Config

	mu     sync.Mutex
	cond   *sync.Cond
	idle   []idleConn
	active int
	closed bool

	dialed   atomic.Uint64
	reused   atomic.Uint64
	rejected atomic.Uint64
	expired  atomic.Uint64
}

func New(factory Factory, cfg Config) *Pool {
	p := &Pool{factory: factory, cfg: cfg.withDefaults()}
	p.cond = sync.NewCond(&p.mu)
	go p.reaper()
	if !p.cfg.NoWarm {
		go p.warmer()
	}
	return p
}

func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	deadline := time.Now().Add(p.cfg.Wait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		if p.closed {
			return nil, ErrClosed
		}
		if p.active < p.cfg.MaxActive {
			if c, ok := p.popIdle(); ok {
				if !healthy(c) {
					_ = c.Close()
					p.expired.Add(1)
					p.cond.Signal()
					continue
				}
				p.active++
				p.reused.Add(1)
				return c, nil
			}
			p.active++
			p.mu.Unlock()
			c, err := p.factory(ctx)
			p.mu.Lock()
			if err != nil {
				p.active--
				p.cond.Signal()
				return nil, err
			}
			p.dialed.Add(1)
			return c, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			p.rejected.Add(1)
			return nil, ErrTimeout
		}
		timer := time.AfterFunc(remaining, func() {
			p.mu.Lock()
			p.cond.Broadcast()
			p.mu.Unlock()
		})
		p.cond.Wait()
		timer.Stop()
		select {
		case <-ctx.Done():
			p.rejected.Add(1)
			return nil, ctx.Err()
		default:
		}
	}
}

func (p *Pool) Put(c net.Conn) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.active--
	if p.active < 0 {
		p.active = 0
	}
	if p.closed {
		_ = c.Close()
		p.cond.Signal()
		return
	}
	if len(p.idle) >= p.cfg.MaxIdle {
		_ = c.Close()
		p.cond.Signal()
		return
	}
	p.idle = append(p.idle, idleConn{c: c, addedAt: time.Now()})
	p.cond.Signal()
}

func (p *Pool) Discard(c net.Conn) {
	if c != nil {
		_ = c.Close()
	}
	p.mu.Lock()
	p.active--
	if p.active < 0 {
		p.active = 0
	}
	p.cond.Signal()
	p.mu.Unlock()
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	for _, ic := range p.idle {
		_ = ic.c.Close()
	}
	p.idle = nil
	p.cond.Broadcast()
}

func (p *Pool) Stats() Stats {
	p.mu.Lock()
	idle := len(p.idle)
	active := p.active
	p.mu.Unlock()
	return Stats{
		Idle:     idle,
		Active:   active,
		Dialed:   p.dialed.Load(),
		Reused:   p.reused.Load(),
		Rejected: p.rejected.Load(),
		Expired:  p.expired.Load(),
	}
}

func (p *Pool) popIdle() (net.Conn, bool) {
	for len(p.idle) > 0 {
		ic := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		if time.Since(ic.addedAt) > p.cfg.IdleTTL || !healthy(ic.c) {
			_ = ic.c.Close()
			p.expired.Add(1)
			continue
		}
		return ic.c, true
	}
	return nil, false
}

func healthy(c net.Conn) bool {
	_ = c.SetReadDeadline(time.Now().Add(time.Millisecond))
	one := make([]byte, 1)
	_, err := c.Read(one)
	_ = c.SetReadDeadline(time.Time{})
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return false
}

func (p *Pool) reaper() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}
		kept := p.idle[:0]
		for _, ic := range p.idle {
			if time.Since(ic.addedAt) > p.cfg.IdleTTL {
				_ = ic.c.Close()
				p.expired.Add(1)
				continue
			}
			kept = append(kept, ic)
		}
		p.idle = kept
		p.mu.Unlock()
	}
}

func (p *Pool) warmer() {
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		p.mu.Lock()
		need := p.cfg.MaxIdle - len(p.idle)
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		if need <= 0 {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		c, err := p.factory(ctx)
		cancel()
		if err != nil {
			continue
		}
		p.dialed.Add(1)
		p.mu.Lock()
		if p.closed || len(p.idle) >= p.cfg.MaxIdle {
			p.mu.Unlock()
			_ = c.Close()
			continue
		}
		p.idle = append(p.idle, idleConn{c: c, addedAt: time.Now()})
		p.cond.Signal()
		p.mu.Unlock()
	}
}
