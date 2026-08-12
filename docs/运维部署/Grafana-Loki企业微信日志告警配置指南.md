# Grafana + Loki 企业微信日志告警配置指南

## 1. 目标和告警链路

本文用于配置以下告警链路：

```text
Kubernetes Pod 日志 -> Vector -> Loki -> Grafana 告警规则 -> 企业微信群机器人
```

当前第一阶段规则的目标是：Grafana 每分钟检查一次 Loki；最近 5 分钟出现至少一条
`ERROR` 日志并持续满足 1 分钟时，向企业微信群发送告警。

该规则只读取日志，不调用业务接口，也不会修改业务数据。

## 2. 从零开始：进入告警规则页面

### 2.1 打开新建告警规则页面

登录 Grafana 后，按下面的位置进入：

```text
Grafana 左侧菜单
-> Alerting
-> Alert rules
-> New alert rule
```

进入后，页面应依次显示以下区域：

```text
1. Enter alert rule name
2. Define query and alert condition
3. Add folder and labels
4. Set evaluation behavior
5. Configure notifications
6. Configure notification message
```

下面严格按照这 6 个区域填写。

## 3. 页面第 1 区：填写规则名称

位置：页面顶部的 `1. Enter alert rule name`。

找到 `Name` 输入框，填写：

```text
CFTP 服务 ERROR 日志告警
```

填完后直接向下滚动，不需要单独保存。

## 4. 页面第 2 区：填写 Loki 查询和触发条件

位置：`2. Define query and alert condition`。

### 4.1 选择数据源和查询模式

在这个区域顶部依次设置：

| 页面位置/字段 | 点击或填写的内容 |
| --- | --- |
| 左上角数据源下拉框 | 选择 `loki` |
| 查询编辑器右上角 | 点击 `Code`，不要使用 `Builder` |
| `Options -> Type` | 选择 `Instant` |
| 顶部查询时间 | 保持 `10m to now` |

### 4.2 填写查询

删除查询编辑框里的原内容，完整粘贴：

```logql
(
  sum(count_over_time({level="ERROR"}[5m]))
  or on() vector(0)
)
```

然后点击查询编辑器右上方的 `Run queries`。

### 4.3 填写 Alert condition

在查询框下方找到 `Alert condition`，按下面设置：

| 页面字段 | 设置值 |
| --- | --- |
| 第一个下拉框 | `WHEN QUERY` |
| 第二个下拉框 | `IS ABOVE` |
| 右侧数字输入框 | `0` |

最终这一行应显示：

```text
WHEN QUERY IS ABOVE 0
```

点击蓝色按钮 `Preview alert rule condition`。

看到下面任一结果都说明配置正确：

```text
数值 0 + 绿色 Normal：当前最近 5 分钟没有 ERROR
数值大于 0 + Pending/Alerting：当前最近 5 分钟存在 ERROR
```

你当前截图显示 `0 / Normal`，所以页面第 1、2 区已经配置正确，不需要修改。

`or on() vector(0)` 不应删除。它确保没有匹配日志时返回数值 `0`，而不是 `No data`。

### 4.4 当前查询范围

当前 `{level="ERROR"}` 会统计 Loki 中所有带有 `level="ERROR"` 标签的日志。首次配置时
可以先使用它打通告警链路。

正式使用前，建议在查询编辑器中点击 `Label browser`，查看是否存在 `namespace` 标签。
如果存在值 `cftp-test`，将查询改为：

```logql
(
  sum(count_over_time({namespace="cftp-test", level="ERROR"}[5m]))
  or on() vector(0)
)
```

如果实际标签叫 `namespace_name` 或 `kubernetes_namespace_name`，使用 Label browser 显示的
真实名称替换 `namespace`。修改后必须再次点击 `Run queries` 和
`Preview alert rule condition`，确认结果为数值且没有语法错误。

## 5. 页面第 3 区：创建 Folder 并添加 Labels

位置：`3. Add folder and labels`。

