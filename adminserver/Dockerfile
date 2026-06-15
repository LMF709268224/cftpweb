# ==========================================
# 阶段 1: 前端编译 (Node)
# ==========================================
FROM docker.io/library/node:20-alpine AS frontend-builder
COPY shared/ /shared/
WORKDIR /app/web
COPY adminserver/web/ ./
# 如果检测到 package.json 存在，就执行依赖安装和编译 (支持 pnpm / npm)
RUN if [ -f pnpm-lock.yaml ]; then \
    npm install -g pnpm && pnpm install && pnpm run build; \
    elif [ -f package.json ]; then \
    npm install && npm run build; \
    else \
    mkdir -p build; \
    fi

# ==========================================
# 阶段 2: 后端编译 (Go)
# ==========================================
FROM docker.io/library/golang:1.26.2-alpine AS backend-builder
WORKDIR /app
# 拷贝当前服务代码和 vendor 依赖
COPY adminserver/go.mod adminserver/go.sum ./
# 拷贝当前服务代码
COPY adminserver .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o adminserver .

# ==========================================
# 阶段 3: 最终运行环境 (Alpine)
# ==========================================
FROM docker.io/library/alpine:latest

# 安装证书以支持 HTTPS (Casdoor等外部请求需要)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 拷贝阶段2编译好的后端二进制文件
COPY --from=backend-builder /app/adminserver .
RUN chmod +x adminserver

# 拷贝阶段1编译好的前端静态文件 (挂载到 vue-web/dist 目录下)
COPY --from=frontend-builder /app/web/build ./web/build

EXPOSE 8080
CMD ["./adminserver"]
