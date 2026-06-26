<script setup lang="ts">
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Eye,
  Loader2,
  RefreshCw,
  Search,
  ShieldX,
  StepForward,
} from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

type ActionKind = "trigger-next-stage" | "terminate-pipeline" | "force-completed" | "force-signup-exam"

type PendingAction = {
  kind: ActionKind
  pipelineUlid: string
  courseUnitUlid?: string
}

const pageSize = 20
const logPageSize = 20

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
const expandedStages = ref<Record<string, boolean>>({})
const pendingAction = ref<PendingAction | null>(null)
const actionReason = ref("")

function asRecord(value: unknown): JsonRecord {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : {}
}

const detailPipelineRecord = computed(() => asRecord(detail.value?.pipeline))
const selectedPipelineUlid = computed(() => String(detailPipelineRecord.value.pipeline_ulid || selectedSummary.value?.pipeline_ulid || ""))
const selectedCandidateUlid = computed(() => String(detailPipelineRecord.value.candidate_ulid || selectedSummary.value?.candidate_ulid || ""))
const selectedPipelineCcUlid = computed(() => String(detailPipelineRecord.value.pipeline_cc_ulid || selectedSummary.value?.pipeline_cc_ulid || ""))
const selectedCurrentStageUlid = computed(() => String(detailPipelineRecord.value.current_stage_ulid || selectedSummary.value?.current_stage_ulid || ""))
const selectedStatus = computed(() => detailPipelineRecord.value.status ?? selectedSummary.value?.status)
const stages = computed(() => {
  const value = detail.value?.stages
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
})
const totalUnits = computed(() => stages.value.reduce((count, stage) => count + courseUnits(stage).length, 0))
const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => pipelines.value.length >= pageSize)
const canPrevLogs = computed(() => logOffset.value > 0)
const canNextLogs = computed(() => logs.value.length >= logPageSize)
const canTriggerNextStage = computed(() => {
  const currentStageUlid = String(detailPipelineRecord.value.current_stage_ulid || "")
  const currentStage = stages.value.find((stage) => stageUlid(stage) === currentStageUlid) || stages.value[0]
  return String(selectedStatus.value ?? "") === "1" && String(stageRecord(currentStage).status ?? "") === "3"
})

const statusOptions = [
  { value: "all", label: "全部状态" },
  { value: "1", label: "运行中" },
  { value: "2", label: "等待最终资格" },
  { value: "3", label: "已完成" },
  { value: "4", label: "已终止" },
]

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
    if (normalized === "1") return "运行中"
    if (normalized === "2") return "等待最终资格"
    if (normalized === "3") return "已完成"
    if (normalized === "4") return "已终止"
  }
  if (scope === "stage") {
    if (normalized === "1") return "等待考生"
    if (normalized === "2") return "运行中"
    if (normalized === "3") return "已完成"
    if (normalized === "4") return "已终止"
  }
  if (scope === "unit") {
    if (normalized === "1") return "待学习"
    if (normalized === "2") return "学习中"
    if (normalized === "3") return "已完成"
    if (normalized === "4") return "待报名考试"
    if (normalized === "5") return "已预约考试"
    if (normalized === "6") return "考试失败"
  }
  return String(value || "-")
}

function stageUlid(stage: JsonRecord) {
  return String(stageRecord(stage).stage_ulid || stage.stage_ulid || "")
}

function stageStatus(stage: JsonRecord) {
  return stageRecord(stage).status ?? stage.status
}

function stageRecord(stage: JsonRecord | undefined) {
  return asRecord(stage?.stage)
}

function stageName(stage: JsonRecord) {
  const record = stageRecord(stage)
  const name = record.name
  if (name) return String(name)
  const seqNo = record.seq_no || record.sort_order
  return seqNo ? `阶段 ${seqNo}` : "阶段"
}

function courseUnits(stage: JsonRecord) {
  const value = stage.course_units
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function courseUnitUlid(unit: JsonRecord) {
  return String(unit.course_unit_ulid || "")
}

function courseUnitStatus(unit: JsonRecord) {
  return unit.status
}

function toggleStage(stage: JsonRecord) {
  const id = stageUlid(stage)
  expandedStages.value[id] = !expandedStages.value[id]
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
    toast.error("管线实例加载失败")
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
  expandedStages.value = {}
  await loadDetail(pipelineUlid(pipeline))
  await loadLogs(pipelineUlid(pipeline), 0)
}

async function loadDetail(pipelineId: string) {
  if (!pipelineId) return
  detailLoading.value = true
  try {
    detail.value = await apiClient<JsonRecord>(`/api/prog/pipelines/${encodeURIComponent(pipelineId)}`)
    for (const stage of stages.value) {
      expandedStages.value[stageUlid(stage)] = true
    }
  } catch (err) {
    console.error(err)
    detail.value = null
    toast.error("管线详情加载失败")
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
    toast.error("状态流转日志加载失败")
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
      toast.success("已推进下一阶段")
    } else if (action.kind === "terminate-pipeline") {
      await apiClient(`/api/prog/pipelines/${encodeURIComponent(action.pipelineUlid)}/terminate`, { method: "POST", body })
      toast.success("管线已终止")
    } else if (action.kind === "force-completed" && action.courseUnitUlid) {
      await apiClient(`/api/prog/course-units/${encodeURIComponent(action.courseUnitUlid)}/force-completed`, { method: "POST", body })
      toast.success("课时单元已强制完成")
    } else if (action.kind === "force-signup-exam" && action.courseUnitUlid) {
      await apiClient(`/api/prog/course-units/${encodeURIComponent(action.courseUnitUlid)}/force-signup-exam`, { method: "POST", body })
      toast.success("课时单元已重置为可预约考试")
    }
    pendingAction.value = null
    actionReason.value = ""
    await reloadSelected()
  } catch (err) {
    console.error(err)
    toast.error("操作失败")
  } finally {
    actionLoading.value = false
  }
}

