为了让您直观、全面地回顾本次重构过程，我将从对话开始到结束的所有关键步骤、时间节点以及每个阶段的耗时和交付物汇总如下：

### 🕒 整体耗时概览

* **开始时间**：2026-06-20 04:51:53 (北京时间)
* **结束时间**：2026-06-20 06:56:53 (北京时间)
* **总对话耗时**：**2 小时 5 分钟**
* **项目演进**：`candidateserver` & `adminserver` (2大项目 + 臃肿的 `vendor`) ➡️ **`candbff` & `candweb` & `adminbff` & `adminweb` (4个规范的微服务项目，完全移除 `vendor`)**。

---

### 📊 各阶段关键步骤及耗时汇总

| 阶段 | 核心任务与步骤 | 耗时 | 交付成果 / 变更内容 |
| :--- | :--- | :--- | :--- |
| **阶段 1：设计与拆分规划**<br>*(04:51 - 05:14)* | 1. 确认 `candidateserver` 中 `vendor` 的问题与废除方案。<br>2. 确立拆分设计：`candbff` (REST API 后端) 与 `candweb` (Go embed Vue 静态服务)。<br>3. 规划排除本地 Docker 编译，改为服务器物理编译与构建。 | **22 分 18 秒** | 制定了实施方案；创建了 `candbff` 基础骨架。 |
| **阶段 2：Dockerfile 与构建流程标准化**<br>*(05:16 - 05:39)* | 1. 废除原 Dockerfile 的复杂本地编译，采用纯 Alpine 二进制打包。<br>2. 编写 `image_build.sh` 执行跨平台 CGO 禁用静态编译、Docker 打包并直接载入宿主机 `k3s` 的自动化脚本。 | **23 分 11 秒** | `candbff` & `candweb` 的 [Dockerfile](file:///d:/CFtP/mufan/cftpweb/candweb/Dockerfile) 与 [image_build.sh](file:///d:/CFtP/mufan/cftpweb/candweb/image_build.sh)。 |
| **阶段 3：解决 Private 依赖与 Go Mod 冲突**<br>*(05:39 - 06:00)* | 1. 废除本地 `replace` 改用拉取 Private Git 依赖。<br>2. 解决拉取时报错：`module declares its path as: cftp but was required as: github.com/...`<br>3. 确定是因为远程 `cftp/go.mod` 声明的模块名依然是旧的 `cftp`，协助您推送了对远程 repo 的模块路径更新。 | **20 分 51 秒** | 成功通过远程私有库获取 `cftp` 包，彻底告别了本地依赖和 `replace`。 |
| **阶段 4：Bundle 架构重构适配与 Vue 前端升级**<br>*(06:00 - 06:46)* | 1. 适配最新 `cftp` 模块的变化（移除了 `PipelineGuid`，Stripe 参数等），修复 `candbff` 编译报错。<br>2. **前端全面改版**：前端完全匹配 Bundle 购买机制，重构 `CoursesPage.vue` 与 `PurchaseDialog.vue`，从请求/购买 pipelines 全面替换为请求/购买 bundles (/api/mall/bundles/{bundleId}/purchase)。<br>3. 增加自动解析兼容策略以保持未改动页面的后向兼容性。 | **45 分 34 秒** | [membership.go](file:///d:/CFtP/mufan/cftpweb/candbff/handler/membership.go) 等核心 Handler 改写；[PurchaseDialog.vue](file:///d:/CFtP/mufan/cftpweb/candweb/vue-web/src/components/PurchaseDialog.vue) 前端逻辑；`candbff` 编译 100% 通过。 |
| **阶段 5：Admin Portal 重构与分裂**<br>*(06:48 - 06:55)* | 1. 对 `adminserver` 执行同等规则分裂。<br>2. 创建 `adminbff`，重构 `router.go` 剥离前端路由并仅做 API `/api/...` 网关。<br>3. 将 `web/` (Next.js 项目) 移至 `adminweb/web`。使用 `pnpm install` 解决本地 workspace npm 依赖链接。<br>4. 在 `adminweb/main.go` 中通过 Go 嵌入编译出的 Next.js Static Export，编写自定义 Next.js 动态 HTML 路由回退匹配器。<br>5. 物理删除原 `adminserver/`，运行全局 `tidy.ps1` 校验。 | **7 分 48 秒** | [adminbff/](file:///d:/CFtP/mufan/cftpweb/adminbff) 与 [adminweb/](file:///d:/CFtP/mufan/cftpweb/adminweb) 目录；`tidy.ps1` 自动优化所有 Go Mod；完成编译验证。 |

---

### 📂 重构前后的变化对比

```mermaid
graph TD
    subgraph 重构前 (2个混杂且带 vendor 的大单体)
        A[candidateserver <br> 包含 Go BFF + 物理 Vue dist + vendor]
        B[adminserver <br> 包含 Go BFF + 物理 Next.js dist + vendor]
    end

    subgraph 重构后 (4个职责单一、Go 编译嵌入 SPA、原生 go.mod 的现代微服务)
        C1[candbff <br> Go REST API 网关]
        C2[candweb <br> Go 静态嵌入 Vue dist]
        D1[adminbff <br> Go REST API 网关]
        D2[adminweb <br> Go 静态嵌入 Next.js build]
    end

    A -->|去 vendor / 拆分| C1
    A -->|Go FS Embed| C2
    B -->|去 vendor / 拆分| D1
    B -->|Next.js HTML 匹配| D2
```

这次的高效重构为后续的持续集成（CI/CD）和微服务治理打下了非常干净、高内聚的基础。再次感谢您的信任和配合！