"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { AlertTriangle, BookOpen, CheckCircle2, Eye, Plus, RefreshCw, Save, Trash2, UploadCloud, Users } from "lucide-react"
import { toast } from "sonner"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { cn, formatBackendDate } from "@/lib/utils"

type LmsCourse = {
  course_id: string
  category_id?: string
  title?: string
  description?: string
  thumbnail_object_key?: string
  duration_min?: number
  certification_enabled?: boolean
  certification_def_id?: string
  is_published?: boolean
  published_at?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type CourseForm = {
  category_id: string
  title: string
  description: string
  thumbnail_object_key: string
  duration_min: string
  certification_enabled: boolean
  certification_def_id: string
}

type CatalogOption = {
  catalog_id: string
  name?: string
  description?: string
}

type CredentialDefinitionOption = {
  cred_def_id: string
  name?: string
  category?: string
}

type CourseEnrollment = {
  enrollment_id: string
  candidate_id: string
  status?: string
  progress_percentage?: number
  joined_at?: string
  completed_at?: string
}

type CandidateProgress = {
  enrollment_id: string
  candidate_id: string
  course_id: string
  status?: string
  progress_percentage?: number
  completed_lesson_ids?: string[]
  passed_quiz_ids?: string[]
  joined_at?: string
  completed_at?: string
}

type BrokenAsset = {
  object_key: string
  asset_type?: string
  status?: string
  error_message?: string
  reconciled_at?: string
  created_at?: string
  updated_at?: string
  course_id?: string
  course_title?: string
  chapter_id?: string
  chapter_title?: string
  lesson_id?: string
  lesson_title?: string
  material_id?: string
  material_title?: string
}

const emptyForm: CourseForm = {
  category_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  duration_min: "",
  certification_enabled: false,
  certification_def_id: "",
}

function formFromCourse(course: LmsCourse | null): CourseForm {
  if (!course) return emptyForm
  return {
    category_id: course.category_id || "",
    title: course.title || "",
    description: course.description || "",
    thumbnail_object_key: course.thumbnail_object_key || "",
    duration_min: course.duration_min ? String(course.duration_min) : "",
    certification_enabled: Boolean(course.certification_enabled),
    certification_def_id: course.certification_def_id || "",
  }
}

function formToPayload(form: CourseForm, version?: number) {
  return {
    category_id: form.category_id.trim(),
    title: form.title.trim(),
    description: form.description.trim(),
    thumbnail_object_key: form.thumbnail_object_key.trim(),
    duration_min: Number(form.duration_min || 0),
    certification_enabled: form.certification_enabled,
    certification_def_id: form.certification_enabled ? form.certification_def_id.trim() : "",
    version: version || 0,
  }
}

export default function LmsCoursesPage() {
  const { t } = useTranslation()
  const page = t.lmsCoursesPage
  const [courses, setCourses] = useState<LmsCourse[]>([])
  const [catalogs, setCatalogs] = useState<CatalogOption[]>([])
  const [credentialDefinitions, setCredentialDefinitions] = useState<CredentialDefinitionOption[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [form, setForm] = useState<CourseForm>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("")
  const [publishedOnly, setPublishedOnly] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [thumbnailUploading, setThumbnailUploading] = useState(false)
  const [preview, setPreview] = useState<any>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [enrollmentStatus, setEnrollmentStatus] = useState("all")
  const [enrollments, setEnrollments] = useState<CourseEnrollment[]>([])
  const [enrollmentsLoading, setEnrollmentsLoading] = useState(false)
  const [progressDetail, setProgressDetail] = useState<CandidateProgress | null>(null)
  const [progressLoadingFor, setProgressLoadingFor] = useState("")
  const [brokenAssetType, setBrokenAssetType] = useState("all")
  const [brokenAssets, setBrokenAssets] = useState<BrokenAsset[]>([])
  const [brokenAssetsLoading, setBrokenAssetsLoading] = useState(false)
  const [brokenAssetsNextPageToken, setBrokenAssetsNextPageToken] = useState("")

  const selectedCourse = useMemo(
    () => courses.find((course) => course.course_id === selectedId) || null,
    [courses, selectedId]
  )

  const loadCourses = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (categoryFilter.trim()) params.set("category_id", categoryFilter.trim())
      if (publishedOnly) params.set("published_only", "true")
      const query = params.toString()
      const res = await apiClient(`/api/lms/courses${query ? `?${query}` : ""}`)
      const nextCourses = res?.courses || []
      setCourses(nextCourses)
      if (selectedId && !nextCourses.some((course: LmsCourse) => course.course_id === selectedId)) {
        setSelectedId("")
        setForm(emptyForm)
        setPreview(null)
        setEnrollments([])
        setProgressDetail(null)
      }
    } finally {
      setLoading(false)
    }
  }, [categoryFilter, publishedOnly, selectedId])

  useEffect(() => {
    loadCourses()
  }, [loadCourses])

  useEffect(() => {
    const loadRelations = async () => {
      const [catalogRes, credentialRes] = await Promise.all([
        apiClient("/api/catalogs"),
        apiClient("/api/credentials/definitions"),
      ])
      setCatalogs(catalogRes?.catalogs || [])
      setCredentialDefinitions(credentialRes?.definitions || [])
    }

    loadRelations().catch((error) => {
      console.error(error)
      toast.error(page.loadRelationsFailed)
    })
  }, [page.loadRelationsFailed])

  const selectCourse = (course: LmsCourse) => {
    setSelectedId(course.course_id)
    setForm(formFromCourse(course))
    setPreview(null)
    setEnrollments([])
    setProgressDetail(null)
  }

  const startNewCourse = () => {
    setSelectedId("")
    setForm(emptyForm)
    setPreview(null)
    setEnrollments([])
    setProgressDetail(null)
  }

  const saveCourse = async () => {
    if (!form.title.trim()) {
      toast.error(page.fillRequired)
      return
    }
    if (form.certification_enabled && !form.certification_def_id.trim()) {
      toast.error(page.selectCertificationDef)
      return
    }

    setSaving(true)
    try {
      if (selectedCourse) {
        await apiClient(`/api/lms/courses/${selectedCourse.course_id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(formToPayload(form, selectedCourse.version)),
        })
        toast.success(page.updateSuccess)
      } else {
        const res = await apiClient("/api/lms/courses", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(formToPayload(form)),
        })
        setSelectedId(res?.course_id || "")
        toast.success(page.createSuccess)
      }
      await loadCourses()
    } finally {
      setSaving(false)
    }
  }

  const persistCourse = async (nextForm: CourseForm, course: LmsCourse) => {
    await apiClient(`/api/lms/courses/${course.course_id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(formToPayload(nextForm, course.version)),
    })
  }

  const uploadThumbnail = async (file: File | null) => {
    if (!file) return
    if (!selectedCourse) {
      toast.error(page.createBeforeThumbnail)
      return
    }

    setThumbnailUploading(true)
    try {
      const contentType = file.type || "application/octet-stream"
      const upload = await apiClient("/api/lms/upload-url", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          upload_type: 1,
          file_name: file.name,
          content_type: contentType,
          course_id: selectedCourse.course_id,
        }),
      })

      const uploadRes = await fetch(upload.upload_url, {
        method: "PUT",
        headers: { "Content-Type": contentType },
        body: file,
      })
      if (!uploadRes.ok) throw new Error(`thumbnail upload failed: ${uploadRes.status}`)

      const nextForm = { ...form, thumbnail_object_key: upload.object_key || "" }
      setForm(nextForm)
      await persistCourse(nextForm, selectedCourse)
      toast.success(page.thumbnailUploadSuccess)
      await loadCourses()
    } catch (error) {
      console.error(error)
      toast.error(page.thumbnailUploadFailed)
    } finally {
      setThumbnailUploading(false)
    }
  }

  const publishCourse = async (publish: boolean) => {
    if (!selectedCourse) return
    await apiClient(`/api/lms/courses/${selectedCourse.course_id}/${publish ? "publish" : "unpublish"}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ version: selectedCourse.version || 0 }),
    })
    toast.success(publish ? page.publishSuccess : page.unpublishSuccess)
    await loadCourses()
  }

  const deleteCourse = async () => {
    if (!selectedCourse || !window.confirm(page.confirmDelete)) return
    await apiClient(`/api/lms/courses/${selectedCourse.course_id}`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ version: selectedCourse.version || 0 }),
    })
    toast.success(page.deleteSuccess)
    setSelectedId("")
    setForm(emptyForm)
    setPreview(null)
    setEnrollments([])
    setProgressDetail(null)
    await loadCourses()
  }

  const loadCompleteCourse = async () => {
    if (!selectedCourse) return
    setPreviewLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/complete`)
      setPreview(res?.complete_course || res)
    } finally {
      setPreviewLoading(false)
    }
  }

  const loadCourseEnrollments = async () => {
    if (!selectedCourse) return
    setEnrollmentsLoading(true)
    try {
      const params = new URLSearchParams({ page_size: "50" })
      if (enrollmentStatus !== "all") params.set("status", enrollmentStatus)
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/enrollments?${params.toString()}`)
      setEnrollments(res?.enrollments || [])
      setProgressDetail(null)
    } finally {
      setEnrollmentsLoading(false)
    }
  }

  const loadCandidateProgress = async (candidateId: string) => {
    if (!selectedCourse || !candidateId) return
    setProgressLoadingFor(candidateId)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/candidates/${candidateId}/progress`)
      setProgressDetail(res)
    } finally {
      setProgressLoadingFor("")
    }
  }

  const loadBrokenAssets = async (pageToken = "") => {
    setBrokenAssetsLoading(true)
    try {
      const params = new URLSearchParams({ page_size: "20" })
      if (pageToken) params.set("page_token", pageToken)
      if (brokenAssetType !== "all") params.set("asset_type", brokenAssetType)
      const res = await apiClient(`/api/lms/broken-assets?${params.toString()}`)
      setBrokenAssets(pageToken ? [...brokenAssets, ...(res?.assets || [])] : res?.assets || [])
      setBrokenAssetsNextPageToken(res?.next_page_token || "")
    } finally {
      setBrokenAssetsLoading(false)
    }
  }

  const describeBrokenAssetOwner = (asset: BrokenAsset) => {
    const owner =
      asset.lesson_title ||
      asset.material_title ||
      asset.chapter_title ||
      asset.course_title ||
      asset.lesson_id ||
      asset.material_id ||
      asset.chapter_id ||
      asset.course_id
    return owner || t.common.na
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-foreground">{page.title}</h1>
              <p className="mt-2 text-muted-foreground">{page.subtitle}</p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={loadCourses} disabled={loading}>
                <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
                {page.refresh}
              </Button>
              <Button onClick={startNewCourse}>
                <Plus className="mr-2 h-4 w-4" />
                {page.newCourse}
              </Button>
            </div>
          </div>

          <div className="mb-4 flex flex-wrap items-end gap-3">
            <div className="w-72 space-y-2">
              <Label htmlFor="categoryFilter">{page.categoryFilter}</Label>
              <Select value={categoryFilter || "all"} onValueChange={(value) => setCategoryFilter(value === "all" ? "" : value)}>
                <SelectTrigger id="categoryFilter" className="w-full">
                  <SelectValue placeholder={page.categoryPlaceholder} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{page.categoryPlaceholder}</SelectItem>
                  {catalogs.map((catalog) => (
                    <SelectItem key={catalog.catalog_id} value={catalog.catalog_id}>
                      {catalog.name || catalog.catalog_id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <label className="flex h-10 items-center gap-2 rounded-md border px-3 text-sm">
              <Checkbox checked={publishedOnly} onCheckedChange={(checked) => setPublishedOnly(Boolean(checked))} />
              {page.publishedOnly}
            </label>
          </div>

          <div className="grid gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <h2 className="font-semibold">{page.courseList}</h2>
                <Badge variant="outline">{courses.length}</Badge>
              </div>
              <div className="max-h-[680px] overflow-y-auto">
                {loading ? (
                  <div className="p-4 text-sm text-muted-foreground">{t.common.loading}</div>
                ) : courses.length === 0 ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{page.noCourses}</div>
                ) : (
                  courses.map((course) => (
                    <button
                      key={course.course_id}
                      onClick={() => selectCourse(course)}
                      className={cn(
                        "flex w-full flex-col gap-2 border-b px-4 py-3 text-left transition-colors last:border-b-0 hover:bg-accent",
                        selectedId === course.course_id && "bg-accent"
                      )}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="truncate font-medium text-foreground">{course.title || t.common.unknownCourse}</span>
                        <Badge variant={course.is_published ? "default" : "outline"}>
                          {course.is_published ? page.published : page.draft}
                        </Badge>
                      </div>
                      <div className="truncate text-xs text-muted-foreground">{course.course_id}</div>
                      <div className="flex items-center justify-between text-xs text-muted-foreground">
                        <span>{course.category_id || t.common.na}</span>
                        <span>
                          {page.version} {course.version || 0}
                        </span>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </section>

            <section className="space-y-4">
              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.courseEditor}</h2>
                  {selectedCourse && (
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" onClick={() => publishCourse(!selectedCourse.is_published)}>
                        <UploadCloud className="mr-2 h-4 w-4" />
                        {selectedCourse.is_published ? page.unpublish : page.publish}
                      </Button>
                      <Button variant="destructive" size="sm" onClick={deleteCourse}>
                        <Trash2 className="mr-2 h-4 w-4" />
                        {page.delete}
                      </Button>
                    </div>
                  )}
                </div>

                <div className="border-b bg-muted/40 px-4 py-3 text-sm text-muted-foreground">{page.courseFlowHint}</div>

                <div className="grid gap-4 p-4 lg:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="title">{page.titleLabel}</Label>
                    <Input id="title" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="categoryId">{page.categorySelect}</Label>
                    <Select value={form.category_id || "none"} onValueChange={(value) => setForm({ ...form, category_id: value === "none" ? "" : value })}>
                      <SelectTrigger id="categoryId" className="w-full">
                        <SelectValue placeholder={page.noCategory} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{page.noCategory}</SelectItem>
                        {catalogs.map((catalog) => (
                          <SelectItem key={catalog.catalog_id} value={catalog.catalog_id}>
                            {catalog.name || catalog.catalog_id}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="duration">{page.durationMin}</Label>
                    <Input
                      id="duration"
                      type="number"
                      min="0"
                      value={form.duration_min}
                      onChange={(event) => setForm({ ...form, duration_min: event.target.value })}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="thumbnail">{page.thumbnailUpload}</Label>
                    <Input
                      id="thumbnail"
                      type="file"
                      accept="image/*"
                      disabled={!selectedCourse || thumbnailUploading}
                      onChange={(event) => {
                        uploadThumbnail(event.target.files?.[0] || null)
                        event.currentTarget.value = ""
                      }}
                    />
                    <div className="truncate text-xs text-muted-foreground">
                      {form.thumbnail_object_key || (selectedCourse ? page.noThumbnail : page.createBeforeThumbnail)}
                    </div>
                  </div>
                  <div className="space-y-2 lg:col-span-2">
                    <Label htmlFor="description">{page.description}</Label>
                    <Textarea
                      id="description"
                      value={form.description}
                      onChange={(event) => setForm({ ...form, description: event.target.value })}
                    />
                  </div>
                  <label className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
                    <Checkbox
                      checked={form.certification_enabled}
                      onCheckedChange={(checked) =>
                        setForm({
                          ...form,
                          certification_enabled: Boolean(checked),
                          certification_def_id: checked ? form.certification_def_id : "",
                        })
                      }
                    />
                    {page.certificationEnabled}
                  </label>
                  <div className="space-y-2">
                    <Label htmlFor="certificationDefId">{page.certificationDefId}</Label>
                    <Select
                      value={form.certification_def_id || "none"}
                      disabled={!form.certification_enabled}
                      onValueChange={(value) => setForm({ ...form, certification_def_id: value === "none" ? "" : value })}
                    >
                      <SelectTrigger id="certificationDefId" className="w-full">
                        <SelectValue placeholder={page.noCertificationDef} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{page.noCertificationDef}</SelectItem>
                        {credentialDefinitions.map((definition) => (
                          <SelectItem key={definition.cred_def_id} value={definition.cred_def_id}>
                            {definition.name || definition.cred_def_id}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {selectedCourse && (
                  <div className="border-t px-4 py-3 text-xs text-muted-foreground">
                    <span className="mr-4">ID: {selectedCourse.course_id}</span>
                    <span className="mr-4">
                      {page.version}: {selectedCourse.version || 0}
                    </span>
                    <span>{formatBackendDate(selectedCourse.updated_at || selectedCourse.created_at)}</span>
                  </div>
                )}

                <div className="flex justify-end border-t px-4 py-3">
                  <Button onClick={saveCourse} disabled={saving}>
                    {selectedCourse ? <Save className="mr-2 h-4 w-4" /> : <Plus className="mr-2 h-4 w-4" />}
                    {selectedCourse ? page.saveCourse : page.createCourse}
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <h2 className="font-semibold">{page.completePreview}</h2>
                  <Button variant="outline" size="sm" onClick={loadCompleteCourse} disabled={!selectedCourse || previewLoading}>
                    {previewLoading ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : <Eye className="mr-2 h-4 w-4" />}
                    {page.loadComplete}
                  </Button>
                </div>
                <div className="p-4">
                  {!selectedCourse ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <BookOpen className="h-4 w-4" />
                      {page.selectCourseHint}
                    </div>
                  ) : preview ? (
                    <pre className="max-h-96 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
                      {JSON.stringify(preview, null, 2)}
                    </pre>
                  ) : (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <CheckCircle2 className="h-4 w-4" />
                      {page.noPreview}
                    </div>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.enrollments}</h2>
                  <div className="flex items-center gap-2">
                    <Select value={enrollmentStatus} onValueChange={setEnrollmentStatus}>
                      <SelectTrigger className="h-9 w-36">
                        <SelectValue placeholder={page.enrollmentStatus} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">{page.statusAll}</SelectItem>
                        <SelectItem value="learning">{page.statusLearning}</SelectItem>
                        <SelectItem value="completed">{page.statusCompleted}</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={loadCourseEnrollments} disabled={!selectedCourse || enrollmentsLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", enrollmentsLoading && "animate-spin")} />
                      {page.loadEnrollments}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  {!selectedCourse ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Users className="h-4 w-4" />
                      {page.selectCourseForEnrollments}
                    </div>
                  ) : enrollments.length === 0 ? (
                    <div className="text-sm text-muted-foreground">{page.noEnrollments}</div>
                  ) : (
                    <div className="overflow-hidden rounded-md border">
                      {enrollments.map((enrollment) => (
                        <div
                          key={enrollment.enrollment_id}
                          className="grid gap-3 border-b px-3 py-3 text-sm last:border-b-0 lg:grid-cols-[minmax(0,1.5fr)_100px_90px_140px_110px]"
                        >
                          <div className="min-w-0">
                            <div className="truncate font-medium">{enrollment.candidate_id}</div>
                            <div className="truncate text-xs text-muted-foreground">{enrollment.enrollment_id}</div>
                          </div>
                          <Badge variant={enrollment.status === "completed" ? "default" : "outline"} className="w-fit">
                            {enrollment.status || t.common.na}
                          </Badge>
                          <div className="text-muted-foreground">
                            {enrollment.progress_percentage || 0}%
                          </div>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(enrollment.joined_at)}</div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => loadCandidateProgress(enrollment.candidate_id)}
                            disabled={progressLoadingFor === enrollment.candidate_id}
                          >
                            {progressLoadingFor === enrollment.candidate_id ? (
                              <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                            ) : (
                              <Eye className="mr-2 h-4 w-4" />
                            )}
                            {page.viewProgress}
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {progressDetail && (
                    <div className="mt-4 rounded-md bg-muted p-3 text-sm">
                      <div className="mb-2 font-medium">{page.progressDetail}</div>
                      <div className="grid gap-2 text-muted-foreground md:grid-cols-2">
                        <div>{page.candidate}: {progressDetail.candidate_id}</div>
                        <div>{page.status}: {progressDetail.status || t.common.na}</div>
                        <div>{page.progress}: {progressDetail.progress_percentage || 0}%</div>
                        <div>{page.completedAt}: {formatBackendDate(progressDetail.completed_at)}</div>
                        <div className="md:col-span-2">
                          {page.completedLessons}: {(progressDetail.completed_lesson_ids || []).join(", ") || t.common.na}
                        </div>
                        <div className="md:col-span-2">
                          {page.passedQuizzes}: {(progressDetail.passed_quiz_ids || []).join(", ") || t.common.na}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.brokenAssets}</h2>
                  <div className="flex items-center gap-2">
                    <Select
                      value={brokenAssetType}
                      onValueChange={(value) => {
                        setBrokenAssetType(value)
                        setBrokenAssets([])
                        setBrokenAssetsNextPageToken("")
                      }}
                    >
                      <SelectTrigger className="h-9 w-36">
                        <SelectValue placeholder={page.assetType} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">{page.assetTypeAll}</SelectItem>
                        <SelectItem value="thumbnail">{page.assetTypeThumbnail}</SelectItem>
                        <SelectItem value="material">{page.assetTypeMaterial}</SelectItem>
                        <SelectItem value="lesson">{page.assetTypeLesson}</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={() => loadBrokenAssets()} disabled={brokenAssetsLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", brokenAssetsLoading && "animate-spin")} />
                      {page.loadBrokenAssets}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  {brokenAssets.length === 0 ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <AlertTriangle className="h-4 w-4" />
                      {page.noBrokenAssets}
                    </div>
                  ) : (
                    <div className="overflow-hidden rounded-md border">
                      {brokenAssets.map((asset) => (
                        <div
                          key={`${asset.object_key}-${asset.asset_type}-${asset.created_at}`}
                          className="grid gap-3 border-b px-3 py-3 text-sm last:border-b-0 lg:grid-cols-[120px_minmax(0,1.4fr)_minmax(0,1fr)_120px]"
                        >
                          <Badge variant="outline" className="w-fit">
                            {asset.asset_type || t.common.na}
                          </Badge>
                          <div className="min-w-0">
                            <div className="truncate font-medium">{asset.object_key}</div>
                            <div className="truncate text-xs text-muted-foreground">{asset.error_message || page.noErrorMessage}</div>
                          </div>
                          <div className="min-w-0 text-muted-foreground">
                            <div className="truncate">{describeBrokenAssetOwner(asset)}</div>
                            <div className="truncate text-xs">{asset.course_id || asset.chapter_id || asset.lesson_id || asset.material_id || t.common.na}</div>
                          </div>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(asset.updated_at || asset.created_at)}</div>
                        </div>
                      ))}
                    </div>
                  )}

                  {brokenAssetsNextPageToken && (
                    <div className="mt-3 flex justify-end">
                      <Button variant="outline" size="sm" onClick={() => loadBrokenAssets(brokenAssetsNextPageToken)} disabled={brokenAssetsLoading}>
                        {page.loadMore}
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            </section>
          </div>
        </div>
      </main>
    </div>
  )
}
