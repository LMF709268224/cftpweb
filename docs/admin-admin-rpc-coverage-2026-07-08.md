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
- 当前 adminbff 已经覆盖主要业务链路，但还缺一批后台向接口，尤其是：
  - `gpay` 的支付/订阅/Webhook/对账管理。
  - `gmall` 的系统邮件任务和 NATS 消息监控。
  - `gprog` 的邮件任务、Driver 事件、NATS 消息监控。
  - `gexam` 的审计消息、提醒邮件管理。
  - 多个 `AdminPurgeCandidate...` 高风险运维接口。

## 全服务调用覆盖

| 微服务 | 说明 | SDK RPC | adminbff 已调用 | candbff 已调用 | 两端都未调用 |
| --- | --- | ---: | ---: | ---: | ---: |
| `cfgserver` | 系统配置 | 1 | 0 | 0 | 1 |
| `gaudit` | 审计日志 | 2 | 2 | 0 | 0 |
| `gcc` | 管线配置 | 18 | 9 | 4 | 6 |
| `gcreds` | 资格、申请、证书、PDF 模板 | 36 | 16 | 10 | 13 |
| `gexam` | 考试实例、结果、Webhook、提醒邮件 | 20 | 8 | 5 | 10 |
| `glms` | 课程、章节、课时、测验、资源包 | 133 | 90 | 16 | 27 |
| `gmail` | 邮件、邮件模板 | 18 | 14 | 0 | 4 |
| `gmall` | 商品、订单、购买、价格、商城事件 | 63 | 19 | 23 | 24 |
| `gmbr` | 会员 / Membership | 19 | 15 | 6 | 3 |
| `gmid` | ID 映射 / 中台 ID 查询 | 6 | 1 | 1 | 5 |
| `gmsg` | 站内信、消息模板 | 17 | 6 | 6 | 6 |
| `gpay` | 支付、发票、订阅、Stripe 对账 | 27 | 1 | 1 | 25 |
| `gpdf` | PDF 生成底层服务 | 1 | 0 | 0 | 1 |
| `gprog` | 管线实例、阶段流转、证书任务 | 32 | 12 | 11 | 13 |

说明：

- 上表是所有 RPC 的调用覆盖，不代表每个未调用接口都应该接到 admin。
- 下面只整理“看起来应该给管理后台/系统运维用，但 adminbff 还没接”的接口。

## 后台向但尚未接入 adminbff 的接口

### `cfgserver`

| RPC | proto/语义 | 建议 |
| --- | --- | --- |
| `GetSystemConfig` | 系统配置查询。 | 如果 admin 需要展示环境、开关、业务参数，应接；目前 Handler 里没有 `ConfigServiceClient` 字段。 |

### `gcc` 管线配置

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `CreateUploadURL` | proto 管理员分区：生成管线封面图上传预签名 URL。 | 未接。 | 如果管线配置需要上传封面图，应该接。 |

### `gcreds` 资格、申请、证书、PDF

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `ListCredentials` | 管理员/开发用：分页及条件列出资格档案。 | 未接。 | 建议接到证书/资格档案管理，否则 admin 只能看定义和申请，不能全局查已发资格。 |
| `IgnoreVersionFile` | 管理员忽略资格文件对账。 | 未接。 | 如果有文件对账失败，需要后台干预入口。 |
| `GetPdfTemplate` | 获取 PDF 证书模板简要信息。 | 未接；当前用 `GetPdfTemplateDetail`。 | 可不接，除非列表/详情需要轻量查询。 |
| `CreatePdfRequest` | 提交生成 PDF 证书请求，由 pipeline 触发或后台生成。 | 未接。 | 如果管理员需要手动补发 PDF，可接。 |
| `UpdatePdfRequest` | 更新 PDF 证书生成请求。 | 未接。 | 需要先确认允许后台修改哪些字段。 |
| `GetPdfRequest` | 获取 PDF 证书生成请求简要信息。 | 未接。 | PDF 生成流水详情页可接。 |
| `GetPdfRequestDetail` | 获取 PDF 证书生成请求详细配置。 | 未接。 | 建议接到 PDF 生成流水详情页。 |
| `CheckResourcesExist` | 批量校验关联外部资源是否存在。 | 未接。 | 适合做配置发布前校验或运维检查。 |
| `AdminPurgeCandidateCredentials` | 管理员清除特定考生在特定资格定义下的资格数据，仅限开发/运维调试。 | 未接。 | 高风险。应纳入统一“考生数据清理”运维页，不要散落到普通页面。 |

