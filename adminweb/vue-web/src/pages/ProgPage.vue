<script setup lang="ts">
import { ArrowLeft, Eye, Loader2, RefreshCw, Search, ShieldX, StepForward, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type ActionKind = "trigger-next-stage" | "terminate-pipeline" | "force-completed" | "force-signup-exam"
type DetailTab = "overview" | "stages" | "units" | "logs" | "raw"

type PendingAction = {
  kind: ActionKind
  pipelineUlid: string
  courseUnitUlid?: string
}

type UnitListItem = {
  key: string
  stageIndex: number
  unitIndex: number
  stage: JsonRecord
  unit: JsonRecord
}

const pageSize = 20
const logPageSize = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.progAdmin)

const pipelines = ref<JsonRecord[]>([])
const pipelineNameByCc = ref<Record<string, string>>({})
const selectedSummary = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const logs = ref<JsonRecord[]>([])
const selectedLog = ref<JsonRecord | null>(null)
const logDetail = ref<JsonRecord | null>(null)

const loading = ref(false)
const detailLoading = ref(false)
const logsLoading = ref(false)
const logDetailLoading = ref(false)
const actionLoading = ref(false)
const certificateLoading = ref(false)

const candidateFilter = ref("")
const statusFilter = ref("all")
const offset = ref(0)
const logOffset = ref(0)
const activeTab = ref<DetailTab>("overview")
const selectedStageIndex = ref(0)
const selectedUnitKey = ref("")
const pendingAction = ref<PendingAction | null>(null)
const actionReason = ref("")

function asRecord(value: unknown): JsonRecord {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : {}
}

function asArray(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

const detailPipelineRecord = computed(() => asRecord(detail.value?.pipeline))
const selectedPipelineUlid = computed(() => String(detailPipelineRecord.value.pipeline_ulid || selectedSummary.value?.pipeline_ulid || ""))
const selectedCandidateUlid = computed(() => String(detailPipelineRecord.value.candidate_ulid || selectedSummary.value?.candidate_ulid || ""))
const selectedPipelineCcUlid = computed(() => String(detailPipelineRecord.value.pipeline_cc_ulid || selectedSummary.value?.pipeline_cc_ulid || ""))
const selectedCurrentStageUlid = computed(() => String(detailPipelineRecord.value.current_stage_ulid || selectedSummary.value?.current_stage_ulid || ""))
const selectedStatus = computed(() => detailPipelineRecord.value.status ?? selectedSummary.value?.status)
const stages = computed(() => asArray(detail.value?.stages))
const units = computed<UnitListItem[]>(() => {
  const list: UnitListItem[] = []
  stages.value.forEach((stage, stageIndex) => {
    courseUnits(stage).forEach((unit, unitIndex) => {
      list.push({
        key: `${stageIndex}:${unitIndex}:${courseUnitUlid(unit)}`,
        stageIndex,
        unitIndex,
        stage,
        unit,
      })
    })
  })
  return list
})
const selectedStage = computed(() => stages.value[selectedStageIndex.value] || null)
const selectedUnit = computed(() => units.value.find((item) => item.key === selectedUnitKey.value) || units.value[0] || null)
const totalUnits = computed(() => units.value.length)
const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => pipelines.value.length >= pageSize)
const canPrevLogs = computed(() => logOffset.value > 0)
const canNextLogs = computed(() => logs.value.length >= logPageSize)
const canViewCertificate = computed(() => Boolean(selectedPipelineUlid.value && selectedCandidateUlid.value))
const canTerminatePipeline = computed(() => Boolean(selectedPipelineUlid.value && !["3", "4"].includes(String(selectedStatus.value ?? ""))))
const canTriggerNextStage = computed(() => {
  const currentStageUlid = String(detailPipelineRecord.value.current_stage_ulid || "")
  const currentStage = stages.value.find((stage) => stageUlid(stage) === currentStageUlid) || stages.value[0]
  return String(selectedStatus.value ?? "") === "1" && String(stageRecord(currentStage).status ?? "") === "3"
})