async function viewCertificate() {
  if (!selectedPipelineUlid.value || !selectedCandidateUlid.value) {
    toast.error("缺少 pipeline_ulid 或 candidate_ulid")
    return
  }
  certificateLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(
      `/api/prog/pipelines/${encodeURIComponent(selectedPipelineUlid.value)}/certificate-url?candidate_ulid=${encodeURIComponent(selectedCandidateUlid.value)}`,
    )
    const url = String(data.view_url || "")
    if (!url) {
      toast.error("当前管线没有可查看证书")
      return
    }
    window.open(url, "_blank", "noopener,noreferrer")
  } catch (err) {
    console.error(err)
    toast.error("证书链接获取失败")
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
}

watch([candidateFilter, statusFilter, offset], () => loadPipelines())
onMounted(async () => {
  await Promise.all([loadPipelineCatalog(), loadPipelines()])
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">管线管理</h1>
        <p class="mt-2 text-slate-600">查看考生正在运行的管线实例、阶段状态和流转日志。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="selectedSummary" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="backToList">
          <ArrowLeft class="h-4 w-4" />
          返回列表
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="reloadSelected">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>
    </header>

    <template v-if="!selectedSummary">
      <div class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm md:grid-cols-[1fr_180px]">
        <label class="relative grid gap-2 text-sm font-bold">
          考生筛选
          <Search class="absolute bottom-3 left-3 h-4 w-4 text-slate-400" />
          <input v-model="candidateFilter" class="rounded-xl border border-slate-200 py-3 pl-9 pr-4" placeholder="Candidate ULID" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          ״̬ɸѡ
          <select v-model="statusFilter" class="rounded-xl border border-slate-200 px-4 py-3">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
      </div>

      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <h2 class="text-xl font-black">管线实例</h2>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ pipelines.length }}</span>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <button
          v-for="pipeline in pipelines"
          v-else
          :key="pipelineUlid(pipeline)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-5 text-left last:border-b-0 hover:bg-sky-50"
          type="button"
          @click="openPipeline(pipeline)"
        >
          <div>
            <div class="text-lg font-black">{{ pipelineDisplayName(pipeline) }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ pipeline.candidate_ulid || "-" }}</div>
            <div class="mt-2 text-sm text-slate-600">当前阶段：{{ pipeline.current_stage_ulid || "-" }}</div>
          </div>
          <div class="text-right">
            <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(pipeline.status))">{{ statusLabel(pipeline.status) }}</span>
            <div class="mt-3 text-xs text-slate-500">{{ formatDate(String(pipeline.started_at || pipeline.created_at || "")) }}</div>
          </div>
        </button>
        <div v-if="!loading && !pipelines.length" class="p-12 text-center text-slate-500">暂无管线实例</div>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - pageSize)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += pageSize">下一页</button>
        </div>
      </section>
    </template>

    <template v-else>
      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black">{{ pipelineDisplayName(selectedSummary) }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ selectedPipelineUlid }}</p>
          </div>
          <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(selectedStatus))">
            {{ statusLabel(selectedStatus) }}
          </span>
        </div>

        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs font-black uppercase text-slate-400">Candidate</div>
            <div class="mt-2 break-all text-sm font-bold">{{ selectedCandidateUlid || "-" }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs font-black uppercase text-slate-400">Pipeline CC</div>
            <div class="mt-2 break-all text-sm font-bold">{{ selectedPipelineCcUlid || "-" }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs font-black uppercase text-slate-400">当前阶段</div>
            <div class="mt-2 break-all text-sm font-bold">{{ selectedCurrentStageUlid || "-" }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs font-black uppercase text-slate-400">统计</div>
            <div class="mt-2 text-sm font-bold">阶段 {{ stages.length }} / 课程单元 {{ totalUnits }}</div>
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">管线操作</h2>
            <p class="mt-1 text-sm text-slate-500">仅展示 gprog 当前支持的人工干预动作。</p>
          </div>
          <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-50" type="button" :disabled="certificateLoading" @click="viewCertificate">
            <Eye class="h-4 w-4" />
            查看证书
          </button>
        </div>
        <div class="flex flex-wrap gap-3 p-5">
          <button
            class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-40"
            type="button"
            :disabled="!canTriggerNextStage"
            @click="openAction({ kind: 'trigger-next-stage', pipelineUlid: selectedPipelineUlid })"
          >
            <StepForward class="h-4 w-4" />
            推进下一阶段
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:opacity-40"
            type="button"
            :disabled="!selectedPipelineUlid"
            @click="openAction({ kind: 'terminate-pipeline', pipelineUlid: selectedPipelineUlid })"
          >
            <ShieldX class="h-4 w-4" />
            终止管线
          </button>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-200 p-5">
          <h2 class="text-xl font-black">阶段树</h2>
          <p class="mt-1 text-sm text-slate-500">展开阶段后可查看课程单元并执行强制完成/重置待预约。</p>
        </div>
        <div v-if="detailLoading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载详情...
        </div>
        <div v-else-if="!stages.length" class="p-12 text-center text-slate-500">暂无阶段数据</div>
        <div v-else class="divide-y divide-slate-100">
          <div v-for="stage in stages" :key="stageUlid(stage)" class="p-5">
            <button class="flex w-full items-center justify-between text-left" type="button" @click="toggleStage(stage)">
              <div class="flex items-center gap-3">
                <ChevronDown v-if="expandedStages[stageUlid(stage)]" class="h-5 w-5 text-slate-400" />
                <ChevronRight v-else class="h-5 w-5 text-slate-400" />
                <div>
                  <div class="font-black">{{ stageName(stage) }}</div>
                  <div class="mt-1 text-sm text-slate-500">{{ stageUlid(stage) }}</div>
                </div>
              </div>
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(stageStatus(stage), 'stage'))">
                {{ statusLabel(stageStatus(stage), "stage") }}
              </span>
            </button>

            <div v-if="expandedStages[stageUlid(stage)]" class="mt-4 grid gap-3">
              <div
                v-for="unit in courseUnits(stage)"
                :key="courseUnitUlid(unit)"
                class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
              >
                <div class="flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <div class="font-black">{{ unit.course_unit_cc_ulid || unit.course_unit_ulid || "课程单元" }}</div>
                    <div class="mt-1 text-sm text-slate-500">进度：{{ unit.course_progress || "-" }} · 考试：{{ unit.exam_ulid || "-" }} · 重试：{{ unit.retried_count ?? 0 }}</div>
                    <div class="mt-1 text-xs text-slate-400">{{ courseUnitUlid(unit) }}</div>
                  </div>
                  <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(courseUnitStatus(unit), 'unit'))">
                    {{ statusLabel(courseUnitStatus(unit), "unit") }}
                  </span>
                </div>
                <div class="mt-4 flex flex-wrap gap-2">
                  <button
                    class="rounded-xl border bg-white px-4 py-2 text-sm font-bold"
                    type="button"
                    @click="openAction({ kind: 'force-completed', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(unit) })"
                  >
                    强制完成
                  </button>
                  <button
                    class="rounded-xl border bg-white px-4 py-2 text-sm font-bold"
                    type="button"
                    @click="openAction({ kind: 'force-signup-exam', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(unit) })"
                  >
                    重置待预约
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
        <div class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex items-center justify-between border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">状态流转日志</h2>
              <p class="mt-1 text-sm text-slate-500">查看管线的每一次状态变化。</p>
            </div>
            <button class="rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="loadLogs()">加载日志</button>
          </div>
          <div v-if="logsLoading" class="p-10 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载...
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
            <div class="font-black">{{ log.to_status || "-" }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ log.entity_type || "-" }} · {{ log.entity_ulid || "-" }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(log.created_at || "")) }}</div>
          </button>
          <div v-if="!logsLoading && !logs.length" class="p-10 text-center text-slate-500">暂无日志</div>
          <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevLogs" @click="loadLogs(selectedPipelineUlid, Math.max(0, logOffset - logPageSize))">上一页</button>
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNextLogs" @click="loadLogs(selectedPipelineUlid, logOffset + logPageSize)">下一页</button>
          </div>
        </div>

        <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 class="mb-4 text-xl font-black">日志详情</h2>
          <div v-if="logDetailLoading" class="p-10 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载详情...
          </div>
          <pre v-else class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(logDetail || selectedLog || {}, null, 2) }}</pre>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="mb-4 text-xl font-black">完整管线详情</h2>
        <pre class="max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(detail || selectedSummary, null, 2) }}</pre>
      </section>
    </template>

    <div v-if="pendingAction" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-lg rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">确认操作</h2>
        <p class="mt-2 text-sm text-slate-600">即将执行：{{ pendingAction.kind }}</p>
        <textarea v-model="actionReason" class="mt-5 min-h-28 w-full rounded-xl border border-slate-200 p-4" placeholder="操作原因，可选" />
        <div class="mt-5 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" :disabled="actionLoading" @click="pendingAction = null">取消</button>
          <button class="rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="actionLoading" @click="submitAction">
            {{ actionLoading ? "提交中..." : "确认执行" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
