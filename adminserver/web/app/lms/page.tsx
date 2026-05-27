"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { BookOpen, CheckCircle2, Eye, Plus, RefreshCw, Save, Trash2, UploadCloud } from "lucide-react"
import { toast } from "sonner"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  const [selectedId, setSelectedId] = useState("")
  const [form, setForm] = useState<CourseForm>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("")
  const [publishedOnly, setPublishedOnly] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [preview, setPreview] = useState<any>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

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
      }
    } finally {
      setLoading(false)
    }
  }, [categoryFilter, publishedOnly, selectedId])

  useEffect(() => {
    loadCourses()
  }, [loadCourses])

  const selectCourse = (course: LmsCourse) => {
    setSelectedId(course.course_id)
    setForm(formFromCourse(course))
    setPreview(null)
  }

  const startNewCourse = () => {
    setSelectedId("")
    setForm(emptyForm)
    setPreview(null)
  }

  const saveCourse = async () => {
    if (!form.title.trim()) {
      toast.error(page.fillRequired)
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
              <Input
                id="categoryFilter"
                value={categoryFilter}
                placeholder={page.categoryPlaceholder}
                onChange={(event) => setCategoryFilter(event.target.value)}
              />
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

                <div className="grid gap-4 p-4 lg:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="title">{page.titleLabel}</Label>
                    <Input id="title" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="categoryId">{page.categoryId}</Label>
                    <Input
                      id="categoryId"
                      value={form.category_id}
                      onChange={(event) => setForm({ ...form, category_id: event.target.value })}
                    />
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
                    <Label htmlFor="thumbnail">{page.thumbnailObjectKey}</Label>
                    <Input
                      id="thumbnail"
                      value={form.thumbnail_object_key}
                      onChange={(event) => setForm({ ...form, thumbnail_object_key: event.target.value })}
                    />
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
                      onCheckedChange={(checked) => setForm({ ...form, certification_enabled: Boolean(checked) })}
                    />
                    {page.certificationEnabled}
                  </label>
                  <div className="space-y-2">
                    <Label htmlFor="certificationDefId">{page.certificationDefId}</Label>
                    <Input
                      id="certificationDefId"
                      value={form.certification_def_id}
                      disabled={!form.certification_enabled}
                      onChange={(event) => setForm({ ...form, certification_def_id: event.target.value })}
                    />
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
            </section>
          </div>
        </div>
      </main>
    </div>
  )
}