### `gexam` 考试管理

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `ListAuditMessages` | 管理端：分页列出考试审计日志。 | 未接。 | 建议接到考试审计/运维页。 |
| `GetAuditMessageDetail` | 获取考试审计日志详情，包含完整 payload。 | 未接。 | 与 `ListAuditMessages` 配套。 |
| `ListExamStatusTransitions` | 管理端/开发端：全局分页列出考试状态变迁历史。 | 未接；当前只接了单个考试的 transitions。 | 建议接到考试管理的全局流转日志。 |
| `ListReminderMails` | 提醒邮件列表查询。 | 未接。 | 考试提醒邮件失败需要管理员干预时应接。 |
| `RetryReminderMail` | 重试提醒邮件发送。 | 未接。 | 与红点/干预流程相关，建议接。 |
| `IgnoreReminderMail` | 忽略/取消提醒邮件发送。 | 未接。 | 与失败任务处理配套。 |
| `GetReminderMailDetail` | 获取提醒邮件详情，包含完整 payload。 | 未接。 | 与列表配套。 |
| `AdminPurgeCandidateExams` | 管理端清理指定考生考试数据，仅用于开发调试/测试。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 |

不建议直接接到 admin 普通页面的系统接口：

- `CreateExam`：业务系统发起考试发放。
- `TermUrlCallback`：外部考试平台回调。

### `glms` 课程、测验、资源包

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `AdminPurgeCandidateCourses` | 清理指定考生课程数据。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 |
| `DeprecateResourcePackAdmin` | 资源包废弃，不可逆。 | 未接。 | 当前业务希望“下架”走 `RevertResourcePackToDraftAdmin`，废弃接口先不要接普通 UI。 |
| `EnrollCandidateCourseAdmin` | 单个课程报名。 | 未接；当前有 `BatchEnrollCandidateCoursesAdmin`。 | 可选。单个报名入口需要时再接。 |
| `GetCourseMaterialDetailAdmin` | 课程资料详情版。 | 未接；当前用 `GetCourseMaterialAdmin`。 | 如果详情版字段更完整，建议替换。 |
| `GetChapterDetailAdmin` | 章节详情版。 | 未接；当前用 `GetChapterAdmin`。 | 若包含聚合信息，建议接到课程配置详情。 |
| `GetLessonDetailAdmin` | 课时详情版。 | 未接；当前用 `GetLessonAdmin`。 | 若包含资料/内容/进度扩展，建议接。 |
| `GetPrerequisiteDetailAdmin` | 前置条件详情版。 | 未接；当前用 `GetPrerequisiteAdmin`。 | 可选。 |
| `GetQuizDetailAdmin` | 测验详情版。 | 未接；当前测验、题目、选项拆开请求。 | 如果能一次返回测验题目树，建议接。 |
| `GetQuizQuestionDetailAdmin` | 题目详情版。 | 未接；当前用 `GetQuizQuestionAdmin`。 | 如果含选项/解析，建议接。 |
| `GetQuizOptionDetailAdmin` | 选项详情版。 | 未接；当前用 `GetQuizOptionAdmin`。 | 可选。 |
| `GradeQuizAttemptAdmin` | 管理员评分测验尝试。 | 未接。 | P1，和考试/测验人工干预相关。 |
| `ListLessonsByCourseAdmin` | 按课程列出课时。 | 未接。 | 如果下拉框需要跨章节选择课时，建议接。 |
| `ListPrerequisitesByRequiredEntityAdmin` | 按被依赖对象反查前置条件。 | 未接。 | 适合做“哪些配置依赖了我”的安全提示。 |
| `UpdateResourcePackFileThumbnailAdmin` | 更新资源包文件缩略图。 | 未接。 | 如果资源文件封面要单独维护，建议接。 |

### `gmail` 邮件中心

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `GetMailDetail` | 邮件详情。 | 未接；当前用 `GetMail`。 | 如果详情版包含 payload/错误日志，建议接到邮件详情。 |
| `ListTemplates` | 模板列表。 | 未接；当前用 `GetTemplateList`。 | 需要确认两者差异，若 `ListTemplates` 支持分页/过滤更好，应替换。 |
| `GetTemplateDetail` | 模板详情。 | 未接；当前用 `GetTemplate`。 | 若详情版字段更完整，建议替换。 |
| `GetBuiltInPath` | 获取单个内置模板路径信息。 | 未接；已接 `GetAllBuiltInPaths`。 | 可选。 |

### `gmall` 商品、订单、商城事件

