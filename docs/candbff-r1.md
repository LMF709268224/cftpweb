# candbff Code Review — R1

> 2026-06-26 | 10 findings | ranked by severity

---

## #1 角色名大小写敏感导致登录被拒绝

- **文件:** `candbff/handler/auth.go:192`
- **严重程度:** CRITICAL

```go
if role.Name == studentRole {
```

Go 的 `==` 是区分大小写的。如果 Casdoor 里角色存的是 `"Role_Student_Basic"`，而环境变量 `ROLE_STUDENT_BASIC` 设的是 `"role_student_basic"`（或者反过来），这个判断永远为 `false`。结果是所有学生都无法登录，提示 `"only cftp students are allowed to login"`，没有任何诊断日志。

代码库其他地方（exam.go、mall.go、pipeline.go 等）都已经用了 `strings.EqualFold` 或 `strings.ToUpper` 做大小写不敏感比较，唯独这里遗漏了。

**修复方向:** `role.Name == studentRole` → `strings.EqualFold(role.Name, studentRole)`。

---

## #2 completed/totalAmount 是页面级，totalOrders 是全局级 — API 返回不一致

- **文件:** `candbff/handler/payment.go:68-69, 99-105`
- **严重程度:** HIGH

```go
totalOrders := int(resp.GetTotal())   // ← 全局匹配总数

for _, item := range resp.GetItems() { // ← 只遍历当前页
    if statusStr == "completed" {
        completed++        // ← 页面内完成数
        totalAmount += amount  // ← 页面内金额
    }
}
```

API 返回 `{"total_orders": 100, "completed": 3, "total_amount": 45.0}`。前端看到 `completed: 3` 以为是全局有 3 笔已完成，实际只是当前这一页 10 条里有 3 条。翻到第 2 ҳ `completed` 变成 7，第 3 页变成 2 —— 同一个数字三种含义，前端无法正确展示。

`completed` 和 `totalAmount` 要么去掉（让前端自己算），要么应反映全局值。

---

## #3 CanViewInvoice 永远为 true

- **文件:** `candbff/handler/payment.go:94`
- **严重程度:** MEDIUM

```go
CanViewInvoice: strings.TrimSpace(item.GetOrderUlid()) != "",
```

只要订单存在，`OrderUlid` 就一定非空。所以不管是 pending、cancelled、failed 还是 processing 的订单，`can_view_invoice` 都是 `true`。

用户看到已取消的订单旁边有「查看发票」按钮，点进去拿到 404 或报错。

**修复方向:** 根据支付状态判断，只有已支付完成的订单才允许查看发票。

---

## #4 int32 溢出 — page 超大时 offset 变负数

- **文件:** `candbff/handler/payment.go:52,59`
- **严重程度:** MEDIUM

```go
page := parsePositiveIntQuery(r, "page", 1)   // ← 无上限
pageSize := parsePositiveIntQuery(r, "page_size", ...)  // ← 上限 50

offset := (page - 1) * pageSize   // int 运算
Offset: int32(offset),            // ← ǿת int32
```

传 `?page=50000000` ʱ:

```
offset = 49,999,999 × 50 = 2,499,999,950
int32 最大值 = 2,147,483,647
→ 溢出变成 -1,794,967,346
```

负数 offset 发到 gRPC 后端，行为未定义。

**修复方向:** `parsePositiveIntQuery` 加上限，或 offset 用 int64 算完再安全裁切。

---

## #5 订单产品名只显示原始 ULID，不解析为可读名称

- **文件:** `candbff/handler/payment.go:80, 175-180`
- **严重程度:** MEDIUM

```go
name := orderProductName(item.GetBizType(), item.GetBizRefUlid())
// → "Pipeline Order - 01ARZ3NDKW..."
```

```go
func orderProductName(bizType string, bizRefULID string) string {
    label := orderBizTypeLabel(bizType)
    return label + " - " + strings.TrimSpace(bizRefULID)
}
```

旧代码会调用 `pipelineName()`（通过 Gcc.GetPipeline gRPC）解析 ULID 为实际的流水线名称（如 "AWS Certified Solutions Architect"），新代码只拼了一个静态标签加原始 ULID。

用户看到 `"Pipeline Order - 01ARZ3NDKW..."`，有多个流水线订单时全是魔鬼数字，完全分不清。ULID 对用户毫无意义，所有给 web 展示的都不应该暴露 ULID。

另外 `pipelineName` 函数仍在文件底部（第 202 行），已成为死代码。

---

## #6 IsCftpStudent 遍历 Roles 时未检查 nil 元素

