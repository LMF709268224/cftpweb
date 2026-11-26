<script setup lang="ts">
import { Activity, AlertCircle, ArrowUpRight, BarChart3, Check, ClipboardList, CreditCard, Loader2, Mail, RefreshCw, Search, X } from "lucide-vue-next"
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import CandidateEnrollmentsDialog from "@/components/CandidateEnrollmentsDialog.vue"
import CandidateRegistrations from "@/components/CandidateRegistrations.vue"
import DashboardUserActions from "@/components/DashboardUserActions.vue"
import { apiClient } from "@/lib/apiClient"
import { formatMinorAmount } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { useCandidateMemberships } from "@/lib/useCandidateMemberships"

type RevenueItem = {
  currency: string
  amount_minor: number
  order_count: number
}

type RevenueWindow = {
  currency: string
  previous_amount_minor: number
  current_amount_minor: number
}

type FunnelData = {
  candidate_accounts: number
  unlinked_accounts: number
  with_pipeline: number
  completed_pipeline: number
  exact: boolean
  aggregation_sample_limit: number
}

type PendingAgeData = {
  created_older_48h: number
  exact: boolean
  aggregation_sample_limit: number
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
  user_group: "staff" | "candidate"
  status: string
  email_verified: boolean
  mfa_enabled?: boolean
  last_login_at?: string
  created_at: string
}

type DashboardData = {
  candidate_total: number
  user_stats: UserStats
  user_role_stats: RoleStat[]
  users: DashboardUser[]
  user_total: number
  user_page: number
  user_page_size: number
  today_revenue: RevenueItem[]
  today_revenue_exact: boolean
  revenue_48h: RevenueWindow[]
  new_candidates_48h: number | null
  unverified_candidates: number
  aggregation_sample_limit: number
  generated_at: string
}

const loading = ref(false)
const reviewsLoading = ref(false)
const mailStatsLoading = ref(false)
const funnelLoading = ref(false)
const pendingAgeLoading = ref(false)
const pendingReviews = ref<{ total: number; total_label: string; total_exact: boolean } | null>(null)
const mailCounts = ref<Record<string, number> | null>(null)
const funnel = ref<FunnelData | null>(null)
const pendingAge = ref<PendingAgeData | null>(null)
const refreshing = computed(() => loading.value || reviewsLoading.value || mailStatsLoading.value || funnelLoading.value || pendingAgeLoading.value)
const data = ref<DashboardData | null>(null)
const selectedUser = ref<DashboardUser | null>(null)
const enrollmentUser = ref<DashboardUser | null>(null)
const keyword = ref("")
const userGroup = ref<"staff" | "candidate">("candidate")
const userGroups = ["candidate", "staff"] as const
const statusFilter = ref("all")
const userPage = ref(1)
const userPageSize = 10
let filterReloadTimer: number | undefined
const { lang, t } = useAdminLanguage()
const copy = computed(() => t.value.dashboard)
const memberships = useCandidateMemberships(() => data.value?.users || [])

function isCount(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0
}
const totalPaidOrders = computed(() => {
  const items = data.value?.today_revenue
  if (!Array.isArray(items) || items.some((item) => !isCount(item.order_count))) return null
  return items.reduce((sum, item) => sum + item.order_count, 0)
})
const userTotal = computed(() => data.value?.user_total)
const canPrevUsers = computed(() => userPage.value > 1)
const canNextUsers = computed(() => userTotal.value !== undefined && userPage.value * userPageSize < userTotal.value)
const revenueText = computed(() => {
  if (!data.value) return copy.value.unavailable
  const items = data.value.today_revenue
  if (!Array.isArray(items) || typeof data.value.today_revenue_exact !== "boolean" || items.some((item) => typeof item.currency !== "string" || !item.currency || !Number.isSafeInteger(item.amount_minor))) return copy.value.unavailable
  if (!items.length) return copy.value.noRevenue
  return items.map((item) => `${item.currency} ${formatMinorAmount(item.amount_minor) ?? "-"}`).join(" / ")
})
const revenueWindows = computed(() => {
  if (!data.value?.today_revenue_exact || !Array.isArray(data.value.revenue_48h)) return null
  const windows = data.value.revenue_48h
  if (windows.some((item) => !item.currency || !Number.isSafeInteger(item.previous_amount_minor) || !Number.isSafeInteger(item.current_amount_minor))) return null
  return windows
})
function revenueChange(item: RevenueWindow) {
  if (item.previous_amount_minor === 0) return item.current_amount_minor > 0 ? copy.value.newRevenue : "0%"
  const percent = Math.round((item.current_amount_minor - item.previous_amount_minor) / item.previous_amount_minor * 100)
  return `${percent > 0 ? "+" : ""}${percent}%`
}
function revenueBarWidth(amount: number, item: RevenueWindow) {
  const max = Math.max(item.previous_amount_minor, item.current_amount_minor, 1)
  return `${Math.round(amount / max * 100)}%`
}
const filteredUsers = computed(() => {
  return data.value?.users || []
})

