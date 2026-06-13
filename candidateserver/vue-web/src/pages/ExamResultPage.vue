<script setup lang="ts">
import { onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowLeft, Award, CheckCircle2, ClipboardList, ExternalLink, Loader2 } from "lucide-vue-next"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const { t } = useTranslation()
const examId = decodeURIComponent(String(route.query.examId || ""))
const loading = ref(true)
const result = ref<any>(null)

onMounted(async () => {
  if (!examId) {
    loading.value = false
    return
  }
  try {
    result.value = await apiClient(`/api/exams/${encodeURIComponent(examId)}/result`)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell>
    <RouterLink to="/exams" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" />
      {{ t.examsPage.backToExams }}
    </RouterLink>

    <div v-if="loading" class="flex items-center gap-2 text-muted-foreground">
      <Loader2 class="h-4 w-4 animate-spin" /> {{ t.common.loading }}
    </div>
    <div v-else-if="!examId" class="rounded-lg border bg-card p-8 text-center text-muted-foreground">{{ t.examsPage.selectExamFirst }}</div>
    <div v-else-if="!result || result.has_result === false" class="rounded-lg border bg-card p-8 text-center text-muted-foreground">{{ t.examsPage.noScoreDetails }}</div>
    <div v-else class="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <section class="rounded-2xl border bg-card p-6 shadow-sm">
        <div class="mb-4 flex flex-wrap items-center gap-2">
          <span :class="['badge border', statusBadgeClassForStatusValue(result.is_passed ? 'SUCCESS' : 'FAILED')]">
            <Award class="mr-1 h-3 w-3" />
            {{ result.is_passed ? t.examsPage.statusPassed : t.examsPage.statusFailed }}
          </span>
          <span class="badge">{{ result.exam_id || examId }}</span>
        </div>
        <h1 class="text-3xl font-bold text-foreground">{{ t.examsPage.resultTitle }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.examsPage.resultDesc }}</p>
        <div class="mt-6 grid gap-4 sm:grid-cols-2">
          <div class="rounded-xl border bg-background p-4">
            <div class="text-xs text-muted-foreground">{{ t.examsPage.score }}</div>
            <div class="mt-1 text-2xl font-bold text-foreground">{{ typeof result.total_score === 'number' ? result.total_score.toFixed(2) : t.common.unknown }}</div>
          </div>
          <div class="rounded-xl border bg-background p-4">
            <div class="text-xs text-muted-foreground">{{ t.examsPage.passStatus }}</div>
            <div class="mt-1 inline-flex items-center gap-2 text-lg font-semibold">
              <CheckCircle2 :class="['h-5 w-5', result.is_passed ? 'text-blue-600' : 'text-yellow-600']" />
              {{ result.is_passed ? t.examsPage.statusPassed : t.examsPage.statusFailed }}
            </div>
          </div>
        </div>
        <div class="mt-6 flex flex-wrap gap-2">
          <RouterLink to="/exams" class="btn btn-primary"><ClipboardList class="h-4 w-4" /> {{ t.examsPage.backToExams }}</RouterLink>
          <RouterLink to="/certificates" class="btn btn-outline">{{ t.examsPage.viewCertificate }} <ExternalLink class="h-4 w-4" /></RouterLink>
        </div>
      </section>
      <section class="rounded-2xl border bg-card p-6 shadow-sm">
        <h2 class="mb-4 text-lg font-semibold text-foreground">{{ t.examsPage.scoreDetails }}</h2>
        <pre v-if="result.score_details_raw" class="max-h-[560px] overflow-auto whitespace-pre-wrap rounded-xl border bg-muted/30 p-4 text-xs leading-6 text-muted-foreground">{{ result.score_details_raw }}</pre>
        <div v-else class="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">{{ t.examsPage.noScoreDetails }}</div>
      </section>
    </div>
  </AppShell>
</template>
