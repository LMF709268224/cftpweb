# Mall 前端接入说明

本文档给前端实现者使用，描述 `gmall` 相关的购买、解锁、免考选择、资格资料申请、支付、订单状态查询流程。

核心原则：

- 前端展示认证和课程结构时，配置从 `gcc` 查。
- 前端判断当前考生能不能买、是否要先解锁时，先问 `gmall.CheckPipelineEligibility`。
- 前端展示免考选择时，先从 `gcc` 找出可免考 Unit，再从 `gcreds` 判断考生是否已经持有可用资格。
- 前端不能只在本地相信免考选择，`gmall` 会在下单时再次校验资格。
- 缺少免考资格时，先引导考生走 `gmall.CreateCredentialApplicationOrder` 和 `gcreds` 上传/提交资料，审核通过后再回来选择免考。
- 所有支付都走 `gmall.PreviewPayment` 和 `gmall.InitiatePayment`，不要直接调 `gpay`。

## 1. 服务职责

| 服务 | 前端使用场景 |
|---|---|
| `gcc` | 查询 Pipeline/Stage/Unit 配置，判断哪些 Unit 允许免考、对应哪些资格 |
| `gmall` | 创建解锁订单、Pipeline 订单、Stage 订单、补考订单、资格申请订单，预览金额，发起支付，查询订单状态 |
| `gcreds` | 查询考生资格、查询可上传资格定义、请求上传 URL、提交资格资料申请、查看审核状态 |
| `gprog` | 查询考生实际 Pipeline/Stage/CourseUnit 进度，考试失败后申请重考 |

## 2. 常用状态

### 2.1 支付订单总表状态

`orders` 是支付总表，前端一般用于收银台、支付结果页、订单列表。

| 字段 | 状态 | 含义 |
|---|---|---|
| `order_status` | `PENDING` | 支付订单刚创建 |
| `order_status` | `ACTIVE` | 支付订单进行中 |
| `order_status` | `COMPLETED` | 支付订单已完成 |
| `order_status` | `CANCELLED` | 支付订单取消或失败后关闭 |
| `order_status` | `CLOSED` | 支付订单关闭 |
| `payment_status` | `UNPAID` | 尚未支付 |
| `payment_status` | `WAIT_PAY` | 已生成支付凭证，等待用户支付 |
| `payment_status` | `PAID` | 支付成功 |
| `payment_status` | `FAILED` | 支付失败或取消 |
| `payment_status` | `REFUND_OFFLINE` | 线下退款 |

### 2.2 业务订单状态

业务订单状态才是前端引导用户下一步操作的主要依据。

| 业务 | 状态 | 含义 | 前端动作 |
|---|---|---|---|
| Bundle 订单 | `WAIT_BUNDLE_PAYMENT` | 等待支付认证 Bundle 费用 | 展示收银台 |
| Bundle 订单 | `COMPLETED` | Bundle 购买和履约完成 | 跳转学习/进度页或刷新认证状态 |
| Bundle 订单 | `FAILED` | 支付或履约失败 | 展示失败，可重新购买或联系支持 |
| Bundle 订单 | `CANCELLED` | 已取消 | 展示取消 |
| Stage 订单 | `WAIT_EXEMPTION_SELECTION` | 等待选择免考 | 展示免考选择页 |
| Stage 订单 | `WAIT_STAGE_PAYMENT` | 等待支付 Stage 费用 | 展示收银台 |
| Stage 订单 | `COMPLETED` | Stage 候选人动作完成 | 返回进度页等待 Stage 激活 |
| Stage 订单 | `FAILED` | Stage 支付失败 | 展示失败 |
| Stage 订单 | `CANCELLED` | Stage 订单取消 | 展示取消 |
| 补考订单 | `WAIT_RETAKE_PAYMENT` | 等待支付补考费 | 展示收银台 |
| 补考订单 | `COMPLETED` | 补考费已支付 | 调用 `gprog.CandidateApplyRetake` |
| 补考订单 | `FAILED` | 补考支付失败 | 展示失败，可重试 |
| 补考订单 | `CANCELLED` | 补考订单取消 | 展示取消 |
| 解锁订单 | `WAIT_UNLOCK_PAYMENT` | 等待支付解锁费 | 展示收银台 |
| 解锁订单 | `PAID` | 解锁费已支付，Casdoor 授权未完成 | 轮询，不要让用户继续买 |
| 解锁订单 | `COMPLETED` | 解锁完成 | 重新调用 `CheckPipelineEligibility` |
| 解锁订单 | `FAILED` | 解锁支付失败 | 展示失败，可重试 |
| 解锁订单 | `CANCELLED` | 解锁订单取消 | 展示取消 |
| 资格申请订单 | `WAIT_REVIEW_FEE_PAYMENT` | 等待支付资格审核费 | 展示收银台 |
| 资格申请订单 | `UPLOAD_READY` | 可以上传资格资料 | 展示上传资料页 |
| 资格申请订单 | `UNDER_REVIEW` | 资料已提交，审核中 | 展示审核中 |
| 资格申请订单 | `RESOLVED` | 资格申请已结案 | 重新检查资格，回到购买/免考 |
| 资格申请订单 | `FAILED` | 资格申请订单失败 | 展示失败 |
| 资格申请订单 | `CANCELLED` | 资格申请订单取消 | 展示取消 |

