<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, CheckCircle2, ChevronLeft, Clock, FileText, Loader2, X } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useDialogAccessibility } from "@/lib/dialogAccessibility"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const attemptId = computed(() => String(route.query.attemptId || ""))
const returnTo = computed(() => {
  const raw = route.query.returnTo
  const value = Array.isArray(raw) ? String(raw[0] || "").trim() : String(raw || "").trim()
  if (!value || !value.startsWith("/") || value.startsWith("//")) return ""
  return value
})
const loading = ref(true)
const submitting = ref(false)
const remainingSeconds = ref<number>(0)
const timerInterval = ref<number | null>(null)
const paper = ref<any>(null)
const answers = ref<Record<string, string[]>>({})
const result = ref<any>(null)
const detailedAnswers = ref<any[]>([])
const showDetail = ref(false)
const submitConfirmOpen = ref(false)
const submitConfirmDialogRef = ref<HTMLElement | null>(null)
const QUIZ_DRAFT_STORAGE_PREFIX = "cftp:quiz-draft:"
const QUIZ_DRAFT_MAX_AGE_MS = 1000 * 60 * 60 * 24 * 7

useBodyScrollLock(() => submitConfirmOpen.value)

const questions = computed(() => paper.value?.questions || [])
const answeredCount = computed(() => questions.value.filter((q: any) => (answers.value[questionIdOf(q)]?.length || 0) > 0).length)
const allAnswered = computed(() => questions.value.every((q: any) => (answers.value[questionIdOf(q)]?.length || 0) > 0))
const quizPassed = computed(() => {
  if (Number(result.value?.pass_status) === 1) return true
  if (Number(result.value?.pass_status) === 2) return false
  return result.value?.is_passed === true
})

function firstString(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim()
    if (typeof value === "number" && Number.isFinite(value)) return String(value)
  }
  return ""
}

function questionIdOf(question: any) {
  return firstString(question?.question_id, question?.question_ulid, question?.questionUlid)
}

function optionIdOf(option: any) {
  return firstString(option?.option_id, option?.option_ulid, option?.optionUlid)
}

function normalizeQuestion(question: any) {
  const options = Array.isArray(question?.options)
    ? question.options.map((option: any) => ({ ...option, option_id: optionIdOf(option) }))
    : question?.options
  return { ...question, question_id: questionIdOf(question), options }
}

function formatQuizQuestionCount(current: number, total: number) {
  return (t.value.learning?.quizQuestionCount || "")
    .replace("{{current}}", String(current))
    .replace("{{total}}", String(total))
}

