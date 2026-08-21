package window

import (
	"sync"
	"time"
)

type Sample struct {
	At    time.Time
	Bytes int64
}

type Sliding struct {
	mu      sync.Mutex
	span    time.Duration
	samples []Sample
}

func New(span time.Duration) *Sliding {
	if span <= 0 {
		span = time.Minute
	}
	return &Sliding{span: span}
}

func (s *Sliding) Add(n int64) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.samples = append(s.samples, Sample{At: now, Bytes: n})
	s.gcLocked(now)
}

func (s *Sliding) Rate() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.gcLocked(now)
	var total int64
	for _, sm := range s.samples {
		total += sm.Bytes
	}
	sec := s.span.Seconds()
	if sec <= 0 {
		return 0
	}
	return float64(total) / sec
}

func (s *Sliding) Sum() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now())
	var total int64
	for _, sm := range s.samples {
		total += sm.Bytes
	}
	return total
}

func (s *Sliding) gcLocked(now time.Time) {
	cut := now.Add(-s.span)
	i := 0
	for i < len(s.samples) && s.samples[i].At.Before(cut) {
		i++
	}
	if i > 0 {
		s.samples = append([]Sample(nil), s.samples[i:]...)
	}
}
