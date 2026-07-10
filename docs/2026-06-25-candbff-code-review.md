# candbff Code Review Summary - 2026-06-25


## Scope

本次 review 覆盖 `candbff` 的 handler、server、config 以及关键考生端流程：

- 登录鉴权与用户信息
- 认证/管线/课程/考试/重考
- 资格申请与证书
- 商城、订单、支付、发票
- 会员中心
- 消息、资源包、PDF 预览

当前 `candbff` 工作区无未提交 diff；`go test ./...` 已通过。

## P1 - 必须优先修

### 1. 发票接口缺少考生归属校验

位置：

- `candbff/handler/invoice.go:31`
- `candbff/handler/invoice.go:57`
- `candbff/server/router.go:183`

现状：

- `/api/invoices/{orderId}`
- `/api/invoices/{orderId}/pdf`

这两个接口只通过 URL 中的 `orderId` 调用 `Gpay.GetInvoice`。

问题：

当前没有确认该 `orderId` 是否属于 `CandidateID(r)`。如果登录考生拿到或猜到别人的订单 ULID，就可能查询别人的发票信息或下载 PDF。

影响：

- 发票 URL й¶
- 发票 PDF й¶
- 订单金额、邮箱、支付信息等敏感数据泄露

建议：

- 短期：在 BFF 调 `Gpay.GetInvoice` 前，用当前考生的订单列表或订单汇总确认 `orderId/pay_order_ulid` 属于该 candidate，不属于则返回 `403`。
- 中期：推动 `gpay.GetInvoiceRequest` 增加 `candidate_ulid`，由 gpay 在服务端强校验归属。
- 前端不应仅依赖 `can_view_invoice` 控制安全，后端必须做归属校验。

### 2. `/api/pipeline/resource-preview` 存在 SSRF 风险

位置：

- `candbff/handler/pipeline.go:560`
- `candbff/handler/pipeline.go:588`
- `candbff/handler/pipeline.go:754`

现状：

`PreviewResourceURL` 接收前端传入的 `src`，只校验：

- URL 可解析
- scheme 是 `http` 或 `https`
- host 非空

然后 BFF 直接用 `http.DefaultClient.Do` 请求该 URL。

问题：

登录用户可以让 candbff 访问任意地址，包括：

- 内网 service
- localhost/loopback
- link-local 地址
- 云厂商 metadata endpoint
- 非预期外部地址

影响：

- SSRF
- 内网探测
- 可能读取内部服务响应并转发给前端
- BFF 成为任意 URL 代理

建议：

- 不要让前端传任意 URL 给 BFF 代理。
- 优先改成只接受后端签发的资源 ID 或 preview token。
- 如必须保留 URL 模式，需要增加 allowlist，仅允许对象存储或受信 CDN 域名。
- 禁止 private、loopback、link-local、multicast IP。
- 限制 redirect 后的 host。
- 使用自定义 HTTP client，设置连接超时、响应头超时、最大响应体大小。

## P2 - 高优先级修复

### 3. `/api/orders/{orderId}` 已注册但 handler 是空实现

位置：

- `candbff/server/router.go:179`
- `candbff/handler/payment.go:33`

现状：

`GetOrder` handler 为空：

```go
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
    // Reserved for an order detail page. The order list is built from business-order list APIs.
}
```

问题：

该接口会返回空 body 的 `200 OK`，前端会认为请求成功，但拿不到任何 JSON。

影响：

- 订单详情页可能白屏或解析失败
- 排查时会误判成前端问题
- API 契约不清晰

建议：

- 要么实现订单详情聚合。
- 要么暂时返回标准 JSON 错误，例如 `501 Not Implemented`。
- 不建议继续返回空 `200`。

### 4. 考生订单列表在不传 `biz_type` 时会拉全量多类型订单

位置：

- `candbff/handler/payment.go:104`
- `candbff/handler/payment.go:135`

现状：

`GET /api/orders` 如果不传 `biz_type`，会依次拉取以下所有类型订单，并且每种类型拉所有分页：

- `PIPELINE_PAYMENT`
- `STAGE_PAYMENT`
- `COURSE_RETAKE_PAYMENT`
- `PIPELINE_UNLOCK`
- `CREDENTIAL_APPLICATION`
- `BUNDLE_PURCHASE`

然后在 BFF 内存里排序、切片分页。

问题：

第一页请求也会触发所有订单类型的全量分页拉取。

