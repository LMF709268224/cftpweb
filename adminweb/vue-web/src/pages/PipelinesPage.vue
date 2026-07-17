<script setup lang="ts">
import { Award, BookOpen, ChevronDown, Copy, GraduationCap, Inbox, Info, Layers, Loader2, Plus, RefreshCw, Save, Send, ShieldCheck, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type PipelineForm = {
  name: string
  pipeline_gpath: string
  category_tips: string
  structure_json: string
}

type LayerKey = "overview" | "stages" | "units" | "certs" | "unlock_quals" | "certs_quals" | "raw"

type UnitListItem = {
  stageIndex: number
  unitIndex: number
  path: string
  stage: JsonRecord
  unit: JsonRecord
}

const emptyStructure = () => ({
  unlock_quals: [],
  certs: [],
  certs_quals: [],
  stages: [],
})

const emptyForm: PipelineForm = {
  name: "",
  pipeline_gpath: "",
  category_tips: "default",
  structure_json: JSON.stringify(emptyStructure(), null, 2),
}

const pipelines = ref<JsonRecord[]>([])
const courseOptions = ref<JsonRecord[]>([])
const credentialOptions = ref<JsonRecord[]>([])
const pdfTemplateOptions = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<PipelineForm>({ ...emptyForm })
const structure = ref<JsonRecord>(emptyStructure())
const loading = ref(false)
const courseOptionsLoading = ref(false)
const credentialOptionsLoading = ref(false)
const pdfTemplateOptionsLoading = ref(false)
const saving = ref(false)
const creating = ref(false)
const categoryFilter = ref("")
const onlyCurrent = ref(false)
const pageToken = ref("")
const nextToken = ref("")
const prevToken = ref("")
const hasMore = ref(false)
const activeLayer = ref<LayerKey>("overview")
const selectedStageIndex = ref(0)
const selectedUnitPath = ref("")
const selectedCertIndex = ref(0)
const selectedUnlockQualIndex = ref(0)
const selectedCertQualIndex = ref(0)
const deprecateConfirmOpen = ref(false)
const deprecating = ref(false)
const deleteConfirmOpen = ref(false)
const pendingDeletePipeline = ref<JsonRecord | null>(null)
const deletingPipeline = ref(false)
const limit = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.pipelineConfigAdmin)

const canPrev = computed(() => !!prevToken.value)
const canNext = computed(() => hasMore.value)
const inEditor = computed(() => !!selected.value || creating.value)
const selectedId = computed(() => selected.value ? pipelineUlid(selected.value) : "")
const published = computed(() => {
  const status = String(selected.value?.status || "").toLowerCase()
  return status === "active" || (selected.value?.is_current && status !== "deprecated")
})
const canDeleteSelectedPipeline = computed(() => !!selectedId.value && pipelineStatusKey(selected.value) !== "DEPRECATED" && !published.value)
const structureLocked = computed(() => creating.value || published.value || !selectedId.value)

const stages = computed(() => asArray(structure.value.stages))
const certs = computed(() => asArray(structure.value.certs))
const unlockQuals = computed(() => asArray(structure.value.unlock_quals))
const certQuals = computed(() => asArray(structure.value.certs_quals))
const units = computed<UnitListItem[]>(() => {
  const list: UnitListItem[] = []
  stages.value.forEach((stage, stageIndex) => {
    asArray(stage.units).forEach((unit, unitIndex) => {
      list.push({
        stageIndex,
        unitIndex,
        path: `${stageIndex}:${unitIndex}`,
        stage,
        unit,
      })
    })
  })
  return list
})

const selectedStage = computed(() => stages.value[selectedStageIndex.value] || null)
const selectedStageUnits = computed(() => selectedStage.value ? asArray(selectedStage.value.units) : [])
const selectedUnitItem = computed(() => units.value.find((item) => item.path === selectedUnitPath.value) || units.value[0] || null)
const selectedCert = computed(() => certs.value[selectedCertIndex.value] || null)
const selectedUnlockQual = computed(() => unlockQuals.value[selectedUnlockQualIndex.value] || null)
const selectedCertQual = computed(() => certQuals.value[selectedCertQualIndex.value] || null)

const layerItems = computed(() => [
  { key: "overview" as const, title: copy.value.layers.overview.title, desc: copy.value.layers.overview.desc, count: selected.value ? 1 : 0 },
  { key: "stages" as const, title: copy.value.layers.stages.title, desc: copy.value.layers.stages.desc, count: stages.value.length },
  { key: "units" as const, title: copy.value.layers.units.title, desc: copy.value.layers.units.desc, count: units.value.length },
  { key: "certs" as const, title: copy.value.layers.certs.title, desc: copy.value.layers.certs.desc, count: certs.value.length },
  { key: "unlock_quals" as const, title: copy.value.layers.unlockQuals.title, desc: copy.value.layers.unlockQuals.desc, count: unlockQuals.value.length },
  { key: "certs_quals" as const, title: copy.value.layers.certQuals.title, desc: copy.value.layers.certQuals.desc, count: certQuals.value.length },
])

function asArray(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

function isStructureLocked() {
  return Boolean(structureLocked.value)
}

function asMutableArray(parent: JsonRecord, key: string) {
  if (!Array.isArray(parent[key])) parent[key] = []
  return parent[key] as JsonRecord[]
}

function pipelineUlid(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["pipeline_ulid", "pipeline_id"]) || "")
}

function pipelineName(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["name", "title"]) || copy.value.unnamedPipeline)
}

function pipelineStatus(pipeline: JsonRecord) {
  return pickFirst(pipeline, ["status", "raw_status"])
}

function pipelineStatusKey(pipeline: JsonRecord | null | undefined) {
  return String(pipeline ? pipelineStatus(pipeline) : "").trim().toUpperCase().replace(/^PIPELINE_STATUS_/, "")
}

function pipelineStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  const normalized = raw.toUpperCase().replace(/^PIPELINE_STATUS_/, "")
  const labels: Record<string, string> = {
    ACTIVE: copy.value.status.active,
    PUBLISHED: copy.value.status.published,
    DRAFT: copy.value.status.draft,
    DEPRECATED: copy.value.status.deprecated,
    INACTIVE: copy.value.status.inactive,
    ARCHIVED: copy.value.status.archived,
    PENDING: copy.value.status.pending,
    PENDING_CREATE: copy.value.status.pendingCreate,
    COMPLETED: copy.value.status.completed,
    CANCELLED: copy.value.status.cancelled,
    FAILED: copy.value.status.failed,
  }
  return labels[normalized] || raw || "-"
}

function itemTitle(item: JsonRecord | null | undefined, fallback: string) {
  if (!item) return fallback
  return String(pickFirst(item, ["name", "title", "unit_name", "stage_name", "name_hint", "qual_name", "qual_ulid", "unit_ulid", "stage_ulid"]) || fallback)
}

function itemId(item: JsonRecord | null | undefined, keys: string[]) {
  if (!item) return ""
  return String(pickFirst(item, keys) || "")
}

function courseId(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["course_ulid", "course_id"]) || "")
}

function courseTitle(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["title", "name", "course_title"]) || courseId(course) || copy.value.fields.glmsCourse)
}

function courseStatusValue(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["status", "raw_status"]) || "").trim()
}

function courseCanBeConfigured(course: JsonRecord) {
  const normalized = courseStatusValue(course).toUpperCase()
  return normalized === "ACTIVE" || normalized === "PUBLISHED" || course.is_published === true
}

function courseOptionLabel(course: JsonRecord) {
  const id = courseId(course)
  const status = courseStatusValue(course)
  const statusText = status ? pipelineStatusLabel(status) : ""
  const version = course.version ? `v${course.version}` : ""
  return [courseTitle(course), version, statusText, id].filter(Boolean).join(" · ")
}

function courseById(id: string) {
  return courseOptions.value.find((course) => courseId(course) === id) || null
}

function unitCourseId(unit: JsonRecord | null | undefined) {
  return String(pickFirst(unit || {}, ["glms_course_ulid", "glms_course_id"]) || "")
}

function unitCourseLabel(unit: JsonRecord | null | undefined) {
  const id = unitCourseId(unit)
  if (!id) return copy.value.stageUnitSummaryCourseMissing
  const course = courseById(id)
  return course ? courseTitle(course) : id
}

function credentialId(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["cred_def_ulid", "cred_def_id", "qual_ulid", "qual_id", "id"]) || "")
}

function credentialName(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["name", "name_hint", "title"]) || credentialId(definition) || copy.value.unnamedQualification)
}

