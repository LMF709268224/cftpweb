# Aurora 产品配方补编（DESIGN-PRODUCT.md）

> Phase 3 产品级重构专用，是 `DESIGN.md`（视觉契约）之上的**产品结构配方**。模块 Agent 施工顺序：先读 DESIGN.md（token / `.ui-*`），再按本文件落地共享组件与新交互配方。铁律不变：只用已有 API、不改 locale 文件、文案只用页面既有 key 与 `t.value.common.*`。

`common` 命名空间已定稿（zh/en 同步存在）：
`moreActions / historyVersions / dangerZone / searchCandidate / selectedCount / clearSelection / expand / collapse / copied / noResults / selectTypeFirst / viewFiles / deprecatedMuted / todayTodos / progress / openLink / reasonUnavailable / unsaved / backToList / advancedFilters`

---

## (a) 共享组件用法

### UlidChip —— 全站所有 ULID / 确认号 / Object Key
```vue
<script setup lang="ts">
import UlidChip from "@/components/UlidChip.vue"
</script>
<template>
  <!-- 默认中间截断（前6…后5，≤14 位不截断），title=全值，点击复制并 Copy→Check 反馈 1.2s -->
  <UlidChip :value="row.exam_ulid" />
  <!-- 审计类页面需要完整值：不截断（仍是等宽小字胶囊+复制） -->
  <UlidChip :value="log.object_key" :truncate="false" />
</template>
```
- 空值自动渲染 `-`，不需要页面再做 `|| "-"`。
- 组件内部已 `@click.stop`，可安全放进「整行可点」的表格行里。
- 不要再包 `title` / 复制按钮 / toast——组件已闭环（图标反馈，无文案）。

### CandidatePicker —— 替换所有"裸考生 ULID 输入框"
```vue
<script setup lang="ts">
import CandidatePicker from "@/components/CandidatePicker.vue"
const candidateUlid = ref("")
</script>
<template>
  <label class="grid gap-1.5">
    <span class="ui-label">{{ copy.candidateLabel }}</span><!-- 页面既有 key -->
    <CandidatePicker v-model="candidateUlid" />
  </label>
</template>
```
- v-model 值就是 candidate ULID string，直接接入页面现有查询参数，**查询函数不变**。
- 内置：250ms debounce 搜索（`/api/dashboard/ops` 已有接口）、键盘 ↑/↓/Enter/Escape、直接粘贴 26 位 ULID 回车即选、选中卡片（姓名 + UlidChip + 清除）。
- 从路由 query 预填时直接 `candidateUlid.value = String(route.query.candidate || "")` 即可，组件会显示已选卡片（无姓名时只显示 chip）。

### RowMenu —— 行内「⋯ 更多」菜单（P6 危险操作收敛）
```vue
<script setup lang="ts">
import RowMenu from "@/components/RowMenu.vue"
import { CopyPlus, Eye, Trash2 } from "lucide-vue-next"
const rowMenuItems = computed(() => [
  { key: "detail", label: copy.value.viewDetail, icon: Eye },
  { key: "clone", label: copy.value.clone, icon: CopyPlus },
  { key: "delete", label: copy.value.delete, icon: Trash2, danger: true },
])
function onRowMenu(key: string, row: Row) {
  if (key === "detail") openDetail(row)
  if (key === "clone") cloneRow(row)
  if (key === "delete") confirmDelete(row) // 保留现有确认弹窗
}
</script>
<template>
  <td class="px-4 py-3.5 text-right">
    <div class="flex items-center justify-end gap-2">
      <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click.stop="edit(row)">{{ copy.edit }}</button>
      <RowMenu :items="rowMenuItems" @select="(key) => onRowMenu(key, row)" />
    </div>
  </td>
</template>
```
- 触发按钮 aria-label/title 已内置 `common.moreActions`；点击外部 / Escape 自动关闭；危险项自动红字并在上方加分隔线。
- 行内只保留 1 个主操作按钮（编辑或详情），其余全部进菜单；`disabled: true` 项自动降权禁用。

### groupByLineage —— 版本族分组（P4，纯 computed）
```ts
import { groupByLineage } from "@/lib/versionGroup"

// 列表已按更新时间倒序 → 每组 primary = 最新版本，其余进 history
const lineageGroups = computed(() =>
  groupByLineage(items.value, (p) => p.name || "", (p) => p.category || ""),
)
// => Array<{ stem: string; primary: T; history: T[] }>
const expandedStems = ref<Record<string, boolean>>({}) // 展开状态本地 ref
```
- 名称尾部所有 ` (Copy)` / `（副本）` / ` (副本)` 后缀会被叠加去除后作为 stem；第三参可选（如按分类再隔离）。
- 不改任何请求；筛选后的数组同样可以直接传入。