const formattedTimer = computed(() => {
  if (remainingSeconds.value <= 0) return ""
  const m = Math.floor(remainingSeconds.value / 60)
  const s = remainingSeconds.value % 60
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`
})

function startTimer() {
  stopTimer()
  if (remainingSeconds.value <= 0) return
  timerInterval.value = window.setInterval(() => {
    remainingSeconds.value--
    if (remainingSeconds.value <= 0) {
      stopTimer()
      toast.error(t.value.learning.quizTimeUp)
      void submitQuiz()
    }
  }, 1000)
}

function stopTimer() {
  if (timerInterval.value) {
    window.clearInterval(timerInterval.value)
    timerInterval.value = null
  }
}

function formatQuizAnsweredCount(current: number, total: number) {
  return (t.value.learning?.quizAnsweredCount || "")
    .replace("{{current}}", String(current))
    .replace("{{total}}", String(total))
}

function quizDraftStorageKey() {
  return attemptId.value ? `${QUIZ_DRAFT_STORAGE_PREFIX}${attemptId.value}` : ""
}

function hasAnswerValues(value: Record<string, string[]>) {
  return Object.values(value).some((ids) => Array.isArray(ids) && ids.length > 0)
}

function sanitizeAnswersForCurrentPaper(value: unknown) {
  if (!value || typeof value !== "object") return {}
  const source = value as Record<string, unknown>
  const sanitized: Record<string, string[]> = {}

  questions.value.forEach((question: any) => {
    const questionId = questionIdOf(question)
    if (!questionId) return
    const optionIds = new Set((Array.isArray(question.options) ? question.options : []).map(optionIdOf).filter(Boolean))
    const selected = Array.isArray(source[questionId]) ? (source[questionId] as unknown[]) : []
    const uniqueSelected = Array.from(new Set(selected.map((id) => firstString(id)).filter(Boolean)))
      .filter((id) => optionIds.size === 0 || optionIds.has(id))
    const normalizedSelected = Number(question.question_type) === 2 ? uniqueSelected : uniqueSelected.slice(-1)
    if (normalizedSelected.length > 0) sanitized[questionId] = normalizedSelected
  })

  return sanitized
}

function clearQuizDraft() {
  const key = quizDraftStorageKey()
  if (!key) return
  try {
    window.localStorage.removeItem(key)
  } catch (err) {
    console.warn("Failed to clear quiz draft", err)
  }
}

function persistQuizDraft(nextAnswers = answers.value) {
  const key = quizDraftStorageKey()
  if (!key) return
  const sanitized = sanitizeAnswersForCurrentPaper(nextAnswers)
  if (!hasAnswerValues(sanitized)) {
    clearQuizDraft()
    return
  }
  try {
    window.localStorage.setItem(key, JSON.stringify({ attemptId: attemptId.value, answers: sanitized, updatedAt: Date.now() }))
  } catch (err) {
    console.warn("Failed to save quiz draft", err)
  }

  // Silent remote sync
  const submissions = Object.entries(sanitized).map(([questionId, selectedOptionIds]) => ({ question_id: questionId, selected_option_ids: selectedOptionIds }))
  apiClient(`/api/quizzes/attempts/${attemptId.value}/draft`, { method: "POST", body: JSON.stringify({ submissions }) }).catch(err => {
    console.warn("Failed to sync quiz draft to server", err)
  })
}

function restoreQuizDraft() {
  const key = quizDraftStorageKey()
  if (!key) return
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return
    const payload = JSON.parse(raw) as { attemptId?: string; answers?: Record<string, string[]>; updatedAt?: number }
    if (payload.attemptId && payload.attemptId !== attemptId.value) {
      window.localStorage.removeItem(key)
      return
    }
    if (payload.updatedAt && Date.now() - payload.updatedAt > QUIZ_DRAFT_MAX_AGE_MS) {
      window.localStorage.removeItem(key)
      return
    }
    const restoredAnswers = sanitizeAnswersForCurrentPaper(payload.answers)
    if (!hasAnswerValues(restoredAnswers)) {
      window.localStorage.removeItem(key)
      return
    }
    answers.value = restoredAnswers
    toast.info(t.value.learning?.quizDraftRestored || "")
  } catch (err) {
    console.warn("Failed to restore quiz draft", err)
    clearQuizDraft()
  }
}

function quizLeaveConfirmMessage() {
  return t.value.learning?.quizLeaveConfirm || "You have unsubmitted answers. Leave this quiz?"
}

function quizSubmitConfirmMessage() {
  return t.value.learning?.quizSubmitConfirm || "Submit this quiz now? You cannot change your answers after submitting."
}

function openSubmitConfirm() {
  if (!paper.value?.questions || submitting.value || !allAnswered.value) return
  submitConfirmOpen.value = true
}

function closeSubmitConfirm() {
  submitConfirmOpen.value = false
}

function confirmSubmitQuiz() {
  closeSubmitConfirm()
  void submitQuiz()
}

useDialogAccessibility(() => submitConfirmOpen.value, submitConfirmDialogRef, closeSubmitConfirm)

function shouldConfirmLeavingQuiz() {
  return !result.value && answeredCount.value > 0
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!shouldConfirmLeavingQuiz()) return
  event.preventDefault()
  event.returnValue = quizLeaveConfirmMessage()
}

async function loadPaper() {
  if (!attemptId.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const res = await apiClient(`/api/quizzes/attempts/${attemptId.value}/paper`)
    paper.value = {
      ...res,
      questions: Array.isArray(res?.questions) ? res.questions.map(normalizeQuestion) : [],
    }
    
    if (res?.draft_submissions && res.draft_submissions.length > 0) {
      const serverAnswers: Record<string, string[]> = {}
      res.draft_submissions.forEach((sub: any) => {
         const qid = firstString(sub.question_id, sub.question_ulid, sub.questionUlid)
         const opts = Array.isArray(sub.selected_option_ids) ? sub.selected_option_ids : (sub.selectedOptionIds || [])
         if (qid && opts.length > 0) {
            serverAnswers[qid] = opts
         }
      })
      if (hasAnswerValues(serverAnswers)) {
          answers.value = sanitizeAnswersForCurrentPaper(serverAnswers)
      } else {
          restoreQuizDraft()
      }
    } else {
      restoreQuizDraft()
    }

    if (res?.remaining_seconds > 0) {
      remainingSeconds.value = Number(res.remaining_seconds)
      startTimer()
    }
  } finally {
    loading.value = false
  }
}

function handleSelectOption(questionId: string, optionId: string, isMultipleChoice: boolean) {
  const current = answers.value[questionId] || []
  if (!isMultipleChoice) {
    answers.value = { ...answers.value, [questionId]: [optionId] }
    persistQuizDraft()
    return
  }
  answers.value = {
    ...answers.value,
    [questionId]: current.includes(optionId) ? current.filter((id) => id !== optionId) : [...current, optionId],
  }
  persistQuizDraft()
}

async function submitQuiz() {
  if (!paper.value?.questions || submitting.value) return
  closeSubmitConfirm()
  const sanitizedAnswers = sanitizeAnswersForCurrentPaper(answers.value)
  const submissions = Object.entries(sanitizedAnswers).map(([questionId, selectedOptionIds]) => ({ question_id: questionId, selected_option_ids: selectedOptionIds }))
  submitting.value = true
  try {
    result.value = await apiClient(`/api/quizzes/attempts/${attemptId.value}/submit`, { method: "POST", body: JSON.stringify({ submissions }) })
    clearQuizDraft()
    toast.success(t.value.learning?.quizSubmittedDesc || t.value.common.success)
  } catch (err: any) {
    if (err?.error_code === "FAILED_PRECONDITION" || err?.status === 412 || err?.status === 400) {
      toast.info(t.value.learning.quizTimeEndedLoadingResults)
      await loadAttemptDetail()
      if (detailedAnswers.value.length > 0) {
         result.value = { score: 0, max_score: 0 } 
      }
    } else {
      throw err
    }
  } finally {
    submitting.value = false
    stopTimer()
  }
}

async function loadAttemptDetail() {
  try {
    const res = await apiClient(`/api/quizzes/attempts/${attemptId.value}/detail`)
    if (res && res.answers_json) {
      detailedAnswers.value = JSON.parse(res.answers_json) || []
      showDetail.value = true
    }
  } catch (err) {
    toast.error(t.value.learning?.quizDetailLoadFailed || t.value.common.error)
  }
}

function getAnswerDetail(questionId: string) {
  return detailedAnswers.value.find((a: any) => a.question_id === questionId || a.questionUlid === questionId)
}

function goBackToLearning() {
  if (returnTo.value) {
    router.push(returnTo.value)
    return
  }
  router.back()
}

onMounted(() => {
  window.addEventListener("beforeunload", handleBeforeUnload)
  void loadPaper()
})

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", handleBeforeUnload)
  stopTimer()
})

onBeforeRouteLeave(() => {
  if (!shouldConfirmLeavingQuiz()) return true
  return window.confirm(quizLeaveConfirmMessage())
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <FileText class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.learning?.quizPrefix }}</span>
      </header>

      <main class="quiz-page px-5 py-8 md:px-8 lg:px-10">
    <div v-if="loading" class="quiz-state flex min-h-[60vh] items-center justify-center">
      <div class="flex items-center gap-3 rounded-2xl bg-white px-5 py-4 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <Loader2 class="h-5 w-5 animate-spin text-primary" />
        <span>{{ t.common.loading }}</span>
      </div>
    </div>

    <div v-else-if="!paper" class="quiz-state flex min-h-[60vh] flex-col items-center justify-center gap-4">
      <AlertCircle class="h-12 w-12 text-destructive" />
      <h2 class="text-lg font-semibold text-foreground">{{ t.learning?.quizNotFound }}</h2>
      <button class="btn btn-primary cursor-pointer" @click="goBackToLearning"><ChevronLeft class="h-4 w-4" /> {{ t.common.back }}</button>
    </div>

    <div v-else-if="result" data-testid="quiz-result" :data-passed="quizPassed ? 'true' : 'false'" class="quiz-result mx-auto max-w-2xl py-12">
      <div class="quiz-result-card card border-t-4 border-t-primary p-8 text-center shadow-lg">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
          <CheckCircle2 class="h-8 w-8 text-primary" />
        </div>
        <h1 class="text-2xl font-bold">{{ t.learning?.quizCompleted }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.learning?.quizSubmittedDesc }}</p>
        <div class="quiz-score-card my-6 rounded-lg border bg-card p-6 shadow-sm">
          <span class="text-sm font-medium text-muted-foreground">{{ t.learning?.quizScore }}</span>
          <div class="mt-2 flex items-baseline justify-center gap-2">
            <span class="text-4xl font-bold text-foreground">{{ result.score || 0 }}</span>
            <span class="text-xl text-muted-foreground">/ {{ result.max_score || 0 }}</span>
          </div>
          <div :class="['mx-auto mt-4 w-fit rounded-full px-4 py-1 text-sm font-semibold', quizPassed ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700']">
            {{ quizPassed ? t.learning?.quizPassed : t.learning?.quizFailed }}
          </div>
        </div>
        <div class="flex flex-col gap-3 justify-center sm:flex-row">
          <button data-testid="quiz-return" class="btn btn-outline cursor-pointer px-6" @click="goBackToLearning"><ChevronLeft class="h-4 w-4" /> {{ t.learning?.quizReturn }}</button>
          <button v-if="!showDetail" data-testid="quiz-detail" class="btn btn-primary cursor-pointer px-6" @click="loadAttemptDetail">{{ t.learning?.quizDetailButton }}</button>
        </div>
      </div>

      <!-- Detailed grading result block -->
      <div v-if="showDetail" class="quiz-detail-list mt-8 space-y-6 text-left">
        <h2 class="text-xl font-bold">{{ t.learning?.quizDetailTitle }}</h2>
        <div v-for="(question, index) in questions" :key="question.question_id" class="quiz-detail-card overflow-hidden rounded-md bg-white shadow-sm border border-border">
          <div class="quiz-card-header flex items-center justify-between border-b bg-muted/30 px-6 py-3 text-sm font-medium text-muted-foreground">
            <span>{{ formatQuizQuestionCount(Number(index) + 1, questions.length) }}</span>
            <span class="rounded border bg-background px-2 py-0.5 text-xs">{{ question.points || 0 }} {{ t.learning?.quizPts }}</span>
          </div>
          <div class="quiz-card-body p-6">
            <h3 class="quiz-question-title mb-6 text-lg font-medium leading-relaxed text-foreground">{{ question.question_text }}</h3>
            <div class="space-y-3">
              <div
                v-for="option in question.options || []"
                :key="option.option_id"
                data-testid="quiz-detail-option"
                :data-question-id="question.question_id"
                :data-option-id="option.option_id"
                :data-correct="getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id) ? 'true' : 'false'"
                :class="[
                  'quiz-option flex w-full items-start gap-3 rounded-md border p-4 text-left',
                  getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id) ? 'border-emerald-200 bg-emerald-50' : 
                  (getAnswerDetail(question.question_id)?.selected_option_ids?.includes(option.option_id) ? 'border-rose-200 bg-rose-50' : 'border-border bg-slate-50 opacity-60')
                ]"
              >
                <div :class="[
                  'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center border',
                  question.question_type === 2 ? 'rounded-md' : 'rounded-full',
                  getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id) ? 'border-emerald-600 bg-emerald-600' :
                  (getAnswerDetail(question.question_id)?.selected_option_ids?.includes(option.option_id) ? 'border-rose-600 bg-rose-600' : 'border-muted-foreground/30 bg-background')
                ]">
                  <span v-if="getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id)" class="h-2.5 w-2.5 rounded-full bg-white" />
                  <span v-else-if="getAnswerDetail(question.question_id)?.selected_option_ids?.includes(option.option_id)" class="h-2.5 w-2.5 rounded-full bg-white" />
                </div>
                <div class="flex flex-col">
                  <span :class="['text-sm', getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id) ? 'font-medium text-emerald-800' : (getAnswerDetail(question.question_id)?.selected_option_ids?.includes(option.option_id) ? 'font-medium text-rose-800' : 'text-muted-foreground')]">
                    {{ option.option_text }}
                  </span>
                  <span v-if="getAnswerDetail(question.question_id)?.correct_option_ids?.includes(option.option_id)" class="text-xs text-emerald-600 mt-1">{{ t.learning?.quizCorrectAnswer }}</span>
                  <span v-else-if="getAnswerDetail(question.question_id)?.selected_option_ids?.includes(option.option_id)" class="text-xs text-rose-600 mt-1">{{ t.learning?.quizYourWrongChoice }}</span>
                </div>
              </div>
            </div>
            <div v-if="question.explanation || getAnswerDetail(question.question_id)?.explanation" class="quiz-explanation mt-6 rounded-md bg-blue-50 p-4 border border-blue-100">
              <div class="flex items-center gap-2 text-blue-800 font-semibold mb-2">
                <AlertCircle class="h-4 w-4" />
                <span>{{ t.learning?.quizExplanation }}</span>
              </div>
              <p class="text-sm text-blue-900 leading-relaxed whitespace-pre-wrap">{{ question.explanation || getAnswerDetail(question.question_id)?.explanation }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="quiz-session mx-auto max-w-3xl space-y-8">
      <div class="quiz-session-header rounded-md bg-white p-6 sm:p-8">
        <button class="quiz-back btn btn-ghost -ml-2 mb-4 cursor-pointer text-muted-foreground" @click="goBackToLearning"><ChevronLeft class="h-4 w-4" /> {{ t.learning?.quizReturn }}</button>
        <div class="quiz-title-row flex items-start justify-between gap-4">
          <div>
            <h1 class="text-2xl font-bold text-foreground sm:text-3xl">{{ paper.title || t.learning?.quizPrefix }}</h1>
            <p v-if="paper.description" class="mt-2 text-muted-foreground">{{ paper.description }}</p>
          </div>
          <div v-if="formattedTimer" class="flex shrink-0 items-center gap-2 rounded-full border border-rose-200 bg-rose-50 px-3 py-1.5 text-sm font-bold text-rose-700 shadow-sm animate-pulse">
            <Clock class="h-4 w-4" /> {{ formattedTimer }}
          </div>
          <div v-else-if="paper.time_limit > 0" class="flex shrink-0 items-center gap-2 rounded-full border bg-muted/30 px-3 py-1.5 text-sm font-medium text-foreground">
            <Clock class="h-4 w-4 text-primary" /> {{ paper.time_limit }} {{ t.learning?.quizMin }}
          </div>
        </div>
      </div>

      <div class="quiz-question-list space-y-6">
        <div v-for="(question, index) in questions" :key="question.question_id" class="quiz-question-card overflow-hidden rounded-md bg-white">
          <div class="quiz-card-header flex items-center justify-between border-b bg-muted/30 px-6 py-3 text-sm font-medium text-muted-foreground">
            <span>{{ formatQuizQuestionCount(Number(index) + 1, questions.length) }}</span>
            <span class="rounded border bg-background px-2 py-0.5 text-xs">{{ question.points || 0 }} {{ t.learning?.quizPts }}</span>
          </div>
          <div class="quiz-card-body p-6">
            <h3 class="quiz-question-title mb-6 text-lg font-medium leading-relaxed text-foreground">{{ question.question_text }}</h3>
            <div class="space-y-3">
              <button
                v-for="option in question.options || []"
                :key="option.option_id"
                data-testid="quiz-option"
                :data-question-id="question.question_id"
                :data-option-id="option.option_id"
                :class="[
                  'quiz-option flex w-full cursor-pointer items-start gap-3 rounded-md border p-4 text-left transition-colors',
                  (answers[question.question_id] || []).includes(option.option_id)
                    ? 'border-slate-200 bg-slate-50'
                    : 'border-border hover:bg-slate-50',
                ]"
                @click="handleSelectOption(question.question_id, option.option_id, question.question_type === 2)"
              >
                <div :class="[
                  'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center border',
                  question.question_type === 2 ? 'rounded-md' : 'rounded-full',
                  (answers[question.question_id] || []).includes(option.option_id) ? 'border-primary bg-white' : 'border-muted-foreground/30 bg-background',
                ]">
                  <span v-if="(answers[question.question_id] || []).includes(option.option_id)" class="h-2.5 w-2.5 rounded-full bg-primary" />
                </div>
                <span :class="['text-sm', (answers[question.question_id] || []).includes(option.option_id) ? 'font-medium text-foreground' : 'text-muted-foreground']">{{ option.option_text }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="quiz-submit-bar sticky bottom-4 flex flex-col items-center justify-between gap-4 rounded-md bg-white/95 p-4 shadow-sm backdrop-blur-md sm:flex-row sm:p-6">
        <div class="text-sm text-muted-foreground">{{ formatQuizAnsweredCount(answeredCount, questions.length) }}</div>
        <button type="button" data-testid="quiz-submit" class="btn btn-primary w-full cursor-pointer px-8 disabled:cursor-not-allowed sm:w-auto" :disabled="submitting || !allAnswered" @click="openSubmitConfirm">
          <span v-if="submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-r-transparent" />
          <FileText v-else class="h-4 w-4" />
          {{ submitting ? t.common.loading : t.learning?.quizSubmit }}
        </button>
      </div>
    </div>
      </main>
    </div>

    <div v-if="submitConfirmOpen" class="quiz-confirm-backdrop fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div
        ref="submitConfirmDialogRef"
        class="quiz-confirm-dialog w-full max-w-md overflow-hidden rounded-[16px] bg-white shadow-[0_24px_60px_rgba(15,23,42,0.24)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="quiz-submit-confirm-title"
        aria-describedby="quiz-submit-confirm-description"
        tabindex="-1"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <FileText class="h-5 w-5" />
            </div>
            <h2 id="quiz-submit-confirm-title" class="text-lg font-semibold text-slate-950">
              {{ t.learning?.quizSubmit }}
            </h2>
          </div>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition-colors hover:border-primary/30 hover:bg-primary/5 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
            :aria-label="t.common.close"
            :title="t.common.close"
            @click="closeSubmitConfirm"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="px-5 py-5">
          <p id="quiz-submit-confirm-description" class="text-sm leading-6 text-slate-600">
            {{ quizSubmitConfirmMessage() }}
          </p>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 bg-slate-50 px-5 py-4">
          <button type="button" class="btn btn-outline min-w-20 rounded-lg" @click="closeSubmitConfirm">
            {{ t.common.cancel }}
          </button>
          <button type="button" data-testid="quiz-submit-confirm" class="btn btn-primary min-w-24 rounded-lg shadow-sm shadow-primary/20" @click="confirmSubmitQuiz">
            {{ t.learning?.quizSubmit }}
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .quiz-state {
    min-height: 42vh;
    padding: 24px 0;
    text-align: center;
  }

  .quiz-result {
    padding: 4px 0 20px;
  }

  .quiz-result-card {
    padding: 16px;
  }

  .quiz-result-card h1,
  .quiz-session-header h1 {
    font-size: 22px;
    line-height: 30px;
  }

  .quiz-score-card {
    margin: 16px 0;
    padding: 16px;
  }

  .quiz-detail-list {
    display: flex;
    flex-direction: column;
    margin-top: 16px;
    gap: 12px;
  }

  .quiz-detail-list > :not([hidden]) ~ :not([hidden]),
  .quiz-session > :not([hidden]) ~ :not([hidden]),
  .quiz-question-list > :not([hidden]) ~ :not([hidden]) {
    margin-top: 0;
  }

  .quiz-detail-list > h2 {
    font-size: 18px;
    line-height: 26px;
  }

  .quiz-session-header,
  .quiz-submit-bar {
    border-radius: 12px;
  }

  .quiz-detail-card,
  .quiz-question-card {
    border-radius: 8px;
  }

  .quiz-card-header {
    padding: 10px 12px;
  }

  .quiz-card-body {
    padding: 12px;
  }

  .quiz-question-title {
    margin-bottom: 12px;
    font-size: 16px;
    line-height: 24px;
  }

  .quiz-card-body > .space-y-3 {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .quiz-card-body > .space-y-3 > :not([hidden]) ~ :not([hidden]) {
    margin-top: 0;
  }

  .quiz-option {
    gap: 10px;
    padding: 12px;
  }

  .quiz-explanation {
    margin-top: 12px;
    padding: 12px;
  }

  .quiz-session {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .quiz-session-header {
    padding: 16px;
  }

  .quiz-back {
    margin-bottom: 12px;
  }

  .quiz-title-row {
    flex-direction: column;
    gap: 12px;
  }

  .quiz-question-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .quiz-submit-bar {
    bottom: 8px;
    gap: 10px;
    padding: 12px;
  }

  .quiz-submit-bar .btn {
    min-height: 42px;
  }

  .quiz-confirm-backdrop {
    padding: 12px;
  }

  .quiz-confirm-dialog {
    border-radius: 12px;
  }

  .quiz-confirm-dialog > div {
    padding-right: 16px;
    padding-left: 16px;
  }
}
</style>
