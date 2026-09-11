package agent

import (
	"sync"
	"time"

	"github.com/chenlong/keystone/internal/types"
)

// TraceCollector accumulates TraceEvents during a run.
type TraceCollector struct {
	traceID string
	mu      sync.Mutex
	events  []types.TraceEvent
	seq     int
}

// NewTraceCollector creates a collector for a single run.
func NewTraceCollector(traceID string) *TraceCollector {
	return &TraceCollector{
		traceID: traceID,
	}
}

// AddModelCall records an LLM call event.
func (tc *TraceCollector) AddModelCall(model string, inputTokens, outputTokens int, status types.TraceEventStatus, latencyMs int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.seq++
	tc.events = append(tc.events, types.TraceEvent{
		TraceID:   tc.traceID,
		Seq:       tc.seq,
		Type:      types.TraceModelCall,
		Status:    status,
		LatencyMs: latencyMs,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"model":        model,
			"inputTokens":  inputTokens,
			"outputTokens": outputTokens,
		},
	})
}

// AddToolCall records a tool execution event.
func (tc *TraceCollector) AddToolCall(toolID string, params map[string]any, result *types.ToolResult, status types.TraceEventStatus, latencyMs int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.seq++
	payload := map[string]any{
		"toolId": toolID,
		"params": params,
	}
	if result != nil {
		payload["success"] = result.Success
		if result.Error != "" {
			payload["error"] = result.Error
		}
		if result.Data != nil {
			payload["data"] = result.Data
		}
	}
	tc.events = append(tc.events, types.TraceEvent{
		TraceID:   tc.traceID,
		Seq:       tc.seq,
		Type:      types.TraceToolCall,
		Status:    status,
		LatencyMs: latencyMs,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// Events returns all collected trace events.
func (tc *TraceCollector) Events() []types.TraceEvent {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	out := make([]types.TraceEvent, len(tc.events))
	copy(out, tc.events)
	return out
}
