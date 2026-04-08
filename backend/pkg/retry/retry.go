package retry

import (
	"context"
	"math"
	"time"
)

// Do runs fn until success or attempts exhausted with exponential backoff.
func Do(ctx context.Context, attempts int, base time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		d := time.Duration(float64(base) * math.Pow(2, float64(i)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	return err
}
