# Aurora 设计系统契约（CFTP 管理后台）

> 本文件是全站 UI 重构的**唯一风格来源**。所有页面 Agent 必须严格按本文件的 token 与配方施工，禁止发明规范之外的颜色、圆角、阴影、字重。所有配方类（`.ui-*`、`.tnum`、`.skeleton`、过渡类）已在 `src/styles/globals.css` 中定义，直接使用即可。
> 样板页：`src/pages/DashboardPage.vue`（表格/筛选/统计卡/分页/空态/骨架屏全配方示范）、`src/components/AdminLayout.vue`（壳层）、`src/pages/LoginPage.vue`（品牌面）。

## (a) 设计理念

Aurora 是一套「明亮、清透、值得信赖」的金融教育 SaaS 视觉语言，气质对标 Stripe Dashboard / Linear / Notion：极浅的冷灰画布上放置纯白圆角卡片，靛蓝作为唯一品牌色克制点缀（激活态、主按钮、focus ring），文字层级靠**字号与灰度**而非粗字重区分，动效短促（120–180ms）只为反馈不为炫技。整体印象：专业、干净、有呼吸感。

## (b) Token 清单（globals.css `@theme` 已定义，类名可直接用）

| Token | 值 | Tailwind 类示例 |
|---|---|---|
| primary-50 | `#EEF2FF` | `bg-primary-50`（激活浅底） |
| primary-100 | `#E0E7FF` | `bg-primary-100` |
| primary-200/300/400 | `#C7D2FE / #A5B4FC / #818CF8` | `focus:border-primary-400` |
| primary-500 | `#6366F1` | `ring-primary-500/10` |
| primary-600 | `#4F46E5` | `bg-primary-600`（主按钮/激活） |
| primary-700 | `#4338CA` | `hover:bg-primary-700`、`text-primary-700` |
| 品牌渐变 | `linear-gradient(135deg,#6366F1,#8B5CF6)` | 仅限 logo 徽章 / 登录页，内联 style |
| canvas 画布 | `#F7F8FC` | `bg-canvas` |
| surface 卡片 | `#FFFFFF` | `bg-white` |
| edge 边框 | `#E8ECF4` | `border-edge` |
| hairline 分隔线 | `#EEF1F6` | `divide-hairline`、`border-hairline` |
| ink 标题字 | `#0F172A` | `text-ink` |
| body 正文字 | `#475569` | `text-body`（或 `text-slate-600`） |
| faint 弱化字 | `#94A3B8` | `text-faint`（或 `text-slate-400`） |
| success 50/200/600 | `#ECFDF5 / #A7F3D0 / #059669` | `text-success-600` |
| warning 50/200/600 | `#FFFBEB / #FDE68A / #D97706` | `text-warning-600` |
| danger 50/200/600 | `#FEF2F2 / #FECACA / #DC2626` | `text-danger-600` |
| info 50/200/600 | `#F0F9FF / #BAE6FD / #0284C7` | `text-info-600` |

排印纪律：
- **全站禁用 `font-black`**；最大字重 `font-bold` 且仅用于统计大数字。
- 页面主标题 `text-2xl font-semibold tracking-tight`；卡片/区块标题 `text-base font-semibold text-ink`；表头 `text-xs font-medium uppercase tracking-wider text-slate-400`；正文 `text-sm`；辅助 `text-xs text-slate-400`；统计大数字 `tnum text-3xl font-bold`。
- 数字、金额、ID、日期、页码一律加 `tnum`（= `font-variant-numeric: tabular-nums`）。

形状纪律：卡片/弹窗 `rounded-2xl`；按钮/输入框等控件 `rounded-[10px]`（小按钮 `rounded-lg`）；徽章/头像 `rounded-full`。**禁用 `rounded-3xl`**。控件高度统一 `h-10`，小按钮 `h-8`。

阴影纪律：卡片阴影已内置于 `.ui-card`；弹窗 `shadow-2xl`（内置于 `.ui-modal-panel`）；除此之外不加阴影。

## (c) 组件配方（可直接复制）

