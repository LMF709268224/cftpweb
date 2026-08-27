<script setup lang="ts">
import { ClipboardCheck, Download, FileSpreadsheet, Loader2, RefreshCw, Search, Upload, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

type EssayResponse = {
  question_seq: number
  question_name: string
  section_name: string
  candidate_response: string
  max_score: number
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.examGrading)
const items = ref<JsonRecord[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const previewing = ref(false)
const importing = ref(false)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const currentCursor = ref("")
const cursorHistory = ref<string[]>([])
const programCode = ref("")
const examCode = ref("")
const keyword = ref("")
const selected = ref<JsonRecord | null>(null)
const essays = ref<EssayResponse[]>([])
const detailOpen = ref(false)
const gradingFile = ref<File | null>(null)
const gradingPreview = ref<JsonRecord | null>(null)
const importConfirmed = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
let listRequestId = 0
let detailRequestId = 0

const objectiveScore = computed(() => Number(selected.value?.objective_score) || 0)
const previewItems = computed(() => Array.isArray(gradingPreview.value?.items)
  ? gradingPreview.value.items.filter((item): item is JsonRecord => !!item && typeof item === "object")
  : [])
const exportURL = computed(() => selected.value
  ? `/api/exams/${encodeURIComponent(examID(selected.value))}/essay-grade/export`
  : "")

function candidateName(item: JsonRecord | null) {
  if (!item) return "-"
  return [item.candidate_first_name, item.candidate_last_name].filter(Boolean).join(" ") || copy.value.unknownCandidate
}

function examID(item: JsonRecord | null) {
  return String(item?.exam_ulid || "")
}

function resetPagination() {
  currentCursor.value = ""
  cursorHistory.value = []
  nextCursor.value = ""
}

async function load(cursor = currentCursor.value) {
  const requestId = ++listRequestId
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: "20" })
    if (cursor) params.set("cursor", cursor)
    if (programCode.value.trim()) params.set("program_code", programCode.value.trim())
    if (examCode.value.trim()) params.set("exam_code", examCode.value.trim())
    if (keyword.value.trim()) params.set("keyword", keyword.value.trim())
    const data = await apiClient<JsonRecord>(`/api/exams/pending-grading?${params}`)
    if (requestId !== listRequestId) return
    items.value = Array.isArray(data.items) ? data.items.filter((item): item is JsonRecord => !!item && typeof item === "object") : []
    total.value = Number(data.total) || items.value.length
    hasMore.value = Boolean(data.has_more)
    nextCursor.value = String(data.next_cursor || "")
    currentCursor.value = cursor
  } catch (err) {
    if (requestId !== listRequestId) return
    console.error(err)
    items.value = []
    toast.error(apiErrorMessage(err, copy.value.toasts.listFailed))
  } finally {
    if (requestId === listRequestId) loading.value = false
  }
}

async function search() {
  resetPagination()
  await load("")
}

async function nextPage() {
  if (!hasMore.value || !nextCursor.value) return
  cursorHistory.value.push(currentCursor.value)
  await load(nextCursor.value)
}

async function previousPage() {
  if (!cursorHistory.value.length) return
  await load(cursorHistory.value.pop() || "")
}

function resetImport() {
  gradingFile.value = null
  gradingPreview.value = null
  importConfirmed.value = false
  if (fileInput.value) fileInput.value.value = ""
}

function closeDetail() {
  detailRequestId += 1
  detailOpen.value = false
  selected.value = null
  essays.value = []
  resetImport()
}