function credentialOptionLabel(definition: JsonRecord) {
  const id = credentialId(definition)
  const category = String(pickFirst(definition, ["category", "type"]) || "")
  return [credentialName(definition), category, id].filter(Boolean).join(" · ")
}

function credentialById(id: string) {
  return credentialOptions.value.find((definition) => credentialId(definition) === id) || null
}

function qualificationId(qualification: JsonRecord | null | undefined) {
  return String(pickFirst(qualification || {}, ["qual_ulid", "qual_id"]) || "")
}

function pdfTemplateId(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["template_ulid", "template_id", "id"]) || "")
}

function pdfTemplateName(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["name", "title"]) || pdfTemplateId(template) || copy.value.unnamedPdfTemplate)
}

function pdfTemplateOptionLabel(template: JsonRecord) {
  const id = pdfTemplateId(template)
  const version = template.version ? `v${template.version}` : ""
  return [pdfTemplateName(template), version, id].filter(Boolean).join(" · ")
}

function pdfTemplateById(id: string) {
  return pdfTemplateOptions.value.find((template) => pdfTemplateId(template) === id) || null
}

function normalizeStructureShape(value: JsonRecord | null | undefined) {
  const next = value || {}
  return {
    unlock_quals: asArray(next.unlock_quals).map(normalizeQualificationShape),
    certs: asArray(next.certs).map(normalizeQualificationShape),
    certs_quals: asArray(next.certs_quals).map(normalizeQualificationShape),
    stages: asArray(next.stages).map((stage) => ({
      ...stage,
      stage_ulid: String(pickFirst(stage, ["stage_ulid", "stage_id"]) || ""),
      units: asArray(stage.units).map(normalizeUnitShape),
    })),
  }
}

function normalizeQualificationShape(qual: JsonRecord) {
  const next = { ...qual }
  next.qual_ulid = String(pickFirst(next, ["qual_ulid", "qual_id"]) || "")
  next.pdf_template_ulid = String(pickFirst(next, ["pdf_template_ulid", "pdf_template_id"]) || "")
  delete next.qual_id
  delete next.pdf_template_id
  return next
}

function normalizeUnitShape(unit: JsonRecord) {
  const next = { ...unit }
  next.unit_ulid = String(pickFirst(next, ["unit_ulid", "unit_id"]) || "")
  next.glms_course_ulid = String(pickFirst(next, ["glms_course_ulid", "glms_course_id"]) || "")
  next.exam_ulid = String(pickFirst(next, ["exam_ulid", "exam_id"]) || "")
  next.cert_qual_ulid = String(pickFirst(next, ["cert_qual_ulid", "cert_qual_id"]) || "")
  next.cert_pdf_template_ulid = String(pickFirst(next, ["cert_pdf_template_ulid", "cert_pdf_template_id"]) || "")
  next.exemption_quals = Array.isArray(next.exemption_quals) ? next.exemption_quals : []
  delete next.unit_id
  delete next.glms_course_id
  delete next.exam_id
  delete next.cert_qual_id
  delete next.cert_pdf_template_id
  delete next.allow_retake
  delete next.learning_minutes
  delete next.base_fee
  delete next.retake_fee
  delete next.exemption_audit_fee
  return next
}

function structureFromPipeline(pipeline: JsonRecord | null) {
  if (!pipeline) return emptyStructure()
  return normalizeStructureShape(pipeline)
}

function setStructure(next: JsonRecord) {
  structure.value = normalizeStructureShape(next)
  syncStructureJson()
  ensureSelections()
}

function syncStructureJson() {
  form.value.structure_json = JSON.stringify(structure.value, null, 2)
}

function ensureSelections() {
  if (selectedStageIndex.value >= stages.value.length) selectedStageIndex.value = Math.max(0, stages.value.length - 1)
  if (!selectedUnitPath.value || !units.value.some((item) => item.path === selectedUnitPath.value)) selectedUnitPath.value = units.value[0]?.path || ""
  if (selectedCertIndex.value >= certs.value.length) selectedCertIndex.value = Math.max(0, certs.value.length - 1)
  if (selectedUnlockQualIndex.value >= unlockQuals.value.length) selectedUnlockQualIndex.value = Math.max(0, unlockQuals.value.length - 1)
  if (selectedCertQualIndex.value >= certQuals.value.length) selectedCertQualIndex.value = Math.max(0, certQuals.value.length - 1)
}

function formFromPipeline(pipeline: JsonRecord | null): PipelineForm {
  if (!pipeline) return { ...emptyForm }
  return {
    name: String(pipeline.name || ""),
    category_tips: String(pipeline.category_tips || ""),
    pipeline_gpath: String(pipeline.pipeline_gpath || ""),
    structure_json: JSON.stringify(structureFromPipeline(pipeline), null, 2),
  }
}

function parseStructure() {
  try {
    const parsed = JSON.parse(form.value.structure_json || "{}")
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error(copy.value.toasts.structureMustObject)
      return null
    }
    return normalizeStructureShape(parsed as JsonRecord)
  } catch {
    toast.error(copy.value.toasts.structureInvalidJson)
    return null
  }
}

function validateStructureForSave(next: JsonRecord) {
  const certList = asArray(next.certs)
  for (const [certIndex, cert] of certList.entries()) {
    const certQualId = qualificationId(cert)
    const templateId = String(cert.pdf_template_ulid || "")
    if (!certQualId.trim()) {
      toast.error(copy.value.toasts.structureCertQualificationRequired(certIndex + 1))
      selectedCertIndex.value = certIndex
      activeLayer.value = "certs"
      return false
    }
    if (credentialOptions.value.length && !credentialById(certQualId)) {
      toast.error(copy.value.toasts.structureCertQualificationUnavailable(certIndex + 1))
      selectedCertIndex.value = certIndex
      activeLayer.value = "certs"
      return false
    }
    if (!templateId.trim()) {
      toast.error(copy.value.toasts.structureCertPdfTemplateRequired(certIndex + 1))
      selectedCertIndex.value = certIndex
      activeLayer.value = "certs"
      return false
    }
    if (pdfTemplateOptions.value.length && !pdfTemplateById(templateId)) {
      toast.error(copy.value.toasts.structureCertPdfTemplateUnavailable(certIndex + 1))
      selectedCertIndex.value = certIndex
      activeLayer.value = "certs"
      return false
    }
  }

  for (const [qualIndex, qual] of asArray(next.unlock_quals).entries()) {
    const qualId = qualificationId(qual)
    if (!qualId.trim()) {
      toast.error(copy.value.toasts.structureUnlockQualificationRequired(qualIndex + 1))
      selectedUnlockQualIndex.value = qualIndex
      activeLayer.value = "unlock_quals"
      return false
    }
    if (credentialOptions.value.length && !credentialById(qualId)) {
      toast.error(copy.value.toasts.structureUnlockQualificationUnavailable(qualIndex + 1))
      selectedUnlockQualIndex.value = qualIndex
      activeLayer.value = "unlock_quals"
      return false
    }
  }

  for (const [qualIndex, qual] of asArray(next.certs_quals).entries()) {
    const qualId = qualificationId(qual)
    if (!qualId.trim()) {
      toast.error(copy.value.toasts.structureCompletionQualificationRequired(qualIndex + 1))
      selectedCertQualIndex.value = qualIndex
      activeLayer.value = "certs_quals"
      return false
    }
    if (credentialOptions.value.length && !credentialById(qualId)) {
      toast.error(copy.value.toasts.structureCompletionQualificationUnavailable(qualIndex + 1))
      selectedCertQualIndex.value = qualIndex
      activeLayer.value = "certs_quals"
      return false
    }
  }

  const stageList = asArray(next.stages)
  if (!stageList.length) {
    toast.error(copy.value.toasts.structureStagesRequired)
    return false
  }
  for (const [stageIndex, stage] of stageList.entries()) {
    if (!String(stage.name || "").trim()) {
      toast.error(copy.value.toasts.structureStageNameRequired(stageIndex + 1))
      return false
    }
    const unitList = asArray(stage.units)
    if (!unitList.length) {
      toast.error(copy.value.toasts.structureStageUnitsRequired(stageIndex + 1))
      selectedStageIndex.value = stageIndex
      activeLayer.value = "units"
      return false
    }
    for (const [unitIndex, unit] of unitList.entries()) {
      if (!unitCourseId(unit).trim()) {
        toast.error(copy.value.toasts.structureUnitCourseRequired(stageIndex + 1, unitIndex + 1))
        selectedStageIndex.value = stageIndex
        selectedUnitPath.value = `${stageIndex}:${unitIndex}`
        activeLayer.value = "units"
        return false
      }
      if (courseOptions.value.length && !courseById(unitCourseId(unit))) {
        toast.error(copy.value.toasts.structureUnitCourseUnavailable(stageIndex + 1, unitIndex + 1))
        selectedStageIndex.value = stageIndex
        selectedUnitPath.value = `${stageIndex}:${unitIndex}`
        activeLayer.value = "units"
        return false
      }
    }
  }
  return true
}

