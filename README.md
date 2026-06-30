# CFtP Web Portal (cftpweb)

本仓库包含了 CFtP 系统的 Web 端网关与静态资源服务，采用了前后端分离和职责分明的现代微服务架构。

---

## 🏗️ 架构设计

系统分为 **Candidate (考生端)** 与 **Admin (管理端)** 两大板块，每个板块均分裂为独立的两个服务，共有 4 个核心微服务：

```
                              ┌────────────────┐
                              │     Browser    │
                              └───────┬────────┘
                                      │
                 ┌────────────────────┴────────────────────┐
                 │                                         │
       [Candidate Portal]                           [Admin Portal]
        /              \                             /            \
┌──────────────┐ ┌──────────────┐           ┌──────────────┐ ┌──────────────┐
│   candweb    │ │   candbff    │           │   adminweb   │ │   adminbff   │
│ (Static/SPA) │ │ (REST APIs)  │           │ (Static/SPA) │ │ (REST APIs)  │
└──────────────┘ └──────┬───────┘           └──────────────┘ └──────┬───────┘
                        │                                           │
                        ├─────────────► Upstream gRPC ◄─────────────┤
                        │       (GLMS, GCREDS, GMALL, GMSG...)      │
```

### 1. Candidate Portal (考生端)
* **[candweb](file:///d:/CFtP/mufan/cftpweb/candweb)**:
  * 职责：使用 Go 的 `//go:embed` 机制嵌入编译后的 Vue SPA 静态文件 (`vue-web/dist`)，并启动高效的静态文件 Web 服务器。
  * 路由：支持单页应用 (SPA) 客户端路由 fallback，自动将非静态资源请求回退至 `index.html`。
* **[candbff](file:///d:/CFtP/mufan/cftpweb/candbff)**:
  * 职责：RESTful API 聚合网关，为考生端提供所需的后台接口。
  * 通信：对接上游各 gRPC 业务微服务 (如 `glms`、`gcreds`、`gmall`、`gmbr` 等)。

### 2. Admin Portal (管理端)
* **[adminweb](file:///d:/CFtP/mufan/cftpweb/adminweb)**:
  * 职责：使用 `//go:embed` 机制嵌入编译后的 Vue SPA 静态文件 (`vue-web/dist`)，并启动高效的静态文件 Web 服务器。
  * 路由：支持单页应用 (SPA) 客户端路由 fallback，自动将非静态资源请求回退至 `index.html`。
* **[adminbff](file:///d:/CFtP/mufan/cftpweb/adminbff)**:
  * 职责：管理端 RESTful API 聚合网关。
  * 路由：仅处理 `/api/...` 请求，未匹配的请求统一返回 API `404 Not Found` JSON。

---

## 📂 目录结构

```
cftpweb/
├── candbff/          # 考生端 BFF 网关 (Go)
├── candweb/          # 考生端 Web 服务 (Go + Vue SPA 嵌入)
│   └── vue-web/      # Vue 前端源码
├── adminbff/         # 管理端 BFF 网关 (Go)
├── adminweb/         # 管理端 Web 服务 (Go + Vue SPA 嵌入)
│   └── vue-web/      # Vue 前端源码
├── shared/           # 前端共享模块 / 公共 TS 类型定义
├── docs/             # 历史设计文档与防复发复盘清单
├── tidy.ps1          # 自动化整理所有 Go 模块依赖的 PowerShell 脚本
├── lint.ps1          # 格式化与静态检查脚本
└── up.md             # 拆分与去 vendor 工作的详细耗时总结
```

---

## 🛠️ 依赖管理

本仓库已彻底移除 `vendor/` 目录和 `vendor` 编译模式，全面采用原生的 Go Modules 规范管理依赖：
* **私有库拉取**：直接引入并拉取私有库 `github.com/afnandelfin620-star/cftptest/cftp`，不使用任何本地 `replace` 相对路径配置。
* **依赖清理**：在修改依赖后，可通过根目录下的 PowerShell 脚本整理各个子模块：
  ```powershell
  .\tidy.ps1
  ```

---

## 🚀 构建与部署流程

每个微服务在各自目录下都配备了统一规范的部署配置：
1. **[Dockerfile](file:///d:/CFtP/mufan/cftpweb/candweb/Dockerfile)**：采用超轻量级的 `alpine:latest`，容器内不进行任何编译，只拷贝预先编译好的 host 静态二进制文件，并补充安装 CA 证书及本地时区。
2. **[image_build.sh](file:///d:/CFtP/mufan/cftpweb/candweb/image_build.sh)**：
   * 在宿主机进行前端打包（Vue 的 `npm run build`）。
   * 通过 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 静态交叉编译 Go 宿主机二进制文件（包含嵌入的静态文件）。
   * 使用 `buildah` 打包容器镜像，并直接导入至本地的 `k3s` 容器中。

### 独立编译示例：
```bash
# 编译 candbff
cd candbff
go build -o candbff .

# 编译 candweb
cd candweb/vue-web
npm run build
cd ..
go build -o candweb .
```
