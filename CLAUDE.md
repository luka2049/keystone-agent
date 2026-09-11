# CLAUDE.md — Keystone Project Skill

> This file guides Claude Code (and any AI assistant) working on the Keystone project.
> It captures architecture conventions, code patterns, and workflow rules.

## Project Identity

**Keystone** is a contract-first, pluggable agent foundation platform.
- Backend: Go 1.22+ (module path `github.com/chenlong/keystone`)
- Frontend: React 18 + TypeScript + Vite + @xyflow/react
- Contract: `contracts/openapi.yaml` is the SINGLE SOURCE OF TRUTH
- Infra: PostgreSQL / Redis / Chroma / Ollama (via docker-compose)

## Golden Rules

1. **Contract-first**: Never hand-edit `frontend/src/generated/*` or `backend/internal/api/gen/*`.
   Change `contracts/openapi.yaml` → run `make gen` → implement against generated types.
2. **Agent is Spec, not Class**: Agents are JSON config objects (`AgentSpec`).
   New Agent = new config, zero code.
3. **Pluggable interfaces**: Tool / ModelAdapter / RAGAdapter are interfaces.
   New capability = implement interface + register. Kernel never changes.
4. **Three guards against runaway**: `maxIterations` + `context.WithTimeout` + `maxConcurrency` semaphore.
5. **Secrets never leave server**: API keys stored as `${secret.XXX}` references, resolved server-side.
6. **Every step is traced**: `TraceEvent` records each LLM/tool call for debugging replay.

## Backend Architecture (Go)

```
backend/
├── cmd/server/main.go          # Entry point: wire store→registries→engine→API
├── go.mod                      # module github.com/chenlong/keystone
└── internal/
    ├── types/types.go           # Domain types aligned with OpenAPI schemas
    ├── tool/                    # Tool interface + Registry + Executors
    │   ├── tool.go              # Tool interface, ToolContext, ToolCall alias
    │   ├── registry.go          # Concurrent-safe registration center
    │   ├── http_executor.go     # HTTP tool execution (type=http)
    │   └── code_executor.go     # Code tool execution stub (type=code, M2: goja/otto)
    ├── model/                   # ModelAdapter interface + adapters
    │   ├── adapter.go           # ModelAdapter interface, ChatRequest/Response
    │   ├── registry.go          # Model registration center
    │   ├── openai.go            # OpenAI-compatible adapter (OpenAI/Tongyi/Ollama)
    │   └── mock.go              # Deterministic mock for testing (no API key needed)
    ├── agent/                   # Agent Runtime core
    │   ├── engine.go            # LLM↔tool loop with 3 guards
    │   ├── trace.go             # TraceCollector (model_call/tool_call events)
    │   └── engine_test.go       # Unit tests: loop test + maxIterations guard
    ├── api/handler.go           # HTTP handlers (follows OpenAPI contract)
    ├── store/memory.go          # In-memory store (M2: PostgreSQL + Redis)
    └── workflow/engine.go       # Workflow DAG parser (M4)
```

### Key Interfaces

```go
// Tool — every tool implements this
type Tool interface {
    ID() string
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, params map[string]any, tc ToolContext) (*types.ToolResult, error)
}

// ModelAdapter — every model provider implements this
type ModelAdapter interface {
    ID() string
    ProviderType() string
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    SupportsTools() bool
}
```

### Agent Engine Loop (engine.go)

```
1. Build messages: [systemPrompt, ...sessionHistory, userMessage]
2. Truncate if exceeds contextWindow
3. Loop (maxIterations guard):
   a. Call ModelAdapter.Chat(messages, toolDefs)
   b. Record TraceEvent(model_call)
   c. If no tool_calls or tools disabled → return answer
   d. Execute tool_calls concurrently (goroutine + semaphore)
   e. Record TraceEvent(tool_call) for each
   f. Append assistant + tool messages to conversation
   g. Go to a (increment iteration)
4. If maxIterations reached → return RunTimeout
```

### Coding Conventions (Go)

- Package doc comment at top of each file (// Package xxx ...)
- Constructor naming: `NewXxx()` returns `*Xxx`
- Registry pattern: `sync.RWMutex` + `map[string]T` for concurrent-safe registries
- Types with JSON tags matching OpenAPI schema field names
- Error-first return: `if err != nil { return nil, err }`
- Trace everything: model calls, tool calls, with latency + status
- Test with MockAdapter (no external API dependency)

### Build & Test Commands

```bash
# Build
cd backend && go build ./cmd/server

# Test
cd backend && go test ./... -v -cover

# Run server
cd backend && go run ./cmd/server  # → :8080

# Run with env
KEYSTONE_OPENAI_API_KEY=sk-xxx go run ./cmd/server
KEYSTONE_ADDR=:9090 go run ./cmd/server

# Quality gate (from project root)
make check    # gen-contract + test + lint
make gen      # regenerate from contract
make up       # docker compose up -d
make dev-back # start backend
make dev-front # start frontend
```

### Adding a New Model Adapter

1. Implement `ModelAdapter` interface (ID, ProviderType, Chat, SupportsTools)
2. Register in `main.go`: `modelReg.Register(myAdapter)`
3. Reference via `agentSpec.modelRef`

### Adding a New Tool

1. Create `ToolSpec` with `inputSchema` (JSON Schema) + `execution` config
2. POST to `/api/v1/tools` (dynamic registration) OR
3. Implement Go `Tool` interface + `toolReg.Register(t)` (built-in)

## Frontend Architecture (React)

```
frontend/
├── package.json          # React 18, @xyflow/react, zustand, @tanstack/react-query
├── vite.config.ts        # Dev server :5173 + proxy /api → :8080
├── tsconfig.json         # Strict TS, path alias @/* → ./src/*
├── index.html
└── src/
    ├── main.tsx          # Entry: BrowserRouter + QueryClientProvider
    ├── generated/        # openapi-typescript output (DO NOT EDIT)
    ├── api/              # API client functions
    ├── components/       # Reusable UI components
    ├── pages/            # Route-level pages
    ├── store/            # Zustand stores
    └── hooks/            # Custom hooks
```

## Milestone Progress

| Milestone | Status | Scope |
|---|---|---|
| M0 | ✅ Done | Contract-first: OpenAPI + scaffold |
| M1 | ✅ Done | Runtime core: engine / tool registry / sessions / mock+openai adapters / HTTP API |
| M2 | Todo | API gateway + PostgreSQL / Redis persistence |
| M3 | Todo | FED console: Agent CRUD / session debugger / tool registration UI |
| M4 | Todo | Visual orchestration: react-flow + workflow engine |
| M5 | Todo | Observability dashboard / Docker delivery / demo |

## Git Conventions

- Commit format: `MX: <Chinese summary>` (e.g. `M1: 实现 Agent Runtime 核心`)
- Branch: `master` (direct push for now)
- Keep commits atomic: one milestone phase per commit

## File Naming

- Go: `snake_case.go` (e.g. `http_executor.go`)
- Frontend: `camelCase.tsx` / `camelCase.ts`
- Docs: `kebab-case.md`
- Config: `lowercase.yaml` / `lowercase.json`
