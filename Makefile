.PHONY: help run build web test vet e2e tidy up down logs clean

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

run: ## 本地启动后端 + 页面（需先配置 .env 与 MySQL）
	go run ./cmd/server

web: ## 构建前端（npm install + vite build → web/dist）
	cd web && npm install && npm run build

build: web ## 编译前后端到 bin/goask（前端产物 embed 进二进制）
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/goask ./cmd/server

test: ## 运行单元测试
	go test ./...

vet: ## 静态检查
	go vet ./...

e2e: ## 端到端冒烟测试（需本地服务已启动）
	python3 scripts/e2e.py

tidy: ## 整理 Go 依赖
	go mod tidy

up: ## docker compose 启动（MySQL + 应用）
	docker compose -f deployments/docker-compose.yml up -d --build

down: ## docker compose 停止
	docker compose -f deployments/docker-compose.yml down

logs: ## 跟踪应用容器日志
	docker compose -f deployments/docker-compose.yml logs -f app

clean: ## 清理构建产物
	rm -rf bin
