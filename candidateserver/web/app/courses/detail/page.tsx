"use client"

import React, { Suspense, useEffect, useMemo, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { ArrowLeft, Award, BookOpen, CheckCircle2, Clock, CreditCard, ExternalLink, Lock, Play, Sparkles, ArrowRight } from "lucide-react"
import { toast } from "sonner"

import { apiClient } from "@/lib/apiClient"
import { cn } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { PurchaseDialog } from "@/components/purchase-dialog"
import {
  courseUnitNextStepActionFromStatus,
  stageStatusHintLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
} from "@cftpweb/shared"

type PipelineDetail = {
  config?: PipelineConfig
  instance?: Record<string, any>
  next_step?: PipelineNextStep
  pipeline_status?: string
  current_stage_ulid?: string
  current_stage_status?: string
  current_stage_name?: string
  current_unit_status?: string
}

type PipelineConfig = {
  pipeline_id: string
  pipeline_guid: string
  version: number
  name: string
  category_tips?: string
  status: string
  unlock_stripe_price_id?: string
  package_stripe_price_id?: string
  stages?: StageConfig[]
  final_quals?: Qualification[]
}

type StageConfig = {
  stage_id?: string
  name?: string
  sort_order?: number
  runtime_status?: string | number
  units?: UnitConfig[]
}

type UnitConfig = {
  unit_id?: string
  name?: string
  glms_course_id?: string
  runtime_status?: string | number
  stripe_price_id?: string
  allow_retake?: boolean
}

type Qualification = {
  qual_id?: string
  name_hint?: string
}


type PipelineNextStep = {
  action?: string
  message?: string
  stage_id?: string
  stage_name?: string
  course_unit_ulid?: string
  course_unit_cc_ulid?: string
  course_id?: string
  program?: string
  exam_id?: string
  form_code?: string
  allow_retake?: boolean
  allow_exemption?: boolean
  status?: string
  pipeline_status?: string
}
type TimelineLog = {
  transition_ulid?: string
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
type CourseSummary = {
  course_id?: string
  title?: string
  category_tips?: string
  duration_min?: number
  status?: string
}

const pipelineStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "PIPELINE", status)

const stageStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "STAGE", status)

const currentUnitStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "COURSE_UNIT", status)

const pipelineIsTerminal = (status?: string | number | null) => {
  const normalized = String(status ?? "").trim()
  return normalized === "3" || normalized === "4"
}

