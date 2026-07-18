<script setup lang="ts">
import { Activity, ArrowUpRight, BarChart3, ClipboardCheck, Copy, CreditCard, GitBranch, Loader2, Mail, MessageSquare, Receipt, RefreshCw, Search, Shield, UserCheck, Users, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import RowMenu from "@/components/RowMenu.vue"
import UlidChip from "@/components/UlidChip.vue"
import { apiClient } from "@/lib/apiClient"
import { copyTextToClipboard } from "@/lib/clipboard"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass } from "@/lib/status"

type StageBucket = {
  stage_id: string
  status: string
  count: number
}

type RevenueItem = {
  currency: string
  amount_minor: number
  order_count: number
}

type UserStats = {
  total: number
  active: number
  inactive: number
  admins: number
  members: number
  email_verified: number
}

type RoleStat = {
  key: string
  label: string
  count: number
}

type DashboardUser = {
  id: string
  candidate_ulid?: string
  name: string
  email: string
  phone: string
  location: string
  roles: string[]
  role_label: string
  status: string
  email_verified: boolean
  created_at: string
}

type DashboardData = {
  candidate_total: number
  user_stats: UserStats
  user_role_stats: RoleStat[]
  profile_completion_percent: number
  users: DashboardUser[]
  user_total: number
  user_page: number
  user_page_size: number
  stage_buckets: StageBucket[]
  today_revenue: RevenueItem[]
  generated_at: string
}

const loading = ref(false)
const data = ref<DashboardData | null>(null)
const keyword = ref("")
const roleFilter = ref("all")
const statusFilter = ref("all")
const userPage = ref(1)
const userPageSize = 10
const redDots = ref<Record<string, number>>({})
let filterReloadTimer: number | undefined
const { lang, t } = useAdminLanguage()
const copy = computed(() => t.value.dashboard)
const common = computed(() => t.value.common)
const navCopy = computed(() => t.value.layout.nav)

const userStats = computed<UserStats>(() => data.value?.user_stats || { total: 0, active: 0, inactive: 0, admins: 0, members: 0, email_verified: 0 })
const totalPipelines = computed(() => data.value?.stage_buckets.reduce((sum, item) => sum + Number(item.count || 0), 0) || 0)
const totalPaidOrders = computed(() => data.value?.today_revenue.reduce((sum, item) => sum + Number(item.order_count || 0), 0) || 0)
const profileCompletion = computed(() => Math.max(0, Math.min(100, Number(data.value?.profile_completion_percent || 0))))
const userTotal = computed(() => Number(data.value?.user_total || data.value?.users.length || 0))
const canPrevUsers = computed(() => userPage.value > 1)
const canNextUsers = computed(() => userPage.value * userPageSize < userTotal.value)
const revenueText = computed(() => {
  const items = data.value?.today_revenue || []
  if (!items.length) return copy.value.noRevenue
  return items.map((item) => `${item.currency} ${(Number(item.amount_minor || 0) / 100).toFixed(2)}`).join(" / ")
})
const roleOptions = computed(() => [
  { value: "all", label: copy.value.filters.allRoles },
  { value: "admin", label: copy.value.roles.admin },
  { value: "student", label: copy.value.roles.student },
  { value: "member", label: copy.value.roles.member },
])
const filteredUsers = computed(() => {
  return data.value?.users || []
})

const summaryCards = computed(() => [
  { label: copy.value.summary.totalUsers, value: userStats.value.total, tone: "text-ink", icon: Users },
  { label: copy.value.summary.activeUsers, value: userStats.value.active, tone: "text-success-600", icon: UserCheck },
  { label: copy.value.summary.admins, value: userStats.value.admins, tone: "text-primary-600", icon: Shield },
  { label: copy.value.summary.students, value: data.value?.candidate_total ?? 0, tone: "text-primary-700", icon: Users },
  { label: copy.value.summary.members, value: userStats.value.members, tone: "text-info-600", icon: Users },
])