---

## (b) 实体链接配方 + deep-link 约定表

### 链接样式（正文色 + 虚线下划线 + hover 主色 + 小图标）
```html
<RouterLink
  :to="{ path: '/prog', query: { candidate: row.candidate_ulid } }"
  class="inline-flex items-center gap-1 text-sm text-slate-600 underline decoration-slate-300 decoration-dashed underline-offset-4 transition-colors hover:text-primary-700 hover:decoration-primary-400"
  @click.stop
>
  {{ row.candidate_name }}
  <ArrowUpRight class="h-3.5 w-3.5 text-slate-400" />
</RouterLink>
```
图标用 lucide `ArrowUpRight`（站内跳转）或 `ExternalLink`（新开页面场景），`h-3.5 w-3.5`。放在可点击行内必须 `@click.stop`。

### deep-link 约定表（双方都只用页面**已有**筛选字段与查询函数）

| 目标路由 | query 参数 | 发起方（示例） | 接收方职责 |
|---|---|---|---|
| `/prog` | `candidate=<考生ULID>` | 考试详情「认证实例」、看板考生行 | onMounted 读 `route.query.candidate` → 预填考生筛选 → 执行现有查询 |
| `/exams` | `candidate=<考生ULID>` | 认证详情 hero「查看考试」 | 同上，预填考生筛选 |
| `/pipelines` | `category=<分类>` | 商品配置「绑定认证」 | onMounted 读 `category` → 预填分类筛选 → 现有查询 |
| `/resource-pack-files` | `pack=<资源包ID>` | 资源包行「查看文件」（文案 `common.viewFiles`） | onMounted 读 `pack` → 预填资源包筛选 → 现有查询 |

接收方样板：
```ts
import { useRoute } from "vue-router"
const route = useRoute()
onMounted(() => {
  const candidate = String(route.query.candidate || "")
  if (candidate) candidateUlid.value = candidate // 已有筛选字段 / CandidatePicker v-model
  void load() // 页面已有查询函数
})
```
只读 query、不写 query、不新增路由。

---

## (c) 竖向时间线配方（状态历史 / 状态日志）

```html
<ol class="relative space-y-0 pl-0">
  <li v-for="(event, index) in events" :key="index" class="relative flex gap-3 pb-6 last:pb-0">
    <!-- 连线：非末项在圆点下方画 hairline 竖线 -->
    <div class="flex flex-col items-center">
      <span
        class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
        :class="index === 0 ? 'bg-primary-600 ring-4 ring-primary-500/15' : 'bg-slate-300'"
      />
      <span v-if="index < events.length - 1" class="mt-1 w-px flex-1 bg-hairline" />
    </div>
    <div class="min-w-0 flex-1 -mt-0.5">
      <div class="text-sm font-medium" :class="index === 0 ? 'text-ink' : 'text-slate-600'">{{ event.title }}</div>
      <div class="tnum mt-0.5 text-xs text-slate-400">{{ event.time }}</div>
      <div v-if="event.hash" class="mt-1"><UlidChip :value="event.hash" /></div>
    </div>
  </li>
</ol>
```
- 当前（最新）节点 = 主色圆点 + `ring-primary-500/15` 光晕，标题 `text-ink`；历史节点灰点灰字。
- 事件按时间倒序渲染（index 0 = 最新）。哈希 / ID 一律 UlidChip。

---

## (d) 横向 Stepper 配方（认证阶段三态）

```html
<ol class="flex items-start gap-0 overflow-x-auto pb-1">
  <li v-for="(stage, index) in stages" :key="stage.id" class="flex min-w-0 flex-1 items-start">
    <div class="flex min-w-0 flex-col items-center gap-1.5 px-1 text-center" style="min-width: 72px">
      <!-- 三态圆点 -->
      <span
        v-if="stage.state === 'done'"
        class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-600 text-white"
      ><Check class="h-3.5 w-3.5" /></span>
      <span
        v-else-if="stage.state === 'current'"
        class="tnum flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-50 text-xs font-medium text-primary-700 ring-4 ring-primary-500/15"
      >{{ index + 1 }}</span>
      <span
        v-else
        class="tnum flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-100 text-xs font-medium text-slate-400"
      >{{ index + 1 }}</span>
      <span
        class="w-full truncate text-xs"
        :class="stage.state === 'current' ? 'font-medium text-primary-700' : stage.state === 'done' ? 'text-slate-600' : 'text-slate-400'"
        :title="stage.name"
      >{{ stage.name }}</span>
    </div>
    <!-- 连接线（最后一项不画） -->
    <div
      v-if="index < stages.length - 1"
      class="mt-3 h-0.5 min-w-4 flex-1 rounded-full"
      :class="stage.state === 'done' ? 'bg-primary-600' : 'bg-slate-200'"
    />
  </li>
</ol>
```
- 三态：`done`（主色实心 + Check）→ `current`（主色浅底 + 光晕 + 序号）→ `pending`(灰底序号)。态由页面现有阶段字段 computed 得出。
- 迷你进度（列表卡片右侧）：`<span class="tnum text-xs text-slate-400">{{ currentIndex + 1 }}/{{ stages.length }}</span>` 前缀 label 用 `t.value.common.progress`。

