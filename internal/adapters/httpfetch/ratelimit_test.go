package httpfetch_test

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
)

// TestLimiter_Unlimited verifies that a zero-rate limiter returns immediately for all callers.
func TestLimiter_Unlimited(t *testing.T) {
	l := httpfetch.NewLimiter(0, 0)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait(%d) unexpected error: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("unlimited limiter took %v for 10 calls, want < 50ms", elapsed)
	}
}

// TestLimiter_RateRespected verifies that a 10 req/s limiter enforces ~100ms between calls.
func TestLimiter_RateRespected(t *testing.T) {
	// 10 req/s → 100ms base interval, no jitter for determinism
	l := httpfetch.NewLimiter(10, 0)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait(%d) unexpected error: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	// 4 full intervals of 100ms = 400ms minimum; allow generous slack for CI
	const minExpected = 350 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("5 calls elapsed %v, want >= %v (4 intervals * 100ms)", elapsed, minExpected)
	}
}

// TestLimiter_JitterAppliesToIntervals verifies (CA-13) that jitter produces variance
// in per-call intervals when using a seeded RNG for reproducibility.
func TestLimiter_JitterAppliesToIntervals(t *testing.T) {
	// 20 req/s = 50ms base, ±25ms jitter → intervals in [25ms, 75ms] theoretically
	// Use exported constructor; jitter variance is inherent by design.
	// We seed the internal rng via an unexported helper exposed for tests.
	rng := rand.New(rand.NewSource(42))
	l := httpfetch.NewLimiterWithRng(20, 25*time.Millisecond, rng)

	ctx := context.Background()
	const n = 30
	times := make([]time.Time, n)
	for i := 0; i < n; i++ {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait(%d) error: %v", i, err)
		}
		times[i] = time.Now()
	}

	intervals := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		intervals[i] = float64(times[i+1].Sub(times[i]))
	}

	// Compute mean
	var sum float64
	for _, v := range intervals {
		sum += v
	}
	mean := sum / float64(len(intervals))

	// Compute variance
	var variance float64
	for _, v := range intervals {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(intervals))

	t.Logf("mean interval: %v, variance: %.2e ns^2", time.Duration(mean), variance)

	if variance <= 0 {
		t.Error("expected variance > 0 with jitter enabled, got 0")
	}

	// All intervals should be within [0, 75ms + generous slack for CI]
	const maxInterval = 150 * time.Millisecond // 75ms theoretical + 75ms CI slack
	for i, iv := range intervals {
		d := time.Duration(iv)
		if d < 0 || d > maxInterval {
			t.Errorf("interval[%d] = %v, want in [0, %v]", i, d, maxInterval)
		}
	}
}

// TestLimiter_ContextCanceled verifies that a canceled context unblocks Wait with an error.
func TestLimiter_ContextCanceled(t *testing.T) {
	// 1 req/s → first call returns immediately, second waits ~1s
	l := httpfetch.NewLimiter(1, 0)
	ctx := context.Background()
	// Consume the first free slot
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("first Wait error: %v", err)
	}

	// Second call should block ~1s; cancel after 50ms
	cancelCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := l.Wait(cancelCtx)
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestLimiter_PauseUntilBlocksAllCallers verifies that PauseUntil delays all goroutines.
func TestLimiter_PauseUntilBlocksAllCallers(t *testing.T) {
	// 100 req/s → effectively no rate throttle between calls; pause window dominates
	l := httpfetch.NewLimiter(100, 0)
	pause := 200 * time.Millisecond
	l.PauseUntil(time.Now().Add(pause))

	var wg sync.WaitGroup
	results := make([]time.Duration, 3)
	start := time.Now()

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := l.Wait(context.Background()); err != nil {
				t.Errorf("goroutine %d Wait error: %v", idx, err)
				return
			}
			results[idx] = time.Since(start)
		}(i)
	}
	wg.Wait()

	for i, elapsed := range results {
		// Allow 50ms slack below the pause window
		const minElapsed = 150 * time.Millisecond
		if elapsed < minElapsed {
			t.Errorf("goroutine %d returned after %v, want >= %v (PauseUntil ~200ms)", i, elapsed, minElapsed)
		}
	}
}
