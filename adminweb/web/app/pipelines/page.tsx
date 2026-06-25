"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { Copy, Package, Plus, RefreshCw, Save, Send, Trash2 } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ADMIN_PIPELINE_STATUS_LABELS, LMS_COURSE_STATUS_LABELS, statusLabel } from "@/lib/status-labels"

type Pipeline = {
  pipeline_id: string
  pipeline_ulid?: string
  pipeline_guid: string
  version: number
  name: string
  category_tips?: string
  status: string
  is_current: boolean
  created_at: string
  unlock_quals?: Qualification[]
  respath?: string
  certs?: Qualification[]
  certs_quals?: Qualification[]
  cert_quals?: Qualification[]
  final_quals?: Qualification[]
  stages?: StageConfig[]
}

type Qualification = {
  qual_ulid?: string
  qual_id?: string
  name_hint?: string
  name?: string
  title?: string
  pdf_template_ulid?: string
  pdf_template_id?: string
}

type CredentialDefinition = {
  cred_def_id: string
  cred_def_ulid?: string
  name?: string
  category?: string
}

type PdfTemplate = {
  template_id: string
  template_ulid?: string
  pdf_template_ulid?: string
  name?: string
  description?: string
}

type StageConfig = {
  stage_ulid?: string
  stage_id?: string
  name: string
  sort_order: number
  units: UnitConfig[]
}

type UnitConfig = {
  unit_ulid?: string
  unit_id?: string
  name?: string
  glms_course_ulid?: string
  glms_course_id: string
  program?: string
  exam_ulid?: string
  exam_id?: string
  form_code?: string
  allow_retake?: boolean
  allow_exemption?: boolean
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
  unlock_quals: Qualification[]
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
  unlock_quals: [],
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

function firstString(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim()
  }
  return ""
}

function qualificationUlidOf(qualification: Partial<Qualification> | null | undefined) {
  return firstString(qualification?.qual_ulid, qualification?.qual_id)
}

function credentialDefinitionUlidOf(definition: Partial<CredentialDefinition> | null | undefined) {
  return firstString(definition?.cred_def_ulid, definition?.cred_def_id)
}

function pdfTemplateUlidOf(template: Partial<PdfTemplate> | null | undefined) {
  return firstString(template?.pdf_template_ulid, template?.template_ulid, template?.template_id)
}

function qualificationPdfTemplateUlidOf(qualification: Partial<Qualification> | null | undefined) {
  return firstString(qualification?.pdf_template_ulid, qualification?.pdf_template_id)
}

function stageUlidOf(stage: Partial<StageConfig> | null | undefined) {
  return firstString(stage?.stage_ulid, stage?.stage_id)
}

function unitUlidOf(unit: Partial<UnitConfig> | null | undefined) {
  return firstString(unit?.unit_ulid, unit?.unit_id)
}

function unitCourseUlidOf(unit: Partial<UnitConfig> | null | undefined) {
  return firstString(unit?.glms_course_ulid, unit?.glms_course_id)
}

function unitExamUlidOf(unit: Partial<UnitConfig> | null | undefined) {
  return firstString(unit?.exam_ulid, unit?.exam_id)
}

function pipelineToForm(pipeline: Pipeline | null): PipelineForm {
  if (!pipeline) return emptyForm
  return {
    name: pipeline.name || "",
    category_tips: pipeline.category_tips || "",
    unlock_quals: normalizeQualifications(pipeline.unlock_quals),
    respath: pipeline.respath || "",
    certs: normalizeQualifications(pipeline.certs),
    certs_quals: normalizeQualifications(pipeline.certs_quals),
    cert_quals: normalizeQualifications(pipeline.cert_quals),
    final_quals: normalizeQualifications(pipeline.final_quals),
    stages: (pipeline.stages || []).map((stage) => ({
      stage_ulid: stageUlidOf(stage),
      stage_id: stageUlidOf(stage),
      name: stage.name || "",
      sort_order: Number(stage.sort_order || 0),
      units: (stage.units || []).map((unit) => ({
        unit_ulid: unitUlidOf(unit),
        unit_id: unitUlidOf(unit),
        name: unit.name || "",
        glms_course_ulid: unitCourseUlidOf(unit),
        glms_course_id: unitCourseUlidOf(unit),
        program: unit.program || "",
        exam_ulid: unitExamUlidOf(unit),
        exam_id: unitExamUlidOf(unit),
        form_code: unit.form_code || "",
        allow_retake: Boolean(unit.allow_retake),
        allow_exemption: Boolean(unit.allow_exemption),
        exemption_quals: unit.exemption_quals || [],
      })),
    })),
  }
}

