<script setup lang="ts">
import { CheckCircle2, Loader2, PlayCircle, RefreshCw, Search, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
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

const statusFilter = ref("")
const resultStatusFilter = ref("")
const candidateFilter = ref("")
const confirmationFilter = ref("")
const courseUnitFilter = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.exams)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value < totalPages.value)
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
  { value: "PENDING", label: copy.value.statusOptions.pending },
  { value: "PASS", label: copy.value.statusOptions.pass },
  { value: "FAIL", label: copy.value.statusOptions.fail },
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

function examStatusLabel(value: unknown) {
  const status = String(value || "").toUpperCase()
  if (status === "OPEN") return copy.value.statusOptions.open
  if (status === "SCHEDULED") return copy.value.statusOptions.scheduled
  if (status === "DONE") return copy.value.statusOptions.done
  if (status === "CANCELLED") return copy.value.statusOptions.cancelled
  return status || "-"
}

function resultStatusLabel(value: unknown, passed?: unknown) {
  const status = String(value || "").toUpperCase()
  if (status === "PASS" || passed === true) return copy.value.statusOptions.pass
  if (status === "FAIL" || (passed === false && status)) return copy.value.statusOptions.fail
  if (status === "PENDING") return copy.value.statusOptions.pending
  return status || "-"
}

function candidateDisplay(item: JsonRecord) {
  const name = [item.candidate_first_name, item.candidate_middle_name, item.candidate_last_name].filter(Boolean).join(" ")
  return name || String(item.candidate_email || item.candidate_ulid || copy.value.defaults.candidate)
}

function scoreDetails(source: JsonRecord | null) {
  const raw = String(source?.score_details_json || "")
  if (!raw) return "-"
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

async function loadExams(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(targetPage),
      page_size: String(pageSize),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    if (resultStatusFilter.value) params.set("result_status", resultStatusFilter.value)
    if (candidateFilter.value.trim()) params.set("candidate_ulid", candidateFilter.value.trim())
    if (confirmationFilter.value.trim()) params.set("confirmation_number", confirmationFilter.value.trim())
    if (courseUnitFilter.value.trim()) params.set("course_unit_ulid", courseUnitFilter.value.trim())

    const data = await apiClient<JsonRecord>(`/api/exams?${params}`)
    exams.value = asArray(data.exams)
    total.value = Number(data.total || exams.value.length || 0)
    page.value = targetPage
    if (selectedExamUlid.value && !exams.value.some((item) => examUlid(item) === selectedExamUlid.value)) {
      clearSelection()
    }
  } catch (err) {
    console.error(err)
    exams.value = []
    total.value = 0
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
    toast.error(copy.value.toasts.syncFailed)
  } finally {
    actionLoading.value = false
  }
}

async function search() {
  clearSelection()
  await loadExams(1)
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

function changePage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value) return
  loadExams(nextPage)
}

function field(source: JsonRecord | null, keys: string[]) {
  return label(pickFirst(source || {}, keys))
}

onMounted(() => loadExams(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1600px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="refreshAll">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="grid gap-4 xl:grid-cols-[1fr_1fr_1.2fr_1.2fr_1.2fr_auto]">
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.flowStatus }}
          <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 px-3">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.resultStatus }}
          <select v-model="resultStatusFilter" class="h-11 rounded-xl border border-slate-200 px-3">
            <option v-for="option in resultOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.candidateUlid }}
          <input v-model="candidateFilter" class="h-11 rounded-xl border border-slate-200 px-3" :placeholder="copy.filters.candidatePlaceholder" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.confirmationNumber }}
          <input v-model="confirmationFilter" class="h-11 rounded-xl border border-slate-200 px-3" :placeholder="copy.filters.confirmationPlaceholder" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.courseUnitUlid }}
          <input v-model="courseUnitFilter" class="h-11 rounded-xl border border-slate-200 px-3" :placeholder="copy.filters.courseUnitPlaceholder" />
        </label>
        <button class="mt-7 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 text-sm font-black text-white shadow-sm" type="button" @click="search">
          <Search class="h-4 w-4" />
          {{ copy.filters.search }}
        </button>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
      </div>

      <div v-if="loading" class="px-6 py-14 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loadingList }}
      </div>
      <div v-else-if="!exams.length" class="px-6 py-14 text-center text-slate-500">{{ copy.emptyList }}</div>
      <div v-else class="overflow-x-auto">
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
                <div class="mt-1 max-w-[220px] truncate font-mono text-xs text-slate-400">{{ label(exam.candidate_ulid) }}</div>
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
                <button class="inline-flex h-9 items-center justify-center whitespace-nowrap rounded-xl border border-blue-200 bg-blue-50 px-3 text-xs font-black text-blue-700 hover:bg-blue-100" type="button" @click="openExam(exam)">
                  {{ copy.viewDetails }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-5 py-4">
        <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="changePage(page - 1)">{{ copy.prev }}</button>
        <span class="text-sm font-bold text-slate-600">{{ copy.pageText(page, totalPages) }}</span>
        <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="changePage(page + 1)">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-6 py-5">
            <div>
              <h2 class="text-xl font-black">{{ copy.detailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <button
                v-if="selectedExamUlid"
                class="inline-flex h-10 items-center gap-2 rounded-xl bg-[#0b4ea2] px-4 text-sm font-black text-white shadow-sm disabled:opacity-50"
                type="button"
                :disabled="actionLoading"
                @click="syncExamResult"
              >
                <PlayCircle class="h-4 w-4" :class="actionLoading ? 'animate-spin' : ''" />
                {{ copy.syncResult }}
              </button>
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
            <div v-if="detailLoading" class="px-6 py-16 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loadingDetail }}
            </div>
            <div v-else-if="selectedSummary" class="space-y-5 p-5">
          <div class="rounded-2xl bg-blue-50 p-5">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p class="text-xs font-black uppercase text-blue-600">{{ copy.currentExam }}</p>
                <h3 class="mt-1 break-all text-2xl font-black">{{ field(detail || selectedSummary, ["exam_code", "exam_ulid"]) }}</h3>
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
              <pre class="mt-3 max-h-52 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100">{{ scoreDetails(result || detail) }}</pre>
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
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ label(transition.status_type) }}</span>
                  <p class="mt-2 font-bold">{{ label(transition.event_type) }}</p>
                </div>
                <div class="min-w-0">
                  <p class="font-bold">{{ label(transition.from_status) }} → {{ label(transition.to_status) }}</p>
                  <p class="mt-1 break-all font-mono text-xs text-blue-700">{{ label(transition.msg_fp) }}</p>
                </div>
                <div class="text-sm font-bold text-slate-500">{{ formatDate(String(transition.transitioned_at || transition.created_at || "")) || "-" }}</div>
              </div>
            </div>
          </article>

          <details class="rounded-2xl border border-slate-200 p-4">
            <summary class="cursor-pointer text-sm font-black text-slate-700">{{ copy.rawFields }}</summary>
            <pre class="mt-4 max-h-96 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100">{{ JSON.stringify({ detail: detail || selectedSummary, result, transitions }, null, 2) }}</pre>
          </details>
            </div>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
