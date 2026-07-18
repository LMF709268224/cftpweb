<script setup lang="ts">
import { ArrowLeft, ArrowUpRight, BookOpen, Check, ChevronRight, Eye, FileBadge, Inbox, Layers, Loader2, RefreshCw, RotateCcw, ScrollText, Search, ShieldX, StepForward, TriangleAlert, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { toast } from "vue-sonner"
import CandidatePicker from "@/components/CandidatePicker.vue"
import JsonPreview from "@/components/JsonPreview.vue"
import UlidChip from "@/components/UlidChip.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type ActionKind = "trigger-next-stage" | "terminate-pipeline" | "force-completed" | "force-signup-exam"
type DetailTab = "overview" | "stages" | "units" | "certificateTasks" | "logs"
type DetailTabItem = { key: DetailTab; title: string; desc: string; count: number }
type StageStepState = "done" | "current" | "pending"

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

const route = useRoute()

const pipelines = ref<JsonRecord[]>([])
const pipelineNameByCc = ref<Record<string, string>>({})
const selectedSummary = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const logs = ref<JsonRecord[]>([])
const selectedLog = ref<JsonRecord | null>(null)
const logDetail = ref<JsonRecord | null>(null)
const certificateTasks = ref<JsonRecord[]>([])
const selectedCertificateTask = ref<JsonRecord | null>(null)
const certificateTaskDetail = ref<JsonRecord | null>(null)

const total = ref(0)
const loading = ref(false)
const detailLoading = ref(false)
const logsLoading = ref(false)
const logDetailLoading = ref(false)
const certificateTasksLoading = ref(false)
const certificateTaskDetailLoading = ref(false)
const actionLoading = ref(false)
const certificateLoading = ref(false)
const retryingCertificateTask = ref("")

const candidateFilter = ref("")
const statusFilter = ref("all")
const appliedCandidateFilter = ref("")
const appliedStatusFilter = ref("all")
const offset = ref(0)
const logOffset = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const logsHasMore = ref(false)
const logsTotal = ref(0)
const logsNextCursor = ref("")
const logsCursorStack = ref<string[]>([""])
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
const selectedCandidateName = computed(() => String(detailPipelineRecord.value.candidate_name || selectedSummary.value?.candidate_name || ""))
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
const currentStageNumber = computed(() => {
  const list = stages.value
  if (!list.length) return 0
  const index = list.findIndex((stage) => stageUlid(stage) && stageUlid(stage) === selectedCurrentStageUlid.value)
  if (index >= 0) return index + 1
  const done = list.filter((stage) => stageStepState(stage) === "done").length
  return Math.min(list.length, Math.max(1, done))
})
const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => hasMore.value)
const canPrevLogs = computed(() => logOffset.value > 0)
const canNextLogs = computed(() => logsHasMore.value)
const canViewCertificate = computed(() => Boolean(selectedPipelineUlid.value && selectedCandidateUlid.value && isCompletedPipelineStatus(selectedStatus.value)))
const canShowCertificateTasks = computed(() => isCertificateTaskPipelineStatus(selectedStatus.value))
const canTerminatePipeline = computed(() => Boolean(selectedPipelineUlid.value && !isCompletedPipelineStatus(selectedStatus.value) && !isCancelledPipelineStatus(selectedStatus.value)))
const canTriggerNextStage = computed(() => {
  const currentStageUlid = String(detailPipelineRecord.value.current_stage_ulid || "")
  const currentStage = stages.value.find((stage) => stageUlid(stage) === currentStageUlid) || stages.value[0]
  return String(selectedStatus.value ?? "") === "1" && String(stageRecord(currentStage).status ?? "") === "3"
})
const detailTabs = computed<DetailTabItem[]>(() => {
  const tabs: DetailTabItem[] = [
    { key: "overview", title: copy.value.tabs.overview.title, desc: copy.value.tabs.overview.desc, count: selectedSummary.value ? 1 : 0 },
    { key: "stages", title: copy.value.tabs.stages.title, desc: copy.value.tabs.stages.desc, count: stages.value.length },
    { key: "units", title: copy.value.tabs.units.title, desc: copy.value.tabs.units.desc, count: units.value.length },
  ]
  if (canShowCertificateTasks.value) {
    tabs.push({ key: "certificateTasks", title: copy.value.tabs.certificateTasks.title, desc: copy.value.tabs.certificateTasks.desc, count: certificateTasks.value.length })
  }
  tabs.push({ key: "logs", title: copy.value.tabs.logs.title, desc: copy.value.tabs.logs.desc, count: logs.value.length })
  return tabs
})

const statusOptions = computed(() => [
  { value: "all", label: copy.value.status.all },
  { value: "1", label: copy.value.status.pipeline.running },
  { value: "2", label: copy.value.status.pipeline.waitingFinalQualification },
  { value: "3", label: copy.value.status.pipeline.completed },
  { value: "4", label: copy.value.status.pipeline.issuingCertificate },
  { value: "5", label: copy.value.status.pipeline.cancelled },
])

function pipelineUlid(pipeline: JsonRecord | null | undefined) {
  return String(pickFirst(pipeline || {}, ["pipeline_ulid", "pipeline_id"]) || "")
}

function pipelineCcUlid(pipeline: JsonRecord | null | undefined) {
  return String(pickFirst(pipeline || {}, ["pipeline_cc_ulid", "pipeline_config_ulid"]) || "")
}

function certificateTaskUlid(task: JsonRecord | null | undefined) {
  return String(pickFirst(task || {}, ["task_ulid", "task_id", "id"]) || "")
}

function pipelineDisplayName(pipeline: JsonRecord | null | undefined) {
  const cc = pipelineCcUlid(pipeline)
  return pipelineNameByCc.value[cc] || String(pickFirst(pipeline || {}, ["name", "pipeline_name"]) || pipelineUlid(pipeline) || "Pipeline")
}

function normalizedPipelineStatus(value: unknown) {
  return String(value ?? "").trim().toUpperCase().replace(/^PIPELINE_STATUS_/, "")
}