## 3. 页面加载时的基础数据

进入认证购买页时，建议并行加载：

1. `gcc.GetPipelineDetail`

用于展示认证结构、Stage、Unit、是否允许免考、免考资格要求。

关键字段：

```text
pipeline_id
name
unlock_quals
unlock_stripe_price_id
stages[].stage_id
stages[].units[].unit_id
stages[].units[].name
stages[].units[].allow_exemption
stages[].units[].exemption_quals
stages[].units[].stripe_price_id
stages[].units[].exemption_stripe_price_id
```

2. `gmall.CheckPipelineEligibility`

用于决定主按钮显示“购买”“解锁”还是“不可操作”。

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "pipeline_cc_ulid": "<pipeline_cc_ulid>",
  "bundle_ulid": "<bundle_ulid>"
}
```

响应：

```json
{
  "eligible": true,
  "can_unlock": false,
  "can_purchase": true,
  "blockers": []
}
```

3. `gcreds.ListCandidateCredentials` 或逐项 `gcreds.CheckCandidateQualification`

用于判断哪些免考项当前可选。

推荐做法：

- 如果页面上可免考 Unit 不多，可以对每个 `exemption_quals` 调 `CheckCandidateQualification`。
- 如果资格很多，可以先 `ListCandidateCredentials`，本地匹配 `cred_def_ulid`。

## 4. 购买入口判断

前端不要直接调用 `CreateBundleOrder` 试错，先调用 `CheckPipelineEligibility`。

```ts
if (res.can_purchase) {
  // 展示 Buy Certification
} else if (res.can_unlock) {
  // 展示 Unlock Certification
} else {
  // 禁用按钮，展示 blockers
}
```

常见 blocker：

| `blocker_type` | 含义 | 前端建议 |
|---|---|---|
| `MISSING_UNLOCK_QUALIFICATION` | 缺少解锁资格 | 展示缺失资格，引导申请资料 |
| `MISSING_CERTS_QUALIFICATION` | 缺少最终取证前置资格 | 展示缺失资格 |
| `ALREADY_PURCHASED` | 已购买过该 Pipeline | 跳转进度页 |
| `IN_PROGRESS_PURCHASE` | 已有进行中的购买订单 | 跳转订单继续支付或处理 |
| `PIPELINE_NOT_FOUND` | Pipeline 配置不存在 | 展示不可购买 |

## 5. 解锁流程

当 `CheckPipelineEligibility.can_unlock = true` 时：

1. 调用 `gmall.CreatePipelineUnlockOrder`

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "pipeline_cc_ulid": "<pipeline_cc_ulid>"
}
```

2. 根据 `order_status` 处理：

| `order_status` | 前端动作 |
|---|---|
| `WAIT_UNLOCK_PAYMENT` | 调 `PreviewPayment`/`InitiatePayment`，`biz_type = PIPELINE_UNLOCK` |
| `PAID` | 支付已成功但授权未完成，轮询 `GetPipelineUnlockOrderSummary` |
| `COMPLETED` | 解锁完成，重新调用 `CheckPipelineEligibility` |
| `FAILED` / `CANCELLED` | 展示失败，可重新创建解锁订单 |

支付：

