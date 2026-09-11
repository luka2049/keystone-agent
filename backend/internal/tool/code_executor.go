package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/chenlong/keystone/internal/types"
)

// executeCode executes a JavaScript expression in a sandbox.
// M1 provides a basic echo/identity executor for testing.
// M2 will integrate otto/goja or a WASM sandbox.
func executeCode(ctx context.Context, spec types.ToolSpec, params map[string]any) (*types.ToolResult, error) {
	cfg := spec.Execution.Code
	if cfg == nil {
		return &types.ToolResult{Success: false, Error: "no code config"}, nil
	}

	start := time.Now()
	latency := int(time.Since(start).Milliseconds())

	// M1 stub: echo params back as result (real JS sandbox comes in M2)
	return &types.ToolResult{
		Success:   true,
		Data: map[string]any{
			"echo":    params,
			"script":  cfg.Script,
			"message": fmt.Sprintf("code executor M1 stub: language=%s", cfg.Language),
		},
		LatencyMs: latency,
	}, nil
}
