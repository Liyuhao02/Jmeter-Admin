.PHONY: build-frontend build-backend build-all clean clean-share run build-linux dev dev-backend dev-frontend

# 构建前端
build-frontend:
	cd web && npm install && npm run build

# 构建后端（嵌入前端）
build-backend:
	CGO_ENABLED=1 go build -o jmeter-admin .
	CGO_ENABLED=1 go build -o jmeter-agent ./cmd/agent/

# 构建全部
build-all: build-frontend build-backend

# 构建 Linux 版本
build-linux:
	cd web && npm install && npm run build
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o jmeter-admin .
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o jmeter-agent ./cmd/agent/

# 清理构建产物
clean:
	rm -f jmeter-admin jmeter-agent
	rm -rf web/dist

# 打包/分享前清理本地产物和运行数据
clean-share:
	rm -f jmeter-admin jmeter-agent jmeter.log *.out config.yaml
	rm -rf .gocache temp data uploads results web/dist web/node_modules web/.vite
	find . -name '.DS_Store' -delete

# 运行
run:
	./jmeter-admin

# 开发模式（同时启动前后端）
dev:
	@echo "启动后端和前端开发服务器..."
	@CGO_ENABLED=1 go run . & cd web && npm run dev

# 开发模式（前端热更新，后端单独运行）
dev-backend:
	CGO_ENABLED=1 go run .

dev-frontend:
	cd web && BACKEND_PORT=$(or $(BACKEND_PORT),8080) FRONTEND_PORT=$(or $(FRONTEND_PORT),3000) npm run dev