---

## (e) 危险区卡片配方（详情 / 编辑器底部）

```html
<section class="mt-6 rounded-2xl border border-danger-200 bg-danger-50/40 p-4 md:p-5">
  <h3 class="flex items-center gap-2 text-sm font-semibold text-danger-600">
    <TriangleAlert class="h-4 w-4" />
    {{ t.common.dangerZone }}
  </h3>
  <div class="mt-3 flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-center">
    <p class="text-sm text-slate-600">{{ copy.deleteHint }}</p><!-- 页面既有 key，无则省略此行 -->
    <button class="ui-btn ui-btn-danger shrink-0" type="button" @click="confirmDelete">{{ copy.delete }}</button>
  </div>
</section>
```
- 危险按钮触发的**确认弹窗全部保留**（DESIGN.md 确认框配方）。
- 多个危险操作纵向堆叠，每条一行说明 + 按钮；次危险（如下架）可用 `ui-btn-secondary` + `text-danger-600` 描边变体：`class="ui-btn ui-btn-secondary text-danger-600 hover:border-danger-200 hover:bg-danger-50"`。

---

## (f) 状态 chips 行配方（P7 高频筛选）

```html
<div class="flex flex-wrap items-center gap-2">
  <button
    v-for="chip in statusChips"
    :key="chip.value"
    type="button"
    class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
    :class="statusFilter === chip.value
      ? 'bg-primary-600 text-white shadow-sm'
      : 'border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900'"
    :aria-pressed="statusFilter === chip.value"
    @click="setStatus(chip.value)"
  >
    {{ chip.label }}
    <span v-if="chip.count !== undefined" class="tnum text-xs" :class="statusFilter === chip.value ? 'text-white/80' : 'text-slate-400'">{{ chip.count }}</span>
  </button>
</div>
```
- 完整三态：active = `bg-primary-600 text-white shadow-sm`；inactive = `border border-slate-200 bg-white text-slate-600`；hover = `hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900`。
- 点击即触发现有查询（沿用页面 debounce/watch 逻辑）；chips 覆盖不了的低频条件收进「更多筛选」折叠区（触发按钮：`ui-btn ui-btn-secondary` + `Filter` 图标 + `t.common.advancedFilters`，展开区放原有输入框）。
- 保留原下拉时，chips 与下拉互为同步（同一个 ref）。

---

## (g) 版本族分组行 / 展开行配方（P4）

```html
<template v-for="group in lineageGroups" :key="group.stem">
  <!-- 主行：正常权重，沿用页面原行配方（ui-table-row） -->
  <tr class="ui-table-row cursor-pointer" @click="openPrimary(group.primary)">
    <td class="px-4 py-3.5">
      <div class="flex items-center gap-2">
        <span class="font-medium text-ink">{{ group.primary.name }}</span>
        <!-- 历史版本切换（有历史才显示） -->
        <button
          v-if="group.history.length"
          type="button"
          class="tnum inline-flex h-6 shrink-0 items-center gap-1 rounded-full bg-slate-100 px-2 text-xs font-medium text-slate-500 transition-colors hover:bg-slate-200/70 hover:text-slate-700"
          :aria-expanded="Boolean(expandedStems[group.stem])"
          :title="t.common.historyVersions"
          @click.stop="expandedStems[group.stem] = !expandedStems[group.stem]"
        >
          <History class="h-3 w-3" />
          {{ group.history.length }}
          <ChevronDown class="h-3 w-3 transition-transform duration-150" :class="expandedStems[group.stem] ? 'rotate-180' : ''" />
        </button>
      </div>
      <div class="mt-0.5"><UlidChip :value="group.primary.id" /></div>
    </td>
    <!-- 其余列沿用页面原配方 -->
  </tr>
  <!-- 历史子行：浅灰底 + 左侧缩进竖线 + 整体降权 -->
  <template v-if="expandedStems[group.stem]">
    <tr v-for="old in group.history" :key="old.id" class="ui-table-row bg-slate-50/60" @click="openPrimary(old)">
      <td class="px-4 py-2.5">
        <div class="flex items-center gap-2 border-l-2 border-slate-200 pl-3">
          <span class="text-sm text-slate-500">{{ old.name }}</span>
          <span v-if="isDeprecated(old)" class="ui-badge bg-slate-100 text-slate-500">{{ t.common.deprecatedMuted }}</span>
        </div>
        <div class="mt-0.5 pl-3"><UlidChip :value="old.id" /></div>
      </td>
      <!-- 其余列同主行但文字 text-slate-500 -->
    </tr>
  </template>
</template>
```
- 展开状态 `const expandedStems = ref<Record<string, boolean>>({})`，纯本地。
- 历史行内的操作仍走 RowMenu（可克隆/删除），已下架/已废弃行整行 `opacity` 不用，改用灰字 + `deprecatedMuted` 徽章降权。
- 卡片式列表（非 table）同理：主卡片下插入 `bg-slate-50/60` 的子卡片列表，带 `border-l-2 border-slate-200 pl-3` 缩进。

