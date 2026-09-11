// Package agent implements the Agent runtime: execution engine,
// scheduler, session state, and trace collection.
//
// The Engine is the core LLM↔tool loop:
//  1. Build messages: [systemPrompt, ...sessionHistory, userMessage]
//  2. Call ModelAdapter.Chat(messages, toolDefinitions)
//  3. If response has tool_calls: execute tools concurrently (goroutine + semaphore)
//  4. Append tool results to messages, go to step 2 (increment iteration)
//  5. If response is plain text: return as answer
//  6. Guards: maxIterations, context.WithTimeout, maxConcurrency semaphore
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chenlong/keystone/internal/model"
	"github.com/chenlong/keystone/internal/tool"
	"github.com/chenlong/keystone/internal/types"
)

// Engine orchestrates the Agent execution loop.
type Engine struct {
	models *model.Registry
	tools  *tool.Registry
}

// NewEngine creates an Agent execution engine.
func NewEngine(models *model.Registry, tools *tool.Registry) *Engine {
	return &Engine{
		models: models,
		tools:  tools,
	}
}

// RunRequest contains all context needed for a single Agent run.
type RunRequest struct {
	Session    *types.Session
	AgentSpec  *types.AgentSpec
	UserMsg    string
	TraceID    string
	RunID      string
}

