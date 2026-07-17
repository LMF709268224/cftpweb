<script setup lang="ts">
import { CheckCircle2, History, Inbox, Loader2, PlayCircle, RefreshCw, Search, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient, ApiError } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

const pageSize = 10

const exams = ref<JsonRecord[]>([])
const selectedSummary = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const result = ref<JsonRecord | null>(null)
const transitions = ref<JsonRecord[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const actionLoading = ref(false)
const detailDialogOpen = ref(false)
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)

const statusFilter = ref("")
const resultStatusFilter = ref("")
const candidateFilter = ref("")
const confirmationFilter = ref("")
const courseUnitFilter = ref("")
const appliedCandidateFilter = ref("")
const appliedConfirmationFilter = ref("")
const appliedCourseUnitFilter = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.exams)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const selectedExamUlid = computed(() => examUlid(detail.value || selectedSummary.value))
const candidateName = computed(() => {
  const source = detail.value || selectedSummary.value || {}
  return [source.candidate_first_name, source.candidate_middle_name, source.candidate_last_name].filter(Boolean).join(" ") || "-"
})

const statusOptions = computed(() => [
  { value: "", label: copy.value.statusOptions.allFlow },
  { value: "OPEN", label: copy.value.statusOptions.open },
  { value: "SCHEDULED", label: copy.value.statusOptions.scheduled },
  { value: "DONE", label: copy.value.statusOptions.done },
  { value: "CANCELLED", label: copy.value.statusOptions.cancelled },
])

const resultOptions = computed(() => [
  { value: "", label: copy.value.statusOptions.allResult },
  { value: "NONE", label: copy.value.statusOptions.none },
  { value: "AVAILABLE", label: copy.value.statusOptions.available },
  { value: "FETCHED", label: copy.value.statusOptions.fetched },
  { value: "CANCELLED", label: copy.value.statusOptions.cancelled },
  { value: "NO_SHOW", label: copy.value.statusOptions.noShow },
  { value: "BYPASSED", label: copy.value.statusOptions.bypassed },
])

