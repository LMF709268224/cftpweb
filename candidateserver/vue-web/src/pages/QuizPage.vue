<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, CheckCircle2, ChevronLeft, Clock, FileText } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const attemptId = computed(() => String(route.query.attemptId || ""))
const loading = ref(true)
const submitting = ref(false)
const paper = ref<any>(null)
const answers = ref<Record<string, string[]>>({})
const result = ref<any>(null)

const questions = computed(() => paper.value?.questions || [])
const allAnswered = computed(() => questions.value.every((q: any) => (answers.value[q.question_id]?.length || 0) > 0))

function formatQuizQuestionCount(current: number, total: number) {
  return (t.value.learning?.quizQuestionCount || "")
    .replace("{{current}}", String(current))
    .replace("{{total}}", String(total))
}

function formatQuizAnsweredCount(current: number, total: number) {
  return (t.value.learning?.quizAnsweredCount || "")
    .replace("{{current}}", String(current))
    .replace("{{total}}", String(total))
}

async function loadPaper() {
  if (!attemptId.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    paper.value = await apiClient(`/api/quizzes/attempts/${attemptId.value}/paper`)
  } finally {
    loading.value = false
  }
}

function handleSelectOption(questionId: string, optionId: string, isMultipleChoice: boolean) {
  const current = answers.value[questionId] || []
  if (!isMultipleChoice) {
    answers.value = { ...answers.value, [questionId]: [optionId] }
    return
  }
  answers.value = {
    ...answers.value,
    [questionId]: current.includes(optionId) ? current.filter((id) => id !== optionId) : [...current, optionId],
  }
}

async function submitQuiz() {
  if (!paper.value?.questions) return
  const submissions = Object.entries(answers.value).map(([questionId, selectedOptionIds]) => ({ question_id: questionId, selected_option_ids: selectedOptionIds }))
  submitting.value = true
  try {
    result.value = await apiClient(`/api/quizzes/attempts/${attemptId.value}/submit`, { method: "POST", body: JSON.stringify({ submissions }) })
    toast.success(t.value.learning?.quizSubmittedDesc || t.value.common.success)
  } finally {
    submitting.value = false
  }
}

onMounted(loadPaper)
</script>

<template>
  <AppShell>
    <div v-if="loading" class="flex min-h-[60vh] items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-r-transparent" />
    </div>

    <div v-else-if="!paper" class="flex min-h-[60vh] flex-col items-center justify-center gap-4">
      <AlertCircle class="h-12 w-12 text-destructive" />
      <h2 class="text-lg font-semibold text-foreground">{{ t.learning?.quizNotFound }}</h2>
      <button class="btn btn-primary" @click="router.back()"><ChevronLeft class="h-4 w-4" /> {{ t.common.back }}</button>
    </div>

    <div v-else-if="result" class="mx-auto max-w-2xl py-12">
      <div class="card border-t-4 border-t-primary p-8 text-center shadow-lg">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
          <CheckCircle2 class="h-8 w-8 text-primary" />
        </div>
        <h1 class="text-2xl font-bold">{{ t.learning?.quizCompleted }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.learning?.quizSubmittedDesc }}</p>
        <div class="my-6 rounded-xl border bg-card p-6 shadow-sm">
          <span class="text-sm font-medium text-muted-foreground">{{ t.learning?.quizScore }}</span>
          <div class="mt-2 flex items-baseline justify-center gap-2">
            <span class="text-4xl font-bold text-foreground">{{ result.score || 0 }}</span>
            <span class="text-xl text-muted-foreground">/ {{ result.max_score || 0 }}</span>
          </div>
          <div :class="['mx-auto mt-4 w-fit rounded-full px-4 py-1 text-sm font-semibold', result.is_passed ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700']">
            {{ result.is_passed ? t.learning?.quizPassed : t.learning?.quizFailed }}
          </div>
        </div>
        <button class="btn btn-primary px-8" @click="router.back()"><ChevronLeft class="h-4 w-4" /> {{ t.learning?.quizReturn }}</button>
      </div>
    </div>

    <div v-else class="mx-auto max-w-3xl space-y-8">
      <div class="rounded-2xl border bg-card p-6 shadow-sm sm:p-8">
        <button class="btn btn-ghost -ml-2 mb-4 text-muted-foreground" @click="router.back()"><ChevronLeft class="h-4 w-4" /> {{ t.learning?.quizReturn }}</button>
        <div class="flex items-start justify-between gap-4">
          <div>
            <h1 class="text-2xl font-bold text-foreground sm:text-3xl">{{ paper.title || t.learning?.quizPrefix }}</h1>
            <p v-if="paper.description" class="mt-2 text-muted-foreground">{{ paper.description }}</p>
          </div>
          <div v-if="paper.time_limit > 0" class="flex shrink-0 items-center gap-2 rounded-full border bg-muted/30 px-3 py-1.5 text-sm font-medium text-foreground">
            <Clock class="h-4 w-4 text-primary" /> {{ paper.time_limit }} {{ t.learning?.quizMin }}
          </div>
        </div>
      </div>

      <div class="space-y-6">
        <div v-for="(question, index) in questions" :key="question.question_id" class="overflow-hidden rounded-2xl border border-border/50 bg-card shadow-sm transition-all hover:border-primary/20 hover:shadow-md">
          <div class="flex items-center justify-between border-b bg-muted/30 px-6 py-3 text-sm font-medium text-muted-foreground">
            <span>{{ formatQuizQuestionCount(Number(index) + 1, questions.length) }}</span>
            <span class="rounded border bg-background px-2 py-0.5 text-xs">{{ question.points || 0 }} {{ t.learning?.quizPts }}</span>
          </div>
          <div class="p-6">
            <h3 class="mb-6 text-lg font-medium leading-relaxed text-foreground">{{ question.question_text }}</h3>
            <div class="space-y-3">
              <button
                v-for="option in question.options || []"
                :key="option.option_id"
                :class="[
                  'flex w-full items-start gap-3 rounded-xl border p-4 text-left transition-all',
                  (answers[question.question_id] || []).includes(option.option_id)
                    ? 'border-primary bg-primary/5 ring-1 ring-primary/20'
                    : 'border-border hover:border-border/80 hover:bg-muted/50',
                ]"
                @click="handleSelectOption(question.question_id, option.option_id, question.question_type === 2)"
              >
                <div :class="[
                  'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center border',
                  question.question_type === 2 ? 'rounded-md' : 'rounded-full',
                  (answers[question.question_id] || []).includes(option.option_id) ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/30 bg-background',
                ]">
                  <CheckCircle2 v-if="(answers[question.question_id] || []).includes(option.option_id)" class="h-3.5 w-3.5" />
                </div>
                <span :class="['text-sm', (answers[question.question_id] || []).includes(option.option_id) ? 'font-medium text-foreground' : 'text-muted-foreground']">{{ option.option_text }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="sticky bottom-4 flex flex-col items-center justify-between gap-4 rounded-2xl border bg-card/80 p-4 shadow-lg backdrop-blur-md sm:flex-row sm:p-6">
        <div class="text-sm text-muted-foreground">{{ formatQuizAnsweredCount(Object.keys(answers).length, questions.length) }}</div>
        <button class="btn btn-primary w-full px-8 sm:w-auto" :disabled="submitting || !allAnswered" @click="submitQuiz">
          <span v-if="submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-r-transparent" />
          <FileText v-else class="h-4 w-4" />
          {{ submitting ? t.common.loading : t.learning?.quizSubmit }}
        </button>
      </div>
    </div>
  </AppShell>
</template>
