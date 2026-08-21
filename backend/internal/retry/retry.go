package retry

import (
	"context"
	"time"

	"simplefrp/internal/backoff"
)

func Do(ctx context.Context, attempts int, p backoff.Policy, fn func() error) error {
	if attempts <= 0 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = fn()
		if last == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		delay := p.Delay(i)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}