function asArray(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

function examUlid(item: JsonRecord | null | undefined) {
  return String(pickFirst(item || {}, ["exam_ulid", "exam_id"]) || "")
}

function label(value: unknown) {
  return value === undefined || value === null || value === "" ? "-" : String(value)
}

function normalizedStatus(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

function normalizedResultStatus(value: unknown) {
  return normalizedStatus(value).replace(/^RESULT_STATUS_/, "")
}

function examStatusLabel(value: unknown) {
  const status = normalizedStatus(value)
  if (["OPEN", "CREATED", "EXAM_STATUS_OPEN"].includes(status)) return copy.value.statusOptions.open
  if (["SCHEDULED", "EXAM_STATUS_SCHEDULED"].includes(status)) return copy.value.statusOptions.scheduled
  if (["DONE", "COMPLETED", "EXAM_STATUS_DONE", "EXAM_STATUS_COMPLETED"].includes(status)) return copy.value.statusOptions.done
  if (["CANCELLED", "CANCELED", "EXAM_STATUS_CANCELLED", "EXAM_STATUS_CANCELED"].includes(status)) return copy.value.statusOptions.cancelled
  return status || "-"
}

function resultStatusLabel(value: unknown, passed?: unknown) {
  const status = normalizedResultStatus(value)
  if (status === "NONE") return copy.value.statusOptions.none
  if (status === "AVAILABLE") return copy.value.statusOptions.available
  if (status === "FETCHED") return copy.value.statusOptions.fetched
  if (status === "CANCELLED" || status === "CANCELED") return copy.value.statusOptions.cancelled
  if (status === "NO_SHOW") return copy.value.statusOptions.noShow
  if (status === "BYPASSED") return copy.value.statusOptions.bypassed
  if (!status && passed === true) return copy.value.statusOptions.fetched
  if (!status && passed === false) return copy.value.statusOptions.available
  return status || "-"
}

function transitionStatusTypeLabel(value: unknown) {
  const type = normalizedStatus(value)
  if (type.includes("EXAM")) return copy.value.transitionStatusTypes.exam
  if (type.includes("RESULT")) return copy.value.transitionStatusTypes.result
  return label(value)
}

function transitionEventLabel(value: unknown) {
  const event = String(value || "").trim().toLowerCase()
  const labels: Record<string, string> = {
    appointment_scheduled: copy.value.transitionEvents.appointmentScheduled,
    appointment_rescheduled: copy.value.transitionEvents.appointmentRescheduled,
    appointment_cancelled: copy.value.transitionEvents.appointmentCancelled,
    result_created: copy.value.transitionEvents.resultCreated,
    result_fetched: copy.value.transitionEvents.resultFetched,
  }
  return labels[event] || label(value)
}

function transitionStatusLabel(transition: JsonRecord, value: unknown) {
  const type = normalizedStatus(transition.status_type)
  if (type.includes("EXAM")) return examStatusLabel(value)
  if (type.includes("RESULT")) return resultStatusLabel(value)
  return label(value)
}

function candidateDisplay(item: JsonRecord) {
  const name = [item.candidate_first_name, item.candidate_middle_name, item.candidate_last_name].filter(Boolean).join(" ")
  return name || String(item.candidate_email || item.candidate_ulid || copy.value.defaults.candidate)
}

function candidateIdentifier(item: JsonRecord) {
  return String(pickFirst(item, ["candidate_ulid", "candidate_id"]) || "")
}

async function loadExams(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(pageSize),
    })

    let cursor = ""

    if (targetPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (targetPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    if (statusFilter.value) params.set("status", statusFilter.value)
    if (resultStatusFilter.value) params.set("result_status", resultStatusFilter.value)
    if (appliedCandidateFilter.value) params.set("candidate_ulid", appliedCandidateFilter.value)
    if (appliedConfirmationFilter.value) params.set("confirmation_number", appliedConfirmationFilter.value)
    if (appliedCourseUnitFilter.value) params.set("course_unit_ulid", appliedCourseUnitFilter.value)

    const isValidUlid = (id: string) => /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/i.test(id)
    let invalidUlidMessage = ""
    if (candidateFilter.value.trim() && !isValidUlid(candidateFilter.value.trim())) {
      invalidUlidMessage = copy.value.filters.invalidCandidateUlid
    } else if (courseUnitFilter.value.trim() && !isValidUlid(courseUnitFilter.value.trim())) {
      invalidUlidMessage = copy.value.filters.invalidCourseUnitUlid
    }
    if (invalidUlidMessage) {
      toast.error(invalidUlidMessage)
      exams.value = []
      total.value = 0
      hasMore.value = false
      nextCursor.value = ""
      prevCursor.value = ""
      return
    }

    const data = await apiClient<JsonRecord>(`/api/exams?${params}`)
    total.value = Number(data.total) || 0
    exams.value = asArray(data.exams)
    total.value = Number(data.total || exams.value.length || 0)
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    page.value = targetPage
    if (selectedExamUlid.value && !exams.value.some((item) => examUlid(item) === selectedExamUlid.value)) {
      clearSelection()
    }
  } catch (err) {
    console.error(err)
    exams.value = []
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

async function openExam(item: JsonRecord) {
  selectedSummary.value = item
  detail.value = null
  result.value = null
  transitions.value = []
  detailDialogOpen.value = true
  await loadExamDetail(examUlid(item))
}

async function loadExamDetail(id: string) {
  if (!id) return
  detailLoading.value = true
  try {
    const [detailData, resultData, transitionsData] = await Promise.all([
      apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}`),
      loadExamResult(id),
      apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}/transitions`),
    ])
    detail.value = detailData
    result.value = resultData
    transitions.value = asArray(transitionsData.transitions)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
  } finally {
    detailLoading.value = false
  }
}

async function loadExamResult(id: string) {
  try {
    return await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}/result`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    console.error(err)
    return null
  }
}

async function syncExamResult() {
  if (!selectedExamUlid.value) return
  actionLoading.value = true
  try {
    result.value = await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(selectedExamUlid.value)}/sync-result`, { method: "POST" })
    toast.success(copy.value.toasts.syncSuccess)
    await loadExamDetail(selectedExamUlid.value)
    await loadExams(page.value)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.syncFailed))
  } finally {
    actionLoading.value = false
  }
}

