# Admin / Candidate 侧边栏红点接口梳理

日期：2026-07-08

## 背景

admin 和 cand 都需要在侧边栏展示“需要处理”的红点或数字徽标。这个徽标不能靠前端从列表第一页、展示文案或本地兜底状态里推断，否则会出现数量不准、分页漏数、状态映射错误，甚至为了绕过接口校验而改坏业务结构的问题。

红点应该由微服务按明确业务状态直接返回 count，BFF 只做鉴权、参数转换和聚合，前端只展示 count。

## 当前现状

### cand 已有能力

- cand 侧边栏已经接了消息未读数。
- 前端位置：`candweb/vue-web/src/components/Sidebar.vue`
- 缓存和轮询：`candweb/vue-web/src/lib/unreadCountCache.ts`
- BFF 接口：`GET /api/messages/unread-count`
- BFF 实现：`candbff/handler/message.go`
- 微服务能力：`gmsg.GetMessageCount(UserUlid, Status=UNREAD)`

这条链路可以继续保留。

### admin 缺失能力

- admin 侧边栏目前没有统一的 badge/count 机制。
- 前端位置：`adminweb/vue-web/src/components/AdminLayout.vue`
- 目前 nav item 只有 `path / label / icon`，没有 count 字段，也没有统一轮询接口。

### 不适合直接复用的能力

- admin 审核中心虽然有 `ListApplications` 和 `AuditApplication`，但当前 BFF 的 status 过滤是在拿到一页结果后本地过滤，代码里已有 TODO：需要 gcreds 给 `ListApplicationsRequest` 增加 status 字段后才能下推筛选。这个不能用于红点 count。
- cand dashboard 里 `GetDashboardTodos` 现在返回空数组，并注明等待后端聚合服务。这说明 cand 的待办类红点也需要稳定后端接口。

## 建议的统一接口形态

### Admin BFF 建议接口

`GET /api/nav-badges`

返回示例：

```json
{
  "updated_at": "2026-07-08T10:00:00Z",
  "items": [
    {
      "key": "mails",
      "count": 3,
      "severity": "warning",
      "href": "/mails?tab=interventions"
    },
    {
      "key": "applications",
      "count": 12,
      "severity": "info",
      "href": "/applications?status=pending"
    }
  ]
}
```

### Candidate BFF 建议接口

`GET /api/nav-badges`

返回示例：

```json
{
  "updated_at": "2026-07-08T10:00:00Z",
  "items": [
    {
      "key": "messages",
      "count": 41,
      "severity": "info",
      "href": "/messages"
    },
    {
      "key": "credentials",
      "count": 2,
      "severity": "warning",
      "href": "/credentials"
    }
  ]
}
```

### 前端展示规则

- `count <= 0` 不显示红点。
- `count > 0` 显示数字徽标。
- 数字太大时前端可以显示 `99+`，但原始 count 不应被前端重算。
- 轮询间隔建议 30 到 60 秒。
- 用户进入对应模块并完成处理后，下一次轮询或主动刷新会更新红点。

## Admin 需要红点的模块

### 1. 邮件中心 `/mails`

需要红点。

触发条件：

- 邮件发送失败且需要管理员干预。
- 邮件重试耗尽。
- 邮件模板缺失、模板渲染失败。
- 收件人非法、供应商拒绝、队列暂停等无法自动恢复的状态。

现有能力：

- `SendMail`
- `GetMail`
- `ListSentMails`
- `GetMailStatus`
- `CancelMail`
- `GetMailStats`
- 邮件模板 CRUD / render / exists / builtin paths

缺口：

- 没有“需要管理员干预”的 count。
- 没有通用邮件失败干预列表。
- 没有通用 retry / ignore / resolve 邮件失败的 admin 接口。

建议微服务新增：

```proto
rpc GetMailInterventionCountAdmin(GetMailInterventionCountAdminRequest)
  returns (GetMailInterventionCountAdminResponse);

rpc ListMailInterventionsAdmin(ListMailInterventionsAdminRequest)
  returns (ListMailInterventionsAdminResponse);

rpc RetryMailAdmin(RetryMailAdminRequest)
  returns (RetryMailAdminResponse);

rpc IgnoreMailFailureAdmin(IgnoreMailFailureAdminRequest)
  returns (IgnoreMailFailureAdminResponse);
```