### 页面容器 + 页面头部（每页统一，无 eyebrow）
```html
<main class="ui-page"><!-- = mx-auto w-full max-w-[1440px] px-4 py-6 md:px-6 lg:px-8 lg:py-8 -->
  <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
    <div class="min-w-0">
      <h1 class="ui-page-title">{{ copy.title }}</h1>
      <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
    </div>
    <div class="flex flex-wrap gap-3"><!-- 操作按钮组 --></div>
  </header>
  <!-- 内容第一块统一 mt-6，后续块 mt-4 或 mt-6 -->
</main>
```

### 按钮四态（`.ui-btn` + 一个变体；小尺寸再加 `.ui-btn-sm`）
```html
<button class="ui-btn ui-btn-primary" type="button">保存</button>
<button class="ui-btn ui-btn-secondary" type="button">取消</button>
<button class="ui-btn ui-btn-ghost" type="button">更多</button>
<button class="ui-btn ui-btn-danger" type="button">删除</button>
<button class="ui-btn ui-btn-secondary ui-btn-sm" type="button">上一页</button>
```
展开等价（仅当无法用组件类时手写）：
`ui-btn` = `inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-[10px] px-4 text-sm font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20 disabled:pointer-events-none disabled:opacity-50`；
`primary` = `bg-primary-600 text-white shadow-sm hover:bg-primary-700 active:scale-[0.98]`；
`secondary` = `border border-slate-200 bg-white text-slate-700 shadow-sm hover:border-slate-300 hover:bg-slate-50 active:scale-[0.98]`；
`ghost` = `text-slate-600 hover:bg-slate-100 hover:text-slate-900`；
`danger` = `bg-red-600 text-white shadow-sm hover:bg-red-700 active:scale-[0.98]`。
按钮内图标：`class="h-4 w-4"`（secondary 里的图标加 `text-slate-400`）。RouterLink 当按钮用时同样直接挂这些类。

### 输入 / 选择 / 搜索框（`<select>` 也用 `.ui-input`）
```html
<label class="grid gap-1.5">
  <span class="ui-label">{{ copy.fieldLabel }}</span><!-- = text-sm font-medium text-slate-600 -->
  <input v-model="x" class="ui-input" :placeholder="copy.placeholder" />
</label>

<select v-model="x" class="ui-input">…</select>

<!-- 带图标搜索框（含清除按钮） -->
<div class="relative">
  <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
  <input v-model="keyword" class="ui-input pl-9 pr-10" :placeholder="copy.searchPlaceholder" />
  <button v-if="keyword" class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.clear" @click="clear">
    <X class="h-4 w-4" />
  </button>
</div>
```
`ui-input` 展开 = `h-10 w-full rounded-[10px] border border-slate-200 bg-white px-3 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10`。

### 卡片区块（头部 + divide-y 内容 + 计数徽章）
```html
<section class="ui-card mt-4 overflow-hidden">
  <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
    <div class="min-w-0">
      <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
      <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
    </div>
    <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ total }} …</span>
  </div>
  <div class="divide-y divide-hairline"><!-- 行内容 --></div>
</section>
```
统计卡：`ui-card p-4 md:p-5`，label `text-sm font-medium text-slate-500`，右上图标容器 `flex h-8 w-8 items-center justify-center rounded-[10px] bg-slate-50 text-slate-400`（图标 `h-4 w-4`），数字 `tnum mt-3 text-3xl font-bold text-ink`。可点击卡片额外加 `.ui-card-hover`。

### 表格（桌面）
```html
<div class="hidden overflow-x-auto md:block">
  <table class="w-full text-left text-sm">
    <thead class="ui-table-head"><!-- = border-b border-edge bg-slate-50/60 text-xs font-medium uppercase tracking-wider text-slate-400 -->
      <tr><th class="px-4 py-3 font-medium">…</th></tr>
    </thead>
    <tbody class="divide-y divide-hairline">
      <tr class="ui-table-row"><!-- = transition-colors hover:bg-primary-50/40 -->
        <td class="px-4 py-3.5 text-slate-600">…</td>
        <td class="tnum px-4 py-3.5 text-right">1,024</td><!-- 数字列 -->
      </tr>
    </tbody>
  </table>
</div>
```
主列（名称）单元格：`<div class="font-medium text-ink">名称</div><div class="tnum text-xs text-slate-400">ID</div>`。
div 网格式伪表格（如 OrdersPage）同理：表头行用 `ui-table-head` 的等价类 + `px-4 py-3`，数据行 `ui-table-row` + `px-4 py-3.5`。