// 今日待办：/api/system/reddots 计数卡（计数 > 0 才显示），点击直达对应模块
const todoCards = computed(() => [
  { path: "/applications", label: navCopy.value.applications, icon: ClipboardCheck, count: Number(redDots.value.applications || 0) },
  { path: "/exams", label: navCopy.value.exams, icon: ClipboardCheck, count: Number(redDots.value.exams || 0) },
  { path: "/prog", label: navCopy.value.prog, icon: GitBranch, count: Number(redDots.value.prog || 0) },
  { path: "/orders", label: navCopy.value.orders, icon: CreditCard, count: Number(redDots.value.orders || 0) },
  { path: "/invoices", label: navCopy.value.invoices, icon: Receipt, count: Number(redDots.value.invoices || 0) },
  { path: "/messages", label: navCopy.value.messages, icon: MessageSquare, count: Number(redDots.value.messages || 0) },
].filter((card) => card.count > 0))
const todoTotal = computed(() => todoCards.value.reduce((sum, card) => sum + card.count, 0))

function truncateUlid(raw: string) {
  if (raw.length <= 14) return raw
  return `${raw.slice(0, 6)}…${raw.slice(-5)}`
}

function userMenuItems(user: DashboardUser) {
  const ulid = (user.candidate_ulid || "").trim()
  return [
    { key: "copy-ulid", label: ulid ? truncateUlid(ulid) : "-", icon: Copy, disabled: !ulid },
  ]
}

async function onUserMenuSelect(user: DashboardUser, key: string) {
  if (key !== "copy-ulid") return
  const ulid = (user.candidate_ulid || "").trim()
  if (!ulid) return
  try {
    await copyTextToClipboard(ulid)
    toast.success(common.value.copied)
  } catch (err) {
    console.error(err)
  }
}

function stageLabel(stageId: string) {
  return stageId === copy.value.noStage ? copy.value.noStage : stageId
}

function roleBadgeClass(role: string) {
  const normalized = role.toLowerCase()
  if (normalized.includes("admin")) return "bg-primary-50 text-primary-700"
  if (normalized.includes("student")) return "bg-emerald-50 text-emerald-700"
  return "bg-slate-100 text-slate-600"
}

function roleLabel(role: string) {
  const normalized = role.toLowerCase()
  if (normalized.includes("student")) return copy.value.roles.student
  if (normalized.includes("admin")) return copy.value.roles.admin
  if (normalized.includes("member")) return copy.value.roles.member
  return role || "-"
}

function userStatusClass(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === "active") return "bg-emerald-50 text-emerald-700"
  if (normalized === "inactive") return "bg-amber-50 text-amber-700"
  return "bg-slate-100 text-slate-600"
}

function userStatusLabel(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === "active") return copy.value.filters.active
  if (normalized === "inactive") return copy.value.filters.inactive
  if (normalized === "deleted") return copy.value.filters.deleted
  return status || "-"
}

function formatDate(raw: string) {
  if (!raw) return "-"
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) return raw.slice(0, 10)
  return parsed.toLocaleDateString(lang.value === "zh" ? "zh-CN" : "en-US")
}

async function loadDashboard(page = userPage.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      user_page: String(page),
      user_page_size: String(userPageSize),
    })
    const normalizedKeyword = keyword.value.trim()
    if (normalizedKeyword) params.set("user_keyword", normalizedKeyword)
    if (roleFilter.value !== "all") params.set("user_role", roleFilter.value)
    if (statusFilter.value !== "all") params.set("user_status", statusFilter.value)
    data.value = await apiClient<DashboardData>(`/api/dashboard/ops?${params}`)
    userPage.value = Number(data.value.user_page || page)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.loadFailed)
  } finally {
    loading.value = false
  }
}

async function fetchRedDots() {
  try {
    redDots.value = (await apiClient<Record<string, number>>("/api/system/reddots")) || {}
  } catch (err) {
    console.error(err)
  }
}

function refreshAll() {
  void loadDashboard(userPage.value)
  void fetchRedDots()
}

function loadUserPage(page: number) {
  if (page < 1) return
  void loadDashboard(page)
}

function clearUserKeyword() {
  if (!keyword.value) return
  keyword.value = ""
}

onMounted(() => {
  void loadDashboard()
  void fetchRedDots()
})

watch([keyword, roleFilter, statusFilter], () => {
  if (filterReloadTimer) window.clearTimeout(filterReloadTimer)
  filterReloadTimer = window.setTimeout(() => {
    void loadDashboard(1)
  }, 250)
})
</script>

