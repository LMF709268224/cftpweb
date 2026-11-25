# 考生端认证阶段列表：学习状态与考试预约

## 本轮已完成（2026-11-25 已部署 dev / prod）

- 阶段和课程状态使用浅灰背景、常规字重及默认光标，与蓝色操作按钮区分。
- 课程整行不再是链接；只有蓝色学习按钮进入学习页，点击状态不会跳转。
- 已有真实学习进度或已完成的课程，按钮显示“继续学习”。
- 学习信息与考试预约分开显示，支持中英文。
- 学习数据来自 GLMS 真实报名记录：`completed` 显示“学习：已完成”；`learning` 且实际完成进度大于 0 显示“学习：进行中（X%）”；进度为 0 时如实显示“学习进度：0%”。缺少记录或字段时显示“学习状态暂不可用”，不默认为未开始或 0%。
- 考试数据使用 GPROG 当前课程单元的 `exam_ulid` 查询 GEXAM，并校验考试 ID、考生 ID、课程单元 ID，避免误用考试配置 ID 或历史补考预约。
- GEXAM `OPEN` 显示“暂未预约考试 / Exam Not Booked”，`SCHEDULED` 显示“已预约考试 / Exam Booked”及接口返回的有效预约日期时间，`DONE` 显示“考试已完成 / Exam Completed”。时间使用网站现有日期格式和时区处理。
- 有考试配置、尚无考试实例，且 GPROG 明确处于 `WAITING_STUDY` / `WAITING_SIGNUP_EXAM` 的单元显示暂未预约；这两个流程状态位于考试创建之前。其他未知状态、接口失败或归属校验失败均显示预约状态暂不可用，不能当成未预约。没有考试的课程不显示预约标签。
- 本次仅调整展示与只读数据聚合，不改变学习、预约、补考、付款或权限流程。发布时需同时更新 `candbff` 和 `candweb`。

## 微服务缺口：准确区分 Not Started 与 In Progress

客户希望学习状态完整区分 Not Started / In Progress / Completed（未开始 / 进行中 / 已完成）。当前接口无法可靠支持“未开始”这一项：

- `glms.ListCandidateEnrollments` 的 `CandidateEnrollmentSummary.status` 只有 `learning/completed`；创建报名记录时就设置为 `learning`。
- `progress_percentage` 根据已完成课时和已通过测验计算。因此 0% 也可能是已开始学习、尚未完成第一项，不能据此推断未开始。
- GPROG `CourseUnitSummary.status` 是包含学习、考试报名、预约及结果的整个单元流程状态；`WAITING_STUDY` 不能直接映射成“未开始”。

需要 GLMS 提供权威的课程学习状态（包含 `not_started`），或可靠的课程首次学习时间 / `has_started` 字段，并明确新报名、正在学习但未完成第一项、结课等状态口径。接口支持后再接入 Not Started；当前不添加猜测或前端本地记录作为替代。

## 验证范围

- BFF：真实零进度与缺数据保持区别；当前考试实例选择；拒绝其他考生、其他单元、旧考试记录及查询失败的数据。
- 浏览器：中文与英文；已完成学习但未预约考试；学习进行中且已预约；无考试课程；考试结束；缺失数据；状态不可点击；移动端布局。
- 已通过 `go test ./handler`、Vue 类型检查、E2E TypeScript 检查及两项浏览器回归（本次状态展示、原有已完成课程重新进入）；已检查桌面及手机截图。

## 2026-11-25 部署记录

- 代码版本：`cd320be7`（完整提交 `cd320be7289b5bb64fa99f255c4e5b232e4fea65`），已推送。
- 使用 `13.212.15.51` 上仓库原有脚本，依次执行：
  - `bash -l /home/ubuntu/cftpweb/docker_abc.sh --env test candbff candweb`（dev）。
  - `bash -l /home/ubuntu/cftpweb/docker_abc.sh --env prod candbff candweb`。
- 两次发布均成功，四个容器均为 `healthy`；两个 BFF 的 `/health` 返回 `OK`，两套考生端页面均返回 HTTP 200。
- BFF 构建信息确认源码版本为上述提交且 `vcs.modified=false`。两个环境公开返回的 `CourseDetailPage-CB7nojUh.js` 与构建产物 SHA-256 一致：`26225aa9aa407a4f87f370835e96ee67f946c73f692a4a46d1715ad9f0ece40a`。
- 分别针对 dev / prod 实际部署的前端资源执行 `pathway-status.spec.ts`，均通过。测试拦截业务 API 使用测试数据，验证中英文、状态与按钮区分、缺数据表现和手机布局；未修改线上考生、学习或考试数据，不代表使用真实账号完成了业务全流程验证。
- 保留上一版镜像供回退：`localhost/{candbff,candweb}:{dev,prod}-before-pathway-status-20261125`。
- 本次未发布 Admin 服务，也未修改微服务；“未开始”的接口缺口继续保留。