```json
{
  "biz_type": "PIPELINE_UNLOCK",
  "biz_ref_ulid": "<pipeline_unlock_order_ulid>",
  "success_url": "https://example.com/payment/success",
  "cancel_url": "https://example.com/payment/cancel",
  "coupon_codes": []
}
```

解锁订单的 `PAID` 是中间态，不是完成态。只有 `COMPLETED` 才表示 Casdoor 权限已经授予。

## 6. 免考展示规则

一个 Unit 可以展示为“可免考”，需要同时满足：

- `unit.allow_exemption = true`
- `unit.exemption_quals` 非空

资格判断规则：

- 对于同一个 Unit，`exemption_quals` 是“满足任意一个即可免考”。
- 只要考生持有其中任意一个 `ACTIVE` 资格，前端就可以让该 Unit 的免考复选框可选。
- 如果一个都没有，前端可以展示“申请免考资格”按钮。
- 即使前端判断可选，后端仍会在 `CreateBundleOrder` 或 `SelectStageExemptions` 中重新校验。

建议展示字段：

| 前端字段 | 来源 |
|---|---|
| Unit 名称 | `gcc.UnitConfig.name` |
| 是否支持免考 | `allow_exemption` 和 `exemption_quals` |
| 所需资格名称 | `gcc.Qualification.name_hint` 或 `gcreds.GetCredentialDefinitionDetail.name` |
| 考生是否已满足 | `gcreds.CheckCandidateQualification` 或 `ListCandidateCredentials` |
| 申请材料要求 | `gcreds.GetCredentialDefinitionDetail.file_constraints` |

### 6.1 不要用 `ListCandidateEligibleDefinitions` 发现免考资格

`gcreds.ListCandidateEligibleDefinitions` 不是“列出当前 Pipeline 下可以申请哪些免考资格”的接口。

它的实际含义是：

```text
列出当前考生已经拥有上传权限的资格定义。
```

也就是说，它依赖 Casdoor upload permission。只有当某个资格申请订单已经进入 `UPLOAD_READY`，或审核费支付完成后 `gmall` 给该考生授予了该资格的上传权限时，这个接口才可能返回对应资格。

Pipeline 解锁订单支付完成后，只会授予 Pipeline 解锁权限：

```text
pipeline unlock permission
```

它不会授予免考资格资料上传权限：

```text
credential upload permission
```

所以用户刚支付完 CFTP 解锁订单后，调用 `ListCandidateEligibleDefinitions` 返回空是正常的。

前端发现免考资格的正确入口是：

1. 调 `gcc.GetPipelineDetail`。
2. 从 `stages[].units[]` 找 `allow_exemption = true` 且 `exemption_quals` 非空的 Unit。
3. 把这些 `exemption_quals` 当作可申请的免考资格 ID。
4. 对每个 `qual_id` 调 `gcreds.CheckCandidateQualification` 或 `gcreds.GetLatestCredential`。
5. 如果没有 ACTIVE 资格，展示“申请免考资格”，点击后调 `gmall.CreateCredentialApplicationOrder`。

`ListCandidateEligibleDefinitions` 可以用在已经进入上传资料流程后的辅助页面，例如列出“当前我已经被允许上传哪些资格资料”，但不适合用作 Pipeline 免考资格发现入口。

## 7. 整包或首单购买时的免考流程

适用于：

- `payment_mode = FULL_PIPELINE`
- `payment_mode = BY_STAGE` 且首单 PipelineOrder 包含当前应购买 Stage

### 7.1 展示免考选择

1. 前端调用 `gcc.GetPipelineDetail`，遍历所有 `stages[].units[]`。
2. 对每个可免考 Unit，读取 `exemption_quals`。
3. 调用 `gcreds.CheckCandidateQualification` 判断是否已有 ACTIVE 资格。
4. 已满足资格的 Unit：复选框可选。
5. 未满足资格的 Unit：复选框禁用，展示“申请资格/上传资料”入口。

### 7.2 提交 Bundle 购买订单

当前顶层认证购买入口是 `gmall.CreateBundleOrder`。不要调用旧的 `CreatePipelineOrder`。

