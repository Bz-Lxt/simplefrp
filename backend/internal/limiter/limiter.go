package limiter

import (
	"sync"
	"time"
)

type Bucket struct {
	mu       sync.Mutex
	rate     float64
	capacity float64
	tokens   float64
	last     time.Time
}

func New(rate, capacity float64) *Bucket {
	if rate <= 0 {
		rate = 1
	}
	if capacity <= 0 {
		capacity = rate
	}
	return &Bucket{rate: rate, capacity: capacity, tokens: capacity, last: time.Now()}
}

func (b *Bucket) Allow() bool {
	return b.AllowN(1)
}

func (b *Bucket) AllowN(n float64) bool {
	if n <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}

func (b *Bucket) Wait(n float64) time.Duration {
	if b.AllowN(n) {
		return 0
	}
	need := n
	b.mu.Lock()
	deficit := need - b.tokens
	rate := b.rate
	b.mu.Unlock()
	if rate <= 0 {
		return time.Second
	}
	return time.Duration(deficit / rate * float64(time.Second))
}