### 5.1 创建并选择 Folder

1. 在 `Folder` 一行点击右侧的 `New folder`。
2. 弹窗中的名称填写：

```text
CFTP 运维告警
```

3. 点击弹窗里的创建/确认按钮。
4. 回到规则页面后，检查 `Folder` 下拉框已显示 `CFTP 运维告警`；如果没有，手动从
   `Select folder` 下拉框中选择它。

Folder 只用于整理 Grafana 告警规则，不是 Kubernetes Namespace。

### 5.2 逐个添加 Labels

1. 在 `Labels` 一行点击 `Add labels`。
2. 在弹出的 Key/Value 输入区域逐个添加下面 4 组值。
3. 每填写一组后点击 `Add`，或者按当前界面的确认按钮保存该标签。

| Key | Value | 用途 |
| --- | --- | --- |
| `severity` | `critical` | 表示需要立即关注 |
| `environment` | `test` | 表示测试环境 |
| `source` | `loki` | 表示告警来自日志 |
| `team` | `cftp` | 便于后续通知路由和静默 |

这些标签不是 Loki 查询条件，不会改变查询结果；它们用于搜索告警、静默告警和通知路由。

第 3 区最终应显示：

```text
Folder: CFTP 运维告警
Labels: severity=critical, environment=test, source=loki, team=cftp
```

## 6. 页面第 4 区：配置检查周期

位置：`4. Set evaluation behavior`。必须先完成第 3 区的 Folder 选择，才能配置这一块。

### 6.1 创建并选择 Evaluation group

1. 在 `Select an evaluation group...` 右侧点击 `New evaluation group`。
2. 弹窗中的 Group name 填写：

```text
cftp-log-errors
```

3. 弹窗中的 Evaluation interval 填写或选择：

```text
1m
```

4. 点击创建/确认按钮。
5. 回到页面后，检查下拉框已选择 `cftp-log-errors`。

这表示 Grafana 每 1 分钟执行一次 Loki 查询。

### 6.2 设置 Pending period

在 `Pending period` 下方：

1. 点击快捷按钮 `1m`；或者在输入框填入 `1m`。
2. 确认输入框最终显示 `1m`。

### 6.3 设置 Keep firing for

在 `Keep firing for` 下方：

1. 点击快捷按钮 `None`。
2. 确认输入框显示 `0s`。

| 配置项 | 设置值 | 含义 |
| --- | --- | --- |
| Pending period | `1m` | 条件持续满足 1 分钟后才进入告警状态 |
| Keep firing for | `0s` / `None` | 条件恢复后立即结束告警 |

因此，一条 ERROR 日志出现后，最多等待约 1 至 2 分钟收到通知。该日志离开 5 分钟查询
窗口后，规则会恢复为 Normal。

### 6.4 设置 No data 和 Error handling

在 `Keep firing for` 下方点击并展开 `Configure no data and error handling`，设置：

| 场景 | 设置值 | 原因 |
| --- | --- | --- |
| Alert state if no data | `Normal` | 正常无错误日志不应告警；查询本身也已用 `vector(0)` 兜底 |
| Alert state if execution error or timeout | `Alerting` | Loki 查询失败时也需要通知，避免监控失效却无人发现 |

如果查询执行错误触发告警，先检查 Loki 数据源和 Grafana 到 Loki 的网络，而不是应用日志。

## 7. 先创建企业微信 Contact point

在填写页面第 5 区之前，需要先创建企业微信接收方。如果 Contact point 下拉框里已经能选到
`CFTP-WeCom`，直接跳到第 8 节。

### 7.1 在企业微信创建群机器人

1. 打开用于接收告警的企业微信群。
2. 进入群设置，选择 `群机器人`。
3. 添加机器人，名称建议填写 `CFTP 系统告警`。
4. 保存机器人生成的 Webhook 地址。

Webhook 地址格式类似：

```text
https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=REDACTED
```