建议响应字段：

- `count`
- `by_reason`
- `oldest_created_at`
- `items[].mail_ulid`
- `items[].status`
- `items[].reason`
- `items[].last_error_code`
- `items[].last_error_message`
- `items[].retry_count`
- `items[].next_retry_at`

BFF 可补接口：

- `GET /api/mails/interventions/count`
- `GET /api/mails/interventions`
- `POST /api/mails/{mail_ulid}/retry`
- `POST /api/mails/{mail_ulid}/ignore`

### 2. 审核中心 `/applications`

需要红点。

触发条件：

- 有资格申请等待审核。
- 用户补交材料后需要重新审核。
- 审核流程卡住，需要人工处理。

现有能力：

- `ListApplications`
- `GetApplication`
- `AuditApplication`

缺口：

- `ListApplicationsRequest` 目前没有可靠 status 下推筛选，BFF 只能拿一页后过滤，不能算红点。
- 缺少 `CountApplicationsAdmin` 或可筛选并返回 total 的 list 接口。

建议微服务新增：

```proto
rpc CountApplicationsAdmin(CountApplicationsAdminRequest)
  returns (CountApplicationsAdminResponse);
```

或者增强：

```proto
message ListApplicationsRequest {
  string candidate_ulid = 1;
  string cred_def_ulid = 2;
  uint32 page = 3;
  uint32 page_size = 4;
  repeated ApplicationStatus statuses = 5;
}
```

红点建议统计状态：

- `PENDING`
- `RESUBMITTED`
- `WAITING_REVIEW`
- 具体名称以 gcreds 枚举为准。

### 3. 考试管理 `/exams`

需要红点。

触发条件：

- 考试预约失败或预约状态异常。
- 考试已完成但成绩未同步。
- 成绩同步失败。
- webhook 处理失败或待重放。
- 考试实例卡在需要管理员处理的状态。

现有能力：

- `ListAdminExams`
- `GetAdminExamDetail`
- `GetAdminExamResult`
- `GetAdminExamTransitions`
- `SyncAdminExamResult`
- `ListWebhookMessages`
- `GetWebhookMessageDetail`
- `ReprocessWebhookMessage`

缺口：

- 缺少考试维度的 intervention count。
- 缺少 webhook 待处理 / 失败 count。
- 缺少“可人工同步成绩”的 count。

建议微服务新增：

```proto
rpc GetExamInterventionCountAdmin(GetExamInterventionCountAdminRequest)
  returns (GetExamInterventionCountAdminResponse);

rpc GetExamWebhookInterventionCountAdmin(GetExamWebhookInterventionCountAdminRequest)
  returns (GetExamWebhookInterventionCountAdminResponse);
```

建议返回分组：

- `booking_failed_count`
- `result_missing_count`
- `result_sync_failed_count`
- `webhook_failed_count`
- `manual_sync_required_count`

### 4. 证书生成流水 `/pdf-requests` 或 `/prog/certificate-tasks`

需要红点。

触发条件：

- PDF / 证书生成失败。
- gpdf 超时或重试耗尽。
- 证书任务卡住，需要管理员手动重试。

现有能力：

- `GET /api/pdf-requests`
- `GET /api/prog/certificate-tasks`
- `GET /api/prog/certificate-tasks/{task_ulid}`
- `POST /api/prog/certificate-tasks/{task_ulid}/retry`

缺口：

- 缺少失败 / 待干预 certificate task count。
- PDF request list 是否支持 status 过滤和 total 需要微服务确认。

建议微服务新增：

```proto
rpc GetCertificateTaskInterventionCountAdmin(GetCertificateTaskInterventionCountAdminRequest)
  returns (GetCertificateTaskInterventionCountAdminResponse);
```

或者增强现有列表：

- 支持 `status`
- 支持 `need_intervention`
- 返回准确 `total`

### 5. 订单管理 `/orders`

建议有红点，但是否显示取决于业务是否要求管理员处理订单异常。

触发条件：

