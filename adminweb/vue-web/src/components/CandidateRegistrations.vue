<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"

type Registration = { pipeline_ulid: string; pipeline_cc_ulid: string; status: string | number; created_at?: string; completed_at?: string }
const props = defineProps<{ candidateId: string }>()
const { t, lang } = useAdminLanguage()
const copy = computed(() => t.value.dashboard.registrations)
const rows = ref<Registration[]>([])
const names = ref<Record<string, string>>({})
const loading = ref(false)
const failed = ref(false)
const status = ref("")
const cursor = ref("")
const hasMore = ref(false)
let controller: AbortController | undefined
const statuses = computed(() => [
  { value: "1", code: "RUNNING", label: t.value.progAdmin.status.pipeline.running },
  { value: "2", code: "WAIT_FINAL_ELIG", label: t.value.progAdmin.status.pipeline.waitingFinalQualification },
  { value: "3", code: "COMPLETED", label: t.value.progAdmin.status.pipeline.completed },
  { value: "4", code: "ISSUING_CERT", label: t.value.progAdmin.status.pipeline.issuingCertificate },
  { value: "5", code: "CANCELLED", label: t.value.progAdmin.status.pipeline.cancelled },
])
function statusLabel(value: string | number) {
  const code = String(value).replace(/^PIPELINE_STATUS_/, "")
  return statuses.value.find((item) => item.value === code || item.code === code)?.label || t.value.dashboard.unknownStatus
}
function dateLabel(value?: string) {
  if (!value) return "—"
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleDateString(lang.value === "zh" ? "zh-CN" : "en-US")
}
async function load(reset = false) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const init = { signal: request.signal }
  if (reset) { rows.value = []; cursor.value = ""; hasMore.value = false }
  loading.value = true
  failed.value = false
  try {
    const params = new URLSearchParams({ candidate_ulid: props.candidateId, page_size: "20" })
    if (status.value) params.set("status", status.value)
    if (cursor.value) params.set("cursor", cursor.value)
    const result = await apiClient<{ pipelines?: Registration[]; has_more?: boolean; next_cursor?: string }>(`/api/prog/pipelines?${params}`, init)
    if (request.signal.aborted) return
    if (!Array.isArray(result?.pipelines) || typeof result.has_more !== "boolean") throw new Error("Incomplete registration page")
    if (result.has_more && (!result.next_cursor || result.next_cursor === cursor.value)) throw new Error("Registration cursor did not advance")
    const items = result.pipelines
    rows.value = [...rows.value, ...items]
    cursor.value = result.next_cursor || ""
    hasMore.value = result.has_more
    await Promise.all([...new Set(items.map((item) => item.pipeline_cc_ulid).filter((id) => id && !names.value[id]))].map(async (id) => {
      try {
        const config = await apiClient<{ name?: string }>(`/api/pipelines/${encodeURIComponent(id)}`, init)
        if (!request.signal.aborted && config.name) names.value[id] = config.name
      } catch (err) {
        if (!request.signal.aborted) console.error(err)
      }
    }))
  } catch (err) {
    if (request.signal.aborted) return
    console.error(err)
    failed.value = true
  } finally {
    if (!request.signal.aborted) loading.value = false
  }
}
watch(status, () => void load(true))
onMounted(() => void load())
onUnmounted(() => controller?.abort())
</script>

<template>
  <section class="border-t border-slate-200 py-5" :aria-label="copy.title">
    <h3 class="text-lg font-black">{{ copy.title }}</h3>
    <p class="mt-1 text-sm text-slate-500">{{ copy.hint }}</p>
    <label class="mt-4 grid gap-2 text-sm font-bold">
      {{ copy.status }}
      <select v-model="status" class="h-11 rounded-xl border border-slate-200 bg-white px-3">
        <option value="">{{ t.dashboard.filters.allStatus }}</option>
        <option v-for="item in statuses" :key="item.value" :value="item.value">{{ item.label }}</option>
      </select>
    </label>
    <div class="mt-4 grid gap-3">
      <article v-for="row in rows" :key="row.pipeline_ulid" class="rounded-xl border border-slate-200 p-4">
        <h4 class="break-words font-bold">{{ names[row.pipeline_cc_ulid] || copy.nameUnavailable }}</h4>
        <dl class="mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <div><dt class="text-slate-500">{{ copy.id }}</dt><dd class="break-all">{{ row.pipeline_ulid }}</dd></div>
          <div><dt class="text-slate-500">{{ copy.status }}</dt><dd>{{ statusLabel(row.status) }}</dd></div>
          <div><dt class="text-slate-500">{{ copy.created }}</dt><dd>{{ dateLabel(row.created_at) }}</dd></div>
          <div><dt class="text-slate-500">{{ copy.completed }}</dt><dd>{{ dateLabel(row.completed_at) }}</dd></div>
        </dl>
      </article>
    </div>
    <p v-if="loading" class="py-4 text-sm text-slate-500" role="status">{{ t.dashboard.loading }}</p>
    <div v-else-if="failed" class="py-4">
      <p role="alert" class="text-sm text-red-700">{{ copy.failed }}</p>
      <button type="button" class="mt-3 rounded-xl border px-4 py-2 text-sm font-bold" @click="load()">{{ t.dashboard.enrollments.retry }}</button>
    </div>
    <p v-else-if="!rows.length" class="py-4 text-sm text-slate-500">{{ copy.empty }}</p>
    <button v-else-if="hasMore" type="button" class="mt-4 rounded-xl border px-4 py-2 text-sm font-bold" @click="load()">{{ t.dashboard.enrollments.loadMore }}</button>
  </section>
</template>
