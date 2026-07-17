<script setup lang="ts">
import { CheckCircle2, Loader2, PlayCircle, RefreshCw, Search, X } from "lucide-vue-next"
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1600px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="refreshAll">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div class="grid gap-4 xl:grid-cols-[1fr_1fr_1.2fr_1.2fr_1.2fr_auto]">
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.flowStatus }}
          <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 px-3" @change="search">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.resultStatus }}
          <select v-model="resultStatusFilter" class="h-11 rounded-xl border border-slate-200 px-3" @change="search">
            <option v-for="option in resultOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.candidateUlid }}
          <div class="relative">
            <input v-model="candidateFilter" class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10" :placeholder="copy.filters.candidatePlaceholder" @keyup.enter="search" />
            <button
              v-if="candidateFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearCandidateSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.confirmationNumber }}
          <div class="relative">
            <input v-model="confirmationFilter" class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10" :placeholder="copy.filters.confirmationPlaceholder" @keyup.enter="search" />
            <button
              v-if="confirmationFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearConfirmationSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.courseUnitUlid }}
          <div class="relative">
            <input v-model="courseUnitFilter" class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10" :placeholder="copy.filters.courseUnitPlaceholder" @keyup.enter="search" />
            <button
              v-if="courseUnitFilter"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
              type="button"
              :aria-label="copy.filters.clearInput"
              :title="copy.filters.clearInput"
              @click="clearCourseUnitSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <button class="mt-0 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white shadow-sm xl:mt-7" type="button" @click="search">
          <Search class="h-4 w-4" />
          {{ copy.filters.search }}
        </button>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:px-5">
        <div class="min-w-0">
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
          
        <span class="shrink-0 rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
      </div>

      <div v-if="loading" class="px-4 py-10 text-center text-slate-500 md:px-6 md:py-14">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loadingList }}
      </div>
      <div v-else-if="!exams.length" class="px-4 py-10 text-center text-slate-500 md:px-6 md:py-14">{{ copy.emptyList }}</div>
      <div v-else>
        <div class="divide-y divide-slate-100 md:hidden">
          <div v-for="exam in exams" :key="examUlid(exam)" class="space-y-3 px-4 py-4 transition hover:bg-sky-50" :class="examUlid(exam) === selectedExamUlid ? 'bg-sky-50' : ''">
            <div class="min-w-0">
              <div class="break-words text-lg font-black text-slate-950">{{ field(exam, ["exam_code", "program_code", "exam_ulid"]) }}</div>
              <div class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ examUlid(exam) }}</div>
            </div>
            <div class="rounded-2xl bg-slate-50 px-3 py-2">
              <div class="text-xs font-black text-slate-400">{{ copy.columns.candidate }}</div>
              <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ candidateDisplay(exam) }}</div>
              <div v-if="candidateIdentifier(exam)" class="mt-1 break-all font-mono text-xs text-slate-500">{{ candidateIdentifier(exam) }}</div>
            </div>
            <div class="grid gap-2">
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.columns.result }}</span>
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ resultStatusLabel(exam.result_status, exam.is_passed) }}</span>
              </div>
              <div class="grid gap-1 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.columns.confirmation }}</span>
                <span class="break-all font-mono text-xs font-bold text-slate-600">{{ label(exam.confirmation_number) }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.columns.appointment }}</span>
                <span class="text-right text-sm font-semibold text-slate-700">{{ formatDate(String(exam.appointment_start_time || "")) || "-" }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2">
                <span class="text-xs font-black text-slate-400">{{ copy.columns.status }}</span>
                <span class="whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(exam.exam_status)">
                  {{ examStatusLabel(exam.exam_status) }}
                </span>
              </div>
            </div>
            <button class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline" type="button" @click="openExam(exam)">
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
        <div class="hidden overflow-x-auto md:block">
          <table class="min-w-full text-left text-sm">
            <thead class="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
              <tr>
                <th class="px-5 py-3">{{ copy.columns.exam }}</th>
                <th class="px-5 py-3">{{ copy.columns.candidate }}</th>
                <th class="px-5 py-3">{{ copy.columns.result }}</th>
                <th class="px-5 py-3">{{ copy.columns.confirmation }}</th>
                <th class="px-5 py-3">{{ copy.columns.appointment }}</th>
                <th class="px-5 py-3">{{ copy.columns.status }}</th>
                <th class="w-32 whitespace-nowrap px-5 py-3 text-right">{{ copy.columns.action }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="exam in exams" :key="examUlid(exam)" class="transition hover:bg-sky-50" :class="examUlid(exam) === selectedExamUlid ? 'bg-sky-50' : ''">
                <td class="px-5 py-4">
                  <div class="font-black text-slate-950">{{ field(exam, ["exam_code", "program_code", "exam_ulid"]) }}</div>
                  <div class="mt-1 max-w-[220px] truncate font-mono text-xs font-bold text-blue-700">{{ examUlid(exam) }}</div>
                </td>
                <td class="px-5 py-4">
                  <div class="font-semibold text-slate-800">{{ candidateDisplay(exam) }}</div>
                  <div v-if="candidateIdentifier(exam)" class="mt-1 max-w-[220px] truncate font-mono text-xs text-slate-400">{{ candidateIdentifier(exam) }}</div>
                </td>
                <td class="px-5 py-4">
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ resultStatusLabel(exam.result_status, exam.is_passed) }}</span>
                </td>
                <td class="px-5 py-4">
                  <span class="break-all font-mono text-xs font-bold text-slate-600">{{ label(exam.confirmation_number) }}</span>
                </td>
                <td class="whitespace-nowrap px-5 py-4 font-semibold text-slate-700">{{ formatDate(String(exam.appointment_start_time || "")) || "-" }}</td>
                <td class="px-5 py-4">
                  <span class="whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(exam.exam_status)">
                    {{ examStatusLabel(exam.exam_status) }}
                  </span>
                </td>
                <td class="w-32 whitespace-nowrap px-5 py-4 text-right">
                  <button class="text-sm font-bold text-blue-700 transition hover:underline" type="button" @click="openExam(exam)">
                    {{ copy.viewDetails }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="flex flex-col items-stretch justify-end gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row sm:items-center md:px-5">
        <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="changePage(page - 1)">{{ copy.prev }}</button>
        <span class="text-center text-sm font-bold text-slate-600 sm:text-left">{{ copy.pageText(page, totalPages) }}</span>
        <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="changePage(page + 1)">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section class="flex h-full max-h-none w-full max-w-[1180px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.detailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <button
                class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                :aria-label="copy.close"
                @click="closeDetailDialog"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <div v-if="detailLoading" class="px-4 py-12 text-center text-slate-500 md:px-6 md:py-16">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loadingDetail }}
            </div>
            <div v-else-if="selectedSummary" class="space-y-4 p-4 md:space-y-5 md:p-5">
          <div class="rounded-2xl bg-blue-50 p-4 md:p-5">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p class="text-xs font-black uppercase text-blue-600">{{ copy.currentExam }}</p>
                <h3 class="mt-1 break-all text-xl font-black md:text-2xl">{{ field(detail || selectedSummary, ["exam_code", "exam_ulid"]) }}</h3>
                <p class="mt-1 break-all font-mono text-sm font-bold text-blue-800">{{ selectedExamUlid }}</p>
              </div>
              <span class="rounded-full border px-3 py-1 text-sm font-black" :class="badgeClass((detail || selectedSummary)?.exam_status)">
                {{ examStatusLabel((detail || selectedSummary)?.exam_status) }}
              </span>
            </div>
          </div>

          <div class="grid gap-4 2xl:grid-cols-2">
            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">{{ copy.sections.ownership }}</h4>
              <div class="grid gap-3">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.pipeline }}</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail, ["pipeline_ulid"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.courseUnit }}</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail, ["course_unit_ulid"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.certification }}</div>
                  <div class="mt-1 font-bold">{{ field(detail, ["certification_name"]) }}</div>
                </div>
              </div>
            </article>

            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">{{ copy.sections.candidate }}</h4>
              <div class="grid gap-3 sm:grid-cols-2">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.name }}</div>
                  <div class="mt-1 font-bold">{{ candidateName }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.email }}</div>
                  <div class="mt-1 break-all font-bold">{{ field(detail || selectedSummary, ["candidate_email"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3 sm:col-span-2">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.candidateUlid }}</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail || selectedSummary, ["candidate_ulid"]) }}</div>
                </div>
              </div>
            </article>
          </div>

          <div class="grid gap-4 2xl:grid-cols-2">
            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">{{ copy.sections.appointment }}</h4>
              <div class="grid gap-3 sm:grid-cols-2">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.confirmationNumber }}</div>
                  <div class="mt-1 break-all font-bold">{{ field(detail || selectedSummary, ["confirmation_number"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.site }}</div>
                  <div class="mt-1 font-bold">{{ field(detail || selectedSummary, ["site_name"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.startTime }}</div>
                  <div class="mt-1 font-bold">{{ formatDate(String((detail || selectedSummary)?.appointment_start_time || "")) || "-" }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.endTime }}</div>
                  <div class="mt-1 font-bold">{{ formatDate(String((detail || selectedSummary)?.appointment_end_time || "")) || "-" }}</div>
                </div>
              </div>
            </article>

            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 flex items-center gap-2 text-lg font-black">
                {{ copy.sections.result }}
                <CheckCircle2 v-if="(result || detail)?.is_passed === true" class="h-5 w-5 text-emerald-500" />
              </h4>
              <div class="grid gap-3 sm:grid-cols-3">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.resultStatus }}</div>
                  <div class="mt-1 font-bold">{{ resultStatusLabel((result || detail || selectedSummary)?.result_status, (result || detail || selectedSummary)?.is_passed) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.totalScore }}</div>
                  <div class="mt-1 font-bold">{{ field(result || detail, ["total_score"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.passed }}</div>
                  <div class="mt-1 font-bold">{{ (result || detail)?.is_passed === true ? copy.yes : (result || detail)?.is_passed === false ? copy.no : "-" }}</div>
                </div>
              </div>
            </article>
          </div>

          <article class="rounded-2xl border border-slate-200">
            <div class="border-b border-slate-200 px-4 py-3">
              <h4 class="text-lg font-black">{{ copy.sections.transitions }}</h4>
              <p class="mt-1 text-sm text-slate-500">{{ copy.transitionsDescription }}</p>
            </div>
            <div v-if="!transitions.length" class="px-4 py-8 text-center text-slate-500">{{ copy.emptyTransitions }}</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="transition in transitions" :key="String(transition.msg_fp || transition.transitioned_at)" class="grid gap-3 px-4 py-4 lg:grid-cols-[160px_minmax(0,1fr)_180px]">
                <div>
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ transitionStatusTypeLabel(transition.status_type) }}</span>
                  <p class="mt-2 font-bold">{{ transitionEventLabel(transition.event_type) }}</p>
                </div>
                <div class="min-w-0">
                  <p class="font-bold">{{ transitionStatusLabel(transition, transition.from_status) }} → {{ transitionStatusLabel(transition, transition.to_status) }}</p>
                  <p class="mt-1 break-all font-mono text-xs text-blue-700">{{ label(transition.msg_fp) }}</p>
                </div>
                <div class="text-sm font-bold text-slate-500">{{ formatDate(String(transition.transitioned_at || transition.created_at || "")) || "-" }}</div>
              </div>
            </div>
          </article>

            </div>
          </div>

          <div v-if="selectedExamUlid" class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-4 py-4 md:px-6">
            <button
              class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 text-sm font-black text-white shadow-sm disabled:opacity-50 sm:w-auto"
              type="button"
              :disabled="actionLoading"
              @click="syncExamResult"
            >
              <PlayCircle class="h-4 w-4" :class="actionLoading ? 'animate-spin' : ''" />
              {{ copy.syncResult }}
            </button>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