proto 明确把以下接口放在 `Admin-Facing RPCs` 分区，但 adminbff 还没接：

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `AdminSyncOrderMeta` | 管理员触发同步并重新写入历史订单的 meta 元数据快照。 | 未接。 | 如果商品/管线变化导致订单快照不一致，应接到订单运维。 |
| `IgnoreMailTask` | 管理员忽略发生错误的邮件发送任务。 | 未接。 | P1。邮件失败需要管理员干预时应接。 |
| `ResendMailTask` | 管理员重新触发发送失败的邮件任务。 | 未接。 | P1。 |
| `GetMailTaskSummary` | 查询邮件任务概要。 | 未接。 | 与列表/详情配套。 |
| `GetMailTaskDetail` | 查询邮件发送任务详细状态与错误日志。 | 未接。 | P1。 |
| `ListMailTasks` | 列表查询系统邮件发送日志。 | 未接。 | P1，适合红点和干预列表。 |
| `GetNatsMessageSummary` | 查询 NATS 消息收件箱状态概要。 | 未接。 | 运维监控。 |
| `GetNatsMessageDetail` | 查询 NATS 消息详情与异常日志。 | 未接。 | 运维监控。 |
| `ListNatsMessages` | 列表监控系统订阅的 NATS 消息流动状态。 | 未接。 | 运维监控。 |

### `gmbr` 会员 / Membership

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `AdminListMemberships` | 管理员会员配置列表。 | 未接；当前用 `ListMemberships`。 | 需要问微服务差异。如果 Admin 版字段更完整，应替换。 |
| `AdminGetActiveMembershipCoupons` | 获取当前会员可用优惠券。 | 未接。 | 如果会员配置页要展示/校验优惠券，应接。 |

### `gmsg` 站内信

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `RevokeMessage` | 管理端消息撤回。 | 未接。 | 如果站内信误发需要撤回，建议接。 |
| `GetMessageStats` | 管理端消息统计。 | 未接。 | 可用于红点/统计面板。 |
| `GetTemplateDetail` | 管理端模板详情，与 `GetTemplate` 等价。 | 未接；当前用 `GetTemplate`。 | 可不接，除非统一命名。 |
| `GetBuiltInPath` | 获取单个内置路径信息。 | 未接。 | 可选。 |
| `GetAllBuiltInPaths` | 获取所有内置路径信息列表。 | 未接。 | 建议接，方便创建模板时选择路径而不是手填。 |

额外问题：

- adminbff 现在有 `DeleteMessageTemplate` 路由，但代码里明确标注 `gmsg does not provide DeleteTemplate yet`，当前返回 `501 not implemented`。如果站内信模板需要删除能力，要让微服务补 `DeleteTemplate`。

### `gpay` 支付、发票、订阅、Stripe 对账

proto 明确写了 `Admin Queries` 分区；当前 adminbff 只接了 `ListInvoices`。

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `ListSubscriptions` | 分页查询订阅列表。 | 未接。 | 会员/订阅管理需要。 |
| `ListWebhookEvents` | 分页查询本地记录的 Webhook 事件。 | 未接。 | 支付 Webhook 审计需要。 |
| `GetWebhookEventDetail` | 查询 Webhook 事件详情。 | 未接。 | 与列表配套。 |
| `ListOrderItems` | 查询订单条目列表。 | 未接。 | 订单详情/对账需要。 |
| `ListOrders` | 分页查询所有订单列表。 | 未接；admin 目前主要用 `gmall.ListOrders`。 | 需要确认 `gpay.ListOrders` 与 `gmall.ListOrders` 的职责差异。 |
| `BatchGetOrderAmounts` | 批量查询订单金额。 | 未接。 | 可以补齐 admin 列表金额展示。 |
| `BatchGetInvoiceAmounts` | 批量查询 Invoice 金额。 | 未接。 | 可用于发票列表/对账。 |
| `AlignPaymentRequests` | 将 payment_requests 财务数据与 Stripe 同步对齐。 | 未接。 | 高风险运维，建议放支付运维页。 |
| `RebuildSubscriptions` | 从 payment_requests 重建 subscriptions 表并与 Stripe 对齐。 | 未接。 | 高风险运维。 |
| `AlignSubscriptionBillings` | 对齐订阅账单并补全 invoice 记录。 | 未接。 | 高风险运维。 |
| `AdminSyncPaymentRequestCurrency` | 同步校对 payment_requests 币种与 Stripe 数据。 | 未接。 | 运维对账。 |

调试/测试接口：

- `VerifyProduct`
- `VerifyPrice`

这两个在 proto 中标注为调试/测试。如果 admin 需要 Stripe 配置校验页，可以接，否则不建议放普通业务页。

### `gprog` 管线实例、阶段流转、证书任务

