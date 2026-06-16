# ==========================================
# 阶段 1: 前端编译 (Node)
# ==========================================
FROM docker.io/library/node:20-alpine AS frontend-builder

COPY shared/ /shared/
WORKDIR /app/vue-web
# 这里的 vue-web 目录将在你后续创建 Vue 项目时出现
# 如果为了防止还没建好 vue-web 目录导致构建报错，可以先用 COPY . . 兜底
COPY candidateserver/vue-web/ ./
# 如果检测到 package.json 存在，就执行依赖安装和编译 (支持 pnpm / npm)
# 强制删除可能被意外拷入的 node_modules 和所有 lock 文件
RUN rm -rf node_modules && \
    if [ -f package.json ]; then \
        npm ci && \
        node -e "require.resolve('@embedpdf/vue-pdf-viewer'); require.resolve('@vue/devtools-api'); require.resolve('@vue/devtools-api', { paths: [require.resolve('vue-router')] }); console.log('frontend dependency check ok')" && \
        npm run build; \
    else \
        mkdir -p dist; \
    fi
# ==========================================
# 阶段 2: 后端编译 (Go)
# ==========================================
FROM docker.io/library/golang:1.26.2-alpine AS backend-builder
WORKDIR /app
# 拷贝当前服务代码和 vendor 依赖
COPY candidateserver/go.mod candidateserver/go.sum ./
# 拷贝当前服务代码
COPY candidateserver .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o candidateserver .

# ==========================================
# 阶段 3: 最终运行环境 (Alpine)
# ==========================================
FROM docker.io/library/alpine:latest

# 安装证书以支持 HTTPS (Casdoor等外部请求需要)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 拷贝阶段2编译好的后端二进制文件
COPY --from=backend-builder /app/candidateserver .
RUN chmod +x candidateserver

# 拷贝阶段1编译好的前端静态文件 (挂载到 vue-web/dist 目录下)
COPY --from=frontend-builder /app/vue-web/dist ./vue-web/dist

EXPOSE 8080
CMD ["./candidateserver"]
