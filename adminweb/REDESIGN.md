# Adminweb UI/UX 重构说明

本分支是管理后台的整体重构版本之一。共有两个可对比的候选版本：

| 分支 | 设计方向 | 气质 |
|---|---|---|
| `redesign/adminweb-v1-aurora` | **Aurora 极光** — 明亮靛蓝 | 清透、专业、值得信赖的教育 SaaS（参考 Stripe/Linear） |
| `redesign/adminweb-v2-graphite` | **Graphite 石墨翡翠** — 深色侧栏 + 翡翠点睛 | 沉稳、贵气、高数据密度的金融机构后台（参考 Vercel/Mercury） |

两个版本**产品结构与交互完全一致**，仅视觉语言不同；后端 API、路由、i18n key 未做任何破坏性改动。

## 一、设计系统层（Phase 1）

- `vue-web/src/styles/globals.css`：Tailwind v4 `@theme` 全套设计 token（品牌色/语义色/表面色）、`.ui-*` 组件配方类（按钮/输入/卡片/徽章/表头/弹窗/空态）、骨架屏、页面与弹窗过渡、统一 focus 与滚动条。
- 全站字重纪律（禁 font-black）、圆角/间距/阴影体系、tabular-nums 数字。
- `AdminLayout` / `LoginPage` / `DashboardPage` 全部重写；23 个页面全部迁移到设计系统。
- 完整风格契约见 `vue-web/DESIGN.md`。

## 二、产品与交互层（Phase 3）

完整规格见仓库外部文档 PRODUCT-REDESIGN.md（工作目录）与 `vue-web/DESIGN-PRODUCT.md`。核心变化：

1. **实体链接体系**：考试↔认证、商品→认证配置、资源包→资源文件等跨模块引用可点击跳转并自动带条件查询（deep-link：`/prog?candidate=`、`/exams?candidate=`、`/pipelines?category=`、`/resource-pack-files?pack=`）。
2. **ULID 治理**：全站裸 ULID → `UlidChip`（中间截断 + 悬浮全值 + 一键复制）。
3. **版本族分组**：认证配置/商品配置/资源包按名称主干聚族，最新版为主行，历史版本折叠，已下架降权（`lib/versionGroup.ts`）。
4. **考生选择器**：`CandidatePicker` 按姓名/邮箱即搜即选（复用 dashboard 用户搜索接口），替换考试/认证/订单/考生权限的裸 ULID 输入。
5. **审核工作台**：审核中心改左右分栏（队列 + 详情 + 吸底审核操作条）。
6. **课程配置 Tab 工作台**：大纲（章节→课时嵌套树）/ 资料 / 测验 / 基本信息四个 Tab + 吸底保存条。
7. **详情结构化**：状态流转 → 竖向时间线；认证阶段 → 横向步骤条；原始 JSON → 折叠预览；只读数据 → 定义列表瓦片（不再是灰色假输入框）。
8. **危险操作规范**：删除/终止类操作收进 `RowMenu`（⋯ 菜单）或红色危险区，与常规操作隔离。
9. **筛选 chips 化**：高频状态筛选一键即查；关键词 250ms debounce。
10. 新增共享组件：`UlidChip.vue`、`CandidatePicker.vue`、`RowMenu.vue`；i18n 新增 `common` 命名空间（仅新增 key）。

## 三、遗留建议（不在本仓库范围内）

- **登录页**：现网登录实际渲染的是 Casdoor（`cftpcasdoor.llwan.top`）的默认页面。本仓库内的 LoginPage 已重做为品牌跳转页，但甲方第一眼看到的 Casdoor 表单需在 Casdoor 管理台 → 应用配置中设置自定义 CSS/背景图才能与新视觉统一。
- 若干历史遗留的硬编码 fallback 文案（如"高级配置"等）建议后续补充 i18n key。
- 主 JS chunk 约 1.1MB（构建有 >500kB 提示），后续可做路由级代码分割。

## 四、本地运行

```bash
cd adminweb/vue-web
npm install
npm run dev   # 代理 /api 到 https://cftpadmin.llwan.top
npm run build # vue-tsc 类型检查 + vite 构建
```
