<script setup lang="ts">
import { Loader2, X } from "lucide-vue-next"
import { computed, onMounted, onUnmounted, ref } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"

type Enrollment = {
  enrollment_id: string
  course_ulid?: string
  status?: string
  progress_percentage?: number
  joined_at?: string
  completed_at?: string
}

type EnrollmentPage = {
  enrollments?: Enrollment[]
  has_more?: boolean
  next_cursor?: string
}

const props = defineProps<{ candidateId: string; candidateName: string }>()
const emit = defineEmits<{ close: [] }>()
const { lang, t } = useAdminLanguage()
const copy = computed(() => t.value.dashboard.enrollments)
const rows = ref<Enrollment[]>([])
const loading = ref(false)
const error = ref("")
const nextCursor = ref("")
const hasMore = ref(false)
const closeButton = ref<HTMLButtonElement | null>(null)
let requestId = 0

function formatDate(raw?: string) {
  if (!raw) return "-"
  const date = new Date(raw)
  return Number.isNaN(date.getTime()) ? raw : date.toLocaleDateString(lang.value === "zh" ? "zh-CN" : "en-US")
}

function statusLabel(status?: string) {
  if (status?.toLowerCase() === "learning") return copy.value.learning
  if (status?.toLowerCase() === "completed") return copy.value.completed
  return status || "-"
}

async function loadPage() {
  if (loading.value || !props.candidateId) return
  const currentRequest = ++requestId
  const cursor = nextCursor.value
  // Loading replaces the retry/load-more button; keep keyboard focus inside the dialog.
  closeButton.value?.focus({ preventScroll: true })
  loading.value = true
  error.value = ""
  try {
    const params = new URLSearchParams({ candidate_id: props.candidateId, page_size: "20" })
    if (cursor) params.set("cursor", cursor)
    const page = await apiClient<EnrollmentPage>(`/api/lms/enrollments?${params}`)
    if (currentRequest !== requestId) return
    const next = page.next_cursor || ""
    if (page.has_more && (!next || next === cursor)) throw new Error("Enrollment pagination cursor did not advance")
    rows.value = [...rows.value, ...(page.enrollments || [])]
    nextCursor.value = next
    hasMore.value = Boolean(page.has_more)
  } catch (err) {
    if (currentRequest !== requestId) return
    console.error(err)
    error.value = copy.value.loadFailed
  } finally {
    if (currentRequest === requestId) loading.value = false
  }
}

onMounted(loadPage)
onUnmounted(() => { requestId += 1 })
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4" @click.self="emit('close')">
      <section v-modal-dialog="() => emit('close')" class="flex max-h-[90dvh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl" aria-labelledby="candidate-enrollments-title">
        <header class="flex shrink-0 items-start justify-between gap-4 border-b border-slate-100 px-5 py-4">
          <div class="min-w-0">
            <h2 id="candidate-enrollments-title" class="text-xl font-black text-slate-950">{{ copy.title }}</h2>
            <p class="mt-1 break-words text-sm text-slate-600">{{ candidateName || "-" }}</p>
            <p class="mt-1 break-all text-xs text-slate-500">{{ candidateId }}</p>
          </div>
          <button ref="closeButton" data-dialog-initial-focus class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" type="button" :aria-label="t.dashboard.close" @click="emit('close')">
            <X class="h-5 w-5" aria-hidden="true" />
          </button>
        </header>
        <div class="overflow-y-auto px-5 py-4" :aria-busy="loading">
          <div v-if="rows.length" class="grid gap-4">
            <article v-for="row in rows" :key="row.enrollment_id" class="rounded-xl border border-slate-200 p-4">
              <dl class="grid gap-3 text-sm sm:grid-cols-2">
                <div><dt class="font-bold text-slate-500">{{ copy.courseId }}</dt><dd class="mt-1 break-all">{{ row.course_ulid || "-" }}</dd></div>
                <div><dt class="font-bold text-slate-500">{{ copy.enrollmentId }}</dt><dd class="mt-1 break-all">{{ row.enrollment_id }}</dd></div>
                <div><dt class="font-bold text-slate-500">{{ copy.status }}</dt><dd class="mt-1">{{ statusLabel(row.status) }}</dd></div>
                <div><dt class="font-bold text-slate-500">{{ copy.progress }}</dt><dd class="mt-1">{{ row.progress_percentage ?? 0 }}%</dd></div>
                <div><dt class="font-bold text-slate-500">{{ copy.joinedAt }}</dt><dd class="mt-1">{{ formatDate(row.joined_at) }}</dd></div>
                <div><dt class="font-bold text-slate-500">{{ copy.completedAt }}</dt><dd class="mt-1">{{ formatDate(row.completed_at) }}</dd></div>
              </dl>
            </article>
          </div>
          <p v-if="loading" class="py-6 text-center text-sm text-slate-500" role="status"><Loader2 class="mr-2 inline h-5 w-5 animate-spin" />{{ t.dashboard.loading }}</p>
          <div v-else-if="error" class="py-6 text-center">
            <p role="alert" class="text-sm text-red-700">{{ error }}</p>
            <button class="mt-3 rounded-xl border border-slate-200 px-4 py-2 text-sm font-bold" type="button" @click="loadPage">{{ copy.retry }}</button>
          </div>
          <p v-else-if="!rows.length" class="py-6 text-center text-sm text-slate-500">{{ copy.empty }}</p>
          <button v-else-if="hasMore" class="mt-4 w-full rounded-xl border border-slate-200 px-4 py-2 text-sm font-bold" type="button" @click="loadPage">{{ copy.loadMore }}</button>
        </div>
      </section>
    </div>
  </Teleport>
</template>