function applyRawStructure() {
  const parsed = parseStructure()
  if (!parsed) return
  setStructure(parsed)
  toast.success(copy.value.toasts.rawApplied)
}

function eventValue(event: Event) {
  return (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null)?.value || ""
}

function eventNumber(event: Event) {
  const value = Number(eventValue(event))
  return Number.isFinite(value) ? value : 0
}

function eventChecked(event: Event) {
  return Boolean((event.target as HTMLInputElement | null)?.checked)
}

function fieldValue(item: JsonRecord | null | undefined, key: string) {
  const value = item?.[key]
  return value === undefined || value === null ? "" : String(value)
}

function numberValue(item: JsonRecord | null | undefined, key: string) {
  const value = Number(item?.[key])
  return Number.isFinite(value) ? value : 0
}

function boolValue(item: JsonRecord | null | undefined, key: string) {
  return Boolean(item?.[key])
}

function setField(item: JsonRecord | null | undefined, key: string, value: unknown) {
  if (!item || isStructureLocked()) return
  item[key] = value
  syncStructureJson()
}

function addStage() {
  if (isStructureLocked()) return
  const list = asMutableArray(structure.value, "stages")
  list.push({ stage_ulid: "", name: copy.value.defaults.stageName, sort_order: list.length + 1, units: [] })
  selectedStageIndex.value = list.length - 1
  activeLayer.value = "stages"
  syncStructureJson()
  toast.info(copy.value.toasts.stageAddedNeedsUnit)
}

function removeStage(index = selectedStageIndex.value) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, "stages").splice(index, 1)
  ensureSelections()
  syncStructureJson()
}

function addUnit(stageIndex = selectedStageIndex.value) {
  if (isStructureLocked()) return
  if (!stages.value.length) addStage()
  const stage = stages.value[stageIndex] || stages.value[0]
  const list = asMutableArray(stage, "units")
  const maxSort = list.reduce((max, u) => Math.max(max, Number(u.sort_order) || 0), 0)
  list.push({ unit_ulid: "", name: copy.value.defaults.unitName, sort_order: maxSort + 1, glms_course_ulid: "", exemption_quals: [], allow_exemption: false })
  selectedUnitPath.value = `${Math.max(0, stageIndex)}:${list.length - 1}`
  activeLayer.value = "units"
  syncStructureJson()
}

function removeSelectedUnit() {
  if (isStructureLocked() || !selectedUnitItem.value) return
  const item = selectedUnitItem.value
  asMutableArray(item.stage, "units").splice(item.unitIndex, 1)
  ensureSelections()
  syncStructureJson()
}

function moveSelectedUnit(targetStageIndex: number) {
  if (isStructureLocked() || !selectedUnitItem.value) return
  const item = selectedUnitItem.value
  if (targetStageIndex === item.stageIndex) return
  const targetStage = stages.value[targetStageIndex]
  if (!targetStage) return
  asMutableArray(item.stage, "units").splice(item.unitIndex, 1)
  const targetUnits = asMutableArray(targetStage, "units")
  const maxSort = targetUnits.reduce((max, u) => Math.max(max, Number(u.sort_order) || 0), 0)
  item.unit.sort_order = maxSort + 1
  targetUnits.push(item.unit)
  selectedUnitPath.value = `${targetStageIndex}:${targetUnits.length - 1}`
  syncStructureJson()
}

function addCert() {
  addGenericItem("certs", { qual_ulid: "", name_hint: copy.value.defaults.certificateName, pdf_template_ulid: "" }, "certs")
  selectedCertIndex.value = certs.value.length - 1
}

function addUnlockQual() {
  addGenericItem("unlock_quals", { qual_ulid: "", name_hint: copy.value.defaults.unlockQualName }, "unlock_quals")
  selectedUnlockQualIndex.value = unlockQuals.value.length - 1
}

function addCertQual() {
  addGenericItem("certs_quals", { qual_ulid: "", name_hint: copy.value.defaults.certQualName }, "certs_quals")
  selectedCertQualIndex.value = certQuals.value.length - 1
}

function addGenericItem(key: "certs" | "unlock_quals" | "certs_quals", value: JsonRecord, layer: LayerKey) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, key).push(value)
  activeLayer.value = layer
  syncStructureJson()
}

function removeGenericItem(key: "certs" | "unlock_quals" | "certs_quals", index: number) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, key).splice(index, 1)
  ensureSelections()
  syncStructureJson()
}

function applyUnitCourse(unit: JsonRecord | null | undefined, courseUlid: string) {
  if (!unit || isStructureLocked()) return
  unit.glms_course_ulid = courseUlid
  delete unit.glms_course_id
  const course = courseById(courseUlid)
  const currentName = String(unit.name || "").trim()
  if (course && (!currentName || currentName === copy.value.defaults.unitName)) {
    unit.name = courseTitle(course)
  }
  syncStructureJson()
}

function applyQualification(item: JsonRecord | null | undefined, qualUlid: string) {
  if (!item || isStructureLocked()) return
  item.qual_ulid = qualUlid
  delete item.qual_id
  const definition = credentialById(qualUlid)
  if (definition) item.name_hint = credentialName(definition)
  syncStructureJson()
}

function applyUnitCertificateQualification(unit: JsonRecord | null | undefined, qualUlid: string) {
  if (!unit || isStructureLocked()) return
  unit.cert_qual_ulid = qualUlid
  delete unit.cert_qual_id
  syncStructureJson()
}

function toggleExemptionQual(unit: JsonRecord | null | undefined, qualId: string, checked: boolean) {
  if (!unit || isStructureLocked()) return
  let current = Array.isArray(unit.exemption_quals) ? unit.exemption_quals : []
  if (checked) {
    if (!current.includes(qualId)) current.push(qualId)
  } else {
    current = current.filter((id: string) => id !== qualId)
  }
  unit.exemption_quals = current
  syncStructureJson()
}

function applyPdfTemplate(item: JsonRecord | null | undefined, key: "pdf_template_ulid" | "cert_pdf_template_ulid", templateUlid: string) {
  if (!item || isStructureLocked()) return
  item[key] = templateUlid
  if (key === "pdf_template_ulid") delete item.pdf_template_id
  if (key === "cert_pdf_template_ulid") delete item.cert_pdf_template_id
  syncStructureJson()
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(limit),
    })
    if (pageToken.value) params.set("page_token", pageToken.value)
    if (categoryFilter.value.trim()) params.set("category_tips", categoryFilter.value.trim())
    if (onlyCurrent.value) params.set("only_current", "true")
    const data = await apiClient<JsonRecord>(`/api/pipelines?${params}`)
    const list = Array.isArray(data.pipelines) ? data.pipelines : []
    pipelines.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    hasMore.value = Boolean(data.has_more)
    nextToken.value = String(data.next_cursor || "")
    prevToken.value = String(data.prev_cursor || "")
  } catch (err) {
    console.error(err)
    pipelines.value = []
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function loadCourseOptions() {
  courseOptionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/lms/courses?page_size=100")
    const list = Array.isArray(data.courses) ? data.courses : []
    courseOptions.value = list
      .filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
      .filter(courseCanBeConfigured)
  } catch (err) {
    console.error(err)
    courseOptions.value = []
    toast.error(copy.value.toasts.courseOptionsLoadFailed)
  } finally {
    courseOptionsLoading.value = false
  }
}

async function loadCredentialOptions() {
  credentialOptionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    credentialOptions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    credentialOptions.value = []
    toast.error(copy.value.toasts.credentialOptionsLoadFailed)
  } finally {
    credentialOptionsLoading.value = false
  }
}

async function loadPdfTemplateOptions() {
  pdfTemplateOptionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    pdfTemplateOptions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    pdfTemplateOptions.value = []
    toast.error(copy.value.toasts.pdfTemplateOptionsLoadFailed)
  } finally {
    pdfTemplateOptionsLoading.value = false
  }
}

