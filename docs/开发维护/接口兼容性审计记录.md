# 猜测式兜底与接口契约扫描记录

日期：2026-07-08

## 背景

之前在 admin 商品发布流程里，为了绕过 `PublishBundle` 的校验，曾经把 `pricing_json.units[0].access` 的价格复制到 `pricing_json.unlocks[pipeline_ulid]`。这个做法虽然让发布接口暂时通过，但改变了商品发布后的业务结构，导致 cand 端把普通 Bundle 购买引导成“先解锁认证，再购买商品”的双订单流程。

这个问题的根因不是“前端缺一个保底”，而是接口契约没有弄清楚：

1. `pricing_json.units[].access` 和 `pricing_json.unlocks[pipeline_ulid]` 的语义不同。
2. admin 前端不应该为了通过发布校验，静默改写持久化业务 JSON。
3. 如果接口不通，应该先确认是我们调用方式不对、契约变更，还是微服务接口有问题。

已在根目录 `AGENTS.md` 追加项目规则：禁止猜测式 fallback / auto-fill；禁止静默复制语义不同的业务字段；遇到契约不清先记录请求、响应和状态，再找微服务确认。

## 扫描范围

本次重点扫描：

1. `adminbff`
2. `candbff`
3. `adminweb/vue-web/src`
4. `candweb/vue-web/src`

重点关键词：

1. `fallback` / `兜底` / `补齐` / `auto-fill`
2. `silently` / `suppress` / `ignore`
3. `default` / `can_purchase` / `can_unlock`
4. `pricing_json` / `items_json` / `unlocks`
5. `ulid.Make()` / 自动生成 ID
6. 可能会吞掉微服务错误或自动改写请求的逻辑

## 当前结论

没有在 `adminbff` / `candbff` 里发现类似“为了让下游接口通过，静默改写持久化业务 JSON”的确认问题。

`adminbff/handler/bundle.go` 当前商品相关接口基本是直转微服务：

1. `UpdateBundlePricing` 直接调用 `UpdateBundleStructure`。
2. `DuplicateBundle` 直接调用 `DuplicateBundleDraft`，BFF 只生成新 `bundle_ulid` 并传入名称。
3. `PublishBundle` 直接调用 `PublishBundle`，不再在 BFF 或前端发布阶段自动补 `pricing_json.unlocks`。

这说明之前那类“用购买价补解锁价”的行为目前没有在 BFF 层继续存在。

## 已确认保留的合理逻辑

这些逻辑看起来是 fallback，但语义上不是“绕过接口契约”：

1. 分页参数、页面大小、空展示文案的默认值：属于 UI 和查询参数默认值，不改写业务数据。
2. `adminbff` 创建 Bundle 草稿时生成 `bundle_ulid`：微服务接口要求传入新 ID，且前端已经不要求管理员手填。
3. `adminbff` 创建资源文件时生成 `file_id`：同样是服务端生成 ID，不让管理员手填。
4. cand 商品卡片里 thumbnail、标题等展示 fallback：只影响展示，不写回业务 JSON。

## 高风险点

### 1. Bundle 替换绑定认证时，课程单元 ID 按数组顺序映射

位置：`adminweb/vue-web/src/pages/BundlesPage.vue`

当前 `replacePipelineBindingInForm()` 的处理大致是：

1. 把 `items_json` 里的旧 pipeline ID 替换成新 pipeline ID。
2. 获取旧 pipeline 和新 pipeline。
3. 读取两个 pipeline 的课程单元 ID 列表。
4. 按数组下标建立 `oldUnitId -> newUnitId` 映射。
5. 用这个映射替换 `pricing_json.units[].unit_id`。

风险：

1. 如果旧认证和新认证的课程单元顺序完全一致，这个逻辑能工作。
2. 如果新旧认证课程单元数量、顺序、含义不一致，它可能把价格绑定到错误的课程单元。
3. 这不是像 `unlocks` 那样的发布保底，但它仍然是一个“前端猜测业务结构关系”的逻辑。

建议：

