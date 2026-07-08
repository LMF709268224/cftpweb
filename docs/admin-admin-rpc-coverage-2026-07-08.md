# Admin 微服务接口接入覆盖盘点

日期：2026-07-08

## 口径

- 微服务 SDK：`github.com/afnandelfin620-star/cftptest/cftp v0.0.0-20260705015142-e0830875b701`
- 全量 RPC：从 generated gRPC client 的 `*_grpc.pb.go` 扫描所有 service client 方法。
- 已接入口径：从 `adminbff/handler` 扫描实际调用的 `h.<Service>.<RpcName>(...)`。
- 候选端对照：从 `candbff/handler` 扫描实际调用，用来区分纯候选端接口。
- 后台向接口识别规则：
  - 方法名包含 `Admin` / `ForAdmin`。
  - proto 注释明确写在 `Admin-Facing`、`Admin/System`、`Admin Queries`、`管理端`、`管理员` 分区。
  - 无 Admin 命名但语义明显是后台/运维/审计/模板/统计/重试/忽略/同步/对账/全局列表的接口。

## 重要结论

- 不能只按 `Admin` 后缀扫描。很多后台接口没有 Admin 后缀，例如 `gmail.CreateTemplate`、`gpay.ListInvoices`、`gmall.ListMailTasks`、`gprog.ListDriverEvents`。
- `gmbr` 是 membership service，也就是会员/订阅相关微服务。
- 当前 adminbff 已经覆盖主要业务链路和本批次确定的运维查询/干预接口；剩余未接主要是：
  - `gpay` 的支付订单职责差异和高风险对账/重建接口。
  - 多个 `AdminPurgeCandidate...` 高风险运维接口。
  - 若干上传、会员优惠券、模板删除能力等需要进一步确认的接口。

## 全服务调用覆盖

| 微服务 | 说明 | SDK RPC | adminbff 已调用 | candbff 已调用 | 两端都未调用 |
| --- | --- | ---: | ---: | ---: | ---: |
| `cfgserver` | 系统配置 | 1 | 0 | 0 | 1 |
| `gaudit` | 审计日志 | 2 | 2 | 0 | 0 |
| `gcc` | 管线配置 | 18 | 9 | 4 | 6 |
| `gcreds` | 资格、申请、证书、PDF 模板 | 36 | 23 | 10 | 6 |
| `gexam` | 考试实例、结果、Webhook、提醒邮件 | 20 | 15 | 5 | 3 |
| `glms` | 课程、章节、课时、测验、资源包 | 133 | 91 | 16 | 26 |
| `gmail` | 邮件、邮件模板 | 18 | 17 | 0 | 1 |
| `gmall` | 商品、订单、购买、价格、商城事件 | 63 | 28 | 23 | 15 |
| `gmbr` | 会员 / Membership | 19 | 15 | 6 | 3 |
| `gmid` | ID 映射 / 中台 ID 查询 | 6 | 1 | 1 | 5 |
| `gmsg` | 站内信、消息模板 | 17 | 10 | 6 | 1 |
| `gpay` | 支付、发票、订阅、Stripe 对账 | 27 | 8 | 1 | 18 |
| `gpdf` | PDF 生成底层服务 | 1 | 0 | 0 | 1 |
| `gprog` | 管线实例、阶段流转、证书任务 | 32 | 23 | 11 | 2 |

说明：

- 上表是所有 RPC 的调用覆盖，不代表每个未调用接口都应该接到 admin。
- 下面只整理“看起来应该给管理后台/系统运维用，但 adminbff 还没接”的接口。

## 后台向但尚未接入 adminbff 的接口

### `cfgserver`

| RPC | proto/语义 | 建议 | 处理 |
| --- | --- | --- | --- |
| `GetSystemConfig` | 系统配置查询。 | 如果 admin 需要展示环境、开关、业务参数，应接；目前 Handler 里没有 `ConfigServiceClient` 字段。 | TODO:先不做