function isCancelledPipelineStatus(value: unknown) {
  return ["5", "CANCELLED"].includes(normalizedPipelineStatus(value))
}

function isCompletedPipelineStatus(value: unknown) {
  return ["3", "COMPLETED"].includes(normalizedPipelineStatus(value))
}

function isCertificateTaskPipelineStatus(value: unknown) {
  return ["3", "4", "COMPLETED", "ISSUING_CERT"].includes(normalizedPipelineStatus(value))
}

function statusLabel(value: unknown, scope: "pipeline" | "stage" | "unit" = "pipeline") {
  const normalized = scope === "pipeline" ? normalizedPipelineStatus(value) : String(value ?? "")
  if (scope === "pipeline") {
    if (normalized === "1" || normalized === "RUNNING") return copy.value.status.pipeline.running
    if (normalized === "2" || normalized === "WAIT_FINAL_ELIG") return copy.value.status.pipeline.waitingFinalQualification
    if (normalized === "3" || normalized === "COMPLETED") return copy.value.status.pipeline.completed
    if (normalized === "4" || normalized === "ISSUING_CERT") return copy.value.status.pipeline.issuingCertificate
    if (normalized === "5" || normalized === "CANCELLED") return copy.value.status.pipeline.cancelled
  }
  if (scope === "stage") {
    if (normalized === "1") return copy.value.status.stage.waitingCandidate
    if (normalized === "2") return copy.value.status.stage.running
    if (normalized === "3") return copy.value.status.stage.completed
    if (normalized === "4") return copy.value.status.stage.terminated
  }
  if (scope === "unit") {
    if (normalized === "1") return copy.value.status.unit.notStarted
    if (normalized === "2") return copy.value.status.unit.readyForExamSignup
    if (normalized === "3") return copy.value.status.unit.studying
    if (normalized === "4") return copy.value.status.unit.examScheduled
    if (normalized === "5") return copy.value.status.unit.examFailed
    if (normalized === "6") return copy.value.status.unit.completed
  }
  return String(value || "-")
}

function actionLabel(kind: ActionKind) {
  return copy.value.actions[kind]
}

// Style-only helpers: map raw status codes to Aurora badge recipes (DESIGN.md).
function statusBadgeClass(value: unknown, scope: "pipeline" | "stage" | "unit" = "pipeline") {
  const normalized = scope === "pipeline" ? normalizedPipelineStatus(value) : String(value ?? "")
  if (scope === "pipeline") {
    if (normalized === "1" || normalized === "RUNNING") return "bg-primary-50 text-primary-700"
    if (normalized === "2" || normalized === "WAIT_FINAL_ELIG") return "bg-amber-50 text-amber-700"
    if (normalized === "3" || normalized === "COMPLETED") return "bg-emerald-50 text-emerald-700"
    if (normalized === "4" || normalized === "ISSUING_CERT") return "bg-primary-50 text-primary-700"
    if (normalized === "5" || normalized === "CANCELLED") return "bg-red-50 text-red-700"
  }
  if (scope === "stage") {
    if (normalized === "1") return "bg-amber-50 text-amber-700"
    if (normalized === "2") return "bg-primary-50 text-primary-700"
    if (normalized === "3") return "bg-emerald-50 text-emerald-700"
    if (normalized === "4") return "bg-red-50 text-red-700"
  }
  if (scope === "unit") {
    if (normalized === "1") return "bg-slate-100 text-slate-600"
    if (normalized === "2") return "bg-amber-50 text-amber-700"
    if (normalized === "3") return "bg-primary-50 text-primary-700"
    if (normalized === "4") return "bg-primary-50 text-primary-700"
    if (normalized === "5") return "bg-red-50 text-red-700"
    if (normalized === "6") return "bg-emerald-50 text-emerald-700"
  }
  return "bg-slate-100 text-slate-600"
}

function certificateTaskBadgeClass(value: unknown) {
  const normalized = String(value ?? "").trim().toUpperCase()
  if (normalized === "COMPLETED") return "bg-emerald-50 text-emerald-700"
  if (normalized === "FAILED" || normalized === "CANCELLED") return "bg-red-50 text-red-700"
  if (normalized === "PENDING") return "bg-amber-50 text-amber-700"
  if (["PROCESSING", "RUNNING", "RETRYING"].includes(normalized)) return "bg-primary-50 text-primary-700"
  return "bg-slate-100 text-slate-600"
}

function entityStatusLabel(entityType: unknown, value: unknown) {
  const normalizedType = String(entityType || "").toUpperCase()
  if (normalizedType === "STAGE") return statusLabel(value, "stage")
  if (normalizedType === "COURSE_UNIT") return statusLabel(value, "unit")
  return statusLabel(value, "pipeline")
}

function certificateTaskStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  const normalized = raw.toUpperCase()
  const labels = copy.value.certificateTasks.statusLabels as Record<string, string>
  return labels[normalized] || labels[raw] || raw || copy.value.certificateTasks.unknownStatus
}

function entityTypeLabel(value: unknown) {
  const normalizedType = String(value || "").toUpperCase()
  const labels = copy.value.entityTypes as Record<string, string>
  return labels[normalizedType] || String(value || "-")
}

function isDateField(key: string) {
  return key.endsWith("_at") || key.endsWith("_time") || key === "created_at" || key === "updated_at"
}

// Frontend-only helper: which keys render as UlidChip inside definition tiles (P2).
function isChipKey(key: string) {
  const normalized = key.toLowerCase()
  return normalized.endsWith("_ulid") || normalized.endsWith("_key") || normalized.endsWith("_hash") || ["task_id", "certificate_id", "request_id", "msg_fp"].includes(normalized)
}

function chipValue(value: string) {
  return value === "-" ? "" : value
}

function fieldLabel(group: "overview" | "stage" | "unit" | "log", key: string) {
  const labels: Record<string, string> = copy.value.fieldLabels
  const groupLabels = {
    overview: copy.value.overviewFieldLabels,
    stage: copy.value.stageFieldLabels,
    unit: copy.value.unitFieldLabels,
    log: copy.value.logFieldLabels,
  }[group] as Record<string, string>
  return groupLabels[key] || labels[key] || key
}

function readableValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "-"
  if (typeof value === "boolean") return value ? copy.value.yes : copy.value.no
  if (typeof value === "object") return JSON.stringify(value, null, 2)
  return String(value)
}

function detailFieldValue(group: "overview" | "stage" | "unit" | "log", key: string, value: unknown, record?: JsonRecord) {
  if (key === "status") {
    if (group === "stage") return statusLabel(value, "stage")
    if (group === "unit") return statusLabel(value, "unit")
    return statusLabel(value, "pipeline")
  }
  if (key === "from_status" || key === "to_status") return entityStatusLabel(record?.entity_type, value)
  if (key === "entity_type") return entityTypeLabel(value)
  if (isDateField(key)) return formatDate(value) || readableValue(value)
  return readableValue(value)
}

function certificateTaskFieldLabel(key: string) {
  const labels = copy.value.certificateTasks.fieldLabels as Record<string, string>
  const normalized = key.toLowerCase()
  return labels[key] || labels[normalized] || fieldLabel("overview", normalized)
}

function certificateTaskFieldValue(key: string, value: unknown) {
  const normalized = key.toLowerCase()
  if (normalized === "status") return certificateTaskStatusLabel(value)
  if (isDateField(normalized)) return formatDate(value) || readableValue(value)
  return readableValue(value)
}

function certificateTaskEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: certificateTaskFieldLabel(key),
    value: certificateTaskFieldValue(key, value),
  }))
}

function detailEntries(group: "overview" | "stage" | "unit" | "log", record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: fieldLabel(group, key),
    value: detailFieldValue(group, key, value, record),
  }))
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

// Frontend-only helper: map stage status to Stepper tri-state (DESIGN-PRODUCT.md (d)).
function stageStepState(stage: JsonRecord | null | undefined): StageStepState {
  const status = String(stageStatus(stage) ?? "")
  if (status === "3") return "done"
  if (status === "2") return "current"
  if (stageUlid(stage) && stageUlid(stage) === selectedCurrentStageUlid.value) return "current"
  return "pending"
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
    const data = await apiClient<JsonRecord>("/api/pipelines?page_size=100")
    total.value = Number(data.total) || 0
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
    const currentPage = Math.floor(offset.value / pageSize) + 1
    const params = new URLSearchParams({
      page_size: String(pageSize),
    })

    let cursor = ""

    if (currentPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (currentPage < lastPage.value) {

      cursor = prevCursor.value


    }



    if (cursor) params.set("cursor", cursor)


    if (candidateFilter.value.trim()) params.set("candidate_ulid", candidateFilter.value.trim())
    if (statusFilter.value !== "all") params.set("status", statusFilter.value)

    const isValidUlid = (id: string) => /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/i.test(id)
    if (candidateFilter.value.trim() && !isValidUlid(candidateFilter.value.trim())) {
      toast.error(copy.value.filters.invalidCandidateUlid)
      pipelines.value = []
      hasMore.value = false
      nextCursor.value = ""
      prevCursor.value = ""
      return
    }

    const data = await apiClient<JsonRecord>(`/api/prog/pipelines?${params}`)
    const list = Array.isArray(data.pipelines) ? data.pipelines : []

    pipelines.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    const isBackward = currentPage < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = currentPage
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = currentPage
  } catch (err) {
    console.error(err)
    pipelines.value = []
    hasMore.value = false
    nextCursor.value = ""
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
  certificateTasks.value = []
  selectedCertificateTask.value = null
  certificateTaskDetail.value = null
  logsCursorStack.value = [""]
  logsNextCursor.value = ""
  logsHasMore.value = false
  activeTab.value = "overview"
  selectedStageIndex.value = 0
  selectedUnitKey.value = ""
  await loadDetail(pipelineUlid(pipeline))
  await Promise.all([loadLogs(pipelineUlid(pipeline), 0), loadCertificateTasks()])
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
    const currentPage = Math.floor(targetOffset / logPageSize) + 1
    const params = new URLSearchParams({
      page_size: String(logPageSize),
    })
    const cursor = logsCursorStack.value[currentPage - 1] || ""
    if (cursor) params.set("cursor", cursor)
    const data = await apiClient<JsonRecord>(`/api/prog/pipelines/${encodeURIComponent(pipelineId)}/logs?${params}`)
    const list = Array.isArray(data.logs) ? data.logs : []

    logs.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    logsHasMore.value = Boolean(data.has_more)
    logsTotal.value = Number(data.total) || 0
    logsNextCursor.value = String(data.next_cursor || "")
    logsCursorStack.value = logsCursorStack.value.slice(0, currentPage)
    logsCursorStack.value[currentPage] = logsNextCursor.value
    logOffset.value = targetOffset
    selectedLog.value = logs.value[0] || null
    if (selectedLog.value) await loadLogDetail(String(selectedLog.value.transition_ulid || ""))
  } catch (err) {
    console.error(err)
    logs.value = []
    selectedLog.value = null
    logDetail.value = null
    logsHasMore.value = false
    logsNextCursor.value = ""
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

async function loadCertificateTasks() {
  if (!selectedCandidateUlid.value || !canShowCertificateTasks.value) {
    certificateTasks.value = []
    selectedCertificateTask.value = null
    certificateTaskDetail.value = null
    certificateTasksLoading.value = false
    return
  }
  certificateTasksLoading.value = true
  try {
    const params = new URLSearchParams({
      candidate_ulid: selectedCandidateUlid.value,
      limit: "20",
      offset: "0",
    })
    if (selectedPipelineUlid.value) params.set("pipeline_ulid", selectedPipelineUlid.value)
    const data = await apiClient<JsonRecord>(`/api/prog/certificate-tasks?${params}`)
    const list = Array.isArray(data.tasks) ? data.tasks : []

    certificateTasks.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selectedCertificateTask.value = certificateTasks.value[0] || null
    if (selectedCertificateTask.value) await loadCertificateTaskDetail(certificateTaskUlid(selectedCertificateTask.value))
  } catch (err) {
    console.error(err)
    certificateTasks.value = []
    selectedCertificateTask.value = null
    certificateTaskDetail.value = null
    toast.error(copy.value.toasts.certificateTasksLoadFailed)
  } finally {
    certificateTasksLoading.value = false
  }
}

async function loadCertificateTaskDetail(taskUlid: string) {
  if (!taskUlid) {
    certificateTaskDetail.value = null
    return
  }
  certificateTaskDetailLoading.value = true
  try {
    certificateTaskDetail.value = await apiClient<JsonRecord>(`/api/prog/certificate-tasks/${encodeURIComponent(taskUlid)}`)
  } catch (err) {
    console.error(err)
    certificateTaskDetail.value = null
    toast.error(copy.value.toasts.certificateTaskDetailLoadFailed)
  } finally {
    certificateTaskDetailLoading.value = false
  }
}

async function retryCertificateTask(task: JsonRecord) {
  const taskUlid = certificateTaskUlid(task)
  if (!taskUlid) return
  retryingCertificateTask.value = taskUlid
  try {
    const data = await apiClient<JsonRecord>(`/api/prog/certificate-tasks/${encodeURIComponent(taskUlid)}/retry`, { method: "POST" })
    toast.success(String(data.message || copy.value.toasts.certificateTaskRetried))
    await reloadSelected()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.certificateTaskRetryFailed))
  } finally {
    retryingCertificateTask.value = ""
  }
}

