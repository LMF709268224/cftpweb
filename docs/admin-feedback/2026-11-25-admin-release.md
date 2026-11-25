# Admin 优化与公开会员入口发布（2026-11-25）

## 发布范围

- 发布主分支中已经完成的 Dashboard、用户管理、审计补充、缺数据处理、课程编辑与 PDF 预览，以及商品报名费修复。
- 功能详情和未完成项仍以本目录各页面清单为准。缺接口、缺业务规则及尚待确认的大流程改造没有随发布自动实现。
- 正式服单独修复分支 `release/prod-admin-enrollment-fee-20261125` 已合回 `main`，合并提交 `d1ab3434`。原修复在主分支已存在，因此此次合并只统一历史，代码树与合并前一致。
- Admin 优化部署版本为 `d1ab3434`。dev / prod 均使用 `/home/ubuntu/cftpweb/docker_abc.sh`，更新 `candbff`、`adminbff`、`candweb`、`adminweb` 四个服务。
- 公开会员入口为 `2a981d01`，随后使用同一脚本仅更新 dev / prod 的 `candweb`。未登录目录页可进入 `/membership`；访客复用会员页的方案接口、卡片和对比表，不请求个人会员记录；登录后认证目录页隐藏重复入口。访客购买仍走已有登录流程。

## 验证与部署进度

- 本地已通过 Admin BFF 的 `go test ./handler ./server`、Vue 类型检查及 E2E 类型检查。
- 已通过 56 项浏览器回归：工作台指标、用户分组和会员／报名记录、缺数据状态、权限查询、课程配置和编辑、PDF 预览、商品及报名费。
- dev 已部署完成，四个容器均为 `healthy`；Admin 与考生端返回 HTTP 200，两个 BFF `/health` 均返回 `OK`。Admin BFF 构建信息确认版本为 `d1ab3434` 且 `vcs.modified=false`。
- prod 已部署完成；在本次 Admin 发布时，dev / prod 共八个容器均为 `healthy`，四个 BFF `/health` 均返回 `OK`，服务器源码为 `d1ab3434`。
- dev / prod 各八项线上页面回归已全部通过。dev 首轮两项、prod 首轮一项遇到页面或资源加载超时，单独重跑后通过；未因此修改业务代码。浏览器回归加载实际部署的前端资源，但业务接口使用测试数据，未修改线上账号、课程或报名。
- 上一版镜像已保留为 `localhost/{candbff,adminbff,candweb,adminweb}:{dev,prod}-before-admin-feedback-20261125`。
- 公开会员入口通过 Vue 类型检查、E2E 类型检查、前端构建和本地三项定向回归；dev / prod 各三项线上页面回归通过。prod 首轮访问页面连接超时，单独重跑通过。两个考生前端容器均为 `healthy`，服务器源码版本为 `2a981d01`；公开会员方案接口在 dev 返回两档当前方案。两环境考生前端上一版镜像另存为 `localhost/candweb:{dev,prod}-before-public-membership-20261125`。

## 分支同步

- `dev`、`lingh`、`review`、`v2`、报名费发布分支此前已推送至 `d1ab3434`；`main` 包含公开会员入口提交 `2a981d01`。用户要求撤回旧分支合并，后续分支同步已暂停。
- 用户此前确认 `v3`、`redesign-aurora` 保留旧版本备份后统一到 `main`；最新明确 `v3` 已废弃，不应把旧分支合入主分支。
- 已创建并推送两条备份分支：`archive/v3-before-sync-20261125`（`d5d4e5c5`）、`archive/redesign-aurora-before-sync-20261125`（`bb8d44df`）。
- 本地曾创建 `312ff223`，使用 `ours` 合并策略连接旧分支历史，代码树与 `d1ab3434` 完全相同。此提交没有推送，也没有部署；已按用户要求撤回。之后的 `2a981d01` 仅修改考生前端会员入口，不包含旧分支提交。
- 已核对 `v3` 的 `d5d4e5c5` 和 `redesign-aurora` 的 `bb8d44df` 均不是当前 `main` 的祖先。原分支与两条备份分支保持旧版本；不再执行旧分支合并或继续同步。
- 工作区中其他未提交的 Admin、考生端、BFF、工具和导入文件不纳入上述代码发布。