没有选择任何免考：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "bundle_cc_ulid": "<bundle_ulid>",
  "payment_mode": "FULL_PIPELINE",
  "bundle_order_ulid": "<bundle_order_ulid>",
  "selected_exemptions_json": "{\"<pipeline_cc_ulid>\":{\"stages\":[]}}"
}
```

选择免考：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "bundle_cc_ulid": "<bundle_ulid>",
  "payment_mode": "FULL_PIPELINE",
  "bundle_order_ulid": "<bundle_order_ulid>",
  "selected_exemptions_json": "{\"<pipeline_cc_ulid>\":{\"stages\":[{\"index\":0,\"stage_cc_ulid\":\"<stage_cc_ulid>\",\"exempted_unit_cc_ulids\":[\"<unit_cc_ulid>\"]}]}}"
}
```

`selected_exemptions_json` 最外层必须是以 `pipeline_cc_ulid` 为 key 的 Map。这样同一个 Bundle 包含多个 Pipeline 时，gmall 才能知道这些免考选择属于哪个 Pipeline。

```json
{
  "<pipeline_cc_ulid>": {
    "stages": [
      {
        "index": 0,
        "stage_cc_ulid": "<stage_cc_ulid>",
        "exempted_unit_cc_ulids": ["<unit_cc_ulid_1>", "<unit_cc_ulid_2>"]
      }
    ]
  }
}
```

### 7.3 创建订单后的状态

| `order_status` | 前端动作 |
|---|---|
| `WAIT_BUNDLE_PAYMENT` | 调 `PreviewPayment`/`InitiatePayment`，`biz_type = BUNDLE_PURCHASE`，`biz_ref_ulid = bundle_order_ulid` |
| `COMPLETED` | 跳转 `gprog.GetPipelineDetail` 进度页或刷新认证状态 |
| `FAILED` / `CANCELLED` | 展示失败 |

### 7.4 资格刚审核通过后的处理

如果用户先创建了 Bundle 订单，后来又完成某个免考资格审核，在支付前应使用同一个 `bundle_order_ulid` 再次调用 `CreateBundleOrder`，传入最新的 `selected_exemptions_json`，让 gmall 基于同一张未支付订单重新计算价格快照。然后重新 `PreviewPayment`。

注意：当前 proto 仍保留 `SyncPipelineOrderExemptions`，但它使用的是旧的 `pipeline_order_ulid`，不适用于新的 Bundle 顶层购买流程。

## 8. 后续 Stage 按阶段购买时的免考流程

适用于 `BY_STAGE` 流程中，Pipeline 已经实例化，后续 Stage 需要单独购买或选择免考。

### 8.1 创建 Stage 订单

调用 `gmall.CreateStageOrder`：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "pipeline_cc_ulid": "<pipeline_cc_ulid>",
  "pipeline_order_ulid": "<pipeline_order_ulid>",
  "stage_ulid": "<stage_instance_ulid>",
  "stage_cc_ulid": "<stage_cc_ulid>"
}
```

返回状态：

| `order_status` | 前端动作 |
|---|---|
| `WAIT_EXEMPTION_SELECTION` | 展示当前 Stage 的免考选择页 |
| `WAIT_STAGE_PAYMENT` | 直接进入支付页，`biz_type = STAGE_PAYMENT` |
| `COMPLETED` | 无需支付，等待 Stage 激活 |
| `FAILED` / `CANCELLED` | 展示失败 |

### 8.2 选择 Stage 免考

当 `CreateStageOrder` 返回 `WAIT_EXEMPTION_SELECTION` 时，前端展示该 Stage 下可免考 Unit。

用户确认后调用 `gmall.SelectStageExemptions`：

```json
{
  "stage_order_ulid": "<stage_order_ulid>",
  "exemptions_json": "{\"stage_cc_ulid\":\"<stage_cc_ulid>\",\"exempted_unit_cc_ulids\":[\"<unit_cc_ulid>\"]}"
}
```

`exemptions_json` 格式：

```json
{
  "stage_cc_ulid": "<stage_cc_ulid>",
  "exempted_unit_cc_ulids": ["<unit_cc_ulid_1>", "<unit_cc_ulid_2>"]
}
```

选择“不免考”时也要提交空数组：

```json
{
  "stage_order_ulid": "<stage_order_ulid>",
  "exemptions_json": "{\"stage_cc_ulid\":\"<stage_cc_ulid>\",\"exempted_unit_cc_ulids\":[]}"
}
```

返回状态：

| `order_status` | 前端动作 |
|---|---|
| `WAIT_STAGE_PAYMENT` | 调 `PreviewPayment`/`InitiatePayment`，`biz_type = STAGE_PAYMENT` |
| `COMPLETED` | Stage 候选人动作完成，返回进度页 |

如果用户选择了未持有 ACTIVE 资格的免考项，后端会返回：

```text
FailedPrecondition: candidate does not satisfy active credentials for course ... exemption
```

前端应提示用户先申请/上传该免考资格。

## 9. 缺少免考资格时的资料申请流程

当用户想免考某个 Unit，但没有满足 `exemption_quals` 中任意一个资格时，前端应引导申请资格。

这里的 `qual_id` 必须从 `gcc.GetPipelineDetail` 返回的 `unit.exemption_quals` 取得。不要先调用 `gcreds.ListCandidateEligibleDefinitions` 来找可申请资格；该接口只返回已经有上传权限的资格，尚未创建资格申请订单时通常为空。

### 9.1 查询资格定义和材料要求

调用：

```text
gcreds.GetCredentialDefinitionDetail
```

请求：

```json
{
  "cred_def_ulid": "<qual_id>"
}
```

响应里看：

```text
name
description
file_constraints
```

`file_constraints` 决定前端要展示哪些上传项。

### 9.2 创建资格申请订单

调用 `gmall.CreateCredentialApplicationOrder`：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "pipeline_cc_ulid": "<pipeline_cc_ulid>",
  "bundle_ulid": "<bundle_ulid>",
  "qual_ulids": ["<qual_id>"]
}
```

