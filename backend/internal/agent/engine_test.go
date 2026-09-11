package agent

import (
	"context"
	"testing"

	"github.com/chenlong/keystone/internal/model"
	"github.com/chenlong/keystone/internal/tool"
	"github.com/chenlong/keystone/internal/types"
)

// TestEngine_Run_MockWithToolCall verifies the core LLM↔tool loop:
// 1. Mock adapter returns a tool call on first iteration
// 2. Tool executes and returns result
// 3. Mock adapter returns text answer on second iteration
func TestEngine_Run_MockWithToolCall(t *testing.T) {
	// Setup registries
	modelReg := model.NewRegistry()
	toolReg := tool.NewRegistry()

	// Register a mock model adapter
	mock := model.NewMockAdapter("mock.test", "Mock Test Model")
	modelReg.Register(mock)

	// Register a mock tool (use ToolSpec → httpTool with a dummy URL)
	toolSpec := types.ToolSpec{
		ID:          "tool.test.echo",
		Name:        "tool.test.echo",
		Description: "A test echo tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
		Execution: types.ToolExecution{
			Type: types.ToolExecCode,
			Code: &types.ToolCodeConfig{
				Language: "javascript",
				Script:   "return params",
			},
		},
		Enabled: true,
	}
	if err := toolReg.RegisterSpec(toolSpec); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	// Create engine
	engine := NewEngine(modelReg, toolReg)

	// Create agent spec
	agentSpec := types.AgentSpec{
		ID:           types.NewUUID(),
		Name:         "Test Agent",
		SystemPrompt: "You are a test agent.",
		ModelRef:     "mock.test",
		ToolRefs:     []string{"tool.test.echo"},
		Strategy: types.AgentStrategy{
			MaxIterations:  5,
			MaxConcurrency: 2,
			ContextWindow:  8192,
			EnableTool:     true,
			TimeoutMs:      30000,
		},
		Status: types.AgentStatusActive,
	}

	// Create session
	session := &types.Session{
		ID:        types.NewUUID(),
		AgentID:   agentSpec.ID,
		Status:    types.SessionActive,
		Messages:  []types.Message{},
	}

	// Run
	result, traces := engine.Run(context.Background(), RunRequest{
		Session:   session,
		AgentSpec: &agentSpec,
		UserMsg:   "Hello, test the echo tool",
		TraceID:   "trace-001",
		RunID:     "run-001",
	})

	// Verify result
	if result.Status != types.RunSucceeded {
		t.Errorf("expected status succeeded, got %s (error: %s)", result.Status, result.Error)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
	if result.Iterations < 2 {
		t.Errorf("expected at least 2 iterations (tool call + answer), got %d", result.Iterations)
	}

	// Verify traces: should have at least 2 model_call events and 1 tool_call event
	var modelCalls, toolCalls int
	for _, e := range traces {
		switch e.Type {
		case types.TraceModelCall:
			modelCalls++
		case types.TraceToolCall:
			toolCalls++
		}
	}
	if modelCalls < 2 {
		t.Errorf("expected at least 2 model_call trace events, got %d", modelCalls)
	}
	if toolCalls < 1 {
		t.Errorf("expected at least 1 tool_call trace event, got %d", toolCalls)
	}

	t.Logf("✅ Run succeeded: iterations=%d, answer=%q", result.Iterations, result.Answer)
	t.Logf("   Traces: %d events (%d model_call, %d tool_call)", len(traces), modelCalls, toolCalls)
	t.Logf("   Usage: input=%d output=%d total=%d",
		result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
}

// TestEngine_Run_MaxIterations verifies the maxIterations guard.
func TestEngine_Run_MaxIterations(t *testing.T) {
	modelReg := model.NewRegistry()
	toolReg := tool.NewRegistry()

	// Create a mock that always returns tool calls (never settles on text)
	mock := model.NewMockAdapter("mock.loop", "Loop Mock")
	mock.SetAlwaysToolCall(true)
	modelReg.Register(mock)

	// Register a tool
	toolSpec := types.ToolSpec{
		ID:          "tool.test.echo",
		Name:        "tool.test.echo",
		Description: "Echo",
		InputSchema: map[string]any{"type": "object"},
		Execution: types.ToolExecution{
			Type: types.ToolExecCode,
			Code: &types.ToolCodeConfig{Language: "javascript", Script: "return 1"},
		},
		Enabled: true,
	}
	toolReg.RegisterSpec(toolSpec)

	engine := NewEngine(modelReg, toolReg)
	agentSpec := types.AgentSpec{
		ID:           types.NewUUID(),
		Name:         "Loop Agent",
		SystemPrompt: "test",
		ModelRef:     "mock.loop",
		ToolRefs:     []string{"tool.test.echo"},
		Strategy: types.AgentStrategy{
			MaxIterations:  3,
			MaxConcurrency: 2,
			ContextWindow:  8192,
			EnableTool:     true,
			TimeoutMs:      10000,
		},
	}
	session := &types.Session{
		ID:       types.NewUUID(),
		AgentID:  agentSpec.ID,
		Status:   types.SessionActive,
		Messages: []types.Message{},
	}

	result, _ := engine.Run(context.Background(), RunRequest{
		Session:   session,
		AgentSpec: &agentSpec,
		UserMsg:   "loop test",
		TraceID:   "trace-002",
		RunID:     "run-002",
	})

	if result.Status != types.RunTimeout {
		t.Errorf("expected status timeout (max iterations), got %s", result.Status)
	}
	if result.Iterations != 3 {
		t.Errorf("expected 3 iterations, got %d", result.Iterations)
	}
	t.Logf("✅ Max iterations guard worked: iterations=%d, status=%s", result.Iterations, result.Status)
}