### 状态徽章
```html
<span class="ui-badge" :class="badgeClass(status)">{{ label }}</span>
```
`ui-badge` = `inline-flex items-center whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-medium`。**不要再加 `border` 类**（`lib/status.ts` 的 `badgeClass` 已改为无边框浅底：`bg-emerald-50 text-emerald-700` / `bg-amber-50 text-amber-700` / `bg-red-50 text-red-700` / `bg-slate-100 text-slate-600`）。页面本地徽章色仅允许：primary（`bg-primary-50 text-primary-700`）、emerald、amber、red、sky（`bg-sky-50 text-sky-700`）、slate 这六组浅底配色。

### Tabs（底部 2px 指示条式，非胶囊）
```html
<div class="flex gap-5 overflow-x-auto border-b border-hairline px-4 md:px-6">
  <button v-for="tab in tabs" :key="tab.key" type="button"
    class="ui-tab" :class="activeTab === tab.key ? 'ui-tab-active' : ''"
    @click="activeTab = tab.key">
    {{ tab.title }}
    <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
  </button>
</div>
```
`ui-tab` = `inline-flex h-10 shrink-0 items-center gap-2 border-b-2 border-transparent px-1 text-sm font-medium text-slate-500 transition-colors hover:text-slate-700`；`ui-tab-active` = `border-primary-600 text-primary-700`。

### 弹窗（遮罩 + 面板 + 三段结构，带过渡）
```html
<Teleport to="body">
  <Transition name="modal">
    <div v-if="open" class="ui-modal-mask"><!-- = fixed inset-0 z-40 flex items-center justify-center bg-slate-950/40 p-0 backdrop-blur-[2px] md:p-6 -->
      <section class="ui-modal-panel max-w-[1280px]"><!-- 移动端全屏、md 起圆角卡片，按需给 max-w -->
        <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink md:text-lg">{{ title }}</h2>
            <p class="tnum mt-1 break-all text-sm text-slate-500">{{ subtitle }}</p>
          </div>
          <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="close">
            <X class="h-5 w-5" />
          </button>
        </div>
        <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6"><!-- 主体 --></main>
        <div class="flex justify-end gap-3 border-t border-hairline px-4 py-4 md:px-6"><!-- 底部按钮 --></div>
      </section>
    </div>
  </Transition>
</Teleport>
```
过渡类 `modal-*` 已在 globals.css 定义（遮罩 fade 180ms，`.ui-modal-panel` 自动 scale 0.98→1 + fade）。

### 确认框
弹窗配方的缩小版：`ui-modal-panel` 加 `md:max-w-md`，主体为 `text-sm text-slate-600` 的说明文字，底部 `ui-btn ui-btn-secondary`（取消）+ `ui-btn ui-btn-danger` 或 `ui-btn-primary`（确认）。

### 分页条
```html
<div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
  <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page) }}</span>
  <div class="flex flex-col gap-2 sm:flex-row">
    <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
    <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
  </div>
</div>
```

### 空状态
```html
<div class="ui-empty"><!-- = flex flex-col items-center justify-center gap-1 px-6 py-14 text-center -->
  <div class="ui-empty-icon"><Inbox class="h-6 w-6" /></div><!-- = h-12 w-12 rounded-2xl bg-slate-100 text-slate-400 -->
  <p class="ui-empty-title">{{ copy.empty }}</p>
  <p v-if="copy.emptyHint" class="ui-empty-hint">{{ copy.emptyHint }}</p>
</div>
```
表格内空态：`<td colspan="N">` 内放上述结构。

### 加载 / 骨架屏
表格骨架（替代原「Loader2 + 加载中」整行文字）：
```html
<tr v-for="i in 5" :key="`sk-${i}`">
  <td class="px-4 py-3.5"><div class="skeleton h-4 w-28" /></td>
  <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td><!-- 徽章列 -->
</tr>
```
卡片/区块骨架：`<div class="skeleton h-12 w-full" />` 若干行，宽度交错（w-2/5、w-4/5、w-3/5）。
行内加载（按钮内、局部刷新）保留 `<Loader2 class="h-4 w-4 animate-spin" />`，配色随所在文字（默认继承，弱化处 `text-slate-400`）。