const detailTabs = computed(() => [
  { key: "overview" as const, title: copy.value.tabs.overview.title, desc: copy.value.tabs.overview.desc, count: selectedSummary.value ? 1 : 0 },
  { key: "stages" as const, title: copy.value.tabs.stages.title, desc: copy.value.tabs.stages.desc, count: stages.value.length },
  { key: "units" as const, title: copy.value.tabs.units.title, desc: copy.value.tabs.units.desc, count: units.value.length },
  { key: "logs" as const, title: copy.value.tabs.logs.title, desc: copy.value.tabs.logs.desc, count: logs.value.length },
  { key: "raw" as const, title: copy.value.tabs.raw.title, desc: copy.value.tabs.raw.desc, count: 1 },
])

const statusOptions = computed(() => [
  { value: "all", label: copy.value.status.all },
  { value: "1", label: copy.value.status.pipeline.running },
  { value: "2", label: copy.value.status.pipeline.waitingFinalQualification },
  { value: "3", label: copy.value.status.pipeline.completed },
  { value: "4", label: copy.value.status.pipeline.terminated },
])

function pipelineUlid(pipeline: JsonRecord | null | undefined) {
  return String(pickFirst(pipeline || {}, ["pipeline_ulid", "pipeline_id"]) || "")
}

function pipelineCcUlid(pipeline: JsonRecord | null | undefined) {
  return String(pickFirst(pipeline || {}, ["pipeline_cc_ulid", "pipeline_config_ulid"]) || "")
}

function pipelineDisplayName(pipeline: JsonRecord | null | undefined) {
  const cc = pipelineCcUlid(pipeline)
  return pipelineNameByCc.value[cc] || String(pickFirst(pipeline || {}, ["name", "pipeline_name"]) || pipelineUlid(pipeline) || "Pipeline")
}

function statusLabel(value: unknown, scope: "pipeline" | "stage" | "unit" = "pipeline") {
  const normalized = String(value ?? "")
  if (scope === "pipeline") {
    if (normalized === "1") return copy.value.status.pipeline.running
    if (normalized === "2") return copy.value.status.pipeline.waitingFinalQualification
    if (normalized === "3") return copy.value.status.pipeline.completed
    if (normalized === "4") return copy.value.status.pipeline.terminated
  }
  if (scope === "stage") {
    if (normalized === "1") return copy.value.status.stage.waitingCandidate
    if (normalized === "2") return copy.value.status.stage.running
    if (normalized === "3") return copy.value.status.stage.completed
    if (normalized === "4") return copy.value.status.stage.terminated
  }
  if (scope === "unit") {
    if (normalized === "1") return copy.value.status.unit.notStarted
    if (normalized === "2") return copy.value.status.unit.studying
    if (normalized === "3") return copy.value.status.unit.completed
    if (normalized === "4") return copy.value.status.unit.readyForExamSignup
    if (normalized === "5") return copy.value.status.unit.examScheduled
    if (normalized === "6") return copy.value.status.unit.examFailed
  }
  return String(value || "-")
}

function actionLabel(kind: ActionKind) {
  return copy.value.actions[kind]
}

function entityStatusLabel(entityType: unknown, value: unknown) {
  const normalizedType = String(entityType || "").toUpperCase()
  if (normalizedType === "STAGE") return statusLabel(value, "stage")
  if (normalizedType === "COURSE_UNIT") return statusLabel(value, "unit")
  return statusLabel(value, "pipeline")
}

function stageRecord(stage: JsonRecord | null | undefined) {
  return asRecord(stage?.stage)
}

function stageUlid(stage: JsonRecord | null | undefined) {
  return String(stageRecord(stage).stage_ulid || stage?.stage_ulid || "")
}

function stageStatus(stage: JsonRecord | null | undefined) {
  return stageRecord(stage).status ?? stage?.status
}

function stageName(stage: JsonRecord | null | undefined) {
  const record = stageRecord(stage)
  const name = record.name
  if (name) return String(name)
  const seqNo = record.seq_no || record.sort_order
  return seqNo ? copy.value.stageNameWithSeq(String(seqNo)) : copy.value.stageFallback
}

function courseUnits(stage: JsonRecord | null | undefined) {
  return asArray(stage?.course_units)
}

function courseUnitUlid(unit: JsonRecord | null | undefined) {
  return String(unit?.course_unit_ulid || "")
}

function courseUnitStatus(unit: JsonRecord | null | undefined) {
  return unit?.status
}