---

## (h) 吸底操作条配方（编辑器 / split-view 审核面板）

```html
<!-- 容器：卡片/面板内容区需要 pb-24（给操作条让位） -->
<div
  class="sticky bottom-0 z-20 -mx-4 mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-hairline bg-white/95 px-4 py-3 backdrop-blur md:-mx-6 md:px-6"
  style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom))"
>
  <!-- 左：状态提示（未保存等，用 common.unsaved；无未保存态则留空 div 保持布局） -->
  <span v-if="dirty" class="flex items-center gap-1.5 text-xs text-amber-700">
    <CircleAlert class="h-3.5 w-3.5" />
    {{ t.common.unsaved }}
  </span>
  <span v-else />
  <!-- 右：操作按钮组，危险操作放最左侧远离主按钮 -->
  <div class="flex flex-wrap items-center gap-3">
    <button class="ui-btn ui-btn-secondary" type="button" @click="confirmDelete">{{ copy.delete }}</button>
    <button class="ui-btn ui-btn-secondary" type="button" :disabled="publishing" @click="publish">{{ copy.publish }}</button>
    <button class="ui-btn ui-btn-primary" type="button" :disabled="saving" @click="save">
      <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
      {{ copy.save }}
    </button>
  </div>
</div>
```
- 关键点：`sticky bottom-0` + `bg-white/95 backdrop-blur` + `border-t border-hairline`；安全区用 `env(safe-area-inset-bottom)` 内联 padding。
- 负 margin（`-mx-4 md:-mx-6`）用于贴满父卡片左右 padding；父滚动容器需 `overflow-y-auto` 且操作条在其内部末尾。
- 弹窗内吸底 = DESIGN.md 弹窗三段结构的底部 slot（已天然吸底），不要重复造。
- 删除类按钮与主按钮之间至少隔一个次级按钮；点击仍走确认弹窗。

---

## 浮层通用约定（RowMenu / CandidatePicker 已内置，页面自建浮层时遵守）

```html
<Transition name="menu"><!-- globals.css 已定义：scale 0.98 + fade 150ms -->
  <div v-if="open" class="absolute right-0 top-full z-30 mt-1.5 min-w-[10rem] overflow-hidden rounded-xl border border-edge bg-white py-1 shadow-lg shadow-slate-900/5">
    …
  </div>
</Transition>
```
- 浮层是全站唯一允许 `shadow-lg` 的场景（配 `shadow-slate-900/5` 压淡）。
- 点击外部关闭统一用 `document.addEventListener("pointerdown", handler)`（onMounted 挂、onBeforeUnmount 卸），Escape 也要关。
- z-index 阶梯：浮层 `z-30` < 弹窗遮罩 `z-40`。

## 自查清单（每页改完过一遍）
1. 无裸 ULID 直排（全部 UlidChip 或 mono 次级色 + 复制）。
2. 无并排红字删除链接（列表 → RowMenu；详情 → 危险区卡片）。
3. 详情无"假表单"（只读 → 定义列表瓦片；历史 → 时间线；阶段 → Stepper；JSON → JsonPreview）。
4. deep-link 接收方 onMounted 读 query 并触发一次现有查询。
5. lg 以下分栏退化单栏，所有功能可达；确认弹窗、toast、键盘可达性不减配。
