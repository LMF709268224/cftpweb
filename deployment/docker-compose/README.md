# cftpweb Docker Compose 部署

这个目录负责考生门户和管理门户。四个业务容器是 `candweb`、`candbff`、`adminweb`、`adminbff`。公网入口由 `cftptest/deployment/docker-compose/gateway` 的全局 Caddy 统一管理，并在同一域名下将 `/api` 转发给 BFF、将其他请求转发给 Web。业务微服务（包括 `cfgserver`、`gexam`、`gmall` 等）仍由 `cftptest/deployment/docker-compose` 管理。

## 前置条件

1. Docker Engine 和 Compose 插件已安装。
2. 已准备好 Go 1.27.1、Node.js 20+ 和 npm，用于构建四个服务镜像。
3. 后端微服务 Compose 已启动，并且其私有网络名称与 `BACKEND_NETWORK_NAME` 一致。例如后端项目名为 `cftp-dev` 时，网络通常是 `cftp-dev_cftp-net`。
4. `cftptest` 的全局 Caddy 已启动并创建 `GATEWAY_NETWORK_NAME` 指定的共享网络，默认是 `cftp-gateway-net`。
5. `cfgserver` 中已经配置 `canserver` 和 `adminserver` 两套 BFF 配置。

## 构建门户镜像

旧环境继续使用各服务目录中的 `abc.sh`，其 `image_build.sh`、`v1` 镜像和 K3s 发布流程保持不变。新 Docker Compose 环境使用独立脚本，不会调用或修改旧部署脚本。

```bash
cd /home/ubuntu/cftpweb
chmod +x build_compose_images.sh docker_abc.sh

# 只构建全部四个 Docker 镜像
./build_compose_images.sh

# 也可以只构建指定服务
# ./build_compose_images.sh v1 candbff candweb
```

## 启动门户容器

```bash
cd deployment/docker-compose
cp .env.example .env.dev
# 测试服使用 .env.dev；同机部署的正式服使用独立的 .env.prod
cp .env.example .env.prod
# 分别编辑两个文件中的域名、镜像仓库、项目名、网络名和必要的密钥
docker compose --env-file .env.dev up -d
docker compose --env-file .env.dev ps
docker compose --env-file .env.dev logs -f candbff
```

证书查验页面使用同源的 `/api/public/test-verify-creds/api/*` 接口。PDF 只在浏览器内存中验签，BFF 只转发根公钥和 ULID/PDF 哈希状态查询，不接收 PDF 文件。

完成配置后，在仓库根目录执行 `./docker_abc.sh` 会使用 `.env.dev` 更新测试服；执行 `./docker_abc.sh --env prod` 会使用 `.env.prod` 更新正式服。两个命令都会一次完成 `git pull`、四个镜像构建、Compose 配置校验和服务启动。传入服务名时只更新指定容器，例如 `./docker_abc.sh candbff candweb` 或 `./docker_abc.sh --env prod candbff candweb`。

Compose 会创建 `WEB_NETWORK_NAME` 指定的门户私有网络，并把两个 BFF 同时接入 `BACKEND_NETWORK_NAME` 指定的后端网络。四个门户容器还会接入现有的 `GATEWAY_NETWORK_NAME`，并使用 `PORTAL_CONTAINER_PREFIX` 创建供全局 Caddy 使用的网络别名。BFF 的 gRPC 地址默认按服务名解析，例如 `gexam:50051`，无需在环境文件中配置。

默认不暴露宿主机端口。开发环境的全局 Caddy 将请求路由到 `cftp-dev-candweb:8080`、`cftp-dev-candbff:8080`、`cftp-dev-adminweb:8081` 和 `cftp-dev-adminbff:8080`；生产环境将 `PORTAL_CONTAINER_PREFIX` 改为 `cftp-prod`。

## 全局 Caddy 路由

不要在本项目中启动第二个公网 Caddy。宿主机 80/443 由 `cftptest/deployment/docker-compose/gateway` 下的全局 Caddy 独占，其路由应为：

- 开发考生端域名的 `/api/*` 转发到 `cftp-dev-candbff:8080`，其他路径转发到 `cftp-dev-candweb:8080`；
- 开发管理端域名的 `/api/*` 转发到 `cftp-dev-adminbff:8080`，其他路径转发到 `cftp-dev-adminweb:8081`；
- 生产环境使用相同规则，将前缀替换为 `cftp-prod`。

修改全局 Caddy 的 `.env` 或 `Caddyfile` 后，在其目录执行 `docker compose restart caddy`。

## 更新与停止

```bash
# 使用远程镜像仓库时先执行：docker compose pull
docker compose up -d --no-deps candweb adminweb candbff adminbff
docker compose down
```