- 支付成功但业务发放失败。
- Stripe webhook 失败导致订单卡住。
- 退款失败。
- 订单长时间 pending / processing。
- 商品购买后未发放管线、会员、资源包或资格。

缺口：

- 缺少订单异常 count。
- 缺少按原因分组的订单干预列表。

建议微服务新增：

```proto
rpc GetOrderInterventionCountAdmin(GetOrderInterventionCountAdminRequest)
  returns (GetOrderInterventionCountAdminResponse);
```

建议返回分组：

- `payment_paid_biz_failed_count`
- `webhook_failed_count`
- `refund_failed_count`
- `stale_pending_count`

### 6. 发票管理 `/invoices`

建议有红点，但是否显示取决于业务是否要求管理员处理发票异常。

触发条件：

- 发票生成失败。
- 发票发送失败。
- 发票支付状态和订单状态不一致。
- 发票长时间 pending。

建议微服务新增：

```proto
rpc GetInvoiceInterventionCountAdmin(GetInvoiceInterventionCountAdminRequest)
  returns (GetInvoiceInterventionCountAdminResponse);
```

### 7. 站内信 `/messages`

admin 端不建议显示“未读消息”红点，因为 admin 不是普通收件人视角。

只有在站内信存在发送失败、模板异常、需要人工重发时才需要红点。

现有微服务能力：

- gmsg 有 `GetMessageCount`，适合 cand 的用户未读数。
- admin 端已有模板和发送记录接口。

缺口：

- 如果站内信也会发送失败，需要类似邮件的 intervention count。

建议微服务新增：

```proto
rpc GetMessageInterventionCountAdmin(GetMessageInterventionCountAdminRequest)
  returns (GetMessageInterventionCountAdminResponse);
```

### 8. 不建议默认加红点的 admin 模块

以下页面主要是配置或查询，不应默认加红点，除非微服务定义了明确“需要管理员处理”的异常：

- 运营看板 `/dashboard`
- 课程配置 `/lms`
- 资源包配置 `/resource-packs`
- 资源文件配置 `/resource-pack-files`
- 管线配置 `/pipelines`
- 管线管理 `/prog`
- 资格定义 `/credentials`
- 考生权限管理 `/permissions`
- 商品配置 `/bundles`
- 审计日志 `/audit/logs`
- PDF 模板配置 `/pdf-templates`

说明：

- 配置页的 Draft / Active / Deprecated 不等于“待办”。
- 审计日志是排查入口，不是待处理队列。
- 如果后续要给这些模块加红点，微服务必须先给出明确异常状态和 count。

## Candidate 需要红点的模块

### 1. 消息 `/messages`

已经接入。

红点语义：

- 当前用户未读消息数。

现有链路：

- candweb `fetchUnreadCount`
- candbff `GET /api/messages/unread-count`
- gmsg `GetMessageCount(UserUlid, Status=UNREAD)`

### 2. 资格申请 `/credentials`

需要红点。

触发条件：

- 当前用户有资格可以申请，但还没有申请。
- 有被驳回但可补材料的申请。
- 有申请需要继续支付或继续补充信息。

不建议前端自己推断：

- 资格定义、管线阶段、用户已有资格、申请状态之间关系复杂。
- 只靠 `ListCandidateApplications` 和 `CheckCandidateQualifications` 很容易漏掉“可申请但还没申请”的资格。

建议微服务新增：

```proto
rpc GetCandidateCredentialTodoCount(GetCandidateCredentialTodoCountRequest)
  returns (GetCandidateCredentialTodoCountResponse);

rpc ListCandidateCredentialTodos(ListCandidateCredentialTodosRequest)
  returns (ListCandidateCredentialTodosResponse);
```

建议返回分组：

- `available_to_apply_count`
- `resubmit_required_count`
- `payment_required_count`
- `supplement_required_count`

### 3. 考试 `/exams`

需要红点。

触发条件：

- 有考试可以报名但未报名。
- 已报名但未预约。
- 预约失败或需要重新预约。
- 考试完成但成绩未确认、成绩同步异常或允许重考待处理。
- 需要支付重考费用。

建议微服务新增：

```proto
rpc GetCandidateExamTodoCount(GetCandidateExamTodoCountRequest)
  returns (GetCandidateExamTodoCountResponse);

rpc ListCandidateExamTodos(ListCandidateExamTodosRequest)
  returns (ListCandidateExamTodosResponse);
```