### 移动端卡片列表项（md:hidden 双形态之移动形态）
```html
<div class="divide-y divide-hairline md:hidden">
  <article class="px-4 py-4">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="break-words text-sm font-semibold text-ink">{{ 主标题 }}</div>
        <div class="tnum mt-0.5 break-all text-xs text-slate-400">{{ ID }}</div>
      </div>
      <span class="ui-badge shrink-0" :class="badgeClass(status)">{{ 状态 }}</span>
    </div>
    <dl class="mt-3 divide-y divide-hairline text-sm">
      <div class="flex items-start justify-between gap-4 py-2">
        <dt class="shrink-0 text-xs font-medium text-slate-400">{{ 字段名 }}</dt>
        <dd class="m-0 break-all text-right text-slate-600">{{ 值 }}</dd>
      </div>
    </dl>
  </article>
</div>
```
（替代旧的 `rounded-2xl bg-slate-50 px-3 py-2` 灰底字段行 —— 新配方是细分隔线的键值行，无底色。）

### 侧边栏 / 壳层
见 `src/components/AdminLayout.vue`，页面 Agent 不需要动它。品牌渐变（内联 `style="background: linear-gradient(135deg, #6366f1, #8b5cf6)"`）只允许出现在 logo 徽章与登录/回调页。

## (d) 新旧映射表（见旧模式立即替换）

| # | 旧代码 | 新配方 |
|---|---|---|
| 1 | `font-black`（任何位置） | 标题→`font-semibold`；正文/标签→`font-medium`；统计大数字→`font-bold` |
| 2 | `font-bold`（按钮/正文/标签） | `font-medium` |
| 3 | `text-3xl font-black md:text-4xl` 页面主标题 + eyebrow 行 | 删除 eyebrow；`<h1 class="ui-page-title">` + `<p class="ui-page-subtitle">` |
| 4 | `text-xl font-black` 区块标题 | `text-base font-semibold text-ink` |
| 5 | `rounded-3xl`（卡片/弹窗） | `rounded-2xl`（用 `.ui-card` / `.ui-modal-panel`） |
| 6 | `rounded-xl` / `rounded-2xl`（按钮、输入框、tab 胶囊） | `rounded-[10px]`（用 `.ui-btn` / `.ui-input`；tab 改指示条式 `.ui-tab`） |
| 7 | `h-11`（输入/按钮） | `h-10`（`.ui-btn` / `.ui-input` 已内置）；次要小按钮 `h-8`（`.ui-btn-sm`） |
| 8 | `bg-blue-700` / `bg-[#0b4ea2]` / `bg-[#0b7bdc]` / `bg-sky-500` 主按钮 | `.ui-btn .ui-btn-primary`（bg-primary-600） |
| 9 | `text-blue-700` / `text-[#0b4ea2]` / `text-sky-700` 强调文字/链接 | `text-primary-700`（hover 可 `hover:text-primary-600`） |
| 10 | `bg-blue-100 text-blue-700` / `bg-sky-100 text-sky-700` 徽章 | `bg-primary-50 text-primary-700`（挂在 `.ui-badge` 上） |
| 11 | 徽章 `rounded-full border px-3 py-1 text-xs font-black/font-bold` + 旧 badgeClass | `class="ui-badge" :class="badgeClass(x)"`，**去掉 `border` 与 px/py/text 类** |
| 12 | `bg-amber-100/emerald-100/red-100 text-*-700` 本地徽章色 | 对应 `bg-amber-50 / bg-emerald-50 / bg-red-50 text-*-700` |
| 13 | `hover:bg-sky-50` / `hover:bg-sky-50/60` 表格行 | `.ui-table-row`（`hover:bg-primary-50/40 transition-colors`）；选中态 `bg-primary-50/60` |
| 14 | 表头 `bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500` | `.ui-table-head`，th 用 `px-4 py-3 font-medium` |
| 15 | 单元格 `px-6 py-4` | `px-4 py-3.5` |
| 16 | 移动端字段行 `flex … rounded-2xl bg-slate-50 px-3 py-2` | 上文「移动端卡片列表项」的 `<dl>` 细分隔线键值行 |
| 17 | `border-slate-100` / `divide-slate-100` 分隔 | `border-hairline` / `divide-hairline`；卡片外框 `border-edge`（`.ui-card` 内置） |
| 18 | 页面容器 `max-w-[1580px]/[1600px] px-4 py-5 md:px-8 md:py-8` | `.ui-page`（max-w-[1440px]） |
| 19 | 弹窗遮罩 `bg-slate-950/50` 无动画 | `.ui-modal-mask` + `<Transition name="modal">` |
| 20 | 加载整行 `Loader2 … 加载中` 文字（表格/列表主加载） | 骨架屏 `.skeleton` 行（行内小加载仍可用 Loader2） |
| 21 | 空态一行灰字 `p-8 text-center text-slate-500` | `.ui-empty` + `.ui-empty-icon` 结构 |
| 22 | 计数徽章 `rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600` | `tnum ui-badge bg-slate-100 text-slate-600` |
| 23 | 表单标签 `grid gap-2 text-sm font-bold` | `grid gap-1.5` + `<span class="ui-label">` |
| 24 | 金额/ID/日期/页码无 tnum | 一律加 `tnum` |
| 25 | 详情弹窗 tab 胶囊 `rounded-2xl border … bg-sky-50` | `.ui-tab` / `.ui-tab-active` 指示条式 |
| 26 | 关闭按钮 `h-10 w-10 rounded-full border border-slate-200 shadow-sm` | `h-9 w-9 rounded-full text-slate-400 hover:bg-slate-100 hover:text-slate-600`（无边框无阴影） |
| 27 | `shadow-lg shadow-sky-200` 等彩色阴影 | 删除；卡片用 `.ui-card` 内置阴影 |