async function reloadSelected() {
  await loadPipelines()
  if (selectedPipelineUlid.value) {
    await loadDetail(selectedPipelineUlid.value)
    await Promise.all([loadLogs(selectedPipelineUlid.value, logOffset.value), loadCertificateTasks()])
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
    toast.error(apiErrorMessage(err, copy.value.toasts.actionFailed))
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
    toast.error(apiErrorMessage(err, copy.value.toasts.certificateUrlFailed))
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
  certificateTasks.value = []
  selectedCertificateTask.value = null
  certificateTaskDetail.value = null
  activeTab.value = "overview"
}

function resetPipelineSearchState() {
  selectedSummary.value = null
  lastPage.value = 1
  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
}

function searchPipelines() {
  appliedCandidateFilter.value = candidateFilter.value.trim()
  appliedStatusFilter.value = statusFilter.value
  resetPipelineSearchState()
  if (offset.value !== 0) {
    offset.value = 0
    return
  }
  void loadPipelines()
}

function setStatusFilter(value: string) {
  statusFilter.value = value
}

function onCandidateChange(value: string) {
  candidateFilter.value = value
  searchPipelines()
}

watch(statusFilter, () => searchPipelines())
watch(offset, () => loadPipelines())

watch(canShowCertificateTasks, (visible) => {
  if (visible) return
  certificateTasks.value = []
  selectedCertificateTask.value = null
  certificateTaskDetail.value = null
  if (activeTab.value === "certificateTasks") activeTab.value = "overview"
})

onMounted(async () => {
  const deepLinkCandidate = String(route.query.candidate || "")
  if (deepLinkCandidate) {
    candidateFilter.value = deepLinkCandidate
    appliedCandidateFilter.value = deepLinkCandidate
  }
  await Promise.all([loadPipelineCatalog(), loadPipelines()])
})
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="selectedSummary" class="ui-btn ui-btn-secondary" type="button" @click="backToList">
          <ArrowLeft class="h-4 w-4 text-slate-400" />
          {{ copy.backToList }}
        </button>
        <button class="ui-btn ui-btn-primary" type="button" @click="reloadSelected">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <!-- Filters -->
    <section class="ui-card mt-6 p-4 md:p-5">
      <div class="grid gap-4">
        <div class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.status }}</span>
          <div class="flex flex-wrap items-center gap-2">
            <button
              v-for="chip in statusOptions"
              :key="chip.value"
              type="button"
              class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
              :class="statusFilter === chip.value
                ? 'bg-primary-600 text-white shadow-sm'
                : 'border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900'"
              :aria-pressed="statusFilter === chip.value"
              @click="setStatusFilter(chip.value)"
            >
              {{ chip.label }}
            </button>
          </div>
        </div>
        <div class="grid gap-4 lg:grid-cols-[minmax(0,380px)_auto] lg:items-end">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.filters.candidate }}</span>
            <CandidatePicker v-model="candidateFilter" @update:model-value="onCandidateChange" />
          </label>
          <button class="ui-btn ui-btn-primary lg:self-end" type="button" @click="searchPipelines">
            <Search class="h-4 w-4" />
            {{ copy.filters.search }}
          </button>
        </div>
      </div>
    </section>

    <!-- Pipeline list -->
    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
      </div>
      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="px-4 py-4 md:px-6">
          <div class="skeleton h-4 w-2/5" />
          <div class="skeleton mt-2.5 h-3.5 w-4/5" />
          <div class="skeleton mt-2 h-3.5 w-3/5" />
        </div>
      </div>
      <template v-else-if="pipelines.length">
        <div class="hidden grid-cols-[minmax(0,1fr)_150px_180px_110px] gap-4 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid md:px-6">
          <span>{{ copy.columns.pipeline }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
          <span class="text-right">{{ copy.columns.startedAt }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <button
            v-for="pipeline in pipelines"
            :key="pipelineUlid(pipeline)"
            class="ui-table-row flex w-full flex-col gap-3 px-4 py-4 text-left md:grid md:grid-cols-[minmax(0,1fr)_150px_180px_110px] md:items-center md:gap-4 md:px-6 md:py-3.5"
            :class="pipelineUlid(pipeline) === selectedPipelineUlid ? 'bg-primary-50/60' : ''"
            type="button"
            @click="openPipeline(pipeline)"
          >
            <div class="min-w-0">
              <div class="break-words text-sm font-semibold text-ink md:truncate">{{ pipeline.candidate_name || "-" }}</div>
              <div class="mt-0.5 break-words text-sm text-slate-600">{{ pipelineDisplayName(pipeline) }}</div>
              <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5">
                <span class="inline-flex items-center gap-1 text-xs text-slate-400">
                  {{ copy.candidatePrefix }}
                  <UlidChip :value="String(pipeline.candidate_ulid || pipeline.candidate_id || '')" />
                </span>
                <span class="inline-flex items-center gap-1 text-xs text-slate-400">
                  {{ copy.columns.pipeline }}
                  <UlidChip :value="pipelineUlid(pipeline)" />
                </span>
                <span class="inline-flex items-center gap-1 text-xs text-slate-400">
                  {{ copy.currentStagePrefix }}
                  <UlidChip :value="String(pipeline.current_stage_ulid || '')" />
                </span>
              </div>
            </div>
            <span class="flex items-center justify-between gap-4 md:block md:text-center">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
              <span class="ui-badge" :class="statusBadgeClass(pipeline.status)">{{ statusLabel(pipeline.status) }}</span>
            </span>
            <span class="flex items-center justify-between gap-4 md:block md:text-right">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.startedAt }}</span>
              <span class="tnum text-sm text-slate-600">{{ formatDate(String(pipeline.started_at || pipeline.created_at || "")) }}</span>
            </span>
            <span class="inline-flex items-center gap-1 text-sm font-medium text-primary-700 transition-colors hover:text-primary-600 md:justify-end">
              {{ copy.viewDetails }}
              <ChevronRight class="h-4 w-4" />
            </span>
          </button>
        </div>
      </template>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <Inbox class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-end gap-2 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - pageSize)">{{ copy.prev }}</button>
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="offset += pageSize">{{ copy.next }}</button>
      </div>
    </section>

    <!-- Detail modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="selectedSummary" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1200px]">
        <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
            <div class="mt-1.5"><UlidChip :value="selectedPipelineUlid" /></div>
          </div>
          <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="backToList">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
          <main class="min-w-0 space-y-4 md:space-y-6">
        <!-- Hero: candidate-first summary + entity link + stage stepper -->
        <section class="ui-card p-4 md:p-5">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0">
              <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ selectedCandidateName || pipelineDisplayName(selectedSummary) }}</h2>
              <p v-if="selectedCandidateName" class="mt-0.5 break-words text-sm text-slate-500">{{ pipelineDisplayName(selectedSummary) }}</p>
              <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5">
                <RouterLink
                  v-if="selectedCandidateUlid"
                  :to="{ path: '/exams', query: { candidate: selectedCandidateUlid } }"
                  class="inline-flex items-center gap-1 text-sm text-slate-600 underline decoration-slate-300 decoration-dashed underline-offset-4 transition-colors hover:text-primary-700 hover:decoration-primary-400"
                  @click.stop
                >
                  {{ t.common.openLink }}
                  <ArrowUpRight class="h-3.5 w-3.5 text-slate-400" />
                </RouterLink>
                <span v-if="stages.length" class="tnum text-xs text-slate-400">{{ t.common.progress }} {{ currentStageNumber }}/{{ stages.length }}</span>
              </div>
            </div>
            <span class="ui-badge shrink-0" :class="statusBadgeClass(selectedStatus)">
              {{ statusLabel(selectedStatus) }}
            </span>
          </div>

          <!-- Horizontal stage stepper (P3) -->
          <ol v-if="stages.length" class="mt-5 flex items-start gap-0 overflow-x-auto pb-1">
            <li v-for="(stage, index) in stages" :key="stageUlid(stage) || index" class="flex min-w-0 flex-1 items-start">
              <div class="flex min-w-0 flex-col items-center gap-1.5 px-1 text-center" style="min-width: 72px">
                <span
                  v-if="stageStepState(stage) === 'done'"
                  class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-600 text-white"
                ><Check class="h-3.5 w-3.5" /></span>
                <span
                  v-else-if="stageStepState(stage) === 'current'"
                  class="tnum flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-50 text-xs font-medium text-primary-700 ring-4 ring-primary-500/15"
                >{{ index + 1 }}</span>
                <span
                  v-else
                  class="tnum flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-100 text-xs font-medium text-slate-400"
                >{{ index + 1 }}</span>
                <span
                  class="w-full truncate text-xs"
                  :class="stageStepState(stage) === 'current' ? 'font-medium text-primary-700' : stageStepState(stage) === 'done' ? 'text-slate-600' : 'text-slate-400'"
                  :title="stageName(stage)"
                >{{ stageName(stage) }}</span>
              </div>
              <div
                v-if="index < stages.length - 1"
                class="mt-3 h-0.5 min-w-4 flex-1 rounded-full"
                :class="stageStepState(stage) === 'done' ? 'bg-primary-600' : 'bg-slate-200'"
              />
            </li>
          </ol>

          <div class="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-2xl bg-slate-50 p-4">
              <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summary.candidate }}</div>
              <div class="mt-1.5"><UlidChip :value="selectedCandidateUlid" /></div>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summary.pipelineCc }}</div>
              <div class="mt-1.5"><UlidChip :value="selectedPipelineCcUlid" /></div>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summary.currentStage }}</div>
              <div class="mt-1.5"><UlidChip :value="selectedCurrentStageUlid" /></div>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summary.count }}</div>
              <div class="tnum mt-1 text-sm font-medium text-ink">{{ copy.summary.countText(stages.length, totalUnits) }}</div>
            </div>
          </div>
        </section>

        <section class="ui-card overflow-hidden">
            <div class="p-4 md:p-5">
              <h3 class="text-base font-semibold text-ink">{{ copy.levelTitle }}</h3>
              <p class="mt-1 text-sm text-slate-500">{{ copy.levelDescription }}</p>
            </div>

            <div class="flex gap-5 overflow-x-auto border-b border-hairline px-4 md:px-5">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="ui-tab"
                :class="activeTab === tab.key ? 'ui-tab-active' : ''"
                type="button"
                @click="activeTab = tab.key"
              >
                {{ tab.title }}
                <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
              </button>
            </div>

            <section class="max-h-none min-w-0 overflow-y-auto md:max-h-[60vh]">
              <div v-if="detailLoading" class="space-y-3 p-4 md:p-5">
                <div class="skeleton h-12 w-full" />
                <div class="skeleton h-12 w-4/5" />
                <div class="skeleton h-12 w-3/5" />
              </div>

              <div v-else-if="activeTab === 'overview'" class="space-y-4 p-4 md:p-5">
                <div class="grid gap-3 md:grid-cols-2">
                  <template v-for="entry in detailEntries('overview', detailPipelineRecord)" :key="entry.key">
                    <JsonPreview
                      v-if="entry.key === 'course_selection_json'"
                      class="md:col-span-2"
                      :title="entry.label"
                      :text="entry.value"
                      :copy-label="copy.copyJson"
                      :copied-label="copy.copiedJson"
                      :copied-message="copy.toasts.jsonCopied"
                      :copy-error-message="copy.toasts.jsonCopyFailed"
                      max-height="260px"
                    />
                    <div v-else class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium text-slate-400">{{ entry.label }}</div>
                      <div v-if="isChipKey(entry.key)" class="mt-1.5"><UlidChip :value="chipValue(entry.value)" /></div>
                      <div v-else class="tnum mt-1 whitespace-pre-wrap break-all text-sm font-medium text-ink">{{ entry.value }}</div>
                    </div>
                  </template>
                </div>
              </div>

              <div v-else-if="activeTab === 'stages'" class="grid lg:grid-cols-[320px_minmax(0,1fr)]">
                <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                  <div class="border-b border-hairline p-4">
                    <div class="text-sm font-semibold text-ink">{{ copy.stageListTitle }}</div>
                    <div class="mt-0.5 text-xs text-slate-500">{{ copy.stageListDescription }}</div>
                  </div>
                  <button
                    v-for="(stage, index) in stages"
                    :key="stageUlid(stage) || index"
                    class="ui-table-row w-full border-b border-hairline p-4 text-left"
                    :class="selectedStageIndex === index ? 'bg-primary-50/60' : ''"
                    type="button"
                    @click="selectedStageIndex = index"
                  >
                    <div class="text-sm font-semibold text-ink">{{ stageName(stage) }}</div>
                    <div class="tnum mt-1 text-xs text-slate-500">{{ copy.courseUnitCount(courseUnits(stage).length) }}</div>
                    <div class="mt-1.5"><UlidChip :value="stageUlid(stage)" /></div>
                    <span class="ui-badge mt-3" :class="statusBadgeClass(stageStatus(stage), 'stage')">
                      {{ statusLabel(stageStatus(stage), "stage") }}
                    </span>
                  </button>
                  <div v-if="!stages.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <Layers class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.noStages }}</p>
                  </div>
                </div>
                <div class="space-y-4 p-4 md:p-5">
                  <template v-if="selectedStage">
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in detailEntries('stage', stageRecord(selectedStage))" :key="entry.key" class="rounded-2xl bg-slate-50 p-4">
                        <div class="text-xs font-medium text-slate-400">{{ entry.label }}</div>
                        <div v-if="isChipKey(entry.key)" class="mt-1.5"><UlidChip :value="chipValue(entry.value)" /></div>
                        <div v-else class="tnum mt-1 whitespace-pre-wrap break-all text-sm font-medium text-ink">{{ entry.value }}</div>
                      </div>
                    </div>
                  </template>
                  <div v-else class="flex items-center justify-center px-6 py-12 text-sm text-slate-400">{{ copy.selectStage }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'units'" class="grid lg:grid-cols-[360px_minmax(0,1fr)]">
                <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                  <div class="border-b border-hairline p-4">
                    <div class="text-sm font-semibold text-ink">{{ copy.unitListTitle }}</div>
                    <div class="mt-0.5 text-xs text-slate-500">{{ copy.unitListDescription }}</div>
                  </div>
                  <button
                    v-for="item in units"
                    :key="item.key"
                    class="ui-table-row w-full border-b border-hairline p-4 text-left"
                    :class="selectedUnitKey === item.key ? 'bg-primary-50/60' : ''"
                    type="button"
                    @click="selectedUnitKey = item.key"
                  >
                    <div class="text-sm font-semibold text-ink">{{ copy.unitFallback(item.unitIndex + 1) }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ copy.parentStagePrefix }}{{ stageName(item.stage) }}</div>
                    <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
                      <UlidChip :value="String(item.unit.course_unit_cc_ulid || '')" />
                      <UlidChip :value="courseUnitUlid(item.unit)" />
                    </div>
                    <span class="ui-badge mt-3" :class="statusBadgeClass(courseUnitStatus(item.unit), 'unit')">
                      {{ statusLabel(courseUnitStatus(item.unit), "unit") }}
                    </span>
                  </button>
                  <div v-if="!units.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <BookOpen class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.noUnits }}</p>
                  </div>
                </div>
                <div class="space-y-4 p-4 md:p-5">
                  <template v-if="selectedUnit">
                    <div class="rounded-2xl bg-primary-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-primary-700">{{ copy.parentStage }}</div>
                      <div class="mt-1.5 flex flex-wrap items-center gap-2">
                        <span class="text-sm font-medium text-ink">{{ stageName(selectedUnit.stage) }}</span>
                        <UlidChip :value="stageUlid(selectedUnit.stage)" />
                      </div>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in detailEntries('unit', selectedUnit.unit)" :key="entry.key" class="rounded-2xl bg-slate-50 p-4">
                        <div class="text-xs font-medium text-slate-400">{{ entry.label }}</div>
                        <div v-if="isChipKey(entry.key)" class="mt-1.5"><UlidChip :value="chipValue(entry.value)" /></div>
                        <div v-else class="tnum mt-1 whitespace-pre-wrap break-all text-sm font-medium text-ink">{{ entry.value }}</div>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-2">
                      <button
                        class="ui-btn ui-btn-secondary w-full sm:w-auto"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-completed', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.forceCompleted }}
                      </button>
                      <button
                        class="ui-btn ui-btn-secondary w-full sm:w-auto"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-signup-exam', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.resetExamSignup }}
                      </button>
                    </div>
                  </template>
                  <div v-else class="flex items-center justify-center px-6 py-12 text-sm text-slate-400">{{ copy.selectUnit }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'certificateTasks'" class="grid lg:grid-cols-[380px_minmax(0,1fr)]">
                <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                  <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                    <div class="min-w-0">
                      <div class="text-sm font-semibold text-ink">{{ copy.certificateTasks.listTitle }}</div>
                      <div class="mt-0.5 text-xs text-slate-500">{{ copy.certificateTasks.listDescription }}</div>
                    </div>
                    <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click="loadCertificateTasks">{{ copy.load }}</button>
                  </div>
                  <div v-if="certificateTasksLoading" class="space-y-2.5 p-4">
                    <div class="skeleton h-4 w-2/5" />
                    <div class="skeleton h-3.5 w-4/5" />
                    <div class="skeleton h-5 w-16 rounded-full" />
                  </div>
                  <button
                    v-for="task in certificateTasks"
                    v-else
                    :key="certificateTaskUlid(task)"
                    class="ui-table-row w-full border-b border-hairline p-4 text-left last:border-b-0"
                    :class="certificateTaskUlid(selectedCertificateTask) === certificateTaskUlid(task) ? 'bg-primary-50/60' : ''"
                    type="button"
                    @click="selectedCertificateTask = task; loadCertificateTaskDetail(certificateTaskUlid(task))"
                  >
                    <div class="tnum break-all text-sm font-semibold text-ink">{{ task.degree_no || copy.certificateTasks.taskFallback }}</div>
                    <div class="mt-1.5"><UlidChip :value="certificateTaskUlid(task)" /></div>
                    <div class="mt-3 flex flex-wrap items-center gap-2">
                      <span class="ui-badge" :class="certificateTaskBadgeClass(task.status)">
                        {{ certificateTaskStatusLabel(task.status) }}
                      </span>
                      <span class="tnum text-xs text-slate-400">{{ formatDate(String(task.created_at || "")) }}</span>
                    </div>
                  </button>
                  <div v-if="!certificateTasksLoading && !certificateTasks.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <FileBadge class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.certificateTasks.empty }}</p>
                  </div>
                </div>
                <div class="space-y-4 p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0">
                      <h4 class="text-base font-semibold text-ink">{{ copy.certificateTasks.detailTitle }}</h4>
                      <div class="mt-1.5"><UlidChip :value="certificateTaskUlid(selectedCertificateTask)" /></div>
                    </div>
                    <button
                      class="ui-btn ui-btn-primary w-full sm:w-auto"
                      type="button"
                      :disabled="!selectedCertificateTask || retryingCertificateTask === certificateTaskUlid(selectedCertificateTask)"
                      @click="selectedCertificateTask && retryCertificateTask(selectedCertificateTask)"
                    >
                      <Loader2 v-if="retryingCertificateTask === certificateTaskUlid(selectedCertificateTask)" class="h-4 w-4 animate-spin" />
                      <RotateCcw v-else class="h-4 w-4" />
                      {{ retryingCertificateTask === certificateTaskUlid(selectedCertificateTask) ? copy.certificateTasks.retrying : copy.certificateTasks.retry }}
                    </button>
                  </div>
                  <div v-if="certificateTaskDetailLoading" class="space-y-3">
                    <div class="skeleton h-12 w-full" />
                    <div class="skeleton h-12 w-4/5" />
                    <div class="skeleton h-12 w-3/5" />
                  </div>
                  <template v-else-if="selectedCertificateTask">
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in certificateTaskEntries(asRecord((certificateTaskDetail || {}).summary || selectedCertificateTask))" :key="entry.key" class="rounded-2xl bg-slate-50 p-4">
                        <div class="text-xs font-medium text-slate-400">{{ entry.label }}</div>
                        <div v-if="isChipKey(entry.key)" class="mt-1.5"><UlidChip :value="chipValue(entry.value)" /></div>
                        <div v-else class="tnum mt-1 whitespace-pre-wrap break-all text-sm font-medium text-ink">{{ entry.value }}</div>
                      </div>
                    </div>
                    <div v-if="certificateTaskDetail?.error_message" class="rounded-2xl bg-red-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-red-700">{{ copy.certificateTasks.errorMessage }}</div>
                      <div class="mt-2 whitespace-pre-wrap break-words text-sm text-red-700">{{ certificateTaskDetail.error_message }}</div>
                    </div>
                    <div v-if="certificateTaskDetail?.template_params" class="overflow-hidden rounded-2xl border border-hairline">
                      <div class="border-b border-hairline px-4 py-3 text-sm font-semibold text-ink">{{ copy.certificateTasks.templateParams }}</div>
                      <pre class="max-h-[260px] overflow-auto bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ certificateTaskDetail.template_params }}</pre>
                    </div>
                    <JsonPreview
                      :title="copy.certificateTasks.detailTitle"
                      :value="certificateTaskDetail || selectedCertificateTask || {}"
                      :copy-label="copy.copyJson"
                      :copied-label="copy.copiedJson"
                      :copied-message="copy.toasts.jsonCopied"
                      :copy-error-message="copy.toasts.jsonCopyFailed"
                      max-height="360px"
                    />
                  </template>
                  <div v-else class="flex items-center justify-center px-6 py-12 text-sm text-slate-400">{{ copy.certificateTasks.selectTask }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'logs'" class="grid lg:grid-cols-[380px_minmax(0,1fr)]">
                <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                  <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                    <div class="min-w-0">
                      <div class="text-sm font-semibold text-ink">{{ copy.logListTitle }}</div>
                      <div class="mt-0.5 text-xs text-slate-500">{{ copy.logListDescription }}</div>
                    </div>
                    <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click="loadLogs()">{{ copy.load }}</button>
                  </div>
                  <div v-if="logsLoading" class="space-y-2.5 p-4">
                    <div class="skeleton h-4 w-2/5" />
                    <div class="skeleton h-3.5 w-4/5" />
                    <div class="skeleton h-3.5 w-3/5" />
                  </div>
                  <!-- Vertical timeline (P3): newest on top, primary dot on latest -->
                  <ol v-else class="px-2 py-3">
                    <li v-for="(log, index) in logs" :key="String(log.transition_ulid)">
                      <button
                        class="flex w-full gap-3 rounded-xl px-3 py-2.5 text-left transition-colors"
                        :class="selectedLog === log ? 'bg-primary-50/60' : 'hover:bg-slate-50'"
                        type="button"
                        @click="selectedLog = log; loadLogDetail(String(log.transition_ulid || ''))"
                      >
                        <span class="flex flex-col items-center">
                          <span
                            class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
                            :class="index === 0 ? 'bg-primary-600 ring-4 ring-primary-500/15' : 'bg-slate-300'"
                          />
                          <span v-if="index < logs.length - 1" class="mt-1 w-px flex-1 bg-hairline" />
                        </span>
                        <span class="min-w-0 flex-1">
                          <span class="block text-sm font-medium" :class="index === 0 ? 'text-ink' : 'text-slate-600'">
                            {{ entityStatusLabel(log.entity_type, log.from_status) }} → {{ entityStatusLabel(log.entity_type, log.to_status) }}
                          </span>
                          <span class="tnum mt-0.5 block text-xs text-slate-400">{{ formatDate(String(log.created_at || "")) }}</span>
                          <span class="mt-1 flex flex-wrap items-center gap-1.5">
                            <span class="text-xs text-slate-500">{{ entityTypeLabel(log.entity_type) }}</span>
                            <UlidChip :value="String(log.entity_ulid || '')" />
                          </span>
                        </span>
                      </button>
                    </li>
                  </ol>
                  <div v-if="!logsLoading && !logs.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <ScrollText class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.noLogs }}</p>
                  </div>
                  <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center">
                    <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ Math.floor(logOffset / logPageSize) + 1 }} / {{ Math.max(1, Math.ceil(logsTotal / logPageSize)) }}</span>
                    <div class="flex flex-col gap-2 sm:flex-row">
                      <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrevLogs" @click="loadLogs(selectedPipelineUlid, Math.max(0, logOffset - logPageSize))">{{ copy.prev }}</button>
                      <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNextLogs" @click="loadLogs(selectedPipelineUlid, logOffset + logPageSize)">{{ copy.next }}</button>
                    </div>
                  </div>
                </div>
                <div class="space-y-4 p-4 md:p-5">
                  <h4 class="text-base font-semibold text-ink">{{ copy.logDetailTitle }}</h4>
                  <div v-if="logDetailLoading" class="space-y-3">
                    <div class="skeleton h-12 w-full" />
                    <div class="skeleton h-12 w-4/5" />
                    <div class="skeleton h-12 w-3/5" />
                  </div>
                  <template v-else>
                    <div v-if="selectedLog" class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in detailEntries('log', asRecord((logDetail || selectedLog).summary || selectedLog))" :key="entry.key" class="rounded-2xl bg-slate-50 p-4">
                        <div class="text-xs font-medium text-slate-400">{{ entry.label }}</div>
                        <div v-if="isChipKey(entry.key)" class="mt-1.5"><UlidChip :value="chipValue(entry.value)" /></div>
                        <div v-else class="tnum mt-1 whitespace-pre-wrap break-all text-sm font-medium text-ink">{{ entry.value }}</div>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </section>
        </section>

        <!-- Bottom action bar: manual operations (P6) -->
        <section class="ui-card p-4 md:p-5">
          <h3 class="text-sm font-semibold text-ink">{{ copy.manualActionsTitle }}</h3>
          <p class="mt-1 text-xs text-slate-500">{{ copy.manualActionsDescription }}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <span class="inline-flex w-full sm:w-auto" :title="!canTriggerNextStage ? t.common.reasonUnavailable : undefined">
              <button
                class="ui-btn ui-btn-primary w-full sm:w-auto"
                type="button"
                :disabled="!canTriggerNextStage"
                @click="openAction({ kind: 'trigger-next-stage', pipelineUlid: selectedPipelineUlid })"
              >
                <StepForward class="h-4 w-4" />
                {{ copy.actions["trigger-next-stage"] }}
              </button>
            </span>
            <button
              v-if="canViewCertificate"
              class="ui-btn ui-btn-secondary w-full sm:w-auto"
              type="button"
              :disabled="certificateLoading"
              @click="viewCertificate"
            >
              <Loader2 v-if="certificateLoading" class="h-4 w-4 animate-spin" />
              <Eye v-else class="h-4 w-4 text-slate-400" />
              {{ copy.viewCertificate }}
            </button>
          </div>
        </section>

        <!-- Danger zone (P6) -->
        <section class="rounded-2xl border border-danger-200 bg-danger-50/40 p-4 md:p-5">
          <h3 class="flex items-center gap-2 text-sm font-semibold text-danger-600">
            <TriangleAlert class="h-4 w-4" />
            {{ t.common.dangerZone }}
          </h3>
          <div class="mt-3 flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-end">
            <span class="inline-flex shrink-0" :title="!canTerminatePipeline ? t.common.reasonUnavailable : undefined">
              <button
                class="ui-btn ui-btn-danger w-full sm:w-auto"
                type="button"
                :disabled="!canTerminatePipeline"
                @click="openAction({ kind: 'terminate-pipeline', pipelineUlid: selectedPipelineUlid })"
              >
                <ShieldX class="h-4 w-4" />
                {{ copy.actions["terminate-pipeline"] }}
              </button>
            </span>
          </div>
        </section>
          </main>
        </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <!-- Confirm dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="pendingAction" class="ui-modal-mask z-50">
          <section class="ui-modal-panel md:max-w-md">
            <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.confirmTitle }}</h2>
            </div>
            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <p class="text-sm text-slate-600">{{ copy.confirmDescription(actionLabel(pendingAction.kind)) }}</p>
              <textarea v-model="actionReason" class="ui-input mt-4 h-auto min-h-28 py-2.5" :placeholder="copy.reasonPlaceholder" />
            </main>
            <div class="flex flex-col-reverse items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="actionLoading" @click="pendingAction = null">{{ copy.cancel }}</button>
              <button class="ui-btn" :class="pendingAction.kind === 'terminate-pipeline' ? 'ui-btn-danger' : 'ui-btn-primary'" type="button" :disabled="actionLoading" @click="submitAction">
                <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
                {{ actionLoading ? copy.submitting : copy.confirmSubmit }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