proto 明确把以下接口放在 `Admin-Facing` 分区，但 adminbff 还没接：

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `IgnoreMailTask` | admin 忽略发送失败的邮件任务。 | 未接。 | P1。和管线邮件失败红点/干预有关。 |
| `ResendMailTask` | admin 重发发送失败的邮件任务。 | 未接。 | P1。 |
| `ListMailTasks` | 查询考生邮件发送任务列表。 | 未接。 | P1。 |
| `GetMailTaskDetail` | 邮件任务详情。 | 未接。 | P1。 |
| `ListStages` | 分页查询 stage 列表。 | 未接。 | 如果管线管理要按阶段检索实例，应接。 |
| `GetStageDetail` | 获取 stage 详情。 | 未接。 | 与阶段列表配套。 |
| `ListCourseUnits` | 分页查询 course units 列表。 | 未接。 | 如果课程单元需要全局筛选/干预，应接。 |
| `ListDriverEvents` | 分页查询 Driver 驱动事件列表。 | 未接。 | 运维监控。 |
| `GetDriverEventDetail` | 获取 Driver 驱动事件详情。 | 未接。 | 运维监控。 |
| `ListNatsMessages` | 分页查询 NATS 消息审计列表。 | 未接。 | 运维监控。 |
| `GetNatsMessageDetail` | 获取 NATS 消息审计详情。 | 未接。 | 运维监控。 |
| `AdminPurgeCandidatePipeline` | 清理特定考生已购买 pipeline 数据。 | 未接。 | 高风险。放统一“考生数据清理”运维页。 |

### `gpdf`

| RPC | proto/语义 | 当前情况 | 建议 |
| --- | --- | --- | --- |
| `GenPdf` | 底层 PDF 生成。 | 未接。 | 不建议 admin 直接接底层生成接口；admin 应通过 `gcreds` 的 PDF request / template 或 `gprog` 的证书任务链路操作。 |

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

- `gexam.ListReminderMails`
- `gexam.RetryReminderMail`
- `gexam.IgnoreReminderMail`
- `gexam.GetReminderMailDetail`
- `gmall.ListMailTasks`
- `gmall.GetMailTaskDetail`
- `gmall.ResendMailTask`
- `gmall.IgnoreMailTask`
- `gprog.ListMailTasks`
- `gprog.GetMailTaskDetail`
- `gprog.ResendMailTask`
- `gprog.IgnoreMailTask`
- `glms.GradeQuizAttemptAdmin`

### P2：审计/运维可观测性

- `gexam.ListAuditMessages`
- `gexam.GetAuditMessageDetail`
- `gexam.ListExamStatusTransitions`
- `gmall.ListNatsMessages`
- `gmall.GetNatsMessageDetail`
- `gprog.ListDriverEvents`
- `gprog.GetDriverEventDetail`
- `gprog.ListNatsMessages`
- `gprog.GetNatsMessageDetail`
- `gpay.ListWebhookEvents`
- `gpay.GetWebhookEventDetail`

### P3：配置体验和详情增强

- `gcc.CreateUploadURL`
- `gcreds.ListCredentials`
- `gcreds.GetPdfRequestDetail`
- `gmsg.GetAllBuiltInPaths`
- `gmail.GetTemplateDetail`
- `gmail.GetMailDetail`
- `gmbr.AdminListMemberships`
- `gmbr.AdminGetActiveMembershipCoupons`
- `glms.Get*DetailAdmin`
- `glms.ListLessonsByCourseAdmin`
- `glms.ListPrerequisitesByRequiredEntityAdmin`
- `glms.UpdateResourcePackFileThumbnailAdmin`

## 已接模块摘要

- `gaudit`：审计日志列表和详情已接。
- `gcc`：管线 CRUD、发布、废弃、复制、结构/元数据更新、管理员列表已接；封面上传 URL 未接。
- `gcreds`：资格定义、申请审核、PDF 模板、PDF 请求列表、上传权限、资格撤销/过期已接；资格档案列表、PDF 请求详情、资源校验、Purge 未接。
- `gexam`：考试列表/详情/结果/单考试流转、Webhook 列表/详情/重处理已接；审计消息、提醒邮件、全局流转、Purge 未接。
- `glms`：课程配置、章节、课时、测验题库、资源包、资源文件、课程权限、学习进度大部分已接；人工评分、详情版、资源文件缩略图、Purge 未接。
- `gmail`：邮件发送、列表、状态、统计、模板创建/更新/查询、渲染、内置路径列表已接；详情版和单内置路径未接。
- `gmall`：商品配置、商品订单、部分订单列表、展示价同步、候选人 Bundle 清理已接；商城邮件任务、NATS 监控、订单 meta 同步未接。
- `gmbr`：会员配置、用户会员、账单、授权/撤销、邮件任务、候选人会员清理已接；Admin 列表和优惠券未接。
- `gmsg`：发送、发送记录、模板创建/更新/查询/列表已接；消息撤回、统计、内置路径、模板删除未完成。
- `gpay`：只接了发票列表；支付/订阅/Webhook/对账运维基本未接。
- `gprog`：管线实例列表/详情、状态日志、终止、推进阶段、强制课程完成/报名考试、证书任务和重试已接；邮件任务、Driver/NATS 监控、阶段/课程单元全局列表、Purge 未接。