function CourseDetailContent() {
  const searchParams = useSearchParams()
  const pipelineId = searchParams.get("id") || ""
  const { t, lang } = useTranslation()
  const [detail, setDetail] = useState<PipelineDetail | null>(null)
  const [courseSummaries, setCourseSummaries] = useState<Record<string, CourseSummary>>({})
  const [firstCourseThumbnail, setFirstCourseThumbnail] = useState("")
  const [loading, setLoading] = useState(Boolean(pipelineId))
  const [purchaseOpen, setPurchaseOpen] = useState(false)
  const [certificateLoading, setCertificateLoading] = useState(false)
  const [scheduleLoading, setScheduleLoading] = useState(false)
  const purchased = Boolean(detail?.instance && Object.keys(detail.instance).length > 0)
  const instancePipelineId = typeof detail?.instance?.pipeline_ulid === "string" ? detail.instance.pipeline_ulid : ""

  useEffect(() => {
    if (!pipelineId) {
      setDetail(null)
      setLoading(false)
      return
    }

    const loadDetail = async () => {
      setLoading(true)
      try {
        const res = await apiClient(`/api/mall/pipelines/${pipelineId}/runtime`)
        setDetail(res)
      } finally {
        setLoading(false)
      }
    }

    loadDetail()
  }, [pipelineId])

  const pipeline = detail?.config
  const stages = useMemo(() => pipeline?.stages || [], [pipeline])
  const totalUnits = useMemo(() => stages.reduce((total, stage) => total + (stage.units?.length || 0), 0), [stages])
  const firstCourseId = useMemo(
    () => stages.flatMap((stage) => stage.units || []).find((unit) => unit.glms_course_id)?.glms_course_id || "",
    [stages]
  )
  const paymentConfigured = Boolean(pipeline?.unlock_stripe_price_id || pipeline?.package_stripe_price_id)
  const nextStep = detail?.next_step
  const pipelineStatus = detail?.pipeline_status
  const currentStageName = detail?.current_stage_name
  const currentStageStatus = detail?.current_stage_status
  const currentUnitStatus = detail?.current_unit_status
  const nextUnitStatus = nextStep?.status || currentUnitStatus
  const nextStepAction = nextStep?.action || courseUnitNextStepActionFromStatus(nextUnitStatus, Boolean(nextStep?.allow_retake))
  const isPipelineTerminal = pipelineIsTerminal(pipelineStatus)
  const activeStageIndex = useMemo(() => {
    if (!purchased || stages.length === 0) return -1
    if (pipelineIsTerminal(pipelineStatus)) return stages.length
    const nextCourseId = nextStep?.course_id
    if (nextCourseId) {
      const byCourse = stages.findIndex((stage) =>
        (stage.units || []).some((unit) => unit.glms_course_id === nextCourseId)
      )
      if (byCourse >= 0) return byCourse
    }
    const byName = currentStageName
      ? stages.findIndex((stage) => stage.name && stage.name === currentStageName)
      : -1
    return byName >= 0 ? byName : 0
  }, [currentStageName, nextStep?.course_id, pipelineStatus, purchased, stages])
  const stageStateText = (index: number) => {
    if (!purchased) return t.courses.positionNotPurchased
    return stageStatusLabel(t, stages[index]?.runtime_status)
  }
  const stageStateClass = (index: number) => {
    if (!purchased) return "border-slate-200 bg-slate-50 text-slate-600"
    return timelineStatusBadgeClassForStatus("STAGE", stages[index]?.runtime_status)
  }
  const unitStateText = (stageIndex: number, unitIndex: number, unit: UnitConfig) => {
    if (!purchased) return t.courses.positionNotPurchased
    return currentUnitStatusLabel(t, unit.runtime_status)
  }
  const unitStateClass = (stageIndex: number, unitIndex: number, unit: UnitConfig) => {
    if (!purchased) {
      return "border-slate-200 bg-slate-50 text-slate-600"
    }
    return timelineStatusBadgeClassForStatus("COURSE_UNIT", unit.runtime_status)
  }

  const openCertificate = async () => {
    if (!instancePipelineId) return
    setCertificateLoading(true)
    try {
      const res = await apiClient(`/api/pipeline/${instancePipelineId}/certificate-url`)
      if (res?.view_url) {
        window.open(res.view_url, "_blank", "noopener,noreferrer")
      } else {
        toast.error(t.common.error)
      }
    } finally {
      setCertificateLoading(false)
    }
  }

  const handleScheduleExam = async () => {
    const targetPipelineUlid = detail?.instance?.pipeline_ulid
    if (!nextStep?.exam_id || !targetPipelineUlid) return
    setScheduleLoading(true)
    try {
      const termUrlBase = window.location.origin + "/api/public/webhooks/exams/callback"
      const res = await apiClient(`/api/exams/${encodeURIComponent((nextStep as any).exam_id)}/schedule-url?pipeline_ulid=${encodeURIComponent(targetPipelineUlid)}&course_ulid=${encodeURIComponent(nextStep.course_unit_ulid || "")}&url_type=1&term_url_base=${encodeURIComponent(termUrlBase)}`)
      if (res?.url) {
        window.open(res.url, "_blank", "noopener,noreferrer")
      } else {
        toast.error(t.common.error)
      }
    } finally {
      setScheduleLoading(false)
    }
  }

  useEffect(() => {
    if (!purchased) {
      setCourseSummaries({})
      return
    }

    const courseIds = Array.from(
      new Set(
        stages
          .flatMap((stage) => stage.units || [])
          .map((unit) => unit.glms_course_id)
          .filter((id): id is string => Boolean(id))
      )
    )

    if (courseIds.length === 0) {
      setCourseSummaries({})
      return
    }

    let cancelled = false
    Promise.all(
      courseIds.map(async (courseId) => {
        try {
          const res = await apiClient(`/api/mall/courses/${courseId}`)
          return [courseId, res?.course || res] as const
        } catch {
          return [courseId, null] as const
        }
      })
    ).then((items) => {
      if (cancelled) return
      setCourseSummaries(
        Object.fromEntries(items.filter(([, course]) => Boolean(course))) as Record<string, CourseSummary>
      )
    })

    return () => {
      cancelled = true
    }
  }, [stages, purchased])

  useEffect(() => {
    if (!firstCourseId) {
      setFirstCourseThumbnail("")
      return
    }

    let cancelled = false
    const headers = new Headers()
    const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : ""
    if (token) headers.set("Authorization", `Bearer ${token}`)
    fetch(`/api/mall/courses/${encodeURIComponent(firstCourseId)}/thumbnail-url`, {
      credentials: "include",
      headers,
    })
      .then(async (response) => {
        if (!response.ok) return ""
        const data = await response.json()
        return typeof data?.data?.url === "string" ? data.data.url : ""
      })
      .then((url) => {
        if (!cancelled) {
          setFirstCourseThumbnail(url)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setFirstCourseThumbnail("")
        }
      })

    return () => {
      cancelled = true
    }
  }, [firstCourseId])

  const learningHref = (courseId?: string) =>
    courseId
      ? `/courses/learn?courseId=${encodeURIComponent(courseId)}&pipelineId=${encodeURIComponent(pipelineId)}`
      : "/courses"

  const nextStepHref = () => {
    switch (nextStepAction) {
      case "continue_learning":
        return nextStep?.course_id ? learningHref(nextStep.course_id) : learningHref(firstCourseId)
      case "signup_exam":
        return `/exams/signup?unitId=${encodeURIComponent(nextStep?.course_unit_ulid || "")}&pipelineId=${encodeURIComponent(pipelineId)}`
      case "schedule_exam":
      case "view_exam_schedule":
      case "apply_retake":
      case "view_exam_result":
        return "/exams"
      case "view_certificate":
        return instancePipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : "/certificates"
      default:
        return "/courses"
    }
  }

  const nextStepLabel = () => {
    switch (nextStepAction) {
      case "continue_learning":
        return t.courses.openLearning
      case "signup_exam":
        return t.learning.goToExams
      case "schedule_exam":
        return t.learning.actionScheduleExam
      case "view_exam_schedule":
        return t.learning.actionViewExamSchedule
      case "apply_retake":
        return t.learning.actionApplyRetake
      case "view_exam_result":
        return t.learning.actionViewExamResult
      case "view_certificate":
        return t.courses.viewCertificate
      default:
        return t.courses.viewDetails
    }
  }

  const nextStepDescription = () => {
    switch (nextStepAction) {
      case "continue_learning":
        return t.learning.nextStepContinueLearningDesc
      case "signup_exam":
        return t.learning.nextStepGoToExamsDesc
      case "schedule_exam":
      case "view_exam_schedule":
        return t.learning.nextStepGoToExamsDesc
      case "apply_retake":
        return t.learning.nextStepGoToExamsDesc
      case "view_exam_result":
        return t.learning.nextStepGoToExamsDesc
      case "view_certificate":
        return t.learning.nextStepViewCertificateDesc
      default:
        return t.learning.nextStepDesc
    }
  }

  const timelineTitle = t.learning.pipelineTimelineTitle
  const timelineDesc = t.learning.pipelineTimelineDesc
  const timelineEmpty = t.learning.pipelineTimelineEmpty
  const viewTimelineLabel = t.learning.viewTimeline
  const stageBadgeLabel = t.learning.stageOrderLabel
  const handlePurchaseClick = () => {
    if (purchased) return
    setPurchaseOpen(true)
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <Link
            href="/courses"
            className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {t.courses.backToPipelines}
          </Link>

          {loading ? (
            <div className="text-muted-foreground">{t.common.loading}</div>
          ) : !pipeline ? (
            <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">{t.common.na}</div>
          ) : (
            <>
              <div className={cn("mb-8 grid gap-8", firstCourseThumbnail && "lg:grid-cols-[380px_1fr]")}>
                {firstCourseThumbnail && (
                  <div className="relative flex aspect-video items-center justify-center overflow-hidden rounded-lg bg-muted">
                    <img
                      src={firstCourseThumbnail}
                      alt={pipeline.name || t.common.unknownCourse}
                      className="h-full w-full object-cover"
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/45 via-black/5 to-transparent" />
                  </div>
                )}

                <div>
                  <div className="mb-3 flex flex-wrap gap-2">
                    <Badge className="border-0 bg-primary/10 text-primary">{t.courses.pipeline}</Badge>
                    {pipeline.category_tips && <Badge variant="outline">{pipeline.category_tips}</Badge>}
                    {/* {purchased && (
                      <Badge className="border-0 bg-emerald-500/10 text-emerald-700">
                        <CheckCircle2 className="mr-1 h-3 w-3" />
                        {t.courses.purchased}
                      </Badge>
                    )}
                    <Badge variant="outline">{paymentConfigured ? t.courses.configuredPayment : t.courses.noPayment}</Badge> */}
                  </div>
                  <h1 className="mb-2 text-2xl font-bold text-foreground">{pipeline.name || t.common.unknownCourse}</h1>
                  <p className="mb-6 text-muted-foreground">
                    {pipeline.category_tips || t.courses.certificationPath}
                  </p>

                  <div className="mb-6 flex flex-wrap gap-6 text-sm text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <BookOpen className="h-4 w-4" />
                      <span>
                        {stages.length} {t.courses.stages}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-4 w-4" />
                      <span>
                        {totalUnits} {t.courses.units}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Award className="h-4 w-4" />
                      <span>
                        {pipeline.final_quals?.length || 0} {t.credentialsPage.availableQualifications}
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <Button onClick={handlePurchaseClick} disabled={!paymentConfigured || purchased}>
                      <CreditCard className="h-4 w-4" />
                      {purchased ? t.courses.purchased : t.courses.purchasePipeline}
                    </Button>
                    {purchased && instancePipelineId && (
                      <Button variant="outline" onClick={openCertificate} disabled={certificateLoading}>
                        <ExternalLink className="h-4 w-4" />
                        {t.courses.viewCertificate}
                      </Button>
                    )}
                  </div>

                  {purchased && nextStepAction && (
                    <div className="mt-6 rounded-2xl border border-primary/20 bg-primary/5 p-4">
                      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-primary">
                        <Sparkles className="h-4 w-4" />
                        {t.learning.nextStepTitle}
                      </div>
                      <div className="text-sm text-muted-foreground">{nextStepDescription()}</div>
                      <div className="mt-3 flex flex-wrap items-center gap-3">
                        {!isPipelineTerminal && currentStageName && (
                          <Badge variant="outline">
                            {t.learning.currentStageNameLabel}: {currentStageName}
                          </Badge>
                        )}
                        {!isPipelineTerminal && currentStageStatus !== undefined && currentStageStatus !== "" && (
                          <Badge variant="outline" className={timelineStatusBadgeClassForStatus("STAGE", currentStageStatus)}>
                            {t.learning.currentStageStatusLabel}: {stageStatusLabel(t, currentStageStatus)}
                          </Badge>
                        )}
                        {!isPipelineTerminal && currentUnitStatus !== undefined && currentUnitStatus !== "" && (
                          <Badge variant="outline" className={timelineStatusBadgeClassForStatus("COURSE_UNIT", currentUnitStatus)}>
                            {t.learning.unitStatusLabel}: {currentUnitStatusLabel(t, currentUnitStatus)}
                          </Badge>
                        )}
                        {/* <Badge variant="outline">
                          {t.learning.nextStepUnitStatusLabel}: {currentUnitStatusLabel(t, nextUnitStatus)}
                        </Badge> */}
                        {nextStep?.stage_name && <Badge variant="outline">{nextStep.stage_name}</Badge>}
                        {nextStep?.course_id && courseSummaries[nextStep.course_id]?.title && (
                          <Badge variant="outline">
                            {courseSummaries[nextStep.course_id]?.title}
                          </Badge>
                        )}
                        {nextStepAction === "schedule_exam" ? (
                          <Button size="sm" onClick={handleScheduleExam} disabled={scheduleLoading}>
                            {nextStepLabel()}
                            <ArrowRight className="h-4 w-4 ml-1" />
                          </Button>
                        ) : (
                          <Button asChild size="sm">
                            <Link href={nextStepHref()}>
                              {nextStepLabel()}
                              <ArrowRight className="h-4 w-4 ml-1" />
                            </Link>
                          </Button>
                        )}
                      </div>
                    </div>
                  )}

                  {purchased && (
                    <div className="mt-4 rounded-2xl border bg-card p-4">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
                            <Sparkles className="h-4 w-4 text-primary" />
                            {timelineTitle}
                          </div>
                          <p className="text-xs text-muted-foreground">{stageStatusHintLabel(t, currentStageStatus)}</p>
                        </div>
                        <Button asChild size="sm" variant="outline">
                          <Link href={`/courses/timeline?id=${encodeURIComponent(pipelineId)}`}>{viewTimelineLabel}</Link>
                        </Button>
                      </div>
                    </div>
                  )}

                </div>
              </div>

              <section className="space-y-4">
                <div className="flex flex-wrap items-end justify-between gap-3">
                  <div>
                    <h2 className="text-lg font-semibold text-foreground">{t.courses.stageListTitle}</h2>
                    <p className="mt-1 text-sm text-muted-foreground">{t.courses.stageListDesc}</p>
                  </div>
                  <Badge variant="outline">
                    {stages.length} {t.courses.stages} / {totalUnits} {t.courses.units}
                  </Badge>
                </div>
                {stages.length === 0 ? (
                  <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">{t.common.na}</div>
                ) : (
                  stages.map((stage, index) => (
                    <div
                      key={stage.stage_id || index}
                      className={`overflow-hidden rounded-2xl border bg-card shadow-sm ${index === activeStageIndex ? "border-primary/30 ring-1 ring-primary/15" : ""}`}
                    >
                      <div className="flex flex-col gap-4 border-b px-5 py-4 md:flex-row md:items-center md:justify-between">
                        <div className="flex items-center gap-3">
                          <div className={`flex h-10 w-10 items-center justify-center rounded-xl text-sm font-semibold ${index === activeStageIndex ? "bg-primary text-primary-foreground" : "bg-primary/10 text-primary"}`}>
                            {index + 1}
                          </div>
                          <div>
                            <h3 className="font-semibold">{stage.name || `${t.courses.stage} ${index + 1}`}</h3>
                            <p className="text-sm text-muted-foreground">
                              {stage.units?.length || 0} {t.courses.units}
                            </p>
                          </div>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          <Badge variant="outline" className={stageStateClass(index)}>
                            {t.learning.currentStageStatusLabel}: {stageStateText(index)}
                          </Badge>
                          <Badge variant="outline">{stageBadgeLabel} {stage.sort_order || index + 1}</Badge>
                        </div>
                      </div>

                      <div className="divide-y">
                        {(stage.units || []).map((unit, unitIndex) => {
                          const course = unit.glms_course_id ? courseSummaries[unit.glms_course_id] : null
                          const unitKey = unit.unit_id || unit.glms_course_id || `${index}-${unitIndex}`
                          const learningLink = learningHref(unit.glms_course_id)
                          const isCurrentUnit = purchased && index === activeStageIndex && (!nextStep?.course_id || unit.glms_course_id === nextStep.course_id)
                          const canEnterLearning = purchased && Boolean(unit.glms_course_id) && (index <= activeStageIndex || activeStageIndex >= stages.length)
                          const unitDisplayName = course?.title || unit.name || unit.glms_course_id || t.common.unknownCourse
                          const unitMeta = [
                            course?.category_tips,
                            course?.duration_min ? `${course.duration_min} min` : "",
                          ].filter(Boolean).join(" · ")
                          const row = (
                            <>
                              <div className="flex items-center gap-3">
                                {canEnterLearning ? (
                                  <div className={`flex h-8 w-8 items-center justify-center rounded-full ${isCurrentUnit ? "bg-primary text-primary-foreground" : "bg-primary/10 text-primary"}`}>
                                    <Play className="h-3.5 w-3.5 fill-current" />
                                  </div>
                                ) : (
                                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-muted-foreground">
                                    <Lock className="h-3.5 w-3.5" />
                                  </div>
                                )}
                                <div>
                                  <div className="font-medium text-foreground">{unitDisplayName}</div>
                                  {unitMeta && (
                                    <div className="text-xs text-muted-foreground">{unitMeta}</div>
                                  )}
                                </div>
                              </div>
                              <div className="flex flex-wrap items-center justify-end gap-2">
                                <Badge variant="outline" className={unitStateClass(index, unitIndex, unit)}>
                                  {t.learning.unitStatusLabel}: {unitStateText(index, unitIndex, unit)}
                                </Badge>
                                {course?.category_tips && <Badge variant="outline">{course.category_tips}</Badge>}
                                {unit.allow_retake && <Badge variant="outline">{t.courses.reviewCourse}</Badge>}
                                {isCurrentUnit && <Badge variant="default">{t.courses.currentLearningBadge}</Badge>}
                                {canEnterLearning && <Badge variant="default">{t.courses.openLearning}</Badge>}
                              </div>
                            </>
                          )

                          return canEnterLearning ? (
                            <Link
                              key={unitKey}
                              href={learningLink}
                              className="flex items-center justify-between gap-4 px-5 py-3 transition-colors hover:bg-muted/50"
                            >
                              {row}
                            </Link>
                          ) : (
                            <div key={unitKey} className="flex items-center justify-between gap-4 px-5 py-3 opacity-75">
                              {row}
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  ))
                )}
              </section>

              <PurchaseDialog
                open={purchaseOpen}
                onOpenChange={setPurchaseOpen}
                courseName={pipeline.name || t.common.unknownCourse}
                pipelineId={pipeline.pipeline_id}
              />
            </>
          )}
        </div>
      </main>
    </div>
  )
}

export default function CourseDetailPage() {
  return (
    <Suspense fallback={null}>
      <CourseDetailContent />
    </Suspense>
  )
}
