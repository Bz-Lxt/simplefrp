package metrics

import "sync/atomic"

type Counters struct {
	up       atomic.Int64
	down     atomic.Int64
	visitors atomic.Uint64
	reject   atomic.Uint64
	streams  atomic.Int64
	auths    atomic.Uint64
	authFail atomic.Uint64
}

func New() *Counters { return &Counters{} }

func (c *Counters) AddUp(n int64)       { c.up.Add(n) }
func (c *Counters) AddDown(n int64)     { c.down.Add(n) }
func (c *Counters) Visitor()            { c.visitors.Add(1) }
func (c *Counters) Reject()             { c.reject.Add(1) }
func (c *Counters) StreamOpen()         { c.streams.Add(1) }
func (c *Counters) StreamClose()        { c.streams.Add(-1) }
func (c *Counters) AuthOK()             { c.auths.Add(1) }
func (c *Counters) AuthFail()           { c.authFail.Add(1) }

type Snapshot struct {
	BytesUp          int64  `json:"bytes_up"`
	BytesDown        int64  `json:"bytes_down"`
	ActiveStreams    int64  `json:"active_streams"`
	VisitorsTotal    uint64 `json:"visitors_total"`
	VisitorsRejected uint64 `json:"visitors_rejected"`
	AuthOK           uint64 `json:"auth_ok"`
	AuthFail         uint64 `json:"auth_fail"`
}

func (c *Counters) Snapshot() Snapshot {
	return Snapshot{
		BytesUp:          c.up.Load(),
		BytesDown:        c.down.Load(),
		ActiveStreams:    c.streams.Load(),
		VisitorsTotal:    c.visitors.Load(),
		VisitorsRejected: c.reject.Load(),
		AuthOK:           c.auths.Load(),
		AuthFail:         c.authFail.Load(),
	}
}

func (c *Counters) UpPtr() *atomic.Int64   { return &c.up }
func (c *Counters) DownPtr() *atomic.Int64 { return &c.down }
func (c *Counters) Streams() int64         { return c.streams.Load() }
func (c *Counters) Visitors() uint64       { return c.visitors.Load() }
func (c *Counters) Rejected() uint64       { return c.reject.Load() }
