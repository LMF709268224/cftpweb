"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { ChevronDown, ChevronRight, RefreshCw, Search } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { formatBackendDate } from "@/lib/utils"
import {
  ADMIN_PIPELINE_STATUS_LABELS,
  ADMIN_PIPELINE_STATUS_ENUM_NAMES,
  COURSE_UNIT_STATUS_LABELS,
  COURSE_UNIT_STATUS_ENUM_NAMES,
  statusBadgeClassForStatus,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
  STAGE_STATUS_LABELS,
  STAGE_STATUS_ENUM_NAMES,
  statusLabel,
  statusLabelWithDiagnostics,
} from "@/lib/status-labels"

type ProgActionKind = "trigger-next-stage" | "terminate-pipeline" | "force-completed" | "force-signup-exam"

type ProgActionTarget =
  | {
    kind: "trigger-next-stage" | "terminate-pipeline"
    pipelineUlid: string
  }
  | {
    kind: "force-completed" | "force-signup-exam"
    pipelineUlid: string
    courseUnitUlid: string
  }

type ProgPipelineSummary = {
  pipeline_ulid: string
  candidate_ulid?: string
  pipeline_cc_ulid?: string
  status?: number | string
  current_stage_ulid?: string
  started_at?: string
  completed_at?: string
  created_at?: string
}

type ProgPipelineDetail = {
  pipeline?: {
    pipeline_ulid?: string
    candidate_ulid?: string
    pipeline_cc_ulid?: string
    status?: number | string
    current_stage_ulid?: string
    course_selection_json?: string
    started_at?: string
    completed_at?: string
    completed_reason?: string
    last_event_at?: string
    last_reconciled_at?: string
    created_at?: string
  }
  stages?: Array<{
    stage?: {
      stage_ulid?: string
      pipeline_ulid?: string
      stage_cc_ulid?: string
      seq_no?: number
      status?: number | string
      started_at?: string
      completed_at?: string
      completed_reason?: string
    }
    course_units?: Array<{
      course_unit_ulid?: string
      course_unit_cc_ulid?: string
      status?: number | string
      course_progress?: string
      exam_ulid?: string
      retried_count?: number
      completed_at?: string
    }>
  }>
}

type ProgStatusTransitionLogSummary = {
  transition_ulid: string
  entity_type?: string
  entity_ulid?: string
  from_status?: string
  to_status?: string
  reason_code?: string
  reason_message?: string
  trigger_source?: string
  event_type?: string
  created_at?: string
}

type ProgStatusTransitionLogDetail = {
  summary?: ProgStatusTransitionLogSummary
  pipeline_ulid?: string
}

const pageSize = 20

const statusOptions = (t: ReturnType<typeof useTranslation>["t"]) => [
  { value: "all", label: t.progPage.statusAll },
  { value: "1", label: statusLabel(t, ADMIN_PIPELINE_STATUS_LABELS, "1") },
  { value: "2", label: statusLabel(t, ADMIN_PIPELINE_STATUS_LABELS, "2") },
  { value: "3", label: statusLabel(t, ADMIN_PIPELINE_STATUS_LABELS, "3") },
  { value: "4", label: statusLabel(t, ADMIN_PIPELINE_STATUS_LABELS, "4") },
]

function StageDiagnostics({ stageUlid, candidateUlid, status }: { stageUlid: string; candidateUlid: string; status?: number | string }) {
  const { t } = useTranslation()
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (String(status) !== "1" || !stageUlid || !candidateUlid) return // Only fetch if WAIT_CANDIDATE
    let active = true
    setLoading(true)
    apiClient(`/api/mall/stage-orders?stage_ulid=${stageUlid}&candidate_ulid=${candidateUlid}`)
      .then(res => {
        if (active) setOrders(res?.items || res?.orders || [])
      })
      .catch(console.error)
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [stageUlid, candidateUlid, status])

  if (String(status) !== "1") return null

  return (
    <div className="mt-2 rounded bg-yellow-50 p-3 border border-yellow-200 dark:bg-yellow-950/20 dark:border-yellow-900">
      <div className="flex items-center gap-2 text-xs font-semibold text-yellow-800 dark:text-yellow-200">
        <Search className="h-4 w-4" />
        {t.progPage?.wbWaitCandidate}
      </div>
      {loading ? (
        <div className="mt-1 text-xs text-muted-foreground">{t.progPage?.wbQueryOrder}</div>
      ) : orders.length > 0 ? (
        <ul className="mt-1 list-inside list-disc text-xs text-muted-foreground">
          {orders.map((o: any) => (
            <li key={o.order_ulid}>{t.progPage?.wbOrder} {o.order_ulid} - {t.progPage?.wbOrderStatus}: {o.status} ({o.payment_status})</li>
          ))}
        </ul>
      ) : (
        <div className="mt-1 text-xs text-muted-foreground">{t.progPage?.wbNoOrder}</div>
      )}
    </div>
  )
}

