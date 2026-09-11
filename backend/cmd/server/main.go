// Keystone backend entrypoint (M0 scaffold).
//
// 入口职责（M1 起逐步填充）：
//  1. 加载配置（env / flags）
//  2. 初始化 store（PostgreSQL / Redis）
//  3. 注册 Tool / Model 适配器到注册中心
//  4. 启动 HTTP 服务（handler 由 oapi-codegen 生成，见 internal/api/gen）
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("KEYSTONE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// M0 占位：仅健康检查。契约生成后在此挂载 internal/api 路由。
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"keystone"}`))
	})

	log.Printf("keystone agent foundation listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
