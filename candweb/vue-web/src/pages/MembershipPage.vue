<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { AlertCircle, Check, Crown, Loader2, Percent, RefreshCw, Shield, Star, XCircle } from "lucide-vue-next"
import { toast } from "vue-sonner"
import AppShell from "@/components/AppShell.vue"
import AppPagination from "@/components/AppPagination.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type RecordData = Record<string, any>

const { t, lang } = useTranslation()
const activeTab = ref("overview")
const loading = ref(false)
const cancelling = ref(false)
const activeMembership = ref<RecordData | null>(null)
const plans = ref<RecordData[]>([])
const history = ref<RecordData[]>([])
const billings = ref<RecordData[]>([])
const historyPage = ref(1)
const historyPageSize = ref(10)
const historyTotal = ref(0)
const historyTotalPages = ref(0)
const billingPage = ref(1)
const billingPageSize = ref(10)
const billingTotal = ref(0)
const billingTotalPages = ref(0)
const pageSizeOptions = [10, 30, 50, 100]

const tabs = computed(() => [
  { id: "overview", label: lang.value === "zh" ? "当前会员" : "Current" },
  { id: "levels", label: t.value.membership.tabs.levels },
  { id: "history", label: lang.value === "zh" ? "会员历史" : "History" },
  { id: "billings", label: lang.value === "zh" ? "账单记录" : "Billings" },
])

const currentRecord = computed(() => {
  const data = activeMembership.value || {}
  const list = listFrom(data, ["user_memberships", "memberships", "records", "items"])
  return data.membership || data.user_membership || data.record || data.active_membership || list[0] || history.value[0] || data
})

const currentPlan = computed(() => {
  const data = activeMembership.value || {}
  const direct = data.plan || data.membership_config || data.membership_detail || null
  if (direct) return direct
  const membershipUlid = currentRecord.value?.membership_ulid
  const membershipGpath = currentRecord.value?.membership_gpath
  return plans.value.find((plan) => plan.membership_ulid === membershipUlid || plan.membership_gpath === membershipGpath) || null
})

const hasActiveMembership = computed(() => {
  const record = currentRecord.value
  if (!record || Object.keys(record).length === 0) return false
  const status = String(record.status || "").toUpperCase()
  return status === "ACTIVE" || status === "CURRENT" || status === "GRACE" || Boolean(record.membership_record_ulid)
})

const currentMembershipName = computed(() => {
  return currentPlan.value?.name || currentRecord.value?.name || currentRecord.value?.membership_name || currentRecord.value?.membership_ulid || "-"
})

function listFrom(data: any, keys: string[]) {
  for (const key of keys) {
    if (Array.isArray(data?.[key])) return data[key]
  }
  return []
}

function isActiveStatus(status: unknown) {
  const value = String(status || "").toUpperCase()
  return value === "ACTIVE" || value === "CURRENT" || value === "GRACE"
}

function activeRecordFromPayload(data: any) {
  const list = listFrom(data, ["user_memberships", "memberships", "records", "items"])
  return data?.membership || data?.user_membership || data?.record || data?.active_membership || list[0] || null
}

function formatDate(value: unknown) {
  const raw = String(value || "")
  if (!raw) return "-"
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString(lang.value === "zh" ? "zh-CN" : "en-US", { hour12: false })
}

function formatMoney(amount: unknown, currency = "USD") {
  const value = Number(amount || 0)
  if (Number.isNaN(value)) return "-"
  return `${currency} ${(value / 100).toFixed(2)}`
}

function formatSource(source: unknown, langCode: string) {
  const s = String(source || "").toLowerCase()
  if (s === "bundle_purchase") return langCode === "zh" ? "套餐购买" : "Bundle Purchase"
  if (s === "admin_grant") return langCode === "zh" ? "管理员发卡" : "Admin Grant"
  if (s === "renewal") return langCode === "zh" ? "会员续费" : "Renewal"
  return String(source || "-")
}

function statusLabel(status: unknown) {
  const value = String(status || "").toUpperCase()
  if (!value) return "-"
  const zh: Record<string, string> = {
    ACTIVE: "有效",
    CURRENT: "有效",
    GRACE: "宽限期",
    CANCELLED: "已取消",
    EXPIRED: "已过期",
    PENDING: "待处理",
    PAID: "已支付",
    FAILED: "失败",
  }
  return lang.value === "zh" ? zh[value] || value : value
}

