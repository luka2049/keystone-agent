package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chenlong/keystone/internal/types"
)

// executeHTTP performs an HTTP request based on the ToolSpec's http config.
// Params are JSON-encoded into the request body (POST/PUT) or query params (GET).
func executeHTTP(ctx context.Context, spec types.ToolSpec, params map[string]any) (*types.ToolResult, error) {
	cfg := spec.Execution.HTTP
	if cfg == nil {
		return &types.ToolResult{Success: false, Error: "no http config"}, nil
	}

	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := cfg.Method
	if method == "" {
		method = "POST"
	}

	var bodyReader io.Reader
	if method == "POST" || method == "PUT" {
		body, err := json.Marshal(params)
		if err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("marshal params: %v", err)}, nil
		}
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bodyReader)
	if err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("create request: %v", err)}, nil
	}

	// Apply headers from config
	for k, v := range cfg.Headers {
		// Secret interpolation: ${secret.XXX} → env lookup (M1 basic impl)
		req.Header.Set(k, resolveSecret(v))
	}
	if method == "POST" || method == "PUT" {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("http call: %v", err), LatencyMs: latency}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("read body: %v", err), LatencyMs: latency}, nil
	}

	if resp.StatusCode >= 400 {
		return &types.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			LatencyMs: latency,
		}, nil
	}

	var data map[string]any
	_ = json.Unmarshal(respBody, &data) // best-effort JSON parse
	if data == nil {
		data = map[string]any{"raw": string(respBody)}
	}

	return &types.ToolResult{
		Success:   true,
		Data:      data,
		LatencyMs: latency,
	}, nil
}

// resolveSecret resolves ${secret.XXX} references from environment variables.
// This is a basic M1 implementation; M2 will use a proper secret manager.
func resolveSecret(val string) string {
	// Pattern: ${secret.XXX}
	if len(val) > 10 && val[:9] == "${secret." && val[len(val)-1] == '}' {
		key := val[9 : len(val)-1]
		if v := osGetenv(key); v != "" {
			return v
		}
	}
	return val
}

// osGetenv is a variable so it can be mocked in tests.
var osGetenv = func(key string) string {
	return "" // M1 stub: secrets provided via config injection later
}