Webhook 中的 `key` 是密钥。禁止写入 Git、Markdown、截图或公开聊天记录。

### 7.2 从当前规则页面打开 Contact points

1. 找到规则页面的 `5. Configure notifications`。
2. 在 `Contact point` 下拉框右侧，按住 `Ctrl` 点击 `View or create contact points`，让它
   在新标签页打开，避免当前尚未保存的规则内容丢失。
3. 也可以从 Grafana 左侧菜单进入：

```text
Alerting
-> Notification configuration
-> Contact points
```

### 7.3 在 Grafana 创建 WeCom Contact point

1. 在 Contact points 页面点击 `New contact point`。
2. Name 填写：

```text
CFTP-WeCom
```

3. 在 `Integration` 下拉框选择 `WeCom`。
4. 找到 `URL` 输入框，填入企业微信群机器人的完整 Webhook 地址。
5. 暂时保留默认消息模板。
6. 点击页面上的 `Test` 或 `Test contact point`。
7. 打开企业微信群，确认收到 Grafana 测试通知。
8. 回到 Grafana，点击 `Save contact point`。
9. 关闭该浏览器标签页，返回尚未保存的告警规则页面。

如果 Integration 列表没有 `WeCom`，不要直接把企业微信 URL 填进普通 Webhook 后保存。
不同 Grafana 版本的普通 Webhook 消息体可能不符合企业微信格式，应先升级 Grafana 或部署
专用的消息格式转换服务。

## 8. 页面第 5 区：选择通知接收方

位置：规则页面的 `5. Configure notifications`。

1. Alertmanager 保持 `grafana`。
2. 点击 `Contact point` 下拉框。
3. 选择：

```text
CFTP-WeCom
```

4. 如果下拉框中没有它，刷新下拉选项；不要刷新整个规则页面，以免未保存内容丢失。
5. 点击并展开 `Muting, grouping and timings (optional)`。
6. 如果展开后出现下面三个输入框，按表格填写：

| 配置项 | 设置值 | 含义 |
| --- | --- | --- |
| Group wait | `30s` | 首次触发后等待 30 秒，将同时发生的告警合并 |
| Group interval | `5m` | 同一告警组的新增变化最多每 5 分钟通知一次 |
| Repeat interval | `2h` | 问题持续存在时每 2 小时提醒一次 |

如果这三个输入框没有显示，保持默认值即可；它们也可以在
`Alerting -> Notification policies` 中统一配置。不要把 Repeat interval 设置成 `1m`，否则
持续错误可能频繁刷群。

## 9. 页面第 6 区：填写通知文案

位置：`6. Configure notification message`。

在 `Summary (optional)` 文本框填写：

```text
CFTP 测试环境最近 5 分钟检测到 ERROR 日志
```

在 `Description (optional)` 文本框填写：

```text
Loki 最近 5 分钟 ERROR 日志数量大于 0。请打开 Grafana Explore，选择 Loki，查询最近 15 分钟的 ERROR 日志并按 Pod 和服务定位原因。
```

`Runbook URL (optional)` 输入框留空。当前没有专用在线排障页面，不要填写无效地址。

`Add custom annotation` 和 `Link dashboard and panel` 暂时都不用点。

当前告警发送的是错误数量、规则状态和 Grafana 链接，不会自动附带完整日志内容。这能避免
Token、用户信息或业务数据被转发到企业微信群。详细错误必须回到 Grafana Explore 查看。

## 10. 页面底部：保存并验证

### 10.1 保存前逐项检查

检查以下配置：

