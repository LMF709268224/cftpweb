<script setup lang="ts">
import { Activity, BarChart3, CreditCard, Loader2, Mail, RefreshCw, Search, Shield, UserCheck, UserMinus, Users } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
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
  stage_buckets: StageBucket[]
  today_revenue: RevenueItem[]
  generated_at: string
}

const loading = ref(false)
const data = ref<DashboardData | null>(null)
const keyword = ref("")
const roleFilter = ref("all")
const statusFilter = ref("all")

const userStats = computed<UserStats>(() => data.value?.user_stats || { total: 0, active: 0, inactive: 0, admins: 0, email_verified: 0 })
const totalPipelines = computed(() => data.value?.stage_buckets.reduce((sum, item) => sum + Number(item.count || 0), 0) || 0)
const totalPaidOrders = computed(() => data.value?.today_revenue.reduce((sum, item) => sum + Number(item.order_count || 0), 0) || 0)
const profileCompletion = computed(() => Math.max(0, Math.min(100, Number(data.value?.profile_completion_percent || 0))))
const revenueText = computed(() => {
  const items = data.value?.today_revenue || []
  if (!items.length) return "暂无收款"
  return items.map((item) => `${item.currency} ${(Number(item.amount_minor || 0) / 100).toFixed(2)}`).join(" / ")
})
const roleOptions = computed(() => [
  { value: "all", label: "全部角色" },
  ...(data.value?.user_role_stats || []).map((item) => ({ value: item.key, label: item.label })),
])
const filteredUsers = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  return (data.value?.users || []).filter((user) => {
    const text = [user.name, user.email, user.phone, user.location, user.role_label, ...(user.roles || [])].join(" ").toLowerCase()
    const matchesKeyword = !normalizedKeyword || text.includes(normalizedKeyword)
    const roleNeedle = roleFilter.value.replace(/s$/, "")
    const matchesRole = roleFilter.value === "all" || user.role_label.toLowerCase().includes(roleNeedle) || user.roles.some((role) => role.toLowerCase().includes(roleNeedle))
    const matchesStatus = statusFilter.value === "all" || user.status.toLowerCase() === statusFilter.value
    return matchesKeyword && matchesRole && matchesStatus
  })
})

const summaryCards = computed(() => [
  { label: "Total Users", value: userStats.value.total, tone: "text-slate-950", icon: Users },
  { label: "Active Users", value: userStats.value.active, tone: "text-emerald-600", icon: UserCheck },
  { label: "Inactive Users", value: userStats.value.inactive, tone: "text-red-600", icon: UserMinus },
  { label: "Admins", value: userStats.value.admins, tone: "text-blue-600", icon: Shield },
])

function stageLabel(stageId: string) {
  return stageId === "未进入阶段" ? stageId : stageId
}

function roleBadgeClass(role: string) {
  const normalized = role.toLowerCase()
  if (normalized.includes("admin")) return "bg-blue-100 text-blue-700"
  if (normalized.includes("student")) return "bg-emerald-100 text-emerald-700"
  return "bg-slate-100 text-slate-600"
}

function userStatusClass(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === "active") return "bg-blue-100 text-blue-700"
  if (normalized === "inactive") return "bg-amber-100 text-amber-700"
  return "bg-slate-100 text-slate-600"
}

function formatDate(raw: string) {
  if (!raw) return "-"
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) return raw.slice(0, 10)
  return parsed.toLocaleDateString("zh-CN")
}

