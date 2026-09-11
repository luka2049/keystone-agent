# ============================================================
# Keystone - Agent Foundation Platform
# Contract-first: contracts/openapi.yaml 是单一事实源
# 禁止手改 frontend/src/generated/* 与 backend/internal/api/gen/*
# ============================================================

SHELL := /bin/bash
CONTRACT := contracts/openapi.yaml

# ---------- 工具安装（首次使用） ----------
.PHONY: setup
setup:
	@echo "==> 安装契约生成工具链"
	@command -v go >/dev/null || { echo "缺少 Go (>=1.22)，请先安装"; exit 1; }
	@command -v node >/dev/null || { echo "缺少 Node (>=20)，请先安装"; exit 1; }
	@command -v pnpm >/dev/null || npm i -g pnpm
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	pnpm add -D -w openapi-typescript @redocly/cli 2>/dev/null || true

# ---------- 契约层 ----------
.PHONY: gen-contract
gen-contract: ## 校验契约规范性（CI 必跑）
	@echo "==> Lint $(CONTRACT)"
	@npx --yes @redocly/cli lint $(CONTRACT)

# ---------- 双端代码生成 ----------
.PHONY: gen-front
gen-front: ## 从契约生成前端 TS 类型（openapi-typescript）
	@echo "==> 生成 frontend/src/generated/api.ts"
	@cd frontend && npx openapi-typescript ../$(CONTRACT) -o src/generated/api.ts

.PHONY: gen-back
gen-back: ## 从契约生成后端 Go 类型与 handler 骨架（oapi-codegen）
	@echo "==> 生成 backend/internal/api/gen/keystone.gen.go"
	@mkdir -p backend/internal/api/gen
	@cd backend && oapi-codegen -package gen -generate types,server,spec \
		-o internal/api/gen/keystone.gen.go ../$(CONTRACT)

.PHONY: gen
gen: gen-contract gen-front gen-back ## 全量重新生成（契约变更后执行）

# ---------- 开发 / 测试 ----------
.PHONY: up
up: ## 一键拉起基础设施（Postgres/Redis/Chroma/Ollama）
	@docker compose up -d

.PHONY: down
down:
	@docker compose down

.PHONY: dev-back
dev-back: ## 启动 Go 后端（热重载）
	@cd backend && go run ./cmd/server

.PHONY: dev-front
dev-front: ## 启动前端管控面（Vite dev server）
	@cd frontend && pnpm dev

.PHONY: test
test: ## 后端单元测试（含 Agent 引擎核心循环）
	@cd backend && go test ./... -cover

.PHONY: lint
lint: ## 前端与后端静态检查
	@cd frontend && pnpm lint
	@cd backend && go vet ./...

.PHONY: tidy
tidy:
	@cd backend && go mod tidy
	@cd frontend && pnpm install

# ---------- 质量门（交付前必跑） ----------
.PHONY: check
check: gen-contract test lint
	@echo "==> 全部质量门通过 ✅"