const mailCards = computed(() => [
  { status: "SENT", label: t.value.mailsAdmin.statusLabels.sent },
  { status: "SCHEDULING", label: t.value.mailsAdmin.statusLabels.scheduling },
  { status: "FAILED", label: t.value.mailsAdmin.statusLabels.failed },
  { status: "CANCELLED", label: t.value.mailsAdmin.statusLabels.cancelled },
].map((item) => ({ ...item, count: mailCounts.value?.[item.status] })))

function groupCount(group: "staff" | "candidate") {
  const count = group === "staff" ? data.value?.user_stats?.admins : data.value?.candidate_total
  return isCount(count) ? count : copy.value.unavailable
}

async function loadPendingReviews() {
  reviewsLoading.value = true
  pendingReviews.value = null
  try {
    const result = await apiClient<{ total: number; total_label: string; total_exact: boolean }>("/api/applications?status=Pending&page_size=1&include_status_subtotals=false")
    if (!Number.isSafeInteger(result.total) || result.total < 0 || typeof result.total_exact !== "boolean" || typeof result.total_label !== "string") {
      throw new Error("Invalid application count response")
    }
    pendingReviews.value = result
  } catch (err) {
    console.error(err)
  } finally {
    reviewsLoading.value = false
  }
}

async function loadMailStats() {
  mailStatsLoading.value = true
  mailCounts.value = null
  try {
    const result = await apiClient<{ status_counts?: Record<string, number> }>("/api/mails/stats")
    const counts = result?.status_counts
    if (!counts || typeof counts !== "object" || Array.isArray(counts) || Object.values(counts).some((count) => !isCount(count))) {
      throw new Error("Invalid mail statistics response")
    }
    mailCounts.value = counts
  } catch (err) {
    console.error(err)
  } finally {
    mailStatsLoading.value = false
  }
}

async function loadFunnel() {
  funnelLoading.value = true
  funnel.value = null
  try {
    const result = await apiClient<FunnelData>("/api/dashboard/funnel")
    if (!result || !isCount(result.candidate_accounts) || !isCount(result.unlinked_accounts) || !isCount(result.with_pipeline) || !isCount(result.completed_pipeline) || typeof result.exact !== "boolean" || !isCount(result.aggregation_sample_limit)) {
      throw new Error("Invalid dashboard funnel response")
    }
    funnel.value = result
  } catch (err) {
    console.error(err)
  } finally {
    funnelLoading.value = false
  }
}

async function loadPendingAge() {
  pendingAgeLoading.value = true
  pendingAge.value = null
  try {
    const result = await apiClient<PendingAgeData>("/api/dashboard/pending-age")
    if (!result || !isCount(result.created_older_48h) || typeof result.exact !== "boolean" || !isCount(result.aggregation_sample_limit)) {
      throw new Error("Invalid dashboard pending age response")
    }
    pendingAge.value = result
  } catch (err) {
    console.error(err)
  } finally {
    pendingAgeLoading.value = false
  }
}

async function refreshDashboard() {
  await Promise.all([loadDashboard(), loadPendingReviews(), loadMailStats(), loadFunnel(), loadPendingAge()])
}

function selectUserGroup(group: "staff" | "candidate") {
  userGroup.value = group
}

function membershipLabel(user: DashboardUser) {
  if (user.user_group !== "candidate") return "—"
  if (!user.candidate_ulid) return copy.value.identityMissing
  const membership = memberships.value[user.candidate_ulid]
  if (!membership || membership.state === "loading") return copy.value.loading
  if (membership.state === "error") return copy.value.unavailable
  if (membership.state === "none") return copy.value.noActiveMembership
  return membership.name || copy.value.unavailable
}