### `gcc` 管线配置

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `CreateUploadURL` | proto 管理员分区：生成管线封面图上传预签名 URL。 | 未接。 | 如果管线配置需要上传封面图，应该接。 | TODO:先不做

### `gcreds` 资格、申请、证书、PDF

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `UpdatePdfRequest` | 更新 PDF 证书生成请求。 | 未接。 | 需要先确认允许后台修改哪些字段。 | TODO:先不做
| `AdminPurgeCandidateCredentials` | 管理员清除特定考生在特定资格定义下的资格数据，仅限开发/运维调试。 | 未接。 | 高风险。应纳入统一“考生数据清理”运维页，不要散落到普通页面。 | TODO:先不做

### `gexam` 考试管理

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `AdminPurgeCandidateExams` | 管理端清理指定考生考试数据，仅用于开发调试/测试。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 | TODO:先不做

不建议直接接到 admin 普通页面的系统接口：

- `CreateExam`：业务系统发起考试发放。
- `TermUrlCallback`：外部考试平台回调。

### `glms` 课程、测验、资源包

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `AdminPurgeCandidateCourses` | 清理指定考生课程数据。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 | TODO:先不做
| `DeprecateResourcePackAdmin` | 资源包废弃，不可逆。 | 未接。 | 当前业务希望“下架”走 `RevertResourcePackToDraftAdmin`，废弃接口先不要接普通 UI。 | TODO:先不做
| `EnrollCandidateCourseAdmin` | 单个课程报名。 | 未接；当前有 `BatchEnrollCandidateCoursesAdmin`。 | 可选。单个报名入口需要时再接。 | TODO:先不做
| `GradeQuizAttemptAdmin` | 管理员评分测验尝试。 | 未接。 | P1，和考试/测验人工干预相关。 |  TODO:先不做
| `ListPrerequisitesByRequiredEntityAdmin` | 按被依赖对象反查前置条件。 | 未接。 | 适合做“哪些配置依赖了我”的安全提示。 |  TODO:先不做
| `UpdateResourcePackFileThumbnailAdmin` | 更新资源包文件缩略图。 | 未接。 | 如果资源文件封面要单独维护，建议接。 |  TODO:先不做

### `gmbr` 会员 / Membership

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `AdminListMemberships` | 管理员会员配置列表。 | 未接；当前用 `ListMemberships`。 | 需要问微服务差异。如果 Admin 版字段更完整，应替换。 |  TODO:先不做
| `AdminGetActiveMembershipCoupons` | 获取当前会员可用优惠券。 | 未接。 | 如果会员配置页要展示/校验优惠券，应接。 | TODO:先不做

### `gmsg` 站内信

额外问题：

- adminbff 现在有 `DeleteMessageTemplate` 路由，但代码里明确标注 `gmsg does not provide DeleteTemplate yet`，当前返回 `501 not implemented`。如果站内信模板需要删除能力，要让微服务补 `DeleteTemplate`。

### `gpay` 支付、发票、订阅、Stripe 对账

proto 明确写了 `Admin Queries` 分区；当前 adminbff 已接发票、订阅、Webhook、订单条目、批量金额查询和币种同步，剩余未接如下。

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `ListOrders` | 分页查询所有订单列表。 | 未接；admin 目前主要用 `gmall.ListOrders`。 | 需要确认 `gpay.ListOrders` 与 `gmall.ListOrders` 的职责差异。 | TODO:先不做
| `AlignPaymentRequests` | 将 payment_requests 财务数据与 Stripe 同步对齐。 | 未接。 | 高风险运维，建议放支付运维页。 | TODO:先不做
| `RebuildSubscriptions` | 从 payment_requests 重建 subscriptions 表并与 Stripe 对齐。 | 未接。 | 高风险运维。 | TODO:先不做
| `AlignSubscriptionBillings` | 对齐订阅账单并补全 invoice 记录。 | 未接。 | 高风险运维。 | TODO:先不做

