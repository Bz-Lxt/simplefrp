package lease

import (
	"sync"
	"time"
)

var beijing = time.FixedZone("CST", 8*3600)

type Record struct {
	ID      string
	Owner   string
	Until   time.Time
	Meta    string
	Created time.Time
}

type Table struct {
	mu      sync.Mutex
	ttl     time.Duration
	items   map[string]Record
	nowFn   func() time.Time
}

func New(ttl time.Duration) *Table {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Table{
		ttl:   ttl,
		items: make(map[string]Record),
		nowFn: func() time.Time { return time.Now().In(beijing) },
	}
}

func (t *Table) Grant(id, owner string) Record {
	return t.GrantTTL(id, owner, t.ttl, "")
}

func (t *Table) GrantTTL(id, owner string, ttl time.Duration, meta string) Record {
	if ttl <= 0 {
		ttl = t.ttl
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	t.gcLocked(now)
	rec := Record{ID: id, Owner: owner, Until: now.Add(ttl), Meta: meta, Created: now}
	t.items[id] = rec
	return rec
}

func (t *Table) Renew(id string) (Record, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	rec, ok := t.items[id]
	if !ok || !rec.Until.After(now) {
		return Record{}, false
	}
	rec.Until = now.Add(t.ttl)
	t.items[id] = rec
	return rec, true
}

func (t *Table) Get(id string) (Record, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	rec, ok := t.items[id]
	if !ok || !rec.Until.After(now) {
		if ok {
			delete(t.items, id)
		}
		return Record{}, false
	}
	return rec, true
}

func (t *Table) Owner(id string) string {
	rec, ok := t.Get(id)
	if !ok {
		return ""
	}
	return rec.Owner
}

func (t *Table) Release(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.items[id]; !ok {
		return false
	}
	delete(t.items, id)
	return true
}

func (t *Table) ReleaseOwner(owner string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for id, rec := range t.items {
		if rec.Owner == owner {
			delete(t.items, id)
			n++
		}
	}
	return n
}

func (t *Table) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gcLocked(t.nowFn())
	return len(t.items)
}

func (t *Table) Expired(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.items[id]
	if !ok {
		return true
	}
	return !rec.Until.After(t.nowFn())
}

func (t *Table) gcLocked(now time.Time) {
	for id, rec := range t.items {
		if !rec.Until.After(now) {
			delete(t.items, id)
		}
	}
}

func (t *Table) Snapshot() []Record {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.nowFn()
	t.gcLocked(now)
	out := make([]Record, 0, len(t.items))
	for _, rec := range t.items {
		out = append(out, rec)
	}
	return out
}