1. 不要继续扩大这个逻辑。
2. 和微服务确认是否有“克隆商品后替换 pipeline 引用”的官方接口，或是否能由微服务根据旧/新 pipeline 的结构做安全映射。
3. 如果必须前端配置，应改成管理员显式选择单元映射，而不是按数组顺序自动猜。

### 2. cand 购买弹窗存在“没有初始 eligibility 时默认可购买”的展示兜底

位置：`candweb/vue-web/src/components/PurchaseDialog.vue`

当前逻辑在没有初始状态时，会用：

```ts
{ can_purchase: true, can_unlock: false, blockers: [] }
```

风险：

1. 这不会写回业务数据，也不会改变服务端价格结构。
2. 但如果详情加载失败或 eligibility 缺失，UI 可能短暂显示可购买入口。
3. 真正创建订单仍然会调用 BFF/微服务，失败会由后端拦截。

建议：

1. 这个不属于本次 `pricing_json` 类严重问题。
2. 后续可以把默认状态改成“未知/加载中/请刷新”，避免 UI 误导。
3. 修改前需要确认 cand 端无登录、公开商品浏览、会员商品这几种场景对默认购买状态的要求。

### 3. candbff 商品 enrichment 在无候选人上下文时默认可购买

位置：`candbff/handler/mall.go`

`defaultBundleEligibility()` 返回可购买，主要用于没有候选人上下文或公开商品列表展示。

风险：

1. 有候选人上下文时，会调用 `CheckPipelineEligibility`；如果失败，会返回不可用 blocker。
2. 没有候选人上下文时默认可购买可能适合公开商城，但不适合需要严格资格判断的页面。

建议：

1. 保留现状，不作为 bug 直接修改。
2. 如果后续发现登录态页面也走到了无候选人上下文，需要补链路排查，而不是简单把默认值改成 false。

## 未发现类似问题的区域

### adminbff

未发现商品发布、资源包状态、管线配置等接口里有明显“下游报错后自动补字段再重试”的逻辑。

需要注意：`adminbff/handler/response.go` 仍会把部分 `INVALID_REQUEST` / `PRECONDITION_FAILED` 的微服务 message 返回给前端。这个是错误展示策略问题，不是业务 JSON 兜底问题；如果要统一用户友好提示，应单独处理错误映射。

### candbff

未发现 candbff 会为了绕过接口校验而写入或改写商品、价格、管线结构。

存在一些 enrichment 失败后降级展示的逻辑，例如缩略图、会员信息、payment preview、active order 查询。这些目前只影响页面展示，不会修改持久化业务数据。

### adminweb

未发现发布 Bundle 时继续自动补 `pricing_json.unlocks` 的逻辑。

`parseJsonSilently()` 只用于详情预览和摘要计算；保存和发布前走 `parseJson()` / `validateStructureJson()`，会显示 JSON 错误，不属于静默修复。

### candweb

未发现 candweb 会改写服务端业务 JSON。

主要风险仍是 UI 侧用默认 eligibility 展示购买入口，建议后续按产品交互单独优化。

## 后续处理原则

以后遇到接口不通，必须按这个顺序处理：

1. 记录接口名称、请求 payload、响应 code/message、关键数据库或页面状态。
2. 对照 proto / 微服务说明确认字段语义。
3. 如果字段语义不清，先形成问题文档给微服务确认。
4. 只有确认是前端/BFF 调用错误，才修改我们的请求。
5. 只有确认微服务明确要求兼容字段，才允许加兼容逻辑，并且必须写清楚范围和退出条件。
6. 不允许为了让按钮“能点通”而自动补价格、ID、状态、资格等业务字段。

## 当前建议

1. 先拿 `docs/bundle-publish-unlocks-question.md` 给微服务确认 `pricing_json.unlocks` 的发布校验是否合理。
2. 暂时不要在 admin 发布流程恢复任何自动补 `unlocks` 的逻辑。
3. 下一步如果要修 Bundle 替换认证流程，优先和微服务确认是否有官方替换接口；没有的话，再设计显式单元映射 UI。
