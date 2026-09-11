package model

import (
	"context"
	"fmt"
	"time"

	"github.com/chenlong/keystone/internal/tool"
)

// MockAdapter is a deterministic adapter for testing without external API calls.
// It simulates LLM responses including tool call decisions.
type MockAdapter struct {
	id   string
	name string
	// callCount tracks how many times Chat was invoked
	callCount int
	// maxToolCalls: number of calls that return tool calls before returning text
	maxToolCalls int
	// alwaysToolCall: if true, always returns tool calls (never settles on text)
	alwaysToolCall bool
}

// NewMockAdapter creates a mock model adapter for testing.
func NewMockAdapter(id, name string) *MockAdapter {
	return &MockAdapter{
		id:           id,
		name:         name,
		maxToolCalls: 1, // first call returns tool call, second returns text
	}
}

func (a *MockAdapter) ID() string           { return a.id }
func (a *MockAdapter) ProviderType() string  { return "custom" }
func (a *MockAdapter) SupportsTools() bool    { return true }

// Chat returns a mock response:
//   - First N calls: return tool calls (if tools are provided)
//   - After that: return a text answer summarizing the conversation
func (a *MockAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	a.callCount++

	// Simulate latency
	time.Sleep(10 * time.Millisecond)

	resp := &ChatResponse{}

	if a.callCount <= a.maxToolCalls && len(req.Tools) > 0 || (a.alwaysToolCall && len(req.Tools) > 0) {
		// Return a tool call for the first available tool
		t := req.Tools[0]
		resp.ToolCalls = []tool.ToolCall{
			{
				ID:        fmt.Sprintf("call_%d", a.callCount),
				Name:      t.Name,
				Arguments: map[string]any{"query": "mock test"},
			},
		}
		resp.Usage = TokenUsage{
			InputTokens:  50,
			OutputTokens: 10,
			TotalTokens:  60,
		}
		return resp, nil
	}

	// Return a text answer
	resp.Content = fmt.Sprintf("[mock] 收到消息，已处理 %d 轮对话。最后消息: %s",
		a.callCount, lastUserContent(req.Messages))
	resp.Usage = TokenUsage{
		InputTokens:  100,
		OutputTokens: 20,
		TotalTokens:  120,
	}
	return resp, nil
}

// Reset call counter (for test isolation).
func (a *MockAdapter) Reset() {
	a.callCount = 0
}

// SetAlwaysToolCall makes the mock always return tool calls (never settle on text).
// Useful for testing the maxIterations guard.
func (a *MockAdapter) SetAlwaysToolCall(v bool) {
	a.alwaysToolCall = v
}

func lastUserContent(msgs []ChatMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return "(none)"
}