也可以一次提交多个缺失资格：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "pipeline_cc_ulid": "<pipeline_cc_ulid>",
  "bundle_ulid": "<bundle_ulid>",
  "qual_ulids": ["<qual_id_1>", "<qual_id_2>"]
}
```

这里的 `bundle_ulid` 是商品 Bundle 配置 ULID。资格申请订单不要求先创建 Bundle 购买订单，因此不要为了申请免考资格提前创建未支付的 `bundle_order_ulid`。

返回状态：

| `order_status` | 前端动作 |
|---|---|
| `WAIT_REVIEW_FEE_PAYMENT` | 先支付审核费 |
| `UPLOAD_READY` | 直接进入上传资料页 |
| `UNDER_REVIEW` | 展示审核中 |
| `RESOLVED` | 审核已结案，重新检查资格 |
| `FAILED` / `CANCELLED` | 展示失败 |

### 9.3 支付资格审核费

如果 `order_status = WAIT_REVIEW_FEE_PAYMENT`：

```json
{
  "biz_type": "CREDENTIAL_APPLICATION",
  "biz_ref_ulid": "<application_order_ulid>",
  "coupon_codes": []
}
```

发起支付：

```json
{
  "biz_type": "CREDENTIAL_APPLICATION",
  "biz_ref_ulid": "<application_order_ulid>",
  "success_url": "https://example.com/payment/success",
  "cancel_url": "https://example.com/payment/cancel",
  "coupon_codes": []
}
```

支付完成后，轮询：

```text
gmall.GetCredentialApplicationOrderSummary
```

等到 `order_status = UPLOAD_READY`。

### 9.4 上传资料

进入上传页前可先确认权限：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "cred_def_ulid": "<qual_id>"
}
```

接口：

```text
gcreds.CheckUploadPermission
```

`granted = true` 才能请求上传 URL 和提交申请。

每个文件上传前，前端先在浏览器计算 SHA256，然后调用：

```text
gcreds.RequestUploadUrl
```

请求：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "cred_def_ulid": "<qual_id>",
  "file_hash": "<sha256_lower_hex>",
  "file_ext": "pdf",
  "content_type": "application/pdf",
  "file_usage": "Supporting Document"
}
```

响应：

```json
{
  "upload_url": "<presigned_put_url>",
  "file_key": "<s3_key>",
  "signed_headers": {
    "...": "..."
  }
}
```

然后前端用 HTTP `PUT` 把文件上传到 `upload_url`，必须带上 `signed_headers`。

### 9.5 提交资料申请

所有文件上传成功后，调用：

```text
gcreds.SubmitApplication
```

请求：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "cred_def_ulid": "<qual_id>",
  "files": [
    {
      "file_hash": "<sha256_lower_hex>",
      "file_name": "support.pdf",
      "file_type": 2,
      "file_ext": "pdf",
      "file_size": 123456,
      "file_usage": "Supporting Document"
    }
  ]
}
```

注意：

