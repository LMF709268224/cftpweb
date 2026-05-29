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
  MoreHorizontal,
  Eye,
  Bookmark,
  ChevronRight
} from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import { useTranslation } from "@/lib/useLanguage"

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

// 资料类型标签
const resourceTypeLabels: any = {
  video: "视频",
  pdf: "PDF",
  document: "文档",
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
            setAllCourses(res.pipelines.map((p: any) => ({
              id: p.pipeline_id,
              title: p.name || t.common.unknownCourse,
              description: `${p.stages?.length || 0} ${t.courses.stages} · ${(p.stages || []).reduce((total: number, stage: any) => total + (stage.units?.length || 0), 0)} ${t.courses.units}`,
              image: "https://images.unsplash.com/photo-1531482615713-2afd69097998?w=800&auto=format&fit=crop&q=60", // placeholder
              category: "course",
              provider: t.courses.certificationPath,
              duration: `${p.stages?.length || 0} ${t.courses.stages}`,
              students: 0,
              isPurchased: false,
              price: 0,
              paymentConfigured: Boolean(p.unlock_stripe_price_id || p.package_stripe_price_id),
              priceLabel: p.unlock_stripe_price_id || p.package_stripe_price_id ? t.courses.configuredPayment : t.courses.noPayment,
            })))
          }
        } else if (activeTab === "my") {
          const res = await apiClient("/api/pipeline")
          if (res?.list) {
            setMyCourses(res.list.map((p: any) => ({
              id: p.pipeline_cc_ulid,
              title: t.common.unknownCourse, // In real app, need to join with static config
              image: "https://images.unsplash.com/photo-1531482615713-2afd69097998?w=800&auto=format&fit=crop&q=60",
              progress: p.progress ? Math.round(p.progress * 100) : 0,
              totalLessons: 10,
              completedLessons: 0,
              totalHours: 20,
              studiedHours: 0,
              lastStudied: t.common.na,
              currentLesson: t.common.na,
              status: p.status === 2 ? "completed" : "learning",
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
                  <div className="flex gap-6">
                    {/* 课程封面 */}
                    <div className="relative h-32 w-48 shrink-0 overflow-hidden rounded-lg">
                      <img
                        src={course.image}
                        alt={course.title}
                        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
                      />
                      {course.status === "completed" && (
                        <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-green-500">
                            <CheckCircle2 className="h-6 w-6 text-white" />
                          </div>
                        </div>
                      )}
                      {course.status === "learning" && (
                        <Link
                          href={`/courses/${course.id}`}
                          className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity group-hover:opacity-100"
                        >
                          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary shadow-lg">
                            <Play className="h-6 w-6 text-primary-foreground ml-1" />
                          </div>
                        </Link>
                      )}
                    </div>

                    {/* 课程信息 */}
                    <div className="flex flex-1 flex-col justify-between">
                      <div>
                        <div className="flex items-start justify-between">
                          <div>
                            <h3 className="text-lg font-semibold text-card-foreground group-hover:text-primary transition-colors">
                              {course.title}
                            </h3>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {course.status === "completed" ? t.courses.completedAll : `${t.courses.currentLearning}${course.currentLesson}`}
                            </p>
                          </div>
                          <Badge 
                            variant={course.status === "completed" ? "default" : "secondary"}
                            className={cn(
                              course.status === "completed" && "bg-green-500 hover:bg-green-600"
                            )}
                          >
                            {course.status === "completed" ? t.courses.completed : t.courses.learning}
                          </Badge>
                        </div>

                        {/* 进度条 */}
                        <div className="mt-4">
                          <div className="mb-2 flex items-center justify-between text-sm">
                            <span className="text-muted-foreground">{t.courses.courseProgress}</span>
                            <span className="font-medium text-foreground">{course.progress}%</span>
                          </div>
                          <div className="h-2 overflow-hidden rounded-full bg-muted">
                            <div
                              className={cn(
                                "h-full rounded-full transition-all duration-500",
                                course.status === "completed" 
                                  ? "bg-green-500" 
                                  : "bg-primary"
                              )}
                              style={{ width: `${course.progress}%` }}
                            />
                          </div>
                        </div>
                      </div>

                      {/* 统计信息 */}
                      <div className="mt-4 flex items-center gap-6 text-sm text-muted-foreground">
                        <div className="flex items-center gap-1.5">
                          <BookOpen className="h-4 w-4" />
                          <span>{course.completedLessons}/{course.totalLessons} {t.courses.units}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <Clock className="h-4 w-4" />
                          <span>{course.studiedHours}/{course.totalHours} {t.courses.hours}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <span>{t.courses.lastStudied}{course.lastStudied}</span>
                        </div>
                      </div>
                    </div>

                    {/* 操作按钮 */}
                    <div className="flex flex-col items-end justify-between">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>
                            <Eye className="mr-2 h-4 w-4" />
                            {t.courses.viewDetails}
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Bookmark className="mr-2 h-4 w-4" />
                            {t.courses.addBookmark}
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Download className="mr-2 h-4 w-4" />
                            {t.courses.downloadMaterials}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>

                      {course.status === "learning" && (
                        <Button asChild className="gap-2">
                          <Link href={`/courses/${course.id}`}>
                            {t.courses.continueLearning}
                            <ChevronRight className="h-4 w-4" />
                          </Link>
                        </Button>
                      )}
                      {course.status === "completed" && (
                        <Button variant="outline" asChild className="gap-2">
                          <Link href={`/courses/${course.id}`}>
                            {t.courses.reviewCourse}
                            <ChevronRight className="h-4 w-4" />
                          </Link>
                        </Button>
                      )}
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
