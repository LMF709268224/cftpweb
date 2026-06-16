"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { BookOpen, CheckCircle2, Copy, Plus, RefreshCw, Save, Send, Trash2 } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ADMIN_PIPELINE_STATUS_LABELS, LMS_COURSE_STATUS_LABELS, statusLabel } from "@cftpweb/shared"

type Pipeline = {
  pipeline_id: string
  pipeline_guid: string
  version: number
  name: string
  category_tips?: string
  status: string
  is_current: boolean
  created_at: string
  unlock_stripe_product_id?: string
  unlock_stripe_price_id?: string
  package_stripe_product_id?: string
  package_stripe_price_id?: string
  respath?: string
  certs?: Qualification[]
  certs_quals?: Qualification[]
  cert_quals?: Qualification[]
  final_quals?: Qualification[]
  stages?: StageConfig[]
}

type Qualification = {
  qual_id?: string
  name_hint?: string
  name?: string
  title?: string
}

type StageConfig = {
  stage_id?: string
  name: string
  sort_order: number
  units: UnitConfig[]
}

type UnitConfig = {
  unit_id?: string
  name?: string
  glms_course_id: string
  program?: string
  exam_id?: string
  form_code?: string
  stripe_product_id?: string
  stripe_price_id?: string
  exemption_stripe_product_id?: string
  exemption_stripe_price_id?: string
  retake_stripe_product_id?: string
  retake_stripe_price_id?: string
  allow_retake?: boolean
  exemption_quals?: string[]
}

type LmsCourse = {
  course_id: string
  course_guid?: string
  title?: string
  status?: string
  is_published?: boolean
  is_current?: boolean
  version?: number
}

type PipelineForm = {
  name: string
  category_tips: string
  unlock_stripe_product_id: string
  unlock_stripe_price_id: string
  package_stripe_product_id: string
  package_stripe_price_id: string
  respath: string
  certs: Qualification[]
  certs_quals: Qualification[]
  cert_quals: Qualification[]
  final_quals: Qualification[]
  stages: StageConfig[]
}

const emptyForm: PipelineForm = {
  name: "",
  category_tips: "",
  unlock_stripe_product_id: "",
  unlock_stripe_price_id: "",
  package_stripe_product_id: "",
  package_stripe_price_id: "",
  respath: "",
  certs: [],
  certs_quals: [],
  cert_quals: [],
  final_quals: [],
  stages: [],
}

function normalizeQualifications(value: unknown): Qualification[] {
  return Array.isArray(value) ? value.filter(Boolean) as Qualification[] : []
}

function pipelineToForm(pipeline: Pipeline | null): PipelineForm {
  if (!pipeline) return emptyForm
  return {
    name: pipeline.name || "",
    category_tips: pipeline.category_tips || "",
    unlock_stripe_product_id: pipeline.unlock_stripe_product_id || "",
    unlock_stripe_price_id: pipeline.unlock_stripe_price_id || "",
    package_stripe_product_id: pipeline.package_stripe_product_id || "",
    package_stripe_price_id: pipeline.package_stripe_price_id || "",
    respath: pipeline.respath || "",
    certs: normalizeQualifications(pipeline.certs),
    certs_quals: normalizeQualifications(pipeline.certs_quals),
    cert_quals: normalizeQualifications(pipeline.cert_quals),
    final_quals: normalizeQualifications(pipeline.final_quals),
    stages: (pipeline.stages || []).map((stage) => ({
      stage_id: stage.stage_id,
      name: stage.name || "",
      sort_order: Number(stage.sort_order || 0),
      units: (stage.units || []).map((unit) => ({
        unit_id: unit.unit_id,
        name: unit.name || "",
        glms_course_id: unit.glms_course_id || "",
        program: unit.program || "",
        exam_id: unit.exam_id || "",
        form_code: unit.form_code || "",
        stripe_product_id: unit.stripe_product_id || "",
        stripe_price_id: unit.stripe_price_id || "",
        exemption_stripe_product_id: unit.exemption_stripe_product_id || "",
        exemption_stripe_price_id: unit.exemption_stripe_price_id || "",
        retake_stripe_product_id: unit.retake_stripe_product_id || "",
        retake_stripe_price_id: unit.retake_stripe_price_id || "",
        allow_retake: Boolean(unit.allow_retake),
        exemption_quals: unit.exemption_quals || [],
      })),
    })),
  }
}

