# Grafana + Loki 企业微信日志告警配置指南

## 1. 目标和告警链路

本文用于配置以下告警链路：

```text
Kubernetes Pod 日志 -> Vector -> Loki -> Grafana 告警规则 -> 企业微信群机器人
```

当前第一阶段规则的目标是：Grafana 每分钟检查一次 Loki；最近 5 分钟出现至少一条
`ERROR` 日志并持续满足 1 分钟时，向企业微信群发送告警。

该规则只读取日志，不调用业务接口，也不会修改业务数据。

## 2. 当前配置核对

规则名称：

```text
CFTP 服务 ERROR 日志告警
```

Loki 查询：

```logql
(
  sum(count_over_time({level="ERROR"}[5m]))
  or on() vector(0)
)
```

查询选项：

| 配置项 | 设置值 |
| --- | --- |
| 数据源 | `loki` |
| 查询模式 | `Code` |
| Type | `Instant` |
| 查询时间 | `10m to now`，保持默认即可 |
| Alert condition | `WHEN QUERY IS ABOVE 0` |

点击 `Preview alert rule condition` 后显示数值 `0` 和绿色 `Normal`，表示查询和条件
均配置正确，当前最近 5 分钟没有匹配的 ERROR 日志。

`or on() vector(0)` 不应删除。它确保没有匹配日志时返回数值 `0`，而不是 `No data`。

### 2.1 当前查询范围

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

## 3. 创建规则文件夹和标签

在 `3. Add folder and labels` 中操作。

### 3.1 创建 Folder

1. 点击 `New folder`。
2. Folder 名称填写：

```text
CFTP 运维告警
```

3. 创建后选择该 Folder。

Folder 只用于整理 Grafana 告警规则，不是 Kubernetes Namespace。

### 3.2 添加 Labels

点击 `Add labels`，依次添加：

| Key | Value | 用途 |
| --- | --- | --- |
| `severity` | `critical` | 表示需要立即关注 |
| `environment` | `test` | 表示测试环境 |
| `source` | `loki` | 表示告警来自日志 |
| `team` | `cftp` | 便于后续通知路由和静默 |

这些标签不是 Loki 查询条件，不会改变查询结果；它们用于搜索告警、静默告警和通知路由。

## 4. 配置评估行为

在 `4. Set evaluation behavior` 中操作。必须先完成 Folder 选择，才能创建 Evaluation group。

### 4.1 创建 Evaluation group

1. 点击 `New evaluation group`。
2. Group 名称填写：

```text
cftp-log-errors
```

3. Evaluation interval 设置为：

```text
1m
```

这表示 Grafana 每分钟执行一次 Loki 查询。

### 4.2 Pending period 和 Keep firing for

| 配置项 | 设置值 | 含义 |
| --- | --- | --- |
| Pending period | `1m` | 条件持续满足 1 分钟后才进入告警状态 |
| Keep firing for | `0s` / `None` | 条件恢复后立即结束告警 |

因此，一条 ERROR 日志出现后，最多等待约 1 至 2 分钟收到通知。该日志离开 5 分钟查询
窗口后，规则会恢复为 Normal。

### 4.3 No data 和 Error handling

展开 `Configure no data and error handling`，建议配置：

| 场景 | 设置值 | 原因 |
| --- | --- | --- |
| Alert state if no data | `Normal` | 正常无错误日志不应告警；查询本身也已用 `vector(0)` 兜底 |
| Alert state if execution error or timeout | `Alerting` | Loki 查询失败时也需要通知，避免监控失效却无人发现 |

如果查询执行错误触发告警，先检查 Loki 数据源和 Grafana 到 Loki 的网络，而不是应用日志。

## 5. 创建企业微信 Contact point

如果已经创建并测试过企业微信 Contact point，可以跳到第 6 节。

### 5.1 创建企业微信群机器人

1. 打开用于接收告警的企业微信群。
2. 进入群设置，选择 `群机器人`。
3. 添加机器人，名称建议填写 `CFTP 系统告警`。
4. 保存机器人生成的 Webhook 地址。

Webhook 地址格式类似：

```text
https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=REDACTED
```

Webhook 中的 `key` 是密钥。禁止写入 Git、Markdown、截图或公开聊天记录。

### 5.2 在 Grafana 中创建 Contact point

1. 在规则页面第 5 步点击 `View or create contact points`，或者从左侧进入
   `Alerting -> Notification configuration -> Contact points`。
2. 点击 `New contact point`。
3. Name 填写：

```text
CFTP-WeCom
```

4. Integration 选择 `WeCom`。
5. URL 填入企业微信群机器人的完整 Webhook 地址。
6. 暂时保留默认消息模板。
7. 点击 `Test`，确认企业微信群收到 Grafana 测试通知。
8. 点击 `Save contact point`。