async function openDetail(summary: JsonRecord) {
  const id = examID(summary)
  if (!id) return
  const requestId = ++detailRequestId
  selected.value = summary
  essays.value = []
  resetImport()
  detailOpen.value = true
  detailLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}/essay-details`)
    if (requestId !== detailRequestId || !detailOpen.value || examID(selected.value) !== id) return
    selected.value = { ...summary, ...data }
    const details = Array.isArray(data.essays) ? data.essays : []
    essays.value = details.map((item: JsonRecord) => ({
      question_seq: Number(item.question_seq),
      question_name: String(item.question_name || ""),
      section_name: String(item.section_name || ""),
      candidate_response: String(item.candidate_response || ""),
      max_score: Number(item.max_score) || 0,
    }))
  } catch (err) {
    if (requestId !== detailRequestId) return
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.detailFailed))
    closeDetail()
  } finally {
    if (requestId === detailRequestId) detailLoading.value = false
  }
}

function gradingFormData() {
  if (!gradingFile.value) return null
  const form = new FormData()
  form.append("file", gradingFile.value)
  return form
}

async function previewWorkbook(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !selected.value) return
  gradingFile.value = file
  gradingPreview.value = null
  importConfirmed.value = false
  previewing.value = true
  try {
    const form = gradingFormData()
    if (!form) return
    gradingPreview.value = await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(examID(selected.value))}/essay-grade/import/preview`, { method: "POST", body: form })
    toast.success(copy.value.toasts.previewSuccess)
  } catch (err) {
    console.error(err)
    resetImport()
    toast.error(apiErrorMessage(err, copy.value.toasts.previewFailed))
  } finally {
    previewing.value = false
  }
}

async function importWorkbook() {
  if (!selected.value || !gradingPreview.value || !importConfirmed.value || importing.value) return
  const form = gradingFormData()
  if (!form) return
  importing.value = true
  try {
    await apiClient(`/api/exams/${encodeURIComponent(examID(selected.value))}/essay-grade/import`, { method: "POST", body: form })
    toast.success(copy.value.toasts.importSuccess)
    closeDetail()
    resetPagination()
    await load("")
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.importFailed))
  } finally {
    importing.value = false
  }
}

