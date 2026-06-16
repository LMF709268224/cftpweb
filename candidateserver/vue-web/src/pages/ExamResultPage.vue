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
  <AppShell content-class="p-4">
    <RouterLink to="/exams" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" />
      {{ t.examsPage.backToExams }}
    </RouterLink>

    <div v-if="loading" class="flex items-center gap-2 text-muted-foreground">
      <Loader2 class="h-4 w-4 animate-spin" /> {{ t.common.loading }}
    </div>
    <div v-else-if="!examId" class="rounded-[16px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">{{ t.examsPage.selectExamFirst }}</div>
    <div v-else-if="!result || result.has_result === false" class="rounded-[16px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">{{ t.examsPage.noScoreDetails }}</div>
    <div v-else>
      <section class="rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
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
          <div class="rounded-lg bg-background p-4 shadow-sm">
            <div class="text-xs text-muted-foreground">{{ t.examsPage.score }}</div>
            <div class="mt-1 text-2xl font-bold text-foreground">{{ typeof result.total_score === 'number' ? result.total_score.toFixed(2) : t.common.unknown }}</div>
          </div>
          <div class="rounded-lg bg-background p-4 shadow-sm">
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
    </div>
  </AppShell>
</template>
