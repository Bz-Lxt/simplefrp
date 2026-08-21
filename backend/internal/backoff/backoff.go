package backoff

import (
	"math/rand"
	"time"
)

type Policy struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
	Jitter  float64
}

func Default() Policy {
	return Policy{
		Initial: 200 * time.Millisecond,
		Max:     8 * time.Second,
		Factor:  2,
		Jitter:  0.2,
	}
}

func (p Policy) Delay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	d := p.Initial
	if d <= 0 {
		d = 100 * time.Millisecond
	}
	factor := p.Factor
	if factor < 1 {
		factor = 2
	}
	for i := 0; i < attempt; i++ {
		next := time.Duration(float64(d) * factor)
		if p.Max > 0 && next > p.Max {
			d = p.Max
			break
		}
		d = next
	}
	if p.Jitter > 0 {
		span := float64(d) * p.Jitter
		d += time.Duration((rand.Float64()*2 - 1) * span)
		if d < 0 {
			d = p.Initial
		}
	}
	if p.Max > 0 && d > p.Max {
		return p.Max
	}
	return d
}
