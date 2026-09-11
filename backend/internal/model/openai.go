package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chenlong/keystone/internal/tool"
)

// OpenAIAdapter calls any OpenAI-compatible API (OpenAI, Tongyi compat, Ollama compat).
type OpenAIAdapter struct {
	id        string
	name      string
	provider  string
	baseURL   string
	apiKey    string
	model     string
	supportsTools bool
}

// NewOpenAIAdapter creates an adapter from config.
// apiKey: if empty, reads from KEYSTONE_OPENAI_API_KEY env.
// baseURL: if empty, defaults to https://api.openai.com/v1.
func NewOpenAIAdapter(id, name, baseURL, apiKey, model string) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if apiKey == "" {
		apiKey = os.Getenv("KEYSTONE_OPENAI_API_KEY")
	}
	return &OpenAIAdapter{
		id:        id,
		name:      name,
		provider:  "openai",
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		supportsTools: true,
	}
}

func (a *OpenAIAdapter) ID() string         { return a.id }
func (a *OpenAIAdapter) ProviderType() string { return a.provider }
func (a *OpenAIAdapter) SupportsTools() bool  { return a.supportsTools }

// Chat implements ModelAdapter by calling the OpenAI-compatible chat completions API.
func (a *OpenAIAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Build OpenAI request body
	body := map[string]any{
		"model":       a.model,
		"temperature": req.Temperature,
		"messages":    toOpenAIMessages(req.Messages),
	}
	if len(req.Tools) > 0 && a.supportsTools {
		body["tools"] = toOpenAITools(req.Tools)
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(a.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call model: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("model API error %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	result := &ChatResponse{}
	if len(oaiResp.Choices) > 0 {
		choice := oaiResp.Choices[0]
		result.Content = choice.Message.Content
		for _, tc := range choice.Message.ToolCalls {
			args := map[string]any{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			result.ToolCalls = append(result.ToolCalls, tool.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
	}
	result.Usage = TokenUsage{
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
		TotalTokens:  oaiResp.Usage.TotalTokens,
	}
	return result, nil
}

// ---- OpenAI API types ----

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func toOpenAIMessages(msgs []ChatMessage) []map[string]any {
	result := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		msg := map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				calls = append(calls, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": string(argsJSON),
					},
				})
			}
			msg["tool_calls"] = calls
		}
		result = append(result, msg)
	}
	return result
}

func toOpenAITools(tools []tool.ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}
	return result
}

// NewOllamaAdapter creates an OpenAI-compatible adapter pointing to a local Ollama instance.
func NewOllamaAdapter(id, name, model string) *OpenAIAdapter {
	baseURL := os.Getenv("KEYSTONE_OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	return &OpenAIAdapter{
		id:        id,
		name:      name,
		provider:  "ollama",
		baseURL:   baseURL,
		apiKey:    "ollama", // Ollama doesn't need a real key
		model:     model,
		supportsTools: true,
	}
}

// Ensure time is imported (used for future timeout logic)
var _ = time.Second