async function selectPipeline(pipeline: JsonRecord) {
  const id = pipelineUlid(pipeline)
  selected.value = pipeline
  creating.value = false
  form.value = formFromPipeline(pipeline)
  setStructure(structureFromPipeline(pipeline))
  activeLayer.value = "overview"
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(id)}`)
    selected.value = detail
    form.value = formFromPipeline(detail)
    setStructure(structureFromPipeline(detail))
  } catch {
    form.value = formFromPipeline(pipeline)
    setStructure(structureFromPipeline(pipeline))
  }
}

function newPipeline() {
  selected.value = null
  creating.value = true
  form.value = { ...emptyForm }
  setStructure(emptyStructure())
  activeLayer.value = "overview"
}

function back() {
  selected.value = null
  creating.value = false
  form.value = { ...emptyForm }
  setStructure(emptyStructure())
}

async function createPipeline() {
  if (!form.value.name.trim() || !form.value.category_tips.trim() || !form.value.pipeline_gpath.trim()) {
    toast.error(copy.value.toasts.requiredCreateFields)
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pipelines", {
      method: "POST",
      body: JSON.stringify({
        name: form.value.name.trim(),
        category_tips: form.value.category_tips.trim(),
        pipeline_gpath: form.value.pipeline_gpath.trim(),
      }),
    })
    toast.success(copy.value.toasts.created)
    creating.value = false
    await load()
    const id = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === id)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.createFailed))
  } finally {
    saving.value = false
  }
}

async function saveMetadata() {
  if (!selectedId.value) return
  if (!form.value.name.trim()) {
    toast.error(copy.value.toasts.nameRequired)
    return
  }
  saving.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/metadata`, {
      method: "PUT",
      body: JSON.stringify({ new_name: form.value.name.trim() }),
    })
    toast.success(copy.value.toasts.metadataSaved)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
  } finally {
    saving.value = false
  }
}

async function saveStructure() {
  if (!selectedId.value) return
  if (published.value) {
    toast.error(copy.value.toasts.publishedStructureLocked)
    return
  }
  const parsed = parseStructure()
  if (!parsed) return
  if (!validateStructureForSave(parsed)) {
    setStructure(parsed)
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(selectedId.value)}/structure`, {
      method: "PUT",
      body: JSON.stringify(parsed),
    })
    toast.success(copy.value.toasts.structureSaved)
    selected.value = data
    form.value = formFromPipeline(data)
    setStructure(structureFromPipeline(data))
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.structureSaveFailed))
  } finally {
    saving.value = false
  }
}

async function publish() {
  if (!selectedId.value) return
  const parsed = parseStructure()
  if (!parsed) return
  if (!validateStructureForSave(parsed)) {
    setStructure(parsed)
    return
  }
  saving.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/publish`, {
      method: "POST",
      body: JSON.stringify({}),
    })
    toast.success(copy.value.toasts.published)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.publishFailed))
  } finally {
    saving.value = false
  }
}

function deprecate() {
  if (!selectedId.value) return
  deprecateConfirmOpen.value = true
}

function closeDeprecateConfirm() {
  if (deprecating.value) return
  deprecateConfirmOpen.value = false
}

async function confirmDeprecate() {
  if (!selectedId.value) return
  deprecating.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/deprecate`, {
      method: "POST",
      body: JSON.stringify({}),
    })
    toast.success(copy.value.toasts.deprecated)
    deprecateConfirmOpen.value = false
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deprecateFailed))
  } finally {
    deprecating.value = false
  }
}

function removePipeline() {
  if (!selectedId.value) return
  if (pipelineStatusKey(selected.value) === "DEPRECATED") return
  if (published.value) {
    toast.error(copy.value.toasts.publishedDeleteBlocked)
    return
  }
  pendingDeletePipeline.value = selected.value
  deleteConfirmOpen.value = true
}

function closeDeleteConfirm() {
  if (deletingPipeline.value) return
  deleteConfirmOpen.value = false
  pendingDeletePipeline.value = null
}

async function confirmDeletePipeline() {
  const pipeline = pendingDeletePipeline.value
  const id = pipeline ? pipelineUlid(pipeline) : ""
  if (!pipeline || !id) return
  deletingPipeline.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(id)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.deleted)
    deleteConfirmOpen.value = false
    pendingDeletePipeline.value = null
    back()
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deleteFailed))
  } finally {
    deletingPipeline.value = false
  }
}

async function clonePipeline() {
  if (!selected.value) return
  const id = selectedId.value
  if (!id) return
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/pipelines/${id}/duplicate`, {
      method: "POST",
      body: JSON.stringify({
        name: copy.value.copyName(form.value.name.trim() || String(selected.value.name || '')),
      }),
    })
    toast.success(copy.value.toasts.cloned)
    await load()
    const newId = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === newId)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.cloneFailed))
  } finally {
    saving.value = false
  }
}

watch([categoryFilter, onlyCurrent], () => { pageToken.value = ""; load() })

watch(() => form.value.name, (newName, oldName) => {
  if (!creating.value) return
  const autoOld = oldName ? `/gcc/pipeline/${oldName.toLowerCase().replace(/[^a-z0-9]/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, "")}` : ""
  if (!form.value.pipeline_gpath || form.value.pipeline_gpath === autoOld) {
    form.value.pipeline_gpath = newName ? `/gcc/pipeline/${newName.toLowerCase().replace(/[^a-z0-9]/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, "")}` : ""
  }
})

