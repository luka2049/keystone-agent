# Keystone

> **Contract-first, pluggable agent foundation for the enterprise.**
> An enterprise-grade Agent platform: visual control plane (React) + Go Agent Runtime.
> Configure, orchestrate, and operate Agents with zero code — tools, models, and RAG are all pluggable.

[![Contract Lint](https://img.shields.io/badge/contract-openapi%203.0-8BC8EA)](contracts/openapi.yaml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![中文文档](https://img.shields.io/badge/README-中文-555555)](README.zh-CN.md)

---

## Why Keystone

Most Agent projects on the market are **single-Agent demos with hardcoded logic**: changing a prompt or adding a tool requires code changes, making them impossible to reuse across the enterprise with high delivery costs.

Keystone takes a different approach — **it doesn't build one Agent, it builds the foundation that hosts all enterprise Agents**:

- **Contract-Driven**: `contracts/openapi.yaml` is the single source of truth. Frontend TS types and backend Go structs are all generated from it, ensuring configuration model and runtime are always in sync;
- **Pluggable**: Tool / Model / RAG share a unified abstraction interface. Implement the protocol, register, and go — kernel stays untouched;
- **Visual Orchestration**: Build Agent workflows via react-flow drag-and-drop, no developer intervention needed;
- **Enterprise-Ready**: Version snapshots, tenant isolation, full-chain trace, observability dashboard.

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│  Control Plane · FED (React + TypeScript)                     │
│  Visual orchestration / Agent config / Tool registry /       │
│  Session debugger / Monitoring                                │
└───────────────────────────┬──────────────────────────────────┘
                            │ REST (contract-generated TS types)
┌───────────────────────────▼──────────────────────────────────┐
│  API Gateway · Go (auth / rate-limit / model-key gateway)     │
└───────────────────────────┬──────────────────────────────────┘
                            │ internal calls
┌───────────────────────────▼──────────────────────────────────┐
│  Agent Runtime · Go (goroutine concurrent scheduling)         │
│  Execution engine / scheduler / multi-agent orchestration /  │
│  Session state / workflow engine                              │
└───────────────────────────┬──────────────────────────────────┘
                            │ adapter calls
┌───────────────────────────▼──────────────────────────────────┐
│  Plugin Layer · Tool adapters / Model adapters / RAG adapters │
└───────────────────────────┬──────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────┐
│  Infrastructure · PostgreSQL / Redis / Chroma / Ollama        │
└──────────────────────────────────────────────────────────────┘
```

## Quick Start

```bash
# 1. Install toolchain (Go >=1.22, Node >=20, Docker)
make setup

# 2. Generate dual-end code from contract (contract-first)
make gen

# 3. Spin up infrastructure
make up

# 4. Start backend and frontend
make dev-back   # terminal 1
make dev-front  # terminal 2  → http://localhost:5173
```

## Repository Structure

```text
keystone/
├── contracts/
│   └── openapi.yaml        # ⭐ Single source of truth: Agent/Tool/Session/Workflow/Run
├── frontend/               # FED control plane (React + TS + Vite + react-flow)
│   └── src/generated/      # openapi-typescript output (DO NOT EDIT)
├── backend/                # Go Runtime
│   ├── cmd/server/         # Entry point
│   └── internal/
│       ├── types/          # Domain types aligned with OpenAPI schemas
│       ├── api/            # HTTP handlers (OpenAPI contract)
│       ├── agent/          # Execution engine / scheduler / sessions
│       ├── tool/           # Tool registration center
│       ├── model/          # Model adapters (OpenAI / Ollama / Mock)
│       ├── workflow/       # Orchestration graph parser & executor
│       └── store/          # In-memory store (M2: PostgreSQL / Redis)
├── examples/               # Sample agents & custom tools
├── docs/architecture.md    # Architecture & design decisions
├── CLAUDE.md               # Project skill & conventions
├── Makefile                # gen / up / dev / test / check
└── docker-compose.yml      # One-command infrastructure
```

## Core Concepts

| Concept | Description |
|---|---|
| `AgentSpec` | Agent is a config object; platform reads Spec to instantiate runtime, not hardcoded classes |
| `Tool` Protocol | All tools share unified `inputSchema(JSON Schema) + Execute`; register and use |
| `ModelAdapter` | One Agent config switches between OpenAI / Tongyi / Ollama; keys never leave server |
| `WorkflowSpec` | Frontend orchestration graph (nodes + edges) directly drives Runtime execution |
| `TraceEvent` | Every LLM/tool call is recorded for session debugger step-by-step replay |

## Roadmap

- [x] M0 Contract-first: OpenAPI single source of truth + dual-end code generation
- [x] M1 Runtime core: execution engine / tool registry / sessions (Go)
- [ ] M2 API gateway + persistence (PostgreSQL / Redis)
- [ ] M3 FED console: Agent CRUD / session debugger / tool registration UI
- [ ] M4 Visual orchestration: react-flow drag-and-drop + workflow engine
- [ ] M5 Observability dashboard / Docker delivery / demo video

## License

Apache-2.0