function badgeClass(status: unknown) {
  const value = String(status || "").toUpperCase()
  if (["ACTIVE", "CURRENT", "PAID", "SUCCESS"].includes(value)) return "border-emerald-200 bg-emerald-50 text-emerald-700"
  if (["GRACE", "PENDING", "PROCESSING"].includes(value)) return "border-amber-200 bg-amber-50 text-amber-700"
  if (["CANCELLED", "EXPIRED", "FAILED"].includes(value)) return "border-red-200 bg-red-50 text-red-700"
  return "border-slate-200 bg-slate-50 text-slate-600"
}

function parseFeatures(plan: RecordData) {
  const raw = String(plan.features_json || "").trim()
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    const extractText = (item: any): string => {
      if (typeof item === "string") return item
      if (!item) return ""
      if (typeof item === "object") return String(item.title || item.name || item.text || item.label || item.desc || item.description || JSON.stringify(item))
      return String(item)
    }
    
    let arr: any[] = []
    if (Array.isArray(parsed)) arr = parsed
    else if (parsed && Array.isArray(parsed.features)) arr = parsed.features
    else if (typeof parsed === "object") return Object.entries(parsed).map(([key, value]) => `${key}: ${extractText(value)}`)

    let flatItems: any[] = []
    arr.forEach((p) => {
      if (p && typeof p === "object" && Array.isArray(p.items)) {
        flatItems.push(...p.items)
      } else {
        flatItems.push(p)
      }
    })
    return flatItems.map(extractText).filter(Boolean)
  } catch {
    return raw.split(/\r?\n|[,;；，]/).map((item) => item.trim()).filter(Boolean)
  }
}

function totalFrom(data: any, list: RecordData[]) {
  return Number(data?.total ?? data?.total_count ?? data?.total_items ?? list.length ?? 0) || 0
}

function totalPagesFrom(data: any, total: number, pageSize: number) {
  return Number(data?.total_pages || Math.ceil(total / pageSize) || 0)
}

async function loadMembershipHistory() {
  const historyData = await apiClient(`/api/membership/history?page=${historyPage.value}&page_size=${historyPageSize.value}`)
  const nextHistory = listFrom(historyData, ["user_memberships", "memberships", "records", "items", "history"])
  history.value = nextHistory
  historyTotal.value = totalFrom(historyData, nextHistory)
  historyTotalPages.value = totalPagesFrom(historyData, historyTotal.value, historyPageSize.value)
  return nextHistory
}

async function loadMembershipBillings() {
  const billingData = await apiClient(`/api/membership/billings?page=${billingPage.value}&page_size=${billingPageSize.value}`)
  const nextBillings = listFrom(billingData, ["billings", "records", "items"])
  billings.value = nextBillings
  billingTotal.value = totalFrom(billingData, nextBillings)
  billingTotalPages.value = totalPagesFrom(billingData, billingTotal.value, billingPageSize.value)
  return nextBillings
}

async function loadMembership() {
  loading.value = true
  try {
    const [planData, nextHistory] = await Promise.all([
      apiClient("/api/membership/plans?page=1&page_size=50"),
      loadMembershipHistory(),
      loadMembershipBillings(),
    ])
    plans.value = listFrom(planData, ["memberships", "plans", "items"])
    activeMembership.value = await loadActiveMembershipFromHistory(nextHistory) || { user_memberships: nextHistory }
  } catch (err) {
    console.error(err)
    toast.error(lang.value === "zh" ? "会员信息加载失败" : "Failed to load membership")
  } finally {
    loading.value = false
  }
}

async function loadActiveMembershipFromHistory(membershipHistory: RecordData[]) {
  const activeRecord = membershipHistory.find((item) => isActiveStatus(item.status))
  const fallbackPlan = plans.value.find((plan) => plan.membership_ulid === activeRecord?.membership_ulid)
  const membershipGpath = String(activeRecord?.membership_gpath || fallbackPlan?.membership_gpath || "").trim()
  if (!membershipGpath) return null

  try {
    const activeData = await apiClient(`/api/membership/active?membership_gpath=${encodeURIComponent(membershipGpath)}`, {
      suppressErrorToast: true,
    })
    const confirmedRecord = activeRecordFromPayload(activeData)
    const matchedPlan = plans.value.find((plan) => {
      return plan.membership_ulid === confirmedRecord?.membership_ulid || plan.membership_gpath === membershipGpath
    })
    return {
      ...(activeData || {}),
      membership_config: matchedPlan || null,
    }
  } catch {
    return { user_memberships: [activeRecord] }
  }
}