function ensureSelections() {
  if (selectedStageIndex.value >= stages.value.length) selectedStageIndex.value = Math.max(0, stages.value.length - 1)
  if (!selectedUnitKey.value || !units.value.some((item) => item.key === selectedUnitKey.value)) selectedUnitKey.value = units.value[0]?.key || ""
  if (!selectedLog.value && logs.value.length) selectedLog.value = logs.value[0]
}

async function loadPipelineCatalog() {
  try {
    const data = await apiClient<JsonRecord>("/api/pipelines?limit=200&offset=0")
    const list = Array.isArray(data.pipelines) ? data.pipelines : []
    const next: Record<string, string> = {}
    for (const item of list) {
      if (!item || typeof item !== "object" || Array.isArray(item)) continue
      const id = String(item.pipeline_ulid || item.pipeline_id || "")
      if (id) next[id] = String(item.name || item.category_tips || id)
    }
    pipelineNameByCc.value = next
  } catch {
    pipelineNameByCc.value = {}
  }
}

async function loadPipelines() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(pageSize),
      offset: String(offset.value),
    })
    if (candidateFilter.value.trim()) params.set("candidate_ulid", candidateFilter.value.trim())
    if (statusFilter.value !== "all") params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/prog/pipelines?${params}`)
    const list = Array.isArray(data.pipelines) ? data.pipelines : []
    pipelines.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    pipelines.value = []
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

async function openPipeline(pipeline: JsonRecord) {
  selectedSummary.value = pipeline
  detail.value = null
  logs.value = []
  selectedLog.value = null
  logDetail.value = null
  activeTab.value = "overview"
  selectedStageIndex.value = 0
  selectedUnitKey.value = ""
  await loadDetail(pipelineUlid(pipeline))
  await loadLogs(pipelineUlid(pipeline), 0)
}

async function loadDetail(pipelineId: string) {
  if (!pipelineId) return
  detailLoading.value = true
  try {
    detail.value = await apiClient<JsonRecord>(`/api/prog/pipelines/${encodeURIComponent(pipelineId)}`)
    ensureSelections()
  } catch (err) {
    console.error(err)
    detail.value = null
    toast.error(copy.value.toasts.detailLoadFailed)
  } finally {
    detailLoading.value = false
  }
}

async function loadLogs(pipelineId = selectedPipelineUlid.value, targetOffset = logOffset.value) {
  if (!pipelineId) return
  logsLoading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(logPageSize),
      offset: String(targetOffset),
    })
    const data = await apiClient<JsonRecord>(`/api/prog/pipelines/${encodeURIComponent(pipelineId)}/logs?${params}`)
    const list = Array.isArray(data.logs) ? data.logs : []
    logs.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    logOffset.value = targetOffset
    selectedLog.value = logs.value[0] || null
    if (selectedLog.value) await loadLogDetail(String(selectedLog.value.transition_ulid || ""))
  } catch (err) {
    console.error(err)
    logs.value = []
    selectedLog.value = null
    logDetail.value = null
    toast.error(copy.value.toasts.logsLoadFailed)
  } finally {
    logsLoading.value = false
  }
}

async function loadLogDetail(transitionUlid: string) {
  if (!transitionUlid) {
    logDetail.value = null
    return
  }
  logDetailLoading.value = true
  try {
    logDetail.value = await apiClient<JsonRecord>(`/api/prog/pipelines/logs/${encodeURIComponent(transitionUlid)}`)
  } catch (err) {
    console.error(err)
    logDetail.value = null
  } finally {
    logDetailLoading.value = false
  }
}

async function reloadSelected() {
  await loadPipelines()
  if (selectedPipelineUlid.value) {
    await loadDetail(selectedPipelineUlid.value)
    await loadLogs(selectedPipelineUlid.value, logOffset.value)
  }
}

function openAction(action: PendingAction) {
  pendingAction.value = action
  actionReason.value = ""
}

async function submitAction() {
  if (!pendingAction.value) return
  const action = pendingAction.value
  const body = JSON.stringify({ reason_message: actionReason.value.trim() })
  actionLoading.value = true
  try {
    if (action.kind === "trigger-next-stage") {
      await apiClient(`/api/prog/pipelines/${encodeURIComponent(action.pipelineUlid)}/trigger-next-stage`, { method: "POST", body })
      toast.success(copy.value.toasts.nextStageTriggered)
    } else if (action.kind === "terminate-pipeline") {
      await apiClient(`/api/prog/pipelines/${encodeURIComponent(action.pipelineUlid)}/terminate`, { method: "POST", body })
      toast.success(copy.value.toasts.pipelineTerminated)
    } else if (action.kind === "force-completed" && action.courseUnitUlid) {
      await apiClient(`/api/prog/course-units/${encodeURIComponent(action.courseUnitUlid)}/force-completed`, { method: "POST", body })
      toast.success(copy.value.toasts.unitForcedCompleted)
    } else if (action.kind === "force-signup-exam" && action.courseUnitUlid) {
      await apiClient(`/api/prog/course-units/${encodeURIComponent(action.courseUnitUlid)}/force-signup-exam`, { method: "POST", body })
      toast.success(copy.value.toasts.unitResetToExamSignup)
    }
    pendingAction.value = null
    actionReason.value = ""
    await reloadSelected()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.actionFailed)
  } finally {
    actionLoading.value = false
  }
}

async function viewCertificate() {
  if (!selectedPipelineUlid.value || !selectedCandidateUlid.value) {
    toast.error(copy.value.toasts.missingCertificateIds)
    return
  }
  certificateLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(
      `/api/prog/pipelines/${encodeURIComponent(selectedPipelineUlid.value)}/certificate-url?candidate_ulid=${encodeURIComponent(selectedCandidateUlid.value)}`,
    )
    const url = String(data.view_url || "")
    if (!url) {
      toast.error(copy.value.toasts.noCertificate)
      return
    }
    window.open(url, "_blank", "noopener,noreferrer")
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.certificateUrlFailed)
  } finally {
    certificateLoading.value = false
  }
}

function backToList() {
  selectedSummary.value = null
  detail.value = null
  logs.value = []
  selectedLog.value = null
  logDetail.value = null
  activeTab.value = "overview"
}

watch([candidateFilter, statusFilter, offset], () => loadPipelines())
onMounted(async () => {
  await Promise.all([loadPipelineCatalog(), loadPipelines()])
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="selectedSummary" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="backToList">
          <ArrowLeft class="h-4 w-4" />
          {{ copy.backToList }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="reloadSelected">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <div class="grid gap-6">
      <aside class="space-y-4">
        <div class="grid gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <label class="relative grid gap-2 text-sm font-bold">
            {{ copy.filters.candidate }}
            <Search class="absolute bottom-3 left-3 h-4 w-4 text-slate-400" />
            <input v-model="candidateFilter" class="h-10 rounded-xl border border-slate-200 pl-9 pr-3" :placeholder="copy.filters.candidatePlaceholder" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.filters.status }}
            <select v-model="statusFilter" class="h-10 rounded-xl border border-slate-200 px-3">
              <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
        </div>

        <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <div>
              <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ pipelines.length }}</span>
          </div>
          <div v-if="loading" class="px-6 py-10 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <template v-else-if="pipelines.length">
            <div class="grid grid-cols-[minmax(0,1fr)_150px_180px_110px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
              <span>{{ copy.columns.pipeline }}</span>
              <span class="text-center">{{ copy.columns.status }}</span>
              <span class="text-right">{{ copy.columns.startedAt }}</span>
              <span class="text-right">{{ copy.columns.action }}</span>
            </div>
            <button
              v-for="pipeline in pipelines"
              :key="pipelineUlid(pipeline)"
              class="grid w-full grid-cols-[minmax(0,1fr)_150px_180px_110px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50"
              :class="pipelineUlid(pipeline) === selectedPipelineUlid ? 'bg-sky-50' : ''"
              type="button"
              @click="openPipeline(pipeline)"
            >
              <div class="min-w-0">
                <div class="truncate text-lg font-black">{{ pipelineDisplayName(pipeline) }}</div>
                <div class="mt-1 break-all text-sm text-slate-500">{{ copy.candidatePrefix }}{{ pipeline.candidate_name || pipeline.candidate_ulid || "-" }}</div>
                <div class="mt-3 grid gap-1 rounded-xl bg-slate-50 px-3 py-2 text-xs text-slate-500">
                  <div class="break-all font-semibold">Pipeline: {{ pipelineUlid(pipeline) || "-" }}</div>
                  <div class="break-all">{{ copy.currentStagePrefix }}{{ pipeline.current_stage_ulid || "-" }}</div>
                </div>
              </div>
              <span class="self-center justify-self-center whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(pipeline.status))">{{ statusLabel(pipeline.status) }}</span>
              <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(pipeline.started_at || pipeline.created_at || "")) }}</span>
              <span class="inline-flex h-9 items-center self-center justify-self-end rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50">{{ copy.viewDetails }}</span>
            </button>
          </template>
          <div v-else class="px-6 py-10 text-center text-slate-500">{{ copy.empty }}</div>
          <div class="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - pageSize)">{{ copy.prev }}</button>
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += pageSize">{{ copy.next }}</button>
          </div>
        </section>
      </aside>
    </div>

    <div v-if="selectedSummary" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="flex max-h-[88vh] w-full max-w-[1200px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-6 py-4">
          <div>
            <h2 class="text-2xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ selectedPipelineUlid }}</p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="backToList">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="flex-1 overflow-y-auto p-5">
          <main class="min-w-0 space-y-6">
        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 class="text-2xl font-black">{{ pipelineDisplayName(selectedSummary) }}</h2>
              <p class="mt-1 break-all text-sm text-slate-500">{{ selectedPipelineUlid }}</p>
            </div>
            <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(selectedStatus))">
              {{ statusLabel(selectedStatus) }}
            </span>
          </div>
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-xl border border-slate-200 bg-slate-50 p-3">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.candidate }}</div>
              <div class="mt-1 break-all text-sm font-bold">{{ selectedCandidateUlid || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 bg-slate-50 p-3">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.pipelineCc }}</div>
              <div class="mt-1 break-all text-sm font-bold">{{ selectedPipelineCcUlid || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 bg-slate-50 p-3">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.currentStage }}</div>
              <div class="mt-1 break-all text-sm font-bold">{{ selectedCurrentStageUlid || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 bg-slate-50 p-3">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.count }}</div>
              <div class="mt-1 text-sm font-bold">{{ copy.summary.countText(stages.length, totalUnits) }}</div>
            </div>
          </div>
        </section>

        <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
            <div class="border-b border-slate-200 p-4">
              <h3 class="text-lg font-black">{{ copy.levelTitle }}</h3>
              <p class="mt-1 text-sm text-slate-500">{{ copy.levelDescription }}</p>
              <div class="mt-4 flex gap-2 overflow-x-auto pb-1">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="inline-flex h-11 shrink-0 items-center gap-3 rounded-2xl border px-4 text-sm font-black transition"
                  :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 text-slate-950' : 'border-slate-100 bg-white text-slate-700 hover:bg-slate-50'"
                  type="button"
                  @click="activeTab = tab.key"
                >
                    <span>{{ tab.title }}</span>
                    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
                </button>
              </div>

              <div class="mt-4 rounded-2xl border border-slate-200 bg-slate-50 p-3">
                <h4 class="font-black">{{ copy.manualActionsTitle }}</h4>
                <p class="mt-1 text-xs text-slate-500">{{ copy.manualActionsDescription }}</p>
                <div class="mt-3 flex flex-wrap gap-2">
                  <button
                    class="inline-flex items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-2.5 text-sm font-bold text-white disabled:opacity-40"
                    type="button"
                    :disabled="!canTriggerNextStage"
                    @click="openAction({ kind: 'trigger-next-stage', pipelineUlid: selectedPipelineUlid })"
                  >
                    <StepForward class="h-4 w-4" />
                    {{ copy.actions["trigger-next-stage"] }}
                  </button>
                  <button
                    class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-bold text-white disabled:opacity-40"
                    type="button"
                    :disabled="!canTerminatePipeline"
                    @click="openAction({ kind: 'terminate-pipeline', pipelineUlid: selectedPipelineUlid })"
                  >
                    <ShieldX class="h-4 w-4" />
                    {{ copy.actions["terminate-pipeline"] }}
                  </button>
                  <button
                    class="inline-flex items-center justify-center gap-2 rounded-xl border bg-white px-4 py-2.5 text-sm font-bold disabled:opacity-40"
                    type="button"
                    :disabled="certificateLoading || !canViewCertificate"
                    @click="viewCertificate"
                  >
                    <Eye class="h-4 w-4" />
                    {{ copy.viewCertificate }}
                  </button>
                </div>
              </div>
            </div>

            <section class="h-[60vh] min-h-[360px] max-h-[620px] min-w-0 overflow-y-auto">
              <div v-if="detailLoading" class="px-6 py-10 text-center text-slate-500">
                <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                {{ copy.loadingDetails }}
              </div>

              <div v-else-if="activeTab === 'overview'" class="space-y-4 p-5">
                <div class="grid gap-3 md:grid-cols-2">
                  <div v-for="(value, key) in detailPipelineRecord" :key="key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                    <div class="text-xs font-black uppercase text-slate-400">{{ key }}</div>
                    <div class="mt-1 break-all text-sm font-bold">{{ key === 'status' ? statusLabel(value) : (value || '-') }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'stages'" class="grid min-h-[640px] lg:grid-cols-[320px_minmax(0,1fr)]">
                <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
                  <div class="border-b border-slate-200 p-4">
                    <div class="font-black">{{ copy.stageListTitle }}</div>
                    <div class="text-xs text-slate-500">{{ copy.stageListDescription }}</div>
                  </div>
                  <button
                    v-for="(stage, index) in stages"
                    :key="stageUlid(stage) || index"
                    class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                    :class="selectedStageIndex === index ? 'bg-sky-50' : ''"
                    type="button"
                    @click="selectedStageIndex = index"
                  >
                    <div class="font-black">{{ stageName(stage) }}</div>
                    <div class="mt-1 text-sm text-slate-500">{{ copy.courseUnitCount(courseUnits(stage).length) }}</div>
                    <div class="mt-2 break-all text-xs text-slate-500">ID: {{ stageUlid(stage) || "-" }}</div>
                    <span class="mt-3 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(stageStatus(stage), 'stage'))">
                      {{ statusLabel(stageStatus(stage), "stage") }}
                    </span>
                  </button>
                  <div v-if="!stages.length" class="px-6 py-10 text-center text-slate-500">{{ copy.noStages }}</div>
                </div>
                <div class="space-y-5 p-5">
                  <template v-if="selectedStage">
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="(value, key) in stageRecord(selectedStage)" :key="key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                        <div class="text-xs font-black uppercase text-slate-400">{{ key }}</div>
                        <div class="mt-1 break-all text-sm font-bold">{{ key === 'status' ? statusLabel(value, 'stage') : (value || '-') }}</div>
                      </div>
                    </div>
                    <pre class="max-h-[360px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedStage, null, 2) }}</pre>
                  </template>
                  <div v-else class="p-12 text-center text-slate-500">{{ copy.selectStage }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'units'" class="grid min-h-[640px] lg:grid-cols-[360px_minmax(0,1fr)]">
                <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
                  <div class="border-b border-slate-200 p-4">
                    <div class="font-black">{{ copy.unitListTitle }}</div>
                    <div class="text-xs text-slate-500">{{ copy.unitListDescription }}</div>
                  </div>
                  <button
                    v-for="item in units"
                    :key="item.key"
                    class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                    :class="selectedUnitKey === item.key ? 'bg-sky-50' : ''"
                    type="button"
                    @click="selectedUnitKey = item.key"
                  >
                    <div class="font-black">{{ item.unit.course_unit_cc_ulid || item.unit.course_unit_ulid || copy.unitFallback(item.unitIndex + 1) }}</div>
                    <div class="mt-1 text-sm text-slate-500">{{ copy.parentStagePrefix }}{{ stageName(item.stage) }}</div>
                    <div class="mt-2 break-all text-xs text-slate-500">ID: {{ courseUnitUlid(item.unit) || "-" }}</div>
                    <span class="mt-3 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(courseUnitStatus(item.unit), 'unit'))">
                      {{ statusLabel(courseUnitStatus(item.unit), "unit") }}
                    </span>
                  </button>
                  <div v-if="!units.length" class="px-6 py-10 text-center text-slate-500">{{ copy.noUnits }}</div>
                </div>
                <div class="space-y-5 p-5">
                  <template v-if="selectedUnit">
                    <div class="rounded-xl border border-sky-100 bg-sky-50 p-3">
                      <div class="text-xs font-black uppercase text-sky-600">{{ copy.parentStage }}</div>
                      <div class="mt-1 break-all text-sm font-bold">{{ stageName(selectedUnit.stage) }} · {{ stageUlid(selectedUnit.stage) || "-" }}</div>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="(value, key) in selectedUnit.unit" :key="key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                        <div class="text-xs font-black uppercase text-slate-400">{{ key }}</div>
                        <div class="mt-1 break-all text-sm font-bold">{{ key === 'status' ? statusLabel(value, 'unit') : (value ?? '-') }}</div>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-3">
                      <button
                        class="rounded-xl border bg-white px-4 py-2 text-sm font-bold disabled:opacity-40"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-completed', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.forceCompleted }}
                      </button>
                      <button
                        class="rounded-xl border bg-white px-4 py-2 text-sm font-bold disabled:opacity-40"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-signup-exam', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.resetExamSignup }}
                      </button>
                    </div>
                    <pre class="max-h-[300px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedUnit.unit, null, 2) }}</pre>
                  </template>
                  <div v-else class="p-12 text-center text-slate-500">{{ copy.selectUnit }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'logs'" class="grid min-h-[640px] lg:grid-cols-[380px_minmax(0,1fr)]">
                <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
                  <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                    <div>
                      <div class="font-black">{{ copy.logListTitle }}</div>
                      <div class="text-xs text-slate-500">{{ copy.logListDescription }}</div>
                    </div>
                    <button class="rounded-xl border px-3 py-2 text-sm font-bold" type="button" @click="loadLogs()">{{ copy.load }}</button>
                  </div>
                  <div v-if="logsLoading" class="p-10 text-center text-slate-500">
                    <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                    {{ copy.loading }}
                  </div>
                  <button
                    v-for="log in logs"
                    v-else
                    :key="String(log.transition_ulid)"
                    class="w-full border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
                    :class="selectedLog === log ? 'bg-sky-50' : ''"
                    type="button"
                    @click="selectedLog = log; loadLogDetail(String(log.transition_ulid || ''))"
                  >
                    <div class="font-black">{{ entityStatusLabel(log.entity_type, log.from_status) }} -> {{ entityStatusLabel(log.entity_type, log.to_status) }}</div>
                    <div class="mt-1 text-sm text-slate-500">{{ log.entity_type || "-" }} · {{ log.entity_ulid || "-" }}</div>
                    <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(log.created_at || "")) }}</div>
                  </button>
                  <div v-if="!logsLoading && !logs.length" class="p-10 text-center text-slate-500">{{ copy.noLogs }}</div>
                  <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
                    <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevLogs" @click="loadLogs(selectedPipelineUlid, Math.max(0, logOffset - logPageSize))">{{ copy.prev }}</button>
                    <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNextLogs" @click="loadLogs(selectedPipelineUlid, logOffset + logPageSize)">{{ copy.next }}</button>
                  </div>
                </div>
                <div class="space-y-5 p-5">
                  <h4 class="text-lg font-black">{{ copy.logDetailTitle }}</h4>
                  <div v-if="logDetailLoading" class="p-10 text-center text-slate-500">
                    <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                    {{ copy.loadingDetails }}
                  </div>
                  <template v-else>
                    <div v-if="selectedLog" class="grid gap-4 md:grid-cols-2">
                      <div v-for="(value, key) in asRecord((logDetail || selectedLog).summary || selectedLog)" :key="key" class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
                        <div class="text-xs font-black uppercase text-slate-400">{{ key }}</div>
                        <div class="mt-2 break-all text-sm font-bold">
                          {{ key === 'from_status' || key === 'to_status' ? entityStatusLabel(asRecord((logDetail || selectedLog).summary || selectedLog).entity_type, value) : (value || '-') }}
                        </div>
                      </div>
                    </div>
                    <pre class="max-h-[360px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(logDetail || selectedLog || {}, null, 2) }}</pre>
                  </template>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-5 p-5">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.rawHint }}
                </div>
                <pre class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify({ detail, logs }, null, 2) }}</pre>
              </div>
            </section>
        </section>
          </main>
        </div>
      </div>
    </div>

    <div v-if="pendingAction" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-lg rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">{{ copy.confirmTitle }}</h2>
        <p class="mt-2 text-sm text-slate-600">{{ copy.confirmDescription(actionLabel(pendingAction.kind)) }}</p>
        <textarea v-model="actionReason" class="mt-5 min-h-28 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.reasonPlaceholder" />
        <div class="mt-5 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" :disabled="actionLoading" @click="pendingAction = null">{{ copy.cancel }}</button>
          <button class="rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="actionLoading" @click="submitAction">
            {{ actionLoading ? copy.submitting : copy.confirmSubmit }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