function roleBadgeClass(role: string) {
  const normalized = role.toLowerCase()
  if (normalized.includes("admin")) return "bg-blue-100 text-blue-700"
  if (normalized.includes("student")) return "bg-emerald-100 text-emerald-700"
  return "bg-slate-100 text-slate-600"
}

function roleLabel(role: string) {
  const normalized = role.toLowerCase()
  if (normalized.includes("student")) return copy.value.roles.student
  if (normalized.includes("admin")) return copy.value.roles.admin
  return role || "-"
}

function userStatusClass(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === "active") return "bg-blue-100 text-blue-700"
  if (normalized === "inactive") return "bg-amber-100 text-amber-700"
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

function lastLoginLabel(user: DashboardUser) {
  if (typeof user.last_login_at !== "string") return copy.value.unavailable
  if (user.last_login_at === "") return copy.value.noLoginRecord
  const parsed = new Date(user.last_login_at)
  return Number.isNaN(parsed.getTime()) ? copy.value.unavailable : parsed.toLocaleString(lang.value === "zh" ? "zh-CN" : "en-US")
}

function mfaLabel(user: DashboardUser) {
  if (typeof user.mfa_enabled !== "boolean") return copy.value.unavailable
  return user.mfa_enabled ? copy.value.mfaEnabled : copy.value.mfaDisabled
}

function verificationLabel(user: DashboardUser) {
  if (typeof user.email_verified !== "boolean") return copy.value.unavailable
  return user.email_verified ? copy.value.verified : copy.value.unverified
}

const userDetailFields = computed(() => {
  const user = selectedUser.value
  if (!user) return []
  return [
    { label: copy.value.table.user, value: user.name || "-" },
    { label: copy.value.table.email, value: user.email || "-" },
    { label: copy.value.phone, value: user.phone || "-" },
    { label: copy.value.location, value: user.location || "-" },
    { label: copy.value.table.role, value: roleLabel(user.role_label) },
    ...(user.user_group === "candidate" ? [
      { label: copy.value.membership, value: membershipLabel(user) },
      ...((memberships.value[user.candidate_ulid || ""]?.state === "active") ? [
        { label: copy.value.membershipStarts, value: formatDate(memberships.value[user.candidate_ulid || ""]?.record?.started_at || "") },
        { label: copy.value.membershipExpires, value: formatDate(memberships.value[user.candidate_ulid || ""]?.record?.expires_at || "") },
      ] : []),
    ] : []),
    { label: copy.value.table.status, value: userStatusLabel(user.status) },
    { label: copy.value.table.emailVerified, value: verificationLabel(user) },
    ...(user.user_group === "staff" ? [{ label: copy.value.mfa, value: mfaLabel(user) }] : []),
    { label: copy.value.lastLogin, value: lastLoginLabel(user) },
    { label: copy.value.table.created, value: formatDate(user.created_at) },
  ]
})

function closeUserDetail() {
  selectedUser.value = null
}

let dashboardRequestId = 0

async function loadDashboard(page = userPage.value) {
  const requestId = ++dashboardRequestId
  loading.value = true
  try {
    const params = new URLSearchParams({
      user_page: String(page),
      user_page_size: String(userPageSize),
    })
    const normalizedKeyword = keyword.value.trim()
    if (normalizedKeyword) params.set("user_keyword", normalizedKeyword)
    params.set("user_group", userGroup.value)
    if (statusFilter.value !== "all") params.set("user_status", statusFilter.value)
    const response = await apiClient<DashboardData>(`/api/dashboard/ops?${params}`)
    if (requestId !== dashboardRequestId) return
    if (!response || !Array.isArray(response.users) || !isCount(response.user_total) || !isCount(response.user_page) || response.user_page < 1 || response.user_page_size !== userPageSize) throw new Error("Incomplete dashboard user page")
    data.value = response
    userPage.value = response.user_page
  } catch (err) {
    if (requestId !== dashboardRequestId) return
    data.value = null
    console.error(err)
    toast.error(copy.value.loadFailed)
  } finally {
    if (requestId === dashboardRequestId) loading.value = false
  }
}

function loadUserPage(page: number) {
  if (page < 1) return
  void loadDashboard(page)
}

