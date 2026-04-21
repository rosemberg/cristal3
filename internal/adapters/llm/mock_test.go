package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/bergmaia/site-research/internal/domain/ports"
)

func TestMock_FIFO(t *testing.T) {
	resp1 := &ports.GenerateResponse{Text: "first", TokensInput: 10, TokensOutput: 5, Provider: "mock", Model: "mock-model"}
	resp2 := &ports.GenerateResponse{Text: "second", TokensInput: 20, TokensOutput: 10, Provider: "mock", Model: "mock-model"}

	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: resp1},
			{Response: resp2},
		},
	})

	req1 := ports.GenerateRequest{User: "first question"}
	req2 := ports.GenerateRequest{User: "second question"}

	r1, err := mock.Generate(context.Background(), req1)
	if err != nil {
		t.Fatalf("Generate 1: %v", err)
	}
	if r1.Text != "first" {
		t.Errorf("response 1 text = %q, want %q", r1.Text, "first")
	}

	r2, err := mock.Generate(context.Background(), req2)
	if err != nil {
		t.Fatalf("Generate 2: %v", err)
	}
	if r2.Text != "second" {
		t.Errorf("response 2 text = %q, want %q", r2.Text, "second")
	}

	calls := mock.Calls()
	if len(calls) != 2 {
		t.Fatalf("Calls len = %d, want 2", len(calls))
	}
	if calls[0].User != "first question" {
		t.Errorf("calls[0].User = %q, want %q", calls[0].User, "first question")
	}
	if calls[1].User != "second question" {
		t.Errorf("calls[1].User = %q, want %q", calls[1].User, "second question")
	}
}

func TestMock_Exhausted(t *testing.T) {
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "only one"}},
		},
		Loop: false,
	})

	_, err := mock.Generate(context.Background(), ports.GenerateRequest{User: "1"})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on exhausted mock, got none")
		}
	}()
	_, _ = mock.Generate(context.Background(), ports.GenerateRequest{User: "2"})
}

func TestMock_Loop(t *testing.T) {
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "response-A"}},
			{Response: &ports.GenerateResponse{Text: "response-B"}},
		},
		Loop: true,
	})

	texts := make([]string, 6)
	for i := range 6 {
		r, err := mock.Generate(context.Background(), ports.GenerateRequest{User: "q"})
		if err != nil {
			t.Fatalf("Generate %d: %v", i, err)
		}
		texts[i] = r.Text
	}

	expected := []string{"response-A", "response-B", "response-A", "response-B", "response-A", "response-B"}
	for i, want := range expected {
		if texts[i] != want {
			t.Errorf("call %d: got %q, want %q", i, texts[i], want)
		}
	}
}

func TestMock_RecordsCalls(t *testing.T) {
	sentinelErr := errors.New("deliberate error")
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "ok"}},
			{Err: sentinelErr},
			{Response: &ports.GenerateResponse{Text: "ok again"}},
		},
	})

	reqs := []ports.GenerateRequest{
		{System: "sys1", User: "user1", MaxTokens: 128, Temperature: 0.2},
		{System: "sys2", User: "user2", MaxTokens: 256, Temperature: 0.0},
		{System: "sys3", User: "user3", MaxTokens: 64, Temperature: 0.5},
	}

	for i, req := range reqs {
		_, _ = mock.Generate(context.Background(), req)
		_ = i
	}

	calls := mock.Calls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}

	for i, want := range reqs {
		if calls[i].System != want.System {
			t.Errorf("calls[%d].System = %q, want %q", i, calls[i].System, want.System)
		}
		if calls[i].User != want.User {
			t.Errorf("calls[%d].User = %q, want %q", i, calls[i].User, want.User)
		}
		if calls[i].MaxTokens != want.MaxTokens {
			t.Errorf("calls[%d].MaxTokens = %d, want %d", i, calls[i].MaxTokens, want.MaxTokens)
		}
		if calls[i].Temperature != want.Temperature {
			t.Errorf("calls[%d].Temperature = %f, want %f", i, calls[i].Temperature, want.Temperature)
		}
	}
}

func TestMock_Reset(t *testing.T) {
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "first"}},
			{Response: &ports.GenerateResponse{Text: "second"}},
		},
	})

	r1, _ := mock.Generate(context.Background(), ports.GenerateRequest{User: "q1"})
	if r1.Text != "first" {
		t.Errorf("pre-reset first call = %q, want %q", r1.Text, "first")
	}

	mock.Reset()

	if calls := mock.Calls(); len(calls) != 0 {
		t.Errorf("after Reset, Calls() = %d, want 0", len(calls))
	}

	r2, _ := mock.Generate(context.Background(), ports.GenerateRequest{User: "q2"})
	if r2.Text != "first" {
		t.Errorf("post-reset first call = %q, want %q", r2.Text, "first")
	}
}

func TestMock_ErrorResponse(t *testing.T) {
	sentinelErr := errors.New("test provider error")
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Err: sentinelErr},
		},
	})

	_, err := mock.Generate(context.Background(), ports.GenerateRequest{User: "q"})
	if !errors.Is(err, sentinelErr) {
		t.Errorf("expected sentinelErr, got %v", err)
	}
}

func TestMock_DefaultNameModel(t *testing.T) {
	mock := NewMockProvider(MockOptions{
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "ok"}},
		},
	})
	if mock.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "mock")
	}
	if mock.Model() != "mock-model" {
		t.Errorf("Model() = %q, want %q", mock.Model(), "mock-model")
	}
}

func TestMock_CustomNameModel(t *testing.T) {
	mock := NewMockProvider(MockOptions{
		Name:  "custom-provider",
		Model: "custom-model-v1",
		Responses: []MockResponse{
			{Response: &ports.GenerateResponse{Text: "ok"}},
		},
	})
	if mock.Name() != "custom-provider" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "custom-provider")
	}
	if mock.Model() != "custom-model-v1" {
		t.Errorf("Model() = %q, want %q", mock.Model(), "custom-model-v1")
	}
}