function cleanFormForStructure(form: PipelineForm) {
  return {
    unlock_stripe_product_id: form.unlock_stripe_product_id.trim(),
    unlock_stripe_price_id: form.unlock_stripe_price_id.trim(),
    package_stripe_product_id: form.package_stripe_product_id.trim(),
    package_stripe_price_id: form.package_stripe_price_id.trim(),
    certs: form.certs || [],
    certs_quals: form.certs_quals || [],
    stages: form.stages.map((stage) => ({
      stage_id: stage.stage_id || "",
      name: stage.name.trim(),
      sort_order: Number(stage.sort_order || 0),
      units: stage.units.map((unit) => ({
        unit_id: unit.unit_id || "",
        name: (unit.name || "").trim(),
        glms_course_id: unit.glms_course_id.trim(),
        program: (unit.program || "").trim(),
        exam_id: (unit.exam_id || "").trim(),
        form_code: (unit.form_code || "").trim(),
        stripe_product_id: (unit.stripe_product_id || "").trim(),
        stripe_price_id: (unit.stripe_price_id || "").trim(),
        exemption_stripe_product_id: (unit.exemption_stripe_product_id || "").trim(),
        exemption_stripe_price_id: (unit.exemption_stripe_price_id || "").trim(),
        retake_stripe_product_id: (unit.retake_stripe_product_id || "").trim(),
        retake_stripe_price_id: (unit.retake_stripe_price_id || "").trim(),
        allow_retake: Boolean(unit.allow_retake),
        exemption_quals: unit.exemption_quals || [],
      })),
    })),
  }
}

function isPublished(pipeline: Pipeline | null) {
  return Boolean(pipeline?.status?.toLowerCase() === "active" || (pipeline?.is_current && pipeline?.status?.toLowerCase() !== "deprecated"))
}

function isDeprecated(pipeline: Pipeline | null) {
  return Boolean(pipeline?.status?.toLowerCase() === "deprecated")
}

function qualificationLabel(qualification: Qualification, fallback: string) {
  return qualification.name_hint || qualification.name || qualification.title || qualification.qual_id || fallback
}

function pipelineCertificateItems(form: PipelineForm) {
  return normalizeQualifications(form.certs).filter((item, index, list) => {
    const key = item.qual_id || item.name_hint || item.name || item.title || String(index)
    return list.findIndex((candidate) => (candidate.qual_id || candidate.name_hint || candidate.name || candidate.title || "") === key) === index
  })
}

