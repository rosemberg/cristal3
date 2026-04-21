// Package httpfetch provides the HTTP fetcher and its primitives (rate, retry, robots).
package httpfetch

import (
	"errors"
	"sync"
	"time"
)

// CircuitState is the state of the breaker.
type CircuitState int

const (
	// CircuitClosed is the normal operating state.
	CircuitClosed CircuitState = iota
	// CircuitOpen means the breaker has tripped after N consecutive failures.
	CircuitOpen
	// CircuitHalfOpen is the recovery window after the open pause elapses.
	CircuitHalfOpen
	// CircuitAborted is the terminal state reached after K consecutive failures after resume.
	CircuitAborted
)

// ErrCircuitAborted indicates the breaker has reached its terminal state.
// Callers should stop the crawl and emit a report.
var ErrCircuitAborted = errors.New("httpfetch: circuit aborted — too many consecutive failures after resume")

// CircuitBreakerConfig mirrors crawler.circuit_breaker in the YAML:
//   - MaxConsecutive: N failures that open the breaker (default 5).
//   - PauseDuration:  how long the breaker stays open (default 10 minutes).
//   - AbortThreshold: K consecutive failures after resume that abort (default 3).
type CircuitBreakerConfig struct {
	MaxConsecutive int
	PauseDuration  time.Duration
	AbortThreshold int
}

// WithDefaults returns a copy with zero fields replaced by defaults.
func (c CircuitBreakerConfig) WithDefaults() CircuitBreakerConfig {
	if c.MaxConsecutive == 0 {
		c.MaxConsecutive = 5
	}
	if c.PauseDuration == 0 {
		c.PauseDuration = 10 * time.Minute
	}
	if c.AbortThreshold == 0 {
		c.AbortThreshold = 3
	}
	return c
}

// CircuitBreaker implements the breaker described in BRIEF RF-07.
// Safe for concurrent use. Now() is injectable for testability.
type CircuitBreaker struct {
	cfg            CircuitBreakerConfig
	now            func() time.Time
	mu             sync.Mutex
	state          CircuitState
	consecFail     int
	postResumeFail int
	openedAt       time.Time
}

// NewCircuitBreaker builds a breaker. If now is nil, time.Now is used.
func NewCircuitBreaker(cfg CircuitBreakerConfig, now func() time.Time) *CircuitBreaker {
	if now == nil {
		now = time.Now
	}
	return &CircuitBreaker{
		cfg:   cfg.WithDefaults(),
		now:   now,
		state: CircuitClosed,
	}
}

// State returns the current state (snapshot; may race with concurrent calls).
func (b *CircuitBreaker) State() CircuitState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Allow indicates whether the caller may proceed.
//
// If ErrCircuitAborted is returned, caller MUST stop.
// If wait > 0, the breaker is Open and the caller should sleep wait then retry
// (or schedule accordingly). After wait, state transitions to HalfOpen on the next Allow.
// If wait == 0 and err == nil, the caller proceeds and must report success/failure after.
func (b *CircuitBreaker) Allow() (wait time.Duration, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case CircuitClosed:
		return 0, nil

	case CircuitOpen:
		elapsed := b.now().Sub(b.openedAt)
		if elapsed >= b.cfg.PauseDuration {
			b.state = CircuitHalfOpen
			return 0, nil
		}
		return b.cfg.PauseDuration - elapsed, nil

	case CircuitHalfOpen:
		return 0, nil

	case CircuitAborted:
		return 0, ErrCircuitAborted
	}
	return 0, nil
}

// RecordSuccess resets counters and closes the breaker.
func (b *CircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == CircuitAborted {
		return
	}
	if b.state == CircuitOpen {
		// Success during Open is a noop — transitions happen only via Allow.
		return
	}
	b.state = CircuitClosed
	b.consecFail = 0
	b.postResumeFail = 0
}

// RecordFailure increments counters and advances state per the rules.
func (b *CircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == CircuitAborted {
		return
	}
	if b.state == CircuitOpen {
		// Failures during Open are noops — transitions happen only via Allow.
		return
	}

	switch b.state {
	case CircuitClosed:
		b.consecFail++
		if b.consecFail >= b.cfg.MaxConsecutive {
			b.state = CircuitOpen
			b.openedAt = b.now()
			b.postResumeFail = 0
		}

	case CircuitHalfOpen:
		b.postResumeFail++
		if b.postResumeFail >= b.cfg.AbortThreshold {
			b.state = CircuitAborted
		}
	}
}
