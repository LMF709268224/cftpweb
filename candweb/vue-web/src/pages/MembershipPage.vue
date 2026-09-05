<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { AlertCircle, ArrowUpCircle, Check, ChevronDown, Crown, Loader2, PanelLeft, Percent, RefreshCw, ShoppingBag, Star, X, XCircle } from "lucide-vue-next"
import { toast } from "vue-sonner"
import AppShell from "@/components/AppShell.vue"
import AppPagination from "@/components/AppPagination.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useDialogAccessibility } from "@/lib/dialogAccessibility"
import { formatMinorAmount } from "@/lib/display"
import { useTranslation } from "@/lib/language"
import { loadStripeFactory } from "@/lib/stripe"
import { formatBackendDate } from "@/lib/utils"

type RecordData = Record<string, any>

const { t, lang } = useTranslation()
const route = useRoute()
const activeTab = ref("overview")
const loading = ref(false)
const loadError = ref(false)
const cancelling = ref(false)
const cancelRenewConfirmOpen = ref(false)
const cancelRenewConfirmDialogRef = ref<HTMLElement | null>(null)
const upgradeDialogOpen = ref(false)
const upgradeDialogRef = ref<HTMLElement | null>(null)
const upgradePreviewLoading = ref(false)
const upgradePreviewError = ref(false)
const upgrading = ref(false)
const upgradeTargetPlan = ref<RecordData | null>(null)
const upgradePreview = ref<RecordData | null>(null)
const upgradeIdempotencyKey = ref("")
useBodyScrollLock(() => cancelRenewConfirmOpen.value || upgradeDialogOpen.value)
const activeMembership = ref<RecordData | null>(null)
const plans = ref<RecordData[]>([])
const history = ref<RecordData[]>([])
const billings = ref<RecordData[]>([])
const historyPage = ref(1)
const historyPageSize = ref(10)
const historyTotal = ref(0)
const historyTotalPages = ref(0)
const lastHistoryPageSize = ref(historyPageSize.value)
const historyPrevCursor = ref("")
const lastHistoryPage = ref(1)
const historyNextCursor = ref("")
const historyHasMore = ref(false)
const billingPage = ref(1)
const billingPageSize = ref(10)
const billingTotal = ref(0)
const billingTotalPages = ref(0)
const lastBillingPageSize = ref(billingPageSize.value)
const billingPrevCursor = ref("")
const lastBillingPage = ref(1)
const billingNextCursor = ref("")
const billingHasMore = ref(false)
const pageSizeOptions = [10, 30, 50, 100]
let membershipRequestId = 0
let membershipPlansRequestId = 0
let membershipHistoryRequestId = 0
let membershipBillingsRequestId = 0
let upgradePollGeneration = 0
let requestedUpgradeTarget = ""
const upgradeOrderPollAttempts = 8
const upgradeOrderPollIntervalMs = 1500

const tabs = computed(() => [
  { id: "overview", label: t.value.membership.tabsOverview },
  { id: "levels", label: t.value.membership.tabs.levels },
  { id: "history", label: t.value.membership.tabsHistory },
  { id: "billings", label: t.value.membership.tabsBillings },
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
  return findMembershipPlan(currentRecord.value?.membership_ulid, currentRecord.value?.membership_gpath)
})

const hasActiveMembership = computed(() => {
  const record = currentRecord.value
  if (!record || Object.keys(record).length === 0) return false
  const status = String(record.status || "").toUpperCase()
  return status === "ACTIVE" || status === "CURRENT" || status === "GRACE"
})

const currentMembershipName = computed(() => {
  return String(currentPlan.value?.name || membershipDisplayName(currentRecord.value, t.value.membership.membershipRecord))
})

const isAutoRenewCancelled = computed(() => {
  const record = currentRecord.value || {}
  return Boolean(record.cancelled_at || record.canceled_at || record.cancel_requested_at || record.renewal_cancelled_at)
})

const canCancelMembership = computed(() => {
  return Boolean(hasActiveMembership.value && currentRecord.value?.membership_record_ulid && currentRecord.value?.auto_renew && !isAutoRenewCancelled.value)
})

const autoRenewLabel = computed(() => {
  if (isAutoRenewCancelled.value) return t.value.membership.autoRenewCancelled
  if (currentRecord.value?.auto_renew) return t.value.membership.autoRenewEnabled
  return "-"
})

const cancelMembershipButtonLabel = computed(() => {
  if (isAutoRenewCancelled.value) return t.value.membership.autoRenewCancelled
  if (!currentRecord.value?.auto_renew) return t.value.membership.autoRenewDisabled
  return t.value.membership.cancelAutoRenew
})

const currentTierLevel = computed(() => Number(currentPlan.value?.tier_level || currentRecord.value?.tier_level || 0))

function isCurrentPlan(plan: RecordData) {
  const planID = String(plan?.membership_ulid || "").trim()
  const currentID = String(currentRecord.value?.membership_ulid || "").trim()
  return Boolean(hasActiveMembership.value && planID && currentID && planID === currentID)
}

function canUpgradeToPlan(plan: RecordData) {
  if (!hasActiveMembership.value || isCurrentPlan(plan)) return false
  const targetID = String(plan?.membership_ulid || "").trim()
  const targetTier = Number(plan?.tier_level || 0)
  return Boolean(
    targetID
    && targetTier > currentTierLevel.value,
  )
}