async function executeSearch() {
  clearSelection()
  resetCursorPagination()
  await loadExams(1)
}

async function search() {
  appliedCandidateFilter.value = candidateFilter.value.trim()
  appliedConfirmationFilter.value = confirmationFilter.value.trim()
  appliedCourseUnitFilter.value = courseUnitFilter.value.trim()
  await executeSearch()
}

async function clearCandidateSearch() {
  const shouldSearch = Boolean(appliedCandidateFilter.value)
  candidateFilter.value = ""
  if (!shouldSearch) return
  appliedCandidateFilter.value = ""
  await executeSearch()
}

async function clearConfirmationSearch() {
  const shouldSearch = Boolean(appliedConfirmationFilter.value)
  confirmationFilter.value = ""
  if (!shouldSearch) return
  appliedConfirmationFilter.value = ""
  await executeSearch()
}

async function clearCourseUnitSearch() {
  const shouldSearch = Boolean(appliedCourseUnitFilter.value)
  courseUnitFilter.value = ""
  if (!shouldSearch) return
  appliedCourseUnitFilter.value = ""
  await executeSearch()
}

async function refreshAll() {
  await loadExams(page.value)
  if (selectedExamUlid.value) await loadExamDetail(selectedExamUlid.value)
}

function clearSelection() {
  detailDialogOpen.value = false
  selectedSummary.value = null
  detail.value = null
  result.value = null
  transitions.value = []
}

function closeDetailDialog() {
  clearSelection()
}

function resetCursorPagination() {
  page.value = 1
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
}

function changePage(nextPage: number) {
  if (nextPage < 1 || nextPage === page.value) return
  if (nextPage > page.value && !hasMore.value) return
  if (Math.abs(nextPage - page.value) !== 1) return
  loadExams(nextPage)
}

function field(source: JsonRecord | null, keys: string[]) {
  return label(pickFirst(source || {}, keys))
}

onMounted(() => loadExams(1))
</script>

