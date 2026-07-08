# Admin RPC 接入覆盖盘点

日期：2026-07-08

## 口径

- 微服务 SDK：`github.com/afnandelfin620-star/cftptest/cftp v0.0.0-20260705015142-e0830875b701`
- 微服务接口口径：从 generated gRPC client 的 `*_grpc.pb.go` 中扫描方法名以 `Admin` 结尾的 RPC。
- 已接入口径：从 `adminbff/handler` 中扫描实际调用的 `h.<Service>.<RpcName>(...)`。
- 注意：这里统计的是 BFF 是否调用了 `*Admin` RPC，不等同于前端页面体验已经完整；也不覆盖没有 `Admin` 后缀但 admin 端实际使用的接口。

## 总览

| 项目 | 数量 |
| --- | ---: |
| 微服务 SDK 中 `*Admin` RPC | 106 |
| adminbff 已调用的 `*Admin` RPC | 93 |
| 尚未调用的 `*Admin` RPC | 13 |

当前所有未调用的 `*Admin` RPC 都在 `glms` 微服务；`gcc`、`gmall`、`gmsg` 中带 `Admin` 后缀的 RPC 已全部被 adminbff 调用。

## 当前 Admin 已接模块

### adminbff 路由模块

- 认证与用户：`/api/auth`、`/api/user`
- 运营看板：`/api/dashboard/ops`
- 管线配置：`/api/pipelines`
- 管线实例与证书流水：`/api/prog/pipelines`、`/api/prog/certificate-tasks`、`/api/prog/course-units`
- 考试管理：`/api/exams`
- 课程配置：`/api/lms/courses`、章节、课时、资料、补充资料、权限、导入
- 测验题库：`/api/lms/quizzes`、questions、options、attempts
- 资源包配置：`/api/lms/resource-packs`
- 资源文件配置：`/api/lms/resource-pack-files`
- LMS 资产与对象：`/api/lms/assets`、`/api/lms/objects`、`/api/lms/upload-url`、`/api/lms/view-url`、`/api/lms/broken-assets`
- 站内信：`/api/messages`
- 邮件中心：`/api/mails`
- 资格定义：`/api/credentials`
- 审核中心：`/api/applications`
- PDF 模板与生成请求：`/api/pdf-templates`、`/api/pdf-requests`
- 商品、订单、发票：`/api/mall`
- 会员配置：`/api/memberships`
- 审计日志与 Webhook 审计：`/api/audit`
- 考生权限管理：`/api/permissions`

### adminweb 页面模块

- `ApplicationsPage.vue`
- `AuditLogsPage.vue`
- `BundlesPage.vue`
- `CredentialsPage.vue`
- `DashboardPage.vue`
- `ExamsPage.vue`
- `InvoicesPage.vue`
- `LmsPage.vue`
- `MailsPage.vue`
- `MessagesPage.vue`
- `OrdersPage.vue`
- `PdfRequestsPage.vue`
- `PdfTemplatesPage.vue`
- `PermissionsPage.vue`
- `PipelinesPage.vue`
- `ProgPage.vue`
- `ResourcePackFilesPage.vue`
- `ResourcePacksPage.vue`
- `ResourcePage.vue`
- `WebhookAuditPage.vue`

## 按微服务的 Admin RPC 覆盖

| 微服务 | `*Admin` RPC 数量 | 已调用 | 未调用 |
| --- | ---: | ---: | ---: |
| `gcc` | 1 | 1 | 0 |
| `glms` | 103 | 90 | 13 |
| `gmall` | 1 | 1 | 0 |
| `gmsg` | 1 | 1 | 0 |

## 尚未调用的 Admin RPC