function CourseUnitDiagnostics({ courseId, candidateUlid, status }: { courseId?: string; candidateUlid: string; status?: number | string }) {
  const { t } = useTranslation()
  const [progress, setProgress] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (String(status) !== "1" || !courseId || !candidateUlid) return // Only fetch if WAITING_STUDY and has IDs
    let active = true
    setLoading(true)
    apiClient(`/api/lms/courses/${encodeURIComponent(courseId)}/candidates/${encodeURIComponent(candidateUlid)}/progress`)
      .then(res => {
        if (active) setProgress(res)
      })
      .catch(console.error)
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [courseId, candidateUlid, status])

  if (String(status) !== "1") return null

  return (
    <div className="mt-2 rounded bg-blue-50 p-3 border border-blue-200 dark:bg-blue-950/20 dark:border-blue-900">
      <div className="flex items-center gap-2 text-xs font-semibold text-blue-800 dark:text-blue-200">
        <Search className="h-4 w-4" />
        {t.progPage?.wbStudyProgress ? t.progPage.wbStudyProgress.replace("{{courseId}}", courseId || t.common?.unknown) : ""}
      </div>
      {loading ? (
        <div className="mt-1 text-xs text-muted-foreground">{t.progPage?.wbQueryLms}</div>
      ) : progress ? (
        <div className="mt-2 space-y-1 text-xs text-muted-foreground">
          <div>{t.progPage?.wbCompletedLessons}: {progress.completed_lessons_count || 0} / {progress.total_lessons_count || 0}</div>
          <div>{t.progPage?.wbAllRequiredCompleted}: {progress.all_required_lessons_completed ? t.progPage?.yes : t.progPage?.no}</div>
          <div>{t.progPage?.wbUnpassedQuizzes}: {progress.unpassed_quizzes_count || 0}</div>
        </div>
      ) : (
        <div className="mt-1 text-xs text-muted-foreground">{t.progPage?.wbNoProgress}</div>
      )}
    </div>
  )
}