function clearUserKeyword() {
  if (!keyword.value) return
  keyword.value = ""
}

onMounted(refreshDashboard)

watch([keyword, userGroup, statusFilter], () => {
  if (filterReloadTimer) window.clearTimeout(filterReloadTimer)
  filterReloadTimer = window.setTimeout(() => {
    filterReloadTimer = undefined
    void loadDashboard(1)
  }, 250)
})

onUnmounted(() => {
  if (filterReloadTimer) window.clearTimeout(filterReloadTimer)
  filterReloadTimer = undefined
  dashboardRequestId += 1
})
</script>

<template>
  <main class="mx-auto max-w-[1600px] px-4 py-5 md:px-6 md:py-8">
    <header class="mb-5 flex flex-wrap items-start justify-between gap-4 md:mb-6 md:items-center">
      <div class="min-w-0">
        <div class="mb-2 text-xs font-black uppercase tracking-wide text-slate-400">{{ copy.eyebrow }}</div>
        <h1 class="text-3xl font-black tracking-tight text-slate-950 md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-sm text-slate-500">{{ copy.subtitle }}</p>
      </div>
      <div class="grid w-full grid-cols-2 gap-3 sm:flex sm:w-auto sm:flex-wrap">
        <RouterLink to="/audit/logs" class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold shadow-sm hover:border-slate-400">
          <Activity class="h-4 w-4" />
          {{ copy.auditLogs }}
        </RouterLink>
        <RouterLink to="/mails?tab=sent" class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold shadow-sm hover:border-slate-400">
          <Mail class="h-4 w-4" />
          {{ copy.emailActivity }}
        </RouterLink>
        <button
          class="col-span-2 inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-slate-300 bg-white px-5 font-bold shadow-sm transition hover:border-slate-500 disabled:opacity-50 sm:col-span-1"
          type="button"
          :disabled="refreshing"
          @click="refreshDashboard"
        >
          <Loader2 v-if="refreshing" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="mb-5" :aria-label="copy.keyMetrics">
      <h2 class="mb-3 text-lg font-black">{{ copy.keyMetrics }}</h2>
      <div class="grid gap-4 md:grid-cols-2">
        <RouterLink to="/orders" :aria-label="copy.viewOrders" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-blue-400 hover:shadow-md" data-testid="dashboard-revenue">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-bold text-slate-600">{{ copy.todayRevenue }}</h3>
            <CreditCard class="h-5 w-5 text-blue-600" />
          </div>
          <p class="mt-4 break-words text-3xl font-black">{{ loading ? copy.loading : revenueText }}</p>
          <p v-if="totalPaidOrders !== null" class="mt-2 text-sm text-slate-500">{{ copy.paidOrders(totalPaidOrders) }}</p>
          <p v-if="data?.today_revenue_exact === false && isCount(data.aggregation_sample_limit)" class="mt-2 text-xs font-bold text-amber-700">{{ copy.sampledAggregation(data.aggregation_sample_limit) }}</p>
          <div v-if="revenueWindows?.length" class="mt-4 space-y-3 border-t border-slate-100 pt-3">
            <p class="text-xs font-semibold text-slate-500">{{ copy.revenueTrend48h }}</p>
            <div v-for="item in revenueWindows" :key="item.currency" class="space-y-1 text-xs">
              <div class="flex justify-between gap-2 font-semibold"><span>{{ item.currency }}</span><span>{{ revenueChange(item) }}</span></div>
              <div class="flex items-center gap-2"><span class="w-20 shrink-0 text-slate-500">{{ copy.previous24h }}</span><div class="h-2 flex-1 rounded bg-slate-100"><div class="h-full rounded bg-slate-400" :style="{ width: revenueBarWidth(item.previous_amount_minor, item) }" /></div><span class="w-16 text-right">{{ formatMinorAmount(item.previous_amount_minor) }}</span></div>
              <div class="flex items-center gap-2"><span class="w-20 shrink-0 text-slate-500">{{ copy.current24h }}</span><div class="h-2 flex-1 rounded bg-slate-100"><div class="h-full rounded bg-blue-500" :style="{ width: revenueBarWidth(item.current_amount_minor, item) }" /></div><span class="w-16 text-right">{{ formatMinorAmount(item.current_amount_minor) }}</span></div>
            </div>
          </div>
          <p v-else-if="data && typeof data.today_revenue_exact === 'boolean'" class="mt-3 text-xs text-slate-500">{{ data.today_revenue_exact === false ? copy.trendUnavailable : revenueWindows ? copy.noRevenue48h : copy.unavailable }}</p>
          <span class="mt-4 inline-flex items-center gap-1 text-sm font-bold text-blue-700">{{ copy.viewOrders }} <ArrowUpRight class="h-4 w-4" /></span>
        </RouterLink>
        <RouterLink to="/applications" :aria-label="copy.viewReviews" class="rounded-2xl border bg-white p-5 shadow-sm transition hover:border-blue-400 hover:shadow-md" :class="pendingReviews && pendingReviews.total > 0 ? 'border-amber-300' : 'border-slate-200'" data-testid="dashboard-reviews">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-bold text-slate-600">{{ copy.pendingReviews }}</h3>
            <ClipboardList class="h-5 w-5 text-amber-600" />
          </div>
          <p class="mt-4 text-3xl font-black">{{ reviewsLoading ? copy.loading : pendingReviews?.total_label ?? copy.unavailable }}</p>
          <p v-if="pendingReviews" class="mt-2 text-sm" :class="pendingReviews.total > 0 ? 'text-amber-700' : 'text-slate-500'">{{ pendingReviews.total > 0 ? copy.reviewsWaiting : copy.noPendingReviews }}</p>
          <p v-if="pendingReviews && !pendingReviews.total_exact" class="mt-2 text-xs font-bold text-amber-700">{{ copy.partialReviewCount }}</p>
          <div v-if="pendingAge" class="mt-3 border-t border-slate-100 pt-3" data-testid="dashboard-pending-age">
            <p class="text-sm font-bold" :class="pendingAge.created_older_48h > 0 ? 'text-amber-700' : 'text-slate-600'">{{ copy.pendingCreatedOlder48h }}: {{ pendingAge.exact ? '' : '≥' }}{{ pendingAge.created_older_48h }}</p>
            <p class="mt-1 text-xs text-slate-500">{{ copy.pendingAgeEstimateHint }}</p>
          </div>
          <span class="mt-4 inline-flex items-center gap-1 text-sm font-bold text-blue-700">{{ copy.viewReviews }} <ArrowUpRight class="h-4 w-4" /></span>
        </RouterLink>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="border-b border-slate-100 px-4 py-4 md:px-6 md:py-5">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div class="min-w-0">
            <h2 class="text-xl font-black text-slate-950">{{ copy.userManagement }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.userManagementHint }}</p>
          </div>
          <span v-if="data" class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">{{ copy.userPageText(userPage, data.user_total) }}</span>
        </div>
        <div class="mt-4 inline-flex max-w-full gap-1 overflow-x-auto rounded-lg border border-slate-200 bg-slate-50 p-1" :aria-label="copy.userManagement">
          <button v-for="group in userGroups" :key="group" type="button" class="whitespace-nowrap rounded-md px-3 py-2 text-sm font-bold" :class="userGroup === group ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-600 hover:text-slate-950'" :aria-label="copy.groups[group]" :aria-pressed="userGroup === group" @click="selectUserGroup(group)">
            {{ copy.groups[group] }}
            <span v-if="data" aria-hidden="true" class="ml-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs">{{ groupCount(group) }}</span>
          </button>
        </div>
        <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,2fr)_1fr]">
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.table.user }}
            <div class="relative">
              <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input v-model="keyword" class="h-11 w-full rounded-xl border border-slate-200 bg-white pl-10 pr-10" :placeholder="copy.filters.searchPlaceholder" />
              <button
                v-if="keyword"
                class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
                type="button"
                :aria-label="copy.filters.clearInput"
                :title="copy.filters.clearInput"
                @click="clearUserKeyword"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
          </label>
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.table.status }}
            <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 bg-white px-3">
              <option value="all">{{ copy.filters.allStatus }}</option>
              <option value="active">{{ copy.filters.active }}</option>
              <option value="inactive">{{ copy.filters.inactive }}</option>
              <option value="deleted">{{ copy.filters.deleted }}</option>
            </select>
          </label>
        </div>
      </div>

      <div class="md:hidden">
        <div v-if="loading" class="px-4 py-10 text-center text-slate-400">
          <Loader2 class="mr-2 inline h-5 w-5 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!data" class="px-4 py-10 text-center text-slate-500" role="alert">{{ copy.loadFailed }}</div>
        <div v-else-if="filteredUsers.length === 0" class="px-4 py-10 text-center text-slate-400">{{ copy.noUsers }}</div>
        <div v-else class="divide-y divide-slate-100">
          <article v-for="user in filteredUsers" :key="user.id" class="px-4 py-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="break-words font-bold text-slate-950">{{ user.name || "-" }}</div>
                <div class="mt-1 break-all text-xs text-slate-500">{{ user.candidate_ulid || "-" }}</div>
              </div>
            </div>
            <div class="mt-3 grid gap-2 text-sm">
              <div class="flex items-start justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.table.email }}</span>
                <span class="break-all text-right text-slate-700">{{ user.email || "-" }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.table.role }}</span>
                <span class="flex flex-wrap justify-end gap-1">
                  <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="roleBadgeClass(user.role_label)">{{ roleLabel(user.role_label) }}</span>
                </span>
              </div>
              <div v-if="userGroup === 'candidate'" class="flex items-start justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.membership }}</span>
                <span class="text-right text-sm text-slate-700">{{ membershipLabel(user) }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.table.status }}</span>
                <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="userStatusClass(user.status)">{{ userStatusLabel(user.status) }}</span>
              </div>
              <div v-if="userGroup === 'staff'" class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.mfa }}</span>
                <span class="text-right text-slate-700">{{ mfaLabel(user) }}</span>
              </div>
              <div v-else class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.table.emailVerified }}</span>
                <span v-if="typeof user.email_verified !== 'boolean'" class="text-slate-700">{{ copy.unavailable }}</span>
                <span v-else :title="verificationLabel(user)" :class="['inline-flex rounded-full p-1 text-xs font-bold', user.email_verified ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700']">
                  <Check v-if="user.email_verified" class="h-4 w-4" aria-hidden="true" />
                  <X v-else class="h-4 w-4" aria-hidden="true" />
                  <span class="sr-only">{{ user.email_verified ? copy.verified : copy.unverified }}</span>
                </span>
              </div>
              <div class="flex items-start justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.lastLogin }}</span>
                <span class="text-right text-slate-700">{{ lastLoginLabel(user) }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.table.created }}</span>
                <span class="text-right text-slate-700">{{ formatDate(user.created_at) }}</span>
              </div>
            </div>
            <DashboardUserActions class="mt-3" :can-view-enrollments="user.user_group === 'candidate' && !!user.candidate_ulid" @details="selectedUser = user" @enrollments="enrollmentUser = user" />
          </article>
        </div>
      </div>

      <div class="hidden overflow-x-auto md:block">
        <table class="w-full min-w-[960px] text-left text-sm">
          <thead class="border-b border-slate-100 bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-6 py-4">{{ copy.table.user }}</th>
              <th class="px-6 py-4">{{ copy.table.email }}</th>
              <th class="w-24 whitespace-nowrap px-6 py-4">{{ copy.table.role }}</th>
              <th v-if="userGroup === 'candidate'" class="whitespace-nowrap px-6 py-4">{{ copy.membership }}</th>
              <th class="w-24 whitespace-nowrap px-6 py-4">{{ copy.table.status }}</th>
              <th class="w-28 whitespace-nowrap px-6 py-4">{{ userGroup === 'staff' ? copy.mfa : copy.table.emailVerified }}</th>
              <th class="whitespace-nowrap px-6 py-4">{{ copy.lastLogin }}</th>
              <th class="w-28 whitespace-nowrap px-6 py-4">{{ copy.table.created }}</th>
              <th class="w-28 whitespace-nowrap px-6 py-4">{{ copy.table.actions }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="loading">
              <td :colspan="userGroup === 'candidate' ? 9 : 8" class="px-6 py-10 text-center text-slate-400">
                <Loader2 class="mr-2 inline h-5 w-5 animate-spin" />
                {{ copy.loading }}
              </td>
            </tr>
            <tr v-else-if="!data">
              <td :colspan="userGroup === 'candidate' ? 9 : 8" class="px-6 py-10 text-center text-slate-500" role="alert">{{ copy.loadFailed }}</td>
            </tr>
            <tr v-else-if="filteredUsers.length === 0">
              <td :colspan="userGroup === 'candidate' ? 9 : 8" class="px-6 py-10 text-center text-slate-400">{{ copy.noUsers }}</td>
            </tr>
            <template v-else>
              <tr v-for="user in filteredUsers" :key="user.id" class="transition hover:bg-sky-50/60">
                <td class="px-6 py-4">
                  <div class="font-bold text-slate-950">{{ user.name || "-" }}</div>
                  <div class="text-xs text-slate-500">{{ user.candidate_ulid || "-" }}</div>
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-slate-700">{{ user.email || "-" }}</td>
                <td class="w-24 whitespace-nowrap px-6 py-4">
                  <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="roleBadgeClass(user.role_label)">{{ roleLabel(user.role_label) }}</span>
                </td>
                <td v-if="userGroup === 'candidate'" class="min-w-36 px-6 py-4 text-slate-700">{{ membershipLabel(user) }}</td>
                <td class="w-24 whitespace-nowrap px-6 py-4">
                  <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="userStatusClass(user.status)">{{ userStatusLabel(user.status) }}</span>
                </td>
                <td class="w-28 whitespace-nowrap px-6 py-4">
                  <span v-if="userGroup === 'staff'" class="text-slate-700">{{ mfaLabel(user) }}</span>
                  <span v-else-if="typeof user.email_verified !== 'boolean'" class="text-slate-700">{{ copy.unavailable }}</span>
                  <span v-else :title="verificationLabel(user)" :class="['inline-flex rounded-full p-1 text-xs font-bold', user.email_verified ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700']">
                    <Check v-if="user.email_verified" class="h-4 w-4" aria-hidden="true" />
                    <X v-else class="h-4 w-4" aria-hidden="true" />
                    <span class="sr-only">{{ user.email_verified ? copy.verified : copy.unverified }}</span>
                  </span>
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-slate-700">{{ lastLoginLabel(user) }}</td>
                <td class="w-28 whitespace-nowrap px-6 py-4 text-slate-700">{{ formatDate(user.created_at) }}</td>
                <td class="w-28 whitespace-nowrap px-6 py-4">
                  <DashboardUserActions :can-view-enrollments="user.user_group === 'candidate' && !!user.candidate_ulid" @details="selectedUser = user" @enrollments="enrollmentUser = user" />
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <div class="flex flex-col items-stretch justify-end gap-3 border-t border-slate-100 px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <button class="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="loading || !canPrevUsers" @click="loadUserPage(userPage - 1)">
          {{ copy.prev }}
        </button>
        <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.pageText(userPage) }}</span>
        <button class="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="loading || !canNextUsers" @click="loadUserPage(userPage + 1)">
          {{ copy.next }}
        </button>
      </div>
    </section>

    <section class="mt-7 border-t border-slate-200 pt-6" data-testid="dashboard-funnel">
      <div class="flex flex-wrap items-start justify-between gap-2">
        <div><h2 class="text-xl font-black">{{ copy.funnelTitle }}</h2><p class="mt-1 text-sm text-slate-500">{{ copy.funnelHint }}</p></div>
        <p v-if="funnel && !funnel.exact" class="text-xs font-semibold text-amber-700">{{ copy.funnelPartial(funnel.aggregation_sample_limit, funnel.unlinked_accounts) }}</p>
      </div>
      <p v-if="funnelLoading" class="mt-4 text-sm text-slate-500">{{ copy.loading }}</p>
      <p v-else-if="!funnel" class="mt-4 text-sm text-slate-500">{{ copy.unavailable }}</p>
      <div v-else class="mt-4 grid gap-3 md:grid-cols-3">
        <div v-for="(step, index) in [
          { label: copy.funnelAccounts, count: funnel.candidate_accounts },
          { label: copy.funnelWithPipeline, count: funnel.with_pipeline },
          { label: copy.funnelCompleted, count: funnel.completed_pipeline },
        ]" :key="index" class="border-l-4 border-blue-500 bg-white p-4">
          <p class="text-sm text-slate-600">{{ step.label }}</p>
          <p class="mt-2 text-2xl font-black">{{ index > 0 && !funnel.exact ? '≥' : '' }}{{ step.count }}</p>
        </div>
      </div>
    </section>

    <section class="mt-7 border-t border-slate-200 pt-6">
      <div data-testid="dashboard-business">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-black">{{ copy.businessOverview }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.businessOverviewHint }}</p>
          </div>
          <BarChart3 class="h-6 w-6 text-slate-400" />
        </div>
        <div class="mt-5 grid gap-3 sm:grid-cols-2">
          <div class="border-l-4 border-emerald-500 bg-white p-4">
            <p class="text-sm text-slate-600">{{ copy.newCandidates48h }}</p>
            <p class="mt-2 text-2xl font-black">{{ loading ? copy.loading : isCount(data?.new_candidates_48h) ? data?.new_candidates_48h : copy.unavailable }}</p>
          </div>
          <div class="border-l-4 border-amber-500 bg-white p-4">
            <p class="text-sm text-slate-600">{{ copy.unverifiedCandidates }}</p>
            <p class="mt-2 text-2xl font-black">{{ loading ? copy.loading : isCount(data?.unverified_candidates) ? data?.unverified_candidates : copy.unavailable }}</p>
          </div>
        </div>
        <div class="mt-6">
          <h3 class="font-bold">{{ copy.emailStatus }}</h3>
          <p class="mt-1 text-xs text-slate-500">{{ copy.emailStatusHint }}</p>
          <p v-if="mailStatsLoading" class="py-8 text-center text-slate-500">{{ copy.loading }}</p>
          <p v-else-if="!mailCounts" class="py-8 text-center text-slate-500">{{ copy.unavailable }}</p>
          <div v-else class="mt-4 grid grid-cols-2 gap-3 md:grid-cols-4">
            <RouterLink v-for="card in mailCards" :key="card.status" :to="{ path: '/mails', query: { tab: 'sent', status: card.status } }" class="rounded-lg border p-4 transition hover:border-blue-400" :class="card.status === 'FAILED' && card.count !== undefined && card.count > 0 ? 'border-red-200 bg-red-50 text-red-700' : 'border-slate-200 bg-white'">
              <div class="flex items-center justify-between gap-2 text-sm font-bold">{{ card.label }} <ArrowUpRight class="h-4 w-4 shrink-0" /></div>
              <p class="mt-2 text-2xl font-black">{{ card.count === undefined ? copy.unavailable : card.count }}</p>
            </RouterLink>
          </div>
          <p v-if="mailCounts && mailCounts.FAILED > 0" class="mt-4 flex items-start gap-2 rounded-xl bg-red-50 p-3 text-sm text-red-700">
            <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" /> {{ copy.failedMailNotice }}
          </p>
        </div>
      </div>
    </section>
    <CandidateEnrollmentsDialog v-if="enrollmentUser?.candidate_ulid" :key="enrollmentUser.candidate_ulid" :candidate-id="enrollmentUser.candidate_ulid" :candidate-name="enrollmentUser.name" @close="enrollmentUser = null" />
    <Teleport to="body">
      <div v-if="selectedUser" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4" @click.self="closeUserDetail">
        <section v-modal-dialog="closeUserDetail" class="flex max-h-[90dvh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl" aria-labelledby="dashboard-user-detail-title">
          <header class="flex shrink-0 items-center justify-between gap-4 border-b border-slate-100 px-5 py-4">
            <h2 id="dashboard-user-detail-title" class="text-xl font-black text-slate-950">{{ copy.userDetailTitle }}</h2>
            <button data-dialog-initial-focus class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" type="button" :aria-label="copy.close" @click="closeUserDetail">
              <X class="h-5 w-5" aria-hidden="true" />
            </button>
          </header>
          <div class="overflow-y-auto px-5">
            <dl class="divide-y divide-slate-100">
              <div v-for="field in userDetailFields" :key="field.label" class="grid gap-1 py-4 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4">
                <dt class="text-sm font-bold text-slate-500">{{ field.label }}</dt>
                <dd class="min-w-0 break-words text-sm text-slate-950 [overflow-wrap:anywhere]">{{ field.value }}</dd>
              </div>
            </dl>
            <CandidateRegistrations v-if="selectedUser.user_group === 'candidate' && selectedUser.candidate_ulid" :key="selectedUser.candidate_ulid" :candidate-id="selectedUser.candidate_ulid" />
            <p v-else-if="selectedUser.user_group === 'candidate'" class="border-t py-4 text-sm text-slate-500">{{ copy.identityMissing }}</p>
          </div>
        </section>
      </div>
    </Teleport>
  </main>
</template>