onMounted(() => {
  void load()
  void loadCourseOptions()
  void loadCredentialOptions()
  void loadPdfTemplateOptions()
})
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" @click="load">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button v-if="!creating" class="ui-btn ui-btn-primary" type="button" @click="newPipeline">
          <Plus class="h-4 w-4" />
          {{ copy.newPipeline }}
        </button>
      </div>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <div class="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
          <input v-model="categoryFilter" class="ui-input sm:w-64" :placeholder="copy.categoryPlaceholder" />
          <label class="inline-flex h-10 w-full shrink-0 cursor-pointer items-center gap-2 rounded-[10px] border border-slate-200 bg-white px-3 text-sm font-medium text-slate-600 sm:w-auto">
            <input v-model="onlyCurrent" type="checkbox" class="h-4 w-4 accent-primary-600" />
            {{ copy.onlyCurrent }}
          </label>
        </div>
      </div>
      <div v-if="loading" class="divide-y divide-hairline">
        <div
          v-for="i in 5"
          :key="`sk-${i}`"
          class="flex flex-col gap-3 px-4 py-3.5 md:grid md:grid-cols-[minmax(0,1fr)_150px_72px_170px_112px] md:items-center md:gap-4 md:px-6"
        >
          <div>
            <div class="skeleton h-4 w-40 max-w-full" />
            <div class="skeleton mt-2 h-3 w-56 max-w-full" />
          </div>
          <div class="skeleton h-5 w-16 rounded-full md:justify-self-center" />
          <div class="skeleton h-4 w-10 md:justify-self-center" />
          <div class="skeleton h-4 w-24 md:justify-self-end" />
          <div class="skeleton h-4 w-16 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="!pipelines.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Inbox class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <template v-else>
        <div class="ui-table-head hidden grid-cols-[minmax(0,1fr)_150px_72px_170px_112px] gap-4 px-4 py-3 md:grid md:px-6">
          <span>{{ copy.columns.pipeline }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
          <span class="text-center">{{ copy.columns.version }}</span>
          <span class="text-right">{{ copy.columns.updatedAt }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <div
            v-for="pipeline in pipelines"
            :key="pipelineUlid(pipeline)"
            class="ui-table-row flex w-full cursor-pointer flex-col gap-2 px-4 py-3.5 text-left focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20 md:grid md:grid-cols-[minmax(0,1fr)_150px_72px_170px_112px] md:items-center md:gap-4 md:px-6"
            role="button"
            tabindex="0"
            @click="selectPipeline(pipeline)"
            @keydown.enter.prevent="selectPipeline(pipeline)"
            @keydown.space.prevent="selectPipeline(pipeline)"
          >
            <div class="min-w-0">
              <div class="break-words text-sm font-medium text-ink md:truncate">{{ pipelineName(pipeline) }}</div>
              <div class="mt-0.5 line-clamp-1 text-sm text-slate-500">{{ pipeline.description || copy.noDescription }}</div>
              <div class="mt-2 flex flex-wrap items-center gap-2">
                <span class="ui-badge max-w-full bg-slate-100 text-slate-600"><span class="truncate">{{ pipeline.category_tips || "-" }}</span></span>
                <span class="tnum max-w-full break-all text-xs text-slate-400">ID: {{ pipelineUlid(pipeline) || "-" }}</span>
              </div>
            </div>
            <div class="flex items-center justify-between gap-3 md:block md:justify-self-center md:text-center">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
              <span class="ui-badge" :class="badgeClass(pipelineStatus(pipeline))">{{ pipelineStatusLabel(pipelineStatus(pipeline)) }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 md:block md:text-center">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.version }}</span>
              <span class="tnum text-sm text-slate-600">v{{ pipeline.version || 0 }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 md:block md:justify-self-end md:text-right">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.updatedAt }}</span>
              <span class="tnum text-sm text-slate-500">{{ formatDate(String(pipeline.updated_at || pipeline.created_at || "")) }}</span>
            </div>
            <button
              class="inline-flex h-8 items-center justify-center rounded-lg px-3 text-sm font-medium text-primary-700 transition-colors duration-150 hover:bg-primary-50 hover:text-primary-600 md:justify-self-end"
              type="button"
              @click.stop="selectPipeline(pipeline)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </template>
      <div class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="pageToken = prevToken; load()">{{ copy.prev }}</button>
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="pageToken = nextToken; load()">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="inEditor" class="ui-modal-mask z-50">
          <section class="ui-modal-panel max-w-[1320px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:truncate md:text-lg">{{ creating ? copy.newPipeline : form.name || copy.detailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="back"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <section class="min-h-0 flex-1 space-y-5 overflow-y-auto bg-canvas p-4 md:p-6">
              <div class="ui-card p-4 md:p-5">
                <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
                  <div class="min-w-0">
                    <h2 class="text-base font-semibold text-ink">{{ copy.basicTitle }}</h2>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.basicDescription }}</p>
                  </div>
                  <div v-if="!creating" class="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto sm:flex-wrap">
                    <button class="ui-btn ui-btn-secondary" type="button" @click="clonePipeline">
                      <Copy class="h-4 w-4 text-slate-400" />
                      {{ copy.cloneVersion }}
                    </button>
                    <button class="ui-btn ui-btn-secondary" type="button" @click="publish">{{ copy.publish }}</button>
                    <button class="ui-btn ui-btn-secondary" type="button" :disabled="deprecating" @click="deprecate">{{ copy.deprecate }}</button>
                    <button v-if="canDeleteSelectedPipeline" class="ui-btn ui-btn-danger" type="button" @click="removePipeline">
                      <Trash2 class="h-4 w-4" />
                      {{ copy.delete }}
                    </button>
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-1.5 md:col-span-2">
                    <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.fields.name }}</span>
                    <input v-model="form.name" class="ui-input" />
                  </label>
                  <details class="group md:col-span-2">
                    <summary class="inline-flex cursor-pointer select-none items-center gap-1 rounded-lg text-sm font-medium text-slate-500 transition-colors hover:text-slate-700 focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20">
                      {{ (copy as any).advancedConfig || '高级配置' }}
                      <ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" />
                    </summary>
                    <div class="mt-4 grid gap-4 md:grid-cols-2">
                      <label class="grid gap-1.5">
                        <span class="ui-label"><span v-if="creating" class="mr-1 text-red-500">*</span>{{ copy.fields.categoryTips }}</span>
                        <input v-model="form.category_tips" :disabled="!creating" class="ui-input" />
                      </label>
                      <label class="grid gap-1.5">
                        <span class="ui-label"><span v-if="creating" class="mr-1 text-red-500">*</span>{{ copy.fields.respath }}</span>
                        <input v-model="form.pipeline_gpath" :disabled="!creating" class="ui-input" />
                        <p v-if="creating" class="text-xs text-slate-400">{{ copy.respathHint }}</p>
                      </label>
                    </div>
                  </details>
                </div>
              </div>

              <div v-if="!creating" class="ui-card min-h-0 overflow-hidden">
                <div class="border-b border-hairline">
                  <div class="px-4 pt-4 md:px-6 md:pt-5">
                    <h3 class="text-base font-semibold text-ink">{{ copy.layerTitle }}</h3>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.layerDescription }}</p>
                  </div>
                  <div class="mt-2 flex gap-5 overflow-x-auto px-4 md:px-6">
                    <button
                      v-for="layer in layerItems"
                      :key="layer.key"
                      class="ui-tab"
                      :class="activeLayer === layer.key ? 'ui-tab-active' : ''"
                      type="button"
                      @click="activeLayer = layer.key"
                    >
                      {{ layer.title }}
                      <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ layer.count }}</span>
                    </button>
                  </div>
                </div>

                <div v-if="structureLocked" class="border-b border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 md:px-6">
                  {{ copy.structureLockedHint }}
                </div>

                <main class="h-full max-h-none min-h-0 min-w-0 overflow-y-auto md:h-[60vh] md:max-h-[620px] md:min-h-[360px]">
                  <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
                    <h3 class="text-base font-semibold text-ink">{{ layerItems.find((layer) => layer.key === activeLayer)?.title }}</h3>
                    <p class="mt-1 text-sm text-slate-500">{{ layerItems.find((layer) => layer.key === activeLayer)?.desc }}</p>
                  </div>

                  <div v-if="activeLayer === 'overview'" class="grid gap-4 p-4 md:gap-5 md:p-6 lg:grid-cols-2">
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.fields.pipelineId }}</div>
                      <div class="tnum mt-2 break-all text-sm font-medium text-ink">{{ selectedId || "-" }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.fields.pipelineGuid }}</div>
                      <div class="tnum mt-2 break-all text-sm font-medium text-ink">{{ selected?.pipeline_guid || "-" }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.fields.status }}</div>
                      <div class="mt-2"><span class="ui-badge" :class="badgeClass(pipelineStatus(selected || {}))">{{ pipelineStatusLabel(pipelineStatus(selected || {})) }}</span></div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.fields.version }}</div>
                      <div class="tnum mt-2 text-sm font-medium text-ink">v{{ selected?.version || 0 }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.fields.createdAt }}</div>
                      <div class="tnum mt-2 text-sm font-medium text-ink">{{ formatDate(String(selected?.created_at || "")) }}</div>
                    </div>
                  </div>

                  <div v-else-if="activeLayer === 'stages'" class="grid min-h-[620px] lg:grid-cols-[320px_minmax(0,1fr)]">
                    <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                      <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                        <div class="min-w-0">
                          <div class="text-sm font-semibold text-ink">{{ copy.stageListTitle }}</div>
                          <div class="mt-0.5 text-xs text-slate-400">{{ copy.stageListDescription }}</div>
                        </div>
                        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="addStage">{{ copy.newStage }}</button>
                      </div>
                      <button
                        v-for="(stage, index) in stages"
                        :key="`${itemId(stage, ['stage_ulid'])}-${index}`"
                        class="w-full border-b border-hairline p-4 text-left transition-colors hover:bg-primary-50/40"
                        :class="selectedStageIndex === index ? 'bg-primary-50/60' : ''"
                        type="button"
                        @click="selectedStageIndex = index"
                      >
                        <div class="text-sm font-semibold text-ink">{{ itemTitle(stage, copy.stageFallback(index + 1)) }}</div>
                        <div class="tnum mt-1 text-xs text-slate-500">{{ copy.stageMeta(stage.sort_order ?? index + 1, asArray(stage.units).length) }}</div>
                        <div class="tnum mt-1.5 break-all text-xs text-slate-400">ID: {{ itemId(stage, ['stage_ulid']) || "-" }}</div>
                      </button>
                      <div v-if="!stages.length" class="ui-empty">
                        <div class="ui-empty-icon">
                          <Layers class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.noStages }}</p>
                      </div>
                    </div>
                    <div class="space-y-5 p-4 md:p-5">
                      <template v-if="selectedStage">
                        <div class="flex flex-wrap items-center justify-between gap-3">
                          <div class="min-w-0">
                            <h4 class="text-base font-semibold text-ink">{{ copy.stageDetailTitle }}</h4>
                            <p class="mt-0.5 text-sm text-slate-500">{{ copy.stageDetailDescription }}</p>
                          </div>
                          <button class="ui-btn ui-btn-danger ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="removeStage()">{{ copy.deleteStage }}</button>
                        </div>
                        <div class="grid gap-4 md:grid-cols-2">
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.fields.stageId }}</span>
                            <input :value="fieldValue(selectedStage, 'stage_ulid')" disabled class="ui-input tnum" />
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.fields.sortOrder }}</span>
                            <input :value="numberValue(selectedStage, 'sort_order')" :disabled="isStructureLocked()" type="number" class="ui-input tnum" @input="setField(selectedStage, 'sort_order', eventNumber($event))" />
                          </label>
                          <label class="grid gap-1.5 md:col-span-2">
                            <span class="ui-label">{{ copy.fields.name }}</span>
                            <input :value="fieldValue(selectedStage, 'name')" :disabled="isStructureLocked()" class="ui-input" @input="setField(selectedStage, 'name', eventValue($event))" />
                          </label>
                        </div>
                        <section class="rounded-2xl bg-slate-50 p-4">
                          <div class="flex flex-wrap items-start justify-between gap-3">
                            <div class="min-w-0">
                              <h5 class="text-sm font-semibold text-ink">{{ copy.stageUnitsSummaryTitle }}</h5>
                              <p class="mt-1 text-sm text-slate-500">{{ copy.stageUnitsSummaryDescription(selectedStageUnits.length) }}</p>
                            </div>
                            <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ selectedStageUnits.length }}</span>
                          </div>
                          <div v-if="selectedStageUnits.length" class="mt-4 grid gap-3 xl:grid-cols-2">
                            <article
                              v-for="(unit, unitIndex) in selectedStageUnits"
                              :key="`${itemId(unit, ['unit_ulid'])}-${unitIndex}`"
                              class="rounded-2xl border border-edge bg-white p-4"
                            >
                              <div class="flex items-start justify-between gap-3">
                                <div class="min-w-0">
                                  <div class="truncate text-sm font-semibold text-ink">{{ itemTitle(unit, copy.unitFallback(unitIndex + 1)) }}</div>
                                  <div class="tnum mt-1 break-all text-xs text-slate-400">{{ copy.stageUnitSummaryId(itemId(unit, ['unit_ulid'])) }}</div>
                                </div>
                                <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.stageUnitSummarySort(numberValue(unit, 'sort_order') || unitIndex + 1) }}</span>
                              </div>
                              <dl class="mt-4 grid gap-3 text-xs sm:grid-cols-2">
                                <div>
                                  <dt class="font-medium text-slate-400">{{ copy.fields.glmsCourse }}</dt>
                                  <dd class="mt-1 break-all text-slate-600">{{ unitCourseLabel(unit) }}</dd>
                                </div>
                                <div>
                                  <dt class="font-medium text-slate-400">{{ copy.fields.examCode }}</dt>
                                  <dd class="tnum mt-1 break-all text-slate-600">{{ fieldValue(unit, 'exam_ulid') || "-" }}</dd>
                                </div>
                                <div>
                                  <dt class="font-medium text-slate-400">{{ copy.fields.formCode }}</dt>
                                  <dd class="tnum mt-1 break-all text-slate-600">{{ fieldValue(unit, 'form_code') || "-" }}</dd>
                                </div>
                                <div>
                                  <dt class="font-medium text-slate-400">{{ copy.fields.allowExemption }}</dt>
                                  <dd class="mt-1 text-slate-600">{{ boolValue(unit, 'allow_exemption') ? copy.stageUnitSummaryExemptionEnabled : copy.stageUnitSummaryExemptionDisabled }}</dd>
                                </div>
                              </dl>
                            </article>
                          </div>
                          <div v-else class="mt-4 rounded-[10px] border border-dashed border-slate-200 bg-white p-6 text-center text-sm text-slate-500">
                            {{ copy.stageUnitsSummaryEmpty }}
                          </div>
                        </section>
                      </template>
                      <div v-else class="ui-empty">
                        <div class="ui-empty-icon">
                          <Layers class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.selectOrAddStage }}</p>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="activeLayer === 'units'" class="grid min-h-[620px] lg:grid-cols-[360px_minmax(0,1fr)]">
                    <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                      <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                        <div class="min-w-0">
                          <div class="text-sm font-semibold text-ink">{{ copy.unitListTitle }}</div>
                          <div class="mt-0.5 text-xs text-slate-400">{{ copy.unitListDescription }}</div>
                        </div>
                        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="addUnit()">{{ copy.newUnit }}</button>
                      </div>
                      <button
                        v-for="item in units"
                        :key="item.path"
                        class="w-full border-b border-hairline p-4 text-left transition-colors hover:bg-primary-50/40"
                        :class="selectedUnitPath === item.path ? 'bg-primary-50/60' : ''"
                        type="button"
                        @click="selectedUnitPath = item.path"
                      >
                        <div class="text-sm font-semibold text-ink">{{ itemTitle(item.unit, copy.unitFallback(item.unitIndex + 1)) }}</div>
                        <div class="mt-1 text-xs text-slate-500">{{ copy.parentStagePrefix }}{{ itemTitle(item.stage, copy.stageFallback(item.stageIndex + 1)) }}</div>
                        <div class="tnum mt-1.5 break-all text-xs text-slate-400">ID: {{ itemId(item.unit, ['unit_ulid']) || "-" }}</div>
                      </button>
                      <div v-if="!units.length" class="ui-empty">
                        <div class="ui-empty-icon">
                          <BookOpen class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.noUnits }}</p>
                      </div>
                    </div>
                    <div class="space-y-5 p-4 md:p-5">
                      <template v-if="selectedUnitItem">
                        <div class="flex flex-wrap items-center justify-between gap-3">
                          <div class="min-w-0">
                            <h4 class="text-base font-semibold text-ink">{{ copy.unitDetailTitle }}</h4>
                            <p class="mt-0.5 text-sm text-slate-500">{{ copy.unitDetailDescription }}</p>
                          </div>
                          <button class="ui-btn ui-btn-danger ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="removeSelectedUnit">{{ copy.deleteUnit }}</button>
                        </div>
                        <div class="grid gap-4 md:grid-cols-2">
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.fields.parentStage }}</span>
                            <select :value="selectedUnitItem.stageIndex" :disabled="isStructureLocked()" class="ui-input" @change="moveSelectedUnit(eventNumber($event))">
                              <option v-for="(stage, index) in stages" :key="index" :value="index">{{ itemTitle(stage, copy.stageFallback(index + 1)) }}{{ copy.stageOptionId(itemId(stage, ['stage_ulid'])) }}</option>
                            </select>
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.fields.unitId }}</span>
                            <input :value="fieldValue(selectedUnitItem.unit, 'unit_ulid')" disabled class="ui-input tnum" />
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.fields.name }}</span>
                            <input :value="fieldValue(selectedUnitItem.unit, 'name')" :disabled="isStructureLocked()" class="ui-input" @input="setField(selectedUnitItem?.unit, 'name', eventValue($event))" />
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.fields.sortOrder }}</span>
                            <input :value="numberValue(selectedUnitItem.unit, 'sort_order')" :disabled="isStructureLocked()" type="number" class="ui-input tnum" @input="setField(selectedUnitItem?.unit, 'sort_order', eventNumber($event))" />
                          </label>
                          <label class="grid gap-1.5 md:col-span-2">
                            <span class="ui-label">{{ copy.fields.glmsCourse }}</span>
                            <select :value="unitCourseId(selectedUnitItem.unit)" :disabled="isStructureLocked() || courseOptionsLoading" class="ui-input" @change="applyUnitCourse(selectedUnitItem?.unit, eventValue($event))">
                              <option value="">{{ courseOptionsLoading ? copy.loadingCourses : copy.selectCourse }}</option>
                              <option v-for="course in courseOptions" :key="courseId(course)" :value="courseId(course)">{{ courseOptionLabel(course) }}</option>
                            </select>
                            <p class="text-xs text-slate-400">{{ copy.glmsCourseHint }}</p>
                          </label>
                          <div class="rounded-2xl bg-slate-50 p-4 md:col-span-2">
                            <div class="mb-4 text-sm font-semibold text-ink">{{ (copy as any).examConfig || '考试配置' }}</div>
                            <div class="grid gap-4 md:grid-cols-2">
                              <label class="grid gap-1.5">
                                <span class="ui-label">{{ copy.fields.program }}</span>
                                <input :value="fieldValue(selectedUnitItem.unit, 'program')" :disabled="isStructureLocked()" class="ui-input" @input="setField(selectedUnitItem?.unit, 'program', eventValue($event))" />
                              </label>
                              <label class="grid gap-1.5">
                                <span class="ui-label">{{ copy.fields.examCode }}</span>
                                <input :value="fieldValue(selectedUnitItem.unit, 'exam_ulid')" :disabled="isStructureLocked()" class="ui-input tnum" @input="setField(selectedUnitItem?.unit, 'exam_ulid', eventValue($event))" />
                              </label>
                              <label class="grid gap-1.5">
                                <span class="ui-label">{{ copy.fields.formCode }}</span>
                                <input :value="fieldValue(selectedUnitItem.unit, 'form_code')" :disabled="isStructureLocked()" class="ui-input tnum" @input="setField(selectedUnitItem?.unit, 'form_code', eventValue($event))" />
                              </label>
                              <label class="mb-4 inline-flex cursor-pointer items-center gap-2 self-end text-sm font-medium text-slate-600">
                                <input :checked="boolValue(selectedUnitItem.unit, 'allow_exemption')" :disabled="isStructureLocked()" type="checkbox" class="h-4 w-4 accent-primary-600" @change="setField(selectedUnitItem?.unit, 'allow_exemption', eventChecked($event))" />
                                {{ copy.fields.allowExemption }}
                              </label>
                            </div>

                            <div v-if="boolValue(selectedUnitItem.unit, 'allow_exemption')" class="mt-4 border-t border-slate-200 pt-4">
                              <div class="mb-2 flex items-center gap-1 text-sm font-medium text-slate-700">
                                {{ copy.exemptionQualificationsJsonLabel || '豁免资格' }}
                                <span :title="(copy as any).exemptionQualificationsTooltip || '只需满足任一勾选的资格即可免考'" class="flex cursor-help items-center">
                                  <Info class="h-4 w-4 text-slate-400" />
                                </span>
                              </div>
                              <div v-if="credentialOptionsLoading" class="text-xs text-slate-400">{{ copy.loadingQualifications }}</div>
                              <div v-else-if="!credentialOptions.length" class="text-xs text-slate-400">（无可用资格）</div>
                              <div v-else class="grid gap-2 md:grid-cols-2">
                                <label v-for="definition in credentialOptions" :key="credentialId(definition)" class="flex cursor-pointer items-center gap-2 text-sm font-medium text-slate-600">
                                  <input type="checkbox" class="h-4 w-4 shrink-0 accent-primary-600" :value="credentialId(definition)" :checked="(Array.isArray(selectedUnitItem.unit?.exemption_quals) ? selectedUnitItem.unit?.exemption_quals : []).includes(credentialId(definition))" :disabled="isStructureLocked()" @change="toggleExemptionQual(selectedUnitItem?.unit, credentialId(definition), eventChecked($event))" />
                                  <span>{{ credentialOptionLabel(definition) }}</span>
                                </label>
                              </div>
                            </div>
                          </div>
                          <details class="group md:col-span-2">
                            <summary class="inline-flex cursor-pointer select-none items-center gap-1 rounded-lg text-sm font-medium text-slate-500 transition-colors hover:text-slate-700 focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20">
                              {{ (copy as any).certConfig || '资格证书配置' }}
                              <ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" />
                            </summary>
                            <div class="mt-4 grid gap-4 rounded-2xl bg-slate-50 p-4 md:grid-cols-2">
                              <label class="grid gap-1.5 md:col-span-2">
                                <span class="ui-label">{{ copy.unitCertificateQualificationLabel }}</span>
                                <select :value="fieldValue(selectedUnitItem.unit, 'cert_qual_ulid')" :disabled="isStructureLocked() || credentialOptionsLoading" class="ui-input" @change="applyUnitCertificateQualification(selectedUnitItem?.unit, eventValue($event))">
                                  <option value="">{{ credentialOptionsLoading ? copy.loadingQualifications : copy.selectQualification }}</option>
                                  <option v-for="definition in credentialOptions" :key="credentialId(definition)" :value="credentialId(definition)">{{ credentialOptionLabel(definition) }}</option>
                                </select>
                              </label>
                              <label class="grid gap-1.5 md:col-span-2">
                                <span class="ui-label">{{ copy.unitCertificatePdfTemplateLabel }}</span>
                                <select :value="fieldValue(selectedUnitItem.unit, 'cert_pdf_template_ulid')" :disabled="isStructureLocked() || pdfTemplateOptionsLoading" class="ui-input" @change="applyPdfTemplate(selectedUnitItem?.unit, 'cert_pdf_template_ulid', eventValue($event))">
                                  <option value="">{{ pdfTemplateOptionsLoading ? copy.loadingPdfTemplates : copy.selectPdfTemplate }}</option>
                                  <option v-for="template in pdfTemplateOptions" :key="pdfTemplateId(template)" :value="pdfTemplateId(template)">{{ pdfTemplateOptionLabel(template) }}</option>
                                </select>
                              </label>

                            </div>
                          </details>
                        </div>
                      </template>
                      <div v-else class="ui-empty">
                        <div class="ui-empty-icon">
                          <BookOpen class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.selectOrAddUnit }}</p>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="activeLayer === 'certs'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
                    <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                      <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                        <div class="min-w-0">
                          <div class="text-sm font-semibold text-ink">{{ copy.certListTitle }}</div>
                          <div class="mt-0.5 text-xs text-slate-400">{{ copy.certListDescription }}</div>
                        </div>
                        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="addCert">{{ copy.newCert }}</button>
                      </div>
                      <button
                        v-for="(cert, index) in certs"
                        :key="index"
                        class="w-full border-b border-hairline p-4 text-left transition-colors hover:bg-primary-50/40"
                        :class="selectedCertIndex === index ? 'bg-primary-50/60' : ''"
                        type="button"
                        @click="selectedCertIndex = index"
                      >
                        <div class="text-sm font-semibold text-ink">{{ itemTitle(cert, copy.certFallback(index + 1)) }}</div>
                        <div class="tnum mt-1.5 break-all text-xs text-slate-400">{{ copy.qualIdPrefix }}{{ itemId(cert, ['qual_ulid']) || "-" }}</div>
                      </button>
                      <div v-if="!certs.length" class="ui-empty">
                        <div class="ui-empty-icon">
                          <Award class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.noCerts }}</p>
                      </div>
                    </div>
                    <div class="space-y-5 p-4 md:p-5">
                      <template v-if="selectedCert">
                        <div class="flex flex-wrap items-center justify-between gap-3">
                          <h4 class="text-base font-semibold text-ink">{{ copy.certDetailTitle }}</h4>
                          <button class="ui-btn ui-btn-danger ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="removeGenericItem('certs', selectedCertIndex)">{{ copy.deleteCert }}</button>
                        </div>
                        <div class="grid gap-4">
                          <label class="grid gap-1.5">
                            <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.certificateQualificationLabel }}</span>
                            <select :value="fieldValue(selectedCert, 'qual_ulid')" :disabled="isStructureLocked() || credentialOptionsLoading" class="ui-input" @change="applyQualification(selectedCert, eventValue($event))">
                              <option value="">{{ credentialOptionsLoading ? copy.loadingQualifications : copy.selectQualification }}</option>
                              <option v-for="definition in credentialOptions" :key="credentialId(definition)" :value="credentialId(definition)">{{ credentialOptionLabel(definition) }}</option>
                            </select>
                            <p class="text-xs text-slate-400">{{ copy.qualificationSelectHint }}</p>
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.certificatePdfTemplateLabel }}</span>
                            <select :value="fieldValue(selectedCert, 'pdf_template_ulid')" :disabled="isStructureLocked() || pdfTemplateOptionsLoading" class="ui-input" @change="applyPdfTemplate(selectedCert, 'pdf_template_ulid', eventValue($event))">
                              <option value="">{{ pdfTemplateOptionsLoading ? copy.loadingPdfTemplates : copy.selectPdfTemplate }}</option>
                              <option v-for="template in pdfTemplateOptions" :key="pdfTemplateId(template)" :value="pdfTemplateId(template)">{{ pdfTemplateOptionLabel(template) }}</option>
                            </select>
                            <p class="text-xs text-slate-400">{{ copy.pdfTemplateSelectHint }}</p>
                          </label>
                          <label class="grid gap-1.5 md:col-span-2">
                            <span class="ui-label">{{ copy.nameHintLabel }}</span>
                            <input :value="fieldValue(selectedCert, 'name_hint')" :disabled="isStructureLocked()" class="ui-input" @input="setField(selectedCert, 'name_hint', eventValue($event))" />
                          </label>
                        </div>
                      </template>
                      <div v-else class="ui-empty">
                        <div class="ui-empty-icon">
                          <Award class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.selectOrAddCert }}</p>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="activeLayer === 'unlock_quals' || activeLayer === 'certs_quals'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
                    <div class="border-b border-hairline lg:border-b-0 lg:border-r">
                      <div class="flex items-center justify-between gap-3 border-b border-hairline p-4">
                        <div class="min-w-0">
                          <div class="text-sm font-semibold text-ink">{{ activeLayer === 'unlock_quals' ? copy.unlockQualListTitle : copy.certQualListTitle }}</div>
                          <div class="mt-0.5 text-xs text-slate-400">{{ copy.qualListDescription }}</div>
                        </div>
                        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="isStructureLocked()" @click="activeLayer === 'unlock_quals' ? addUnlockQual() : addCertQual()">{{ copy.add }}</button>
                      </div>
                      <template v-if="activeLayer === 'unlock_quals'">
                        <button
                          v-for="(qual, index) in unlockQuals"
                          :key="index"
                          class="w-full border-b border-hairline p-4 text-left transition-colors hover:bg-primary-50/40"
                          :class="selectedUnlockQualIndex === index ? 'bg-primary-50/60' : ''"
                          type="button"
                          @click="selectedUnlockQualIndex = index"
                        >
                          <div class="text-sm font-semibold text-ink">{{ itemTitle(qual, copy.unlockQualFallback(index + 1)) }}</div>
                          <div class="tnum mt-1.5 break-all text-xs text-slate-400">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                        </button>
                        <div v-if="!unlockQuals.length" class="ui-empty">
                          <div class="ui-empty-icon">
                            <ShieldCheck class="h-6 w-6" />
                          </div>
                          <p class="ui-empty-title">{{ copy.noUnlockQuals }}</p>
                        </div>
                      </template>
                      <template v-else>
                        <button
                          v-for="(qual, index) in certQuals"
                          :key="index"
                          class="w-full border-b border-hairline p-4 text-left transition-colors hover:bg-primary-50/40"
                          :class="selectedCertQualIndex === index ? 'bg-primary-50/60' : ''"
                          type="button"
                          @click="selectedCertQualIndex = index"
                        >
                          <div class="text-sm font-semibold text-ink">{{ itemTitle(qual, copy.certQualFallback(index + 1)) }}</div>
                          <div class="tnum mt-1.5 break-all text-xs text-slate-400">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                        </button>
                        <div v-if="!certQuals.length" class="ui-empty">
                          <div class="ui-empty-icon">
                            <GraduationCap class="h-6 w-6" />
                          </div>
                          <p class="ui-empty-title">{{ copy.noCertQuals }}</p>
                        </div>
                      </template>
                    </div>
                    <div class="space-y-5 p-4 md:p-5">
                      <template v-if="activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual">
                        <div class="flex flex-wrap items-center justify-between gap-3">
                          <h4 class="text-base font-semibold text-ink">{{ copy.qualDetailTitle }}</h4>
                          <button
                            class="ui-btn ui-btn-danger ui-btn-sm"
                            type="button"
                            :disabled="isStructureLocked()"
                            @click="activeLayer === 'unlock_quals' ? removeGenericItem('unlock_quals', selectedUnlockQualIndex) : removeGenericItem('certs_quals', selectedCertQualIndex)"
                          >
                            {{ copy.deleteQual }}
                          </button>
                        </div>
                        <div class="grid gap-4">
                          <label class="grid gap-1.5">
                            <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.qualificationDefinitionLabel }}</span>
                            <select
                              :value="qualificationId(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual)"
                              :disabled="isStructureLocked() || credentialOptionsLoading"
                              class="ui-input"
                              @change="applyQualification(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, eventValue($event))"
                            >
                              <option value="">{{ credentialOptionsLoading ? copy.loadingQualifications : copy.selectQualification }}</option>
                              <option v-for="definition in credentialOptions" :key="credentialId(definition)" :value="credentialId(definition)">{{ credentialOptionLabel(definition) }}</option>
                            </select>
                            <p class="text-xs text-slate-400">{{ copy.qualificationSelectHint }}</p>
                          </label>
                          <label class="grid gap-1.5">
                            <span class="ui-label">{{ copy.nameHintLabel }}</span>
                            <input
                              :value="fieldValue(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'name_hint')"
                              :disabled="isStructureLocked()"
                              class="ui-input"
                              @input="setField(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'name_hint', eventValue($event))"
                            />
                          </label>
                        </div>
                        <JsonPreview
                          :title="copy.jsonPreview"
                          :value="activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual"
                          :copy-label="copy.copyJson"
                          :copied-label="copy.copiedJson"
                          :copied-message="copy.toasts.jsonCopied"
                          :copy-error-message="copy.toasts.jsonCopyFailed"
                          max-height="420px"
                        />
                      </template>
                      <div v-else class="ui-empty">
                        <div class="ui-empty-icon">
                          <ShieldCheck class="h-6 w-6" />
                        </div>
                        <p class="ui-empty-title">{{ copy.selectOrAddQual }}</p>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="activeLayer === 'raw'" class="space-y-5 p-4 md:p-6">
                    <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">
                      {{ copy.rawHint }}
                    </div>
                    <textarea v-model="form.structure_json" :disabled="isStructureLocked()" class="tnum min-h-[560px] w-full rounded-[10px] border border-slate-200 bg-white p-4 font-mono text-xs leading-6 text-slate-900 transition-colors focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10 disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-400" />
                    <div class="flex flex-col justify-end gap-3 sm:flex-row">
                      <button class="ui-btn ui-btn-secondary" type="button" :disabled="isStructureLocked()" @click="applyRawStructure">{{ copy.applyRaw }}</button>
                      <button class="ui-btn ui-btn-primary" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
                        <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                        <Send v-else class="h-4 w-4" />
                        {{ copy.saveStructure }}
                      </button>
                    </div>
                  </div>

                  <div v-if="activeLayer !== 'raw'" class="flex flex-col justify-end gap-3 border-t border-hairline p-4 sm:flex-row md:px-6 md:py-5">
                    <button class="ui-btn ui-btn-primary" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
                      <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                      <Send v-else class="h-4 w-4" />
                      {{ copy.saveStructure }}
                    </button>
                  </div>
                </main>
              </div>

              <JsonPreview
                v-if="selected"
                :title="copy.fullDetails"
                :value="selected"
                :copy-label="copy.copyJson"
                :copied-label="copy.copiedJson"
                :copied-message="copy.toasts.jsonCopied"
                :copy-error-message="copy.toasts.jsonCopyFailed"
                max-height="420px"
              />
            </section>

            <div v-if="activeLayer === 'overview' || creating" class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline bg-white px-4 py-4 sm:flex-row md:px-6">
              <button v-if="creating" class="ui-btn ui-btn-primary w-full sm:w-auto sm:min-w-[180px]" type="button" :disabled="saving" @click="createPipeline">
                <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                <Plus v-else class="h-4 w-4" />
                {{ copy.createDraft }}
              </button>
              <button v-else class="ui-btn ui-btn-primary w-full sm:w-auto sm:min-w-[180px]" type="button" :disabled="saving" @click="saveMetadata">
                <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ copy.saveBasic }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="deprecateConfirmOpen && selected" class="ui-modal-mask z-[60]">
          <section class="ui-modal-panel md:max-w-md">
            <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.deprecate }}</h2>
            </div>
            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <p class="text-sm text-slate-600">{{ copy.confirmDeprecate }}</p>
              <div class="mt-4 rounded-2xl bg-slate-50 p-4">
                <div class="break-words text-sm font-semibold text-ink">{{ pipelineName(selected) }}</div>
                <div class="tnum mt-1 break-all text-xs text-slate-400">{{ pipelineUlid(selected) }}</div>
                <div class="tnum mt-1 text-xs text-slate-500">{{ copy.fields.version }}: v{{ selected.version || 0 }}</div>
              </div>
            </main>
            <div class="flex flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="deprecating" @click="closeDeprecateConfirm">{{ copy.cancel }}</button>
              <button class="ui-btn ui-btn-danger" type="button" :disabled="deprecating" @click="confirmDeprecate">
                <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
                {{ copy.deprecate }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="deleteConfirmOpen && pendingDeletePipeline" class="ui-modal-mask z-[60]">
          <section class="ui-modal-panel md:max-w-md">
            <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.deleteConfirmTitle }}</h2>
            </div>
            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <p class="text-sm text-slate-600">{{ copy.deleteConfirmDescription }}</p>
              <div class="mt-4 rounded-2xl bg-slate-50 p-4">
                <div class="break-words text-sm font-semibold text-ink">{{ pipelineName(pendingDeletePipeline) }}</div>
                <div class="tnum mt-1 break-all text-xs text-slate-400">{{ pipelineUlid(pendingDeletePipeline) }}</div>
                <div class="tnum mt-1 text-xs text-slate-500">{{ copy.fields.version }}: v{{ pendingDeletePipeline.version || 0 }}</div>
              </div>
            </main>
            <div class="flex flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="deletingPipeline" @click="closeDeleteConfirm">{{ copy.cancel }}</button>
              <button class="ui-btn ui-btn-danger" type="button" :disabled="deletingPipeline" @click="confirmDeletePipeline">
                <Loader2 v-if="deletingPipeline" class="h-4 w-4 animate-spin" />
                {{ deletingPipeline ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