// Run executes the LLM↔tool loop and returns the result.
func (e *Engine) Run(ctx context.Context, req RunRequest) (*types.RunResult, []types.TraceEvent) {
	agent := req.AgentSpec
	strategy := agent.Strategy
	if strategy.MaxIterations == 0 {
		strategy = types.DefaultStrategy()
	}

	// Overall timeout guard
	timeout := time.Duration(strategy.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	trace := NewTraceCollector(req.TraceID)
	start := time.Now()

	result := &types.RunResult{
		RunID:     req.RunID,
		SessionID: req.Session.ID,
		Status:    types.RunRunning,
		CreatedAt: types.Now(),
	}

	// Resolve model adapter
	ma, ok := e.models.Get(agent.ModelRef)
	if !ok {
		result.Status = types.RunFailed
		result.Error = fmt.Sprintf("model not found: %s", agent.ModelRef)
		return result, trace.Events()
	}

	// Build tool definitions from agent.ToolRefs
	toolDefs := e.buildToolDefs(agent.ToolRefs)

	// Build initial messages: [system, ...history, user]
	chatMsgs := e.buildMessages(agent, req.Session, req.UserMsg)

	// Truncate if exceeds context window
	chatMsgs = e.truncateMessages(chatMsgs, strategy.ContextWindow)

	var totalUsage model.TokenUsage
	iterations := 0

	for iterations < strategy.MaxIterations {
		select {
		case <-ctx.Done():
			result.Status = types.RunTimeout
			result.Error = "run timed out"
			result.Iterations = iterations
			result.Usage = toTypesUsage(totalUsage)
			result.CreatedAt = types.Now()
			return result, trace.Events()
		default:
		}

		// Call model
		callStart := time.Now()
		resp, err := ma.Chat(ctx, model.ChatRequest{
			Messages:    chatMsgs,
			Tools:       toolDefs,
			Temperature: 0.7,
		})
		callLatency := int(time.Since(callStart).Milliseconds())

		if err != nil {
			trace.AddModelCall(agent.ModelRef, 0, 0, types.TraceStatusError, callLatency)
			result.Status = types.RunFailed
			result.Error = fmt.Sprintf("model call failed: %v", err)
			result.Iterations = iterations
			result.Usage = toTypesUsage(totalUsage)
			return result, trace.Events()
		}

		totalUsage.InputTokens += resp.Usage.InputTokens
		totalUsage.OutputTokens += resp.Usage.OutputTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		trace.AddModelCall(agent.ModelRef, resp.Usage.InputTokens, resp.Usage.OutputTokens,
			types.TraceStatusOK, callLatency)

		iterations++

		// If no tool calls or tools disabled, we're done
		if !strategy.EnableTool || len(resp.ToolCalls) == 0 {
			result.Status = types.RunSucceeded
			result.Answer = resp.Content
			result.Iterations = iterations
			result.Usage = toTypesUsage(totalUsage)
			result.Usage.LatencyMs = int(time.Since(start).Milliseconds())
			return result, trace.Events()
		}

		// Append assistant message (with tool calls) to conversation
		chatMsgs = append(chatMsgs, model.ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute tool calls concurrently with semaphore
		toolResults := e.executeToolsConcurrently(ctx, resp.ToolCalls, req, strategy.MaxConcurrency, trace)

		// Append tool results to conversation
		for i, tc := range resp.ToolCalls {
			resultJSON := ""
			if i < len(toolResults) && toolResults[i] != nil {
				resultJSON = fmt.Sprintf("%v", toolResults[i])
			}
			chatMsgs = append(chatMsgs, model.ChatMessage{
				Role:       "tool",
				Content:    resultJSON,
				ToolCallID: tc.ID,
			})
		}
	}

	// Max iterations reached
	result.Status = types.RunTimeout
	result.Error = fmt.Sprintf("max iterations (%d) reached", strategy.MaxIterations)
	result.Iterations = iterations
	result.Usage = toTypesUsage(totalUsage)
	result.Usage.LatencyMs = int(time.Since(start).Milliseconds())
	return result, trace.Events()
}

// buildToolDefs resolves toolRefs to ToolDefinitions.
func (e *Engine) buildToolRefs(refs []string) []tool.ToolDefinition {
	defs := make([]tool.ToolDefinition, 0, len(refs))
	for _, ref := range refs {
		if t, ok := e.tools.Get(ref); ok {
			defs = append(defs, tool.ToDefinition(t))
		}
	}
	return defs
}

// buildToolDefs is an alias for buildToolRefs (exported for clarity).
func (e *Engine) buildToolDefs(refs []string) []tool.ToolDefinition {
	return e.buildToolRefs(refs)
}

// buildMessages constructs the chat message array from agent spec and session history.
func (e *Engine) buildMessages(agent *types.AgentSpec, session *types.Session, userMsg string) []model.ChatMessage {
	msgs := make([]model.ChatMessage, 0, len(session.Messages)+2)

	// System prompt (with variable interpolation)
	systemPrompt := interpolate(agent.SystemPrompt, session)
	msgs = append(msgs, model.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Session history
	for _, m := range session.Messages {
		mm := model.ChatMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if len(m.ToolCalls) > 0 {
			mm.ToolCalls = m.ToolCalls
		}
		msgs = append(msgs, mm)
	}

	// Current user message
	msgs = append(msgs, model.ChatMessage{
		Role:    "user",
		Content: userMsg,
	})

	return msgs
}

// truncateMessages implements basic context window truncation.
// If total content length exceeds the window, drops oldest messages
// (preserving system prompt and most recent messages).
func (e *Engine) truncateMessages(msgs []model.ChatMessage, window int) []model.ChatMessage {
	if window <= 0 {
		return msgs
	}
	totalLen := 0
	for _, m := range msgs {
		totalLen += len(m.Content)
	}
	if totalLen <= window {
		return msgs
	}

	// Keep first message (system) and trim from the front
	system := msgs[0]
	rest := msgs[1:]
	for len(rest) > 1 && totalLen > window {
		totalLen -= len(rest[0].Content)
		rest = rest[1:]
	}
	return append([]model.ChatMessage{system}, rest...)
}

// executeToolsConcurrently runs tool calls in parallel with a semaphore guard.
func (e *Engine) executeToolsConcurrently(
	ctx context.Context,
	calls []tool.ToolCall,
	req RunRequest,
	maxConcurrency int,
	trace *TraceCollector,
) []*types.ToolResult {
	if maxConcurrency < 1 {
		maxConcurrency = 4
	}

	results := make([]*types.ToolResult, len(calls))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c tool.ToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = e.executeOneTool(ctx, c, req, trace)
		}(i, call)
	}
	wg.Wait()
	return results
}

// executeOneTool executes a single tool call.
func (e *Engine) executeOneTool(
	ctx context.Context,
	call tool.ToolCall,
	req RunRequest,
	trace *TraceCollector,
) *types.ToolResult {
	t, ok := e.tools.Get(call.Name)
	if !ok {
		result := &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", call.Name),
		}
		trace.AddToolCall(call.Name, call.Arguments, result, types.TraceStatusError, 0)
		return result
	}

	start := time.Now()
	result, err := t.Execute(ctx, call.Arguments, tool.ToolContext{
		TraceID: req.TraceID,
		RunID:   req.RunID,
	})
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		result = &types.ToolResult{
			Success:   false,
			Error:      err.Error(),
			LatencyMs: latency,
		}
		trace.AddToolCall(call.Name, call.Arguments, result, types.TraceStatusError, latency)
		return result
	}
	if result.LatencyMs == 0 {
		result.LatencyMs = latency
	}

	status := types.TraceStatusOK
	if !result.Success {
		status = types.TraceStatusError
	}
	trace.AddToolCall(call.Name, call.Arguments, result, status, latency)
	return result
}

// interpolate replaces ${...} variables in the system prompt.
func interpolate(prompt string, session *types.Session) string {
	// Basic interpolation: ${biz_context} → session metadata
	// M2 will support richer variable resolution
	result := prompt
	for k, v := range session.Metadata {
		result = replaceAll(result, fmt.Sprintf("${%s}", k), v)
	}
	return result
}

func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toTypesUsage(u model.TokenUsage) types.TokenUsage {
	return types.TokenUsage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.TotalTokens,
		LatencyMs:    0,
	}
}