影响：

- 考生订单多时，页面会越来越慢
- BFF 到 gmall 的 gRPC 调用被放大
- 单个用户也能制造较高负载

建议：

- 最优：让 gmall 提供统一订单列表接口，按 `candidate_ulid + page/page_size + status` 返回已经合并排序的数据。
- 短期：默认要求前端带 `biz_type` 查询，全部订单页只拉每类前 N 条。
- 加保护：限制聚合最大页数或最大扫描条数。

### 5. 资格申请列表固定只取第一页 100 条

位置：

- `candbff/handler/credentials.go:159`

现状：

`ListCandidateApplications` 固定：

```go
Page: 1,
PageSize: 100,
```

问题：

接口忽略前端传入的 `page/page_size/status`。如果申请记录超过 100 条，后面的记录永远不可见。

影响：

- 资格申请历史不完整
- 分页 UI 无法正确工作
- 审核记录可能被用户误认为丢失

建议：

- 使用统一的 `parsePagination`。
- 透传 `Page/PageSize` 到 gcreds。
- 如果 gcreds 支持状态筛选，也应透传 `status`。
- 返回 total/page/page_size，保持与其它列表一致。

### 6. PDF preview token 有固定 fallback secret

位置：

- `candbff/handler/pipeline.go:743`

现状：

`pdfPreviewSigningKey` 优先用 `CasdoorClientSecret`，为空时用 `CasdoorClientId`，再为空时使用固定字符串：

```go
candidate-pdf-preview
```

问题：

如果部署环境误配置，preview token 会退回公开固定密钥，导致 token 可伪造。

影响：

- 公共 PDF preview 链接可被伪造
- 如果配合已知资源 URL 或 lesson ID，可能绕过预期访问控制

建议：

- 增加独立环境变量，例如 `PDF_PREVIEW_SIGNING_SECRET`。
- 启动时校验必须存在，不允许 fallback 到固定字符串。
- 至少在 fallback 时拒绝启动，而不是继续运行。

## P3 - 中优先级/体验与扩展性

### 7. 会员等级列表先分页后过滤，total 不准确

位置：

- `candbff/handler/membership.go:14`
- `candbff/handler/membership.go:23`
- `candbff/handler/membership.go:38`

现状：

`ListMembershipPlans` 先调用 `Gmbr.ListMemberships(page, page_size)`，再在 BFF 过滤：

- `is_current`
- `status == ACTIVE/PUBLISHED`

最后返回：

```go
total: len(plans)
```

问题：

如果当前页里有非 current 或非 active 的记录，会导致：

- 当前页不足 page_size
- 后续页里的有效会员等级可能看不到
- total 不是微服务真实 total

建议：

- 最好让 gmbr 支持 `is_current/status` 过滤。
- 如果微服务暂时不支持，BFF 需要拉取足够数据后再做过滤分页。

### 8. 证书列表固定只取 100 条，并且每条证书额外查详情

位置：

- `candbff/handler/certificate.go:13`
- `candbff/handler/certificate.go:27`
- `candbff/handler/certificate.go:32`

现状：

`ListCertificates` 固定 `Page=1, PageSize=100`，并对每条 credential 再调用：

- `GetCredentialDefinitionDetail`
- `GetCredentialDetail`

问题：

- 超过 100 条证书会丢数据。
- N 条证书会产生 `1 + 2N` 次 gRPC 调用。

建议：

- 接入分页参数。
- 让 gcreds 列表接口直接返回展示所需的定义名称和文件摘要，避免 BFF N+1 查询。
- 如果短期不能改微服务，可加本地请求级 cache，减少同一个 `cred_def_ulid` 重复查。

### 9. 资源包文件归属校验会扫描全部资源包和文件

位置：

- `candbff/handler/resource_pack.go:161`
- `candbff/handler/resource_pack.go:170`
- `candbff/handler/resource_pack.go:173`

现状：

`findResourcePackFileForCandidate` 为了确认文件归属，会：

1. 拉考生的所有资源包，page_size 500。
2. 对每个资源包分页拉所有文件，page_size 500。
3. 在 BFF 内存里找 `file_id`。

问题：

资源包多或文件多时，单个文件预览/缩略图请求会很慢，并且放大 glms 调用。

建议：