<template>
  <main class="ui-page">
    <!-- Page header -->
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" @click="refreshAll">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="loading || detailLoading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <!-- Filters -->
    <section class="ui-card mt-6 p-4 md:p-5">
      <div class="grid gap-4 xl:grid-cols-[1fr_1fr_1.2fr_1.2fr_1.2fr_auto]">
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.flowStatus }}</span>
          <select v-model="statusFilter" class="ui-input" @change="search">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.resultStatus }}</span>
          <select v-model="resultStatusFilter" class="ui-input" @change="search">
            <option v-for="option in resultOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.candidateUlid }}</span>
          <div class="relative">
            <input v-model="candidateFilter" class="ui-input pr-10" :placeholder="copy.filters.candidatePlaceholder" @keyup.enter="search" />
            <button
              v-if="candidateFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearCandidateSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.confirmationNumber }}</span>
          <div class="relative">
            <input v-model="confirmationFilter" class="ui-input pr-10" :placeholder="copy.filters.confirmationPlaceholder" @keyup.enter="search" />
            <button
              v-if="confirmationFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearConfirmationSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.courseUnitUlid }}</span>
          <div class="relative">
            <input v-model="courseUnitFilter" class="ui-input pr-10" :placeholder="copy.filters.courseUnitPlaceholder" @keyup.enter="search" />
            <button
              v-if="courseUnitFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearCourseUnitSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <button class="ui-btn ui-btn-primary xl:self-end" type="button" @click="search">
          <Search class="h-4 w-4" />
          {{ copy.filters.search }}
        </button>
      </div>
    </section>

    <!-- Exam list -->
    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
      </div>

      <!-- Mobile card list -->
      <div class="md:hidden">
        <div v-if="loading" class="space-y-4 px-4 py-4">
          <div v-for="i in 4" :key="`msk-${i}`" class="space-y-2.5">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3.5 w-4/5" />
            <div class="skeleton h-3.5 w-3/5" />
          </div>
        </div>
        <div v-else-if="!exams.length" class="ui-empty">
          <div class="ui-empty-icon">
            <Inbox class="h-6 w-6" />
          </div>
          <p class="ui-empty-title">{{ copy.emptyList }}</p>
        </div>
        <div v-else class="divide-y divide-hairline">
          <article
            v-for="exam in exams"
            :key="examUlid(exam)"
            class="px-4 py-4 transition-colors"
            :class="examUlid(exam) === selectedExamUlid ? 'bg-primary-50/60' : ''"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="break-words text-sm font-semibold text-ink">{{ field(exam, ["exam_code", "program_code", "exam_ulid"]) }}</div>
                <div class="tnum mt-0.5 break-all text-xs text-slate-400">{{ examUlid(exam) }}</div>
              </div>
              <span class="ui-badge shrink-0" :class="badgeClass(exam.exam_status)">{{ examStatusLabel(exam.exam_status) }}</span>
            </div>
            <dl class="mt-3 divide-y divide-hairline text-sm">
              <div class="flex items-start justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.columns.candidate }}</dt>
                <dd class="m-0 min-w-0 text-right">
                  <div class="break-words text-slate-600">{{ candidateDisplay(exam) }}</div>
                  <div v-if="candidateIdentifier(exam)" class="tnum break-all text-xs text-slate-400">{{ candidateIdentifier(exam) }}</div>
                </dd>
              </div>
              <div class="flex items-center justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.columns.result }}</dt>
                <dd class="m-0">
                  <span class="ui-badge bg-slate-100 text-slate-600">{{ resultStatusLabel(exam.result_status, exam.is_passed) }}</span>
                </dd>
              </div>
              <div class="flex items-start justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.columns.confirmation }}</dt>
                <dd class="tnum m-0 break-all text-right text-slate-600">{{ label(exam.confirmation_number) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4 py-2">
                <dt class="shrink-0 text-xs font-medium text-slate-400">{{ copy.columns.appointment }}</dt>
                <dd class="tnum m-0 text-right text-slate-600">{{ formatDate(String(exam.appointment_start_time || "")) || "-" }}</dd>
              </div>
            </dl>
            <button class="ui-btn ui-btn-secondary ui-btn-sm mt-2 w-full" type="button" @click="openExam(exam)">
              {{ copy.viewDetails }}
            </button>
          </article>
        </div>
      </div>

      <!-- Desktop table -->
      <div class="hidden overflow-x-auto md:block">
        <table class="w-full min-w-[1120px] text-left text-sm">
          <thead class="ui-table-head">
            <tr>
              <th class="px-4 py-3 font-medium">{{ copy.columns.exam }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.columns.candidate }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.columns.result }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.columns.confirmation }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.columns.appointment }}</th>
              <th class="px-4 py-3 font-medium">{{ copy.columns.status }}</th>
              <th class="w-28 whitespace-nowrap px-4 py-3 text-right font-medium">{{ copy.columns.action }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-hairline">
            <template v-if="loading">
              <tr v-for="i in 5" :key="`sk-${i}`">
                <td class="px-4 py-3.5">
                  <div class="skeleton h-4 w-28" />
                  <div class="skeleton mt-2 h-3 w-36" />
                </td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-32" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-24" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-4 w-24" /></td>
                <td class="px-4 py-3.5"><div class="skeleton h-5 w-16 rounded-full" /></td>
                <td class="px-4 py-3.5"><div class="skeleton ml-auto h-4 w-14" /></td>
              </tr>
            </template>
            <tr v-else-if="!exams.length">
              <td colspan="7">
                <div class="ui-empty">
                  <div class="ui-empty-icon">
                    <Inbox class="h-6 w-6" />
                  </div>
                  <p class="ui-empty-title">{{ copy.emptyList }}</p>
                </div>
              </td>
            </tr>
            <template v-else>
              <tr
                v-for="exam in exams"
                :key="examUlid(exam)"
                class="ui-table-row"
                :class="examUlid(exam) === selectedExamUlid ? 'bg-primary-50/60' : ''"
              >
                <td class="px-4 py-3.5">
                  <div class="font-medium text-ink">{{ field(exam, ["exam_code", "program_code", "exam_ulid"]) }}</div>
                  <div class="tnum max-w-[220px] truncate text-xs text-slate-400">{{ examUlid(exam) }}</div>
                </td>
                <td class="px-4 py-3.5">
                  <div class="text-slate-600">{{ candidateDisplay(exam) }}</div>
                  <div v-if="candidateIdentifier(exam)" class="tnum max-w-[220px] truncate text-xs text-slate-400">{{ candidateIdentifier(exam) }}</div>
                </td>
                <td class="px-4 py-3.5">
                  <span class="ui-badge bg-slate-100 text-slate-600">{{ resultStatusLabel(exam.result_status, exam.is_passed) }}</span>
                </td>
                <td class="tnum px-4 py-3.5 text-slate-600">{{ label(exam.confirmation_number) }}</td>
                <td class="tnum whitespace-nowrap px-4 py-3.5 text-slate-600">{{ formatDate(String(exam.appointment_start_time || "")) || "-" }}</td>
                <td class="px-4 py-3.5">
                  <span class="ui-badge" :class="badgeClass(exam.exam_status)">{{ examStatusLabel(exam.exam_status) }}</span>
                </td>
                <td class="w-28 whitespace-nowrap px-4 py-3.5 text-right">
                  <button class="text-sm font-medium text-primary-700 transition-colors hover:text-primary-600" type="button" @click="openExam(exam)">
                    {{ copy.viewDetails }}
                  </button>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page, totalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="changePage(page - 1)">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="changePage(page + 1)">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <!-- Detail dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailDialogOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1180px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="closeDetailDialog"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 overflow-y-auto">
              <div v-if="detailLoading" class="space-y-3 p-4 md:p-6">
                <div class="skeleton h-24 w-full" />
                <div class="skeleton h-12 w-4/5" />
                <div class="skeleton h-12 w-full" />
                <div class="skeleton h-12 w-3/5" />
              </div>
              <div v-else-if="selectedSummary" class="space-y-4 p-4 md:space-y-5 md:p-6">
                <div class="rounded-2xl bg-primary-50 p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0">
                      <p class="text-xs font-medium uppercase tracking-wider text-primary-600">{{ copy.currentExam }}</p>
                      <h3 class="mt-1 break-all text-lg font-semibold text-ink md:text-xl">{{ field(detail || selectedSummary, ["exam_code", "exam_ulid"]) }}</h3>
                      <p class="tnum mt-1 break-all text-sm text-primary-700">{{ selectedExamUlid }}</p>
                    </div>
                    <span class="ui-badge shrink-0" :class="badgeClass((detail || selectedSummary)?.exam_status)">
                      {{ examStatusLabel((detail || selectedSummary)?.exam_status) }}
                    </span>
                  </div>
                </div>

                <div class="grid gap-4 2xl:grid-cols-2">
                  <article class="rounded-2xl border border-edge p-4 md:p-5">
                    <h4 class="text-base font-semibold text-ink">{{ copy.sections.ownership }}</h4>
                    <dl class="mt-4 grid gap-4">
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.pipeline }}</dt>
                        <dd class="tnum m-0 mt-1 break-all text-sm font-medium text-ink">{{ field(detail, ["pipeline_ulid"]) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.courseUnit }}</dt>
                        <dd class="tnum m-0 mt-1 break-all text-sm font-medium text-ink">{{ field(detail, ["course_unit_ulid"]) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.certification }}</dt>
                        <dd class="m-0 mt-1 break-words text-sm font-medium text-ink">{{ field(detail, ["certification_name"]) }}</dd>
                      </div>
                    </dl>
                  </article>

                  <article class="rounded-2xl border border-edge p-4 md:p-5">
                    <h4 class="text-base font-semibold text-ink">{{ copy.sections.candidate }}</h4>
                    <dl class="mt-4 grid gap-4 sm:grid-cols-2">
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.name }}</dt>
                        <dd class="m-0 mt-1 break-words text-sm font-medium text-ink">{{ candidateName }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.email }}</dt>
                        <dd class="m-0 mt-1 break-all text-sm font-medium text-ink">{{ field(detail || selectedSummary, ["candidate_email"]) }}</dd>
                      </div>
                      <div class="sm:col-span-2">
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.candidateUlid }}</dt>
                        <dd class="tnum m-0 mt-1 break-all text-sm font-medium text-ink">{{ field(detail || selectedSummary, ["candidate_ulid"]) }}</dd>
                      </div>
                    </dl>
                  </article>
                </div>

                <div class="grid gap-4 2xl:grid-cols-2">
                  <article class="rounded-2xl border border-edge p-4 md:p-5">
                    <h4 class="text-base font-semibold text-ink">{{ copy.sections.appointment }}</h4>
                    <dl class="mt-4 grid gap-4 sm:grid-cols-2">
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.confirmationNumber }}</dt>
                        <dd class="tnum m-0 mt-1 break-all text-sm font-medium text-ink">{{ field(detail || selectedSummary, ["confirmation_number"]) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.site }}</dt>
                        <dd class="m-0 mt-1 break-words text-sm font-medium text-ink">{{ field(detail || selectedSummary, ["site_name"]) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.startTime }}</dt>
                        <dd class="tnum m-0 mt-1 text-sm font-medium text-ink">{{ formatDate(String((detail || selectedSummary)?.appointment_start_time || "")) || "-" }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.endTime }}</dt>
                        <dd class="tnum m-0 mt-1 text-sm font-medium text-ink">{{ formatDate(String((detail || selectedSummary)?.appointment_end_time || "")) || "-" }}</dd>
                      </div>
                    </dl>
                  </article>

                  <article class="rounded-2xl border border-edge p-4 md:p-5">
                    <h4 class="flex items-center gap-2 text-base font-semibold text-ink">
                      {{ copy.sections.result }}
                      <CheckCircle2 v-if="(result || detail)?.is_passed === true" class="h-4 w-4 text-success-600" />
                    </h4>
                    <dl class="mt-4 grid gap-4 sm:grid-cols-3">
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.resultStatus }}</dt>
                        <dd class="m-0 mt-1 text-sm font-medium text-ink">{{ resultStatusLabel((result || detail || selectedSummary)?.result_status, (result || detail || selectedSummary)?.is_passed) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.totalScore }}</dt>
                        <dd class="tnum m-0 mt-1 text-sm font-medium text-ink">{{ field(result || detail, ["total_score"]) }}</dd>
                      </div>
                      <div>
                        <dt class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.passed }}</dt>
                        <dd class="m-0 mt-1 text-sm font-medium text-ink">{{ (result || detail)?.is_passed === true ? copy.yes : (result || detail)?.is_passed === false ? copy.no : "-" }}</dd>
                      </div>
                    </dl>
                  </article>
                </div>

                <article class="rounded-2xl border border-edge">
                  <div class="border-b border-hairline px-4 py-4 md:px-5">
                    <h4 class="text-base font-semibold text-ink">{{ copy.sections.transitions }}</h4>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.transitionsDescription }}</p>
                  </div>
                  <div v-if="!transitions.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <History class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.emptyTransitions }}</p>
                  </div>
                  <div v-else class="divide-y divide-hairline">
                    <div
                      v-for="transition in transitions"
                      :key="String(transition.msg_fp || transition.transitioned_at)"
                      class="grid gap-3 px-4 py-4 md:px-5 lg:grid-cols-[160px_minmax(0,1fr)_180px]"
                    >
                      <div>
                        <span class="ui-badge bg-slate-100 text-slate-600">{{ transitionStatusTypeLabel(transition.status_type) }}</span>
                        <p class="mt-2 text-sm font-medium text-ink">{{ transitionEventLabel(transition.event_type) }}</p>
                      </div>
                      <div class="min-w-0">
                        <p class="text-sm font-medium text-ink">{{ transitionStatusLabel(transition, transition.from_status) }} → {{ transitionStatusLabel(transition, transition.to_status) }}</p>
                        <p class="tnum mt-1 break-all text-xs text-slate-400">{{ label(transition.msg_fp) }}</p>
                      </div>
                      <div class="tnum text-sm text-slate-500">{{ formatDate(String(transition.transitioned_at || transition.created_at || "")) || "-" }}</div>
                    </div>
                  </div>
                </article>
              </div>
            </div>

            <div v-if="selectedExamUlid" class="flex shrink-0 justify-end border-t border-hairline px-4 py-4 md:px-6">
              <button class="ui-btn ui-btn-primary w-full sm:w-auto" type="button" :disabled="actionLoading" @click="syncExamResult">
                <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
                <PlayCircle v-else class="h-4 w-4" />
                {{ copy.syncResult }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