onMounted(() => load())
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div><h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1><p class="mt-2 text-slate-600">{{ copy.subtitle }}</p></div>
      <button class="inline-flex h-11 items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load()"><RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" /> {{ copy.refresh }}</button>
    </header>

    <section class="border-y border-slate-200 bg-white py-4">
      <div class="grid gap-3 md:grid-cols-[1fr_1fr_1.4fr_auto]">
        <input v-model="programCode" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.program" @keyup.enter="search" />
        <input v-model="examCode" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.exam" @keyup.enter="search" />
        <input v-model="keyword" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.keyword" @keyup.enter="search" />
        <button class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-blue-700 px-5 text-sm font-bold text-white" type="button" @click="search"><Search class="h-4 w-4" /> {{ copy.search }}</button>
      </div>
    </section>

    <section class="overflow-hidden border border-slate-200 bg-white">
      <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-5"><div><h2 class="font-black">{{ copy.listTitle }}</h2><p class="mt-1 text-sm text-slate-500">{{ copy.total(total) }}</p></div><ClipboardCheck class="h-5 w-5 text-blue-700" /></div>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-slate-500"><Loader2 class="h-5 w-5 animate-spin" /> {{ copy.loading }}</div>
      <div v-else-if="!items.length" class="py-16 text-center text-slate-500">{{ copy.empty }}</div>
      <div v-else class="divide-y divide-slate-200">
        <button v-for="item in items" :key="examID(item)" class="grid w-full gap-3 px-4 py-4 text-left transition hover:bg-slate-50 md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_140px_120px_auto] md:items-center md:px-5" type="button" @click="openDetail(item)">
          <div class="min-w-0"><div class="truncate font-bold text-slate-950">{{ candidateName(item) }}</div><div class="mt-1 truncate text-xs text-slate-500">{{ item.candidate_email || "-" }}</div></div>
          <div class="min-w-0 text-sm"><div class="truncate font-semibold">{{ item.program_code || "-" }} · {{ item.exam_code || "-" }}</div><div class="mt-1 truncate text-xs text-slate-500">{{ examID(item) }}</div></div>
          <div class="text-sm"><span class="text-slate-500">{{ copy.objective }} </span><b>{{ Number(item.objective_score) || 0 }}</b></div>
          <div class="text-sm"><span class="text-slate-500">{{ copy.essayCount }} </span><b>{{ Number(item.essay_count) || 0 }}</b></div>
          <span class="justify-self-start rounded-full bg-amber-50 px-3 py-1 text-xs font-bold text-amber-700 md:justify-self-end">{{ copy.review }}</span>
        </button>
      </div>
      <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-4 py-3">
        <button class="h-9 rounded-lg border border-slate-300 px-4 text-sm font-bold disabled:opacity-40" type="button" :disabled="!cursorHistory.length || loading" @click="previousPage">{{ copy.previous }}</button>
        <button class="h-9 rounded-lg border border-slate-300 px-4 text-sm font-bold disabled:opacity-40" type="button" :disabled="!hasMore || !nextCursor || loading" @click="nextPage">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-0 md:p-6">
        <div v-modal-dialog="closeDetail" class="flex h-full w-full max-w-[1100px] flex-col overflow-hidden bg-white shadow-2xl md:h-auto md:max-h-[92vh] md:rounded-lg">
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6"><div><h2 class="text-xl font-black">{{ copy.detailTitle }}</h2><p class="mt-1 text-sm text-slate-500">{{ candidateName(selected) }} · {{ examID(selected) }}</p></div><button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200" type="button" :aria-label="copy.close" @click="closeDetail"><X class="h-5 w-5" /></button></header>
          <div class="min-h-0 flex-1 overflow-y-auto">
            <div v-if="detailLoading" class="flex items-center justify-center gap-2 py-20 text-slate-500"><Loader2 class="h-5 w-5 animate-spin" /> {{ copy.loading }}</div>
            <div v-else class="space-y-6 p-4 md:p-6">
              <div class="grid gap-3 border-y border-slate-200 py-4 sm:grid-cols-4">
                <div><div class="text-xs font-bold text-slate-500">{{ copy.program }}</div><div class="mt-1 font-bold">{{ selected?.program_code || "-" }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.exam }}</div><div class="mt-1 font-bold">{{ selected?.exam_code || "-" }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.objective }}</div><div class="mt-1 font-bold">{{ objectiveScore }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.updated }}</div><div class="mt-1 text-sm font-semibold">{{ formatDate(selected?.updated_at) || "-" }}</div></div>
              </div>

              <section class="border border-blue-200 bg-blue-50 p-4"><div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between"><div><h3 class="font-black text-blue-950">{{ copy.workflowTitle }}</h3><p class="mt-1 max-w-2xl text-sm leading-6 text-blue-900">{{ copy.workflowHint }}</p></div><a class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-lg bg-blue-700 px-4 text-sm font-bold text-white" :href="exportURL"><Download class="h-4 w-4" /> {{ copy.exportWorkbook }}</a></div></section>

              <section>
                <div class="mb-3 flex items-center gap-2"><FileSpreadsheet class="h-5 w-5 text-slate-600" /><h3 class="font-black">{{ copy.responsesTitle }}</h3></div>
                <div class="space-y-5">
                  <article v-for="essay in essays" :key="essay.question_seq" class="border-b border-slate-200 pb-5 last:border-b-0">
                    <div class="flex flex-wrap items-start justify-between gap-3"><div><h4 class="font-black">{{ copy.question(essay.question_seq) }}<span v-if="essay.question_name" class="ml-2 text-sm font-semibold text-slate-500">{{ essay.question_name }}</span></h4><p v-if="essay.section_name" class="mt-1 text-sm text-slate-500">{{ essay.section_name }}</p></div><span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{{ copy.maxScore(essay.max_score) }}</span></div>
                    <div class="mt-4 whitespace-pre-wrap break-words border-l-4 border-slate-200 bg-slate-50 p-4 text-sm leading-7 text-slate-800">{{ essay.candidate_response || copy.noResponse }}</div>
                  </article>
                </div>
              </section>

              <section class="border-t border-slate-200 pt-5">
                <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"><div><h3 class="font-black">{{ copy.importTitle }}</h3><p class="mt-1 text-sm text-slate-500">{{ copy.importHint }}</p></div><button class="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-slate-300 bg-white px-4 text-sm font-bold disabled:opacity-50" type="button" :disabled="previewing || importing" @click="fileInput?.click()"><Loader2 v-if="previewing" class="h-4 w-4 animate-spin" /><Upload v-else class="h-4 w-4" /> {{ previewing ? copy.previewing : copy.selectWorkbook }}</button><input ref="fileInput" class="sr-only" type="file" accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" :aria-label="copy.selectWorkbook" @change="previewWorkbook" /></div>
                <p v-if="gradingFile" class="mt-3 break-all text-sm font-semibold text-slate-600">{{ copy.selectedFile }}{{ gradingFile.name }}</p>
                <div v-if="gradingPreview" class="mt-5 space-y-4 border border-slate-200 bg-slate-50 p-4">
                  <div class="flex flex-wrap items-center justify-between gap-3"><h4 class="font-black">{{ copy.previewTitle }}</h4><span class="rounded-full px-3 py-1 text-xs font-bold" :class="gradingPreview.is_passed ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'">{{ gradingPreview.is_passed ? copy.passed : copy.failed }}</span></div>
                  <dl class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                    <div><dt class="text-xs font-bold text-slate-500">{{ copy.graderName }}</dt><dd class="mt-1 font-black">{{ gradingPreview.grader_name || "-" }}</dd></div>
                    <div><dt class="text-xs font-bold text-slate-500">{{ copy.graderID }}</dt><dd class="mt-1 break-all text-sm font-semibold">{{ gradingPreview.grader_id || copy.notProvided }}</dd></div>
                    <div><dt class="text-xs font-bold text-slate-500">{{ copy.essaySubtotal }}</dt><dd class="mt-1 font-black">{{ gradingPreview.essay_score }}</dd></div>
                    <div><dt class="text-xs font-bold text-slate-500">{{ copy.finalScore }}</dt><dd class="mt-1 font-black text-blue-700">{{ gradingPreview.final_score }}</dd></div>
                  </dl>
                  <p v-if="gradingPreview.overall_comment" class="whitespace-pre-wrap border-l-4 border-slate-300 bg-white p-3 text-sm leading-6">{{ gradingPreview.overall_comment }}</p>
                  <div class="overflow-x-auto border border-slate-200 bg-white"><table class="w-full min-w-[640px] text-left text-sm"><thead class="bg-slate-100 text-xs text-slate-600"><tr><th class="px-3 py-2">{{ copy.questionColumn }}</th><th class="px-3 py-2">{{ copy.scoreColumn }}</th><th class="px-3 py-2">{{ copy.commentColumn }}</th></tr></thead><tbody class="divide-y divide-slate-200"><tr v-for="item in previewItems" :key="String(item.question_seq)"><td class="px-3 py-3 font-bold">{{ copy.question(Number(item.question_seq)) }} · {{ item.question_name || "-" }}</td><td class="px-3 py-3 font-black">{{ item.score }} / {{ item.max_score }}</td><td class="px-3 py-3 text-slate-600">{{ item.comment || "-" }}</td></tr></tbody></table></div>
                  <label class="flex cursor-pointer items-start gap-3 border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900"><input v-model="importConfirmed" class="mt-1" type="checkbox" /><span>{{ copy.confirmation }}</span></label>
                </div>
              </section>
            </div>
          </div>
          <footer class="flex shrink-0 justify-end gap-3 border-t border-slate-200 bg-white px-4 py-4 md:px-6"><button class="h-10 rounded-lg border border-slate-300 px-5 font-bold" type="button" :disabled="importing" @click="closeDetail">{{ copy.close }}</button><button v-if="gradingPreview" class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-lg bg-blue-700 px-5 font-bold text-white disabled:opacity-50" type="button" :disabled="importing || !importConfirmed" @click="importWorkbook"><Loader2 v-if="importing" class="h-4 w-4 animate-spin" /><Upload v-else class="h-4 w-4" />{{ importing ? copy.importing : copy.confirmImport }}</button></footer>
        </div>
      </div>
    </Teleport>
  </section>
</template>