export default function ProgPage() {
  const { t } = useTranslation()
  const [pipelines, setPipelines] = useState<ProgPipelineSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [selectedPipelineId, setSelectedPipelineId] = useState("")
  const [selectedPipelineDetail, setSelectedPipelineDetail] = useState<ProgPipelineDetail | null>(null)
  const [candidateFilter, setCandidateFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")
  const [offset, setOffset] = useState(0)
  const [expandedStageIds, setExpandedStageIds] = useState<Record<string, boolean>>({})
  const [actionDialogOpen, setActionDialogOpen] = useState(false)
  const [pendingAction, setPendingAction] = useState<ProgActionTarget | null>(null)
  const [actionReason, setActionReason] = useState("")
  const [actionLoading, setActionLoading] = useState(false)
  const [certificateLoading, setCertificateLoading] = useState(false)
  const [logLoading, setLogLoading] = useState(false)
  const [logDetailLoading, setLogDetailLoading] = useState(false)
  const [transitionLogs, setTransitionLogs] = useState<ProgStatusTransitionLogSummary[]>([])
  const [selectedLogId, setSelectedLogId] = useState("")
  const [selectedLogDetail, setSelectedLogDetail] = useState<ProgStatusTransitionLogDetail | null>(null)
  const [logOffset, setLogOffset] = useState(0)
  const logPageSize = 20

  const loadPipelines = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      params.set("limit", String(pageSize))
      params.set("offset", String(offset))
      if (candidateFilter.trim()) params.set("candidate_ulid", candidateFilter.trim())
      if (statusFilter !== "all") params.set("status", statusFilter)
      const res = await apiClient(`/api/prog/pipelines?${params.toString()}`)
      const nextList = res?.pipelines || []
      setPipelines(nextList)
      if (nextList.length > 0) {
        const keepSelected = selectedPipelineId && nextList.some((item: ProgPipelineSummary) => item.pipeline_ulid === selectedPipelineId)
        if (!keepSelected) {
          setSelectedPipelineId(nextList[0].pipeline_ulid)
        }
      } else {
        setSelectedPipelineId("")
        setSelectedPipelineDetail(null)
      }
    } finally {
      setLoading(false)
    }
  }, [candidateFilter, offset, selectedPipelineId, statusFilter])

  const loadDetail = useCallback(async (pipelineUlid: string) => {
    if (!pipelineUlid) return
    setDetailLoading(true)
    try {
      const res = await apiClient(`/api/prog/pipelines/${encodeURIComponent(pipelineUlid)}`)
      setSelectedPipelineDetail(res || null)
    } catch {
      setSelectedPipelineDetail(null)
      toast.error(t.common.error)
    } finally {
      setDetailLoading(false)
    }
  }, [t.common.error])

  const loadTransitionLogs = useCallback(async (pipelineUlid: string, offsetValue = 0) => {
    if (!pipelineUlid) {
      setTransitionLogs([])
      setSelectedLogId("")
      setSelectedLogDetail(null)
      return
    }
    setLogLoading(true)
    try {
      const params = new URLSearchParams()
      params.set("limit", String(logPageSize))
      params.set("offset", String(offsetValue))
      const res = await apiClient(`/api/prog/pipelines/${encodeURIComponent(pipelineUlid)}/logs?${params.toString()}`)
      const nextLogs = res?.logs || []
      setTransitionLogs(nextLogs)
      setLogOffset(offsetValue)
      if (nextLogs.length > 0) {
        setSelectedLogId((prev) => (prev && nextLogs.some((item: ProgStatusTransitionLogSummary) => item.transition_ulid === prev) ? prev : nextLogs[0].transition_ulid))
      } else {
        setSelectedLogId("")
        setSelectedLogDetail(null)
      }
    } catch {
      setTransitionLogs([])
      setSelectedLogId("")
      setSelectedLogDetail(null)
      toast.error(t.common.error)
    } finally {
      setLogLoading(false)
    }
  }, [logPageSize, t.common.error])

  const loadTransitionLogDetail = useCallback(async (transitionUlid: string) => {
    if (!transitionUlid) {
      setSelectedLogDetail(null)
      return
    }
    setLogDetailLoading(true)
    try {
      const res = await apiClient(`/api/prog/pipelines/logs/${encodeURIComponent(transitionUlid)}`)
      setSelectedLogDetail(res || null)
    } catch {
      setSelectedLogDetail(null)
      toast.error(t.common.error)
    } finally {
      setLogDetailLoading(false)
    }
  }, [t.common.error])

  useEffect(() => {
    loadPipelines().catch(() => toast.error(t.common.error))
  }, [loadPipelines, t.common.error])

  useEffect(() => {
    if (selectedPipelineId) {
      loadDetail(selectedPipelineId).catch(() => toast.error(t.common.error))
    }
  }, [loadDetail, selectedPipelineId, t.common.error])

  useEffect(() => {
    if (selectedPipelineId) {
      loadTransitionLogs(selectedPipelineId).catch(() => toast.error(t.common.error))
    } else {
      setTransitionLogs([])
      setSelectedLogId("")
      setSelectedLogDetail(null)
    }
  }, [loadTransitionLogs, selectedPipelineId, t.common.error])

  useEffect(() => {
    if (selectedLogId) {
      loadTransitionLogDetail(selectedLogId).catch(() => toast.error(t.common.error))
    } else {
      setSelectedLogDetail(null)
    }
  }, [loadTransitionLogDetail, selectedLogId, t.common.error])

  const selectedSummary = useMemo(
    () => pipelines.find((item) => item.pipeline_ulid === selectedPipelineId) || null,
    [pipelines, selectedPipelineId],
  )

  const selectedStatus = selectedPipelineDetail?.pipeline?.status ?? selectedSummary?.status
  const totalStages = selectedPipelineDetail?.stages?.length || 0
  const totalUnits = selectedPipelineDetail?.stages?.reduce((count, stage) => count + (stage.course_units?.length || 0), 0) || 0
  const pipelineBadgeClass = statusBadgeClassForStatus(ADMIN_PIPELINE_STATUS_ENUM_NAMES, selectedStatus)
  const selectedStage = selectedPipelineDetail?.stages?.find((stage) => stage.stage?.stage_ulid === selectedPipelineDetail?.pipeline?.current_stage_ulid) || selectedPipelineDetail?.stages?.[0] || null
  const selectedPipelineUlid = selectedPipelineDetail?.pipeline?.pipeline_ulid || selectedSummary?.pipeline_ulid || ""
  const selectedCandidateUlid = selectedPipelineDetail?.pipeline?.candidate_ulid || selectedSummary?.candidate_ulid || ""
  const selectedStageStatus = selectedStage?.stage?.status
  const selectedStageStatusKey = String(selectedStageStatus ?? "")
  const selectedPipelineStatusKey = String(selectedStatus ?? "")
  const canTriggerNextStage = selectedPipelineStatusKey === "1" && selectedStageStatusKey === "3"
  const canOpenCertificate = Boolean(selectedPipelineUlid && selectedCandidateUlid)

  const reloadSelectedPipeline = useCallback(async () => {
    await loadPipelines()
    if (selectedPipelineUlid) {
      await loadDetail(selectedPipelineUlid)
    }
  }, [loadDetail, loadPipelines, selectedPipelineUlid])

  const openActionDialog = useCallback((target: ProgActionTarget) => {
    setPendingAction(target)
    setActionReason("")
    setActionDialogOpen(true)
  }, [])

  const closeActionDialog = useCallback(() => {
    if (actionLoading) return
    setActionDialogOpen(false)
    setPendingAction(null)
    setActionReason("")
  }, [actionLoading])

  const submitAction = useCallback(async () => {
    if (!pendingAction) return
    setActionLoading(true)
    try {
      const payload = JSON.stringify({ reason_message: actionReason.trim() })
      if (pendingAction.kind === "trigger-next-stage") {
        await apiClient(`/api/prog/pipelines/${encodeURIComponent(pendingAction.pipelineUlid)}/trigger-next-stage`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: payload,
        })
        toast.success(t.progPage.triggerNextStageSuccess)
      } else if (pendingAction.kind === "terminate-pipeline") {
        await apiClient(`/api/prog/pipelines/${encodeURIComponent(pendingAction.pipelineUlid)}/terminate`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: payload,
        })
        toast.success(t.progPage.terminatePipelineSuccess)
      } else if (pendingAction.kind === "force-completed") {
        await apiClient(`/api/prog/course-units/${encodeURIComponent(pendingAction.courseUnitUlid)}/force-completed`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: payload,
        })
        toast.success(t.progPage.forceCompletedSuccess)
      } else if (pendingAction.kind === "force-signup-exam") {
        await apiClient(`/api/prog/course-units/${encodeURIComponent(pendingAction.courseUnitUlid)}/force-signup-exam`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: payload,
        })
        toast.success(t.progPage.forceSignupExamSuccess)
      }
      setActionDialogOpen(false)
      setPendingAction(null)
      setActionReason("")
      await reloadSelectedPipeline()
    } catch {
      toast.error(t.common.error)
    } finally {
      setActionLoading(false)
    }
  }, [actionReason, pendingAction, reloadSelectedPipeline, t.common.error, t.progPage.forceCompletedSuccess, t.progPage.forceSignupExamSuccess, t.progPage.terminatePipelineSuccess, t.progPage.triggerNextStageSuccess])

  const viewCertificate = useCallback(async () => {
    if (!canOpenCertificate) {
      toast.error(t.common.error)
      return
    }
    setCertificateLoading(true)
    try {
      const res = await apiClient(
        `/api/prog/pipelines/${encodeURIComponent(selectedPipelineUlid)}/certificate-url?candidate_ulid=${encodeURIComponent(selectedCandidateUlid)}`,
      )
      const viewUrl = res?.view_url
      if (!viewUrl) {
        toast.error(t.common.error)
        return
      }
      window.open(viewUrl, "_blank", "noopener,noreferrer")
    } catch {
      toast.error(t.common.error)
    } finally {
      setCertificateLoading(false)
    }
  }, [canOpenCertificate, selectedCandidateUlid, selectedPipelineUlid, t.common.error])

  const selectedStageHint = useMemo(() => {
    if (selectedStageStatusKey === "1") return t.learning.stageWaitCandidateHint
    if (selectedStageStatusKey === "2") return t.learning.stageRunningHint
    if (selectedStageStatusKey === "3") return t.learning.stageCompletedHint
    return t.learning.nextStepDesc
  }, [selectedStageStatusKey, t.learning.nextStepDesc, t.learning.stageCompletedHint, t.learning.stageRunningHint, t.learning.stageWaitCandidateHint])
  const canLoadMoreLogs = transitionLogs.length === logPageSize
  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-foreground">{t.progPage.title}</h1>
              <p className="mt-1 text-muted-foreground">{t.progPage.subtitle}</p>
            </div>
            <Button variant="outline" onClick={loadPipelines} disabled={loading}>
              <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
              {t.progPage.refresh}
            </Button>
          </div>

          <div className="mb-4 grid gap-3 md:grid-cols-[minmax(220px,320px)_180px_1fr]">
            <div>
              <Label htmlFor="candidateFilter">{t.progPage.candidateFilter}</Label>
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="candidateFilter"
                  className="pl-9"
                  value={candidateFilter}
                  onChange={(event) => {
                    setCandidateFilter(event.target.value)
                    setOffset(0)
                  }}
                  placeholder={t.progPage.candidateFilterPlaceholder}
                />
              </div>
            </div>
            <div>
              <Label htmlFor="statusFilter">{t.progPage.statusFilter}</Label>
              <Select
                value={statusFilter}
                onValueChange={(value) => {
                  setStatusFilter(value)
                  setOffset(0)
                }}
              >
                <SelectTrigger id="statusFilter">
                  <SelectValue placeholder={t.progPage.statusFilterPlaceholder} />
                </SelectTrigger>
                <SelectContent>
                  {statusOptions(t).map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-4 xl:grid-cols-[420px_1fr]">
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <div>
                  <h2 className="font-semibold">{t.progPage.pipelineList}</h2>
                  <p className="text-xs text-muted-foreground">{t.progPage.pipelineListHint}</p>
                </div>
                <Badge variant="outline">{pipelines.length}</Badge>
              </div>
              <ScrollArea className="h-[calc(100vh-290px)]">
                <div className="divide-y">
                  {loading ? (
                    <div className="p-8 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                  ) : pipelines.length === 0 ? (
                    <div className="p-8 text-center text-sm text-muted-foreground">{t.progPage.noPipelines}</div>
                  ) : (
                    pipelines.map((pipeline) => {
                      const active = selectedPipelineId === pipeline.pipeline_ulid
                      return (
                        <button
                          key={pipeline.pipeline_ulid}
                          type="button"
                          onClick={() => setSelectedPipelineId(pipeline.pipeline_ulid)}
                          className={`block w-full px-4 py-3 text-left transition ${active ? "bg-muted" : "hover:bg-muted/60"}`}
                        >
                          <div className="flex items-center justify-between gap-3">
                            <div className="min-w-0">
                              <div className="truncate font-medium">{pipeline.pipeline_ulid}</div>
                              <div className="mt-1 truncate text-xs text-muted-foreground">{pipeline.candidate_ulid || t.common.na}</div>
                            </div>
                            <Badge variant="outline" className={statusBadgeClassForStatus(ADMIN_PIPELINE_STATUS_ENUM_NAMES, pipeline.status)}>
                              {statusLabelWithDiagnostics(t, ADMIN_PIPELINE_STATUS_LABELS, ADMIN_PIPELINE_STATUS_ENUM_NAMES, pipeline.status)}
                            </Badge>
                          </div>
                          <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                            <span>{t.progPage.pipelineCc}: {pipeline.pipeline_cc_ulid || t.common.na}</span>
                            <span>{t.progPage.currentStage}: {pipeline.current_stage_ulid || t.common.na}</span>
                            <span>{t.progPage.startedAt}: {formatBackendDate(pipeline.started_at)}</span>
                          </div>
                        </button>
                      )
                    })
                  )}
                </div>
              </ScrollArea>
              <div className="flex items-center justify-between border-t p-3">
                <Button variant="outline" size="sm" disabled={offset === 0 || loading} onClick={() => setOffset(Math.max(0, offset - pageSize))}>{t.progPage.prevPage}</Button>
                <span className="text-xs text-muted-foreground">{t.progPage.offset}: {offset}</span>
                <Button variant="outline" size="sm" disabled={pipelines.length < pageSize || loading} onClick={() => setOffset(offset + pageSize)}>{t.progPage.nextPage}</Button>
              </div>
            </section>

            <section className="space-y-4">
              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{t.progPage.pipelineDetail}</h2>
                    <p className="text-xs text-muted-foreground">{t.progPage.pipelineDetailHint}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    {selectedStatus !== undefined && (
                      <Badge variant="outline" className={pipelineBadgeClass}>
                        {statusLabelWithDiagnostics(t, ADMIN_PIPELINE_STATUS_LABELS, ADMIN_PIPELINE_STATUS_ENUM_NAMES, selectedStatus)}
                      </Badge>
                    )}
                    <Button variant="outline" size="sm" onClick={() => selectedPipelineId && loadDetail(selectedPipelineId)} disabled={!selectedPipelineId || detailLoading}>
                      <RefreshCw className={`h-4 w-4 ${detailLoading ? "animate-spin" : ""}`} />
                      {t.progPage.loadDetail}
                    </Button>
                  </div>
                </div>
                <div className="grid gap-4 p-4 md:grid-cols-2">
                  <div className="rounded-md border bg-muted/20 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">{t.progPage.pipelineUlid}</div>
                    <div className="mt-1 break-all font-medium">{selectedPipelineDetail?.pipeline?.pipeline_ulid || selectedSummary?.pipeline_ulid || t.common.na}</div>
                  </div>
                  <div className="rounded-md border bg-muted/20 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">{t.progPage.candidateUlid}</div>
                    <div className="mt-1 break-all font-medium">{selectedPipelineDetail?.pipeline?.candidate_ulid || selectedSummary?.candidate_ulid || t.common.na}</div>
                  </div>
                  <div className="rounded-md border bg-muted/20 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">{t.progPage.pipelineStatus}</div>
                    <div className="mt-1 font-medium">
                      {statusLabelWithDiagnostics(t, ADMIN_PIPELINE_STATUS_LABELS, ADMIN_PIPELINE_STATUS_ENUM_NAMES, selectedStatus)}
                    </div>
                  </div>
                  <div className="rounded-md border bg-muted/20 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">{t.progPage.metrics}</div>
                    <div className="mt-1 font-medium">
                      {t.progPage.stageCount}: {totalStages} · {t.progPage.unitCount}: {totalUnits}
                    </div>
                  </div>
                </div>
                <div className="border-t px-4 py-3 text-sm text-muted-foreground">
                  <div className="grid gap-2 md:grid-cols-2">
                    <div>{t.progPage.pipelineCc}: {selectedPipelineDetail?.pipeline?.pipeline_cc_ulid || selectedSummary?.pipeline_cc_ulid || t.common.na}</div>
                    <div>{t.progPage.currentStage}: {selectedPipelineDetail?.pipeline?.current_stage_ulid || selectedSummary?.current_stage_ulid || t.common.na}</div>
                    <div>{t.progPage.startedAt}: {formatBackendDate(selectedPipelineDetail?.pipeline?.started_at || selectedSummary?.started_at)}</div>
                    <div>{t.progPage.completedAt}: {formatBackendDate(selectedPipelineDetail?.pipeline?.completed_at || selectedSummary?.completed_at)}</div>
                    <div className="md:col-span-2">{t.progPage.reason}: {selectedPipelineDetail?.pipeline?.completed_reason || t.common.na}</div>
                  </div>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{t.progPage.pipelineActionsTitle}</h2>
                    <p className="text-xs text-muted-foreground">{t.progPage.pipelineActionsHint}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={viewCertificate} disabled={!canOpenCertificate || certificateLoading}>
                    {certificateLoading ? t.common.loading : t.progPage.viewCertificate}
                  </Button>
                </div>
                <div className="flex flex-wrap gap-2 px-4 py-4">
                  <Button
                    onClick={() => openActionDialog({ kind: "trigger-next-stage", pipelineUlid: selectedPipelineUlid })}
                    disabled={!canTriggerNextStage}
                  >
                    {t.progPage.triggerNextStage}
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={() => openActionDialog({ kind: "terminate-pipeline", pipelineUlid: selectedPipelineUlid })}
                    disabled={!selectedPipelineUlid}
                  >
                    {t.progPage.terminatePipeline}
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{t.progPage.statusTransitionLogsTitle}</h2>
                    <p className="text-xs text-muted-foreground">{t.progPage.statusTransitionLogsHint}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{transitionLogs.length}</Badge>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => selectedPipelineId && loadTransitionLogs(selectedPipelineId, logOffset)}
                      disabled={!selectedPipelineId || logLoading}
                    >
                      <RefreshCw className={`h-4 w-4 ${logLoading ? "animate-spin" : ""}`} />
                      {t.progPage.loadLogs}
                    </Button>
                  </div>
                </div>
                <div className="grid gap-4 p-4 lg:grid-cols-[360px_1fr]">
                  <div className="space-y-3">
                    {logLoading ? (
                      <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                    ) : transitionLogs.length === 0 ? (
                      <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.progPage.noLogs}</div>
                    ) : (
                      <>
                        <ScrollArea className="h-[460px] rounded-md border">
                          <div className="divide-y">
                            {transitionLogs.map((log) => {
                              const active = selectedLogId === log.transition_ulid
                              // const statusValue = log.to_status || log.from_status
                              return (
                                <button
                                  key={log.transition_ulid}
                                  type="button"
                                  onClick={() => setSelectedLogId(log.transition_ulid)}
                                  className={`block w-full px-3 py-3 text-left transition ${active ? "bg-muted" : "hover:bg-muted/60"}`}
                                >
                                  <div className="flex items-center justify-between gap-2">
                                    <div className="min-w-0">
                                      <div className="truncate text-sm font-medium">{log.transition_ulid}</div>
                                      <div className="mt-1 truncate text-xs text-muted-foreground">{log.entity_type || t.common.na} · {log.entity_ulid || t.common.na}</div>
                                    </div>
                                  </div>
                                  <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                                    <span className="inline-flex items-center gap-2">
                                      {t.progPage.toStatus}:
                                      <Badge variant="outline" className={timelineStatusBadgeClassForStatus(log.entity_type, log.to_status)}>
                                        {timelineStatusLabelWithDiagnostics(t, log.entity_type, log.to_status)}
                                      </Badge>
                                    </span>
                                  </div>
                                </button>
                              )
                            })}
                          </div>
                        </ScrollArea>
                        <div className="flex items-center justify-between">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={logOffset === 0 || logLoading}
                            onClick={() => selectedPipelineId && loadTransitionLogs(selectedPipelineId, Math.max(0, logOffset - logPageSize))}
                          >
                            {t.progPage.prevPage}
                          </Button>
                          <span className="text-xs text-muted-foreground">{t.progPage.offset}: {logOffset}</span>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={!canLoadMoreLogs || logLoading}
                            onClick={() => selectedPipelineId && loadTransitionLogs(selectedPipelineId, logOffset + logPageSize)}
                          >
                            {t.progPage.nextPage}
                          </Button>
                        </div>
                      </>
                    )}
                  </div>
                  <div className="rounded-md border bg-muted/20 p-4 text-sm">
                    {logDetailLoading ? (
                      <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                    ) : selectedLogDetail?.summary ? (
                      <div className="space-y-3">
                        <div>
                          <div className="text-xs text-muted-foreground">{t.progPage.logDetails}</div>
                          <div className="mt-1 break-all font-medium">{selectedLogDetail.summary.transition_ulid}</div>
                        </div>
                        <div className="grid gap-3 md:grid-cols-2">
                          <div><span className="text-xs text-muted-foreground">{t.progPage.entityType}: </span>{selectedLogDetail.summary.entity_type || t.common.na}</div>
                          <div><span className="text-xs text-muted-foreground">{t.progPage.entityUlid}: </span>{selectedLogDetail.summary.entity_ulid || t.common.na}</div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-muted-foreground">{t.progPage.toStatus}: </span>
                            <Badge variant="outline" className={timelineStatusBadgeClassForStatus(selectedLogDetail.summary.entity_type, selectedLogDetail.summary.to_status)}>
                              {timelineStatusLabelWithDiagnostics(t, selectedLogDetail.summary.entity_type, selectedLogDetail.summary.to_status)}
                            </Badge>
                          </div>
                          <div><span className="text-xs text-muted-foreground">{t.progPage.reasonCode}: </span>{selectedLogDetail.summary.reason_code || t.common.na}</div>
                          <div><span className="text-xs text-muted-foreground">{t.progPage.triggerSource}: </span>{selectedLogDetail.summary.trigger_source || t.common.na}</div>
                          <div><span className="text-xs text-muted-foreground">{t.progPage.eventType}: </span>{selectedLogDetail.summary.event_type || t.common.na}</div>
                          <div><span className="text-xs text-muted-foreground">{t.progPage.createdAt}: </span>{formatBackendDate(selectedLogDetail.summary.created_at)}</div>
                          <div className="md:col-span-2"><span className="text-xs text-muted-foreground">{t.progPage.reasonMessage}: </span>{selectedLogDetail.summary.reason_message || t.common.na}</div>
                          <div className="md:col-span-2"><span className="text-xs text-muted-foreground">{t.progPage.pipelineUlid}: </span>{selectedLogDetail.pipeline_ulid || t.common.na}</div>
                        </div>
                      </div>
                    ) : (
                      <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.progPage.logDetailHint}</div>
                    )}
                  </div>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{t.progPage.stageTree}</h2>
                    <p className="text-xs text-muted-foreground">{t.progPage.stageTreeHint}</p>
                  </div>
                  <Badge variant="outline" className={statusBadgeClassForStatus(STAGE_STATUS_ENUM_NAMES, selectedStage?.stage?.status)}>
                    {statusLabelWithDiagnostics(t, STAGE_STATUS_LABELS, STAGE_STATUS_ENUM_NAMES, selectedStage?.stage?.status)}
                  </Badge>
                </div>
                <div className="border-b px-4 py-3 text-sm text-muted-foreground">
                  <div className="rounded-md border border-dashed bg-muted/20 px-3 py-2">
                    <div className="text-xs font-medium text-foreground">{t.progPage.stageReadOnlyTitle}</div>
                    <div className="mt-1">{selectedStageHint}</div>
                  </div>
                </div>
                <div className="p-4">
                  {!selectedPipelineId ? (
                    <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.progPage.selectPipelineHint}</div>
                  ) : detailLoading ? (
                    <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                  ) : !selectedPipelineDetail?.stages?.length ? (
                    <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t.progPage.noStages}</div>
                  ) : (
                    <div className="space-y-3">
                      {selectedPipelineDetail.stages.map((stage, index) => {
                        const summary = stage.stage
                        const stageId = summary?.stage_ulid || `${index}`
                        const expanded = expandedStageIds[stageId] ?? index === 0
                        return (
                          <div key={stageId} className="rounded-lg border">
                            <button
                              type="button"
                              className="flex w-full items-center justify-between gap-3 border-b px-4 py-3 text-left"
                              onClick={() => setExpandedStageIds((prev) => ({ ...prev, [stageId]: !prev[stageId] }))}
                            >
                              <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                  <span className="text-sm font-semibold">{t.progPage.stage} {summary?.seq_no || index + 1}</span>
                                  <Badge variant="outline" className={statusBadgeClassForStatus(STAGE_STATUS_ENUM_NAMES, summary?.status)}>
                                    {statusLabelWithDiagnostics(t, STAGE_STATUS_LABELS, STAGE_STATUS_ENUM_NAMES, summary?.status)}
                                  </Badge>
                                </div>
                                <div className="mt-1 break-all text-xs text-muted-foreground">{summary?.stage_ulid || t.common.na}</div>
                              </div>
                              {expanded ? <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />}
                            </button>
                            {expanded && (
                              <div className="space-y-3 p-4">
                                <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                                  <span>{t.progPage.stageCc}: {summary?.stage_cc_ulid || t.common.na}</span>
                                  <span>{t.progPage.startedAt}: {formatBackendDate(summary?.started_at)}</span>
                                  <span>{t.progPage.completedAt}: {formatBackendDate(summary?.completed_at)}</span>
                                  <span>{t.progPage.reason}: {summary?.completed_reason || t.common.na}</span>
                                </div>
                                <StageDiagnostics
                                  stageUlid={summary?.stage_ulid || ""}
                                  candidateUlid={selectedPipelineDetail?.pipeline?.candidate_ulid || ""}
                                  status={summary?.status}
                                />
                                {stage.course_units?.length ? (
                                  <div className="space-y-2">
                                    {stage.course_units.map((unit) => (
                                      <div key={unit.course_unit_ulid} className="rounded-md border bg-muted/20 p-3">
                                        <div className="flex flex-wrap items-center justify-between gap-3">
                                          <div className="min-w-0">
                                            <div className="font-medium">{unit.course_unit_ulid || t.common.na}</div>
                                            <div className="mt-1 text-xs text-muted-foreground">{t.progPage.unitCc}: {unit.course_unit_cc_ulid || t.common.na}</div>
                                          </div>
                                          <Badge variant="outline" className={statusBadgeClassForStatus(COURSE_UNIT_STATUS_ENUM_NAMES, unit.status)}>
                                            {statusLabelWithDiagnostics(t, COURSE_UNIT_STATUS_LABELS, COURSE_UNIT_STATUS_ENUM_NAMES, unit.status)}
                                          </Badge>
                                        </div>
                                        <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                                          <span>{t.progPage.progress}: {unit.course_progress || t.common.na}</span>
                                          <span>{t.progPage.examUlid}: {unit.exam_ulid || t.common.na}</span>
                                          <span>{t.progPage.retriedCount}: {unit.retried_count ?? 0}</span>
                                          <span>{t.progPage.completedAt}: {formatBackendDate(unit.completed_at)}</span>
                                        </div>
                                        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t pt-3">
                                          <div className="text-xs text-muted-foreground w-full">
                                            <CourseUnitDiagnostics
                                              courseId={unit.course_unit_cc_ulid}
                                              candidateUlid={selectedPipelineDetail?.pipeline?.candidate_ulid || ""}
                                              status={unit.status}
                                            />
                                          </div>
                                          <div className="text-xs text-muted-foreground mt-2">
                                            {(String(unit.status ?? "") === "5" || String(unit.status ?? "") === "1")
                                              ? t.progPage.unitActionsHint
                                              : t.progPage.unitActionUnavailable}
                                          </div>
                                          <div className="flex flex-wrap gap-2 mt-2">
                                            {(String(unit.status ?? "") === "5" || String(unit.status ?? "") === "1") && (
                                              <>
                                                <Button
                                                  size="sm"
                                                  variant="outline"
                                                  onClick={() => openActionDialog({
                                                    kind: "force-completed",
                                                    pipelineUlid: selectedPipelineUlid,
                                                    courseUnitUlid: unit.course_unit_ulid || "",
                                                  })}
                                                  disabled={!unit.course_unit_ulid}
                                                >
                                                  {t.progPage.forceCompleted}
                                                </Button>
                                              </>
                                            )}
                                            {String(unit.status ?? "") === "5" && (
                                              <Button
                                                size="sm"
                                                variant="outline"
                                                onClick={() => openActionDialog({
                                                  kind: "force-signup-exam",
                                                  pipelineUlid: selectedPipelineUlid,
                                                  courseUnitUlid: unit.course_unit_ulid || "",
                                                })}
                                                disabled={!unit.course_unit_ulid}
                                              >
                                                {t.progPage.forceSignupExam}
                                              </Button>
                                            )}
                                          </div>
                                        </div>
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">{t.progPage.noUnits}</div>
                                )}
                              </div>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="border-b px-4 py-3">
                  <h2 className="font-semibold">{t.progPage.opsNoteTitle}</h2>
                </div>
                <div className="px-4 py-3 text-sm text-muted-foreground">
                  {t.progPage.opsNote}
                </div>
              </div>
            </section>
          </div>
        </div>
      </main>

      <Dialog open={actionDialogOpen} onOpenChange={(open) => {
        if (!open) {
          closeActionDialog()
          return
        }
        setActionDialogOpen(true)
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {pendingAction?.kind === "terminate-pipeline"
                ? t.progPage.terminatePipelineConfirmTitle
                : pendingAction?.kind === "trigger-next-stage"
                  ? t.progPage.pipelineActionsTitle
                  : t.progPage.unitActionsTitle}
            </DialogTitle>
            <DialogDescription>
              {pendingAction?.kind === "terminate-pipeline"
                ? t.progPage.terminatePipelineConfirmDesc
                : t.progPage.actionReasonPlaceholder}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="prog-action-reason">{t.progPage.actionReasonLabel}</Label>
            <Textarea
              id="prog-action-reason"
              value={actionReason}
              onChange={(event) => setActionReason(event.target.value)}
              placeholder={t.progPage.actionReasonPlaceholder}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeActionDialog} disabled={actionLoading}>
              {t.common.cancel}
            </Button>
            <Button onClick={submitAction} disabled={actionLoading}>
              {actionLoading ? t.common.loading : t.common.confirm}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

