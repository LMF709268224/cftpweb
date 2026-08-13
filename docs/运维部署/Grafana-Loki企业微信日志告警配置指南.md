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

### 4.4 图三中的 Label browser 要不要配置

对应位置：图三查询框上方、`Kick start your query` 右侧的 `Label browser` 按钮。

本次不用点击，也不用在这里填写任何内容。继续保留当前查询：

```logql
(
  sum(count_over_time({level="ERROR"}[5m]))
  or on() vector(0)
)
```

`Label browser` 只是以后用于查看 Loki 有哪些标签，并不是保存规则必须完成的配置。当前先
统计 Loki 收集到的全部 ERROR，以便把企业微信通知链路配置成功。

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

对应位置：图四 `Keep firing for` 下面已经展开的 `Configure no data and error handling`。

你图四中的三个值已经正确，不需要修改：

| 图四中的字段 | 保持的值 | 含义 |
| --- | --- | --- |
| `Alert state if no data or all values are null` | `Normal` | 没有 ERROR 日志时保持正常 |
| `Alert state if execution error or timeout` | `Error` | Loki 查询失败时生成独立的 `DatasourceError` 告警 |
| `Missing series evaluations to resolve` | `Default: 2` | 连续缺失 2 次评估后再清理对应序列 |

因此，图四保持当前的 `Normal / Error / Default: 2`，直接继续向下配置
`5. Configure notifications` 即可。如果以后收到 `DatasourceError`，应检查 Loki 数据源和
Grafana 到 Loki 的网络，而不是应用 ERROR 日志。

## 7. 先创建企业微信 Contact point

在填写页面第 5 区之前，需要先创建企业微信接收方。如果 Contact point 下拉框里已经能选到
`cftp`，直接跳到第 8 节。你当前截图中已经选择了 `cftp`，因此不需要重新创建。

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
cftp
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
3. 选择你截图中已经创建好的 Contact point：

```text
cftp
```

4. 点击并展开 `Muting, grouping and timings (optional)`。
5. `Mute timings` 保持空，不选择任何 `Select time intervals...`。
6. `Active timings` 保持空，不选择任何 `Select time intervals...`。
7. `Override grouping` 开关保持关闭。
8. `Override timings` 开关保持关闭。

你当前界面右侧已经显示继承的默认值：

| 配置项 | 设置值 | 含义 |
| --- | --- | --- |
| Group wait | `30s` | 首次触发后等待 30 秒，将同时发生的告警合并 |
| Group interval | `5m` | 同一告警组的新增变化最多每 5 分钟通知一次 |
| Repeat interval | `4h` | 问题持续存在时每 4 小时提醒一次 |

这三个值来自 Grafana 的 Notification policy。因为默认值已经合理，所以不要打开
`Override timings`，也不需要寻找或填写三个输入框。`Grouping: grafana_folder, alertname`
同样保持默认，不要打开 `Override grouping`。

第 5 区完成后的状态应与截图一致：

```text
Alertmanager: grafana
Contact point: cftp
Mute timings: 空
Active timings: 空
Override grouping: 关闭
Override timings: 关闭
默认 timings: Group wait 30s, Group interval 5m, Repeat interval 4h
```

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
| `5. Configure notifications` | Alertmanager / Contact point | `grafana` / `cftp` |
| `5. Configure notifications` | Mute timings / Active timings | 都留空 |
| `5. Configure notifications` | Override grouping / Override timings | 都关闭，使用默认 `30s / 5m / 4h` |
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
3. 保持当前默认 Repeat interval 为 `4h`。
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

### 11.6 配置告警后 Explore 显示 No data

配置告警不会删除、移动或消费 Loki 日志，告警查询也不会自动保留在 Explore 页面。打开
`Grafana -> Explore -> loki` 后，如果查询输入框显示灰色提示文字 `Enter a Loki query`，说明
当前查询框是空的；此时页面显示 `No data` 不代表 Loki 中的日志消失了。

按以下步骤重新查询：

1. 在查询输入框填写 `{level="ERROR"}`。
2. 将右上角时间范围先改为 `Last 24 hours`，避免最近 1 小时恰好没有 ERROR。
3. 点击 `Run query`。
4. 如果仍没有结果，点击 `Label browser`，选择一个实际存在的标签和值后执行查询；不要猜测
   Namespace 或容器标签名。
