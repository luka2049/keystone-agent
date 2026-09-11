# Keystone

> **Contract-first, pluggable agent foundation for the enterprise.**
> 企业通用 Agent 基座平台：FED 主导的可视化管控面（React） + Go Agent Runtime。
> 业务方零代码配置即可搭建、编排、运维 Agent；工具、模型、RAG 全部插件化。

[![Contract Lint](https://img.shields.io/badge/contract-openapi%203.0-8BC8EA)](contracts/openapi.yaml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

---

## 为什么是 Keystone

市面上多数 Agent 项目是**写死逻辑的单 Agent Demo**：改 Prompt、接新工具都要动代码，企业无法复用、交付成本高。

Keystone 定位不同——**它不做一个 Agent，而是做承载企业所有 Agent 的基座**：

- **契约驱动**：`contracts/openapi.yaml` 是单一事实源，前端 TS 类型与后端 Go struct 全部由它生成，配置模型与运行时永远一致；
- **可插拔**：Tool / Model / RAG 统一抽象接口，业务方按协议实现即插即用，内核零改动；
- **可视化编排**：react-flow 拖拽搭建 Agent 工作流，业务同学无需开发介入；
- **企业属性**：版本快照、租户隔离、全链路 Trace、可观测面板。

## 架构总览

```
┌──────────────────────────────────────────────────────────────┐
│  管控面 · FED（React + TypeScript）                            │
│  可视化编排 / Agent 配置台 / 工具注册中心 / 会话调试台 / 监控     │
└───────────────────────────┬──────────────────────────────────┘
                            │ REST（契约生成的 TS 类型）
┌───────────────────────────▼──────────────────────────────────┐
│  API 网关 · Go（鉴权 / 限流 / 模型密钥网关 / 日志埋点）          │
└───────────────────────────┬──────────────────────────────────┘
                            │ 内部调用
┌───────────────────────────▼──────────────────────────────────┐
│  Agent Runtime · Go（goroutine 并发调度）                      │
│  执行引擎 / 调度器 / 多 Agent 编排 / 会话状态 / 工作流引擎       │
└───────────────────────────┬──────────────────────────────────┘
                            │ 适配器调用
┌───────────────────────────▼──────────────────────────────────┐
│  插件层 · Tool 适配器 / Model 适配器 / RAG 适配器               │
└───────────────────────────┬──────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────┐
│  基础设施 · PostgreSQL / Redis / Chroma / Ollama（Docker 一键）│
└──────────────────────────────────────────────────────────────┘
```

## 快速开始

```bash
# 1. 安装工具链（Go >=1.22, Node >=20, Docker）
make setup

# 2. 从契约生成双端代码（契约先行）
make gen

# 3. 一键拉起基础设施
make up

# 4. 启动后端与前端
make dev-back   # 终端 1
make dev-front  # 终端 2  → http://localhost:5173
```

## 仓库结构

```text
keystone/
├── contracts/
│   └── openapi.yaml        # ⭐ 单一事实源：Agent/Tool/Session/Workflow/Run 契约
├── frontend/               # FED 管控面（React + TS + Vite + react-flow）
│   └── src/generated/      # openapi-typescript 生成（禁止手改）
├── backend/                # Go Runtime
│   ├── cmd/server/         # 入口
│   └── internal/
│       ├── api/            # oapi-codegen 生成 + handler 实现
│       ├── agent/          # 执行引擎 / 调度器 / 会话
│       ├── tool/           # 工具注册中心
│       ├── model/          # 模型适配器
│       ├── workflow/       # 编排图解析与执行
│       └── store/          # PostgreSQL / Redis
├── examples/               # 示例 Agent 与自定义工具
├── docs/architecture.md    # 架构与设计决策
├── Makefile                # gen / up / dev / test / check
└── docker-compose.yml
```

## 核心标准（一句话版）

| 标准 | 含义 |
|---|---|
| `AgentSpec` | Agent 是配置描述对象，平台读取 Spec 实例化运行，而非硬编码类 |
| `Tool` 协议 | 所有工具统一 `inputSchema(JSON Schema) + Execute`，注册即插即用 |
| `ModelAdapter` | 一套 Agent 配置可切换 OpenAI / 通义 / Ollama，密钥不出服务端 |
| `WorkflowSpec` | 前端编排图（nodes + edges）直接驱动 Runtime 执行引擎 |
| `TraceEvent` | 每步 LLM/工具调用落痕，会话调试台逐步回放 |

## 路线图

- [x] M0 契约先行：OpenAPI 单一事实源 + 双端代码生成链路
- [x] M1 Runtime 核心：执行引擎 / 工具注册中心 / 会话（Go）
- [ ] M2 API 网关 + 持久化（PostgreSQL / Redis）
- [ ] M3 FED 管控面：Agent CRUD / 会话调试台 / 工具注册 UI
- [ ] M4 可视化编排：react-flow 拖拽 + 工作流引擎
- [ ] M5 可观测面板 / Docker 交付 / 演示视频

## License

Apache-2.0