## (e) 动效规范

- **时长/缓动**：进入 150–180ms `ease-out`；离开 100–140ms `ease-in`；hover/颜色 150ms（`transition-colors duration-150` 或 `.ui-btn` 内置 `transition-all duration-150`）。不超过 400ms（仅登录页入场 `.animate-rise` 为 400ms）。
- **该有过渡的元素**：按钮（颜色 + `active:scale-[0.98]`）、表格行 hover 背景、输入框 border/ring、页面切换（AdminLayout 已内置 `<Transition name="page">`：fade + 6px 上移 180ms）、弹窗（`name="modal"`）、下拉/浮层（`name="menu"`：scale 0.98 + fade 150ms）、可点击卡片（`.ui-card-hover`：`-translate-y-px` + 阴影加深）、进度条宽度（`transition-all duration-300`）。
- **不该动的**：表格数据刷新、骨架屏之外的布局尺寸、大面积背景。禁止 bounce/spring 类夸张动效。

## (f) Do & Don't

**Do**
- 只用本文件的 token 与配方类；能用 `.ui-*` 就不要手写等价类。
- 所有文案走 i18n（`copy.xxx`）；保留每页现有的 `aria-label`、`@keydown`、`role`、`tabindex`、`disabled` 逻辑。
- 保留桌面表格 + 移动卡片双形态与所有 `md:` 断点分支，只替换视觉类。
- 数字类内容（金额、计数、ID、日期、页码）加 `tnum`。
- 图标统一 lucide，按钮内 `h-4 w-4`，表格行内 `h-4 w-4`，空态 `h-6 w-6`。

**Don't**
- 不用 `font-black`、`rounded-3xl`、`text-4xl`、彩色阴影、`bg-[#0b4ea2]`/`bg-[#0b7bdc]`/blue/sky 旧主色。
- 不改 `<script>` 业务逻辑、API、路由、i18n key；只允许改样式辅助函数返回的 class 字符串与图标 import。
- 不新增依赖、不引外部字体/CDN、不改 `src/lib/locales/`。
- 不在徽章上加边框；不在页面里使用品牌渐变（渐变仅限 logo 徽章与登录/回调页）。
- 不发明新的颜色浅底组合（徽章只用 primary/emerald/amber/red/sky/slate 六组 50 号浅底）。
- 不删除任何交互元素（清除按钮、刷新、筛选、复制、确认弹窗、toast）。
