// Package types defines domain types aligned with contracts/openapi.yaml.
// These structs are the in-memory representation of the OpenAPI schemas.
// When oapi-codegen is wired up (M2), generated types can replace these.
package types

import (
	"time"

	"github.com/google/uuid"
)

// ========== Agent ==========

type AgentStatus string

const (
	AgentStatusDraft    AgentStatus = "draft"
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
)

type AgentStrategy struct {
	MaxIterations  int  `json:"maxIterations"`
	MaxConcurrency int  `json:"maxConcurrency"`
	ContextWindow  int  `json:"contextWindow"`
	EnableTool     bool `json:"enableTool"`
	TimeoutMs      int  `json:"timeoutMs"`
}

func DefaultStrategy() AgentStrategy {
	return AgentStrategy{
		MaxIterations:  10,
		MaxConcurrency: 4,
		ContextWindow:  8192,
		EnableTool:     true,
		TimeoutMs:      60000,
	}
}

type AgentSpec struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	SystemPrompt string        `json:"systemPrompt"`
	ModelRef     string        `json:"modelRef"`
	ToolRefs     []string      `json:"toolRefs,omitempty"`
	Strategy     AgentStrategy `json:"strategy"`
	Version      string        `json:"version"`
	Status       AgentStatus   `json:"status"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

type AgentVersion struct {
	Version   string     `json:"version"`
	Snapshot  AgentSpec  `json:"snapshot"`
	ChangedBy string     `json:"changedBy,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type AgentList struct {
	Items    []AgentSpec `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// ========== Tool ==========

type ToolExecutionType string

const (
	ToolExecHTTP   ToolExecutionType = "http"
	ToolExecCode   ToolExecutionType = "code"
	ToolExecPlugin ToolExecutionType = "plugin"
)

type ToolHTTPConfig struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
}

type ToolCodeConfig struct {
	Language string `json:"language"`
	Script   string `json:"script"`
}

type ToolPluginConfig struct {
	Ref string `json:"ref"`
}

type ToolExecution struct {
	Type   ToolExecutionType  `json:"type"`
	HTTP   *ToolHTTPConfig    `json:"http,omitempty"`
	Code   *ToolCodeConfig    `json:"code,omitempty"`
	Plugin *ToolPluginConfig  `json:"plugin,omitempty"`
}

type ToolSpec struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  map[string]any  `json:"inputSchema"`
	OutputSchema map[string]any  `json:"outputSchema,omitempty"`
	Execution    ToolExecution   `json:"execution"`
	Enabled      bool            `json:"enabled"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type ToolList struct {
	Items []ToolSpec `json:"items"`
	Total int        `json:"total"`
}

type ToolTestRequest struct {
	Params  map[string]any `json:"params"`
	TraceID string         `json:"traceId,omitempty"`
}

type ToolResult struct {
	Success   bool           `json:"success"`
	Data      map[string]any `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
	LatencyMs int            `json:"latencyMs"`
}

// ========== Model ==========

type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderOllama ProviderType = "ollama"
	ProviderTongyi ProviderType = "tongyi"
	ProviderCustom ProviderType = "custom"
)

type ModelProvider struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	ProviderType  ProviderType `json:"providerType"`
	SupportsTools bool         `json:"supportsTools"`
	ContextWindow int          `json:"contextWindow"`
	Enabled       bool         `json:"enabled"`
}

// ========== Session ==========

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments map[string]any  `json:"arguments"`
}

type Message struct {
	ID          string        `json:"id"`
	Role        MessageRole   `json:"role"`
	Content     string        `json:"content"`
	ToolCalls   []ToolCall    `json:"toolCalls,omitempty"`
	ToolResults []ToolResult  `json:"toolResults,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

type SessionStatus string

const (
	SessionActive  SessionStatus = "active"
	SessionClosed  SessionStatus = "closed"
	SessionExpired SessionStatus = "expired"
)

type Session struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agentId"`
	Status    SessionStatus     `json:"status"`
	Messages  []Message         `json:"messages"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type CreateSessionRequest struct {
	AgentID  string            `json:"agentId"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

// ========== Run ==========

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunTimeout   RunStatus = "timeout"
	RunCancelled RunStatus = "cancelled"
)

type TokenUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
	LatencyMs    int `json:"latencyMs"`
}

type RunResult struct {
	RunID      string     `json:"runId"`
	SessionID  string     `json:"sessionId,omitempty"`
	Status     RunStatus  `json:"status"`
	Answer     string     `json:"answer,omitempty"`
	Iterations int        `json:"iterations,omitempty"`
	Usage      TokenUsage `json:"usage,omitempty"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// ========== Trace ==========

type TraceEventType string

const (
	TraceModelCall TraceEventType = "model_call"
	TraceToolCall  TraceEventType = "tool_call"
	TraceBranch    TraceEventType = "branch"
	TraceSubAgent  TraceEventType = "subagent"
)

type TraceEventStatus string

const (
	TraceStatusOK    TraceEventStatus = "ok"
	TraceStatusError TraceEventStatus = "error"
)

type TraceEvent struct {
	TraceID   string          `json:"traceId"`
	Seq       int             `json:"seq"`
	Type      TraceEventType  `json:"type"`
	NodeID    string          `json:"nodeId,omitempty"`
	Payload   map[string]any  `json:"payload,omitempty"`
	Status    TraceEventStatus `json:"status"`
	LatencyMs int             `json:"latencyMs"`
	Timestamp time.Time       `json:"timestamp"`
}

// ========== Observability ==========

type ObservabilitySummary struct {
	TotalRuns        int            `json:"totalRuns"`
	TotalMessages    int            `json:"totalMessages"`
	TotalTokens      int            `json:"totalTokens"`
	ToolSuccessRate  float64        `json:"toolSuccessRate"`
	AvgLatencyMs     float64        `json:"avgLatencyMs"`
	ErrorByType      map[string]int `json:"errorByType,omitempty"`
}

// ========== Common ==========

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

// NewUUID generates a UUID string.
func NewUUID() string {
	return uuid.NewString()
}

// Now returns current time in UTC.
func Now() time.Time {
	return time.Now().UTC()
}