- **文件:** `candbff/handler/auth.go:191-192`
- **严重程度:** LOW

```go
for _, role := range user.Roles {   // Roles 类型是 []*Role
    if role.Name == studentRole {   // 如果 role 是 nil 指针 → panic
```

`user.Roles` 是 `[]*casdoorsdk.Role`（指针切片）。Go 的 `range` 遍历 nil 切片是安全的（0 次迭代），但如果切片里有 nil 元素，`role.Name` 就会 nil 指针解引用 panic，整个请求直接崩掉，连 JSON 错误响应都发不出来。

正常 Casdoor 不会返回含 null 的 roles 数组，但服务端升级、bug、或将来有代码手动构造 User 结构体时可能触发。

**修复方向:** 循环内加 `if role == nil { continue }`。

---

## #7 candidateOrderStatus 子串匹配可能误分类状态

- **文件:** `candbff/handler/payment.go:145-148`
- **严重程度:** LOW

```go
if strings.Contains(orderStatus, "PAID") {    // "UNPAID" 也命中
    return "completed"
}
if strings.Contains(orderStatus, "FAILED") {
    return "cancelled"
}
```

`strings.Contains("UNPAID", "PAID")` 返回 `true`，会把未支付的订单标为 `"completed"`。

`order_status` 是 proto 里的 free-form string（不是 enum），后端将来完全可能引入 `UNPAID`。`payment_status` 字段的注释里就已经用了 `UNPAID` 这个值。同类问题：`"UNSUCCESSFUL"` 也会命中 `"SUCCESS"`。

---

## #8 补充材料 fallback 被移除

- **文件:** `candbff/handler/pipeline.go:337`
- **严重程度:** LOW

旧代码在 `GetCompleteCourse` 没返回补充材料时有个兜底：

```go
// 旧代码（已删除）
if completeCourse.GetSupplementaryMaterial() == nil {
    suppResp, err := h.Lms.GetCourseSupplementaryMaterialAdmin(...)
    if err == nil && suppResp != nil {
        completeCourse.SupplementaryMaterial = suppResp.GetMaterial()
    } else if err != nil {
        slog.Warn("failed to load supplementary material", ...)
    }
}
```

新代码只有一行注释 `// Supplementary material is now included in GetCompleteCourse response`，删掉了 fallback 和 warn 日志。

proto 里 `SupplementaryMaterial` 是 `omitempty` 指针字段，nil 完全合法。如果 glms candidate 路径 `GetCompleteCourse` 某些情况下没填这个字段，学生就静默看不到补充材料，没有任何诊断信号。

---

## #9 quizProgressByCourse 用管理员 API 查考生数据

- **文件:** `candbff/handler/pipeline.go:1107`
- **严重程度:** LOW

```go
resp, err := h.Lms.ListQuizAttemptsAdmin(r.Context(), &lmspb.ListQuizAttemptsRequest{
    QuizUlid: quizID,
    UserUlid: candidateID,
    PageSize: 20,
})
```

glms proto 里 quiz 相关的查询只有 `ListQuizAttemptsAdmin` 这一个 RPC，没有考生端版本。代码传了 `UserUlid=candidateID` 来限范围，TODO 注释也写明是临时方案。

但 `*Admin` 后缀意味着这是管理员接口。candbff 其他所有地方都用的是考生端 RPC（`GetCourseSummaryCandidate`、`ListCourseMaterialsCandidate` 等），唯独这里跨了边界。如果 glms 将来给 `ListQuizAttemptsAdmin` 加了 RBAC 校验，考生端请求会被拒绝，quiz 进度全部变空。

**修复方向:**  看看这个接口是必须调用的吗？调用这个接口是为了展示数据，还是有作为什么判断用的？

---

## #10 死代码：pipelineName 和 parseOrderTime

- **文件:** `candbff/handler/payment.go:165-173, 202-218`
- **严重程度:** LOW

重构 `ListOrders` 删掉了所有调用者，但这两个函数还留着：

```go
// 第 165 行 — 只被已删除的 sort 逻辑调用
func parseOrderTime(createdAt string) time.Time { ... }

// 第 202 行 — 只被已删除的 buildXxxOrderItems 调用
func (h *Handler) pipelineName(r *http.Request, pipelineULID string, cache map[string]string) string {
    // 内部还有 gRPC 调用 h.Gcc.GetPipeline
}
```

全代码库搜不到任何调用点。`pipelineName` 还拖着一个 `gccpb` import（payment.go 第 8 行），这个 import 也只被它用。

**修复方向:** 删除这两个函数及不再需要的 `gccpb` import。