建议返回分组：

- `signup_available_count`
- `booking_required_count`
- `booking_failed_count`
- `retake_payment_required_count`
- `result_pending_review_count`

### 4. 我的认证 `/my-certifications`

需要红点。

触发条件：

- 有认证正在进行中。
- 有认证等待下一步操作。
- 有认证等待最终资格申请。
- 有证书发放失败或发放中，需要用户稍后查看。

注意：

- “已完成”不应该显示红点。
- “证书已发放但用户未查看”是否红点，需要产品确认；如果只是通知，建议走消息。

建议微服务新增：

```proto
rpc GetCandidateCertificationTodoCount(GetCandidateCertificationTodoCountRequest)
  returns (GetCandidateCertificationTodoCountResponse);

rpc ListCandidateCertificationTodos(ListCandidateCertificationTodosRequest)
  returns (ListCandidateCertificationTodosResponse);
```

建议返回分组：

- `in_progress_count`
- `next_step_required_count`
- `final_qualification_required_count`
- `certificate_issuing_count`
- `certificate_failed_count`

### 5. 订单 `/orders`

建议可选红点。

触发条件：

- 有未支付订单。
- 支付中订单需要继续处理。
- 支付失败但可重试。

建议微服务新增：

```proto
rpc GetCandidateOrderTodoCount(GetCandidateOrderTodoCountRequest)
  returns (GetCandidateOrderTodoCountResponse);
```

### 6. 不建议默认加红点的 cand 模块

- 商城 `/certifications`：这是购买入口，不是待办入口。
- 资源包 `/resource-packs`：已获得资源不代表待处理。
- 证书 `/certificates`：除非产品要做“新证书未查看”提醒，否则不建议红点。
- 会员 `/membership`：除非有即将过期或支付失败提醒，否则不建议红点。

## 微服务侧建议优先级

### P0

这些是用户已经明确提出，且业务上强依赖的红点：

- admin 邮件需要干预 count + retry/ignore/list。
- admin 审核中心待审核 count。
- admin 考试异常 / webhook / 成绩同步待处理 count。
- cand 消息未读数已经有，不需要新增。
- cand 资格申请待办 count。
- cand 考试待办 count。
- cand 我的认证未结束 / 待下一步 count。

### P1

- admin 证书生成失败 / certificate task 待重试 count。
- admin 订单异常 count。
- cand 订单待支付 count。

### P2

- admin 发票异常 count。
- admin 站内信发送异常 count。
- cand 证书新发放未查看 count。
- cand 会员续费 / 即将过期 count。

## 实现注意事项

- 不要让前端根据展示文案判断状态，例如“开放”“已完成”“证书发放中”。
- 不要让 BFF 从列表第一页推 count。
- 不要为绕过发布校验、状态校验或缺字段校验在前端/BFF 自动补业务数据。
- 微服务需要明确“哪些状态算需要处理”，并提供稳定枚举。
- count 接口和 list 接口的过滤条件必须一致，否则红点点进去后的列表数量对不上。
- 干预动作必须可审计：请求里带 `operator_ulid`，返回操作结果和失败原因。

## 建议给微服务确认的问题

1. 邮件失败是否已经有统一的失败状态、重试次数、下一次重试时间、忽略状态？如果有，请提供 admin count/list/retry/ignore 接口。
2. gcreds 的申请状态枚举有哪些？哪些状态算“待管理员审核”？是否可以给 `ListApplicationsRequest` 增加 `statuses` 并保证 `total` 是过滤后的总数？
3. gexam 哪些考试状态和成绩状态算“需要管理员干预”？webhook 是否有 failed / pending / processed 枚举和 count？
4. gprog / gcreds 的证书生成任务是否有 failed / retryable / permanent_failed / processing_timeout 状态？是否能按这些状态 count？
5. cand 的资格待办、考试待办、认证待办是否由各自微服务提供，还是需要一个 candidate aggregation service 统一聚合？
6. 红点 count 是否需要按 business unit / organization / role 做权限过滤？admin 看到的数量必须和他有权限处理的列表一致。
