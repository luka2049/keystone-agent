// Package model implements the model adapter layer.
//
// A single Agent config can switch between OpenAI / Tongyi / Ollama
// without code changes. API keys never leave the server side.
//
// ModelAdapter interface:
//   - ID() string
//   - Chat(ctx, ChatRequest) (*ChatResponse, error)
package model

import (
	"context"

	"github.com/chenlong/keystone/internal/tool"
)

// ChatMessage is the unified message format passed to model adapters.
type ChatMessage struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	ToolCalls []tool.ToolCall   `json:"toolCalls,omitempty"`
	ToolCallID string           `json:"toolCallId,omitempty"` // for role=tool messages
}

// ChatRequest is the unified request to a model adapter.
type ChatRequest struct {
	Messages    []ChatMessage
	Tools       []tool.ToolDefinition
	Model       string
	Temperature float64
}

// ChatResponse is the unified response from a model adapter.
type ChatResponse struct {
	Content   string
	ToolCalls []tool.ToolCall
	Usage     TokenUsage
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ModelAdapter is the unified protocol for all model providers.
type ModelAdapter interface {
	ID() string
	ProviderType() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SupportsTools() bool
}
