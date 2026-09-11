// Package tool implements the tool registration center and execution layer.
//
// All tools implement the Tool interface:
//   - ID() string
//   - Name() string
//   - Description() string
//   - InputSchema() map[string]any
//   - Execute(ctx, params, ToolContext) (*ToolResult, error)
//
// Tool execution types: http, code, plugin (extensible).
package tool

import (
	"context"
	"fmt"

	"github.com/chenlong/keystone/internal/types"
)

// ToolContext carries runtime metadata into tool execution.
type ToolContext struct {
	TraceID  string
	RunID    string
	Metadata map[string]any
}

// ToolCall represents a model's request to execute a tool.
// Alias to types.ToolCall so the model package can reference tool.ToolCall.
type ToolCall = types.ToolCall

// Tool is the unified protocol every tool implements.
type Tool interface {
	ID() string
	Name() string
	Description() string
	InputSchema() map[string]any
	Execute(ctx context.Context, params map[string]any, tc ToolContext) (*types.ToolResult, error)
}

// ToolDefinition is the function-call schema passed to the model adapter.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToDefinition converts a Tool to a ToolDefinition for model function calling.
func ToDefinition(t Tool) ToolDefinition {
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: t.InputSchema(),
	}
}

// httpTool wraps an HTTP-execution ToolSpec as a Tool.
type httpTool struct {
	spec types.ToolSpec
}

func (t *httpTool) ID() string                 { return t.spec.ID }
func (t *httpTool) Name() string               { return t.spec.Name }
func (t *httpTool) Description() string         { return t.spec.Description }
func (t *httpTool) InputSchema() map[string]any { return t.spec.InputSchema }
func (t *httpTool) Execute(ctx context.Context, params map[string]any, tc ToolContext) (*types.ToolResult, error) {
	return executeHTTP(ctx, t.spec, params)
}

// codeTool wraps a code-execution ToolSpec as a Tool.
type codeTool struct {
	spec types.ToolSpec
}

func (t *codeTool) ID() string                 { return t.spec.ID }
func (t *codeTool) Name() string               { return t.spec.Name }
func (t *codeTool) Description() string         { return t.spec.Description }
func (t *codeTool) InputSchema() map[string]any { return t.spec.InputSchema }
func (t *codeTool) Execute(ctx context.Context, params map[string]any, tc ToolContext) (*types.ToolResult, error) {
	return executeCode(ctx, t.spec, params)
}

// pluginTool wraps a plugin-execution ToolSpec as a Tool (stub for M1).
type pluginTool struct {
	spec types.ToolSpec
}

func (t *pluginTool) ID() string                 { return t.spec.ID }
func (t *pluginTool) Name() string               { return t.spec.Name }
func (t *pluginTool) Description() string         { return t.spec.Description }
func (t *pluginTool) InputSchema() map[string]any { return t.spec.InputSchema }
func (t *pluginTool) Execute(ctx context.Context, params map[string]any, tc ToolContext) (*types.ToolResult, error) {
	return &types.ToolResult{
		Success: false,
		Error:   "plugin executor not yet implemented (M1 stub)",
	}, nil
}

// FromSpec creates a Tool implementation from a ToolSpec, choosing the executor
// based on spec.Execution.Type.
func FromSpec(spec types.ToolSpec) (Tool, error) {
	switch spec.Execution.Type {
	case types.ToolExecHTTP:
		if spec.Execution.HTTP == nil {
			return nil, fmt.Errorf("http config missing for tool %s", spec.ID)
		}
		return &httpTool{spec: spec}, nil
	case types.ToolExecCode:
		if spec.Execution.Code == nil {
			return nil, fmt.Errorf("code config missing for tool %s", spec.ID)
		}
		return &codeTool{spec: spec}, nil
	case types.ToolExecPlugin:
		return &pluginTool{spec: spec}, nil
	default:
		return nil, fmt.Errorf("unknown execution type: %s", spec.Execution.Type)
	}
}
