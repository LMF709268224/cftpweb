<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, CalendarClock, CheckCircle2, ClipboardList, ExternalLink, Filter, History, Loader2, Search, ShieldCheck } from "lucide-vue-next"
import { EXAM_STATUS_LABELS, normalizeEnumValueUpper, statusBadgeClassForStatusValue, statusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type TabId = "current" | "history" | "exemption" | "records"

const { t } = useTranslation()
const route = useRoute()
const activeTab = ref<TabId>("current")
const loading = ref(false)
const scheduleLoadingExamId = ref<string | null>(null)
const search = ref("")
const exams = ref<any[]>([])
const total = ref(0)

const tabs = computed(() => [
  { id: "current" as TabId, icon: CalendarClock, label: t.value.examsPage.currentTab },
  { id: "history" as TabId, icon: History, label: t.value.examsPage.historyTab },
  { id: "exemption" as TabId, icon: ShieldCheck, label: t.value.examsPage.exemptionTab },
  { id: "records" as TabId, icon: ClipboardList, label: t.value.examsPage.recordsTab },
])

const emptyCopy = computed(() => ({
  current: { title: t.value.examsPage.noExams, description: t.value.examsPage.noExamsDesc, icon: AlertCircle },
  history: { title: t.value.examsPage.noHistory, description: t.value.examsPage.noHistoryDesc, icon: History },
  exemption: { title: t.value.examsPage.noExemption, description: t.value.examsPage.noExemptionDesc, icon: ShieldCheck },
  records: { title: t.value.examsPage.noRecords, description: t.value.examsPage.noRecordsDesc, icon: ClipboardList },
}))

const filtered = computed(() => exams.value.filter((exam) => [
  exam.exam_id,
  exam.program_code,
  exam.exam_code,
  exam.exam_status,
  exam.result_status,
  exam.confirmation_number,
  exam.site_name,
].filter(Boolean).join(" ").toLowerCase().includes(search.value.toLowerCase())))

function normalizedExamStatus(status?: string | number | null) {
  return normalizeEnumValueUpper(status)
}
function shouldShowExamStatus(status?: string | number | null) {
  const normalized = normalizedExamStatus(status)
  return Boolean(normalized && !["NONE", "UNKNOWN", "UNSPECIFIED"].some((item) => normalized.includes(item)))
}
function hasExamResult(exam: any) {
  const normalized = normalizedExamStatus(exam.result_status)
  return typeof exam.total_score === "number" || exam.is_passed === true || ["DONE", "PASSED", "FAILED", "RESULT_STATUS_PASSED", "RESULT_STATUS_FAILED"].includes(normalized)
}
function hasText(value?: string | null) {
  return Boolean(value?.trim())
}
function hasAppointmentDetails(exam: any) {
  return hasText(exam.confirmation_number) || hasText(exam.site_name) || hasText(exam.appointment_start_time) || hasText(exam.appointment_end_time)
}
function canScheduleExam(exam: any) {
  const status = normalizedExamStatus(exam.exam_status)
  return Boolean(exam.exam_id && status && status.includes("OPEN"))
}

function noResultLabel() {
  return (t.value.examsPage as any).statusNoResult || t.value.examsPage.statusPending
}

async function loadExams() {
  if (activeTab.value === "exemption" || activeTab.value === "records") {
    exams.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set("page", "1")
    params.set("page_size", "50")
    if (activeTab.value === "history") params.set("result_status", "DONE")
    if (search.value.trim()) params.set("confirmation_number", search.value.trim())
    const res = await apiClient(`/api/exams?${params.toString()}`)
    exams.value = res?.exams || []
    total.value = res?.total || 0
  } catch {
    exams.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function handleScheduleExam(exam: any) {
  if (!exam.exam_id || scheduleLoadingExamId.value) return
  scheduleLoadingExamId.value = exam.exam_id
  try {
    const termUrlBase = window.location.origin + `/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-callback`
    const params = new URLSearchParams({ url_type: "schd", term_url_base: termUrlBase })
    const res = await apiClient(`/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-url?${params.toString()}`)
    if (res?.url) {
      toast.info(t.value.examsPage.scheduleRedirecting)
      window.open(res.url, "_blank", "noopener,noreferrer")
    } else {
      toast.error(t.value.examsPage.scheduleURLMissing)
    }
  } catch {
    toast.error(t.value.examsPage.scheduleFailed)
  } finally {
    scheduleLoadingExamId.value = null
  }
}

watch(activeTab, loadExams)
onMounted(() => {
  if (route.query.schedule_return === "1") toast.success(t.value.examsPage.scheduleReturnToast)
  void loadExams()
})
</script>

<template>
  <AppShell content-class="px-4 py-4">
    <div class="mb-4 overflow-hidden rounded-3xl bg-card shadow-sm ring-1 ring-border/50">
      <div class="flex flex-col gap-4 bg-[#eef8fa] p-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-white px-3 py-1 text-xs font-medium text-primary">
            <CalendarClock class="h-3.5 w-3.5" />
            {{ t.sidebar.exams }}
          </div>
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.examsPage.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.examsPage.subtitle }}</p>
        </div>
        <RouterLink to="/courses" class="btn btn-outline rounded-xl bg-white/80 shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary">{{ t.courses.browseCoursesBtn }} <ExternalLink class="h-4 w-4" /></RouterLink>
      </div>
    </div>

    <div class="mb-4 flex flex-col gap-4 rounded-2xl bg-card p-4 shadow-sm ring-1 ring-border/50 sm:flex-row sm:items-center sm:justify-between">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="search" class="input pl-10" :placeholder="t.examsPage.searchPlaceholder" />
      </div>
      <button class="btn btn-outline rounded-xl" @click="loadExams"><Filter class="h-4 w-4" /> {{ t.examsPage.refresh }}</button>
    </div>

    <div class="mb-4 flex w-fit gap-1 overflow-x-auto rounded-2xl bg-card p-1 shadow-sm ring-1 ring-border/50">
      <button v-for="tab in tabs" :key="tab.id" :class="['inline-flex items-center gap-2 whitespace-nowrap rounded-xl px-4 py-2 text-sm font-medium transition-all duration-200', activeTab === tab.id ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20' : 'text-muted-foreground hover:bg-primary/10 hover:text-primary']" @click="activeTab = tab.id">
        <component :is="tab.icon" class="h-4 w-4" /> {{ tab.label }}
      </button>
    </div>

    <div class="rounded-2xl bg-card p-4 shadow-sm ring-1 ring-border/50">
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="filtered.length === 0" class="flex flex-col items-center justify-center py-12 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10">
          <component :is="emptyCopy[activeTab].icon" class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ emptyCopy[activeTab].title }}</h3>
        <p class="max-w-md text-muted-foreground">{{ emptyCopy[activeTab].description }}</p>
      </div>
      <div v-else class="space-y-4">
        <div class="flex items-center justify-between text-sm text-muted-foreground">
          <span>{{ t.examsPage.countPrefix }} {{ total > 0 ? total : filtered.length }} {{ t.examsPage.countSuffix }}</span>
          <span>{{ activeTab === 'history' ? t.examsPage.historyFilterHint : t.examsPage.visibleRecordsHint }}</span>
        </div>
        <div class="grid gap-4">
          <div v-for="exam in filtered" :key="exam.exam_id" class="relative overflow-hidden rounded-2xl border border-border bg-white p-4 shadow-sm transition-all hover:border-primary/25 hover:shadow-md hover:shadow-primary/10">
            <div class="absolute left-0 top-0 h-full w-1 bg-primary" />
            <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span v-if="shouldShowExamStatus(exam.exam_status)" :class="['badge', statusBadgeClassForStatusValue(exam.exam_status)]">{{ statusLabel(t, EXAM_STATUS_LABELS, normalizedExamStatus(exam.exam_status)) }}</span>
                  <span v-if="shouldShowExamStatus(exam.result_status)" :class="['badge', statusBadgeClassForStatusValue(exam.result_status)]">{{ statusLabel(t, EXAM_STATUS_LABELS, normalizedExamStatus(exam.result_status)) }}</span>
                  <span v-if="hasExamResult(exam) && exam.is_passed" :class="['badge gap-1', statusBadgeClassForStatusValue('SUCCESS')]"><CheckCircle2 class="h-3 w-3" /> {{ t.examsPage.statusPassed }}</span>
                  <span v-else-if="!hasExamResult(exam)" :class="['badge', statusBadgeClassForStatusValue('PENDING')]">{{ noResultLabel() }}</span>
                  <span v-else :class="['badge', statusBadgeClassForStatusValue('FAILED')]">{{ t.examsPage.statusFailed }}</span>
                </div>
                <h3 class="text-lg font-semibold text-foreground">{{ exam.exam_code || exam.program_code || exam.exam_id || t.common.unknown }}</h3>
                <div class="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
                  <div v-if="hasText(exam.confirmation_number)"><span class="font-medium text-foreground">{{ t.examsPage.confirmationNumber }}:</span> {{ exam.confirmation_number }}</div>
                  <div v-if="hasText(exam.site_name)"><span class="font-medium text-foreground">{{ t.examsPage.site }}:</span> {{ exam.site_name }}</div>
                  <div v-if="hasText(exam.appointment_start_time)"><span class="font-medium text-foreground">{{ t.examsPage.appointmentStart }}:</span> {{ formatBackendDate(exam.appointment_start_time) }}</div>
                  <div v-if="hasText(exam.appointment_end_time)"><span class="font-medium text-foreground">{{ t.examsPage.appointmentEnd }}:</span> {{ formatBackendDate(exam.appointment_end_time) }}</div>
                  <div v-if="!hasAppointmentDetails(exam) && !hasExamResult(exam)" class="rounded-xl border border-blue-200 bg-blue-50 px-3 py-2 text-blue-700 sm:col-span-2">
                    <div class="flex items-start gap-2">
                      <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                      <div>
                        <div class="font-medium text-blue-800">{{ t.examsPage.notScheduledTitle }}</div>
                        <div class="mt-1 text-xs">{{ t.examsPage.notScheduledDesc }}</div>
                      </div>
                    </div>
                  </div>
                  <div><span class="font-medium text-foreground">{{ t.examsPage.candidate }}:</span> {{ [exam.candidate_first_name, exam.candidate_last_name].filter(Boolean).join(" ") || exam.candidate_email || t.common.unknown }}</div>
                  <div v-if="hasExamResult(exam)"><span class="font-medium text-foreground">{{ t.examsPage.score }}:</span> {{ typeof exam.total_score === 'number' ? exam.total_score.toFixed(2) : t.common.unknown }}</div>
                </div>
              </div>
              <div class="flex flex-wrap gap-2">
                <button v-if="canScheduleExam(exam)" class="btn btn-primary rounded-xl shadow-sm shadow-primary/20" :disabled="scheduleLoadingExamId === exam.exam_id" @click="handleScheduleExam(exam)">
                  <Loader2 v-if="scheduleLoadingExamId === exam.exam_id" class="h-4 w-4 animate-spin" />
                  <ExternalLink v-else class="h-4 w-4" />
                  {{ t.learning.actionScheduleExam }}
                </button>
                <RouterLink v-if="hasExamResult(exam)" :to="`/exams/result?examId=${encodeURIComponent(exam.exam_id)}`" class="btn btn-outline rounded-xl">{{ t.examsPage.viewResult }}</RouterLink>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
