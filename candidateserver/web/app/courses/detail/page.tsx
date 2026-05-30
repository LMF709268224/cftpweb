"use client"

import React, { Suspense, useEffect, useMemo, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { ArrowLeft, Award, BookOpen, CheckCircle2, Clock, CreditCard, Lock, Play } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { PurchaseDialog } from "@/components/purchase-dialog"

type PipelineDetail = {
  config?: PipelineConfig
  instance?: Record<string, any>
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
  units?: UnitConfig[]
}

type UnitConfig = {
  unit_id?: string
  glms_course_id?: string
  stripe_price_id?: string
  allow_retake?: boolean
}

type Qualification = {
  qual_id?: string
  name_hint?: string
}

type CourseSummary = {
  course_id?: string
  title?: string
  category_tips?: string
  duration_min?: number
  status?: string
}

function CourseDetailContent() {
  const searchParams = useSearchParams()
  const pipelineId = searchParams.get("id") || ""
  const { t } = useTranslation()
  const [detail, setDetail] = useState<PipelineDetail | null>(null)
  const [courseSummaries, setCourseSummaries] = useState<Record<string, CourseSummary>>({})
  const [loading, setLoading] = useState(Boolean(pipelineId))
  const [purchaseOpen, setPurchaseOpen] = useState(false)

  useEffect(() => {
    if (!pipelineId) {
      setDetail(null)
      setLoading(false)
      return
    }
    const loadDetail = async () => {
      setLoading(true)
      try {
        const res = await apiClient(`/api/mall/pipelines/${pipelineId}`)
        setDetail(res)
      } finally {
        setLoading(false)
      }
    }
    loadDetail()
  }, [pipelineId])

  const pipeline = detail?.config
  const stages = useMemo(() => pipeline?.stages || [], [pipeline])
  const totalUnits = useMemo(
    () => stages.reduce((total, stage) => total + (stage.units?.length || 0), 0),
    [stages],
  )
  const paymentConfigured = Boolean(pipeline?.unlock_stripe_price_id || pipeline?.package_stripe_price_id)
  const purchased = Boolean(detail?.instance && Object.keys(detail.instance).length > 0)

  useEffect(() => {
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
  }, [stages])

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
              <div className="mb-8 grid gap-8 lg:grid-cols-[380px_1fr]">
                <div className="flex aspect-video items-center justify-center rounded-lg bg-muted">
                  <div className="flex h-16 w-16 items-center justify-center rounded-full bg-card text-primary shadow-sm">
                    {purchased ? <Play className="h-7 w-7 fill-current" /> : <CreditCard className="h-7 w-7" />}
                  </div>
                </div>

                <div>
                  <div className="mb-3 flex flex-wrap gap-2">
                    <Badge className="border-0 bg-primary/10 text-primary">{t.courses.pipeline}</Badge>
                    {pipeline.category_tips && <Badge variant="outline">{pipeline.category_tips}</Badge>}
                    {purchased && (
                      <Badge className="border-0 bg-emerald-500/10 text-emerald-700">
                        <CheckCircle2 className="mr-1 h-3 w-3" />
                        {t.courses.purchased}
                      </Badge>
                    )}
                    <Badge variant="outline">{paymentConfigured ? t.courses.configuredPayment : t.courses.noPayment}</Badge>
                  </div>
                  <h1 className="mb-2 text-2xl font-bold text-foreground">{pipeline.name || t.common.unknownCourse}</h1>
                  <p className="mb-6 text-muted-foreground">{pipeline.pipeline_guid || pipeline.pipeline_id}</p>

                  <div className="mb-6 flex flex-wrap gap-6 text-sm text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <BookOpen className="h-4 w-4" />
                      <span>{stages.length} {t.courses.stages}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-4 w-4" />
                      <span>{totalUnits} {t.courses.units}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Award className="h-4 w-4" />
                      <span>{pipeline.final_quals?.length || 0} {t.credentialsPage.availableQualifications}</span>
                    </div>
                  </div>

                  <Button onClick={() => setPurchaseOpen(true)} disabled={!paymentConfigured || purchased}>
                    <CreditCard className="h-4 w-4" />
                    {purchased ? t.courses.purchased : t.courses.purchasePipeline}
                  </Button>
                </div>
              </div>

              <section className="space-y-4">
                <h2 className="text-lg font-semibold text-foreground">{t.courses.pipelineContent}</h2>
                {stages.length === 0 ? (
                  <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">{t.common.na}</div>
                ) : (
                  stages.map((stage, index) => (
                    <div key={stage.stage_id || index} className="overflow-hidden rounded-lg border bg-card">
                      <div className="flex items-center justify-between border-b px-5 py-4">
                        <div className="flex items-center gap-3">
                          <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-sm font-semibold text-primary">
                            {index + 1}
                          </div>
                          <div>
                            <h3 className="font-semibold">{stage.name || `${t.courses.stage} ${index + 1}`}</h3>
                            <p className="text-sm text-muted-foreground">{stage.units?.length || 0} {t.courses.units}</p>
                          </div>
                        </div>
                        <Badge variant="outline">{stage.sort_order || index + 1}</Badge>
                      </div>

                      <div className="divide-y">
                        {(stage.units || []).map((unit, unitIndex) => {
                          const course = unit.glms_course_id ? courseSummaries[unit.glms_course_id] : null
                          return (
                          <div key={unit.unit_id || unitIndex} className="flex items-center justify-between gap-4 px-5 py-3">
                            <div className="flex items-center gap-3">
                              {purchased ? (
                                <Play className="h-4 w-4 text-primary" />
                              ) : (
                                <Lock className="h-4 w-4 text-muted-foreground" />
                              )}
                              <div>
                                <div className="text-sm font-medium">{course?.title || unit.glms_course_id || `${t.courses.unit} ${unitIndex + 1}`}</div>
                                <div className="text-xs text-muted-foreground">
                                  {unit.glms_course_id || unit.unit_id || t.common.na}
                                  {course?.duration_min ? ` · ${course.duration_min} min` : ""}
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              {course?.category_tips && <Badge variant="outline">{course.category_tips}</Badge>}
                              {unit.allow_retake && <Badge variant="outline">{t.courses.reviewCourse}</Badge>}
                              <Badge variant="outline">{unit.stripe_price_id ? t.courses.configuredPayment : t.courses.noPayment}</Badge>
                            </div>
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
                price={0}
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