async function loadDashboard() {
  loading.value = true
  try {
    data.value = await apiClient<DashboardData>("/api/dashboard/ops")
  } catch (err) {
    console.error(err)
    toast.error("运营看板加载失败")
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <main class="mx-auto max-w-[1600px] px-6 py-8">
    <header class="mb-7 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="mb-2 text-sm font-semibold text-slate-500">Admin Dashboard</div>
        <h1 class="text-4xl font-black tracking-tight">运营看板</h1>
        <p class="mt-2 text-slate-500">查看用户概览、角色分布、阶段状态和今日收款。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <RouterLink to="/audit/webhooks" class="inline-flex items-center gap-2 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm font-bold shadow-sm hover:border-slate-400">
          <Activity class="h-4 w-4" />
          Audit Logs
        </RouterLink>
        <RouterLink to="/mails" class="inline-flex items-center gap-2 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm font-bold shadow-sm hover:border-slate-400">
          <Mail class="h-4 w-4" />
          Email Activity
        </RouterLink>
        <button
          class="inline-flex items-center gap-2 rounded-2xl border border-slate-300 bg-white px-5 py-3 font-bold shadow-sm transition hover:border-slate-500 disabled:opacity-50"
          type="button"
          :disabled="loading"
          @click="loadDashboard"
        >
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          刷新
        </button>
      </div>
    </header>

    <section class="mb-7 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-2 flex items-center justify-between text-sm font-semibold text-slate-600">
        <span>Profile Completion</span>
        <span class="text-amber-600">{{ profileCompletion }}%</span>
      </div>
      <div class="h-2 overflow-hidden rounded-full bg-blue-100">
        <div class="h-full rounded-full bg-[#0b579b] transition-all" :style="{ width: `${profileCompletion}%` }" />
      </div>
    </section>

    <section class="mb-5 grid gap-5 md:grid-cols-2 xl:grid-cols-4">
      <article v-for="card in summaryCards" :key="card.label" class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <p class="text-sm font-bold text-slate-600">{{ card.label }}</p>
          <component :is="card.icon" class="h-5 w-5 text-slate-400" />
        </div>
        <p class="mt-7 text-4xl font-black" :class="card.tone">{{ card.value }}</p>
      </article>
    </section>

    <section class="mb-7 grid gap-4 md:grid-cols-3 2xl:grid-cols-6">
      <article v-for="role in data?.user_role_stats || []" :key="role.key" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="flex items-center justify-between">
          <p class="text-sm font-bold text-slate-600">{{ role.label }}</p>
          <Users class="h-4 w-4 text-slate-400" />
        </div>
        <p class="mt-5 text-3xl font-black text-[#0b579b]">{{ role.count }}</p>
      </article>
    </section>

    <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="border-b border-slate-100 p-6">
        <h2 class="text-xl font-black">User Management</h2>
        <p class="mt-1 text-sm text-slate-500">来自 Casdoor 与 gmid 的用户摘要；这里先展示接口已能直接获取到的最近用户。</p>
        <div class="mt-5 flex flex-wrap gap-3">
          <div class="relative min-w-[280px] flex-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input v-model="keyword" class="h-11 w-full rounded-xl border border-slate-200 pl-10 pr-4 text-sm outline-none focus:border-[#0b579b]" placeholder="Search users by name or email..." />
          </div>
          <select v-model="roleFilter" class="h-11 rounded-xl border border-slate-200 px-4 text-sm outline-none focus:border-[#0b579b]">
            <option v-for="option in roleOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
          <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 px-4 text-sm outline-none focus:border-[#0b579b]">
            <option value="all">全部状态</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
            <option value="deleted">Deleted</option>
          </select>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-6 py-4">User</th>
              <th class="px-6 py-4">Email</th>
              <th class="px-6 py-4">Location</th>
              <th class="px-6 py-4">Role</th>
              <th class="px-6 py-4">Status</th>
              <th class="px-6 py-4">Email Verified</th>
              <th class="px-6 py-4">Created</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="loading">
              <td colspan="7" class="px-6 py-10 text-center text-slate-400">
                <Loader2 class="mr-2 inline h-5 w-5 animate-spin" />
                加载中
              </td>
            </tr>
            <tr v-else-if="filteredUsers.length === 0">
              <td colspan="7" class="px-6 py-10 text-center text-slate-400">暂无用户</td>
            </tr>
            <template v-else>
              <tr v-for="user in filteredUsers" :key="user.id" class="hover:bg-slate-50">
                <td class="px-6 py-4">
                  <div class="font-bold text-slate-950">{{ user.name || "-" }}</div>
                  <div class="text-xs text-slate-500">{{ user.phone || user.candidate_ulid || "-" }}</div>
                </td>
                <td class="px-6 py-4 text-slate-700">{{ user.email || "-" }}</td>
                <td class="px-6 py-4 text-slate-700">{{ user.location || "-" }}</td>
                <td class="px-6 py-4">
                  <span class="rounded-full px-2.5 py-1 text-xs font-bold" :class="roleBadgeClass(user.role_label)">{{ user.role_label || "-" }}</span>
                </td>
                <td class="px-6 py-4">
                  <span class="rounded-full px-2.5 py-1 text-xs font-bold" :class="userStatusClass(user.status)">{{ user.status || "-" }}</span>
                </td>
                <td class="px-6 py-4">
                  <span :class="['rounded-full px-2.5 py-1 text-xs font-bold', user.email_verified ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700']">
                    {{ user.email_verified ? "Verified" : "Unverified" }}
                  </span>
                </td>
                <td class="px-6 py-4 text-slate-700">{{ formatDate(user.created_at) }}</td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </section>

    <section class="mt-7 grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
      <article class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 p-6">
          <div>
            <h2 class="text-xl font-black">阶段分布</h2>
            <p class="mt-1 text-sm text-slate-500">按当前阶段和管线状态统计。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{{ totalPipelines }} pipelines</span>
        </div>
        <div v-if="loading" class="flex h-48 items-center justify-center text-slate-400">
          <Loader2 class="mr-2 h-5 w-5 animate-spin" />
          加载中
        </div>
        <div v-else-if="!data?.stage_buckets.length" class="flex h-48 items-center justify-center text-slate-400">
          暂无阶段数据
        </div>
        <div v-else class="divide-y divide-slate-100">
          <div v-for="item in data.stage_buckets" :key="`${item.stage_id}-${item.status}`" class="flex items-center justify-between gap-4 p-5">
            <div class="min-w-0">
              <p class="break-all text-base font-black">{{ stageLabel(item.stage_id) }}</p>
              <span class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-bold" :class="badgeClass(item.status)">
                {{ item.status || "未知状态" }}
              </span>
            </div>
            <div class="text-right">
              <p class="text-3xl font-black">{{ item.count }}</p>
              <p class="text-xs text-slate-500">人/管线</p>
            </div>
          </div>
        </div>
      </article>

      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-black">业务概览</h2>
            <p class="mt-1 text-sm text-slate-500">我们系统特有的运营指标。</p>
          </div>
          <BarChart3 class="h-6 w-6 text-slate-400" />
        </div>
        <div class="mt-6 grid gap-4 sm:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <Users class="h-4 w-4" />
              <span class="text-sm font-bold">考生总数</span>
            </div>
            <p class="mt-4 text-3xl font-black">{{ data?.candidate_total ?? "-" }}</p>
          </div>
          <div class="rounded-2xl bg-slate-50 p-5">
            <div class="flex items-center gap-2 text-slate-500">
              <CreditCard class="h-4 w-4" />
              <span class="text-sm font-bold">今日收款金额</span>
            </div>
            <p class="mt-4 text-2xl font-black">{{ revenueText }}</p>
            <p class="mt-2 text-sm text-slate-500">已支付订单 {{ totalPaidOrders }} 笔</p>
          </div>
        </div>
        <div class="mt-6 rounded-2xl border border-slate-100 bg-slate-50 p-5 text-sm leading-7 text-slate-600">
          <p>用户统计：来自 Casdoor 用户列表。</p>
          <p>考生总数：来自 Casdoor 用户列表，并通过 gmid 映射到平台 ULID 后统计。</p>
          <p>阶段分布：来自 gprog 管线实例列表，按当前阶段和状态聚合。</p>
          <p>今日收款：来自 gmall 订单列表，只统计今天创建且支付状态为 PAID 的订单。</p>
        </div>
      </article>
    </section>
  </main>
</template>
