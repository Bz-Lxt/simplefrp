package natmap

import (
	"fmt"
	"sync"
	"time"
)

type Endpoint struct {
	Host string
	Port int
}

func (e Endpoint) String() string {
	if e.Host == "" && e.Port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

type Binding struct {
	Public  Endpoint
	Private Endpoint
	Proto   string
	Until   time.Time
}

type Table struct {
	mu    sync.Mutex
	ttl   time.Duration
	fwd   map[string]Binding
	rev   map[string]string
	nowFn func() time.Time
}

func New(ttl time.Duration) *Table {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Table{
		ttl:   ttl,
		fwd:   make(map[string]Binding),
		rev:   make(map[string]string),
		nowFn: time.Now,
	}
}

func key(proto string, e Endpoint) string {
	return proto + "|" + e.String()
}

func (t *Table) Map(public, private Endpoint, proto string) Binding {
	if proto == "" {
		proto = "tcp"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	t.gcLocked(now)
	b := Binding{Public: public, Private: private, Proto: proto, Until: now.Add(t.ttl)}
	pk := key(proto, public)
	rk := key(proto, private)
	if old, ok := t.fwd[pk]; ok {
		delete(t.rev, key(old.Proto, old.Private))
	}
	t.fwd[pk] = b
	t.rev[rk] = pk
	return b
}

func (t *Table) Lookup(public Endpoint, proto string) (Binding, bool) {
	if proto == "" {
		proto = "tcp"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	b, ok := t.fwd[key(proto, public)]
	if !ok || !b.Until.After(now) {
		if ok {
			t.dropLocked(b)
		}
		return Binding{}, false
	}
	b.Until = now.Add(t.ttl)
	t.fwd[key(proto, public)] = b
	return b, true
}

func (t *Table) Reverse(private Endpoint, proto string) (Binding, bool) {
	if proto == "" {
		proto = "tcp"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	pk, ok := t.rev[key(proto, private)]
	if !ok {
		return Binding{}, false
	}
	b, ok := t.fwd[pk]
	now := t.nowFn()
	if !ok || !b.Until.After(now) {
		if ok {
			t.dropLocked(b)
		}
		return Binding{}, false
	}
	return b, true
}

func (t *Table) Unmap(public Endpoint, proto string) bool {
	if proto == "" {
		proto = "tcp"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.fwd[key(proto, public)]
	if !ok {
		return false
	}
	t.dropLocked(b)
	return true
}

func (t *Table) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gcLocked(t.nowFn())
	return len(t.fwd)
}

func (t *Table) dropLocked(b Binding) {
	delete(t.fwd, key(b.Proto, b.Public))
	delete(t.rev, key(b.Proto, b.Private))
}

func (t *Table) gcLocked(now time.Time) {
	for _, b := range t.fwd {
		if !b.Until.After(now) {
			t.dropLocked(b)
		}
	}
}