<template>
  <main class="ui-page">
    <!-- Page header -->
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="grid w-full grid-cols-2 gap-3 sm:flex sm:w-auto sm:flex-wrap">
        <RouterLink to="/audit/logs" class="ui-btn ui-btn-secondary">
          <Activity class="h-4 w-4 text-slate-400" />
          {{ copy.auditLogs }}
        </RouterLink>
        <RouterLink to="/mails" class="ui-btn ui-btn-secondary">
          <Mail class="h-4 w-4 text-slate-400" />
          {{ copy.emailActivity }}
        </RouterLink>
        <button
          class="ui-btn ui-btn-primary col-span-2 sm:col-span-1"
          type="button"
          :disabled="loading"
          @click="refreshAll"
        >
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <!-- Today todos: reddots counters, deep links into modules -->
    <section v-if="todoCards.length" class="mt-6">
      <div class="flex items-center gap-2">
        <h2 class="text-base font-semibold text-ink">{{ common.todayTodos }}</h2>
        <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ todoTotal }}</span>
      </div>
      <div class="mt-3 grid grid-cols-2 gap-3 md:grid-cols-3 md:gap-4 xl:grid-cols-6">
        <RouterLink
          v-for="card in todoCards"
          :key="card.path"
          :to="card.path"
          class="ui-card ui-card-hover group p-4"
          :title="card.label"
        >
          <div class="flex items-center justify-between gap-2">
            <p class="truncate text-sm font-medium text-slate-500">{{ card.label }}</p>
            <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[10px] bg-warning-50 text-warning-600">
              <component :is="card.icon" class="h-4 w-4" />
            </span>
          </div>
          <div class="mt-3 flex items-end justify-between gap-2">
            <p class="tnum text-3xl font-bold text-ink">{{ card.count }}</p>
            <ArrowUpRight class="mb-1 h-4 w-4 shrink-0 text-slate-300 transition-colors duration-150 group-hover:text-primary-600" />
          </div>
        </RouterLink>
      </div>
    </section>

    <!-- Profile completion -->
    <section class="ui-card px-5 py-4" :class="todoCards.length ? 'mt-4' : 'mt-6'">
      <div class="mb-2.5 flex items-center justify-between text-sm">
        <span class="font-medium text-slate-600">{{ copy.profileCompletion }}</span>
        <span class="tnum font-semibold text-primary-700">{{ profileCompletion }}%</span>
      </div>
      <div class="h-2 overflow-hidden rounded-full bg-slate-100">
        <div class="h-full rounded-full bg-primary-600 transition-all duration-300" :style="{ width: `${profileCompletion}%` }" />
      </div>
    </section>

    <!-- Stat cards -->
    <section class="mt-4 grid grid-cols-2 gap-3 md:gap-4 xl:grid-cols-5">
      <article v-for="card in summaryCards" :key="card.label" class="ui-card p-4 md:p-5">
        <div class="flex items-center justify-between gap-2">
          <p class="truncate text-sm font-medium text-slate-500">{{ card.label }}</p>
          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[10px] bg-slate-50 text-slate-400">
            <component :is="card.icon" class="h-4 w-4" />
          </span>
        </div>
        <p class="tnum mt-3 text-3xl font-bold" :class="card.tone">{{ card.value }}</p>
      </article>
    </section>

    <!-- User management -->
    <section class="ui-card mt-4 overflow-hidden">
      <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ copy.userManagement }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.userManagementHint }}</p>
          </div>
          <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ copy.userPageText(userPage, userTotal) }}</span>
        </div>
        <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,2fr)_1fr_1fr]">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.table.user }}</span>
            <div class="relative">
              <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input v-model="keyword" class="ui-input pl-9 pr-10" :placeholder="copy.filters.searchPlaceholder" />
              <button
                v-if="keyword"
                class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.filters.clearInput"
                :title="copy.filters.clearInput"
                @click="clearUserKeyword"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.table.role }}</span>
            <select v-model="roleFilter" class="ui-input">
              <option v-for="option in roleOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.table.status }}</span>
            <select v-model="statusFilter" class="ui-input">
              <option value="all">{{ copy.filters.allStatus }}</option>
              <option value="active">{{ copy.filters.active }}</option>
              <option value="inactive">{{ copy.filters.inactive }}</option>
              <option value="deleted">{{ copy.filters.deleted }}</option>
            </select>
          </label>
        </div>
      </div>

      <!-- Mobile card list -->
      <div class="md:hidden">
        <div v-if="loading" class="space-y-4 px-4 py-4">
          <div v-for="i in 4" :key="i" class="space-y-2.5">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3.5 w-4/5" />
            <div class="skeleton h-3.5 w-3/5" />
          </div>
        </div>
        <div v-else-if="filteredUsers.length === 0" class="ui-empty">
          <div class="ui-empty-icon">
            <Users class="h-6 w-6" />
          </div>
          <p class="ui-empty-title">{{ copy.noUsers }}</p>
        </div>
        <div v-else class="divide-y divide-hairline">
          <article v-for="user in filteredUsers" :key="user.id" class="px-4 py-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="break-words text-sm font-semibold text-ink">{{ user.name || "-" }}</div>
                <div class="mt-1"><UlidChip :value="user.candidate_ulid || ''" /></div>
              </div>
              <div class="flex shrink-0 items-center gap-1.5">
                <span class="ui-badge" :class="userStatusClass(user.status)">{{ userStatusLabel(user.status) }}</span>
                <RowMenu :items="userMenuItems(user)" @select="(key) => onUserMenuSelect(user, key)" />
              </div>
            </div>
            <dl class="mt-3 divide-y divide-hairline text-sm">
              <div class="flex items-start justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.table.email }}</dt>
                <dd class="m-0 break-all text-right text-slate-600">{{ user.email || "-" }}</dd>
              </div>
              <div class="flex items-start justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.table.location }}</dt>
                <dd class="m-0 break-words text-right text-slate-600">{{ user.location || "-" }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.table.role }}</dt>
                <dd class="m-0">
                  <span class="ui-badge" :class="roleBadgeClass(user.role_label)">{{ roleLabel(user.role_label) }}</span>
                </dd>
              </div>
              <div class="flex items-center justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.table.emailVerified }}</dt>
                <dd class="m-0">
                  <span class="ui-badge" :class="user.email_verified ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'">
                    {{ user.email_verified ? copy.verified : copy.unverified }}
                  </span>
                </dd>
              </div>
              <div class="flex items-center justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.table.created }}</dt>
                <dd class="tnum m-0 text-right text-slate-600">{{ formatDate(user.created_at) }}</dd>
              </div>
            </dl>
          </article>
        </div>
      </div>

      <!-- Desktop table -->
      <div class="hidden overflow-x-auto md:block">
        <table class="w-full min-w-[1180px] text-left text-sm">
          <thead class="ui-table-head">
            <tr>
              <th class="px-4 py-3 font-medium">{{ copy.table.user }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.table.email }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.table.location }}</th>
              <th class="w-24 whitespace-nowrap px-4 py-3 font-medium">{{ copy.table.role }}</th>
              <th class="w-24 whitespace-nowrap px-4 py-3 font-medium">{{ copy.table.status }}</th>
              <th class="w-28 whitespace-nowrap px-4 py-3 font-medium">{{ copy.table.emailVerified }}</th>
              <th class="w-28 whitespace-nowrap px-4 py-3 font-medium">{{ copy.table.created }}</th>
              <th class="w-14 px-4 py-3 font-medium"><span class="sr-only">{{ common.moreActions }}</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-hairline">
            <template v-if="loading">
              <tr v-for="i in 5" :key="`sk-${i}`">
                <td class="px-4 py-3.5">
                  <div class="skeleton h-4 w-28" />
                  <div class="skeleton mt-2 h-3 w-36" />
                </td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-44" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-24" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-20" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-8 w-8 rounded-lg" /></td>
              </tr>
            </template>
            <tr v-else-if="filteredUsers.length === 0">
              <td colspan="8">
                <div class="ui-empty">
                  <div class="ui-empty-icon">
                    <Users class="h-6 w-6" />
                  </div>
                  <p class="ui-empty-title">{{ copy.noUsers }}</p>
                </div>
              </td>
            </tr>
            <template v-else>
              <tr v-for="user in filteredUsers" :key="user.id" class="ui-table-row">
                <td class="px-4 py-3.5">
                  <div class="font-medium text-ink">{{ user.name || "-" }}</div>
                  <div class="mt-1"><UlidChip :value="user.candidate_ulid || ''" /></div>
                </td>
                <td class="whitespace-nowrap px-4 py-3.5 text-slate-600">{{ user.email || "-" }}</td>
                <td class="px-4 py-3.5 text-slate-600">{{ user.location || "-" }}</td>
                <td class="w-24 whitespace-nowrap px-4 py-3.5">
                  <span class="ui-badge" :class="roleBadgeClass(user.role_label)">{{ roleLabel(user.role_label) }}</span>
                </td>
                <td class="w-24 whitespace-nowrap px-4 py-3.5">
                  <span class="ui-badge" :class="userStatusClass(user.status)">{{ userStatusLabel(user.status) }}</span>
                </td>
                <td class="w-28 whitespace-nowrap px-4 py-3.5">
                  <span class="ui-badge" :class="user.email_verified ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'">
                    {{ user.email_verified ? copy.verified : copy.unverified }}
                  </span>
                </td>
                <td class="tnum w-28 whitespace-nowrap px-4 py-3.5 text-slate-600">{{ formatDate(user.created_at) }}</td>
                <td class="w-14 px-4 py-3.5 text-right">
                  <RowMenu :items="userMenuItems(user)" @select="(key) => onUserMenuSelect(user, key)" />
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(userPage) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="loading || !canPrevUsers" @click="loadUserPage(userPage - 1)">
            {{ copy.prev }}
          </button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="loading || !canNextUsers" @click="loadUserPage(userPage + 1)">
            {{ copy.next }}
          </button>
        </div>
      </div>
    </section>

    <!-- Block cards -->
    <section class="mt-6 hidden gap-6 xl:grid xl:grid-cols-[1.1fr_0.9fr]">
      <article class="ui-card overflow-hidden">
        <div class="flex items-center justify-between gap-4 border-b border-hairline px-6 py-5">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ copy.stageDistribution }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.stageDistributionHint }}</p>
          </div>
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ totalPipelines }} {{ copy.pipelines }}</span>
        </div>
        <div v-if="loading" class="space-y-3 p-6">
          <div v-for="i in 3" :key="i" class="skeleton h-12 w-full" />
        </div>
        <div v-else-if="!data?.stage_buckets.length" class="ui-empty">
          <div class="ui-empty-icon">
            <BarChart3 class="h-6 w-6" />
          </div>
          <p class="ui-empty-title">{{ copy.noStageData }}</p>
        </div>
        <div v-else class="divide-y divide-hairline">
          <div v-for="item in data.stage_buckets" :key="`${item.stage_id}-${item.status}`" class="flex items-center justify-between gap-4 px-6 py-4">
            <div class="min-w-0">
              <p class="break-all text-sm font-medium text-ink">{{ stageLabel(item.stage_id) }}</p>
              <span class="ui-badge mt-2" :class="badgeClass(item.status)">
                {{ item.status || copy.unknownStatus }}
              </span>
            </div>
            <div class="shrink-0 text-right">
              <p class="tnum text-3xl font-bold text-ink">{{ item.count }}</p>
              <p class="text-xs text-slate-400">{{ copy.peoplePipelines }}</p>
            </div>
          </div>
        </div>
      </article>

      <article class="ui-card p-6">
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ copy.businessOverview }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.businessOverviewHint }}</p>
          </div>
          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[10px] bg-slate-50 text-slate-400">
            <BarChart3 class="h-4 w-4" />
          </span>
        </div>
        <div class="mt-6 grid gap-4 sm:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <Users class="h-4 w-4 text-slate-400" />
              <span class="text-sm font-medium">{{ copy.candidatesTotal }}</span>
            </div>
            <p class="tnum mt-4 text-3xl font-bold text-ink">{{ data?.candidate_total ?? "-" }}</p>
          </div>
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <CreditCard class="h-4 w-4 text-slate-400" />
              <span class="text-sm font-medium">{{ copy.todayRevenue }}</span>
            </div>
            <p class="tnum mt-4 text-2xl font-bold text-ink">{{ revenueText }}</p>
            <p class="tnum mt-2 text-sm text-slate-500">{{ copy.paidOrders(totalPaidOrders) }}</p>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>