| 微服务 | RPC | 当前判断 | 建议 |
| --- | --- | --- | --- |
| `glms` | `DeprecateResourcePackAdmin` | 资源包废弃接口，当前 admin 没有接。之前已确认“下架”走 `RevertResourcePackToDraftAdmin`，废弃不可逆，暂不建议接到普通 UI。 | 保持不接，除非产品明确需要不可逆废弃能力，并加二次确认和权限隔离。 |
| `glms` | `EnrollCandidateCourseAdmin` | 单个课程报名接口未接；当前已接 `BatchEnrollCandidateCoursesAdmin`。 | 可选。如果后台需要“单个考生报名单个课程”的轻量入口，可以接；否则批量接口已覆盖大部分场景。 |
| `glms` | `GetCourseMaterialDetailAdmin` | 课程资料详情版接口未接；当前接的是 `GetCourseMaterialAdmin`。 | 如果详情版返回更多字段，建议替换或补充到资料详情页。 |
| `glms` | `GetChapterDetailAdmin` | 章节详情版接口未接；当前接的是 `GetChapterAdmin`。 | 如果详情版包含课时、资料、统计等聚合信息，可用于课程配置右侧详情。 |
| `glms` | `GetLessonDetailAdmin` | 课时详情版接口未接；当前接的是 `GetLessonAdmin`。 | 如果详情版包含资料、进度或内容扩展字段，建议接到课时详情。 |
| `glms` | `GetPrerequisiteDetailAdmin` | 前置条件详情版接口未接；当前接的是 `GetPrerequisiteAdmin`。 | 如果需要展示依赖对象完整信息，建议接。 |
| `glms` | `GetQuizDetailAdmin` | 测验详情版接口未接；当前接的是 `GetQuizAdmin`，题目和选项另走列表接口。 | 可选。若详情版能一次返回测验、题目、选项，可减少前端多次请求。 |
| `glms` | `GetQuizQuestionDetailAdmin` | 测验题目详情版接口未接；当前接的是 `GetQuizQuestionAdmin`。 | 如果详情版包含选项或解析，建议用于题目编辑详情。 |
| `glms` | `GetQuizOptionDetailAdmin` | 测验选项详情版接口未接；当前接的是 `GetQuizOptionAdmin`。 | 如果详情版与普通版字段一致，可暂不接。 |
| `glms` | `GradeQuizAttemptAdmin` | 管理员评分测验尝试接口未接。 | 建议优先评估。这个和“考试 / 测验需要管理员干预”有关，可能需要接入考试管理或测验尝试详情。 |
| `glms` | `ListLessonsByCourseAdmin` | 按课程列出课时接口未接；当前主要按章节列课时。 | 可选。若下拉框需要跨章节选择课时，建议接。 |
| `glms` | `ListPrerequisitesByRequiredEntityAdmin` | 按被依赖对象反查前置条件接口未接。 | 可选。适合在课程、章节、课时详情里展示“哪些配置依赖了我”。 |
| `glms` | `UpdateResourcePackFileThumbnailAdmin` | 更新资源包文件缩略图接口未接。 | 如果资源文件需要单独维护封面/缩略图，建议接到资源文件配置。 |

## 已覆盖的 Admin RPC 摘要

### gcc

- `ListPipelinesAdmin`

### gmall

- `ListBundlesAdmin`

### gmsg

- `ListMessagesAdmin`

### glms

已覆盖课程、章节、课时、资料、补充资料、前置条件、测验、题目、选项、导入、对象/资产、学习进度、报名、资源包、资源包文件、课程权限等大部分 Admin RPC。

其中需要特别说明：

- 资源包状态流转当前使用 `PublishResourcePackAdmin` 和 `RevertResourcePackToDraftAdmin`。
- `DeprecateResourcePackAdmin` 虽然微服务提供，但当前按业务要求没有接到 admin，因为废弃后不能再上架。
- 课程报名当前使用 `BatchEnrollCandidateCoursesAdmin`，单个报名的 `EnrollCandidateCourseAdmin` 未接。

## 需要进一步确认的点

- `Get*DetailAdmin` 这类详情版接口是否比当前 `Get*Admin` 返回更多字段。如果字段一样，暂不接也可以；如果字段更完整，建议逐步替换详情页调用。
- `GradeQuizAttemptAdmin` 是否应该纳入“考试管理”的人工干预流程。如果考试或测验存在需要人工评分的场景，这个接口应该优先接。
- `UpdateResourcePackFileThumbnailAdmin` 是否是资源文件封面管理的唯一正确入口。如果是，资源文件配置应补上。
- `ListLessonsByCourseAdmin` 和 `ListPrerequisitesByRequiredEntityAdmin` 更像辅助查询接口，是否接取决于 UI 是否需要跨层级选择或依赖反查。

## 结论

- 从 `*Admin` RPC 覆盖看，adminbff 已经接了大多数接口。
- 没有发现整个带 `Admin` 后缀的微服务模块完全没接。
- 目前缺口集中在 `glms` 的 13 个接口，其中最值得优先评估的是 `GradeQuizAttemptAdmin`、`UpdateResourcePackFileThumbnailAdmin` 和若干 `Get*DetailAdmin`。
