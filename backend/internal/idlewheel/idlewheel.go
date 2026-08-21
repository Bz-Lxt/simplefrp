package idlewheel

import (
	"sync"
	"time"
)

type Entry struct {
	ID      string
	Expires time.Time
}

type Wheel struct {
	mu      sync.Mutex
	idle    time.Duration
	items   map[string]time.Time
	nowFn   func() time.Time
}

func New(idle time.Duration) *Wheel {
	if idle <= 0 {
		idle = 60 * time.Second
	}
	return &Wheel{
		idle:  idle,
		items: make(map[string]time.Time),
		nowFn: time.Now,
	}
}

func (w *Wheel) Touch(id string) time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	exp := w.nowFn().Add(w.idle)
	w.items[id] = exp
	return exp
}

func (w *Wheel) Last(id string) (time.Time, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	exp, ok := w.items[id]
	if !ok {
		return time.Time{}, false
	}
	return exp.Add(-w.idle), true
}

func (w *Wheel) Expires(id string) (time.Time, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	exp, ok := w.items[id]
	return exp, ok
}

func (w *Wheel) Remove(id string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.items[id]; !ok {
		return false
	}
	delete(w.items, id)
	return true
}

func (w *Wheel) Sweep() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.nowFn()
	var dead []string
	for id, exp := range w.items {
		if !exp.After(now) {
			dead = append(dead, id)
			delete(w.items, id)
		}
	}
	return dead
}

func (w *Wheel) Idle(id string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	exp, ok := w.items[id]
	if !ok {
		return true
	}
	return !exp.After(w.nowFn())
}

func (w *Wheel) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.items)
}

func (w *Wheel) Snapshot() []Entry {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Entry, 0, len(w.items))
	for id, exp := range w.items {
		out = append(out, Entry{ID: id, Expires: exp})
	}
	return out
}

func (w *Wheel) UntilIdle(id string) time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	exp, ok := w.items[id]
	if !ok {
		return 0
	}
	d := exp.Sub(w.nowFn())
	if d < 0 {
		return 0
	}
	return d
}