- 推动 glms 提供 `GetResourcePackFileForCandidate(candidate_ulid, file_id)`。
- 或让 `GetResourcePackFileViewURL` 返回文件 metadata 与授权结果。
- 短期可加请求级缓存，避免同一个请求里重复扫。

### 10. 未读消息计数最多只统计 99 条

位置：

- `candbff/handler/message.go:60`
- `candbff/handler/dashboard.go:38`

现状：

未读数量通过 `ListMessages(Status=UNREAD, Limit=99)` 后取 `len(messages)`。

问题：

如果未读超过 99，会显示 99，而不是真实数量。

建议：

- 推动 gmsg 提供 `CountUnreadMessages`。
- 或使用列表返回的 total，如果 proto 已支持。
- UI 如果只想显示 `99+`，也应明确在 BFF 返回 capped 标记。

### 11. `records` 模块仍是 stub，但路由已暴露

位置：

- `candbff/handler/record.go:13`
- `candbff/handler/record.go:18`
- `candbff/server/router.go:205`

现状：

- `GET /api/records` 返回成功但 data Ϊ nil。
- `POST /api/records` 返回成功 `{status: "ok"}`，但没有任何真实写入。

问题：

这和空订单详情类似，会让前端误以为功能可用。

建议：

- 如果功能暂不可用，返回 `501 Not Implemented`。
- 如果前端不再使用，移除路由。
- 不建议保留假成功响应。

### 12. `GetLessonURL` 返回旧的 `/pdf-preview?...` 路径

位置：

- `candbff/handler/pipeline.go:423`
- `candbff/handler/pipeline.go:443`
- `candbff/server/router.go:111`

现状：

`GET /api/pipeline/lessons/{lessonId}/url` 返回：

```text
/pdf-preview?lessonId=...
```

但当前 router 中实际 BFF PDF preview API 是：

- `/api/pipeline/lessons/{lessonId}/preview`
- `/api/public/pdf-preview/lessons/{lessonId}`

前端现在主要使用 `preview-url` 和 Vue route `/pdf-preview/lessons/:lessonId`。

问题：

这个接口看起来是旧契约，容易让新调用方拿到不可用或不一致的 URL。

建议：

- 如果前端不用，删除该接口。
- 如果仍要保留，返回当前实际可用的 preview URL。

## 已确认对齐/暂不建议误改

### 1. 资格申请订单字段

当前使用：

- `pipeline_cc_ulid`
- `bundle_ulid`
- `qual_ulids`

并调用：

```go
Mall.CreateCredentialApplicationOrder(...)
```

已对照当前 Go module 中最新 gmall proto：`CreateCredentialApplicationOrderRequest` 使用 `BundleUlid`，不是旧文档里的 `bundle_order_ulid`。

结论：

这里目前不应改回 `bundle_order_ulid`。

### 2. 管线解锁订单字段

当前 `UnlockPipelineInBundle` 调用 `CreatePipelineUnlockOrder` 时传入：

- `candidate_ulid`
- `pipeline_cc_ulid`
- `bundle_ulid`

这与当前 proto 对齐。

### 3. 重考流程

当前 `PrepareRetakePayment` 流程基本符合设计：

1. 先用 `GetCourseUnitRetakePaymentStatus` 和 `ListCourseRetakeOrders` 判断是否已有/已支付重考订单。
2. 已支付时调用 `CandidateApplyRetake`。
3. 未支付时调用 `CreateCourseRetakeOrder`。
4. 如果订单返回没有 payment key，再调用 `InitiatePayment`。

另外，`result_status = NO_SHOW` 当前会被当成最终失败结果处理，因为 `result_status` 非空且非 `NONE`，并且 `IsPassed == false`。

### 4. 认证课程访问边界

`GetPipelineCourse` 会先通过考生可访问课程列表校验 `course_id`，再调用 LMS 获取完整课程。课程正文入口目前有考生归属检查。

## 建议修复顺序

1. 修发票归属校验。
2. 修 `resource-preview` SSRF。
3. 修空实现接口：`GetOrder`、`records`。
4. 修订单列表全量聚合。
5. 修资格申请/证书/会员列表分页。
6. 优化资源包文件归属查询和消息未读计数。
7. 清理旧接口 `GetLessonURL`。

## Validation

命令：

```bash
cd candbff
go test ./...
```

结果：

```text
?    candbff          [no test files]
?    candbff/config   [no test files]
?    candbff/handler  [no test files]
?    candbff/server   [no test files]
```