- `file_type = 1` 表示图片。
- `file_type = 2` 表示 PDF。
- `file_usage` 必须匹配资格定义里的 `file_constraints[].name`。
- 必填材料必须全部上传。
- 文件必须先通过 `RequestUploadUrl` 上传到 S3，否则提交时会校验失败。

### 9.6 查看审核状态

调用：

```text
gcreds.ListCandidateApplications
```

请求：

```json
{
  "candidate_ulid": "<candidate_ulid>",
  "cred_def_ulid": "<qual_id>",
  "page": 1,
  "page_size": 10
}
```

常见状态：

| 状态 | 含义 | 前端动作 |
|---|---|---|
| `Pending` | 已提交，审核中 | 展示等待审核 |
| `Approved` | 审核通过 | 重新检查资格，返回免考选择 |
| `Rejected` | 审核拒绝 | 展示拒绝原因 |
| `Reupload` | 允许重新上传 | 展示重新上传入口，走 `UpdateApplication` |

审核通过后，调用：

```text
gcreds.CheckCandidateQualification
```

如果返回 `eligible = true`，该 Unit 的免考复选框可以启用。

## 10. 支付流程

所有业务支付都统一：

1. `gmall.PreviewPayment`
2. `gmall.InitiatePayment`
3. 跳转或嵌入 Stripe
4. 回到前端后轮询业务订单状态

### 10.1 PreviewPayment

请求：

```json
{
  "biz_type": "BUNDLE_PURCHASE",
  "biz_ref_ulid": "<bundle_order_ulid>",
  "coupon_codes": ["COUPON1"]
}
```

`biz_type` 可选：

```text
BUNDLE_PURCHASE
STAGE_PAYMENT
COURSE_RETAKE_PAYMENT
PIPELINE_UNLOCK
CREDENTIAL_APPLICATION
```

响应金额单位是 minor units，例如 USD cents。

### 10.2 InitiatePayment

请求：

```json
{
  "biz_type": "BUNDLE_PURCHASE",
  "biz_ref_ulid": "<bundle_order_ulid>",
  "success_url": "https://example.com/payment/success",
  "cancel_url": "https://example.com/payment/cancel",
  "coupon_codes": ["COUPON1"]
}
```

响应：

```json
{
  "payment_key": "<stripe_checkout_url_or_client_secret>",
  "pay_order_ulid": "<pay_order_ulid>",
  "message": "payment initiated"
}
```

前端拿到 `payment_key` 后进入 Stripe 支付。

### 10.3 支付成功后不要立即假设业务完成

支付成功页应该轮询对应业务订单：

| 支付类型 | 轮询接口 | 完成状态 |
|---|---|---|
| `BUNDLE_PURCHASE` | `GetBundleOrderSummary` | `COMPLETED` |
| `STAGE_PAYMENT` | `GetStageOrderStatus` | `COMPLETED` |
| `COURSE_RETAKE_PAYMENT` | `GetCourseRetakeOrderStatus` | `COMPLETED` |
| `PIPELINE_UNLOCK` | `GetPipelineUnlockOrderSummary` | `COMPLETED` |
| `CREDENTIAL_APPLICATION` | `GetCredentialApplicationOrderSummary` | `UPLOAD_READY` 或后续状态 |

Stripe webhook 是异步的，成功页刚回来时订单可能仍是 `WAIT_*`，这是正常情况。

## 11. 补考支付流程

考试失败后，`gprog` 会把 CourseUnit 状态置为 `EXAM_FAILED`。如果该 Unit 配了补考费，前端需要先支付补考订单。

流程：

1. `gprog.ValidateRetakeEligibility`
2. `gmall.CreateCourseRetakeOrder`
3. `gmall.PreviewPayment`，`biz_type = COURSE_RETAKE_PAYMENT`
4. `gmall.InitiatePayment`，`biz_type = COURSE_RETAKE_PAYMENT`
5. 轮询 `gmall.GetCourseRetakeOrderStatus` 到 `COMPLETED`
6. 调用 `gprog.CandidateApplyRetake`
7. 调用 `gprog.CandidateSignupExam`

如果直接调用 `CandidateApplyRetake` 返回：

```text
retake payment for course unit ... has not been completed
```

说明应先走补考支付流程。

## 12. 前端推荐页面流程

### 12.1 认证详情页