export default function PipelinesPage() {
  const { t } = useTranslation()
  const page = t.pipelinesPage
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [lmsCourses, setLmsCourses] = useState<LmsCourse[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [form, setForm] = useState<PipelineForm>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("")
  const [onlyCurrent, setOnlyCurrent] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [offset, setOffset] = useState(0)
  const limit = 20

  const [lmsCourseDetails, setLmsCourseDetails] = useState<Record<string, any>>({})

  const fetchCourseDetails = useCallback(async (courseIds: string[]) => {
    const missingIds = courseIds.filter((id) => id && !lmsCourseDetails[id])
    if (missingIds.length === 0) return

    for (const id of missingIds) {
      try {
        const res = await apiClient(`/api/lms/courses/${id}/complete`)
        setLmsCourseDetails((prev) => ({
          ...prev,
          [id]: res?.complete_course || res,
        }))
      } catch (e) {
        console.error("Failed to load details for", id, e)
      }
    }
  }, [lmsCourseDetails])

  useEffect(() => {
    const allCourseIds = form.stages.flatMap((s) => s.units.map((u) => u.glms_course_id)).filter(Boolean)
    const uniqueIds = Array.from(new Set(allCourseIds))
    fetchCourseDetails(uniqueIds)
  }, [form.stages, fetchCourseDetails])

  const selectedPipeline = useMemo(
    () => pipelines.find((pipeline) => pipeline.pipeline_id === selectedId) || null,
    [pipelines, selectedId],
  )
  const published = isPublished(selectedPipeline)
  const certificateItems = useMemo(() => pipelineCertificateItems(form), [form])
  const hasCertificate = certificateItems.length > 0
  const certificateNames = useMemo(
    () => certificateItems.map((item, index) => qualificationLabel(item, `${page.certificate} ${index + 1}`)),
    [certificateItems, page.certificate],
  )

  const lmsCourseName = (courseId: string) => {
    const course = lmsCourses.find((item) => item.course_id === courseId)
    return course?.title || courseId || t.common.na
  }

  const loadPipelines = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (categoryFilter.trim()) params.set("category_tips", categoryFilter.trim())
      if (onlyCurrent) params.set("only_current", "true")
      params.set("limit", String(limit))
      params.set("offset", String(offset))
      const res = await apiClient(`/api/pipelines?${params.toString()}`)
      const nextPipelines = res?.pipelines || []
      setPipelines(nextPipelines)
      if (selectedId && !nextPipelines.some((pipeline: Pipeline) => pipeline.pipeline_id === selectedId)) {
        setSelectedId("")
        setForm(emptyForm)
      }
    } finally {
      setLoading(false)
    }
  }, [categoryFilter, offset, onlyCurrent, selectedId])

  const loadLmsCourses = useCallback(async () => {
    const params = new URLSearchParams()
    params.set("published_only", "true")
    params.set("page_size", "200")
    const res = await apiClient(`/api/lms/courses?${params.toString()}`)
    setLmsCourses(res?.courses || [])
  }, [])

  useEffect(() => {
    loadPipelines()
  }, [loadPipelines])

  useEffect(() => {
    loadLmsCourses().catch(() => toast.error(page.loadLmsFailed))
  }, [loadLmsCourses, page.loadLmsFailed])

  const selectPipeline = async (pipeline: Pipeline) => {
    setSelectedId(pipeline.pipeline_id)
    try {
      const detail = await apiClient(`/api/pipelines/${pipeline.pipeline_id}`)
      setForm(pipelineToForm(detail))
    } catch {
      setForm(pipelineToForm(pipeline))
    }
  }

  const updateStage = (index: number, patch: Partial<StageConfig>) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.map((stage, currentIndex) => (currentIndex === index ? { ...stage, ...patch } : stage)),
    }))
  }

  const updateUnit = (stageIndex: number, unitIndex: number, patch: Partial<UnitConfig>) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.map((stage, currentStageIndex) =>
        currentStageIndex === stageIndex
          ? {
              ...stage,
              units: stage.units.map((unit, currentUnitIndex) =>
                currentUnitIndex === unitIndex ? { ...unit, ...patch } : unit,
              ),
            }
          : stage,
      ),
    }))
  }

  const createPipeline = async () => {
    if (!form.name.trim()) {
      toast.error(page.fillName)
      return
    }
    if (!form.category_tips.trim()) {
      toast.error(page.fillCategoryTips)
      return
    }
    if (!form.respath.trim()) {
      toast.error("Please fill in respath")
      return
    }
    setCreating(true)
    try {
      const res = await apiClient("/api/pipelines", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: form.name.trim(),
          category_tips: form.category_tips.trim(),
          respath: form.respath.trim(),
        }),
      })
      toast.success(page.createSuccess)
      setSelectedId(res?.pipeline_id || "")
      try {
        const detail = await apiClient(`/api/pipelines/${res?.pipeline_id}`)
        setForm(pipelineToForm(detail))
      } catch {
        setForm(pipelineToForm(res))
      }
      await loadPipelines()
    } finally {
      setCreating(false)
    }
  }

  const saveMetadata = async () => {
    if (!selectedPipeline) return
    if (!form.name.trim()) {
      toast.error(page.fillName)
      return
    }
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${selectedPipeline.pipeline_id}/metadata`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ new_name: form.name.trim() }),
      })
      toast.success(page.metadataSuccess)
      await loadPipelines()
    } finally {
      setSaving(false)
    }
  }

  const validateStructure = () => {
    if (!selectedPipeline) return false
    if (form.stages.length === 0) {
      toast.error(page.stageRequired)
      return false
    }
    for (let stageIndex = 0; stageIndex < form.stages.length; stageIndex += 1) {
      const stage = form.stages[stageIndex]
      if (!stage.name.trim()) {
        toast.error(page.stageNameRequired.replace("{{index}}", String(stageIndex + 1)))
        return false
      }
      if (stage.units.length === 0) {
        toast.error(page.unitRequired.replace("{{stage}}", stage.name || String(stageIndex + 1)))
        return false
      }
      for (let unitIndex = 0; unitIndex < stage.units.length; unitIndex += 1) {
        const unit = stage.units[unitIndex]
        if (!unit.name?.trim()) {
          toast.error(page.unitNameRequired.replace("{{stage}}", stage.name || String(stageIndex + 1)).replace("{{index}}", String(unitIndex + 1)))
          return false
        }
        if (!unit.glms_course_id.trim()) {
          toast.error(page.unitCourseRequired.replace("{{stage}}", stage.name || String(stageIndex + 1)).replace("{{index}}", String(unitIndex + 1)))
          return false
        }
      }
    }
    return true
  }

  const saveStructure = async () => {
    if (!selectedPipeline) return
    if (published) {
      toast.error(page.publishedLocked)
      return
    }
    if (!validateStructure()) return
    setSaving(true)
    try {
      const res = await apiClient(`/api/pipelines/${selectedPipeline.pipeline_id}/structure`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cleanFormForStructure(form)),
      })
      toast.success(page.structureSuccess)
      setForm(pipelineToForm(res))
      await loadPipelines()
    } finally {
      setSaving(false)
    }
  }

  const publishPipeline = async () => {
    if (!selectedPipeline) return
    if (!validateStructure()) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${selectedPipeline.pipeline_id}/publish`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      })
      toast.success(page.publishSuccess)
      await loadPipelines()
    } finally {
      setSaving(false)
    }
  }


  const deprecatePipeline = async () => {
    if (!selectedPipeline) return
    if (!window.confirm("Confirm deprecate?")) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${selectedPipeline.pipeline_id}/deprecate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      })
      toast.success(t.common.success)
      await loadPipelines()
    } finally {
      setSaving(false)
    }
  }

  const clonePipeline = async () => {
    if (!selectedPipeline) return
    if (!form.respath.trim()) {
      toast.error("Please fill in respath")
      return
    }
    setCreating(true)
    try {
      const res = await apiClient("/api/pipelines", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: form.name.trim() + " (Copy)",
          category_tips: form.category_tips.trim(),
          respath: form.respath.trim(),
          from_pipeline_id: selectedPipeline.pipeline_id,
        }),
      })
      const newPipelineId = res?.pipeline_id || ""
      toast.success(page.createSuccess)
      setSelectedId(newPipelineId)
      try {
        const detail = await apiClient(`/api/pipelines/${newPipelineId}`)
        setForm(pipelineToForm(detail))
      } catch {
        setForm(pipelineToForm(res))
      }
      await loadPipelines()
    } finally {
      setCreating(false)
    }
  }

  const deletePipeline = async () => {

    if (!selectedPipeline) return
    if (published) {
      toast.error(page.deletePublishedBlocked)
      return
    }
    if (!window.confirm(page.confirmDelete)) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${selectedPipeline.pipeline_id}`, { method: "DELETE" })
      toast.success(page.deleteSuccess)
      setSelectedId("")
      setForm(emptyForm)
      await loadPipelines()
    } finally {
      setSaving(false)
    }
  }

  const addStage = () => {
    setForm((prev) => ({
      ...prev,
      stages: [
        ...prev.stages,
        {
          name: "",
          sort_order: prev.stages.length + 1,
          units: [],
        },
      ],
    }))
  }

  const addUnit = (stageIndex: number) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.map((stage, index) =>
        index === stageIndex
          ? {
              ...stage,
              units: [
                ...stage.units,
                {
                  name: "",
                  glms_course_id: "",
                  program: "",
                  exam_id: "",
                  form_code: "",
                  allow_retake: false,
                  exemption_quals: [],
                },
              ],
            }
          : stage,
      ),
    }))
  }

  const removeStage = (stageIndex: number) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.filter((_, index) => index !== stageIndex),
    }))
  }

  const removeUnit = (stageIndex: number, unitIndex: number) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.map((stage, currentStageIndex) =>
        currentStageIndex === stageIndex
          ? { ...stage, units: stage.units.filter((_, currentUnitIndex) => currentUnitIndex !== unitIndex) }
          : stage,
      ),
    }))
  }

  const copyTestStripeIds = () => {
    setForm((prev) => ({
      ...prev,
      unlock_stripe_product_id: "prod_UZFTQwZK5w3Yzh",
      unlock_stripe_price_id: "price_1Ta6y7CJWnR4MMONhDu5BsaZ",
      package_stripe_product_id: prev.package_stripe_product_id || "prod_UZILCqUwUoOPMO",
      package_stripe_price_id: prev.package_stripe_price_id || "price_1Ta9kYCJWnR4MMON0jjPUI8P",
    }))
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-foreground">{page.title}</h1>
              <p className="mt-1 text-muted-foreground">{page.subtitle}</p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={loadPipelines} disabled={loading}>
                <RefreshCw className="h-4 w-4" />
                {page.refresh}
              </Button>
              <Button onClick={() => { setSelectedId(""); setForm(emptyForm) }}>
                <Plus className="h-4 w-4" />
                {page.newPipeline}
              </Button>
            </div>
          </div>

          <div className="mb-4 grid gap-3 md:grid-cols-[minmax(220px,320px)_160px_1fr]">
            <div>
              <Label htmlFor="categoryFilter">{page.categoryTips}</Label>
              <Input id="categoryFilter" value={categoryFilter} onChange={(event) => { setCategoryFilter(event.target.value); setOffset(0) }} placeholder={page.categoryTipsPlaceholder} />
            </div>
            <label className="mt-6 flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
              <Checkbox checked={onlyCurrent} onCheckedChange={(checked) => { setOnlyCurrent(Boolean(checked)); setOffset(0) }} />
              {page.onlyCurrent}
            </label>
          </div>

          <div className="grid gap-4 xl:grid-cols-[420px_1fr]">
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <h2 className="font-semibold">{page.pipelineList}</h2>
                <Badge variant="outline">{pipelines.length}</Badge>
              </div>
              <div className="divide-y">
                {loading ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                ) : pipelines.length === 0 ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{page.noPipelines}</div>
                ) : (
                  pipelines.map((pipeline) => (
                    <button
                      key={pipeline.pipeline_id}
                      type="button"
                      onClick={() => selectPipeline(pipeline)}
                      className={`block w-full px-4 py-3 text-left transition ${selectedId === pipeline.pipeline_id ? "bg-muted" : "hover:bg-muted/60"}`}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="truncate font-medium">{pipeline.name || t.common.unknownCourse}</span>
                        <Badge variant={isDeprecated(pipeline) ? "secondary" : isPublished(pipeline) ? "default" : "outline"}>{isDeprecated(pipeline) ? page.statusDeprecated : isPublished(pipeline) ? page.active : page.draft}</Badge>
                      </div>
                      <div className="mt-1 truncate text-xs text-muted-foreground">{pipeline.pipeline_id}</div>
                      <div className="mt-1 flex justify-between text-xs text-muted-foreground">
                        <span>{pipeline.pipeline_guid || t.common.na}</span>
                        <span>{page.version} {pipeline.version || 0}</span>
                      </div>
                    </button>
                  ))
                )}
              </div>
              <div className="flex items-center justify-between border-t p-3">
                <Button variant="outline" size="sm" disabled={offset === 0 || loading} onClick={() => setOffset(Math.max(0, offset - limit))}>{page.prevPage}</Button>
                <span className="text-xs text-muted-foreground">{page.offset} {offset}</span>
                <Button variant="outline" size="sm" disabled={pipelines.length < limit || loading} onClick={() => setOffset(offset + limit)}>{page.nextPage}</Button>
              </div>
            </section>

            <section className="space-y-4">
              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <h2 className="font-semibold">{page.basicInfo}</h2>
                  {selectedPipeline && (
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" onClick={saveMetadata} disabled={saving}>
                        <Save className="h-4 w-4" />
                        {page.saveName}
                      </Button>
                      {!published && (
                        <Button variant="outline" size="sm" onClick={publishPipeline} disabled={saving}>
                          <Send className="h-4 w-4" />
                          {page.publish}
                        </Button>
                      )}
                      {published && selectedPipeline.status?.toLowerCase() !== "deprecated" && (
                        <Button variant="outline" size="sm" onClick={deprecatePipeline} disabled={saving}>
                          {page.deprecate}
                        </Button>
                      )}
                      {selectedPipeline.pipeline_guid && (
                        <Button variant="outline" size="sm" onClick={clonePipeline} disabled={creating}>
                          <Copy className="h-4 w-4" />
                          {page.cloneAsDraft}
                        </Button>
                      )}
                      {!published && (
                        <Button variant="destructive" size="sm" onClick={deletePipeline} disabled={saving}>
                          <Trash2 className="h-4 w-4" />
                          {page.delete}
                        </Button>
                      )}
                    </div>
                  )}
                </div>
                <div className="grid gap-4 p-4 md:grid-cols-2">
                  <div>
                    <Label htmlFor="name">{page.name}</Label>
                    <Input id="name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
                  </div>
                  <div>
                    <Label htmlFor="categoryTips">{page.categoryTips}</Label>
                    <Input id="categoryTips" value={form.category_tips} onChange={(event) => setForm({ ...form, category_tips: event.target.value })} disabled={Boolean(selectedPipeline)} />
                  </div>
                  <div className="md:col-span-2">
                    <Label htmlFor="respath">Respath</Label>
                    <Input id="respath" placeholder="e.g. /courses/cfta" value={form.respath} onChange={(event) => setForm({ ...form, respath: event.target.value })} disabled={Boolean(selectedPipeline)} />
                  </div>
                  {selectedPipeline && (
                    <div className="md:col-span-2 flex flex-wrap gap-4 text-xs text-muted-foreground">
                      <span>ID: {selectedPipeline.pipeline_id}</span>
                      <span>GUID: {selectedPipeline.pipeline_guid || t.common.na}</span>
                      <span>{page.version}: {selectedPipeline.version || 0}</span>
                      <span>{statusLabel(t, LMS_COURSE_STATUS_LABELS, selectedPipeline.status)}</span>
                    </div>
                  )}
                  <div className="md:col-span-2 rounded-md border bg-muted/30 p-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-semibold">{page.certificateAfterCompletion}</span>
                      <Badge variant={hasCertificate ? "default" : "outline"} className={hasCertificate ? "bg-emerald-600" : ""}>
                        {hasCertificate ? page.certificateYes : page.certificateNo}
                      </Badge>
                      <span className="text-xs text-muted-foreground">{page.certificateHint}</span>
                    </div>
                    {hasCertificate ? (
                      <div className="mt-2 flex flex-wrap gap-2">
                        {certificateNames.map((name, index) => (
                          <Badge key={`${name}-${index}`} variant="secondary">{name}</Badge>
                        ))}
                      </div>
                    ) : (
                      <p className="mt-2 text-xs text-muted-foreground">{page.noCertificateConfigured}</p>
                    )}
                  </div>
                </div>
                <div className="border-t px-4 py-3 text-right">
                  {!selectedPipeline && (
                    <Button onClick={createPipeline} disabled={creating}>
                      <Plus className="h-4 w-4" />
                      {page.createPipeline}
                    </Button>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <h2 className="font-semibold">{page.pipelineStripe}</h2>
                  <Button variant="outline" size="sm" onClick={copyTestStripeIds}>
                    <Copy className="h-4 w-4" />
                    {page.useTestStripe}
                  </Button>
                </div>
                <div className="grid gap-4 p-4 md:grid-cols-2">
                  <div>
                    <Label htmlFor="unlockProduct">{page.unlockProductId}</Label>
                    <Input id="unlockProduct" value={form.unlock_stripe_product_id} onChange={(event) => setForm({ ...form, unlock_stripe_product_id: event.target.value })} disabled={published} />
                  </div>
                  <div>
                    <Label htmlFor="unlockPrice">{page.unlockPriceId}</Label>
                    <Input id="unlockPrice" value={form.unlock_stripe_price_id} onChange={(event) => setForm({ ...form, unlock_stripe_price_id: event.target.value })} disabled={published} />
                  </div>
                  <div>
                    <Label htmlFor="packageProduct">{page.packageProductId}</Label>
                    <Input id="packageProduct" value={form.package_stripe_product_id} onChange={(event) => setForm({ ...form, package_stripe_product_id: event.target.value })} disabled={published} />
                  </div>
                  <div>
                    <Label htmlFor="packagePrice">{page.packagePriceId}</Label>
                    <Input id="packagePrice" value={form.package_stripe_price_id} onChange={(event) => setForm({ ...form, package_stripe_price_id: event.target.value })} disabled={published} />
                  </div>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{page.structure}</h2>
                    <p className="text-xs text-muted-foreground">{page.structureHint}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={addStage} disabled={!selectedPipeline || published}>
                    <Plus className="h-4 w-4" />
                    {page.addStage}
                  </Button>
                </div>
                <div className="space-y-4 p-4">
                  {!selectedPipeline ? (
                    <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{page.selectPipelineHint}</div>
                  ) : form.stages.length === 0 ? (
                    <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{page.noStages}</div>
                  ) : (
                    form.stages.map((stage, stageIndex) => (
                      <div key={stage.stage_id || stageIndex} className="rounded-lg border">
                        <div className="grid gap-3 border-b p-3 md:grid-cols-[1fr_120px_auto]">
                          <div>
                            <Label htmlFor={`stage-name-${stageIndex}`}>{page.stageName}</Label>
                            <Input id={`stage-name-${stageIndex}`} value={stage.name} onChange={(event) => updateStage(stageIndex, { name: event.target.value })} disabled={published} />
                          </div>
                          <div>
                            <Label htmlFor={`stage-order-${stageIndex}`}>{page.sortOrder}</Label>
                            <Input id={`stage-order-${stageIndex}`} type="number" value={stage.sort_order} onChange={(event) => updateStage(stageIndex, { sort_order: Number(event.target.value) })} disabled={published} />
                          </div>
                          <div className="mt-6 flex gap-2">
                            <Button variant="outline" size="sm" onClick={() => addUnit(stageIndex)} disabled={published}>
                              <Plus className="h-4 w-4" />
                              {page.addUnit}
                            </Button>
                            <Button variant="outline" size="icon-sm" onClick={() => removeStage(stageIndex)} disabled={published} aria-label={page.removeStage}>
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                        <div className="space-y-3 p-3">
                          {stage.units.length === 0 ? (
                            <div className="rounded-md bg-muted p-4 text-sm text-muted-foreground">{page.noUnits}</div>
                          ) : (
                            stage.units.map((unit, unitIndex) => (
                              <div key={unit.unit_id || unitIndex} className="rounded-md border p-3">
                                <div className="grid gap-3 lg:grid-cols-2 mb-3">
                                  <div>
                                    <Label>{page.unitName}</Label>
                                    <Input value={unit.name || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { name: event.target.value })} disabled={published} />
                                  </div>
                                </div>
                                <div className="grid gap-3 lg:grid-cols-[minmax(220px,1.2fr)_1fr_1fr_auto]">
                                  <div>
                                    <Label>{page.glmsCourse}</Label>
                                    <Select value={unit.glms_course_id || "none"} onValueChange={(value) => {
                                      const courseId = value === "none" ? "" : value
                                      const courseName = courseId ? (lmsCourses.find(c => c.course_id === courseId)?.title || courseId) : ""
                                      updateUnit(stageIndex, unitIndex, { 
                                        glms_course_id: courseId,
                                        name: unit.name ? unit.name : courseName
                                      })
                                    }} disabled={published}>
                                      <SelectTrigger className="w-full">
                                        <SelectValue placeholder={page.selectGlmsCourse} />
                                      </SelectTrigger>
                                      <SelectContent>
                                        <SelectItem value="none">{page.selectGlmsCourse}</SelectItem>
                                        {lmsCourses.map((course) => (
                                          <SelectItem key={course.course_id} value={course.course_id}>
                                            {course.title || course.course_id}
                                          </SelectItem>
                                        ))}
                                      </SelectContent>
                                    </Select>
                                    <div className="mt-1 truncate text-xs text-muted-foreground">{lmsCourseName(unit.glms_course_id)}</div>
                                  </div>
                                  <div>
                                    <Label>{page.unitProductId}</Label>
                                    <Input value={unit.stripe_product_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { stripe_product_id: event.target.value })} disabled={published} />
                                  </div>
                                  <div>
                                    <Label>{page.unitPriceId}</Label>
                                    <Input value={unit.stripe_price_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { stripe_price_id: event.target.value })} disabled={published} />
                                  </div>
                                  <div className="mt-6">
                                    <Button variant="outline" size="icon-sm" onClick={() => removeUnit(stageIndex, unitIndex)} disabled={published} aria-label={page.removeUnit}>
                                      <Trash2 className="h-4 w-4" />
                                    </Button>
                                  </div>
                                </div>
                                <div className="mt-3 grid gap-3 lg:grid-cols-2">
                                  <div className="grid gap-3 md:grid-cols-2">
                                    <Input placeholder={page.exemptionProductId} value={unit.exemption_stripe_product_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { exemption_stripe_product_id: event.target.value })} disabled={published} />
                                    <Input placeholder={page.exemptionPriceId} value={unit.exemption_stripe_price_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { exemption_stripe_price_id: event.target.value })} disabled={published} />
                                  </div>
                                  <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
                                    <Input placeholder={page.retakeProductId} value={unit.retake_stripe_product_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { retake_stripe_product_id: event.target.value })} disabled={published} />
                                    <Input placeholder={page.retakePriceId} value={unit.retake_stripe_price_id || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { retake_stripe_price_id: event.target.value })} disabled={published} />
                                    <label className="flex items-center gap-2 rounded-md border px-3 text-sm">
                                      <Checkbox checked={Boolean(unit.allow_retake)} onCheckedChange={(checked) => updateUnit(stageIndex, unitIndex, { allow_retake: Boolean(checked) })} disabled={published} />
                                      {page.allowRetake}
                                    </label>
                                  </div>
                                </div>
                                <div className="mt-3 rounded-md border bg-muted/20 p-3">
                                  <div className="mb-3">
                                    <div className="text-sm font-semibold">Exam</div>
                                    <p className="mt-1 text-xs text-muted-foreground">Program, Exam ID, and Form Code come from the pipeline unit config. Leave them empty if this unit has no formal exam.</p>
                                  </div>
                                  <div className="grid gap-3 md:grid-cols-3">
                                    <Input
                                      placeholder="Program"
                                      value={unit.program || ""}
                                      onChange={(event) => updateUnit(stageIndex, unitIndex, { program: event.target.value })}
                                      disabled={published}
                                    />
                                    <Input
                                      placeholder="Exam ID"
                                      value={unit.exam_id || ""}
                                      onChange={(event) => updateUnit(stageIndex, unitIndex, { exam_id: event.target.value })}
                                      disabled={published}
                                    />
                                    <Input
                                      placeholder="Form Code"
                                      value={unit.form_code || ""}
                                      onChange={(event) => updateUnit(stageIndex, unitIndex, { form_code: event.target.value })}
                                      disabled={published}
                                    />
                                  </div>
                                </div>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
                <div className="border-t px-4 py-3 text-right">
                  {published && <span className="mr-3 text-xs text-muted-foreground">{page.publishedLocked}</span>}
                  <Button onClick={saveStructure} disabled={!selectedPipeline || saving || published}>
                    <Save className="h-4 w-4" />
                    {page.saveStructure}
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center gap-2 border-b px-4 py-3">
                  <BookOpen className="h-4 w-4" />
                  <h2 className="font-semibold">{page.preview}</h2>
                </div>
                <div className="space-y-2 p-4">
                  <div className="rounded-md border bg-background p-3">
                    <div className="flex flex-wrap items-center gap-2 text-sm">
                      <span className="font-semibold">{page.certificateAfterCompletion}</span>
                      <Badge variant={hasCertificate ? "default" : "outline"} className={hasCertificate ? "bg-emerald-600" : ""}>
                        {hasCertificate ? page.certificateYes : page.certificateNo}
                      </Badge>
                      {hasCertificate && <span className="text-xs text-muted-foreground">{certificateNames.join(", ")}</span>}
                    </div>
                    {!hasCertificate && <p className="mt-1 text-xs text-muted-foreground">{page.noCertificateConfigured}</p>}
                  </div>
                  {form.stages.length === 0 ? (
                    <div className="text-sm text-muted-foreground">{page.noPreview}</div>
                  ) : (
                    form.stages.map((stage, index) => (
                      <div key={stage.stage_id || index} className="rounded-md bg-muted p-3">
                        <div className="flex items-center justify-between">
                          <span className="font-medium">{stage.name || page.unnamedStage}</span>
                          <span className="text-xs text-muted-foreground">{stage.units.length} {page.unitsCount}</span>
                        </div>
                        <div className="mt-2 space-y-3 text-sm text-muted-foreground">
                          {stage.units.map((unit, unitIndex) => {
                            const detail = lmsCourseDetails[unit.glms_course_id]
                            const hasExam = Boolean((unit.program || unit.exam_id || "").trim())
                            return (
                              <div key={unit.unit_id || unitIndex} className="rounded-md border bg-background p-3 shadow-sm">
                                <div className="flex items-center gap-2 mb-3">
                                  <CheckCircle2 className="h-4 w-4 text-primary" />
                                  <span className="font-semibold text-foreground">{lmsCourseName(unit.glms_course_id)}</span>
                                  {unit.stripe_price_id && <Badge variant="outline">{unit.stripe_price_id}</Badge>}
                                  {hasExam && <Badge variant="outline" className="border-red-200 bg-red-50 text-red-700">Exam</Badge>}
                                </div>
                                <div className="mb-3 grid gap-2 rounded-md bg-muted/40 p-3 text-xs md:grid-cols-3">
                                  <div>Exam: {hasExam ? "Yes" : "No"}</div>
                                  {hasExam && (
                                    <>
                                      <div>Program: {unit.program || t.common.na}</div>
                                      <div>Exam ID: {unit.exam_id || t.common.na}</div>
                                      <div>Form Code: {unit.form_code || t.common.na}</div>
                                    </>
                                  )}
                                </div>
                                {detail && (
                                  <div className="ml-6 space-y-4 border-l-2 pl-4">
                                    {detail.materials?.length > 0 && (
                                      <div>
                                        <div className="text-xs font-semibold mb-1 text-foreground">{page.materials}</div>
                                        <ul className="list-disc pl-4 text-xs space-y-0.5">
                                          {detail.materials.map((m: any, i: number) => (
                                            <li key={i}>{m.title || t.common.unknown}</li>
                                          ))}
                                        </ul>
                                      </div>
                                    )}
                                    {detail.chapters?.map((chapterDet: any, i: number) => (
                                      <div key={i} className="text-xs">
                                        <div className="font-semibold mb-1 text-foreground">
                                          {page.chapter}: {chapterDet.chapter?.title || t.common.unknown}
                                        </div>
                                        <div className="space-y-2 ml-1">
                                          {chapterDet.lessons?.map((lessonDet: any, li: number) => (
                                            <div key={li} className="pl-3 border-l">
                                              <div className="text-muted-foreground">{page.lesson}: {lessonDet.lesson?.title || t.common.unknown}</div>
                                              {lessonDet.quizzes?.length > 0 && (
                                                <div className="text-[10px] text-orange-600/80 mt-0.5 ml-2">
                                                  {page.quiz}: {lessonDet.quizzes.map((q: any) => q.quiz?.title || t.common.unknown).join(", ")}
                                                </div>
                                              )}
                                            </div>
                                          ))}
                                          {chapterDet.quizzes?.length > 0 && (
                                            <div className="pl-3 border-l text-orange-600 font-medium">
                                              {page.chapterQuiz}: {chapterDet.quizzes.map((q: any) => q.quiz?.title || t.common.unknown).join(", ")}
                                            </div>
                                          )}
                                        </div>
                                      </div>
                                    ))}
                                    {detail.quizzes?.length > 0 && (
                                      <div>
                                        <div className="text-xs font-semibold mb-1 text-red-600">{page.finalExam}</div>
                                        <ul className="list-disc pl-4 text-xs text-red-600/80 space-y-0.5">
                                          {detail.quizzes.map((q: any, i: number) => (
                                            <li key={i}>{q.quiz?.title || t.common.unknown}</li>
                                          ))}
                                        </ul>
                                      </div>
                                    )}
                                  </div>
                                )}
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    ))
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
