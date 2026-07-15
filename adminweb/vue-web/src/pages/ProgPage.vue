<script setup lang="ts">
import { ArrowLeft, Eye, Loader2, RefreshCw, RotateCcw, Search, ShieldX, StepForward, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type ActionKind = "trigger-next-stage" | "terminate-pipeline" | "force-completed" | "force-signup-exam"
type DetailTab = "overview" | "stages" | "units" | "certificateTasks" | "logs"
type DetailTabItem = { key: DetailTab; title: string; desc: string; count: number }

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

function clearCandidateFilter() {
  const shouldSearch = Boolean(appliedCandidateFilter.value)
  candidateFilter.value = ""
  if (shouldSearch) searchPipelines()
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
  await Promise.all([loadPipelineCatalog(), loadPipelines()])
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
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
        <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
          <div class="grid gap-4 lg:grid-cols-[1.5fr_1fr_auto]">
            <label class="grid gap-2 text-sm font-bold">
              {{ copy.filters.candidate }}
              <div class="relative">
                <input v-model="candidateFilter" class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10" :placeholder="copy.filters.candidatePlaceholder" />
                <button
                  v-if="candidateFilter"
                  class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
                  type="button"
                  :aria-label="copy.filters.clearInput"
                  :title="copy.filters.clearInput"
                  @click="clearCandidateFilter"
                >
                  <X class="h-4 w-4" />
                </button>
              </div>
            </label>
            <label class="grid gap-2 text-sm font-bold">
              {{ copy.filters.status }}
              <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 px-3">
                <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
            <button class="mt-0 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white shadow-sm lg:mt-7" type="button" @click="searchPipelines">
              <Search class="h-4 w-4" />
              {{ copy.filters.search }}
            </button>
          </div>
        </div>

        <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:px-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
          
            </div>
          <div v-if="loading" class="px-4 py-10 text-center text-slate-500 md:px-6">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <template v-else-if="pipelines.length">
            <div class="hidden grid-cols-[minmax(0,1fr)_150px_180px_110px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500 md:grid">
              <span>{{ copy.columns.pipeline }}</span>
              <span class="text-center">{{ copy.columns.status }}</span>
              <span class="text-right">{{ copy.columns.startedAt }}</span>
              <span class="text-right">{{ copy.columns.action }}</span>
            </div>
            <button
              v-for="pipeline in pipelines"
              :key="pipelineUlid(pipeline)"
              class="flex w-full flex-col gap-3 border-b border-slate-100 px-4 py-4 text-left transition last:border-b-0 hover:bg-slate-50 md:grid md:grid-cols-[minmax(0,1fr)_150px_180px_110px] md:gap-4 md:px-5"
              :class="pipelineUlid(pipeline) === selectedPipelineUlid ? 'bg-sky-50' : ''"
              type="button"
              @click="openPipeline(pipeline)"
            >
              <div class="min-w-0">
                <div class="break-words text-lg font-black md:truncate">{{ pipelineDisplayName(pipeline) }}</div>
                <div class="mt-1 break-all text-sm text-slate-500">
                  <div class="font-semibold text-slate-800">{{ copy.candidatePrefix }}{{ pipeline.candidate_name || "-" }}</div>
                  <div class="font-mono text-xs text-slate-400">{{ pipeline.candidate_ulid || pipeline.candidate_id || "-" }}</div>
                </div>
                <div class="mt-3 grid gap-1 rounded-xl bg-slate-50 px-3 py-2 text-xs text-slate-500">
                  <div class="break-all font-semibold">Pipeline: {{ pipelineUlid(pipeline) || "-" }}</div>
                  <div class="break-all">{{ copy.currentStagePrefix }}{{ pipeline.current_stage_ulid || "-" }}</div>
                </div>
              </div>
              <span class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:self-center md:justify-self-center md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.status }}</span>
                <span class="inline-flex whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(pipeline.status))">{{ statusLabel(pipeline.status) }}</span>
              </span>
              <span class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:self-center md:justify-self-end md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.startedAt }}</span>
                <span class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pipeline.started_at || pipeline.created_at || "")) }}</span>
              </span>
              <span class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-[#1890ff] transition hover:underline md:w-auto md:self-center md:justify-self-end md:border-0 md:bg-transparent md:px-0 md:py-0">{{ copy.viewDetails }}</span>
            </button>
          </template>
          <div v-else class="px-4 py-10 text-center text-slate-500 md:px-6">{{ copy.empty }}</div>
          <div class="flex flex-col justify-end gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row md:px-5">
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - pageSize)">{{ copy.prev }}</button>
            <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += pageSize">{{ copy.next }}</button>
          </div>
        </section>
      </aside>
    </div>

    <div v-if="selectedSummary" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
      <div class="flex h-full max-h-none w-full max-w-[1200px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
        <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6">
          <div class="min-w-0">
            <h2 class="text-xl font-black md:text-2xl">{{ copy.detailTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ selectedPipelineUlid }}</p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="backToList">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
          <main class="min-w-0 space-y-6">
        <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black md:text-2xl">{{ pipelineDisplayName(selectedSummary) }}</h2>
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
                  class="inline-flex h-11 shrink-0 items-center gap-2 rounded-2xl border px-3 text-sm font-black transition md:gap-3 md:px-4"
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
                    class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 py-2.5 text-sm font-bold text-white disabled:opacity-40 sm:w-auto"
                    type="button"
                    :disabled="!canTriggerNextStage"
                    @click="openAction({ kind: 'trigger-next-stage', pipelineUlid: selectedPipelineUlid })"
                  >
                    <StepForward class="h-4 w-4" />
                    {{ copy.actions["trigger-next-stage"] }}
                  </button>
                  <button
                    class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-bold text-white disabled:opacity-40 sm:w-auto"
                    type="button"
                    :disabled="!canTerminatePipeline"
                    @click="openAction({ kind: 'terminate-pipeline', pipelineUlid: selectedPipelineUlid })"
                  >
                    <ShieldX class="h-4 w-4" />
                    {{ copy.actions["terminate-pipeline"] }}
                  </button>
                  <button
                    v-if="canViewCertificate"
                    class="inline-flex w-full items-center justify-center gap-2 rounded-xl border bg-white px-4 py-2.5 text-sm font-bold disabled:opacity-40 sm:w-auto"
                    type="button"
                    :disabled="certificateLoading"
                    @click="viewCertificate"
                  >
                    <Eye class="h-4 w-4" />
                    {{ copy.viewCertificate }}
                  </button>
                </div>
              </div>
            </div>

            <section class="max-h-none min-w-0 overflow-y-auto md:max-h-[60vh]">
              <div v-if="detailLoading" class="px-4 py-10 text-center text-slate-500 md:px-6">
                <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                {{ copy.loadingDetails }}
              </div>

              <div v-else-if="activeTab === 'overview'" class="space-y-4 p-4 md:p-5">
                <div class="grid gap-3 md:grid-cols-2">
                  <div v-for="entry in detailEntries('overview', detailPipelineRecord)" :key="entry.key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                    <div class="text-xs font-black text-slate-400">{{ entry.label }}</div>
                    <div class="mt-1 whitespace-pre-wrap break-all text-sm font-bold">{{ entry.value }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'stages'" class="grid lg:grid-cols-[320px_minmax(0,1fr)]">
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
                <div class="space-y-5 p-4 md:p-5">
                  <template v-if="selectedStage">
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in detailEntries('stage', stageRecord(selectedStage))" :key="entry.key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                        <div class="text-xs font-black text-slate-400">{{ entry.label }}</div>
                        <div class="mt-1 whitespace-pre-wrap break-all text-sm font-bold">{{ entry.value }}</div>
                      </div>
                    </div>
                  </template>
                  <div v-else class="p-12 text-center text-slate-500">{{ copy.selectStage }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'units'" class="grid lg:grid-cols-[360px_minmax(0,1fr)]">
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
                <div class="space-y-5 p-4 md:p-5">
                  <template v-if="selectedUnit">
                    <div class="rounded-xl border border-sky-100 bg-sky-50 p-3">
                      <div class="text-xs font-black uppercase text-sky-600">{{ copy.parentStage }}</div>
                      <div class="mt-1 break-all text-sm font-bold">{{ stageName(selectedUnit.stage) }} · {{ stageUlid(selectedUnit.stage) || "-" }}</div>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                      <div v-for="entry in detailEntries('unit', selectedUnit.unit)" :key="entry.key" class="rounded-xl border border-slate-100 bg-slate-50 p-3">
                        <div class="text-xs font-black text-slate-400">{{ entry.label }}</div>
                        <div class="mt-1 whitespace-pre-wrap break-all text-sm font-bold">{{ entry.value }}</div>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-3">
                      <button
                        class="w-full rounded-xl border bg-white px-4 py-2 text-sm font-bold disabled:opacity-40 sm:w-auto"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-completed', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.forceCompleted }}
                      </button>
                      <button
                        class="w-full rounded-xl border bg-white px-4 py-2 text-sm font-bold disabled:opacity-40 sm:w-auto"
                        type="button"
                        :disabled="!courseUnitUlid(selectedUnit.unit)"
                        @click="openAction({ kind: 'force-signup-exam', pipelineUlid: selectedPipelineUlid, courseUnitUlid: courseUnitUlid(selectedUnit.unit) })"
                      >
                        {{ copy.resetExamSignup }}
                      </button>
                    </div>
                  </template>
                  <div v-else class="p-12 text-center text-slate-500">{{ copy.selectUnit }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'certificateTasks'" class="grid lg:grid-cols-[380px_minmax(0,1fr)]">
                <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
                  <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                    <div>
                      <div class="font-black">{{ copy.certificateTasks.listTitle }}</div>
                      <div class="text-xs text-slate-500">{{ copy.certificateTasks.listDescription }}</div>
                    </div>
                    <button class="rounded-xl border px-3 py-2 text-sm font-bold" type="button" @click="loadCertificateTasks">{{ copy.load }}</button>
                  </div>
                  <div v-if="certificateTasksLoading" class="p-10 text-center text-slate-500">
                    <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                    {{ copy.loading }}
                  </div>
                  <button
                    v-for="task in certificateTasks"
                    v-else
                    :key="certificateTaskUlid(task)"
                    class="w-full border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
                    :class="certificateTaskUlid(selectedCertificateTask) === certificateTaskUlid(task) ? 'bg-sky-50' : ''"
                    type="button"
                    @click="selectedCertificateTask = task; loadCertificateTaskDetail(certificateTaskUlid(task))"
                  >
                    <div class="font-black">{{ task.degree_no || certificateTaskUlid(task) || copy.certificateTasks.taskFallback }}</div>
                    <div class="mt-1 break-all text-sm text-slate-500">{{ certificateTaskUlid(task) || "-" }}</div>
                    <div class="mt-3 flex flex-wrap items-center gap-2">
                      <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(certificateTaskStatusLabel(task.status))">
                        {{ certificateTaskStatusLabel(task.status) }}
                      </span>
                      <span class="text-xs text-slate-400">{{ formatDate(String(task.created_at || "")) }}</span>
                    </div>
                  </button>
                  <div v-if="!certificateTasksLoading && !certificateTasks.length" class="p-10 text-center text-slate-500">{{ copy.certificateTasks.empty }}</div>
                </div>
                <div class="space-y-5 p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h4 class="text-lg font-black">{{ copy.certificateTasks.detailTitle }}</h4>
                      <p class="mt-1 break-all text-sm text-slate-500">{{ certificateTaskUlid(selectedCertificateTask) || "-" }}</p>
                    </div>
                    <button
                      class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 py-2.5 text-sm font-bold text-white disabled:opacity-40 sm:w-auto"
                      type="button"
                      :disabled="!selectedCertificateTask || retryingCertificateTask === certificateTaskUlid(selectedCertificateTask)"
                      @click="selectedCertificateTask && retryCertificateTask(selectedCertificateTask)"
                    >
                      <Loader2 v-if="retryingCertificateTask === certificateTaskUlid(selectedCertificateTask)" class="h-4 w-4 animate-spin" />
                      <RotateCcw v-else class="h-4 w-4" />
                      {{ retryingCertificateTask === certificateTaskUlid(selectedCertificateTask) ? copy.certificateTasks.retrying : copy.certificateTasks.retry }}
                    </button>
                  </div>
                  <div v-if="certificateTaskDetailLoading" class="p-10 text-center text-slate-500">
                    <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                    {{ copy.loadingDetails }}
                  </div>
                  <template v-else-if="selectedCertificateTask">
                    <div class="grid gap-4 md:grid-cols-2">
                      <div v-for="entry in certificateTaskEntries(asRecord((certificateTaskDetail || {}).summary || selectedCertificateTask))" :key="entry.key" class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
                        <div class="text-xs font-black text-slate-400">{{ entry.label }}</div>
                        <div class="mt-2 whitespace-pre-wrap break-all text-sm font-bold">{{ entry.value }}</div>
                      </div>
                    </div>
                    <div v-if="certificateTaskDetail?.error_message" class="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm font-bold text-red-700">
                      <div class="text-xs font-black uppercase text-red-400">{{ copy.certificateTasks.errorMessage }}</div>
                      <div class="mt-2 whitespace-pre-wrap break-words">{{ certificateTaskDetail.error_message }}</div>
                    </div>
                    <div v-if="certificateTaskDetail?.template_params" class="rounded-2xl border border-slate-200">
                      <div class="border-b border-slate-100 px-4 py-3 text-sm font-black">{{ copy.certificateTasks.templateParams }}</div>
                      <pre class="max-h-[260px] overflow-auto rounded-b-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ certificateTaskDetail.template_params }}</pre>
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
                  <div v-else class="p-12 text-center text-slate-500">{{ copy.certificateTasks.selectTask }}</div>
                </div>
              </div>

              <div v-else-if="activeTab === 'logs'" class="grid lg:grid-cols-[380px_minmax(0,1fr)]">
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
                    <div class="mt-1 text-sm text-slate-500">{{ entityTypeLabel(log.entity_type) }} · {{ log.entity_ulid || "-" }}</div>
                    <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(log.created_at || "")) }}</div>
                  </button>
                  <div v-if="!logsLoading && !logs.length" class="p-10 text-center text-slate-500">{{ copy.noLogs }}</div>
                  <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row sm:items-center md:px-5">
                    <span class="text-sm font-bold text-slate-500">{{ Math.floor(logOffset / logPageSize) + 1 }} / {{ Math.max(1, Math.ceil(logsTotal / logPageSize)) }}</span>
                    <div class="flex flex-col gap-3 sm:flex-row">
                    <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevLogs" @click="loadLogs(selectedPipelineUlid, Math.max(0, logOffset - logPageSize))">{{ copy.prev }}</button>
                    <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNextLogs" @click="loadLogs(selectedPipelineUlid, logOffset + logPageSize)">{{ copy.next }}</button>
                    </div>
                  </div>
                </div>
                <div class="space-y-5 p-4 md:p-5">
                  <h4 class="text-lg font-black">{{ copy.logDetailTitle }}</h4>
                  <div v-if="logDetailLoading" class="p-10 text-center text-slate-500">
                    <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                    {{ copy.loadingDetails }}
                  </div>
                  <template v-else>
                    <div v-if="selectedLog" class="grid gap-4 md:grid-cols-2">
                      <div v-for="entry in detailEntries('log', asRecord((logDetail || selectedLog).summary || selectedLog))" :key="entry.key" class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
                        <div class="text-xs font-black text-slate-400">{{ entry.label }}</div>
                        <div class="mt-2 whitespace-pre-wrap break-all text-sm font-bold">{{ entry.value }}</div>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </section>
        </section>
          </main>
        </div>
      </div>
    </div>

    <div v-if="pendingAction" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 md:p-6">
      <div class="w-full max-w-lg rounded-2xl bg-white p-4 shadow-2xl md:rounded-3xl md:p-6">
        <h2 class="text-xl font-black md:text-2xl">{{ copy.confirmTitle }}</h2>
        <p class="mt-2 text-sm text-slate-600">{{ copy.confirmDescription(actionLabel(pendingAction.kind)) }}</p>
        <textarea v-model="actionReason" class="mt-5 min-h-28 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.reasonPlaceholder" />
        <div class="mt-5 flex flex-col items-stretch justify-end gap-3 sm:flex-row sm:items-center">
          <button class="inline-flex h-11 min-w-[96px] items-center justify-center rounded-xl border px-5 text-sm font-bold disabled:opacity-50" type="button" :disabled="actionLoading" @click="pendingAction = null">{{ copy.cancel }}</button>
          <button class="inline-flex h-11 min-w-[112px] items-center justify-center rounded-xl bg-blue-700 px-5 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="actionLoading" @click="submitAction">
            {{ actionLoading ? copy.submitting : copy.confirmSubmit }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
