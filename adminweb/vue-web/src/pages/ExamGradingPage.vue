<script setup lang="ts">
import { CheckCircle2, ClipboardCheck, Loader2, RefreshCw, Search, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

type EssayGradeDraft = {
  question_seq: number
  question_name: string
  section_name: string
  candidate_response: string
  max_score: number
  score: number | null
  comment: string
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.examGrading)
const items = ref<JsonRecord[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const submitting = ref(false)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const currentCursor = ref("")
const cursorHistory = ref<string[]>([])
const programCode = ref("")
const examCode = ref("")
const keyword = ref("")
const selected = ref<JsonRecord | null>(null)
const essays = ref<EssayGradeDraft[]>([])
const detailOpen = ref(false)
const decision = ref<"" | "passed" | "failed">("")
const overallComment = ref("")
const confirmed = ref(false)
let listRequestId = 0
let detailRequestId = 0

const objectiveScore = computed(() => Number(selected.value?.objective_score) || 0)
const essayScore = computed(() => essays.value.reduce((sum, essay) => sum + (Number(essay.score) || 0), 0))
const finalScore = computed(() => objectiveScore.value + essayScore.value)

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
  prevCursor.value = ""
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
    prevCursor.value = String(data.prev_cursor || "")
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

function closeDetail() {
  detailRequestId += 1
  detailOpen.value = false
  selected.value = null
  essays.value = []
  decision.value = ""
  overallComment.value = ""
  confirmed.value = false
}

async function openDetail(summary: JsonRecord) {
  const id = examID(summary)
  if (!id) return
  const requestId = ++detailRequestId
  selected.value = summary
  essays.value = []
  decision.value = ""
  overallComment.value = ""
  confirmed.value = false
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
      score: item.score === undefined || item.score === null ? null : Number(item.score),
      comment: String(item.grader_comment || ""),
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

function validateGrade() {
  if (!decision.value) return copy.value.validation.decision
  if (!confirmed.value) return copy.value.validation.confirm
  if (!essays.value.length) return copy.value.validation.noEssays
  for (const essay of essays.value) {
    const score = Number(essay.score)
    if (essay.score === null || !Number.isFinite(score) || score < 0 || score > essay.max_score) {
      return copy.value.validation.score(essay.question_seq, essay.max_score)
    }
  }
  return ""
}

async function submitGrade() {
  if (!selected.value || submitting.value) return
  const validation = validateGrade()
  if (validation) {
    toast.error(validation)
    return
  }
  const id = examID(selected.value)
  submitting.value = true
  try {
    await apiClient(`/api/exams/${encodeURIComponent(id)}/essay-grade`, {
      method: "POST",
      body: JSON.stringify({
        is_passed: decision.value === "passed",
        overall_comment: overallComment.value.trim(),
        items: essays.value.map((essay) => ({
          question_seq: essay.question_seq,
          score: Number(essay.score),
          comment: essay.comment.trim(),
        })),
      }),
    })
    toast.success(copy.value.toasts.submitSuccess)
    closeDetail()
    resetPagination()
    await load("")
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.submitFailed))
  } finally {
    submitting.value = false
  }
}

onMounted(() => load())
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex h-11 items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load()">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" /> {{ copy.refresh }}
      </button>
    </header>

    <section class="border-y border-slate-200 bg-white py-4">
      <div class="grid gap-3 md:grid-cols-[1fr_1fr_1.4fr_auto]">
        <input v-model="programCode" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.program" @keyup.enter="search" />
        <input v-model="examCode" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.exam" @keyup.enter="search" />
        <input v-model="keyword" class="h-11 rounded-lg border border-slate-200 px-3" :placeholder="copy.filters.keyword" @keyup.enter="search" />
        <button class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-blue-700 px-5 text-sm font-bold text-white" type="button" @click="search">
          <Search class="h-4 w-4" /> {{ copy.search }}
        </button>
      </div>
    </section>

    <section class="overflow-hidden border border-slate-200 bg-white">
      <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-5">
        <div>
          <h2 class="font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.total(total) }}</p>
        </div>
        <ClipboardCheck class="h-5 w-5 text-blue-700" />
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-slate-500"><Loader2 class="h-5 w-5 animate-spin" /> {{ copy.loading }}</div>
      <div v-else-if="!items.length" class="py-16 text-center text-slate-500">{{ copy.empty }}</div>
      <div v-else class="divide-y divide-slate-200">
        <button v-for="item in items" :key="examID(item)" class="grid w-full gap-3 px-4 py-4 text-left transition hover:bg-slate-50 md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_140px_120px_auto] md:items-center md:px-5" type="button" @click="openDetail(item)">
          <div class="min-w-0">
            <div class="truncate font-bold text-slate-950">{{ candidateName(item) }}</div>
            <div class="mt-1 truncate text-xs text-slate-500">{{ item.candidate_email || "-" }}</div>
          </div>
          <div class="min-w-0 text-sm">
            <div class="truncate font-semibold">{{ item.program_code || "-" }} · {{ item.exam_code || "-" }}</div>
            <div class="mt-1 truncate text-xs text-slate-500">{{ examID(item) }}</div>
          </div>
          <div class="text-sm"><span class="text-slate-500">{{ copy.objective }} </span><b>{{ Number(item.objective_score) || 0 }}</b></div>
          <div class="text-sm"><span class="text-slate-500">{{ copy.essayCount }} </span><b>{{ Number(item.essay_count) || 0 }}</b></div>
          <span class="justify-self-start rounded-full bg-amber-50 px-3 py-1 text-xs font-bold text-amber-700 md:justify-self-end">{{ copy.grade }}</span>
        </button>
      </div>
      <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-4 py-3">
        <button class="h-9 rounded-lg border border-slate-300 px-4 text-sm font-bold disabled:opacity-40" type="button" :disabled="!cursorHistory.length || loading" @click="previousPage">{{ copy.previous }}</button>
        <button class="h-9 rounded-lg border border-slate-300 px-4 text-sm font-bold disabled:opacity-40" type="button" :disabled="!hasMore || !nextCursor || loading" @click="nextPage">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-0 md:p-6">
        <div v-modal-dialog="closeDetail" class="flex h-full w-full max-w-[1100px] flex-col overflow-hidden bg-white shadow-2xl md:max-h-[92vh] md:h-auto md:rounded-lg">
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6">
            <div>
              <h2 class="text-xl font-black">{{ copy.detailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ candidateName(selected) }} · {{ examID(selected) }}</p>
            </div>
            <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200" type="button" :aria-label="copy.close" @click="closeDetail"><X class="h-5 w-5" /></button>
          </header>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <div v-if="detailLoading" class="flex items-center justify-center gap-2 py-20 text-slate-500"><Loader2 class="h-5 w-5 animate-spin" /> {{ copy.loading }}</div>
            <div v-else class="space-y-6 p-4 md:p-6">
              <div class="grid gap-3 border-y border-slate-200 py-4 sm:grid-cols-4">
                <div><div class="text-xs font-bold text-slate-500">{{ copy.program }}</div><div class="mt-1 font-bold">{{ selected?.program_code || "-" }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.exam }}</div><div class="mt-1 font-bold">{{ selected?.exam_code || "-" }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.objective }}</div><div class="mt-1 font-bold">{{ objectiveScore }}</div></div>
                <div><div class="text-xs font-bold text-slate-500">{{ copy.updated }}</div><div class="mt-1 text-sm font-semibold">{{ formatDate(selected?.updated_at) || "-" }}</div></div>
              </div>

              <section v-for="essay in essays" :key="essay.question_seq" class="border-b border-slate-200 pb-6 last:border-b-0">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h3 class="font-black">{{ copy.question(essay.question_seq) }}<span v-if="essay.question_name" class="ml-2 text-sm font-semibold text-slate-500">{{ essay.question_name }}</span></h3>
                    <p v-if="essay.section_name" class="mt-1 text-sm text-slate-500">{{ essay.section_name }}</p>
                  </div>
                  <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{{ copy.maxScore(essay.max_score) }}</span>
                </div>
                <div class="mt-4 whitespace-pre-wrap break-words border-l-4 border-slate-200 bg-slate-50 p-4 text-sm leading-7 text-slate-800">{{ essay.candidate_response || copy.noResponse }}</div>
                <div class="mt-4 grid gap-4 md:grid-cols-[180px_minmax(0,1fr)]">
                  <label class="grid gap-2 text-sm font-bold">{{ copy.score }}
                    <input v-model.number="essay.score" class="h-11 rounded-lg border border-slate-200 px-3" type="number" min="0" :max="essay.max_score" step="0.5" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">{{ copy.comment }}
                    <input v-model="essay.comment" class="h-11 rounded-lg border border-slate-200 px-3" maxlength="500" :placeholder="copy.commentPlaceholder" />
                  </label>
                </div>
              </section>

              <section class="space-y-4 border-t border-slate-200 pt-5">
                <div class="grid gap-3 sm:grid-cols-3">
                  <div class="border border-slate-200 p-3"><div class="text-xs font-bold text-slate-500">{{ copy.objective }}</div><div class="mt-1 text-lg font-black">{{ objectiveScore }}</div></div>
                  <div class="border border-slate-200 p-3"><div class="text-xs font-bold text-slate-500">{{ copy.essaySubtotal }}</div><div class="mt-1 text-lg font-black">{{ essayScore }}</div></div>
                  <div class="border border-blue-200 bg-blue-50 p-3"><div class="text-xs font-bold text-blue-700">{{ copy.finalScore }}</div><div class="mt-1 text-lg font-black text-blue-800">{{ finalScore }}</div></div>
                </div>
                <fieldset>
                  <legend class="text-sm font-black">{{ copy.finalDecision }}</legend>
                  <div class="mt-2 flex flex-wrap gap-3">
                    <label class="flex cursor-pointer items-center gap-2 border border-slate-200 px-4 py-3 text-sm font-bold"><input v-model="decision" type="radio" value="passed" /> {{ copy.passed }}</label>
                    <label class="flex cursor-pointer items-center gap-2 border border-slate-200 px-4 py-3 text-sm font-bold"><input v-model="decision" type="radio" value="failed" /> {{ copy.failed }}</label>
                  </div>
                </fieldset>
                <label class="grid gap-2 text-sm font-bold">{{ copy.overallComment }}
                  <textarea v-model="overallComment" class="min-h-24 rounded-lg border border-slate-200 px-3 py-2" maxlength="1000" :placeholder="copy.overallCommentPlaceholder" />
                </label>
                <label class="flex cursor-pointer items-start gap-3 border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900">
                  <input v-model="confirmed" class="mt-1" type="checkbox" />
                  <span>{{ copy.confirmation }}</span>
                </label>
              </section>
            </div>
          </div>

          <footer class="flex shrink-0 justify-end gap-3 border-t border-slate-200 bg-white px-4 py-4 md:px-6">
            <button class="h-10 rounded-lg border border-slate-300 px-5 font-bold" type="button" :disabled="submitting" @click="closeDetail">{{ copy.cancel }}</button>
            <button class="inline-flex h-10 min-w-[150px] items-center justify-center gap-2 rounded-lg bg-blue-700 px-5 font-bold text-white disabled:opacity-50" type="button" :disabled="submitting || detailLoading" @click="submitGrade">
              <Loader2 v-if="submitting" class="h-4 w-4 animate-spin" /><CheckCircle2 v-else class="h-4 w-4" />{{ submitting ? copy.submitting : copy.submit }}
            </button>
          </footer>
        </div>
      </div>
    </Teleport>
  </section>
</template>
