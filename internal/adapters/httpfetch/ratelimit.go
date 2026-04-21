// Package httpfetch provides the HTTP fetcher and its primitives (rate, retry, robots).
package httpfetch

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// Limiter enforces a global minimum average interval between calls with ±jitter randomness.
// Safe for concurrent use.
type Limiter struct {
	baseInterval time.Duration
	jitter       time.Duration
	rng          *rand.Rand
	mu           sync.Mutex
	lastRelease  time.Time
	pauseUntil   time.Time
}

// NewLimiter builds a Limiter where the average interval between calls is roughly
// 1/reqPerSec, with random variation in [-jitter, +jitter] around the base interval.
// If reqPerSec <= 0, behaves as unlimited (Wait returns immediately ignoring jitter).
func NewLimiter(reqPerSec float64, jitter time.Duration) *Limiter {
	var base time.Duration
	if reqPerSec > 0 {
		base = time.Duration(float64(time.Second) / reqPerSec)
	}
	return &Limiter{
		baseInterval: base,
		jitter:       jitter,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewLimiterWithRng creates a Limiter using the provided *rand.Rand for jitter generation.
// This is primarily intended for tests that require deterministic behavior.
func NewLimiterWithRng(reqPerSec float64, jitter time.Duration, rng *rand.Rand) *Limiter {
	l := NewLimiter(reqPerSec, jitter)
	l.rng = rng
	return l
}

// Wait blocks until a slot is available. Honors ctx cancellation.
// If the global pause window (set by PauseUntil) has not elapsed, waits until then first.
func (l *Limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	var target time.Time
	if l.baseInterval <= 0 {
		// unlimited
		target = time.Now()
	} else {
		// jitter in [-jitter, +jitter]
		jitterNs := int64(0)
		if l.jitter > 0 {
			jitterNs = l.rng.Int63n(int64(2*l.jitter+1)) - int64(l.jitter)
		}
		target = l.lastRelease.Add(l.baseInterval + time.Duration(jitterNs))
		now := time.Now()
		if target.Before(now) {
			target = now
		}
	}
	if l.pauseUntil.After(target) {
		target = l.pauseUntil
	}
	l.lastRelease = target
	l.mu.Unlock()

	sleep := time.Until(target)
	if sleep <= 0 {
		return nil
	}
	t := time.NewTimer(sleep)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PauseUntil sets a floor on the next release time. Any Wait call will not return
// before t. Thread-safe. No-op if t is in the past.
func (l *Limiter) PauseUntil(t time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if t.After(l.pauseUntil) {
		l.pauseUntil = t
	}
}

// PausedUntil returns the current pause floor. Zero time means no pause active.
// Exposed for testing and observability.
func (l *Limiter) PausedUntil() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.pauseUntil
}
