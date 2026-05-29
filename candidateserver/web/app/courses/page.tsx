"use client"

import React from "react"

import { useState, useEffect } from "react"
import Link from "next/link"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { CourseCard } from "@/components/course-card"
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

const pipelineDetailHref = (id: string) => `/courses/detail?id=${encodeURIComponent(id)}`

// 资料类型图标
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
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [resourceFilter, setResourceFilter] = useState<"all" | "video" | "pdf" | "document">("all")

  // State for fetched data
  const [allCourses, setAllCourses] = useState<any[]>([])
  const [myCourses, setMyCourses] = useState<any[]>([])
  const [learningResources, setLearningResources] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  // Fetch data on mount
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        if (activeTab === "all") {
          const res = await apiClient("/api/mall/pipelines")
          // Adjust based on the actual response structure of ListPipelinesRsp
          if (res?.pipelines) {
            setAllCourses(res.pipelines.map((p: any) => {
              const stages = p.stages || []
              const unitCount = stages.reduce((total: number, stage: any) => total + (stage.units?.length || 0), 0)
              const finalQualCount = p.final_quals?.length || 0
              const paymentConfigured = Boolean(p.unlock_stripe_price_id || p.package_stripe_price_id)
              const firstStageNames = stages
                .slice(0, 2)
                .map((stage: any) => stage.name)
                .filter(Boolean)
                .join(" / ")
              return {
                id: p.pipeline_id,
                title: p.name || t.common.unknownCourse,
                description: firstStageNames || `${stages.length} ${t.courses.stages} · ${unitCount} ${t.courses.units}`,
                category: "course",
                provider: p.category_tips || t.courses.certificationPath,
                duration: `${stages.length} ${t.courses.stages}`,
                students: unitCount,
                isPurchased: false,
                price: 0,
                paymentConfigured,
                priceLabel: paymentConfigured ? t.courses.configuredPayment : t.courses.noPayment,
                statusLabel: p.status || t.common.na,
                versionLabel: `${t.courses.version} ${p.version || 0}`,
                stats: [
                  { label: t.courses.stages, value: stages.length },
                  { label: t.courses.units, value: unitCount },
                  { label: t.courses.finalQualifications, value: finalQualCount },
                ],
              }
            }))
          }
        } else if (activeTab === "my") {
          const res = await apiClient("/api/pipeline")
          if (res?.list) {
            setMyCourses(res.list.map((p: any) => ({
              id: p.pipeline_cc_ulid || p.pipeline_ulid,
              instanceId: p.pipeline_ulid,
              currentStageId: p.current_stage_ulid,
              progress: Math.round(Number(p.progress || 0)),
              status: p.status,
              startedAt: p.started_at,
              completedAt: p.completed_at,
            })))
          }
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
                course: t.common.unknownCourse,
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
  }, [activeTab])

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
                <CourseCard key={course.id} {...course} />
              ))}
            </div>
          )}

          {activeTab === "my" && (
            <div className="space-y-4">
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
                              {course.id || t.common.unknownCourse}
                            </h3>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {course.instanceId || t.common.na}
                            </p>
                          </div>
                          <Badge variant="secondary">{course.status || t.common.na}</Badge>
                        </div>

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
                      </div>

                      <div className="mt-4 flex flex-wrap gap-4 text-sm text-muted-foreground">
                        {course.currentStageId && <span>{t.courses.stage}: {course.currentStageId}</span>}
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

              {myCourses.length === 0 && (
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
                          <Button size="sm" className="gap-1.5">
                            <Play className="h-3.5 w-3.5" />
                            {resource.progress > 0 && resource.progress < 100 ? t.courses.continueWatch : t.courses.watch}
                          </Button>
                        )}
                        {(resource.type === "pdf" || resource.type === "document") && (
                          <Button size="sm" className="gap-1.5">
                            <Eye className="h-3.5 w-3.5" />
                            {resource.progress > 0 && resource.progress < 100 ? t.courses.continueRead : t.courses.read}
                          </Button>
                        )}
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <Download className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
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