如果 Integration 列表没有 `WeCom`，不要直接把企业微信 URL 填进普通 Webhook 后保存。
不同 Grafana 版本的普通 Webhook 消息体可能不符合企业微信格式，应先升级 Grafana 或部署
专用的消息格式转换服务。

## 6. 配置通知接收方

返回告警规则的 `5. Configure notifications`：

1. Alertmanager 保持 `grafana`。
2. Contact point 选择：

```text
CFTP-WeCom
```

3. 展开 `Muting, grouping and timings (optional)`。
4. 如果当前 Grafana 提供对应输入框，填写：

| 配置项 | 设置值 | 含义 |
| --- | --- | --- |
| Group wait | `30s` | 首次触发后等待 30 秒，将同时发生的告警合并 |
| Group interval | `5m` | 同一告警组的新增变化最多每 5 分钟通知一次 |
| Repeat interval | `2h` | 问题持续存在时每 2 小时提醒一次 |

如果这三个输入框没有显示，保持默认值即可；它们也可以在
`Alerting -> Notification policies` 中统一配置。不要把 Repeat interval 设置成 `1m`，否则
持续错误可能频繁刷群。

## 7. 配置通知内容

在 `6. Configure notification message` 中填写。

Summary：

```text
CFTP 测试环境最近 5 分钟检测到 ERROR 日志
```

Description：

```text
Loki 最近 5 分钟 ERROR 日志数量大于 0。请打开 Grafana Explore，选择 Loki，查询最近 15 分钟的 ERROR 日志并按 Pod 和服务定位原因。
```

Runbook URL：当前没有专用在线排障页面时留空，不要填写无效地址。

当前告警发送的是错误数量、规则状态和 Grafana 链接，不会自动附带完整日志内容。这能避免
Token、用户信息或业务数据被转发到企业微信群。详细错误必须回到 Grafana Explore 查看。

## 8. 保存和验证

### 8.1 保存规则

检查以下配置：

```text
Name: CFTP 服务 ERROR 日志告警
Query: 最近 5 分钟 level="ERROR" 的数量
Condition: IS ABOVE 0
Folder: CFTP 运维告警
Evaluation interval: 1m
Pending period: 1m
Keep firing for: 0s
Contact point: CFTP-WeCom
```

点击 `Save rule and exit`。

### 8.2 验证通知链路

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

## 9. 常见问题

### 9.1 Preview 显示 0 / Normal

这是正常结果，表示最近 5 分钟没有 ERROR，不代表规则没有生效。

### 9.2 Preview 显示 No data

确认查询中保留：

```logql
or on() vector(0)
```

然后点击 `Run queries`。如果仍是 No data，检查 Loki 数据源是否可用。

### 9.3 企业微信没有收到测试消息

1. 在 Contact point 页面重新点击 `Test`。
2. 检查 Webhook 地址是否完整、机器人是否仍在群内。
3. 检查 Grafana 服务器是否能访问 `qyapi.weixin.qq.com`。
4. 查看 Grafana Pod 日志中的通知发送错误。

### 9.4 告警太多

1. 给查询增加测试环境 Namespace 标签。
2. 将阈值从 `IS ABOVE 0` 调整为 `IS ABOVE 2`，表示最近 5 分钟至少 3 条 ERROR。
3. 保持 Repeat interval 为 `2h` 或更长。
4. 为已知无须处理的固定错误增加精确排除条件，不要笼统排除整个服务。

### 9.5 没有告警但 Grafana Explore 能看到错误

检查 Explore 中错误日志的 `level` 是 Loki 标签还是 JSON 字段。如果 `level` 只是日志正文
中的 JSON 字段而不是标签，查询需要改成：

```logql
(
  sum(count_over_time({namespace="cftp-test"} | json | level="ERROR" [5m]))
  or on() vector(0)
)
```

其中 `namespace` 仍需替换为 Label browser 中的真实 Kubernetes Namespace 标签。

## 10. 后续建议

第一条全局 ERROR 规则稳定运行后，再分别建立以下规则，不要一开始全部开启：

| 规则 | 建议条件 |
| --- | --- |
| ERROR 集中爆发 | 最近 5 分钟 ERROR 数量大于 9 |
| 认证失败激增 | 最近 5 分钟相同认证错误数量大于 4 |
| Loki 查询失败 | 告警规则执行错误立即通知 |
| 核心服务无日志 | 有稳定流量时，指定服务长时间没有任何日志 |

新增规则时应按 Namespace、服务和严重级别限制范围，并先使用 Contact point Test 验证通知，
避免通过真实业务写操作制造错误。