5. 也可以点击查询框下方的 `Query history`，找到之前执行过的查询并重新运行。

如果 `{level="ERROR"}` 没有结果，但希望先确认是否仍有其他级别日志，可以查询：

```logql
{level=~".+"}
```

该查询只覆盖带 `level` 标签的日志。如果 Label browser 中不存在 `level`，应改用界面中实际
存在的 Namespace、应用或容器标签来确认日志采集是否正常。

## 12. 方案评估、运行代价和新增容器

### 12.1 当前方案是否最优

当前规则适合用作测试环境的第一条兜底告警，用于确认
`Vector -> Loki -> Grafana -> 企业微信` 整条链路可用；它不是正式环境的最终最优方案。

当前查询：

```logql
(
  sum(count_over_time({level="ERROR"}[5m]))
  or on() vector(0)
)
```

有以下特点：

1. 查询会统计当前 Loki 数据源中所有带 `level="ERROR"` 标签的日志，不区分 Namespace、
   应用或容器。
2. 外层 `sum` 把所有日志合并成一个数值，通知链路简单、告警实例少，但企业微信消息不能直接
   指出是哪个服务产生了错误，仍需进入 Explore 排查。
3. 单条 ERROR 会在 5 分钟滚动窗口内持续被统计，因此 `Pending period=1m` 不能过滤掉单次
   ERROR；它通常仍会触发告警，并在该日志离开查询窗口后恢复。
4. `No data=Normal` 适合“没有 ERROR 就正常”的日志计数规则，但无法证明 Vector、Loki 和
   应用本身仍在正常工作。采集链路中断时，也可能表现为没有 ERROR。

因此，当前规则可以保留为全局安全网。正式使用时，不建议把“任意一条 ERROR”长期当作
`critical`；应再建立按环境和服务划分的告警规则。

### 12.2 运行代价

| 代价 | 当前规则的影响 |
| --- | --- |
| Loki 查询 | 每分钟执行一次，即每天约 1440 次；每次查询最近 5 分钟。日志量越大，查询消耗的 CPU、内存和存储读取越多 |
| 日志存储 | 新增容器会增加 Vector 传输量和 Loki 的磁盘/对象存储占用；这部分通常比单条告警规则本身的成本更大 |
| 告警噪声 | 任意一条 ERROR 都会触发，单次错误通常产生一条告警通知，恢复通知开启时还会再产生一条恢复消息 |
| 定位时间 | 当前 `sum` 丢失了服务和容器维度，收到通知后必须回到 Explore 查询原始日志 |
| 监控盲区 | Grafana、Loki 或 Vector 自身故障时，这条业务日志规则不能独立保证仍能发出通知 |
| 信息安全 | 当前只发送错误数量、规则状态和 Grafana 链接，不发送原始日志，泄露 Token 或用户数据的风险较低 |

`level` 只有 `DEBUG/INFO/WARN/ERROR` 等少量固定值，基数较低，可以作为现有查询标签使用。
不要把用户 ID、订单号、请求 ID、Trace ID 或 Pod UID 增加为 Loki 索引标签，否则会制造大量
短生命周期日志流，增加写入、存储和查询成本。

### 12.3 再运行一个容器是否自动适用

新增 Pod、同一 Deployment 增加副本，或者同一 Pod 增加容器后，满足以下全部条件时，当前规则
会自动统计它的 ERROR，不需要修改 Grafana 规则：

1. 容器把日志写到标准输出或标准错误输出，而不是只写容器内部文件。
2. Vector 使用 Kubernetes Pod 日志采集源，并且该 Namespace、Pod 或容器没有被采集过滤规则
   或 `vector.dev/exclude` 注解排除。
3. Vector 最终发送给 Loki 的日志包含规范化后的 Loki 标签 `level="ERROR"`。如果新应用只在
   日志正文中输出 `error`，当前标签查询不会命中。
4. 日志被写入当前 Grafana 所使用的同一个 Loki 数据源或 Tenant。

