package vocab

import (
	"context"
	"sync"
	"time"
)

type slotReservation struct {
	id uint64
	at time.Time
}

// PacingGuard enforces Vocabulary.com's anti-cheat timing thresholds across concurrent callers:
//  1. Classic Challenge/Practice (v=3): >= 950ms before first try, >= 475ms before subsequent tries.
//  2. Quiz (v=3): >= 1550ms before answer submission.
//  3. VocabTrainer (v=4): Never complete >= 10 questions within a rolling 30,000ms window
//     (which triggers the 5-minute "trainer-cooldown" lockout).
//
// PacingGuard is safe for concurrent use by multiple goroutines and multiple TrainerSessions.
type PacingGuard struct {
	mu               sync.Mutex
	enabled          bool
	clock            Clock
	nextResID        uint64
	answerTimestamps []time.Time
	reservations     []slotReservation
}

func newPacingGuard(enabled bool, clock Clock) *PacingGuard {
	if clock == nil {
		clock = realClock{}
	}
	return &PacingGuard{
		enabled:          enabled,
		clock:            clock,
		answerTimestamps: make([]time.Time, 0, 16),
		reservations:     make([]slotReservation, 0, 16),
	}
}

// Enabled reports whether the pacing guard is active.
func (p *PacingGuard) Enabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// SetEnabled enables or disables the pacing guard.
func (p *PacingGuard) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// SetClock sets the Clock used by the pacing guard.
func (p *PacingGuard) SetClock(clock Clock) {
	if clock == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clock = clock
}

func (p *PacingGuard) pruneLocked(now time.Time) {
	cutoff := now.Add(-TrainerRollingWindow)
	filtered := p.answerTimestamps[:0]
	for _, ts := range p.answerTimestamps {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	p.answerTimestamps = filtered

	filteredRes := p.reservations[:0]
	for _, r := range p.reservations {
		if r.at.After(cutoff) {
			filteredRes = append(filteredRes, r)
		}
	}
	p.reservations = filteredRes
}

func (p *PacingGuard) earliestActiveTimestampLocked() (time.Time, bool) {
	var earliest time.Time
	found := false
	for _, ts := range p.answerTimestamps {
		if !found || ts.Before(earliest) {
			earliest = ts
			found = true
		}
	}
	for _, r := range p.reservations {
		if !found || r.at.Before(earliest) {
			earliest = r.at
			found = true
		}
	}
	return earliest, found
}

// ReserveTrainerSlot atomically waits until a slot is available in the 30.5s rolling window
// and reserves it prior to dispatching a network request, preventing concurrent goroutines
// from bypassing the 9-answers-per-30s limit.
//
// Callers must invoke `commit()` when the answer submission succeeds, or `release()` if the
// request fails before reaching the server.
func (p *PacingGuard) ReserveTrainerSlot(ctx context.Context) (commit func(), release func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	for {
		p.mu.Lock()
		clock := p.clock
		now := clock.Now()
		if !p.enabled {
			p.mu.Unlock()
			var once sync.Once
			return func() {
				once.Do(func() {
					p.RecordAnswer(clock.Now())
				})
			}, func() {}, nil
		}

		p.pruneLocked(now)
		activeCount := len(p.answerTimestamps) + len(p.reservations)
		if activeCount < TrainerMaxAnswersInWindow {
			p.nextResID++
			resID := p.nextResID
			p.reservations = append(p.reservations, slotReservation{id: resID, at: now})
			p.mu.Unlock()

			var once sync.Once
			commitFn := func() {
				once.Do(func() {
					p.mu.Lock()
					defer p.mu.Unlock()
					p.removeReservationLocked(resID)
					p.answerTimestamps = append(p.answerTimestamps, p.clock.Now())
				})
			}
			releaseFn := func() {
				once.Do(func() {
					p.mu.Lock()
					defer p.mu.Unlock()
					p.removeReservationLocked(resID)
				})
			}
			return commitFn, releaseFn, nil
		}

		earliest, ok := p.earliestActiveTimestampLocked()
		p.mu.Unlock()
		if !ok {
			continue
		}
		waitDur := earliest.Add(TrainerRollingWindow).Sub(now)
		if waitDur <= 0 {
			continue
		}
		if err := clock.Sleep(ctx, waitDur); err != nil {
			return nil, nil, err
		}
	}
}

func (p *PacingGuard) removeReservationLocked(id uint64) {
	for i, r := range p.reservations {
		if r.id == id {
			p.reservations = append(p.reservations[:i], p.reservations[i+1:]...)
			return
		}
	}
}

// CheckAndWaitTrainerWindow ensures fewer than 9 answers are active in the last 30.5s.
// Prefer ReserveTrainerSlot when guarding a network request under concurrency.
func (p *PacingGuard) CheckAndWaitTrainerWindow(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for {
		p.mu.Lock()
		if !p.enabled {
			p.mu.Unlock()
			return nil
		}
		clock := p.clock
		now := clock.Now()
		p.pruneLocked(now)
		if len(p.answerTimestamps)+len(p.reservations) < TrainerMaxAnswersInWindow {
			p.mu.Unlock()
			return nil
		}
		earliest, ok := p.earliestActiveTimestampLocked()
		p.mu.Unlock()
		if !ok {
			return nil
		}
		waitDur := earliest.Add(TrainerRollingWindow).Sub(now)
		if waitDur <= 0 {
			continue
		}
		if err := clock.Sleep(ctx, waitDur); err != nil {
			return err
		}
	}
}

// RecordAnswer records a completed question timestamp for the rolling 30s window.
func (p *PacingGuard) RecordAnswer(ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ts.IsZero() {
		ts = p.clock.Now()
	}
	p.pruneLocked(ts)
	p.answerTimestamps = append(p.answerTimestamps, ts)
}

// EnsureMinResponseTime guarantees that elapsed response time `rt` since `shownAt` is at least `minMS` milliseconds,
// sleeping via the configured Clock if necessary when PacingGuard is enabled.
func (p *PacingGuard) EnsureMinResponseTime(ctx context.Context, shownAt time.Time, minMS int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	p.mu.Lock()
	enabled := p.enabled
	clock := p.clock
	p.mu.Unlock()

	if shownAt.IsZero() {
		if enabled && minMS > 0 {
			if err := clock.Sleep(ctx, time.Duration(minMS)*time.Millisecond); err != nil {
				return 0, err
			}
		}
		return minMS, nil
	}

	now := clock.Now()
	elapsed := now.Sub(shownAt).Milliseconds()
	if elapsed >= minMS {
		return elapsed, nil
	}
	if !enabled {
		return minMS, nil
	}
	waitDur := time.Duration(minMS)*time.Millisecond - now.Sub(shownAt)
	if waitDur > 0 {
		if err := clock.Sleep(ctx, waitDur); err != nil {
			return clock.Now().Sub(shownAt).Milliseconds(), err
		}
	}
	finalElapsed := clock.Now().Sub(shownAt).Milliseconds()
	if finalElapsed < minMS {
		finalElapsed = minMS
	}
	return finalElapsed, nil
}