调试/测试接口：

- `VerifyProduct`
- `VerifyPrice`

这两个在 proto 中标注为调试/测试。如果 admin 需要 Stripe 配置校验页，可以接，否则不建议放普通业务页。

### `gprog` 管线实例、阶段流转、证书任务

proto 明确把以下接口放在 `Admin-Facing` 分区，但 adminbff 还没接：

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `AdminPurgeCandidatePipeline` | 清理特定考生已购买 pipeline 数据。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 | TODO:先不做

### `gpdf`

| RPC | proto/语义 | 当前情况 | 建议 | 处理 |
| --- | --- | --- | --- | --- |
| `GenPdf` | 底层 PDF 生成。 | 未接。 | 不建议 admin 直接接底层生成接口；admin 应通过 `gcreds` 的 PDF request / template 或 `gprog` 的证书任务链路操作。 | TODO:先不做

## 高优先级建议

### P0：高风险运维接口，必须统一设计

- `gcreds.AdminPurgeCandidateCredentials`
- `gexam.AdminPurgeCandidateExams`
- `glms.AdminPurgeCandidateCourses`
- `gmall.AdminPurgeCandidateBundle`：已接。
- `gmbr.AdminPurgeCandidateMembership`：已接。
- `gprog.AdminPurgeCandidatePipeline`

建议做统一的“考生数据清理/重置”页面，进入前显示影响模块，让管理员勾选清理范围，强二次确认，并记录审计日志。

### P1：红点/人工干预相关

- `glms.GradeQuizAttemptAdmin`

### P2：审计/运维可观测性

本批次已接完原 P2 中确定的查询/详情接口，暂未发现剩余 P2 确定项。

### P3：配置体验和详情增强

- `gcc.CreateUploadURL`
- `gmbr.AdminListMemberships`
- `gmbr.AdminGetActiveMembershipCoupons`
- `glms.ListPrerequisitesByRequiredEntityAdmin`
- `glms.UpdateResourcePackFileThumbnailAdmin`

## 已接模块摘要

- `gaudit`：审计日志列表和详情已接。
- `gcc`：管线 CRUD、发布、废弃、复制、结构/元数据更新、管理员列表已接；封面上传 URL 未接。
- `gcreds`：资格定义、资格档案列表、申请审核、PDF 模板、PDF 请求列表/详情、资源校验、上传权限、资格撤销/过期已接；PDF 请求更新和 Purge 未接。
- `gexam`：考试列表/详情/结果/单考试流转、Webhook 列表/详情/重处理、审计消息、提醒邮件、全局流转已接；Purge 未接。
- `glms`：课程配置、章节、课时、测验题库、详情增强、资源包、资源文件、课程权限、学习进度大部分已接；人工评分、资源文件缩略图、Purge 未接。
- `gmail`：邮件发送、列表、详情、状态、统计、模板创建/更新/查询/详情、渲染、内置路径列表和单路径查询已接。
- `gmall`：商品配置、商品订单、部分订单列表、展示价同步、订单 meta 同步、候选人 Bundle 清理、商城邮件任务、NATS 监控已接。
- `gmbr`：会员配置、用户会员、账单、授权/撤销、邮件任务、候选人会员清理已接；Admin 列表和优惠券未接。
- `gmsg`：发送、发送记录、撤回、统计、模板创建/更新/查询/列表/详情、内置路径已接；模板删除未完成。
- `gpay`：发票列表、订阅列表、Webhook 事件列表/详情、订单条目查询、金额批量查询、币种同步已接；支付订单职责差异和高风险对账/重建未接。
- `gprog`：管线实例列表/详情、状态日志、终止、推进阶段、强制课程完成/报名考试、证书任务和重试、邮件任务、Driver/NATS 监控、阶段/课程单元全局列表已接；Purge 未接。
