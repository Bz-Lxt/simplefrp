package config

import (
	"testing"
	"time"
)

func TestNextBackoffCapsAtTwoSeconds(t *testing.T) {
	d := time.Duration(0)
	seen := map[time.Duration]bool{}
	for i := 0; i < 8; i++ {
		d = NextBackoff(d)
		if d > 2*time.Second {
			t.Fatalf("backoff %s exceeds 2s", d)
		}
		seen[d] = true
	}
	if !seen[500*time.Millisecond] || !seen[2*time.Second] {
		t.Fatalf("unexpected sequence, last=%s", d)
	}
}
