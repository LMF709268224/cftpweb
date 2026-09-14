# cftpweb Docker Compose 部署

这个目录只负责考生门户和管理门户的四个容器：`candweb`、`candbff`、`adminweb`、`adminbff`。业务微服务（包括 `cfgserver`、`gexam`、`gmall` 等）仍由 `cftptest/deployment/docker-compose` 管理。

## 前置条件

1. Docker Engine 和 Compose 插件已安装。
2. 已准备好四个镜像。镜像名默认对应 Kubernetes 清单中的 `localhost/{service}:v1`，也可以用 `IMAGE_REGISTRY` 和 `IMAGE_TAG` 覆盖。
3. 后端微服务 Compose 已启动，并且其私有网络名称与 `BACKEND_NETWORK_NAME` 一致。例如后端项目名为 `cftp-dev` 时，网络通常是 `cftp-dev_cftp-net`。
4. `cfgserver` 中已经配置 `canserver` 和 `adminserver` 两套 BFF 配置。

## 启动门户容器

```bash
cd deployment/docker-compose
cp .env.example .env
# 编辑 .env 中的域名、镜像仓库、后端网络名和必要的密钥
docker compose up -d
docker compose ps
docker compose logs -f candbff
```

Compose 会创建 `WEB_NETWORK_NAME` 指定的前端网络，并把两个 BFF 同时接入 `BACKEND_NETWORK_NAME` 指定的后端网络。BFF 的 gRPC 地址全部支持在 `.env` 中显式指定；不指定时会按服务名解析，例如 `gexam:50051`。

默认不暴露宿主机端口，建议通过下面的 Caddy 网关或现有反向代理接入公网。若只需要临时调试，可在 Compose override 中为 `candweb`、`adminweb` 或 BFF 添加 `ports` 映射。

## 启动 Caddy 网关（可选）

`gateway` 子目录提供一个独立 Compose 文件，将：

- `CANDIDATE_PORTAL_DOMAIN` 的 `/api` 路由到 `candbff`，其余请求路由到 `candweb`；
- `ADMIN_PORTAL_DOMAIN` 的 `/api` 路由到 `adminbff`，其余请求路由到 `adminweb`。

网关和门户 Compose 必须使用同一个 `WEB_NETWORK_NAME`，且宿主机 80/443 端口不能被其他进程占用。

```bash
cd deployment/docker-compose/gateway
cp .env.example .env
# 设置两个门户域名和 WEB_NETWORK_NAME
docker compose up -d
```

如果主机上已经运行了 `cftptest/deployment/docker-compose/gateway`，不要再启动第二个占用 80/443 的 Caddy；把本目录的路由合并到现有网关，或为网关设置其他宿主机端口。

## 更新与停止

```bash
docker compose pull
docker compose up -d --no-deps candweb adminweb candbff adminbff
docker compose down
```
