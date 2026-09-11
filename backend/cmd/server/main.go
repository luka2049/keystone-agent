// Keystone backend entrypoint.
//
// Responsibilities:
//  1. Load config (env / flags)
//  2. Initialize store (M1: in-memory; M2: PostgreSQL / Redis)
//  3. Register Tool / Model adapters to registries
//  4. Start HTTP server (handlers follow contracts/openapi.yaml)
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/chenlong/keystone/internal/agent"
	"github.com/chenlong/keystone/internal/api"
	"github.com/chenlong/keystone/internal/model"
	"github.com/chenlong/keystone/internal/store"
	"github.com/chenlong/keystone/internal/tool"
)

func main() {
	addr := os.Getenv("KEYSTONE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// 1. Initialize store
	st := store.NewMemoryStore()

	// 2. Initialize tool registry with built-in tools
	toolReg := tool.NewRegistry()

	// 3. Initialize model registry
	modelReg := model.NewRegistry()

	// Register built-in model adapters
	// Mock adapter (always available for testing without API key)
	mockAdapter := model.NewMockAdapter("mock.default", "Mock Model (for testing)")
	modelReg.Register(mockAdapter)

	// OpenAI adapter (requires KEYSTONE_OPENAI_API_KEY env)
	if os.Getenv("KEYSTONE_OPENAI_API_KEY") != "" {
		openaiAdapter := model.NewOpenAIAdapter(
			"openai.gpt-4o", "GPT-4o", "", "", "gpt-4o",
		)
		modelReg.Register(openaiAdapter)
	}

	// Ollama adapter (requires local Ollama running on :11434)
	modelReg.Register(model.NewOllamaAdapter("ollama.qwen2.5", "Qwen2.5 (Ollama)", "qwen2.5"))

	// 4. Initialize agent engine
	engine := agent.NewEngine(modelReg, toolReg)

	// 5. Initialize API server
	server := api.NewServer(st, engine, toolReg, modelReg)

	log.Printf("keystone agent foundation listening on %s", addr)
	log.Printf("  models: %v", modelAdapterIDs(modelReg))
	log.Printf("  tools:  %d registered", len(toolReg.List()))

	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func modelAdapterIDs(reg *model.Registry) []string {
	providers := reg.List()
	ids := make([]string, 0, len(providers))
	for _, p := range providers {
		ids = append(ids, p.ID)
	}
	return ids
}
