<script setup lang="ts">
import { Activity, BarChart3, CreditCard, Loader2, Mail, RefreshCw, Search, Shield, UserCheck, Users } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
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
let filterReloadTimer: number | undefined
const { lang, t } = useAdminLanguage()
const copy = computed(() => t.value.dashboard)

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
  { label: copy.value.summary.totalUsers, value: userStats.value.total, tone: "text-slate-950", icon: Users },
  { label: copy.value.summary.activeUsers, value: userStats.value.active, tone: "text-emerald-600", icon: UserCheck },
  { label: copy.value.summary.admins, value: userStats.value.admins, tone: "text-blue-600", icon: Shield },
  { label: copy.value.summary.students, value: data.value?.candidate_total ?? 0, tone: "text-[#0b579b]", icon: Users },
  { label: copy.value.summary.members, value: userStats.value.members, tone: "text-cyan-600", icon: Users },
])

function stageLabel(stageId: string) {
  return stageId === copy.value.noStage ? copy.value.noStage : stageId
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
  if (normalized.includes("member")) return copy.value.roles.member
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

function loadUserPage(page: number) {
  if (page < 1) return
  void loadDashboard(page)
}

onMounted(loadDashboard)

watch([keyword, roleFilter, statusFilter], () => {
  if (filterReloadTimer) window.clearTimeout(filterReloadTimer)
  filterReloadTimer = window.setTimeout(() => {
    void loadDashboard(1)
  }, 250)
})
</script>

<template>
  <main class="mx-auto max-w-[1600px] px-6 py-8">
    <header class="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="mb-2 text-xs font-black uppercase tracking-wide text-slate-400">{{ copy.eyebrow }}</div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 text-sm text-slate-500">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <RouterLink to="/audit/webhooks" class="inline-flex h-11 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold shadow-sm hover:border-slate-400">
          <Activity class="h-4 w-4" />
          {{ copy.auditLogs }}
        </RouterLink>
        <RouterLink to="/mails" class="inline-flex h-11 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold shadow-sm hover:border-slate-400">
          <Mail class="h-4 w-4" />
          {{ copy.emailActivity }}
        </RouterLink>
        <button
          class="inline-flex h-11 items-center gap-2 rounded-xl border border-slate-300 bg-white px-5 font-bold shadow-sm transition hover:border-slate-500 disabled:opacity-50"
          type="button"
          :disabled="loading"
          @click="loadDashboard(userPage)"
        >
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="mb-5 rounded-2xl border border-slate-200 bg-white px-5 py-4 shadow-sm">
      <div class="mb-2 flex items-center justify-between text-sm font-black text-slate-700">
        <span>{{ copy.profileCompletion }}</span>
        <span class="text-[#0b579b]">{{ profileCompletion }}%</span>
      </div>
      <div class="h-2 overflow-hidden rounded-full bg-slate-100">
        <div class="h-full rounded-full bg-[#0b579b] transition-all" :style="{ width: `${profileCompletion}%` }" />
      </div>
    </section>

    <section class="mb-4 grid gap-4 md:grid-cols-2 xl:grid-cols-5">
      <article v-for="card in summaryCards" :key="card.label" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="flex items-center justify-between">
          <p class="text-sm font-bold text-slate-600">{{ card.label }}</p>
          <component :is="card.icon" class="h-5 w-5 text-slate-300" />
        </div>
        <p class="mt-5 text-4xl font-black" :class="card.tone">{{ card.value }}</p>
      </article>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="border-b border-slate-100 px-6 py-5">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h2 class="text-xl font-black text-slate-950">{{ copy.userManagement }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.userManagementHint }}</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">{{ copy.userPageText(userPage, userTotal) }}</span>
        </div>
        <div class="mt-5 flex flex-wrap gap-3 rounded-2xl bg-slate-50 p-3">
          <div class="relative min-w-[280px] flex-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input v-model="keyword" class="h-10 w-full rounded-xl border border-slate-200 bg-white pl-10 pr-4 text-sm outline-none focus:border-[#0b579b]" :placeholder="copy.filters.searchPlaceholder" />
          </div>
          <select v-model="roleFilter" class="h-10 rounded-xl border border-slate-200 bg-white px-4 text-sm outline-none focus:border-[#0b579b]">
            <option v-for="option in roleOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
          <select v-model="statusFilter" class="h-10 rounded-xl border border-slate-200 bg-white px-4 text-sm outline-none focus:border-[#0b579b]">
            <option value="all">{{ copy.filters.allStatus }}</option>
            <option value="active">{{ copy.filters.active }}</option>
            <option value="inactive">{{ copy.filters.inactive }}</option>
            <option value="deleted">{{ copy.filters.deleted }}</option>
          </select>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full min-w-[1120px] text-left text-sm">
          <thead class="border-b border-slate-100 bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-6 py-4">{{ copy.table.user }}</th>
              <th class="px-6 py-4">{{ copy.table.email }}</th>
              <th class="px-6 py-4">{{ copy.table.location }}</th>
              <th class="w-24 whitespace-nowrap px-6 py-4">{{ copy.table.role }}</th>
              <th class="w-24 whitespace-nowrap px-6 py-4">{{ copy.table.status }}</th>
              <th class="w-28 whitespace-nowrap px-6 py-4">{{ copy.table.emailVerified }}</th>
              <th class="w-28 whitespace-nowrap px-6 py-4">{{ copy.table.created }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="loading">
              <td colspan="7" class="px-6 py-10 text-center text-slate-400">
                <Loader2 class="mr-2 inline h-5 w-5 animate-spin" />
                {{ copy.loading }}
              </td>
            </tr>
            <tr v-else-if="filteredUsers.length === 0">
              <td colspan="7" class="px-6 py-10 text-center text-slate-400">{{ copy.noUsers }}</td>
            </tr>
            <template v-else>
              <tr v-for="user in filteredUsers" :key="user.id" class="transition hover:bg-sky-50/60">
                <td class="px-6 py-4">
                  <div class="font-bold text-slate-950">{{ user.name || "-" }}</div>
                  <div class="text-xs text-slate-500">{{ user.phone || user.candidate_ulid || "-" }}</div>
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-slate-700">{{ user.email || "-" }}</td>
                <td class="px-6 py-4 text-slate-700">{{ user.location || "-" }}</td>
                <td class="w-24 whitespace-nowrap px-6 py-4">
                  <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="roleBadgeClass(user.role_label)">{{ roleLabel(user.role_label) }}</span>
                </td>
                <td class="w-24 whitespace-nowrap px-6 py-4">
                  <span class="inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold" :class="userStatusClass(user.status)">{{ userStatusLabel(user.status) }}</span>
                </td>
                <td class="w-28 whitespace-nowrap px-6 py-4">
                  <span :class="['inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-bold', user.email_verified ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700']">
                    {{ user.email_verified ? copy.verified : copy.unverified }}
                  </span>
                </td>
                <td class="w-28 whitespace-nowrap px-6 py-4 text-slate-700">{{ formatDate(user.created_at) }}</td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <div class="flex items-center justify-end gap-3 border-t border-slate-100 px-6 py-4">
        <button class="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="loading || !canPrevUsers" @click="loadUserPage(userPage - 1)">
          {{ copy.prev }}
        </button>
        <span class="text-sm font-bold text-slate-500">{{ copy.pageText(userPage) }}</span>
        <button class="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="loading || !canNextUsers" @click="loadUserPage(userPage + 1)">
          {{ copy.next }}
        </button>
      </div>
    </section>

    <section class="mt-7 hidden gap-6 xl:grid-cols-[1.1fr_0.9fr]">
      <article class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 p-6">
          <div>
            <h2 class="text-xl font-black">{{ copy.stageDistribution }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.stageDistributionHint }}</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{{ totalPipelines }} {{ copy.pipelines }}</span>
        </div>
        <div v-if="loading" class="flex h-48 items-center justify-center text-slate-400">
          <Loader2 class="mr-2 h-5 w-5 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!data?.stage_buckets.length" class="flex h-48 items-center justify-center text-slate-400">
          {{ copy.noStageData }}
        </div>
        <div v-else class="divide-y divide-slate-100">
          <div v-for="item in data.stage_buckets" :key="`${item.stage_id}-${item.status}`" class="flex items-center justify-between gap-4 p-5">
            <div class="min-w-0">
              <p class="break-all text-base font-black">{{ stageLabel(item.stage_id) }}</p>
              <span class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-bold" :class="badgeClass(item.status)">
                {{ item.status || copy.unknownStatus }}
              </span>
            </div>
            <div class="text-right">
              <p class="text-3xl font-black">{{ item.count }}</p>
              <p class="text-xs text-slate-500">{{ copy.peoplePipelines }}</p>
            </div>
          </div>
        </div>
      </article>

      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-black">{{ copy.businessOverview }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.businessOverviewHint }}</p>
          </div>
          <BarChart3 class="h-6 w-6 text-slate-400" />
        </div>
        <div class="mt-6 grid gap-4 sm:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <Users class="h-4 w-4" />
              <span class="text-sm font-bold">{{ copy.candidatesTotal }}</span>
            </div>
            <p class="mt-4 text-3xl font-black">{{ data?.candidate_total ?? "-" }}</p>
          </div>
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <CreditCard class="h-4 w-4" />
              <span class="text-sm font-bold">{{ copy.todayRevenue }}</span>
            </div>
            <p class="mt-4 text-2xl font-black">{{ revenueText }}</p>
            <p class="mt-2 text-sm text-slate-500">{{ copy.paidOrders(totalPaidOrders) }}</p>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>
