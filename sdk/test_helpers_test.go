package vocab

import (
	"context"
	"sync"
	"time"
)

// fakeClock is a thread-safe deterministic Clock that advances virtual time immediately on Sleep.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (fc *fakeClock) Now() time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.now
}

func (fc *fakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.now = fc.now.Add(d)
}

func (fc *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fc.mu.Lock()
	fc.sleeps = append(fc.sleeps, d)
	if d > 0 {
		fc.now = fc.now.Add(d)
	}
	fc.mu.Unlock()
	return nil
}

func (fc *fakeClock) Sleeps() []time.Duration {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	out := make([]time.Duration, len(fc.sleeps))
	copy(out, fc.sleeps)
	return out
}