function listFrom(data: any, keys: string[]) {
  for (const key of keys) {
    if (Array.isArray(data?.[key])) return data[key]
  }
  return []
}

function findMembershipPlan(membershipUlid: unknown, membershipGpath: unknown) {
  const normalizedULID = String(membershipUlid || "").trim()
  if (normalizedULID) {
    const exactPlan = plans.value.find((plan) => String(plan?.membership_ulid || "").trim() === normalizedULID)
    if (exactPlan) return exactPlan
  }
  const normalizedGpath = String(membershipGpath || "").trim()
  return normalizedGpath
    ? plans.value.find((plan) => String(plan?.membership_gpath || "").trim() === normalizedGpath) || null
    : null
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
  return formatBackendDate(raw, lang.value)
}

function formatMoney(amount: unknown, currency = "USD") {
  const value = formatMinorAmount(amount || 0)
  return value === null ? "-" : `${String(currency || "USD").toUpperCase()} ${value}`
}

function formatSource(source: unknown) {
  const s = String(source || "").toLowerCase()
  if (s === "initial") return t.value.membership.sourceInitial
  if (s === "bundle_purchase") return t.value.membership.sourceBundlePurchase
  if (s === "admin_grant") return t.value.membership.sourceAdminGrant
  if (s === "renewal") return t.value.membership.sourceRenewal
  return String(source || "-")
}

function membershipPlanForRecord(record: RecordData) {
  return findMembershipPlan(record?.membership_ulid, record?.membership_gpath)
}

function membershipDisplayName(record: RecordData, fallback = "-") {
  const plan = membershipPlanForRecord(record)
  return String(record?.membership_name || record?.name || plan?.name || fallback)
}

function membershipRecordSummary(record: RecordData) {
  const source = formatSource(record?.source)
  const renewalCount = record?.renewal_count
  const parts = [
    source !== "-" ? source : "",
    renewalCount !== undefined && renewalCount !== null ? `${t.value.membership.renewalCount} ${renewalCount}` : "",
  ].filter(Boolean)
  return parts.join(" · ") || t.value.membership.membershipRecord
}

function billingTitle(item: RecordData) {
  const type = String(item?.billing_type || "").trim()
  if (type) return formatSource(type)
  if (item?.stripe_invoice_id) return t.value.membership.stripeInvoice
  return t.value.membership.membershipBilling
}

