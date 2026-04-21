package httpfetch

import (
	"sync"
	"testing"
	"time"
)

// virtualClock is an injectable clock for deterministic circuit breaker tests.
type virtualClock struct {
	mu  sync.Mutex
	cur time.Time
}

func (c *virtualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cur
}

func (c *virtualClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.cur = c.cur.Add(d)
	c.mu.Unlock()
}

// newTestBreaker creates a breaker with N=3, pause=10s, K=2 and a virtual clock.
func newTestBreaker(t *testing.T) (*CircuitBreaker, *virtualClock) {
	t.Helper()
	clk := &virtualClock{cur: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	cfg := CircuitBreakerConfig{
		MaxConsecutive: 3,
		PauseDuration:  10 * time.Second,
		AbortThreshold: 2,
	}
	return NewCircuitBreaker(cfg, clk.Now), clk
}

// TestCircuit_ClosedBySuccess verifies that RecordSuccess 10× leaves the breaker Closed.
func TestCircuit_ClosedBySuccess(t *testing.T) {
	b, _ := newTestBreaker(t)
	for i := 0; i < 10; i++ {
		b.RecordSuccess()
	}
	if got := b.State(); got != CircuitClosed {
		t.Fatalf("expected Closed after 10 successes, got %v", got)
	}
}

// TestCircuit_OpenAfterN verifies that N failures trip the breaker to Open.
func TestCircuit_OpenAfterN(t *testing.T) {
	b, _ := newTestBreaker(t)
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if got := b.State(); got != CircuitOpen {
		t.Fatalf("expected Open after 3 failures, got %v", got)
	}
	wait, err := b.Allow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wait <= 0 {
		t.Fatalf("expected wait > 0 when Open, got %v", wait)
	}
}

// TestCircuit_HalfOpenAfterPause verifies transition to HalfOpen after pause elapses.
func TestCircuit_HalfOpenAfterPause(t *testing.T) {
	b, clk := newTestBreaker(t)
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if b.State() != CircuitOpen {
		t.Fatal("expected Open")
	}

	clk.Advance(10 * time.Second)

	wait, err := b.Allow()
	if err != nil {
		t.Fatalf("unexpected error after pause: %v", err)
	}
	if wait != 0 {
		t.Fatalf("expected wait == 0 after pause, got %v", wait)
	}
	if got := b.State(); got != CircuitHalfOpen {
		t.Fatalf("expected HalfOpen after pause, got %v", got)
	}
}

// TestCircuit_HalfOpenSuccessCloses verifies that a success in HalfOpen closes the breaker.
func TestCircuit_HalfOpenSuccessCloses(t *testing.T) {
	b, clk := newTestBreaker(t)
	// Open the breaker.
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	// Advance past pause to enter HalfOpen.
	clk.Advance(10 * time.Second)
	b.Allow()

	if b.State() != CircuitHalfOpen {
		t.Fatal("expected HalfOpen")
	}

	b.RecordSuccess()
	if got := b.State(); got != CircuitClosed {
		t.Fatalf("expected Closed after HalfOpen success, got %v", got)
	}

	// Counters should be reset: next failures should not immediately open (need N=3).
	b.RecordFailure()
	b.RecordFailure()
	if got := b.State(); got != CircuitClosed {
		t.Fatalf("expected Closed after 2 failures (N=3), got %v", got)
	}
}

// TestCircuit_HalfOpenFailuresAbort verifies K failures in HalfOpen cause Aborted.
func TestCircuit_HalfOpenFailuresAbort(t *testing.T) {
	b, clk := newTestBreaker(t)
	// Open the breaker.
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	// Advance past pause to enter HalfOpen.
	clk.Advance(10 * time.Second)
	b.Allow()

	if b.State() != CircuitHalfOpen {
		t.Fatal("expected HalfOpen")
	}

	// K=2 failures → Aborted.
	b.RecordFailure()
	b.RecordFailure()

	if got := b.State(); got != CircuitAborted {
		t.Fatalf("expected Aborted after K failures in HalfOpen, got %v", got)
	}

	wait, err := b.Allow()
	if err != ErrCircuitAborted {
		t.Fatalf("expected ErrCircuitAborted, got err=%v", err)
	}
	if wait != 0 {
		t.Fatalf("expected wait == 0 on Aborted, got %v", wait)
	}
}

// TestCircuit_AbortedIsTerminal verifies Aborted state is terminal.
func TestCircuit_AbortedIsTerminal(t *testing.T) {
	b, clk := newTestBreaker(t)
	// Trip to Open.
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	clk.Advance(10 * time.Second)
	b.Allow() // → HalfOpen
	b.RecordFailure()
	b.RecordFailure() // → Aborted

	// RecordSuccess is a noop.
	b.RecordSuccess()
	if got := b.State(); got != CircuitAborted {
		t.Fatalf("expected Aborted after RecordSuccess noop, got %v", got)
	}

	// RecordFailure is a noop.
	b.RecordFailure()
	if got := b.State(); got != CircuitAborted {
		t.Fatalf("expected Aborted after RecordFailure noop, got %v", got)
	}

	// Allow keeps returning ErrCircuitAborted.
	_, err := b.Allow()
	if err != ErrCircuitAborted {
		t.Fatalf("expected ErrCircuitAborted, got %v", err)
	}
	_, err = b.Allow()
	if err != ErrCircuitAborted {
		t.Fatalf("expected ErrCircuitAborted on second Allow, got %v", err)
	}
}

// TestCircuit_OpenStateNoopsOnRecordSuccessAndFailure verifies that RecordSuccess/Failure
// in Open state are noops (transitions happen only via Allow).
func TestCircuit_OpenStateNoopsOnRecordSuccessAndFailure(t *testing.T) {
	b, clk := newTestBreaker(t)
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if b.State() != CircuitOpen {
		t.Fatal("expected Open")
	}

	// RecordSuccess in Open should be noop.
	b.RecordSuccess()
	if b.State() != CircuitOpen {
		t.Fatalf("expected Open after RecordSuccess noop, got %v", b.State())
	}

	// RecordFailure in Open should be noop.
	b.RecordFailure()
	if b.State() != CircuitOpen {
		t.Fatalf("expected Open after RecordFailure noop, got %v", b.State())
	}

	// Allow in Closed state (sanity).
	b2, _ := newTestBreaker(t)
	_ = clk
	wait, err := b2.Allow()
	if err != nil || wait != 0 {
		t.Errorf("expected (0, nil) from Closed Allow, got (%v, %v)", wait, err)
	}
}

// TestCircuit_NilNowUsesTimeNow verifies that passing nil as the now function falls back to time.Now.
func TestCircuit_NilNowUsesTimeNow(t *testing.T) {
	cfg := CircuitBreakerConfig{MaxConsecutive: 2, PauseDuration: time.Second, AbortThreshold: 1}
	b := NewCircuitBreaker(cfg, nil)
	if b.now == nil {
		t.Fatal("expected now to be set when nil is passed")
	}
	// Verify it works by doing basic operations.
	b.RecordSuccess()
	if b.State() != CircuitClosed {
		t.Fatal("expected Closed")
	}
}

// TestCircuit_Defaults verifies CircuitBreakerConfig{}.WithDefaults() yields expected defaults.
func TestCircuit_Defaults(t *testing.T) {
	cfg := CircuitBreakerConfig{}.WithDefaults()

	if cfg.MaxConsecutive != 5 {
		t.Errorf("MaxConsecutive: want 5, got %d", cfg.MaxConsecutive)
	}
	if cfg.PauseDuration != 10*time.Minute {
		t.Errorf("PauseDuration: want 10m, got %v", cfg.PauseDuration)
	}
	if cfg.AbortThreshold != 3 {
		t.Errorf("AbortThreshold: want 3, got %d", cfg.AbortThreshold)
	}
}