function cleanFormForStructure(form: PipelineForm) {
  const cleanQualifications = (items: Qualification[]) => normalizeQualifications(items).map((item) => ({
    qual_ulid: qualificationUlidOf(item),
    name_hint: (item.name_hint || item.name || item.title || "").trim(),
    pdf_template_ulid: qualificationPdfTemplateUlidOf(item),
  }))

  return {
    unlock_quals: cleanQualifications(form.unlock_quals),
    certs: cleanQualifications(form.certs),
    certs_quals: cleanQualifications(form.certs_quals),
    stages: form.stages.map((stage) => ({
      stage_ulid: stageUlidOf(stage),
      name: stage.name.trim(),
      sort_order: Number(stage.sort_order || 0),
      units: stage.units.map((unit) => ({
        unit_ulid: unitUlidOf(unit),
        name: (unit.name || "").trim(),
        glms_course_ulid: unitCourseUlidOf(unit),
        program: (unit.program || "").trim(),
        exam_ulid: unitExamUlidOf(unit),
        form_code: (unit.form_code || "").trim(),
        allow_retake: Boolean(unit.allow_retake),
        allow_exemption: Boolean(unit.allow_exemption),
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

function pipelineIdOf(pipeline: Partial<Pipeline> | null | undefined) {
  return pipeline?.pipeline_id || pipeline?.pipeline_ulid || ""
}

function qualificationLabel(qualification: Qualification, fallback: string) {
  return qualification.name_hint || qualification.name || qualification.title || qualificationUlidOf(qualification) || fallback
}

function qualificationKey(qualification: Qualification) {
  return qualificationUlidOf(qualification) || qualification.name_hint || qualification.name || qualification.title || ""
}

function pipelineCertificateItems(form: PipelineForm) {
  return normalizeQualifications(form.certs).filter((item, index, list) => {
    const key = qualificationKey(item) || String(index)
    return list.findIndex((candidate) => qualificationKey(candidate) === key) === index
  })
}

function credentialToQualification(definition: CredentialDefinition): Qualification {
  const credDefUlid = credentialDefinitionUlidOf(definition)
  return {
    qual_ulid: credDefUlid,
    qual_id: credDefUlid,
    name_hint: definition.name || credDefUlid,
  }
}

function hasQualification(list: Qualification[], qualId: string) {
  return normalizeQualifications(list).some((item) => qualificationUlidOf(item) === qualId)
}

function toggleQualification(list: Qualification[], definition: CredentialDefinition, checked: boolean) {
  const current = normalizeQualifications(list)
  const definitionId = credentialDefinitionUlidOf(definition)
  if (!definitionId) return current
  if (checked) {
    if (hasQualification(current, definitionId)) return current
    return [...current, credentialToQualification(definition)]
  }
  return current.filter((item) => qualificationUlidOf(item) !== definitionId)
}

export default function PipelinesPage() {
  const { t } = useTranslation()
  const page = t.pipelinesPage
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [lmsCourses, setLmsCourses] = useState<LmsCourse[]>([])
  const [credentialDefinitions, setCredentialDefinitions] = useState<CredentialDefinition[]>([])
  const [pdfTemplates, setPdfTemplates] = useState<PdfTemplate[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [form, setForm] = useState<PipelineForm>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("")
  const [onlyCurrent, setOnlyCurrent] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [offset, setOffset] = useState(0)
  const limit = 20

  const selectedPipeline = useMemo(
    () => pipelines.find((pipeline) => pipelineIdOf(pipeline) === selectedId) || null,
    [pipelines, selectedId],
  )
  const published = isPublished(selectedPipeline)
  const certificateItems = useMemo(() => pipelineCertificateItems(form), [form])
  const hasCertificate = certificateItems.length > 0
  const certificateNames = useMemo(
    () => certificateItems.map((item, index) => qualificationLabel(item, `${page.certificate} ${index + 1}`)),
    [certificateItems, page.certificate],
  )
  const pdfTemplateById = useMemo(() => {
    const map = new Map<string, PdfTemplate>()
    pdfTemplates.forEach((template) => {
      const id = pdfTemplateUlidOf(template)
      if (id) map.set(id, template)
    })
    return map
  }, [pdfTemplates])
  const credentialDefinitionById = useMemo(() => {
    const map = new Map<string, CredentialDefinition>()
    credentialDefinitions.forEach((definition) => {
      const id = credentialDefinitionUlidOf(definition)
      if (id) map.set(id, definition)
    })
    return map
  }, [credentialDefinitions])

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
      if (selectedId && !nextPipelines.some((pipeline: Pipeline) => pipelineIdOf(pipeline) === selectedId)) {
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

  const loadCredentialDefinitions = useCallback(async () => {
    const res = await apiClient("/api/credentials/definitions")
    setCredentialDefinitions(res?.definitions || [])
  }, [])

  const loadPdfTemplates = useCallback(async () => {
    const res = await apiClient("/api/pdf-templates")
    setPdfTemplates(res?.templates || [])
  }, [])

  useEffect(() => {
    loadPipelines()
  }, [loadPipelines])

  useEffect(() => {
    loadLmsCourses().catch(() => toast.error(page.loadLmsFailed))
  }, [loadLmsCourses, page.loadLmsFailed])

  useEffect(() => {
    loadCredentialDefinitions().catch(() => toast.error(page.loadCredentialsFailed))
  }, [loadCredentialDefinitions, page.loadCredentialsFailed])

  useEffect(() => {
    loadPdfTemplates().catch(() => toast.error(page.loadPdfTemplatesFailed))
  }, [loadPdfTemplates, page.loadPdfTemplatesFailed])

  const selectPipeline = async (pipeline: Pipeline) => {
    const pipelineId = pipelineIdOf(pipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    setSelectedId(pipelineId)
    try {
      const detail = await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}`)
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
      const newPipelineId = pipelineIdOf(res)
      setSelectedId(newPipelineId)
      try {
        if (!newPipelineId) throw new Error("missing pipeline id")
        const detail = await apiClient(`/api/pipelines/${encodeURIComponent(newPipelineId)}`)
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
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    if (!form.name.trim()) {
      toast.error(page.fillName)
      return
    }
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}/metadata`, {
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
        if (!unitCourseUlidOf(unit)) {
          toast.error(page.unitCourseRequired.replace("{{stage}}", stage.name || String(stageIndex + 1)).replace("{{index}}", String(unitIndex + 1)))
          return false
        }
      }
    }
    return true
  }

  const validateFinalCertificateTemplates = () => {
    const missing = normalizeQualifications(form.certs).find((item) => !qualificationPdfTemplateUlidOf(item))
    if (!missing) return true
    toast.error(page.certTemplateRequired.replace("{{name}}", qualificationLabel(missing, page.certificate)))
    return false
  }

  const saveStructure = async () => {
    if (!selectedPipeline) return
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    if (published) {
      toast.error(page.publishedLocked)
      return
    }
    if (!validateStructure()) return
    if (!validateFinalCertificateTemplates()) return
    setSaving(true)
    try {
      const res = await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}/structure`, {
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
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    if (!validateStructure()) return
    if (!validateFinalCertificateTemplates()) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}/publish`, {
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
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    if (!window.confirm("Confirm deprecate?")) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}/deprecate`, {
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
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
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
          name: form.name.trim() + " (Copy)",
          category_tips: form.category_tips.trim(),
          respath: form.respath.trim(),
          from_pipeline_guid: selectedPipeline.pipeline_guid,
          from_pipeline_id: pipelineId,
        }),
      })
      const newPipelineId = pipelineIdOf(res)
      toast.success(page.createSuccess)
      setSelectedId(newPipelineId)
      try {
        if (!newPipelineId) throw new Error("missing pipeline id")
        const detail = await apiClient(`/api/pipelines/${encodeURIComponent(newPipelineId)}`)
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
    const pipelineId = pipelineIdOf(selectedPipeline)
    if (!pipelineId) {
      toast.error(t.common.error)
      return
    }
    if (published) {
      toast.error(page.deletePublishedBlocked)
      return
    }
    if (!window.confirm(page.confirmDelete)) return
    setSaving(true)
    try {
      await apiClient(`/api/pipelines/${encodeURIComponent(pipelineId)}`, { method: "DELETE" })
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
                  glms_course_ulid: "",
                  glms_course_id: "",
                  program: "",
                  exam_ulid: "",
                  exam_id: "",
                  form_code: "",
                  allow_retake: false,
                  allow_exemption: false,
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

  const updateQualificationField = (
    field: "unlock_quals" | "certs_quals" | "certs",
    definition: CredentialDefinition,
    checked: boolean,
  ) => {
    setForm((prev) => ({
      ...prev,
      [field]: toggleQualification(prev[field], definition, checked),
    }))
  }

  const updateQualificationTemplate = (field: "certs", qualId: string, templateId: string) => {
    const template = pdfTemplateById.get(templateId)
    setForm((prev) => ({
      ...prev,
      [field]: normalizeQualifications(prev[field]).map((item) =>
        qualificationUlidOf(item) === qualId
          ? {
              ...item,
              pdf_template_ulid: templateId,
              pdf_template_id: templateId,
              name_hint: item.name_hint || item.name || template?.name || qualId,
            }
          : item,
      ),
    }))
  }

  const toggleUnitExemptionQualification = (stageIndex: number, unitIndex: number, definitionId: string, checked: boolean) => {
    setForm((prev) => ({
      ...prev,
      stages: prev.stages.map((stage, currentStageIndex) =>
        currentStageIndex === stageIndex
          ? {
              ...stage,
              units: stage.units.map((unit, currentUnitIndex) => {
                if (currentUnitIndex !== unitIndex) return unit
                const current = unit.exemption_quals || []
                const next = checked
                  ? Array.from(new Set([...current, definitionId]))
                  : current.filter((item) => item !== definitionId)
                return {
                  ...unit,
                  allow_exemption: next.length > 0 ? true : unit.allow_exemption,
                  exemption_quals: next,
                }
              }),
            }
          : stage,
      ),
    }))
  }

  const exemptionQualificationLabel = (qualId: string) => {
    const definition = credentialDefinitionById.get(qualId)
    return definition?.name || qualId
  }

  const renderQualificationSelector = (
    field: "unlock_quals" | "certs_quals" | "certs",
    label: string,
    description: string,
  ) => {
    const selected = normalizeQualifications(form[field])
    return (
      <div className="rounded-md border bg-muted/20 p-3">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <Label>{label}</Label>
            <p className="mt-1 text-xs text-muted-foreground">{description}</p>
          </div>
          <Badge variant="outline">{selected.length}</Badge>
        </div>
        {credentialDefinitions.length === 0 ? (
          <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">{page.noCredentialDefinitions}</div>
        ) : (
          <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
            {credentialDefinitions.map((definition) => {
              const definitionId = credentialDefinitionUlidOf(definition)
              if (!definitionId) return null
              const checked = hasQualification(selected, definitionId)
              const selectedQualification = selected.find((item) => qualificationUlidOf(item) === definitionId)
              return (
                <div key={`${field}-${definitionId}`} className="flex min-h-12 items-start gap-2 rounded-md border bg-background p-3 text-sm">
                  <Checkbox
                    checked={checked}
                    onCheckedChange={(value) => updateQualificationField(field, definition, Boolean(value))}
                    disabled={published}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">{definition.name || definitionId}</span>
                    <span className="block truncate text-xs text-muted-foreground">{definition.category || definitionId}</span>
                    {field === "certs" && checked && (
                      <span className="mt-2 block">
                        {pdfTemplates.length === 0 ? (
                          <span className="block rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">{page.noPdfTemplates}</span>
                        ) : (
                          <Select
                            value={qualificationPdfTemplateUlidOf(selectedQualification) || undefined}
                            onValueChange={(value) => updateQualificationTemplate("certs", definitionId, value)}
                            disabled={published}
                          >
                            <SelectTrigger className="h-8 text-xs">
                              <SelectValue placeholder={page.selectPdfTemplate} />
                            </SelectTrigger>
                            <SelectContent>
                              {pdfTemplates.map((template) => {
                                const templateId = pdfTemplateUlidOf(template)
                                if (!templateId) return null
                                return (
                                  <SelectItem key={templateId} value={templateId}>
                                    {template.name || templateId}
                                  </SelectItem>
                                )
                              })}
                            </SelectContent>
                          </Select>
                        )}
                      </span>
                    )}
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    )
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
                  pipelines.map((pipeline, index) => {
                    const pipelineId = pipelineIdOf(pipeline)
                    return (
                      <button
                        key={pipelineId || `pipeline-${index}`}
                        type="button"
                        onClick={() => selectPipeline(pipeline)}
                        className={`block w-full px-4 py-3 text-left transition ${selectedId === pipelineId ? "bg-muted" : "hover:bg-muted/60"}`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <span className="truncate font-medium">{pipeline.name || t.common.unknownCourse}</span>
                          <Badge variant={isDeprecated(pipeline) ? "secondary" : isPublished(pipeline) ? "default" : "outline"}>{isDeprecated(pipeline) ? page.statusDeprecated : isPublished(pipeline) ? page.active : page.draft}</Badge>
                        </div>
                        <div className="mt-1 truncate text-xs text-muted-foreground">{pipelineId || t.common.na}</div>
                        <div className="mt-1 flex justify-between text-xs text-muted-foreground">
                          <span>{pipeline.pipeline_guid || t.common.na}</span>
                          <span>{page.version} {pipeline.version || 0}</span>
                        </div>
                      </button>
                    )
                  })
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
                      <span>ID: {pipelineIdOf(selectedPipeline) || t.common.na}</span>
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

              <div className="rounded-lg border border-blue-200 bg-blue-50/60 p-4 text-sm text-blue-900">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h2 className="font-semibold">{page.bundlePricingMovedTitle}</h2>
                    <p className="mt-1 text-blue-800/80">{page.bundlePricingMovedHint}</p>
                  </div>
                  <Button asChild variant="outline" size="sm" className="border-blue-200 bg-white/80 text-blue-900 hover:bg-blue-100">
                    <a href="/bundles">
                      <Package className="h-4 w-4" />
                      {page.openBundleConfig}
                    </a>
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{page.qualificationConfig}</h2>
                    <p className="text-xs text-muted-foreground">{page.qualificationConfigHint}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={saveStructure} disabled={!selectedPipeline || saving || published}>
                    <Save className="h-4 w-4" />
                    {page.saveQualifications}
                  </Button>
                </div>
                <div className="space-y-3 p-4">
                  {renderQualificationSelector("unlock_quals", page.unlockQuals, page.unlockQualsHint)}
                  {renderQualificationSelector("certs_quals", page.certsQuals, page.certsQualsHint)}
                  {renderQualificationSelector("certs", page.certs, page.certsHint)}
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
                      <div key={stageUlidOf(stage) || stageIndex} className="rounded-lg border">
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
                            stage.units.map((unit, unitIndex) => {
                              const courseId = unitCourseUlidOf(unit)
                              const examId = unitExamUlidOf(unit)
                              const exemptionQuals = unit.exemption_quals || []
                              return (
                              <div key={unitUlidOf(unit) || unitIndex} className="rounded-md border p-3">
                                <div className="grid gap-3 lg:grid-cols-2 mb-3">
                                  <div>
                                    <Label>{page.unitName}</Label>
                                    <Input value={unit.name || ""} onChange={(event) => updateUnit(stageIndex, unitIndex, { name: event.target.value })} disabled={published} />
                                  </div>
                                </div>
                                <div className="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_auto]">
                                  <div>
                                    <Label>{page.glmsCourse}</Label>
                                    <Select value={courseId || "none"} onValueChange={(value) => {
                                      const nextCourseId = value === "none" ? "" : value
                                      const courseName = nextCourseId ? (lmsCourses.find(c => c.course_id === nextCourseId)?.title || nextCourseId) : ""
                                      updateUnit(stageIndex, unitIndex, { 
                                        glms_course_ulid: nextCourseId,
                                        glms_course_id: nextCourseId,
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
                                    <div className="mt-1 truncate text-xs text-muted-foreground">{lmsCourseName(courseId)}</div>
                                  </div>
                                  <div className="mt-6">
                                    <Button variant="outline" size="icon-sm" onClick={() => removeUnit(stageIndex, unitIndex)} disabled={published} aria-label={page.removeUnit}>
                                      <Trash2 className="h-4 w-4" />
                                    </Button>
                                  </div>
                                </div>
                                <div className="mt-3">
                                  <label className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
                                    <Checkbox checked={Boolean(unit.allow_retake)} onCheckedChange={(checked) => updateUnit(stageIndex, unitIndex, { allow_retake: Boolean(checked) })} disabled={published} />
                                    {page.allowRetake}
                                  </label>
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
                                      value={examId}
                                      onChange={(event) => updateUnit(stageIndex, unitIndex, { exam_ulid: event.target.value, exam_id: event.target.value })}
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
                                <div className="mt-3 rounded-md border bg-muted/20 p-3">
                                  <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
                                    <div>
                                      <div className="text-sm font-semibold">{page.exemptionConfig}</div>
                                      <p className="mt-1 text-xs text-muted-foreground">{page.exemptionConfigHint}</p>
                                    </div>
                                    <label className="inline-flex items-center gap-2 rounded-md border bg-background px-3 py-2 text-sm">
                                      <Checkbox
                                        checked={Boolean(unit.allow_exemption)}
                                        onCheckedChange={(checked) =>
                                          updateUnit(stageIndex, unitIndex, {
                                            allow_exemption: Boolean(checked),
                                            exemption_quals: checked ? exemptionQuals : [],
                                          })
                                        }
                                        disabled={published}
                                      />
                                      {page.allowExemption}
                                    </label>
                                  </div>
                                  {unit.allow_exemption ? (
                                    credentialDefinitions.length === 0 ? (
                                      <div className="rounded-md border border-dashed p-3 text-xs text-muted-foreground">{page.noCredentialDefinitions}</div>
                                    ) : (
                                      <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                                        {credentialDefinitions.map((definition) => {
                                          const definitionId = credentialDefinitionUlidOf(definition)
                                          if (!definitionId) return null
                                          return (
                                            <label key={`${unitUlidOf(unit) || unitIndex}-exemption-${definitionId}`} className="flex items-start gap-2 rounded-md border bg-background p-3 text-sm">
                                              <Checkbox
                                                checked={exemptionQuals.includes(definitionId)}
                                                onCheckedChange={(checked) => toggleUnitExemptionQualification(stageIndex, unitIndex, definitionId, Boolean(checked))}
                                                disabled={published}
                                              />
                                              <span className="min-w-0">
                                                <span className="block truncate font-medium">{definition.name || definitionId}</span>
                                                <span className="block truncate text-xs text-muted-foreground">{definition.category || definitionId}</span>
                                              </span>
                                            </label>
                                          )
                                        })}
                                      </div>
                                    )
                                  ) : (
                                    <div className="rounded-md border border-dashed p-3 text-xs text-muted-foreground">{page.noExemption}</div>
                                  )}
                                  {exemptionQuals.length > 0 && (
                                    <div className="mt-3 flex flex-wrap gap-2">
                                      {exemptionQuals.map((qualId) => (
                                        <Badge key={qualId} variant="secondary">{exemptionQualificationLabel(qualId)}</Badge>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              </div>
                              )
                            })
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

            </section>
          </div>
        </div>
      </main>
    </div>
  )
}