function statusLabel(status: unknown) {
  const value = String(status || "").toUpperCase()
  if (!value) return "-"
  const labels: Record<string, string> = {
    ACTIVE: t.value.membership.statusActive,
    CURRENT: t.value.membership.statusActive,
    GRACE: t.value.membership.statusGrace,
    CANCELLED: t.value.membership.statusCancelled,
    EXPIRED: t.value.membership.statusExpired,
    PENDING: t.value.membership.statusPending,
    PAID: t.value.membership.statusPaid,
    FAILED: t.value.membership.statusFailed,
  }
  return labels[value] || value
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

async function loadMembershipHistory(suppressErrorToast = false) {
  const requestId = ++membershipHistoryRequestId
  const requestedPage = historyPage.value
  const requestedLastPage = lastHistoryPage.value
  const requestedPageSize = historyPageSize.value
  const params = new URLSearchParams({ page_size: String(requestedPageSize) })
  
  let cursor = ""
  if (requestedPage > requestedLastPage) {
    cursor = historyNextCursor.value
  } else if (requestedPage < requestedLastPage) {
    cursor = historyPrevCursor.value
  }
  
  if (cursor) params.set("cursor", cursor)
  const historyData = await apiClient(`/api/membership/history?${params.toString()}`, { suppressErrorToast })
  if (requestId !== membershipHistoryRequestId) return null

  const nextHistory = listFrom(historyData, ["user_memberships", "memberships", "records", "items", "history"])
  history.value = nextHistory
  historyTotal.value = totalFrom(historyData, nextHistory)
  historyTotalPages.value = totalPagesFrom(historyData, historyTotal.value, requestedPageSize)
  historyHasMore.value = Boolean(historyData?.has_more)
  historyNextCursor.value = String(historyData?.next_cursor || "")
  historyPrevCursor.value = String(historyData?.prev_cursor || "")
  lastHistoryPage.value = requestedPage
  return nextHistory
}

async function loadMembershipBillings(suppressErrorToast = false) {
  const requestId = ++membershipBillingsRequestId
  const requestedPage = billingPage.value
  const requestedLastPage = lastBillingPage.value
  const requestedPageSize = billingPageSize.value
  const params = new URLSearchParams({ page_size: String(requestedPageSize) })
  
  let cursor = ""
  if (requestedPage > requestedLastPage) {
    cursor = billingNextCursor.value
  } else if (requestedPage < requestedLastPage) {
    cursor = billingPrevCursor.value
  }
  
  if (cursor) params.set("cursor", cursor)
  const billingData = await apiClient(`/api/membership/billings?${params.toString()}`, { suppressErrorToast })
  if (requestId !== membershipBillingsRequestId) return null

  const nextBillings = listFrom(billingData, ["billings", "records", "items"])
  billings.value = nextBillings
  billingTotal.value = totalFrom(billingData, nextBillings)
  billingTotalPages.value = totalPagesFrom(billingData, billingTotal.value, requestedPageSize)
  billingHasMore.value = Boolean(billingData?.has_more)
  billingNextCursor.value = String(billingData?.next_cursor || "")
  billingPrevCursor.value = String(billingData?.prev_cursor || "")
  lastBillingPage.value = requestedPage
  return nextBillings
}

async function loadMembershipPlans(suppressErrorToast = false) {
  const requestId = ++membershipPlansRequestId
  const planData = await apiClient("/api/membership/plans?page=1&page_size=50", { suppressErrorToast })
  if (requestId !== membershipPlansRequestId) return null

  const nextPlans = listFrom(planData, ["memberships", "plans", "items"])
  plans.value = nextPlans

  if (activeMembership.value) {
    const record = currentRecord.value
    const matchedPlan = findMembershipPlan(record?.membership_ulid, record?.membership_gpath)
    if (matchedPlan) {
      activeMembership.value = {
        ...activeMembership.value,
        membership_config: matchedPlan,
      }
    }
  }
  return nextPlans
}

async function loadMembership() {
  const requestId = ++membershipRequestId
  loading.value = true
  loadError.value = false
  try {
    const [, nextHistory] = await Promise.all([
      loadMembershipPlans(true),
      loadMembershipHistory(true),
      loadMembershipBillings(true),
    ])
    if (requestId !== membershipRequestId || nextHistory === null) return

    const nextActiveMembership = await loadActiveMembershipFromHistory(nextHistory)
    if (requestId !== membershipRequestId) return
    activeMembership.value = nextActiveMembership || { user_memberships: nextHistory }
  } catch (err) {
    if (requestId !== membershipRequestId) return
    console.error(err)
    loadError.value = true
  } finally {
    if (requestId === membershipRequestId) loading.value = false
  }
}

async function loadActiveMembershipFromHistory(membershipHistory: RecordData[]) {
  const activeRecord = membershipHistory.find((item) => isActiveStatus(item.status))

  try {
    const activeData = await apiClient("/api/membership/active", {
      suppressErrorToast: true,
    })
    const confirmedRecord = activeRecordFromPayload(activeData)
    const membershipGpath = String(confirmedRecord?.membership_gpath || "").trim()
    const matchedPlan = findMembershipPlan(confirmedRecord?.membership_ulid, membershipGpath)
    return {
      ...(activeData || {}),
      membership_config: matchedPlan || null,
    }
  } catch {
    return activeRecord ? { user_memberships: [activeRecord] } : null
  }
}

function openCancelRenewConfirm() {
  if (cancelling.value || !canCancelMembership.value) return
  cancelRenewConfirmOpen.value = true
}

function closeCancelRenewConfirm() {
  cancelRenewConfirmOpen.value = false
}

function confirmCancelMembership() {
  closeCancelRenewConfirm()
  void cancelMembership()
}

useDialogAccessibility(() => cancelRenewConfirmOpen.value, cancelRenewConfirmDialogRef, closeCancelRenewConfirm)

function closeUpgradeDialog() {
  if (upgrading.value) return
  upgradeDialogOpen.value = false
  upgradePreviewLoading.value = false
  upgradePreviewError.value = false
  upgradeTargetPlan.value = null
  upgradePreview.value = null
  upgradeIdempotencyKey.value = ""
}

useDialogAccessibility(() => upgradeDialogOpen.value, upgradeDialogRef, closeUpgradeDialog)

async function openUpgradePreview(plan: RecordData) {
  const targetMembershipULID = String(plan?.membership_ulid || "").trim()
  if (!targetMembershipULID || !canUpgradeToPlan(plan) || upgradePreviewLoading.value) return

  upgradeTargetPlan.value = plan
  upgradePreview.value = null
  upgradePreviewError.value = false
  upgradeIdempotencyKey.value = crypto.randomUUID()
  upgradeDialogOpen.value = true
  upgradePreviewLoading.value = true

  try {
    upgradePreview.value = await apiClient("/api/membership/upgrade/preview", {
      method: "POST",
      suppressErrorToast: true,
      body: JSON.stringify({ target_membership_ulid: targetMembershipULID }),
    })
  } catch (error) {
    console.error("Failed to preview membership upgrade", error)
    upgradePreviewError.value = true
  } finally {
    upgradePreviewLoading.value = false
  }
}

async function confirmMembershipUpgrade() {
  const targetMembershipULID = String(upgradeTargetPlan.value?.membership_ulid || "").trim()
  if (!targetMembershipULID || !upgradePreview.value?.eligible || upgrading.value) return

  upgrading.value = true
  try {
    const body: RecordData = {
      target_membership_ulid: targetMembershipULID,
      idempotency_key: upgradeIdempotencyKey.value,
    }
    const prorationDate = Number(upgradePreview.value?.proration_date)
    if (Number.isSafeInteger(prorationDate) && prorationDate > 0) {
      body.proration_date = prorationDate
    }

    const response = await apiClient("/api/membership/upgrade", {
      method: "POST",
      suppressErrorToast: true,
      body: JSON.stringify(body),
    })
    if (response?.success === false) throw new Error(response?.message || t.value.membership.upgradeFailed)
    const status = String(response?.status || "").trim().toUpperCase()

    if (status === "REQUIRES_ACTION") {
      await confirmMembershipUpgradePayment(String(response?.client_secret || "").trim())
      const orderULID = String(response?.order_ulid || "").trim()
      if (!orderULID) throw new Error(t.value.membership.upgradeOrderMissing)
      upgrading.value = false
      closeUpgradeDialog()
      activeTab.value = "overview"
      void pollMembershipUpgradeOrder(orderULID, targetMembershipULID)
      return
    }

    if (status === "PENDING_PAYMENT") {
      const orderULID = String(response?.order_ulid || "").trim()
      if (!orderULID) throw new Error(t.value.membership.upgradeOrderMissing)
      toast.info(t.value.membership.upgradePaymentPending)
      upgrading.value = false
      closeUpgradeDialog()
      activeTab.value = "overview"
      void pollMembershipUpgradeOrder(orderULID, targetMembershipULID)
      return
    }

    throw new Error(t.value.membership.upgradeStatusInvalid)
  } catch (error) {
    console.error("Failed to upgrade membership", error)
    toast.error(error instanceof Error && error.message ? error.message : t.value.membership.upgradeFailed)
  } finally {
    upgrading.value = false
  }
}

async function confirmMembershipUpgradePayment(clientSecret: string) {
  if (!clientSecret) throw new Error(t.value.membership.upgradePaymentActionMissing)
  try {
    const [config, stripeFactory] = await Promise.all([
      apiClient("/api/public/config", { suppressErrorToast: true }),
      loadStripeFactory(),
    ])
    const publishableKey = String(config?.stripe_publishable_key || "").trim()
    if (!publishableKey) throw new Error(t.value.membership.upgradePaymentActionMissing)

    const stripe = stripeFactory(publishableKey, { locale: lang.value === "zh" ? "zh" : "en" })
    const result = await stripe.confirmCardPayment(clientSecret)
    if (result?.error) {
      const detail = String(result.error.message || "").trim()
      throw new Error(detail ? `${t.value.membership.upgradePaymentVerificationFailed}: ${detail}` : t.value.membership.upgradePaymentVerificationFailed)
    }
    if (String(result?.paymentIntent?.status || "").toLowerCase() !== "succeeded") {
      toast.info(t.value.membership.upgradePaymentPending)
      return
    }
    toast.success(t.value.membership.upgradePaymentConfirmed)
  } catch (error) {
    if (error instanceof Error && error.message.startsWith(t.value.membership.upgradePaymentVerificationFailed)) throw error
    throw new Error(error instanceof Error && error.message === t.value.membership.upgradePaymentActionMissing
      ? error.message
      : t.value.membership.upgradePaymentVerificationFailed)
  }
}

function waitForUpgradeOrderPoll() {
  return new Promise<void>((resolve) => window.setTimeout(resolve, upgradeOrderPollIntervalMs))
}

async function pollMembershipUpgradeOrder(orderULID: string, targetMembershipULID: string) {
  const pollGeneration = ++upgradePollGeneration
  for (let attempt = 0; attempt < upgradeOrderPollAttempts && pollGeneration === upgradePollGeneration; attempt += 1) {
    try {
      const detail = await apiClient(`/api/orders/${encodeURIComponent(orderULID)}`, { suppressErrorToast: true })
      const orderStatus = String(detail?.summary?.order_status || "").trim().toUpperCase()
      const paymentStatus = String(detail?.summary?.payment_status || "").trim().toUpperCase()
      if (orderStatus === "COMPLETED") {
        const nextActiveMembership = await loadActiveMembershipFromHistory(history.value)
        if (pollGeneration !== upgradePollGeneration) return
        if (nextActiveMembership) activeMembership.value = nextActiveMembership

        const activeMembershipULID = String(currentRecord.value?.membership_ulid || "").trim()
        if (activeMembershipULID === targetMembershipULID) {
          await loadMembership()
          if (pollGeneration === upgradePollGeneration) toast.success(t.value.membership.upgradeSucceeded)
          return
        }
      }
      if (["CANCELLED", "CLOSED"].includes(orderStatus) || paymentStatus === "FAILED") {
        toast.error(t.value.membership.upgradeFailed)
        return
      }
    } catch (error) {
      console.warn("Failed to poll membership upgrade order", error)
    }

    if (attempt < upgradeOrderPollAttempts - 1) await waitForUpgradeOrderPoll()
  }

  if (pollGeneration === upgradePollGeneration) toast.warning(t.value.membership.upgradeSyncDelayed)
}

async function openRequestedUpgrade() {
  if (String(route.query.tab || "") === "levels") activeTab.value = "levels"
  const targetMembershipULID = String(route.query.upgrade || "").trim()
  if (!targetMembershipULID || requestedUpgradeTarget === targetMembershipULID) return
  requestedUpgradeTarget = targetMembershipULID
  const targetPlan = plans.value.find((plan) => String(plan?.membership_ulid || "").trim() === targetMembershipULID)
  if (targetPlan && canUpgradeToPlan(targetPlan)) await openUpgradePreview(targetPlan)
}

async function cancelMembership() {
  const recordUlid = currentRecord.value?.membership_record_ulid
  if (!recordUlid || !canCancelMembership.value) return
  cancelling.value = true
  try {
    await apiClient("/api/membership/cancel", {
      method: "POST",
      body: JSON.stringify({ membership_record_ulid: recordUlid, reason: "user_requested" }),
    })
    toast.success(t.value.membership.cancelAutoRenewSubmitted)
    await loadMembership()
  } finally {
    cancelling.value = false
  }
}

function resetHistoryPagination() {
  historyPage.value = 1
  lastHistoryPage.value = 1
  historyPrevCursor.value = ""
  historyNextCursor.value = ""
  historyHasMore.value = false
}

function resetBillingPagination() {
  billingPage.value = 1
  lastBillingPage.value = 1
  billingPrevCursor.value = ""
  billingNextCursor.value = ""
  billingHasMore.value = false
}

function handleHistoryPaginationChange() {
  if (loading.value) return
  if (historyPageSize.value !== lastHistoryPageSize.value) {
    lastHistoryPageSize.value = historyPageSize.value
    resetHistoryPagination()
  }
  void loadMembershipHistory()
  window.scrollTo({ top: 0, behavior: "smooth" })
}

function handleBillingPaginationChange() {
  if (loading.value) return
  if (billingPageSize.value !== lastBillingPageSize.value) {
    lastBillingPageSize.value = billingPageSize.value
    resetBillingPagination()
  }
  void loadMembershipBillings()
  window.scrollTo({ top: 0, behavior: "smooth" })
}

onMounted(async () => {
  await loadMembership()
  await openRequestedUpgrade()
})

onBeforeUnmount(() => {
  membershipRequestId += 1
  membershipPlansRequestId += 1
  membershipHistoryRequestId += 1
  membershipBillingsRequestId += 1
  upgradePollGeneration += 1
})

watch(lang, () => {
  void loadMembershipPlans().catch((error) => {
    console.warn("Failed to refresh localized membership plans", error)
  })
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.membership.title }}</span>
        <button class="membership-refresh-btn ml-auto inline-flex h-9 items-center gap-2 rounded-xl border px-4 text-sm font-semibold" type="button" @click="loadMembership">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ t.membership.refresh }}
        </button>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="membership-page-intro mb-6 flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.membership.title }}</h1>
            <p class="mt-2 text-muted-foreground">{{ t.membership.subtitle }}</p>
          </div>
          <span v-if="hasActiveMembership" class="membership-status-badge rounded-full border px-4 py-2 text-sm font-black" :class="badgeClass(currentRecord.status)">
            {{ statusLabel(currentRecord.status) }}
          </span>
        </div>

        <PageFeedback v-if="loading" kind="loading" :loading-label="t.membership.loading" />
        <PageFeedback
          v-else-if="loadError"
          kind="error"
          :title="t.membership.loadFailed"
          :description="t.membership.loadFailedDesc"
          :action-label="t.membership.retry"
          @action="loadMembership"
        />

        <template v-else>
          <section class="membership-current-card mb-5 overflow-hidden rounded-[18px] border border-slate-200 bg-white shadow-[0_10px_28px_rgba(15,74,82,0.06)]">
            <div class="membership-current-hero relative bg-[#002a66] p-6 text-white">
              <div class="membership-current-crown absolute right-6 top-6 opacity-15">
                <Crown class="h-24 w-24" />
              </div>
              <div class="relative">
                <p class="text-sm font-semibold uppercase tracking-[0.24em] text-white/70">{{ t.membership.currentMembership }}</p>
                <h2 class="membership-current-name mt-3 text-3xl font-black">
                  <span class="membership-current-name-text">{{ hasActiveMembership ? currentMembershipName : t.membership.noActiveMembership }}</span>
                </h2>
                <p class="membership-current-description mt-2 max-w-2xl text-sm text-white/80">
                  {{ hasActiveMembership ? (currentPlan?.description || currentRecord?.description || t.membership.activeMembershipDesc) : t.membership.noActiveMembershipDesc }}
                </p>
              </div>
            </div>
            <div class="membership-summary-grid grid gap-3 p-5 md:grid-cols-4">
              <div class="membership-summary-item rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ t.membership.started }}</div>
                <div class="membership-summary-value mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.started_at) }}</div>
              </div>
              <div class="membership-summary-item rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ t.membership.expires }}</div>
                <div class="membership-summary-value mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.expires_at) }}</div>
              </div>
              <div class="membership-summary-item rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ t.membership.nextBilling }}</div>
                <div class="membership-summary-value mt-2 text-sm font-black text-slate-900">{{ formatDate(currentRecord.next_billing_at) }}</div>
              </div>
              <div class="membership-summary-item rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-bold text-slate-500">{{ t.membership.autoRenew }}</div>
                <div class="membership-summary-value mt-2 text-sm font-black text-slate-900">{{ autoRenewLabel }}</div>
              </div>
            </div>
          </section>

          <div class="membership-tabs mb-4 rounded-[14px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.04)] md:px-6 md:pt-4 md:pb-0">
            <div class="relative md:hidden">
              <select
                v-model="activeTab"
                class="input h-11 cursor-pointer appearance-none rounded-xl border-slate-200 bg-slate-50 pr-10 font-semibold text-foreground shadow-sm shadow-slate-100/70 focus:bg-white"
                :aria-label="t.membership.tabsAriaLabel"
              >
                <option v-for="tab in tabs" :key="tab.id" :value="tab.id">
                  {{ tab.label }}
                </option>
              </select>
              <ChevronDown class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            </div>

            <div class="hidden flex-wrap gap-x-8 gap-y-2 border-b border-border md:flex">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                :class="['relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === tab.id ? 'text-primary' : 'text-foreground hover:text-primary']"
                @click="activeTab = tab.id"
              >
                {{ tab.label }}
                <span v-if="activeTab === tab.id" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
              </button>
            </div>
          </div>

          <section v-if="activeTab === 'overview'" class="membership-overview-grid grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
            <div class="membership-overview-card rounded-[16px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
              <h2 class="membership-section-heading mb-4 text-lg font-semibold text-card-foreground">{{ t.membership.benefits }}</h2>
              <div v-if="currentPlan && parseFeatures(currentPlan).length" class="membership-benefits-grid grid gap-3 sm:grid-cols-2">
                <div v-for="feature in parseFeatures(currentPlan)" :key="feature" class="membership-benefit-item flex gap-3 rounded-xl border border-emerald-100 bg-emerald-50/70 p-4">
                  <Check class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
                  <span class="text-sm font-medium text-slate-700">{{ feature }}</span>
                </div>
              </div>
              <div v-else class="membership-benefit-item flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
                <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
                {{ t.membership.noBenefits }}
              </div>
            </div>

            <div class="membership-overview-card rounded-[16px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
              <h2 class="membership-section-heading mb-4 text-lg font-semibold text-card-foreground">{{ t.membership.actions }}</h2>
              <div class="space-y-3 text-sm text-slate-600">
                <div class="flex justify-between gap-4"><span>{{ t.membership.membershipName }}</span><span class="text-right font-semibold text-slate-800">{{ currentMembershipName }}</span></div>
                <div class="flex justify-between"><span>{{ t.membership.source }}</span><span>{{ formatSource(currentRecord.source) }}</span></div>
                <div class="flex justify-between"><span>{{ t.membership.renewalCount }}</span><span>{{ currentRecord.renewal_count ?? "-" }}</span></div>
                <div class="flex justify-between"><span>{{ t.membership.lastPayment }}</span><span>{{ formatMoney(currentRecord.last_payment_amount_minor, "USD") }}</span></div>
              </div>
              <button
                v-if="hasActiveMembership && currentRecord.membership_record_ulid"
                class="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-xl border border-red-300 bg-red-50 px-5 py-3 font-bold text-red-700 shadow-sm transition-colors hover:border-red-400 hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-100 disabled:text-slate-400 disabled:shadow-none"
                :disabled="cancelling || !canCancelMembership"
                type="button"
                @click="openCancelRenewConfirm"
              >
                <Loader2 v-if="cancelling" class="h-4 w-4 animate-spin" />
                <XCircle v-else class="h-5 w-5 shrink-0" :stroke-width="2.5" />
                {{ cancelMembershipButtonLabel }}
              </button>
            </div>
          </section>

          <section v-if="activeTab === 'levels'" class="membership-levels-grid grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <div v-for="plan in plans" :key="plan.membership_ulid || plan.membership_gpath" class="membership-level-card relative flex h-full flex-col overflow-hidden rounded-[18px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:shadow-md">
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
                  <div class="text-xs text-slate-500">{{ t.membership.tier }}</div>
                  <div class="font-black">{{ plan.tier_level || "-" }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs text-slate-500">{{ t.membership.duration }}</div>
                  <div class="font-black">{{ plan.duration_in_months || "-" }} {{ t.membership.months }}</div>
                </div>
              </div>
              <div v-if="plan.course_discount_coupon" class="mb-4 flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm font-bold text-amber-700">
                <Percent class="h-4 w-4" />
                <span>{{ t.membership.courseDiscountCode }}{{ plan.course_discount_coupon }}</span>
              </div>
              <ul class="flex-1 space-y-2">
                <li v-for="feature in parseFeatures(plan)" :key="feature" class="flex items-center gap-2 text-sm">
                  <Check class="h-4 w-4 shrink-0 text-emerald-500" />
                  <span class="text-card-foreground">{{ feature }}</span>
                </li>
              </ul>
              <button
                v-if="canUpgradeToPlan(plan)"
                type="button"
                class="btn btn-primary mt-5 w-full rounded-xl"
                :disabled="upgradePreviewLoading || upgrading"
                @click="openUpgradePreview(plan)"
              >
                <Loader2 v-if="upgradePreviewLoading && upgradeTargetPlan?.membership_ulid === plan.membership_ulid" class="h-4 w-4 animate-spin" />
                <ArrowUpCircle v-else class="h-4 w-4" />
                {{ t.membership.upgrade }}
              </button>
              <button v-else-if="hasActiveMembership" type="button" class="btn mt-5 w-full cursor-not-allowed rounded-xl border-slate-200 bg-slate-100 text-slate-500" disabled>
                <Check v-if="isCurrentPlan(plan)" class="h-4 w-4" />
                <XCircle v-else class="h-4 w-4" />
                {{ isCurrentPlan(plan) ? t.membership.currentPlan : t.membership.upgradeUnavailable }}
              </button>
              <RouterLink v-else to="/certifications" class="btn btn-primary mt-5 w-full rounded-xl">
                <ShoppingBag class="h-4 w-4" />
                {{ t.membership.purchase }}
              </RouterLink>
            </div>
            <div v-if="!plans.length" class="rounded-[16px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)] md:col-span-2 xl:col-span-3">
              {{ t.membership.noPlans }}
            </div>
          </section>

          <section v-if="activeTab === 'history'" class="membership-records-panel overflow-hidden rounded-[16px] border border-slate-100 bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div v-for="item in history" :key="item.membership_record_ulid || item.membership_order_ulid" class="membership-record-item mb-3 grid gap-4 rounded-[14px] border border-slate-100 bg-slate-50/70 p-4 transition-all last:mb-0 hover:-translate-y-0.5 hover:border-primary/20 hover:bg-white hover:shadow-[0_12px_28px_rgba(15,74,82,0.08)] md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
              <div>
                <div class="break-words text-base font-black leading-6 text-slate-950 md:truncate" :title="membershipDisplayName(item, t.membership.membershipRecord)">{{ membershipDisplayName(item, t.membership.membershipRecord) }}</div>
                <div class="mt-1 text-sm font-medium text-slate-600">{{ formatDate(item.started_at) }} - {{ formatDate(item.expires_at) }}</div>
                <div class="mt-1 break-words text-xs text-muted-foreground md:truncate">{{ membershipRecordSummary(item) }}</div>
              </div>
              <span class="inline-flex h-fit min-w-[76px] items-center justify-center justify-self-start rounded-full border px-3 py-1 text-xs font-black md:justify-self-end" :class="badgeClass(item.status)">{{ statusLabel(item.status) }}</span>
            </div>
            <div v-if="!history.length" class="p-8 text-center text-muted-foreground">{{ t.membership.noHistory }}</div>
            <AppPagination
              v-if="historyTotal > 0"
              v-model:page="historyPage"
              v-model:page-size="historyPageSize"
              :total="historyTotal"
              :total-pages="historyTotalPages"
              :page-size-options="pageSizeOptions"
              :disabled="loading"
              cursor-mode
              :has-more="historyHasMore"
              @page-change="handleHistoryPaginationChange"
            />
          </section>

          <section v-if="activeTab === 'billings'" class="membership-records-panel overflow-hidden rounded-[16px] border border-slate-100 bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div v-for="item in billings" :key="item.billing_record_ulid || item.gpay_order_ulid" class="membership-record-item mb-3 grid gap-4 rounded-[14px] border border-slate-100 bg-slate-50/70 p-4 transition-all last:mb-0 hover:-translate-y-0.5 hover:border-primary/20 hover:bg-white hover:shadow-[0_12px_28px_rgba(15,74,82,0.08)] md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
              <div>
                <div class="break-words text-base font-black leading-6 text-slate-950 md:truncate" :title="billingTitle(item)">{{ billingTitle(item) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ formatMoney(item.amount_minor, item.currency || "USD") }} · {{ formatDate(item.period_start) }} - {{ formatDate(item.period_end) }}</div>
              </div>
              <span class="inline-flex h-fit min-w-[76px] items-center justify-center justify-self-start rounded-full border px-3 py-1 text-xs font-black md:justify-self-end" :class="badgeClass(item.status)">{{ statusLabel(item.status) }}</span>
            </div>
            <div v-if="!billings.length" class="p-8 text-center text-muted-foreground">{{ t.membership.noBillings }}</div>
            <AppPagination
              v-if="billingTotal > 0"
              v-model:page="billingPage"
              v-model:page-size="billingPageSize"
              :total="billingTotal"
              :total-pages="billingTotalPages"
              :page-size-options="pageSizeOptions"
              :disabled="loading"
              cursor-mode
              :has-more="billingHasMore"
              @page-change="handleBillingPaginationChange"
            />
          </section>
        </template>
      </main>
    </div>

    <div v-if="upgradeDialogOpen" class="app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div
        ref="upgradeDialogRef"
        class="max-h-[calc(100dvh-2rem)] w-full max-w-lg overflow-y-auto rounded-[16px] bg-white shadow-[0_24px_60px_rgba(15,23,42,0.24)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="membership-upgrade-title"
        aria-describedby="membership-upgrade-description"
        tabindex="-1"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-700">
              <ArrowUpCircle class="h-5 w-5" />
            </div>
            <div class="min-w-0">
              <h2 id="membership-upgrade-title" class="text-lg font-semibold text-slate-950">{{ t.membership.upgradePreviewTitle }}</h2>
              <p id="membership-upgrade-description" class="mt-1 truncate text-sm text-slate-500">
                {{ upgradeTargetPlan?.name || t.membership.membershipRecord }}
              </p>
            </div>
          </div>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/30 disabled:cursor-not-allowed disabled:opacity-50"
            :aria-label="t.common.close"
            :title="t.common.close"
            :disabled="upgrading"
            @click="closeUpgradeDialog"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="px-5 py-5">
          <div v-if="upgradePreviewLoading" class="flex min-h-48 items-center justify-center gap-3 text-sm font-medium text-slate-600">
            <Loader2 class="h-5 w-5 animate-spin text-blue-600" />
            {{ t.membership.upgradePreviewLoading }}
          </div>
          <div v-else-if="upgradePreviewError" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800" role="alert">
            <div class="flex gap-3">
              <AlertCircle class="mt-0.5 h-5 w-5 shrink-0" />
              <span>{{ t.membership.upgradePreviewFailed }}</span>
            </div>
          </div>
          <template v-else-if="upgradePreview">
            <div
              v-if="!upgradePreview.eligible"
              class="flex gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
              role="alert"
            >
              <AlertCircle class="mt-0.5 h-5 w-5 shrink-0" />
              <div>
                <div class="font-semibold">{{ t.membership.upgradeIneligible }}</div>
                <div class="mt-1 leading-6">{{ upgradePreview.ineligibility_reason || t.membership.upgradeIneligibleDefault }}</div>
              </div>
            </div>
            <div v-else class="flex gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
              <Check class="mt-0.5 h-5 w-5 shrink-0" />
              <span class="font-semibold">{{ t.membership.upgradeEligible }}</span>
            </div>

            <dl class="mt-5 divide-y divide-slate-100 rounded-lg border border-slate-200">
              <div class="flex items-center justify-between gap-4 px-4 py-3">
                <dt class="text-sm text-slate-600">{{ t.membership.upgradeFrom }}</dt>
                <dd class="text-right text-sm font-semibold text-slate-900">{{ upgradePreview.current_membership_name || currentMembershipName }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 px-4 py-3">
                <dt class="text-sm text-slate-600">{{ t.membership.upgradeTo }}</dt>
                <dd class="text-right text-sm font-semibold text-slate-900">{{ upgradePreview.target_membership_name || upgradeTargetPlan?.name || "-" }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 px-4 py-3">
                <dt class="text-sm text-slate-600">{{ t.membership.immediateCharge }}</dt>
                <dd class="text-right text-base font-black text-slate-950">{{ formatMoney(upgradePreview.immediate_charge_amount_minor, upgradePreview.currency) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 px-4 py-3">
                <dt class="text-sm text-slate-600">{{ t.membership.nextCycleRenewal }}</dt>
                <dd class="text-right text-sm font-semibold text-slate-900">{{ formatMoney(upgradePreview.next_cycle_renewal_amount_minor, upgradePreview.currency) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 px-4 py-3">
                <dt class="text-sm text-slate-600">{{ t.membership.currentPeriodEnds }}</dt>
                <dd class="text-right text-sm font-semibold text-slate-900">{{ formatDate(upgradePreview.current_period_ends_at) }}</dd>
              </div>
            </dl>
            <p class="mt-4 text-xs leading-5 text-slate-500">{{ t.membership.upgradeBillingNotice }}</p>
          </template>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 bg-slate-50 px-5 py-4">
          <button type="button" class="btn btn-outline min-w-24 rounded-lg" :disabled="upgrading" @click="closeUpgradeDialog">
            {{ t.common.cancel }}
          </button>
          <button
            type="button"
            class="btn btn-primary min-w-32 rounded-lg"
            :disabled="upgradePreviewLoading || upgradePreviewError || !upgradePreview?.eligible || upgrading"
            @click="confirmMembershipUpgrade"
          >
            <Loader2 v-if="upgrading" class="h-4 w-4 animate-spin" />
            <ArrowUpCircle v-else class="h-4 w-4" />
            {{ upgrading ? t.membership.upgrading : t.membership.confirmUpgrade }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="cancelRenewConfirmOpen" class="app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div
        ref="cancelRenewConfirmDialogRef"
        class="w-full max-w-md overflow-hidden rounded-[16px] bg-white shadow-[0_24px_60px_rgba(15,23,42,0.24)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="membership-cancel-renew-title"
        aria-describedby="membership-cancel-renew-description"
        tabindex="-1"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-red-50 text-red-600">
              <AlertCircle class="h-5 w-5" />
            </div>
            <h2 id="membership-cancel-renew-title" class="text-lg font-semibold text-slate-950">
              {{ t.membership.cancelAutoRenew }}
            </h2>
          </div>
          <button
            type="button"
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/30 md:h-10 md:w-10"
            :aria-label="t.common.close"
            :title="t.common.close"
            @click="closeCancelRenewConfirm"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="px-5 py-5">
          <p id="membership-cancel-renew-description" class="text-sm leading-6 text-slate-600">
            {{ t.membership.cancelAutoRenewConfirm }}
          </p>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 bg-slate-50 px-5 py-4">
          <button type="button" class="btn btn-outline min-w-28 rounded-lg" @click="closeCancelRenewConfirm">
            {{ t.membership.keepAutoRenew }}
          </button>
          <button
            type="button"
            class="btn min-w-28 rounded-lg border-red-600 bg-red-600 text-white shadow-sm shadow-red-200 hover:border-red-700 hover:bg-red-700 focus-visible:ring-red-500/40"
            @click="confirmCancelMembership"
          >
            {{ t.membership.cancelAutoRenew }}
          </button>
        </div>
      </div>
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

@media (max-width: 767px) {
  .membership-refresh-btn {
    padding-inline: 12px;
  }

  .membership-page-intro {
    gap: 8px;
    margin-bottom: 16px;
  }

  .membership-status-badge {
    padding: 6px 12px;
  }

  .membership-loading-state {
    padding: 32px 12px;
  }

  .membership-current-card {
    margin-bottom: 12px;
  }

  .membership-current-hero {
    padding: 16px;
  }

  .membership-current-crown {
    top: 16px;
    right: 16px;
  }

  .membership-current-crown > svg {
    width: 72px;
    height: 72px;
  }

  .membership-current-name {
    margin-top: 8px;
  }

  .membership-current-name-text {
    color: #ffffff;
  }

  .membership-current-description {
    margin-top: 6px;
    line-height: 20px;
  }

  .membership-summary-grid {
    gap: 8px;
    padding: 12px;
  }

  .membership-summary-item {
    padding: 12px;
  }

  .membership-summary-value {
    margin-top: 4px;
  }

  .membership-tabs {
    margin-bottom: 12px;
    padding: 12px;
  }

  .membership-overview-grid,
  .membership-levels-grid {
    gap: 12px;
  }

  .membership-overview-card,
  .membership-level-card,
  .membership-records-panel,
  .membership-record-item {
    padding: 12px;
  }

  .membership-section-heading {
    margin-bottom: 12px;
  }

  .membership-benefits-grid {
    gap: 8px;
  }

  .membership-benefit-item {
    gap: 8px;
    padding: 12px;
  }

  .membership-record-item {
    gap: 12px;
  }
}
</style>