async function cancelMembership() {
  const recordUlid = currentRecord.value?.membership_record_ulid
  if (!recordUlid) return
  const ok = window.confirm(lang.value === "zh" ? "确认取消当前会员吗？" : "Cancel current membership?")
  if (!ok) return
  cancelling.value = true
  try {
    await apiClient("/api/membership/cancel", {
      method: "POST",
      body: JSON.stringify({ membership_record_ulid: recordUlid, reason: "user_requested" }),
    })
    toast.success(lang.value === "zh" ? "已提交取消会员请求" : "Membership cancellation submitted")
    await loadMembership()
  } finally {
    cancelling.value = false
  }
}

function handleHistoryPaginationChange() {
  if (loading.value) return
  void loadMembershipHistory()
}

function handleBillingPaginationChange() {
  if (loading.value) return
  void loadMembershipBillings()
}

onMounted(() => {
  void loadMembership()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center justify-between border-b border-border bg-white px-5">
        <div class="flex items-center gap-3">
          <Crown class="h-4 w-4 text-slate-700" />
          <span class="text-sm font-medium text-foreground">{{ t.membership.title }}</span>
        </div>
        <button class="membership-refresh-btn inline-flex h-9 items-center gap-2 rounded-xl border px-4 text-sm font-semibold" type="button" @click="loadMembership">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ lang === "zh" ? "刷新" : "Refresh" }}
        </button>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6 flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.membership.title }}</h1>
            <p class="mt-2 text-muted-foreground">{{ t.membership.subtitle }}</p>
          </div>
          <span v-if="hasActiveMembership" class="rounded-full border px-4 py-2 text-sm font-black" :class="badgeClass(currentRecord.status)">
            {{ statusLabel(currentRecord.status) }}
          </span>
        </div>

        <div v-if="loading" class="rounded-[16px] bg-white p-12 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <Loader2 class="mx-auto mb-3 h-7 w-7 animate-spin text-primary" />
          {{ lang === "zh" ? "正在加载会员信息..." : "Loading membership..." }}
        </div>

        <template v-else>
          <section class="mb-5 overflow-hidden rounded-[18px] border border-slate-200 bg-white shadow-[0_10px_28px_rgba(15,74,82,0.06)]">
            <div class="relative bg-gradient-to-br from-[#0b4ea2] via-[#1976c9] to-[#12b886] p-6 text-white">
              <div class="absolute right-6 top-6 opacity-20">
                <Crown class="h-24 w-24" />
              </div>
              <div class="relative">
                <p class="text-sm font-semibold uppercase tracking-[0.24em] text-white/70">{{ lang === "zh" ? "当前会员" : "Current membership" }}</p>
                <h2 class="mt-3 text-3xl font-black">{{ hasActiveMembership ? currentMembershipName : (lang === "zh" ? "暂无有效会员" : "No active membership") }}</h2>
                <p class="mt-2 max-w-2xl text-sm text-white/80">
                  {{ hasActiveMembership ? (currentPlan?.description || currentRecord?.description || (lang === "zh" ? "你的会员权益当前可用。" : "Your membership benefits are available.")) : (lang === "zh" ? "你还没有有效会员，可以查看下方会员等级。" : "You do not have an active membership yet. Review available plans below.") }}
                </p>
              </div>
            </div>
            <div class="grid gap-3 p-5 md:grid-cols-4">
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ lang === "zh" ? "开始时间" : "Started" }}</div>
                <div class="mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.started_at) }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ lang === "zh" ? "过期时间" : "Expires" }}</div>
                <div class="mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.expires_at) }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ lang === "zh" ? "下次扣费" : "Next billing" }}</div>
                <div class="mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.next_billing_at) }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ lang === "zh" ? "自动续费" : "Auto renew" }}</div>
                <div class="mt-2 text-sm font-black text-slate-900">{{ currentRecord.auto_renew ? (lang === "zh" ? "已开启" : "Enabled") : "-" }}</div>
              </div>
            </div>
          </section>

          <div class="mb-4 rounded-[14px] bg-white px-5 pt-4 shadow-[0_10px_24px_rgba(15,74,82,0.04)] md:px-6">
            <div class="flex flex-wrap gap-x-8 gap-y-2 border-b border-[#edf0f2]">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                :class="['relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === tab.id ? 'text-primary' : 'text-[#111827] hover:text-primary']"
                @click="activeTab = tab.id"
              >
                {{ tab.label }}
                <span v-if="activeTab === tab.id" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
              </button>
            </div>
          </div>

          <section v-if="activeTab === 'overview'" class="grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
            <div class="rounded-[16px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
              <h2 class="mb-4 text-lg font-semibold text-card-foreground">{{ lang === "zh" ? "会员权益" : "Benefits" }}</h2>
              <div v-if="currentPlan && parseFeatures(currentPlan).length" class="grid gap-3 sm:grid-cols-2">
                <div v-for="feature in parseFeatures(currentPlan)" :key="feature" class="flex gap-3 rounded-xl border border-emerald-100 bg-emerald-50/70 p-4">
                  <Check class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
                  <span class="text-sm font-medium text-slate-700">{{ feature }}</span>
                </div>
              </div>
              <div v-else class="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
                <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
                {{ lang === "zh" ? "当前会员暂无可展示的权益配置。" : "No benefit details are configured for the current membership." }}
              </div>
            </div>

            <div class="rounded-[16px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
              <h2 class="mb-4 text-lg font-semibold text-card-foreground">{{ lang === "zh" ? "会员操作" : "Membership actions" }}</h2>
              <div class="space-y-3 text-sm text-slate-600">
                <div class="flex justify-between"><span>{{ lang === "zh" ? "会员记录" : "Record" }}</span><span class="font-mono text-xs">{{ currentRecord.membership_record_ulid || "-" }}</span></div>
                <div class="flex justify-between"><span>{{ lang === "zh" ? "来源" : "Source" }}</span><span>{{ formatSource(currentRecord.source, lang) }}</span></div>
                <div class="flex justify-between"><span>{{ lang === "zh" ? "续费次数" : "Renewals" }}</span><span>{{ currentRecord.renewal_count ?? "-" }}</span></div>
                <div class="flex justify-between"><span>{{ lang === "zh" ? "最近支付" : "Last payment" }}</span><span>{{ formatMoney(currentRecord.last_payment_amount_minor, "USD") }}</span></div>
              </div>
              <button
                v-if="hasActiveMembership && currentRecord.membership_record_ulid"
                class="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-xl border border-red-200 px-5 py-3 font-bold text-red-600 hover:bg-red-50 disabled:opacity-50"
                :disabled="cancelling"
                type="button"
                @click="cancelMembership"
              >
                <Loader2 v-if="cancelling" class="h-4 w-4 animate-spin" />
                <XCircle v-else class="h-4 w-4" />
                {{ lang === "zh" ? "取消会员" : "Cancel membership" }}
              </button>
            </div>
          </section>

          <section v-if="activeTab === 'levels'" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <div v-for="plan in plans" :key="plan.membership_ulid || plan.membership_gpath" class="relative overflow-hidden rounded-[18px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:shadow-md">
              <div class="absolute left-0 top-0 h-full w-1" :class="Number(plan.tier_level || 0) >= 3 ? 'bg-amber-500' : Number(plan.tier_level || 0) >= 2 ? 'bg-primary' : 'bg-slate-300'" />
              <div class="mb-4 flex items-start justify-between gap-3">
                <div>
                  <h3 class="text-lg font-semibold text-card-foreground">{{ plan.name || "-" }}</h3>
                  <p class="mt-1 text-sm text-muted-foreground">{{ plan.description || plan.ideal_for || "-" }}</p>
                </div>
                <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <Star v-if="Number(plan.tier_level || 0) <= 1" class="h-6 w-6" />
                  <Crown v-else class="h-6 w-6" />
                </div>
              </div>
              <div class="mb-4 grid grid-cols-2 gap-3 text-sm">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs text-slate-500">{{ lang === "zh" ? "等级" : "Tier" }}</div>
                  <div class="font-black">{{ plan.tier_level || "-" }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs text-slate-500">{{ lang === "zh" ? "时长" : "Duration" }}</div>
                  <div class="font-black">{{ plan.duration_in_months || "-" }} {{ lang === "zh" ? "个月" : "months" }}</div>
                </div>
              </div>
              <div v-if="plan.course_discount_coupon" class="mb-4 flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm font-bold text-amber-700">
                <Percent class="h-4 w-4" />
                <span>{{ lang === "zh" ? "专属课程折扣码：" : "Course Discount Code: " }}{{ plan.course_discount_coupon }}</span>
              </div>
              <ul class="space-y-2">
                <li v-for="feature in parseFeatures(plan)" :key="feature" class="flex items-center gap-2 text-sm">
                  <Check class="h-4 w-4 shrink-0 text-emerald-500" />
                  <span class="text-card-foreground">{{ feature }}</span>
                </li>
              </ul>
            </div>
            <div v-if="!plans.length" class="rounded-[16px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)] md:col-span-2 xl:col-span-3">
              {{ lang === "zh" ? "暂无可展示的会员等级。" : "No membership plans are available." }}
            </div>
          </section>

          <section v-if="activeTab === 'history'" class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div v-for="item in history" :key="item.membership_record_ulid || item.membership_order_ulid" class="grid gap-3 border-b border-slate-100 p-5 last:border-b-0 md:grid-cols-[1fr_auto]">
              <div>
                <div class="font-black text-slate-900">{{ item.membership_name || item.membership_ulid || "-" }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ formatDate(item.started_at) }} - {{ formatDate(item.expires_at) }}</div>
                <div class="mt-1 font-mono text-xs text-slate-400">{{ item.membership_record_ulid }}</div>
              </div>
              <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(item.status)">{{ statusLabel(item.status) }}</span>
            </div>
            <div v-if="!history.length" class="p-8 text-center text-muted-foreground">{{ lang === "zh" ? "暂无会员历史。" : "No membership history." }}</div>
            <AppPagination
              v-if="historyTotal > 0"
              v-model:page="historyPage"
              v-model:page-size="historyPageSize"
              :total="historyTotal"
              :total-pages="historyTotalPages"
              :page-size-options="pageSizeOptions"
              :disabled="loading"
              :locale="lang"
              @page-change="handleHistoryPaginationChange"
            />
          </section>

          <section v-if="activeTab === 'billings'" class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div v-for="item in billings" :key="item.billing_record_ulid || item.gpay_order_ulid" class="grid gap-3 border-b border-slate-100 p-5 last:border-b-0 md:grid-cols-[1fr_auto]">
              <div>
                <div class="font-black text-slate-900">{{ item.billing_type || item.stripe_invoice_id || "-" }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ formatMoney(item.amount_minor, item.currency || "USD") }} · {{ formatDate(item.period_start) }} - {{ formatDate(item.period_end) }}</div>
                <div class="mt-1 font-mono text-xs text-slate-400">{{ item.gpay_order_ulid || item.stripe_invoice_id || "-" }}</div>
              </div>
              <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(item.status)">{{ statusLabel(item.status) }}</span>
            </div>
            <div v-if="!billings.length" class="p-8 text-center text-muted-foreground">{{ lang === "zh" ? "暂无账单记录。" : "No billing records." }}</div>
            <AppPagination
              v-if="billingTotal > 0"
              v-model:page="billingPage"
              v-model:page-size="billingPageSize"
              :total="billingTotal"
              :total-pages="billingTotalPages"
              :page-size-options="pageSizeOptions"
              :disabled="loading"
              :locale="lang"
              @page-change="handleBillingPaginationChange"
            />
          </section>
        </template>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.membership-refresh-btn {
  border-color: #e2e8f0;
  background: #ffffff;
  color: #334155;
  box-shadow: 0 8px 18px -16px rgba(15, 23, 42, 0.35);
  transition: transform 0.2s ease, border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.membership-refresh-btn:hover {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  box-shadow: 0 14px 28px -18px rgba(37, 99, 235, 0.42);
  transform: scale(1.02);
}

.membership-refresh-btn:active {
  transform: scale(0.98);
}

.membership-refresh-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.16), 0 14px 28px -18px rgba(37, 99, 235, 0.42);
}
</style>
