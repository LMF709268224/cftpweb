<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, CalendarClock, CheckCircle2, ClipboardList, ExternalLink, History, Loader2, RefreshCw, Search, ShieldCheck } from "lucide-vue-next"
import { EXAM_STATUS_LABELS, normalizeEnumValueUpper, statusBadgeClassForStatusValue, statusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import AppPagination from "@/components/AppPagination.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"

type TabId = "current" | "history" | "exemption" | "records"

const { t, lang } = useTranslation()
const route = useRoute()
const router = useRouter()
const activeTab = ref<TabId>("current")
const loading = ref(false)
const scheduleLoadingExamId = ref<string | null>(null)
const pendingScheduleExamIds = ref<Record<string, number>>({})
const retakeLoadingUnitId = ref<string | null>(null)
const search = ref("")
const exams = ref<any[]>([])
const total = ref(0)
const totalPages = ref(0)
const totalLabel = ref("")
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const page = ref(1)
const pageSize = ref(10)
const lastPageSize = ref(pageSize.value)
const pageSizeOptions = [10, 30, 50, 100]
const retakePaymentSession = ref<{
  paymentKey?: string
  orderId?: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
} | null>(null)
const retakePaymentDialogOpen = ref(false)
const SCHEDULE_PENDING_TTL_MS = 2 * 60 * 1000

const tabs = computed(() => [
  { id: "current" as TabId, icon: CalendarClock, label: t.value.examsPage.currentTab },
  { id: "history" as TabId, icon: History, label: t.value.examsPage.historyTab },
  { id: "exemption" as TabId, icon: ShieldCheck, label: t.value.examsPage.exemptionTab },
  { id: "records" as TabId, icon: ClipboardList, label: t.value.examsPage.recordsTab },
])

const emptyCopy = computed(() => ({
  current: { title: t.value.examsPage.noExams, description: t.value.examsPage.noExamsDesc, icon: AlertCircle },
  history: { title: t.value.examsPage.noHistory, description: t.value.examsPage.noHistoryDesc, icon: History },
  exemption: { title: t.value.examsPage.noExemption, description: t.value.examsPage.noExemptionDesc, icon: ShieldCheck },
  records: { title: t.value.examsPage.noRecords, description: t.value.examsPage.noRecordsDesc, icon: ClipboardList },
}))

const filtered = computed(() => exams.value.filter((exam) => [
  exam.exam_id,
  exam.program_code,
  exam.exam_code,
  exam.exam_status,
  exam.result_status,
  exam.confirmation_number,
  exam.site_name,
].filter(Boolean).join(" ").toLowerCase().includes(search.value.toLowerCase())))

function normalizedExamStatus(status?: string | number | null) {
  return normalizeEnumValueUpper(status)
}
function normalizedCourseUnitStatus(status?: string | number | null) {
  return normalizeEnumValueUpper(status)
}
function isWaitingSignupExamUnit(exam: any) {
  const status = normalizedCourseUnitStatus(exam.course_unit_status)
  return status.includes("WAITING_SIGNUP_EXAM")
}
function isExamOpenUnit(exam: any) {
  const status = normalizedCourseUnitStatus(exam.course_unit_status)
  return status.includes("EXAM_OPEN")
}
function isCurrentExamRestarted(exam: any) {
  return isWaitingSignupExamUnit(exam)
}
function shouldUseCurrentCourseUnitState(exam: any) {
  return activeTab.value !== "history" && isCurrentExamRestarted(exam)
}
function shouldShowExamStatus(status?: string | number | null) {
  const normalized = normalizedExamStatus(status)
  return Boolean(normalized && !["NONE", "UNKNOWN", "UNSPECIFIED"].some((item) => normalized.includes(item)))
}
function shouldShowStoredExamDetails(exam: any) {
  return !shouldUseCurrentCourseUnitState(exam)
}
function hasExamResult(exam: any) {
  if (shouldUseCurrentCourseUnitState(exam)) return false
  const normalized = normalizedExamStatus(exam.result_status)
  return typeof exam.total_score === "number" || typeof exam.is_passed === "boolean" || ["DONE", "PASSED", "FAILED", "NO_SHOW", "RESULT_STATUS_PASSED", "RESULT_STATUS_FAILED"].includes(normalized)
}
function hasExplicitPassStatus(exam: any) {
  if (shouldUseCurrentCourseUnitState(exam)) return false
  return typeof exam.is_passed === "boolean"
}
function hasPassStatusBadge(exam: any) {
  if (shouldUseCurrentCourseUnitState(exam)) return false
  return hasExamResult(exam) || hasExplicitPassStatus(exam)
}
function shouldShowPrimaryExamStatusBadge(exam: any) {
  return shouldShowStoredExamDetails(exam) && shouldShowExamStatus(exam.exam_status) && !hasPassStatusBadge(exam)
}
function hasText(value?: string | null) {
  return Boolean(value?.trim())
}
function hasTermUrlReturn(exam: any) {
  return hasText(exam.last_termurl_timestamp)
}
function examIdOf(exam: any) {
  return String(exam?.exam_id || "").trim()
}
function cleanupExpiredPendingScheduleIds(now = Date.now()) {
  let changed = false
  const next = { ...pendingScheduleExamIds.value }
  for (const [examId, startedAt] of Object.entries(next)) {
    if (now - startedAt > SCHEDULE_PENDING_TTL_MS) {
      delete next[examId]
      changed = true
    }
  }
  if (changed) pendingScheduleExamIds.value = next
}
function markSchedulePending(examId?: string | null) {
  const normalized = String(examId || "").trim()
  if (!normalized) return
  pendingScheduleExamIds.value = {
    ...pendingScheduleExamIds.value,
    [normalized]: Date.now(),
  }
}
function isSchedulePending(exam: any) {
  const examId = examIdOf(exam)
  const startedAt = examId ? pendingScheduleExamIds.value[examId] : undefined
  return Boolean(startedAt && Date.now() - startedAt <= SCHEDULE_PENDING_TTL_MS)
}
function isWaitingScheduleSync(exam: any) {
  return activeTab.value !== "history" && (hasTermUrlReturn(exam) || isSchedulePending(exam)) && !hasExamResult(exam)
}
function hasAppointmentDetails(exam: any) {
  if (!shouldShowStoredExamDetails(exam)) return false
  return hasText(exam.confirmation_number) || hasText(exam.site_name) || hasText(exam.appointment_start_time) || hasText(exam.appointment_end_time)
}
function hasAppointmentEnded(exam: any) {
  if (!hasText(exam.appointment_end_time)) return false
  const safeStr = exam.appointment_end_time.endsWith("Z") ? exam.appointment_end_time.slice(0, -1) : exam.appointment_end_time
  const endTime = new Date(safeStr).getTime()
  return Number.isFinite(endTime) && endTime <= Date.now()
}
function shouldShowNoResultBadge(exam: any) {
  return false
}
function isExamCompletedWithoutResult(exam: any) {
  if (hasExamResult(exam)) return false
  const status = normalizedExamStatus(exam.exam_status)
  return status.includes("PASSED") || status.includes("DONE") || status.includes("COMPLETED")
}
function examStatusLabel(exam: any) {
  if (isExamCompletedWithoutResult(exam)) {
    return (t.value.examsPage as any).statusExamCompleted || t.value.examsPage.statusScheduled
  }
  return statusLabel(t.value, EXAM_STATUS_LABELS, normalizedExamStatus(exam.exam_status))
}
function canScheduleExamFromBackend(exam: any) {
  if (hasExamResult(exam) || (activeTab.value !== "history" && hasTermUrlReturn(exam))) return false
  const status = normalizedExamStatus(exam.exam_status)
  return Boolean(exam.exam_id && ((status && status.includes("OPEN")) || (activeTab.value !== "history" && isExamOpenUnit(exam))))
}
function canScheduleExam(exam: any) {
  return canScheduleExamFromBackend(exam) && !isSchedulePending(exam)
}
function canSignupExam(exam: any) {
  return Boolean(activeTab.value !== "history" && exam.course_unit_ulid && isWaitingSignupExamUnit(exam))
}
function isWaitingExamConfirmation(exam: any) {
  if (!shouldShowStoredExamDetails(exam)) return false
  return normalizedExamStatus(exam.exam_status) === "WAITING_EXAM_CONFIRMATION"
}
function isExamFailedUnit(exam: any) {
  return normalizeEnumValueUpper(exam.course_unit_status).includes("EXAM_FAILED")
}
function retakeAction(exam: any) {
  const action = String(exam?.retake?.action || "").trim().toUpperCase()
  if (action) return action
  return exam?.retake_eligible ? "CREATE_RETAKE_ORDER" : "NONE"
}
function canApplyRetake(exam: any) {
  return Boolean(exam.course_unit_ulid && exam.course_unit_cc_ulid && isExamFailedUnit(exam) && ["CREATE_RETAKE_ORDER", "CONTINUE_PAYMENT", "APPLY_RETAKE"].includes(retakeAction(exam)))
}
function retakeButtonLabel(exam: any) {
  switch (retakeAction(exam)) {
    case "CREATE_RETAKE_ORDER":
      return (t.value.examsPage as any).payRetakeFee || t.value.examsPage.applyRetake
    case "CONTINUE_PAYMENT":
      return (t.value.examsPage as any).continueRetakePayment || t.value.examsPage.applyRetake
    default:
      return t.value.examsPage.applyRetake
  }
}
function retakeMessage(exam: any) {
  return exam?.retake?.message || exam.retake_message || t.value.examsPage.examFailedDesc
}
function retakeAttemptCount(exam: any) {
  return exam?.retake?.next_retried_count || exam.next_retried_count || exam.retried_count || 0
}
function noResultLabel() {
  return (t.value.examsPage as any).statusNoResult || t.value.examsPage.statusPending
}
function resultPublishedLabel() {
  return (t.value.examsPage as any).statusResultPublished || t.value.examsPage.statusPending
}
function scheduleSyncPendingLabel() {
  return (t.value.examsPage as any).statusScheduleSyncPending || t.value.examsPage.statusWaitingExamConfirmation
}
function scheduleSyncPendingTitle() {
  return (t.value.examsPage as any).scheduleSyncPendingTitle || scheduleSyncPendingLabel()
}
function scheduleSyncPendingDesc() {
  return (t.value.examsPage as any).scheduleSyncPendingDesc || t.value.examsPage.waitingExamConfirmationDesc
}
function passStatusLabel(exam: any) {
  return exam.is_passed === true ? (t.value.examsPage as any).statusQualified || t.value.examsPage.statusPassed : (t.value.examsPage as any).statusUnqualified || t.value.examsPage.statusFailed
}
function examStatusBadgeClass(status?: string | number | null) {
  const normalized = normalizedExamStatus(status)
  if (normalized.includes("PASSED") || normalized.includes("DONE") || normalized.includes("SUCCESS")) {
    return "border-[#6CE9A6] bg-[#ECFDF3] text-[#027A48]"
  }
  return statusBadgeClassForStatusValue(status)
}
function syncPendingScheduleState(latestExams: any[]) {
  cleanupExpiredPendingScheduleIds()
  const pendingIds = Object.keys(pendingScheduleExamIds.value)
  if (pendingIds.length === 0) return

  let changed = false
  const next = { ...pendingScheduleExamIds.value }
  for (const examId of pendingIds) {
    const exam = latestExams.find((item) => examIdOf(item) === examId)
    if (!exam || !canScheduleExamFromBackend(exam)) {
      delete next[examId]
      changed = true
    }
  }
  if (changed) pendingScheduleExamIds.value = next
}
function hasPendingScheduleExams() {
  cleanupExpiredPendingScheduleIds()
  return Object.keys(pendingScheduleExamIds.value).length > 0
}
function refreshAfterScheduleReturn() {
  if (!hasPendingScheduleExams()) return
  void loadExams(activeTab.value, search.value, false, true)
}
function handleVisibilityChange() {
  if (document.visibilityState === "visible") refreshAfterScheduleReturn()
}

async function loadExams(tab: TabId = activeTab.value, keyword = search.value, showLoading = true, suppressErrorToast = false) {
  if (tab === "exemption" || tab === "records") {
    exams.value = []
    total.value = 0
    totalPages.value = 0
    totalLabel.value = "0"
    hasMore.value = false
    nextCursor.value = ""
    return
  }
  if (showLoading) loading.value = true
  try {
    const params = new URLSearchParams()
    params.set("page_size", String(pageSize.value))
    
    let cursor = ""
    if (page.value > lastPage.value) {
      cursor = nextCursor.value
    } else if (page.value < lastPage.value) {
      cursor = prevCursor.value
    }
    
    if (cursor) params.set("cursor", cursor)
    if (tab === "history") params.set("status", "DONE")
    if (keyword.trim()) params.set("confirmation_number", keyword.trim())
    const res = await apiClient(`/api/exams?${params.toString()}`, { suppressErrorToast })
    const nextExams = res?.exams || []
    exams.value = nextExams
    syncPendingScheduleState(nextExams)
    total.value = Number(res?.total || 0)
    totalLabel.value = String(res?.total_label || total.value)
    totalPages.value = Number(res?.total_pages || Math.ceil(total.value / pageSize.value) || 0)
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(res?.has_more)
    lastPage.value = page.value
nextCursor.value = String(res?.next_cursor || "")
    prevCursor.value = String(res?.prev_cursor || "")
    lastPage.value = page.value
} catch {
    exams.value = []
    total.value = 0
    totalPages.value = 0
    totalLabel.value = "0"
    hasMore.value = false
    nextCursor.value = ""
  } finally {
    if (showLoading) loading.value = false
  }
}

async function handleScheduleExam(exam: any) {
  if (!exam.exam_id || scheduleLoadingExamId.value) return
  scheduleLoadingExamId.value = exam.exam_id
  try {
    const termUrlBase = window.location.origin + "/api/public/webhooks/exams/callback"
    const params = new URLSearchParams({ url_type: "schd", term_url_base: termUrlBase })
    const res = await apiClient(`/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-url?${params.toString()}`)
    if (res?.url) {
      markSchedulePending(exam.exam_id)
      toast.info(t.value.examsPage.scheduleRedirecting)
      window.open(res.url, "_blank", "noopener,noreferrer")
    } else {
      toast.error(t.value.examsPage.scheduleURLMissing)
    }
  } catch {
    toast.error(t.value.examsPage.scheduleFailed)
  } finally {
    scheduleLoadingExamId.value = null
  }
}

async function handleApplyRetake(exam: any) {
  if (!canApplyRetake(exam) || retakeLoadingUnitId.value) return
  if (!exam.bundle_order_ulid) {
    toast.error(t.value.common.error)
    return
  }
  retakeLoadingUnitId.value = exam.course_unit_ulid
  try {
    const currentUrl = window.location.href
    const payment = await apiClient(`/api/exams/units/${encodeURIComponent(exam.course_unit_ulid)}/retake-payment`, {
      method: "POST",
      body: JSON.stringify({
        course_unit_cc_ulid: exam.course_unit_cc_ulid,
        bundle_order_ulid: exam.bundle_order_ulid,
        retried_count: retakeAttemptCount(exam),
        success_url: currentUrl,
        cancel_url: currentUrl,
      }),
    })
    if (payment?.payment_required && !payment?.paid) {
      retakePaymentSession.value = {
        paymentKey: payment.payment_key,
        orderId: payment.course_retake_order_ulid,
        bizType: "COURSE_RETAKE_PAYMENT",
        bizRefUlid: payment.course_retake_order_ulid,
        source: "retake",
        returnPath: "/exams",
      }
      retakePaymentDialogOpen.value = true
      return
    }
    if (payment?.paid && payment?.course_unit_status) {
      toast.success(t.value.examsPage.retakeApplied)
      await router.push(`/exams/signup?unitId=${encodeURIComponent(payment.course_unit_ulid || exam.course_unit_ulid)}&pipelineId=${encodeURIComponent(exam.pipeline_ulid || "")}&returnTo=${encodeURIComponent("/exams")}`)
      return
    }
    await apiClient(`/api/exams/units/${encodeURIComponent(exam.course_unit_ulid)}/retake`, { method: "POST" })
    toast.success(t.value.examsPage.retakeApplied)
    await router.push(`/exams/signup?unitId=${encodeURIComponent(exam.course_unit_ulid)}&pipelineId=${encodeURIComponent(exam.pipeline_ulid || "")}&returnTo=${encodeURIComponent("/exams")}`)
  } catch {
    // apiClient has already shown the localized error.
  } finally {
    retakeLoadingUnitId.value = null
  }
}

function resetCursorPagination() {
  page.value = 1
  lastPage.value = 1
  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
}

watch(activeTab, (tab) => {
  resetCursorPagination()
  void loadExams(tab, search.value)
})

function handlePaginationChange() {
  if (loading.value) return
  if (pageSize.value !== lastPageSize.value) {
    lastPageSize.value = pageSize.value
    resetCursorPagination()
  }
  void loadExams(activeTab.value, search.value)
}

const examsPolling = usePolling(
  () => loadExams(activeTab.value, search.value, false, true),
  { shouldPoll: () => activeTab.value === "current" || activeTab.value === "history" },
)

onMounted(() => {
  if (route.query.schedule_return === "1") toast.success(t.value.examsPage.scheduleReturnToast)
  void loadExams()
  examsPolling.start()
  window.addEventListener("focus", refreshAfterScheduleReturn)
  document.addEventListener("visibilitychange", handleVisibilityChange)
})

onBeforeUnmount(() => {
  window.removeEventListener("focus", refreshAfterScheduleReturn)
  document.removeEventListener("visibilitychange", handleVisibilityChange)
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <CalendarClock class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.examsPage.title }}</span>
      </header>

      <main class="px-4 py-6 md:px-6 lg:px-8">
        <div class="mb-5 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.examsPage.title }}</h1>
            <p class="hidden mt-2 text-muted-foreground">{{ t.examsPage.subtitle }}</p>
          </div>
          <div class="flex justify-end">
        <RouterLink to="/certifications" class="inline-flex h-9 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm shadow-primary/20 transition-colors hover:bg-primary/90">
          {{ t.courses.browseCoursesBtn }} <ExternalLink class="h-4 w-4" />
        </RouterLink>
          </div>
        </div>

    <div class="mb-4 hidden flex-col gap-3 rounded-[14px] bg-white p-3 shadow-[0_10px_24px_rgba(15,74,82,0.05)] sm:flex-row sm:items-center sm:justify-between md:p-4">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="search" class="input pl-10" :placeholder="t.examsPage.searchPlaceholder" />
      </div>
      <button class="exam-refresh-btn inline-flex h-9 items-center gap-2 rounded-xl border px-4 text-sm font-semibold" @click="() => void loadExams()">
        <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
        {{ t.examsPage.refresh }}
      </button>
    </div>

    <div class="mb-4 hidden rounded-[14px] bg-white px-5 pt-4 shadow-[0_10px_24px_rgba(15,74,82,0.04)] md:px-6">
      <div class="flex flex-wrap gap-x-8 gap-y-2 border-b border-[#edf0f2]">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['relative inline-flex cursor-pointer items-center gap-2 whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === tab.id ? 'text-primary' : 'text-[#111827] hover:text-primary']"
          @click="activeTab = tab.id"
        >
          <component :is="tab.icon" class="h-4 w-4" /> {{ tab.label }}
          <span v-if="activeTab === tab.id" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
        </button>
      </div>
    </div>

    <div class="rounded-[16px] bg-white p-3 shadow-[0_10px_24px_rgba(15,74,82,0.05)] md:p-4">
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="filtered.length === 0" class="flex flex-col items-center justify-center py-12 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <component :is="emptyCopy[activeTab].icon" class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ emptyCopy[activeTab].title }}</h3>
        <p class="max-w-md text-muted-foreground">{{ emptyCopy[activeTab].description }}</p>
      </div>
      <div v-else class="space-y-4">
        <div class="hidden items-center justify-between text-sm text-muted-foreground">
          <span>{{ t.examsPage.countPrefix }} {{ total > 0 ? total : filtered.length }} {{ t.examsPage.countSuffix }}</span>
          <span>{{ activeTab === 'history' ? t.examsPage.historyFilterHint : t.examsPage.visibleRecordsHint }}</span>
        </div>
        <div class="grid gap-3">
          <div v-for="exam in filtered" :key="exam.exam_id" class="relative overflow-hidden rounded-[14px] bg-white p-3 shadow-[0_8px_22px_rgba(15,74,82,0.05)] transition-all hover:shadow-md hover:shadow-primary/10 md:p-5">
            <div class="absolute left-0 top-0 h-full w-0.5 bg-primary md:w-1" />
            <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
              <div class="min-w-0 space-y-3 pl-1 md:space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span v-if="isExamFailedUnit(exam)" :class="['badge', statusBadgeClassForStatusValue('FAILED')]">{{ t.examsPage.examFailedTitle }}</span>
                  <template v-else>
                    <span v-if="shouldShowPrimaryExamStatusBadge(exam)" :class="['badge', examStatusBadgeClass(exam.exam_status)]">{{ examStatusLabel(exam) }}</span>
                    <span v-if="isWaitingScheduleSync(exam)" :class="['badge', statusBadgeClassForStatusValue('PENDING')]">{{ scheduleSyncPendingLabel() }}</span>
                  </template>
                  <span v-if="!isExamFailedUnit(exam) && hasPassStatusBadge(exam)" :class="['badge gap-1', exam.is_passed === true ? examStatusBadgeClass('SUCCESS') : statusBadgeClassForStatusValue('FAILED')]">
                    <CheckCircle2 v-if="exam.is_passed === true" class="h-3 w-3" />
                    {{ passStatusLabel(exam) }}
                  </span>
                </div>
                <h3 class="text-lg font-semibold leading-6 text-foreground">{{ exam.exam_code || exam.program_code || exam.exam_id || t.common.unknown }}</h3>
                <div class="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2 sm:gap-x-8 xl:grid-cols-[minmax(260px,auto)_minmax(220px,auto)]">
                  <div v-if="shouldShowStoredExamDetails(exam) && hasText(exam.confirmation_number)" class="rounded-lg bg-slate-50 px-3 py-2 sm:bg-transparent sm:px-0 sm:py-0">
                    <div class="text-xs font-medium text-muted-foreground sm:inline sm:text-sm sm:text-foreground">{{ t.examsPage.confirmationNumber }}:</div>
                    <div class="mt-1 break-all font-medium leading-5 text-foreground sm:mt-0 sm:inline sm:break-words sm:font-normal sm:text-muted-foreground"> {{ exam.confirmation_number }}</div>
                  </div>
                  <div v-if="shouldShowStoredExamDetails(exam) && hasText(exam.site_name)" class="min-w-0">
                    <span class="font-medium text-foreground">{{ t.examsPage.site }}:</span>
                    <span class="break-words"> {{ exam.site_name }}</span>
                  </div>
                  <div v-if="shouldShowStoredExamDetails(exam) && hasText(exam.appointment_start_time)" class="min-w-0"><span class="font-medium text-foreground">{{ t.examsPage.appointmentStart }}:</span> {{ formatBackendDate(exam.appointment_start_time) }}</div>
                  <div v-if="shouldShowStoredExamDetails(exam) && hasText(exam.appointment_end_time)" class="min-w-0"><span class="font-medium text-foreground">{{ t.examsPage.appointmentEnd }}:</span> {{ formatBackendDate(exam.appointment_end_time) }}</div>
                  <div v-if="isWaitingScheduleSync(exam)" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800 sm:col-span-2">
                    <div class="flex items-start gap-2">
                      <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                      <div>
                        <div class="font-medium text-amber-900">{{ scheduleSyncPendingTitle() }}</div>
                        <div class="mt-1 text-xs">{{ scheduleSyncPendingDesc() }}</div>
                      </div>
                    </div>
                  </div>
                  <div v-else-if="isWaitingExamConfirmation(exam)" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800 sm:col-span-2">
                    <div class="flex items-start gap-2">
                      <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                      <div class="text-xs">{{ t.examsPage.waitingExamConfirmationDesc }}</div>
                    </div>
                  </div>
                  <div v-if="!isWaitingScheduleSync(exam) && !isWaitingExamConfirmation(exam) && !hasAppointmentDetails(exam) && !hasExamResult(exam)" class="rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-blue-700 sm:col-span-2">
                    <div class="flex items-start gap-2">
                      <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                      <div>
                        <div class="font-medium text-blue-800">{{ t.examsPage.notScheduledTitle }}</div>
                        <div class="mt-1 text-xs">{{ t.examsPage.notScheduledDesc }}</div>
                      </div>
                    </div>
                  </div>
                  <div v-if="isExamFailedUnit(exam)" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-red-700 sm:col-span-2">
                    <div class="flex items-start gap-2">
                      <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
                      <div>
                        <div class="font-medium text-red-800">{{ t.examsPage.examFailedTitle }}</div>
                        <div class="mt-1 text-xs">{{ retakeMessage(exam) }}</div>
                      </div>
                    </div>
                  </div>
                  <div class="min-w-0"><span class="font-medium text-foreground">{{ t.examsPage.candidate }}:</span> <span class="break-words">{{ [exam.candidate_first_name, exam.candidate_last_name].filter(Boolean).join(" ") || exam.candidate_email || t.common.unknown }}</span></div>
                  <div v-if="hasExamResult(exam)" class="min-w-0"><span class="font-medium text-foreground">{{ t.examsPage.score }}:</span> {{ typeof exam.total_score === 'number' ? exam.total_score.toFixed(2) : t.common.unknown }}</div>
                </div>
              </div>
              <div class="flex flex-wrap gap-2 lg:min-w-[140px] lg:justify-end">
                <RouterLink v-if="canSignupExam(exam)" :to="`/exams/signup?unitId=${encodeURIComponent(exam.course_unit_ulid)}&pipelineId=${encodeURIComponent(exam.pipeline_ulid || '')}&returnTo=${encodeURIComponent('/exams')}`" class="btn btn-primary h-10 w-full rounded-lg px-5 shadow-sm shadow-primary/20 sm:w-auto">
                  {{ t.learning.actionSignupExam }}
                </RouterLink>
                <button v-if="canApplyRetake(exam)" class="btn btn-primary h-10 w-full rounded-lg px-5 shadow-sm shadow-primary/20 sm:w-auto" :disabled="retakeLoadingUnitId === exam.course_unit_ulid" @click="handleApplyRetake(exam)">
                  <Loader2 v-if="retakeLoadingUnitId === exam.course_unit_ulid" class="h-4 w-4 animate-spin" />
                  <RefreshCw v-else class="h-4 w-4" />
                  {{ retakeButtonLabel(exam) }}
                </button>
                <button v-if="canScheduleExam(exam)" class="btn btn-primary h-10 w-full rounded-lg px-5 shadow-sm shadow-primary/20 sm:w-auto" :disabled="scheduleLoadingExamId === exam.exam_id" @click="handleScheduleExam(exam)">
                  <Loader2 v-if="scheduleLoadingExamId === exam.exam_id" class="h-4 w-4 animate-spin" />
                  <ExternalLink v-else class="h-4 w-4" />
                  {{ t.learning.actionScheduleExam }}
                </button>
                <RouterLink v-if="hasExamResult(exam)" :to="`/exams/result?examId=${encodeURIComponent(exam.exam_id)}`" class="btn btn-primary h-10 w-full rounded-lg px-5 shadow-sm shadow-primary/20 sm:w-auto">{{ t.examsPage.viewResult }}</RouterLink>
              </div>
            </div>
          </div>
        </div>
        <AppPagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="total"
          :total-pages="totalPages"
          :total-label="totalLabel"
          :page-size-options="pageSizeOptions"
          :disabled="loading"
          :locale="lang"
          cursor-mode
          :has-more="hasMore"
          @page-change="handlePaginationChange"
        />
      </div>
    </div>
    <PaymentSessionDialog
      v-if="retakePaymentSession"
      v-model:open="retakePaymentDialogOpen"
      :title="t.examsPage.applyRetake"
      :subtitle="retakePaymentSession.orderId"
      :payment-key="retakePaymentSession.paymentKey"
      :biz-type="retakePaymentSession.bizType"
      :biz-ref-ulid="retakePaymentSession.bizRefUlid"
      :order-id="retakePaymentSession.orderId"
      :source="retakePaymentSession.source"
      :return-path="retakePaymentSession.returnPath"
    />
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.exam-refresh-btn {
  border-color: #e2e8f0;
  background: #ffffff;
  color: #334155;
  box-shadow: 0 8px 18px -16px rgba(15, 23, 42, 0.35);
  transition: transform 0.2s ease, border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.exam-refresh-btn:hover {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  box-shadow: 0 14px 28px -18px rgba(37, 99, 235, 0.42);
  transform: scale(1.02);
}

.exam-refresh-btn:active {
  transform: scale(0.98);
}

.exam-refresh-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.16), 0 14px 28px -18px rgba(37, 99, 235, 0.42);
}

</style>