1. `gcc.GetPipelineDetail`
2. `gmall.CheckPipelineEligibility`
3. 展示：
   - `can_purchase`：显示购买按钮
   - `can_unlock`：显示解锁按钮
   - 否则展示 blockers
4. 同时解析可免考 Unit，调用 `gcreds` 判断每个 Unit 的免考资格状态。

### 12.2 免考选择页

1. 展示所有可免考 Unit。
2. 已持有 ACTIVE 资格的 Unit，允许勾选。
3. 未持有资格的 Unit，展示“申请免考资格”。
4. 用户确认后：
   - Bundle 购买：提交 `CreateBundleOrder(selected_exemptions_json)`，`biz_type = BUNDLE_PURCHASE`
   - Stage 购买：提交 `SelectStageExemptions(exemptions_json)`

### 12.3 资格申请页

1. `gcreds.GetCredentialDefinitionDetail`
2. `gmall.CreateCredentialApplicationOrder`，`qual_ulids` 使用 `gcc.GetPipelineDetail` 中对应 Unit 的 `exemption_quals`，并传入当前商品 Bundle 的 `bundle_ulid`
3. 如果 `WAIT_REVIEW_FEE_PAYMENT`，先支付审核费。
4. 到 `UPLOAD_READY` 后：
   - `gcreds.RequestUploadUrl`
   - HTTP PUT 上传文件
   - `gcreds.SubmitApplication`
5. 审核通过后回到免考选择页，重新检查资格。

### 12.4 收银台页

1. `gmall.PreviewPayment`
2. 展示金额、优惠、税费。
3. 用户确认后 `gmall.InitiatePayment`
4. 使用 `payment_key` 进入支付。
5. 支付返回后轮询对应业务订单。

## 13. 常见错误处理

| 错误 | 含义 | 前端处理 |
|---|---|---|
| `pipeline ... requires unlock` | Pipeline 还没解锁 | 先走解锁流程 |
| `candidate does not meet unlock qualifications` | 缺解锁资格 | 展示缺失资格，引导资料申请 |
| `candidate does not satisfy active credentials for course ... exemption` | 选择了未满足资格的免考 | 引导申请该免考资格 |
| `expected WAIT_EXEMPTION_SELECTION` | Stage 订单不在免考选择状态 | 重新查询 Stage 订单状态 |
| `expected WAIT_STAGE_PAYMENT` | Stage 订单不在可支付状态 | 重新查询订单状态 |
| `expected WAIT_BUNDLE_PAYMENT` | Bundle 订单不在可支付状态 | 重新查询订单状态 |
| `retake payment ... has not been completed` | 补考费未支付 | 走补考支付流程 |
| `candidate already has an in-progress ... order` | 已有进行中订单 | 跳转继续处理旧订单 |

## 14. 关键注意事项

- `pipeline_cc_ulid`、`stage_cc_ulid`、`course_unit_cc_ulid` 都是配置层 ID，来自 `gcc`。
- `pipeline_ulid`、`stage_ulid`、`course_unit_ulid` 是运行实例 ID，来自 `gprog`。
- 顶层购买使用 `CreateBundleOrder`，后续支付的 `biz_ref_ulid` 是 `bundle_order_ulid`，`biz_type = BUNDLE_PURCHASE`。
- `CreateBundleOrder.bundle_cc_ulid`/`ListBundleOrders.bundle_ulid` 指的是商品 Bundle 配置 ULID，不是运行时 Pipeline ULID。
- `CreateStageOrder` 的 `biz_ref_ulid` 后续是 `stage_order_ulid`，不是 `stage_cc_ulid`。
- 免考选择传的是配置层 Unit ID：`unit_id` / `course_unit_cc_ulid`。
- 免考资格必须是 `ACTIVE`，申请中、拒绝、过期都不能用于免考。
- `exemption_quals` 对同一个 Unit 是“满足任意一个即可”。
- `ListCandidateEligibleDefinitions` 不是免考资格发现接口；免考资格发现以 `gcc.GetPipelineDetail` 的 `exemption_quals` 为准。
- 解锁资格 `unlock_quals` 是逐项要求，缺失项会阻止解锁。
- 支付完成后要轮询业务订单，不要只看 Stripe 返回页。
- `PIPELINE_UNLOCK` 的 `PAID` 是中间态，`COMPLETED` 才能继续购买。
