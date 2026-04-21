package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/bergmaia/site-research/internal/domain/ports"
)

// MockResponse is either a successful Response or an Err (exactly one non-zero).
type MockResponse struct {
	Response *ports.GenerateResponse
	Err      error
}

// MockProvider implements ports.LLMProvider for tests.
// Returns responses in FIFO order; panics if out of responses (unless Loop is true).
type MockProvider struct {
	mu        sync.Mutex
	name      string
	model     string
	responses []MockResponse
	calls     []ports.GenerateRequest
	loop      bool
	idx       int
}

// MockOptions builds a MockProvider.
type MockOptions struct {
	Name      string         // default "mock"
	Model     string         // default "mock-model"
	Responses []MockResponse // required, at least one
	Loop      bool           // if true, wrap around when exhausted
}

// NewMockProvider builds a mock. Panics if Responses is empty.
func NewMockProvider(opts MockOptions) *MockProvider {
	if len(opts.Responses) == 0 {
		panic("llm: MockProvider requires at least one response")
	}
	name := opts.Name
	if name == "" {
		name = "mock"
	}
	model := opts.Model
	if model == "" {
		model = "mock-model"
	}
	return &MockProvider{
		name:      name,
		model:     model,
		responses: opts.Responses,
		calls:     nil,
		loop:      opts.Loop,
		idx:       0,
	}
}

// Generate returns the next queued response/error and records the call.
func (m *MockProvider) Generate(ctx context.Context, req ports.GenerateRequest) (*ports.GenerateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if m.idx >= len(m.responses) {
		if !m.loop {
			panic(fmt.Sprintf("llm: MockProvider exhausted all %d responses", len(m.responses)))
		}
		m.idx = 0
	}

	r := m.responses[m.idx]
	m.idx++
	m.calls = append(m.calls, req)

	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

// Calls returns a snapshot of recorded requests.
func (m *MockProvider) Calls() []ports.GenerateRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make([]ports.GenerateRequest, len(m.calls))
	copy(snapshot, m.calls)
	return snapshot
}

// Reset clears recorded calls and restarts the response cursor.
func (m *MockProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.idx = 0
}

// Name returns the mock provider name.
func (m *MockProvider) Name() string { return m.name }

// Model returns the mock model name.
func (m *MockProvider) Model() string { return m.model }
