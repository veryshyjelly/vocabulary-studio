package vocab

import (
	"context"
	"testing"
	"time"
)

func TestPacingGuardDeterministicAndConcurrent(t *testing.T) {
	start := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	fc := newFakeClock(start)
	pg := newPacingGuard(true, fc)

	ctx := context.Background()

	// 1. Test EnsureMinResponseTime for first try (950ms) when only 200ms elapsed
	shownAt := fc.Now()
	fc.Advance(200 * time.Millisecond)
	rt, err := pg.EnsureMinResponseTime(ctx, shownAt, MinChallengeFirstTryMS)
	if err != nil {
		t.Fatalf("EnsureMinResponseTime: %v", err)
	}
	if rt != 950 {
		t.Errorf("rt = %d, want 950", rt)
	}
	sleeps := fc.Sleeps()
	if len(sleeps) != 1 || sleeps[0] != 750*time.Millisecond {
		t.Errorf("expected 1 sleep of 750ms, got %v", sleeps)
	}

	// 2. Test Rolling 30.5s Window (9 answers allowed in window; 10th must sleep until window expires)
	t0 := fc.Now()
	for i := 0; i < TrainerMaxAnswersInWindow; i++ {
		commit, _, err := pg.ReserveTrainerSlot(ctx)
		if err != nil {
			t.Fatalf("ReserveTrainerSlot %d: %v", i, err)
		}
		commit()
		fc.Advance(100 * time.Millisecond)
	}

	// Now 9 answers have been recorded starting at t0 + 0ms, and current time is t0 + 900ms.
	// The 10th reservation must wait until t0 + 30500ms (i.e. 29600ms sleep).
	before10th := fc.Now()
	commit10, _, err := pg.ReserveTrainerSlot(ctx)
	if err != nil {
		t.Fatalf("10th ReserveTrainerSlot: %v", err)
	}
	commit10()
	after10th := fc.Now()
	if waited := after10th.Sub(before10th); waited != 29600*time.Millisecond {
		t.Errorf("10th answer waited %v, want 29.6s (%v)", waited, 29600*time.Millisecond)
	}
	if after10th.Sub(t0) < TrainerRollingWindow {
		t.Errorf("10th answer completed at +%v, violating %v window", after10th.Sub(t0), TrainerRollingWindow)
	}

	// 3. Verify failed request release() frees the reserved slot immediately
	pg2 := newPacingGuard(true, fc)
	for i := 0; i < TrainerMaxAnswersInWindow-1; i++ {
		c, _, _ := pg2.ReserveTrainerSlot(ctx)
		c()
	}
	// Reserve the 9th slot and then release() it (simulating a failed network call)
	_, release9, err := pg2.ReserveTrainerSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	release9()

	// Next reservation should succeed with 0 wait because the 9th slot was released!
	tBefore := fc.Now()
	c9, _, err := pg2.ReserveTrainerSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c9()
	if fc.Now() != tBefore {
		t.Errorf("expected 0 wait after released slot, but clock advanced by %v", fc.Now().Sub(tBefore))
	}
}