| 页面区域 | 字段/位置 | 最终值 |
| --- | --- | --- |
| `1. Enter alert rule name` | Name | `CFTP 服务 ERROR 日志告警` |
| `2. Define query and alert condition` | Data source | `loki` |
| `2. Define query and alert condition` | Mode / Type | `Code` / `Instant` |
| `2. Define query and alert condition` | Query | `sum(count_over_time({level="ERROR"}[5m])) or on() vector(0)` |
| `2. Define query and alert condition` | Alert condition | `WHEN QUERY IS ABOVE 0` |
| `3. Add folder and labels` | Folder | `CFTP 运维告警` |
| `3. Add folder and labels` | Labels | `severity=critical`, `environment=test`, `source=loki`, `team=cftp` |
| `4. Set evaluation behavior` | Evaluation group / interval | `cftp-log-errors` / `1m` |
| `4. Set evaluation behavior` | Pending period | `1m` |
| `4. Set evaluation behavior` | Keep firing for | `0s` / `None` |
| `5. Configure notifications` | Alertmanager / Contact point | `grafana` / `CFTP-WeCom` |
| `6. Configure notification message` | Summary | `CFTP 测试环境最近 5 分钟检测到 ERROR 日志` |
| `6. Configure notification message` | Runbook URL | 留空 |

向页面最底部滚动，点击 `Save rule and exit`。如果按钮提示配置不完整，按错误提示回到对应
的第 1 至第 6 区补齐；最常见的是未选择 Folder 或 Evaluation group。

### 10.2 验证通知链路

优先使用 Contact point 页面自带的 `Test` 功能验证企业微信通知，不要为了测试而在业务
服务中制造真实错误。

保存规则后，在 `Alerting -> Alert rules` 中确认规则状态为 `Normal`。当 Loki 中出现真实
ERROR 时，规则状态应依次变为：

```text
Normal -> Pending -> Alerting
```

企业微信群收到消息后，进入 Grafana Explore，查询：

```logql
{level="ERROR"}
```

时间范围选择 `Last 15 minutes`。如果已经确认 Namespace 标签，则使用：

```logql
{namespace="cftp-test", level="ERROR"}
```

## 11. 常见问题

### 11.1 Preview 显示 0 / Normal

这是正常结果，表示最近 5 分钟没有 ERROR，不代表规则没有生效。

### 11.2 Preview 显示 No data

确认查询中保留：

```logql
or on() vector(0)
```

然后点击 `Run queries`。如果仍是 No data，检查 Loki 数据源是否可用。

### 11.3 企业微信没有收到测试消息

1. 在 Contact point 页面重新点击 `Test`。
2. 检查 Webhook 地址是否完整、机器人是否仍在群内。
3. 检查 Grafana 服务器是否能访问 `qyapi.weixin.qq.com`。
4. 查看 Grafana Pod 日志中的通知发送错误。

### 11.4 告警太多

1. 给查询增加测试环境 Namespace 标签。
2. 将阈值从 `IS ABOVE 0` 调整为 `IS ABOVE 2`，表示最近 5 分钟至少 3 条 ERROR。
3. 保持 Repeat interval 为 `2h` 或更长。
4. 为已知无须处理的固定错误增加精确排除条件，不要笼统排除整个服务。

### 11.5 没有告警但 Grafana Explore 能看到错误

检查 Explore 中错误日志的 `level` 是 Loki 标签还是 JSON 字段。如果 `level` 只是日志正文
中的 JSON 字段而不是标签，查询需要改成：

```logql
(
  sum(count_over_time({namespace="cftp-test"} | json | level="ERROR" [5m]))
  or on() vector(0)
)
```

其中 `namespace` 仍需替换为 Label browser 中的真实 Kubernetes Namespace 标签。

## 12. 后续建议

第一条全局 ERROR 规则稳定运行后，再分别建立以下规则，不要一开始全部开启：

| 规则 | 建议条件 |
| --- | --- |
| ERROR 集中爆发 | 最近 5 分钟 ERROR 数量大于 9 |
| 认证失败激增 | 最近 5 分钟相同认证错误数量大于 4 |
| Loki 查询失败 | 告警规则执行错误立即通知 |
| 核心服务无日志 | 有稳定流量时，指定服务长时间没有任何日志 |

新增规则时应按 Namespace、服务和严重级别限制范围，并先使用 Contact point Test 验证通知，
避免通过真实业务写操作制造错误。
