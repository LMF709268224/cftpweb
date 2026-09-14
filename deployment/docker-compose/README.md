# cftpweb Docker Compose 部署

这个目录负责考生门户和管理门户。四个业务容器是 `candweb`、`candbff`、`adminweb`、`adminbff`；另外使用两个轻量 Caddy 容器 `cand`、`admin` 作为同源入口，在容器内部将 `/api` 转发给对应 BFF、将其他请求转发给对应 Web。业务微服务（包括 `cfgserver`、`gexam`、`gmall` 等）仍由 `cftptest/deployment/docker-compose` 管理。

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
cp .env.example .env
# 编辑 .env 中的域名、镜像仓库、后端网络名和必要的密钥
docker compose up -d
docker compose ps
docker compose logs -f candbff
```

完成 `.env` 配置后，也可以在仓库根目录执行 `./docker_abc.sh`，一次完成 `git pull`、四个镜像构建、Compose 配置校验和服务启动。传入服务名时只更新指定容器，例如 `./docker_abc.sh candbff candweb`。

Compose 会创建 `WEB_NETWORK_NAME` 指定的门户私有网络，并把两个 BFF 同时接入 `BACKEND_NETWORK_NAME` 指定的后端网络。两个门户入口还会接入现有的 `GATEWAY_NETWORK_NAME`。BFF 的 gRPC 地址全部支持在 `.env` 中显式指定；不指定时会按服务名解析，例如 `gexam:50051`。

默认不暴露宿主机端口。开发环境的门户入口容器名为 `cftp-dev-cand` 和 `cftp-dev-admin`，监听容器端口 3000，与 `cftptest/deployment/docker-compose/gateway/Caddyfile` 的目标一致；生产环境将 `PORTAL_CONTAINER_PREFIX` 改为 `cftp-prod`。

## 全局 Caddy 路由

不要在本项目中启动第二个公网 Caddy。宿主机 80/443 由 `cftptest/deployment/docker-compose/gateway` 下的全局 Caddy 独占，其路由应为：

- 开发考生端域名转发到 `cftp-dev-cand:3000`；
- 开发管理端域名转发到 `cftp-dev-admin:3000`；
- 生产考生端域名转发到 `cftp-prod-cand:3000`；
- 生产管理端域名转发到 `cftp-prod-admin:3000`。

修改全局 Caddy 的 `.env` 或 `Caddyfile` 后，在其目录执行 `docker compose restart caddy`。

## 更新与停止

```bash
# 使用远程镜像仓库时先执行：docker compose pull
docker compose up -d --no-deps candweb adminweb candbff adminbff
docker compose down
```