当前规则会把新增容器的数据合并进总数，但不会在告警中显示新增容器的名称。部署后应在
`Grafana -> Explore -> loki -> Label browser` 中确认实际的 Namespace 和容器标签名称及取值，
再用真实标签执行一次查询。例如实际标签名确认为 `namespace` 和 `container` 后：

```logql
{namespace="cftp-test", container="adminbff"}
```

能看到该容器最新日志，且下面的查询能看到它的错误日志，才表示它已被当前规则覆盖：

```logql
{namespace="cftp-test", container="adminbff", level="ERROR"}
```

如果 Label browser 显示的是 `namespace_name`、`kubernetes_namespace_name`、
`container_name` 等其他名称，必须使用界面中的真实名称替换示例，不能猜测标签名。

“再运行一个容器”还需要区分容器的用途：

| 新增对象 | 是否直接适用 | 需要注意 |
| --- | --- | --- |
| 新业务 Pod/容器或现有 Deployment 副本 | 通常适用 | 必须满足上面的采集、标签和 Loki Tenant 条件 |
| Grafana 副本 | 不能只把副本数改为 2 | 多个 Grafana 实例需要共享 MySQL/PostgreSQL，并配置 Grafana Alerting HA；否则规则数据可能不一致或产生重复通知。默认情况下每个 Grafana 副本都会执行全部规则，Loki 查询负载会随副本数增加 |
| Vector 副本 | 只有按节点部署时才适合扩展 | Kubernetes 中通常使用 DaemonSet 保持每个节点一个采集实例；两个 Vector 同时读取同一节点的同一批日志，可能重复写入 Loki，进而造成错误计数和告警重复 |
| Loki 副本 | 需要按 Loki 的分布式/高可用部署方式配置 | 不能把两个互不共享数据的单机 Loki 当成一个数据源使用 |

Grafana Alerting HA 的作用是让多个 Grafana 实例都能评估规则，并让内置 Alertmanager 尽力去重
通知；它优先保证不漏通知，因此网络异常时仍可能偶尔产生重复通知。当前只有一个 Grafana 实例
时，不需要为了这一条规则专门增加 Grafana 副本。

### 12.4 推荐的最终方案

建议采用分层告警，而不是只保留一条全局规则：

| 层级 | 用途 | 建议 |
| --- | --- | --- |
| 全局兜底 | 验证链路并发现未分类错误 | 保留当前规则；正式环境建议标记为 `warning` |
| 服务错误突增 | 发现某个稳定服务在 5 分钟内连续报错 | 按 Namespace 和稳定的应用/容器标签分组，阈值可先设为大于 2 |
| 明确严重错误 | 启动失败、数据库不可用等确认需要立即处理的错误 | 使用精确错误特征，出现 1 次即触发 `critical` |
| 可用性告警 | Pod 重启、Ready 失败、HTTP 不可达 | 使用 Kubernetes/Prometheus/探活指标；不要仅依赖“有没有 ERROR 日志” |
| 采集链路健康 | Grafana 无法查询 Loki、Vector 停止采集 | 单独监控 `DatasourceError` 和日志采集组件状态 |

如果 Label browser 已确认稳定标签名是 `namespace` 和 `app`，服务分组规则可以使用：

```logql
sum by (namespace, app) (
  count_over_time({namespace="cftp-test", level="ERROR"}[5m])
)
```

若没有 `app` 标签，可以改用实际存在且取值稳定的 `service` 或 `container` 标签。不要按 Pod
名称、用户 ID、订单号或请求 ID 建立告警实例；Pod 名称会随发布和扩缩容变化，其余字段基数
更高，都会增加告警数量和 Loki 查询成本。

## 13. 后续建议

第一条全局 ERROR 规则稳定运行后，再分别建立以下规则，不要一开始全部开启：

| 规则 | 建议条件 |
| --- | --- |
| ERROR 集中爆发 | 最近 5 分钟 ERROR 数量大于 9 |
| 认证失败激增 | 最近 5 分钟相同认证错误数量大于 4 |
| Loki 查询失败 | 告警规则执行错误立即通知 |
| 核心服务无日志 | 有稳定流量时，指定服务长时间没有任何日志 |

新增规则时应按 Namespace、服务和严重级别限制范围，并先使用 Contact point Test 验证通知，
避免通过真实业务写操作制造错误。
