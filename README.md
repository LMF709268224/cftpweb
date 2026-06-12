# cftpweb

2026-05-29 17:00 更新
**adminserver（4 commits）- 管理端课程/管线配置**
1.GLMS 课程列表接入分页：支持 page_size / page_token，减少一次加载全部课程的压力。
2.课程管理适配新版 GLMS：课程导入、封面上传、file_hash、signed_headers 等接口变化。
3.管线配置页面接入 GCC：支持管线创建、结构配置、发布、删除、课程单元关联。
4.移除课程“下架”相关 TODO/按钮，因为课程不再提供下架接口；提示改成“创建新版本或编辑草稿版本”。
5.去掉课程导入弹窗里的演示 JSON，避免误以为系统自带示例数据。

**candidateserver（7 commits）- 考生端商城/课程/真实数据展示**
1.接入 GCC 管线数据：展示阶段、课程单元、资格数量、支付配置状态等真实结构。
2.增加管线详情页。
3.mall 处理逻辑适配新版 GCC proto，补齐 category_tips 等字段。
4.认证材料上传适配新版 gcreds：增加必填校验、文件 SHA-256、signed_headers 上传。
5.登录身份解析简化为依赖 Gmid.GetUlidByUUID，失败即返回错误，不再 fallback 到 Casdoor IDs。
6.清理大量演示假数据：首页、考试、档案、订单、会员、课程卡片改为真实接口数据或空状态。

2026-06-03 14:10 更新
**adminserver（课检与管线人工干预）**
1. 支持在管理后台编辑和更新已存在的课检（包括修改最大尝试次数等配置）。
2. 在管线管理页面的实例运维区域，新增对 `Course Unit` 的 `Force Completed` 和 `Force Signup Exam` 操作，通过接入后端 `gprog` API 接口，实现对考生管线节点进度的强制人工干预。

**candidateserver（真实状态展示）**
1. 移除考生端课程学习页和详情页中对管线 (Pipeline)、阶段 (Stage)、课程单元 (Course Unit) 状态的前端“本地化添油加醋”，强制改为直接展示后端微服务返回的原始状态文本（如 `RUNNING`、`WAITING_STUDY` 等），确保无歧义。


管理员界面应该能看到一切
干预一切 不能让考生卡住