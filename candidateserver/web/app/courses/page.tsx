"use client"

import React from "react"

import { useState, useEffect } from "react"
import Link from "next/link"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { CourseCard } from "@/components/course-card"
import { clearPendingMallPayment } from "@/components/payment-return-handler"
import { cn } from "@/lib/utils"
import { 
  Search, 
  SlidersHorizontal, 
  Play, 
  FileText, 
  Video, 
  FileIcon,
  Download,
  Clock,
  CheckCircle2,
  BookOpen,
  Eye,
  Bookmark,
  ChevronRight
} from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { useTranslation } from "@/lib/useLanguage"
import { toast } from "sonner"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel, timelineStatusBadgeClassForStatus } from "@cftpweb/shared"

const pipelineDetailHref = (id: string) => `/courses/detail?id=${encodeURIComponent(id)}`

const certificationDisplayName = (value?: string) => {
  return String(value || "")
    .replace(/\bPipeline\b/g, "Certification")
    .replace(/管线/g, "认证")
}

const loadPipelineThumbnailUrl = async (pipelineId: string) => {
  if (!pipelineId) return ""
  try {
    const headers = new Headers()
    const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : ""
    if (token) headers.set("Authorization", `Bearer ${token}`)
    const response = await fetch(`/api/mall/pipelines/${encodeURIComponent(pipelineId)}/thumbnail-url`, {
      credentials: "include",
      headers,
    })
    if (!response.ok) return ""
    const data = await response.json()
    return typeof data?.data?.url === "string" ? data.data.url : ""
  } catch {
    return ""
  }
}

// 资料类型图标
const mapCandidatePipeline = (pipeline: any, unknownCourse: string) => ({
  id: pipeline.pipeline_cc_ulid || pipeline.pipeline_ulid,
  instanceId: pipeline.pipeline_ulid,
  configId: pipeline.pipeline_cc_ulid,
  title: certificationDisplayName(pipeline.pipeline_name) || pipeline.pipeline_cc_ulid || pipeline.pipeline_ulid || unknownCourse,
  currentStage: pipeline.current_stage_name || pipeline.current_stage_ulid,
  progress: pipeline.progress_available ? Math.round(Number(pipeline.progress)) : undefined,
  progressAvailable: Boolean(pipeline.progress_available),
  statusValue: pipeline.status,
  startedAt: pipeline.started_at,
  completedAt: pipeline.completed_at,
})

const resourceTypeIcons: any = {
  video: Video,
  pdf: FileText,
  document: FileIcon,
}

// 资料类型颜色
const resourceTypeColors: any = {
  video: "bg-red-100 text-red-600",
  pdf: "bg-orange-100 text-orange-600",
  document: "bg-blue-100 text-blue-600",
}

