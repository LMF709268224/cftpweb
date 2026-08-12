<script setup lang="ts">
import { onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowLeft, CheckCircle2, ClipboardList, ExternalLink } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const { t } = useTranslation()
const examId = decodeURIComponent(String(route.query.examId || ""))
const loading = ref(true)
const loadError = ref(false)
const result = ref<any>(null)

async function loadResult() {
  if (!examId) {
    loadError.value = false
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = false
  try {
    result.value = await apiClient(`/api/exams/${encodeURIComponent(examId)}/result`)
  } catch (error) {
    console.error(error)
    loadError.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadResult()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <ClipboardList class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.examsPage.resultTitle }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <RouterLink to="/exams" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" />
          {{ t.examsPage.backToExams }}
        </RouterLink>

    <PageFeedback v-if="loading" kind="loading" :loading-label="t.common.loading" />
    <PageFeedback
      v-else-if="loadError"
      kind="error"
      :title="t.examsPage.resultLoadFailed"
      :description="t.examsPage.resultLoadFailedDesc"
      :action-label="t.examsPage.retry"
      @action="loadResult"
    />
    <PageFeedback v-else-if="!examId" kind="empty" :title="t.examsPage.selectExamFirst" />
    <PageFeedback v-else-if="!result || result.has_result === false" kind="empty" :title="t.examsPage.noScoreDetails" />
    <div v-else>
      <section class="rounded-xl bg-white p-4 shadow-none md:rounded-[16px] md:p-6 md:shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
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
          <RouterLink v-if="result.is_passed" to="/certificates" class="btn btn-outline">{{ t.examsPage.viewCertificate }} <ExternalLink class="h-4 w-4" /></RouterLink>
        </div>
      </section>
    </div>
      </main>
    </div>
  </AppShell>
</template>