export default function CoursesPage() {
  const { t, lang } = useTranslation()
  const [activeTab, setActiveTab] = useState("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [resourceFilter, setResourceFilter] = useState<"all" | "video" | "pdf" | "document">("all")
  const [refreshKey, setRefreshKey] = useState(0)

  // State for fetched data
  const [allCourses, setAllCourses] = useState<any[]>([])
  const [myCourses, setMyCourses] = useState<any[]>([])
  const [learningResources, setLearningResources] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  const refreshMyCourses = async () => {
    const res = await apiClient("/api/pipeline")
    const list = Array.isArray(res?.list) ? res.list : []
    const mapped = list.map((pipeline: any) => mapCandidatePipeline(pipeline, t.common.unknownCourse))
    setMyCourses(mapped)
    return mapped
  }

  useEffect(() => {
    if (typeof window === "undefined") return

    const url = new URL(window.location.href)
    const paymentStatus = url.searchParams.get("payment_status")
    if (!paymentStatus) return

    const paymentAction = url.searchParams.get("payment_action")
    const purchasedPipelineId = url.searchParams.get("pipeline_id")
    const isUnlock = paymentAction === "unlock"
    const copy = {
      purchaseSuccess: lang === "zh" ? "购买成功，课程列表已刷新。" : "Purchase successful. The course list has been refreshed.",
      unlockSuccess: lang === "zh" ? "解锁成功，课程列表已刷新。" : "Unlock successful. The course list has been refreshed.",
      cancelled: lang === "zh" ? "支付已取消，你可以稍后继续处理订单。" : "Payment cancelled. You can continue the order later.",
      failed: lang === "zh" ? "支付失败，请稍后重试或联系管理员。" : "Payment failed. Please try again later or contact support.",
    }

    if (paymentStatus === "success") {
      toast.success(isUnlock ? copy.unlockSuccess : copy.purchaseSuccess)
      if (!isUnlock) {
        const refreshPurchasedEligibility = async () => {
          if (!purchasedPipelineId) return
          try {
            await apiClient(`/api/mall/pipelines/${encodeURIComponent(purchasedPipelineId)}/eligibility`)
            setAllCourses((courses) => courses.map((course) => (
              course.id === purchasedPipelineId
                ? { ...course, eligibilityRefreshKey: Date.now() }
                : course
            )))
          } catch (error) {
            console.error(error)
          }
        }

        void refreshPurchasedEligibility()
      }
    } else if (paymentStatus === "cancelled") {
      toast.warning(copy.cancelled)
    } else if (paymentStatus === "failed") {
      toast.error(copy.failed)
    }

    clearPendingMallPayment()
    setRefreshKey((value) => value + 1)
    url.searchParams.delete("payment_status")
    url.searchParams.delete("payment_action")
    url.searchParams.delete("order_id")
    url.searchParams.delete("pipeline_id")
    window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)
  }, [lang])

  // Fetch data on mount
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        if (activeTab === "all") {
          const res = await apiClient("/api/mall/pipelines")
          // Adjust based on the actual response structure of ListPipelinesRsp
          if (res?.pipelines) {
            const cards = await Promise.all(res.pipelines.map(async (p: any) => {
              const stages = p.stages || []
              const unitCount = stages.reduce((total: number, stage: any) => total + (stage.units?.length || 0), 0)
              const finalQualCount = p.final_quals?.length || 0
              const image = await loadPipelineThumbnailUrl(p.pipeline_id)
              const firstStageNames = stages
                .slice(0, 2)
                .map((stage: any) => stage.name)
                .filter(Boolean)
                .join(" / ")
              return {
                id: p.pipeline_id,
                title: certificationDisplayName(p.name) || t.common.unknownCourse,
                description: firstStageNames || `${stages.length} ${t.courses.stages} · ${unitCount} ${t.courses.units}`,
                category: "course",
                provider: p.category_tips || t.courses.certificationPath,
                isPurchased: false,
                image,
                students: typeof p.purchase_count === "number" ? p.purchase_count : undefined,
                versionLabel: `${t.courses.version} ${p.version || 0}`,
                stats: [
                  { label: t.courses.stages, value: stages.length },
                  { label: t.courses.units, value: unitCount },
                  { label: t.courses.finalQualifications, value: finalQualCount },
                ],
              }
            }))
            setAllCourses(cards)
          }
        } else if (activeTab === "my") {
          await refreshMyCourses()
        } else if (activeTab === "resources") {
          const res = await apiClient("/api/pipeline/materials")
          if (res?.materials) {
            setLearningResources(res.materials.map((m: any) => {
              // Map material type
              let type = "document";
              if (m.type === 1) type = "video";
              else if (m.type === 2) type = "pdf";

              return {
                id: m.id,
                title: m.title || t.common.unknown,
                type: type,
                duration: m.duration_seconds ? `${Math.floor(m.duration_seconds/60)}:${m.duration_seconds%60}` : "",
                course: m.course_title || m.course_id || t.common.unknownCourse,
                size: m.file_size ? `${Math.round(m.file_size / 1024 / 1024)} MB` : t.common.unknown,
                isWatched: m.progress_value === 100,
                progress: m.progress_value || 0,
              }
            }))
          }
        }
      } catch (e) {
        console.error(e)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [activeTab, refreshKey])

  const tabs = [
    { id: "all", label: t.courses.tabs.all },
    { id: "my", label: t.courses.tabs.my },
    { id: "resources", label: t.courses.tabs.materials },
  ]

  const filteredCourses = allCourses.filter((course) => 
    course.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    course.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const filteredResources = learningResources.filter((resource) => {
    const matchesSearch = resource.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      resource.course.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesFilter = resourceFilter === "all" || resource.type === resourceFilter
    return matchesSearch && matchesFilter
  })

  const openResource = async (resource: any) => {
    if (!resource?.id) return
    try {
      const res = await apiClient(`/api/pipeline/materials/${resource.id}/url`)
      if (res?.url) {
        window.open(res.url, "_blank", "noopener,noreferrer")
      } else {
        toast.error(t.common.error)
      }
    } catch {
      // apiClient already shows the localized error.
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.courses.title}</h1>
            <p className="mt-1 text-muted-foreground">{t.courses.subtitle}</p>
          </div>

          {/* Search and Filter Bar */}
          <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="relative max-w-md flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t.courses.searchPlaceholder}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            {activeTab === "resources" && (
              <div className="flex gap-2">
                {(["all", "video", "pdf", "document"] as const).map((filter) => (
                  <Button
                    key={filter}
                    variant={resourceFilter === filter ? "default" : "outline"}
                    size="sm"
                    onClick={() => setResourceFilter(filter)}
                    className="gap-1.5"
                  >
                    {filter === "all" && t.messagesPage.all}
                    {filter === "video" && <><Video className="h-3.5 w-3.5" />{t.courses.video}</>}
                    {filter === "pdf" && <><FileText className="h-3.5 w-3.5" />{t.courses.pdf}</>}
                    {filter === "document" && <><FileIcon className="h-3.5 w-3.5" />{t.courses.document}</>}
                  </Button>
                ))}
              </div>
            )}
            {activeTab !== "resources" && (
              <Button variant="outline" size="sm" className="gap-2">
                <SlidersHorizontal className="h-4 w-4" />
                {t.courses.filterBtn}
              </Button>
            )}
          </div>

          {/* Tabs */}
          <div className="mb-8 flex gap-1 rounded-xl bg-muted p-1 w-fit">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => {
                  setActiveTab(tab.id)
                  setSearchQuery("")
                }}
                className={cn(
                  "px-4 py-2 text-sm font-medium rounded-lg transition-all duration-200",
                  activeTab === tab.id
                    ? "bg-card text-card-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          {activeTab === "all" && (
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
              {filteredCourses.map((course) => (
                <CourseCard key={`${course.id}-${course.eligibilityRefreshKey || 0}`} {...course} />
              ))}
            </div>
          )}

          {activeTab === "my" && (
            <div className="space-y-4">
              {loading && myCourses.length === 0 && (
                <div className="flex items-center justify-center gap-2 rounded-xl border border-border bg-card py-12 text-muted-foreground">
                  <Clock className="h-5 w-5 animate-spin" />
                  <span>{lang === "zh" ? "正在刷新我的认证..." : "Refreshing my certifications..."}</span>
                </div>
              )}

              {myCourses.map((course) => (
                <div
                  key={course.id}
                  className="group relative overflow-hidden rounded-xl border border-border bg-card p-6 transition-all duration-300 hover:shadow-lg hover:border-primary/20"
                >
                  <div className="flex gap-5">
                    <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                      <BookOpen className="h-7 w-7" />
                    </div>

                    <div className="flex flex-1 flex-col justify-between">
                      <div>
                        <div className="flex items-start justify-between">
                          <div>
                            <h3 className="text-lg font-semibold text-card-foreground group-hover:text-primary transition-colors">
                              {course.title || t.common.unknownCourse}
                            </h3>
                          </div>
                          <Badge variant="outline" className={timelineStatusBadgeClassForStatus("PIPELINE", course.statusValue)}>
                            {statusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, course.statusValue)}
                          </Badge>
                        </div>

                        {course.progressAvailable && (
                        <div className="mt-4">
                          <div className="mb-2 flex items-center justify-between text-sm">
                            <span className="text-muted-foreground">{t.courses.courseProgress}</span>
                            <span className="font-medium text-foreground">{course.progress}%</span>
                          </div>
                          <div className="h-2 overflow-hidden rounded-full bg-muted">
                            <div
                              className="h-full rounded-full bg-primary transition-all duration-500"
                              style={{ width: `${course.progress}%` }}
                            />
                          </div>
                        </div>
                        )}
                      </div>

                      <div className="mt-4 flex flex-wrap gap-4 text-sm text-muted-foreground">
                        {course.currentStage && <span>{t.courses.stage}: {course.currentStage}</span>}
                        {course.startedAt && <span>{course.startedAt}</span>}
                        {course.completedAt && <span>{course.completedAt}</span>}
                      </div>
                    </div>

                    <div className="flex flex-col items-end justify-between">
                      <Button asChild className="gap-2">
                        <Link href={pipelineDetailHref(course.id)}>
                          {t.courses.viewDetails}
                          <ChevronRight className="h-4 w-4" />
                        </Link>
                      </Button>
                    </div>
                  </div>
                </div>
              ))}

              {!loading && myCourses.length === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                    <BookOpen className="h-8 w-8 text-muted-foreground" />
                  </div>
                  <h3 className="text-lg font-semibold text-foreground mb-2">{t.courses.noCourses}</h3>
                  <p className="text-muted-foreground mb-4">{t.courses.noCoursesDesc}</p>
                  <Button onClick={() => setActiveTab("all")}>
                    {t.courses.browseCoursesBtn}
                  </Button>
                </div>
              )}
            </div>
          )}

          {activeTab === "resources" && (
            <div className="space-y-3">
              {filteredResources.map((resource) => {
                const TypeIcon = resourceTypeIcons[resource.type]
                return (
                  <div
                    key={resource.id}
                    onClick={() => openResource(resource)}
                    className="group relative overflow-hidden rounded-xl border border-border bg-card p-4 transition-all duration-300 hover:shadow-md hover:border-primary/20 cursor-pointer"
                  >
                    <div className="flex items-center gap-4">
                      {/* 类型图标 */}
                      <div className={cn(
                        "flex h-12 w-12 shrink-0 items-center justify-center rounded-xl",
                        resourceTypeColors[resource.type]
                      )}>
                        <TypeIcon className="h-6 w-6" />
                      </div>

                      {/* 资料信息 */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="font-medium text-card-foreground truncate group-hover:text-primary transition-colors">
                            {resource.title}
                          </h3>
                          {resource.progress === 100 && (
                            <CheckCircle2 className="h-4 w-4 shrink-0 text-green-500" />
                          )}
                        </div>
                        <div className="mt-1 flex items-center gap-3 text-sm text-muted-foreground">
                          <Badge variant="outline" className="text-xs">
                            {resource.course}
                          </Badge>
                          <span>{t.courses[resource.type as keyof typeof t.courses] || resource.type}</span>
                          {resource.type === "video" && (
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {resource.duration}
                            </span>
                          )}
                          {(resource.type === "pdf" || resource.type === "document") && (
                            <span>{resource.pages} {t.courses.pages}</span>
                          )}
                          <span>{resource.size}</span>
                        </div>
                      </div>

                      {/* 进度指示 */}
                      {resource.progress > 0 && resource.progress < 100 && (
                        <div className="flex items-center gap-2 text-sm">
                          <div className="w-24 h-1.5 rounded-full bg-muted overflow-hidden">
                            <div 
                              className="h-full bg-primary rounded-full"
                              style={{ width: `${resource.progress}%` }}
                            />
                          </div>
                          <span className="text-muted-foreground w-10 text-right">{resource.progress}%</span>
                        </div>
                      )}

                      {/* 操作按钮 */}
                      <div className="flex items-center gap-2">
                        {resource.type === "video" && (
                          <Button size="sm" className="gap-1.5" onClick={(event) => { event.stopPropagation(); openResource(resource) }}>
                            <Play className="h-3.5 w-3.5" />
                            {resource.progress > 0 && resource.progress < 100 ? t.courses.continueWatch : t.courses.watch}
                          </Button>
                        )}
                        {(resource.type === "pdf" || resource.type === "document") && (
                          <Button size="sm" className="gap-1.5" onClick={(event) => { event.stopPropagation(); openResource(resource) }}>
                            <Eye className="h-3.5 w-3.5" />
                            {resource.progress > 0 && resource.progress < 100 ? t.courses.continueRead : t.courses.read}
                          </Button>
                        )}
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={(event) => { event.stopPropagation(); openResource(resource) }}>
                          <Download className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={(event) => event.stopPropagation()}>
                          <Bookmark className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                )
              })}

              {filteredResources.length === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                    <FileText className="h-8 w-8 text-muted-foreground" />
                  </div>
                  <h3 className="text-lg font-semibold text-foreground mb-2">{t.courses.noResources}</h3>
                  <p className="text-muted-foreground">{t.courses.noResourcesDesc}</p>
                </div>
              )}
            </div>
          )}

          {/* Empty State for all courses */}
          {activeTab === "all" && filteredCourses.length === 0 && (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                <Search className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">{t.common.na}</h3>
              <p className="text-muted-foreground">{t.common.na}</p>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
