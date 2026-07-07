<script setup lang="ts">
import { Copy, FileJson, Loader2, Plus, RefreshCw, Save, Send, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type BundleForm = {
  bundle_ulid: string
  bundle_gpath: string
  name: string
  description: string
  items_json: string
  pricing_json: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
}

type DetailTab = "summary" | "meta" | "pricing" | "schema" | "actions" | "raw"
type Mode = "detail" | "create"
type BundleItemType = "pipeline" | "membership" | "resource_pack"
type SummaryField = {
  label: string
  value: string
}
type DetailField = {
  key: string
  label: string
  value: unknown
}
type LinkedItemView = {
  type: string
  ref: string
}
type PriceRefView = {
  label: string
  priceId: string
  productId: string
}
type UnitPricingView = {
  unitId: string
  prices: PriceRefView[]
}
type UnlockPricingView = {
  targetId: string
  priceId: string
  productId: string
}
type PricingPreviewView = {
  units: UnitPricingView[]
  unlocks: UnlockPricingView[]
  packageCoupon: string
  memberships: number
  qualReviews: number
}
type TargetOption = {
  id: string
  title: string
  subtitle: string
}

const emptyForm: BundleForm = {
  bundle_ulid: "",
  bundle_gpath: "",
  name: "",
  description: "",
  items_json: "[]",
  pricing_json: "{}",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
}

const bundles = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<BundleForm>({ ...emptyForm })
const createItemType = ref<BundleItemType>("pipeline")
const createItemRef = ref("")
const pipelineOptions = ref<JsonRecord[]>([])
const membershipOptions = ref<JsonRecord[]>([])
const resourcePackOptions = ref<JsonRecord[]>([])
const targetOptionsLoading = ref(false)
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const deprecating = ref(false)
const deleting = ref(false)
const duplicating = ref(false)
const detailOpen = ref(false)
const statusFilter = ref("")
const offset = ref(0)
const schemas = ref<JsonRecord | null>(null)
const activeTab = ref<DetailTab>("summary")
const mode = ref<Mode>("detail")
const showDeleteConfirm = ref(false)
const replacementPipelineId = ref("")
const limit = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.bundlesAdmin)

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => bundles.value.length >= limit)
const statusActionBusy = computed(() => publishing.value || deprecating.value || deleting.value || duplicating.value)
const selectedId = computed(() => selected.value ? bundleUlid(selected.value) : "")
const selectedJson = computed(() => JSON.stringify(selected.value || {}, null, 2))
const schemasJson = computed(() => JSON.stringify(schemas.value || {}, null, 2))
const createItemTypeOptions = computed(() => [
  { value: "pipeline" as const, label: copy.value.createItemTypes.pipeline },
  { value: "membership" as const, label: copy.value.createItemTypes.membership },
  { value: "resource_pack" as const, label: copy.value.createItemTypes.resourcePack },
])
const createTargetOptions = computed<TargetOption[]>(() => {
  if (createItemType.value === "membership") return membershipOptions.value.map(membershipTargetOption).filter(hasTargetId)
  if (createItemType.value === "resource_pack") return resourcePackOptions.value.map(resourcePackTargetOption).filter(hasTargetId)
  return pipelineOptions.value.map(pipelineTargetOption).filter(hasTargetId)
})
const replacementPipelineOptions = computed<TargetOption[]>(() => pipelineOptions.value.map(pipelineTargetOption).filter(hasTargetId))
const selectedCreateTarget = computed(() => createTargetOptions.value.find((option) => option.id === createItemRef.value) || null)
const createItemsJson = computed(() => {
  if (!createItemRef.value.trim()) return "[]"
  return JSON.stringify([
    {
      item_type: createItemType.value,
      ref_ulid: createItemRef.value.trim(),
    },
  ], null, 2)
})
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "meta" as const, title: copy.value.tabs.meta, count: 1 },
  { key: "pricing" as const, title: copy.value.tabs.pricing, count: 2 },
  { key: "schema" as const, title: copy.value.tabs.schema, count: schemas.value ? 1 : 0 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: 4 },
  { key: "raw" as const, title: copy.value.tabs.raw, count: 1 },
])
const summaryFields = computed<SummaryField[]>(() => {
  const bundle = selected.value
  if (!bundle) return []
  return [
    { label: copy.value.summary.displayPrice, value: displayPrice(bundle) },
    { label: copy.value.summary.version, value: String(bundle.version ?? "-") },
  ]
})
const linkedItemsPreview = computed<LinkedItemView[]>(() => {
  const parsed = parseJsonSilently(form.value.items_json)
  if (!Array.isArray(parsed)) return []
  return parsed
    .filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    .map((item) => ({
      type: String(pickFirst(item, ["item_type", "type"]) || "-"),
      ref: String(pickFirst(item, ["ref_ulid", "ref_id", "ulid", "id"]) || "-"),
    }))
})
const currentPipelineRefs = computed(() => pipelineRefsFromItemsJson(form.value.items_json))
const currentPipelineRef = computed(() => currentPipelineRefs.value[0] || "")
const missingUnlockPipelineRefs = computed(() => {
  const pricing = asRecord(parseJsonSilently(form.value.pricing_json))
  const unlocks = asRecord(pricing?.unlocks)
  return currentPipelineRefs.value.filter((id) => !unlocks || !Object.prototype.hasOwnProperty.call(unlocks, id))
})
const pricingPreview = computed<PricingPreviewView | null>(() => {
  const parsed = asRecord(parseJsonSilently(form.value.pricing_json))
  if (!parsed) return null
  const units = Array.isArray(parsed.units)
    ? parsed.units.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)).map((unit) => {
      const prices = [
        priceRef(copy.value.pricingPreview.access, unit.access),
        priceRef(copy.value.pricingPreview.retake, unit.retake),
        priceRef(copy.value.pricingPreview.exemption, unit.exemption),
      ].filter((item): item is PriceRefView => !!item)
      return {
        unitId: String(unit.unit_id || "-"),
        prices,
      }
    })
    : []
  const unlocks = Object.entries(asRecord(parsed.unlocks) || {}).map(([targetId, value]) => {
    const ref = asRecord(value)
    return {
      targetId,
      priceId: String(ref?.stripe_price_id || "-"),
      productId: String(ref?.stripe_product_id || "-"),
    }
  })
  return {
    units,
    unlocks,
    packageCoupon: String(parsed.package_coupon || "-"),
    memberships: Array.isArray(parsed.memberships) ? parsed.memberships.length : 0,
    qualReviews: Array.isArray(parsed.qual_reviews) ? parsed.qual_reviews.length : 0,
  }
})
const selectedFields = computed<DetailField[]>(() => {
  if (!selected.value) return []
  return Object.entries(selected.value)
    .filter(([key]) => key !== "version")
    .map(([key, value]) => ({
      key,
      label: copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || key.replaceAll("_", " "),
      value,
    }))
})

function bundleUlid(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["bundle_ulid", "bundle_id"]) || "")
}

function bundleName(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["name", "title"]) || copy.value.unnamed)
}

function bundleStatus(bundle: JsonRecord | null | undefined) {
  return pickFirst(bundle || {}, ["status", "raw_status"])
}

function normalizeItemType(value: unknown) {
  return String(value || "").trim().toLowerCase().replace(/-/g, "_")
}

function itemReference(record: JsonRecord) {
  return String(pickFirst(record, ["ref_ulid", "ref_id", "ulid", "id", "item_id", "pipeline_id", "pipeline_cc_ulid", "membership_id", "resource_pack_id"]) || "").trim()
}

function isPipelineItem(record: JsonRecord) {
  const type = normalizeItemType(pickFirst(record, ["item_type", "type", "itemType", "kind"]))
  return type.includes("pipeline") || !!String(pickFirst(record, ["pipeline_id", "pipeline_cc_ulid"]) || "").trim()
}

function pipelineRefsFromItemsJson(value: string) {
  const parsed = parseJsonSilently(value)
  const items = Array.isArray(parsed) ? parsed : parsed && typeof parsed === "object" ? [parsed] : []
  const refs = new Set<string>()
  for (const item of items) {
    const record = asRecord(item)
    if (!record || !isPipelineItem(record)) continue
    const ref = itemReference(record)
    if (ref) refs.add(ref)
  }
  return Array.from(refs)
}

function bundleTargetSummary(bundle: JsonRecord | null | undefined) {
  const directPipelineId = String(pickFirst(bundle || {}, ["pipeline_id", "pipeline_cc_ulid"]) || "")
  if (directPipelineId) return `${copy.value.createItemTypes.pipeline} · ${directPipelineId}`

  const directMembershipId = String(pickFirst(bundle || {}, ["membership_id", "membership_ulid"]) || "")
  if (directMembershipId) return `${copy.value.createItemTypes.membership} · ${directMembershipId}`

  const parsed = parseJsonSilently(String(bundle?.items_json || ""))
  const items = Array.isArray(parsed) ? parsed : parsed && typeof parsed === "object" ? [parsed] : []
  for (const item of items) {
    const record = asRecord(item)
    if (!record) continue
    const itemType = String(record.item_type || record.type || record.itemType || record.kind || "").toLowerCase()
    const ref = String(record.ref_ulid || record.item_id || record.id || record.pipeline_id || record.pipeline_cc_ulid || record.membership_id || record.resource_pack_id || "").trim()
    if (!ref) continue
    if (itemType.includes("pipeline")) return `${copy.value.createItemTypes.pipeline} · ${ref}`
    if (itemType.includes("membership")) return `${copy.value.createItemTypes.membership} · ${ref}`
    if (itemType.includes("resource")) return `${copy.value.createItemTypes.resourcePack} · ${ref}`
    return `${itemType || copy.value.fields.linkedTarget} · ${ref}`
  }
  return ""
}

function displayPrice(bundle: JsonRecord | null | undefined) {
  const currency = String(bundle?.display_currency || "")
  const min = Number(bundle?.display_amount_min || 0) / 100
  const max = Number(bundle?.display_amount_max || 0) / 100
  if (!currency || (!min && !max)) return "-"
  if (min === max) return `${currency} ${min.toFixed(2)}`
  return `${currency} ${min.toFixed(2)} - ${max.toFixed(2)}`
}

function targetStatusText(target: JsonRecord | null | undefined) {
  return String(pickFirst(target || {}, ["status", "raw_status", "runtime_status"]) || "")
}

function targetVersionText(target: JsonRecord | null | undefined) {
  const version = pickFirst(target || {}, ["version", "revision"])
  return version === undefined || version === null || version === "" ? "" : `v${version}`
}

function targetUsable(target: JsonRecord | null | undefined) {
  const status = targetStatusText(target).toLowerCase()
  return !status.includes("deprecated") && !status.includes("deleted")
}

function hasTargetId(option: TargetOption) {
  return !!option.id
}

function targetSubtitle(parts: string[]) {
  return parts.filter(Boolean).join(" · ")
}

function pipelineTargetOption(target: JsonRecord): TargetOption {
  const id = String(pickFirst(target, ["pipeline_ulid", "pipeline_id"]) || "")
  const title = String(pickFirst(target, ["name", "title", "category_tips"]) || id || copy.value.unnamedTarget)
  return {
    id,
    title,
    subtitle: targetSubtitle([String(pickFirst(target, ["category_tips", "program"]) || ""), targetVersionText(target), targetStatusText(target), id]),
  }
}

function membershipTargetOption(target: JsonRecord): TargetOption {
  const id = String(pickFirst(target, ["membership_ulid", "membership_id"]) || "")
  const title = String(pickFirst(target, ["name", "title", "membership_gpath"]) || id || copy.value.unnamedTarget)
  return {
    id,
    title,
    subtitle: targetSubtitle([String(pickFirst(target, ["membership_gpath", "level", "tier"]) || ""), targetVersionText(target), targetStatusText(target), id]),
  }
}

function resourcePackTargetOption(target: JsonRecord): TargetOption {
  const id = String(pickFirst(target, ["pack_id", "resource_pack_ulid", "resource_pack_id"]) || "")
  const title = String(pickFirst(target, ["title", "name", "respath"]) || id || copy.value.unnamedTarget)
  return {
    id,
    title,
    subtitle: targetSubtitle([String(pickFirst(target, ["category", "respath"]) || ""), targetVersionText(target), targetStatusText(target), id]),
  }
}

function isStructuredValue(value: unknown) {
  return Array.isArray(value) || (!!value && typeof value === "object")
}

function jsonText(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2)
}

function detailFieldText(value: unknown) {
  const text = String(value ?? "").trim()
  return text || "-"
}

function parseJsonSilently(value: string) {
  try {
    return JSON.parse(value || "")
  } catch {
    return null
  }
}

function asRecord(value: unknown): JsonRecord | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : null
}

function priceRef(label: string, value: unknown): PriceRefView | null {
  const ref = asRecord(value)
  if (!ref) return null
  return {
    label,
    priceId: String(ref.stripe_price_id || "-"),
    productId: String(ref.stripe_product_id || "-"),
  }
}

function parseJson(value: string, field: string) {
  try {
    return JSON.parse(value || "")
  } catch {
    toast.error(copy.value.jsonInvalid(field))
    return null
  }
}

function isBlank(value: unknown) {
  return String(value ?? "").trim() === ""
}

function validateItemsJson() {
  const parsed = parseJson(form.value.items_json, "items_json")
  if (parsed === null) return false
  if (!Array.isArray(parsed) || parsed.length === 0) {
    toast.error(copy.value.toasts.itemsRequired)
    return false
  }

  for (const [index, item] of parsed.entries()) {
    const record = asRecord(item)
    if (!record || isBlank(record.item_type) || isBlank(record.ref_ulid)) {
      toast.error(copy.value.toasts.itemRequired(index + 1))
      return false
    }
  }
  return true
}

function validatePriceObject(value: unknown, label: string) {
  if (value === undefined || value === null || value === "") return true
  const record = asRecord(value)
  if (!record || isBlank(record.stripe_price_id) || isBlank(record.stripe_product_id)) {
    toast.error(copy.value.toasts.priceRefRequired(label))
    return false
  }
  return true
}

function validatePricingJson() {
  const parsed = parseJson(form.value.pricing_json, "pricing_json")
  if (parsed === null) return false
  const pricing = asRecord(parsed)
  if (!pricing) {
    toast.error(copy.value.toasts.pricingObjectRequired)
    return false
  }

  if (Array.isArray(pricing.units)) {
    for (const [index, value] of pricing.units.entries()) {
      const unit = asRecord(value)
      if (!unit || isBlank(unit.unit_id)) {
        toast.error(copy.value.toasts.unitIdRequired(index + 1))
        return false
      }
      if (!validatePriceObject(unit.access, copy.value.pricingPreview.access)) return false
      if (!validatePriceObject(unit.retake, copy.value.pricingPreview.retake)) return false
      if (!validatePriceObject(unit.exemption, copy.value.pricingPreview.exemption)) return false
    }
  }

  const unlocks = asRecord(pricing.unlocks)
  if (unlocks) {
    for (const [targetId, value] of Object.entries(unlocks)) {
      if (!validatePriceObject(value, copy.value.toasts.unlockPriceLabel(targetId))) return false
    }
  }

  return true
}

function validateStructureJson() {
  return validateItemsJson() && validatePricingJson()
}

function cloneJsonRecord(record: JsonRecord) {
  return JSON.parse(JSON.stringify(record)) as JsonRecord
}

function validPriceRecord(value: unknown) {
  const record = asRecord(value)
  if (!record || isBlank(record.stripe_price_id) || isBlank(record.stripe_product_id)) return null
  return record
}

function firstUsableUnlockPrice(pricing: JsonRecord) {
  const unlocks = asRecord(pricing.unlocks)
  if (unlocks) {
    for (const value of Object.values(unlocks)) {
      const price = validPriceRecord(value)
      if (price) return cloneJsonRecord(price)
    }
  }

  if (Array.isArray(pricing.units)) {
    for (const value of pricing.units) {
      const unit = asRecord(value)
      const accessPrice = validPriceRecord(unit?.access)
      if (accessPrice) return cloneJsonRecord(accessPrice)
    }
  }

  return null
}

function ensurePipelineUnlocksInForm(targetIds = missingUnlockPipelineRefs.value) {
  const uniqueTargetIds = Array.from(new Set(targetIds.map((id) => id.trim()).filter(Boolean)))
  if (!uniqueTargetIds.length) {
    toast.error(copy.value.toasts.unlockPriceNoMissing)
    return false
  }

  const pricing = parseJson(form.value.pricing_json, "pricing_json")
  const pricingRecord = asRecord(pricing)
  if (!pricingRecord) {
    toast.error(copy.value.toasts.relinkInvalidJson)
    return false
  }

  const fallbackPrice = firstUsableUnlockPrice(pricingRecord)
  if (!fallbackPrice) {
    toast.error(copy.value.toasts.unlockPriceNoSource)
    return false
  }

  const unlocks = asRecord(pricingRecord.unlocks) || {}
  for (const targetId of uniqueTargetIds) {
    if (!Object.prototype.hasOwnProperty.call(unlocks, targetId)) {
      unlocks[targetId] = cloneJsonRecord(fallbackPrice)
    }
  }
  pricingRecord.unlocks = unlocks
  form.value.pricing_json = JSON.stringify(pricingRecord, null, 2)
  toast.success(copy.value.toasts.unlockPriceFilled(uniqueTargetIds.length))
  return true
}

function validatePublishPricing() {
  if (!validateStructureJson()) return false
  if (missingUnlockPipelineRefs.value.length) {
    toast.error(copy.value.toasts.unlockPriceMissing(missingUnlockPipelineRefs.value.join(", ")))
    activeTab.value = "pricing"
    return false
  }
  return true
}

function formFromBundle(bundle: JsonRecord | null): BundleForm {
  if (!bundle) return { ...emptyForm }
  return {
    bundle_ulid: String(bundle.bundle_ulid || bundle.bundle_id || ""),
    bundle_gpath: String(bundle.bundle_gpath || ""),
    name: String(bundle.name || ""),
    description: String(bundle.description || ""),
    items_json: String(bundle.items_json || "[]"),
    pricing_json: String(bundle.pricing_json || "{}"),
    thumbnail_object_key: String(bundle.thumbnail_object_key || ""),
    thumbnail_file_hash: String(bundle.thumbnail_file_hash || ""),
  }
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset.value),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mall/bundles?${params}`)
    const list = Array.isArray(data.bundles) ? data.bundles : []
    bundles.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value && bundles.value.length) {
      await selectBundle(bundles.value[0], false)
    }
  } catch (err) {
    console.error(err)
    bundles.value = []
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

function recordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

async function loadCreateTargetOptions() {
  targetOptionsLoading.value = true
  const failures: string[] = []
  const loadRecords = async (label: string, request: Promise<JsonRecord>, keys: string[]) => {
    try {
      const data = await request
      for (const key of keys) {
        const list = recordList(data[key])
        if (list.length) return list
      }
      return []
    } catch (err) {
      console.error(err)
      failures.push(label)
      return []
    }
  }
  try {
    const [pipelines, memberships, packs] = await Promise.all([
      loadRecords("pipelines", apiClient<JsonRecord>("/api/pipelines?limit=200&offset=0&only_current=true"), ["pipelines", "items"]),
      loadRecords("memberships", apiClient<JsonRecord>("/api/memberships?page=1&page_size=200"), ["memberships", "membership_configs", "items"]),
      loadRecords("resource-packs", apiClient<JsonRecord>("/api/lms/resource-packs?page_size=200"), ["packs", "items"]),
    ])
    pipelineOptions.value = pipelines.filter(targetUsable)
    membershipOptions.value = memberships.filter(targetUsable)
    resourcePackOptions.value = packs.filter(targetUsable)
  } finally {
    if (failures.length) {
      console.warn("Some product target options failed to load", failures)
      toast.error(copy.value.toasts.targetOptionsLoadFailed)
    }
    targetOptionsLoading.value = false
  }
}

async function selectBundle(bundle: JsonRecord, open = true) {
  const id = bundleUlid(bundle)
  selected.value = bundle
  detailOpen.value = open
  mode.value = "detail"
  activeTab.value = "summary"
  showDeleteConfirm.value = false
  replacementPipelineId.value = ""
  form.value = formFromBundle(bundle)
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/mall/bundles/${encodeURIComponent(id)}`)
    const actualBundle = (detail.bundle && typeof detail.bundle === "object" ? detail.bundle : detail) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
  } catch {
    form.value = formFromBundle(bundle)
  }
}

function newBundle() {
  selected.value = null
  detailOpen.value = true
  mode.value = "create"
  activeTab.value = "meta"
  showDeleteConfirm.value = false
  createItemType.value = "pipeline"
  createItemRef.value = ""
  form.value = { ...emptyForm }
  void loadCreateTargetOptions()
}

function closeDetail() {
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

async function createBundle() {
  form.value.items_json = createItemsJson.value
  if (!form.value.bundle_gpath.trim() || !form.value.name.trim()) {
    toast.error(copy.value.toasts.createRequired)
    return
  }
  if (!createItemRef.value.trim()) {
    toast.error(copy.value.toasts.targetRequired)
    return
  }
  if (!validateStructureJson()) return

  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles", {
      method: "POST",
      body: JSON.stringify({
        bundle_gpath: form.value.bundle_gpath.trim(),
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success(copy.value.toasts.created)
    await load()
    const id = String(pickFirst(data, ["bundle_ulid", "bundle_id"]) || "")
    const created = bundles.value.find((item) => bundleUlid(item) === id)
    if (created) await selectBundle(created)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.createFailed))
  } finally {
    saving.value = false
  }
}

async function saveMeta() {
  if (!selectedId.value) return
  saving.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/meta`, {
      method: "PUT",
      body: JSON.stringify({
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success(copy.value.toasts.metaSaved)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
  } finally {
    saving.value = false
  }
}

async function savePricing() {
  if (!selectedId.value) return
  if (!validateStructureJson()) return
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles/pricing", {
      method: "PUT",
      body: JSON.stringify({
        bundle_ulid: selectedId.value,
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
      }),
    })
    toast.success(copy.value.toasts.pricingSaved)
    const actualBundle = (data.bundle && typeof data.bundle === "object" ? data.bundle : data) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
  } finally {
    saving.value = false
  }
}

async function duplicateBundle() {
  if (!selected.value || !selectedId.value || statusActionBusy.value) return
  duplicating.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/duplicate`, {
      method: "POST",
      body: JSON.stringify({ name: copy.value.duplicateName(bundleName(selected.value)) }),
    })
    toast.success(copy.value.toasts.duplicated)
    await load()
    const actualBundle = (data.bundle && typeof data.bundle === "object" ? data.bundle : data) as JsonRecord
    const id = String(pickFirst(actualBundle, ["bundle_ulid", "bundle_id"]) || "")
    const created = bundles.value.find((item) => bundleUlid(item) === id) || actualBundle
    await selectBundle(created)
    activeTab.value = "pricing"
    await loadCreateTargetOptions()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.duplicateFailed))
  } finally {
    duplicating.value = false
  }
}

function replacePipelineBindingInForm() {
  const fromId = currentPipelineRef.value.trim()
  const toId = replacementPipelineId.value.trim()
  if (!fromId || !toId) {
    toast.error(copy.value.toasts.relinkRequiresTarget)
    return false
  }
  if (fromId === toId) {
    toast.error(copy.value.toasts.relinkNoChange)
    return false
  }

  const items = parseJson(form.value.items_json, "items_json")
  const pricing = parseJson(form.value.pricing_json, "pricing_json")
  if (!Array.isArray(items) || pricing === null) {
    toast.error(copy.value.toasts.relinkInvalidJson)
    return false
  }
  const pricingRecord = asRecord(pricing)
  if (!pricingRecord) {
    toast.error(copy.value.toasts.relinkInvalidJson)
    return false
  }

  let changed = false
  const refKeys = ["ref_ulid", "ref_id", "ulid", "id", "item_id", "pipeline_id", "pipeline_cc_ulid"]
  for (const item of items) {
    const record = asRecord(item)
    if (!record) continue
    if (!isPipelineItem(record) && !refKeys.some((key) => String(record[key] || "").trim() === fromId)) continue
    for (const key of refKeys) {
      if (String(record[key] || "").trim() === fromId) {
        record[key] = toId
        changed = true
      }
    }
  }

  const unlocks = asRecord(pricingRecord.unlocks)
  if (unlocks && Object.prototype.hasOwnProperty.call(unlocks, fromId)) {
    if (!Object.prototype.hasOwnProperty.call(unlocks, toId)) {
      unlocks[toId] = unlocks[fromId]
    }
    delete unlocks[fromId]
    changed = true
  } else if (changed) {
    const fallbackPrice = firstUsableUnlockPrice(pricingRecord)
    if (fallbackPrice) {
      const nextUnlocks = unlocks || {}
      nextUnlocks[toId] = fallbackPrice
      pricingRecord.unlocks = nextUnlocks
    }
  }

  if (!changed) {
    toast.error(copy.value.toasts.relinkNoChange)
    return false
  }

  form.value.items_json = JSON.stringify(items, null, 2)
  form.value.pricing_json = JSON.stringify(pricingRecord, null, 2)
  toast.success(copy.value.toasts.relinkApplied)
  return true
}

async function replaceAndSavePipelineBinding() {
  if (!replacePipelineBindingInForm()) return
  await savePricing()
}

async function publish() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  if (!validatePublishPricing()) return
  publishing.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/publish`, { method: "POST" })
    toast.success(copy.value.toasts.published)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.publishFailed))
  } finally {
    publishing.value = false
  }
}

async function deprecate() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deprecating.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/deprecate`, { method: "POST" })
    toast.success(copy.value.toasts.deprecated)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deprecateFailed))
  } finally {
    deprecating.value = false
  }
}

async function removeBundle() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deleting.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.deleted)
    selected.value = null
    detailOpen.value = false
    form.value = { ...emptyForm }
    showDeleteConfirm.value = false
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deleteFailed))
  } finally {
    deleting.value = false
  }
}

async function syncDisplayPricing() {
  await apiClient("/api/mall/bundles/sync-display-pricing", {
    method: "POST",
    body: JSON.stringify({ bundle_ulid: selectedId.value || undefined }),
  })
  toast.success(copy.value.toasts.displayPricingSynced)
  await load()
}

async function loadSchemas() {
  schemas.value = await apiClient<JsonRecord>("/api/mall/bundles/schemas")
  activeTab.value = "schema"
}

watch([statusFilter, offset], () => {
  selected.value = null
  void load()
})
watch(createItemType, () => {
  createItemRef.value = ""
})
watch(createItemsJson, (value) => {
  if (mode.value === "create") form.value.items_json = value
})
watch(activeTab, (tab) => {
  if (tab === "pricing" && !pipelineOptions.value.length) void loadCreateTargetOptions()
})
onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newBundle">
          <Plus class="h-4 w-4" />
          {{ copy.newBundle }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="syncDisplayPricing">
          <RefreshCw class="h-4 w-4" />
          {{ copy.syncDisplayPricing }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div class="flex items-center gap-3">
            <div>
              <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ bundles.length }}</span>
          </div>
          <select v-model="statusFilter" class="h-10 w-full rounded-xl border border-slate-200 px-4 text-sm md:w-64">
            <option value="">{{ copy.allStatus }}</option>
            <option value="Draft">{{ copy.statusOptions.Draft }}</option>
            <option value="Active">{{ copy.statusOptions.Active }}</option>
            <option value="Deprecated">{{ copy.statusOptions.Deprecated }}</option>
          </select>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <template v-else>
          <div class="grid grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
            <span>{{ copy.columns.bundle }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-center">{{ copy.columns.version }}</span>
            <span class="text-right">{{ copy.columns.updatedAt }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <div
            v-for="bundle in bundles"
            :key="bundleUlid(bundle)"
            class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
            :class="mode === 'detail' && selectedId === bundleUlid(bundle) ? 'bg-sky-50' : ''"
            role="button"
            tabindex="0"
            @click="selectBundle(bundle)"
            @keydown.enter.prevent="selectBundle(bundle)"
            @keydown.space.prevent="selectBundle(bundle)"
          >
            <div class="min-w-0">
              <div class="truncate text-lg font-black">{{ bundleName(bundle) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ bundle.description || copy.noDescription }}</div>
              <div class="mt-2 flex flex-wrap gap-2 text-xs font-semibold text-slate-500">
                <span class="rounded-full bg-slate-100 px-2 py-1">{{ copy.displayPricePrefix }}{{ displayPrice(bundle) }}</span>
                <span v-if="bundleTargetSummary(bundle)" class="rounded-full bg-slate-100 px-2 py-1">{{ copy.linkedTargetPrefix }}{{ bundleTargetSummary(bundle) }}</span>
                <span class="rounded-full bg-slate-100 px-2 py-1">{{ copy.fields.idPrefix }}{{ bundleUlid(bundle) || "-" }}</span>
              </div>
            </div>
            <span class="self-center justify-self-center rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(bundle))">{{ bundleStatus(bundle) || "-" }}</span>
            <span class="self-center text-center text-sm font-black text-slate-700">{{ copy.fields.versionPrefix }} {{ bundle.version || 0 }}</span>
            <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(bundle.updated_at || bundle.created_at || "")) }}</span>
            <button class="self-center justify-self-end text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click.stop="selectBundle(bundle)">
              {{ copy.viewDetails }}
            </button>
          </div>
        </template>
        <div v-if="!loading && !bundles.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">{{ copy.next }}</button>
        </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1320px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <template v-if="mode === 'create'">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-2xl font-black">{{ copy.createTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.createDescription }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="flex-1 space-y-5 overflow-y-auto p-5">
            <section class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="text-lg font-black text-slate-950">{{ copy.createSections.linkTarget }}</h3>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.createSections.linkTargetDesc }}</p>
                </div>
                <Loader2 v-if="targetOptionsLoading" class="h-5 w-5 animate-spin text-slate-400" />
              </div>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.itemType }}
                  <select v-model="createItemType" class="rounded-xl border border-slate-200 bg-white px-4 py-3">
                    <option v-for="option in createItemTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                  </select>
                  <p class="text-xs font-semibold text-slate-500">{{ copy.itemTypeHint }}</p>
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.linkedTarget }}
                  <select v-model="createItemRef" class="rounded-xl border border-slate-200 bg-white px-4 py-3" :disabled="targetOptionsLoading || !createTargetOptions.length">
                    <option value="" disabled>{{ targetOptionsLoading ? copy.loadingTargets : copy.selectLinkedTarget }}</option>
                    <option v-for="option in createTargetOptions" :key="option.id" :value="option.id">{{ option.title }} · {{ option.subtitle }}</option>
                  </select>
                  <p class="text-xs font-semibold text-slate-500">{{ createTargetOptions.length ? copy.linkedTargetHint : copy.noLinkedTargets }}</p>
                </label>
              </div>
              <div v-if="selectedCreateTarget" class="mt-4 rounded-2xl border border-sky-100 bg-white p-4">
                <div class="text-xs font-black uppercase tracking-wide text-slate-400">{{ copy.selectedTarget }}</div>
                <div class="mt-2 text-base font-black text-slate-950">{{ selectedCreateTarget.title }}</div>
                <div class="mt-1 break-all text-xs font-semibold text-slate-500">{{ selectedCreateTarget.subtitle }}</div>
              </div>
            </section>

            <section class="rounded-2xl border border-slate-200 p-5">
              <h3 class="text-lg font-black text-slate-950">{{ copy.createSections.basicInfo }}</h3>
              <p class="mt-1 text-sm text-slate-500">{{ copy.createSections.basicInfoDesc }}</p>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.bundleGpath }}
                  <input v-model="form.bundle_gpath" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.placeholders.bundleGpath" />
                  <p class="text-xs font-semibold text-slate-500">{{ copy.bundleGpathHint }}</p>
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.name }}
                  <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" :placeholder="copy.placeholders.name" />
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.thumbnailObjectKey }}
                  <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.placeholders.thumbnailObjectKey" />
                  <p class="text-xs font-semibold text-slate-500">{{ copy.optionalImageHint }}</p>
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.thumbnailFileHash }}
                  <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.placeholders.thumbnailFileHash" />
                </label>
                <label class="grid gap-2 text-sm font-bold md:col-span-2">
                  {{ copy.fields.description }}
                  <textarea v-model="form.description" class="min-h-24 rounded-xl border border-slate-200 p-4" maxlength="1200" :placeholder="copy.placeholders.description" />
                </label>
              </div>
            </section>

            <details class="rounded-2xl border border-slate-200 bg-white p-5">
              <summary class="cursor-pointer text-sm font-black text-slate-700">{{ copy.advancedJsonTitle }}</summary>
              <p class="mt-2 text-sm text-slate-500">{{ copy.advancedJsonHint }}</p>
              <div class="mt-4 grid gap-4 xl:grid-cols-2">
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.itemsJson }}
                  <textarea :value="createItemsJson" readonly class="min-h-[180px] rounded-xl border border-slate-200 bg-slate-50 p-4 font-mono text-xs leading-6 text-slate-600" />
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.fields.pricingJson }}
                  <textarea v-model="form.pricing_json" class="min-h-[180px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  <p class="text-xs font-semibold text-slate-500">{{ copy.pricingJsonCreateHint }}</p>
                </label>
              </div>
            </details>
          </div>
          <div class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-5 py-4">
            <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="createBundle">
              <Plus class="h-4 w-4" />
              {{ copy.createDraft }}
            </button>
          </div>
        </template>

        <div v-else-if="!selected" class="flex items-start justify-between gap-4 p-6">
          <div>
            <h2 class="text-2xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <template v-else>
          <div class="border-b border-slate-200 p-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 class="text-2xl font-black">{{ bundleName(selected) }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ selectedId }}</p>
              </div>
              <div class="flex items-center gap-3">
                <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>

          <div class="border-b border-slate-200 p-4">
            <div class="flex gap-2 overflow-x-auto">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="inline-flex h-11 shrink-0 items-center gap-3 rounded-2xl border px-4 text-sm font-black transition"
                :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 text-slate-950' : 'border-slate-100 bg-white text-slate-700 hover:bg-slate-50'"
                type="button"
                @click="activeTab = tab.key"
              >
                <span>{{ tab.title }}</span>
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden">
            <main class="h-[60vh] min-h-[360px] max-h-[620px] min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                    <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                  </div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-white p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.description }}</div>
                  <p class="mt-2 whitespace-pre-wrap text-sm font-semibold leading-6 text-slate-700">{{ form.description || "-" }}</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in selectedFields" :key="field.key" class="grid gap-2 text-sm font-bold" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                    <span class="text-xs font-black uppercase text-slate-400">{{ field.label }}</span>
                    <pre
                      v-if="isStructuredValue(field.value)"
                      class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs leading-5 text-slate-700"
                    >{{ jsonText(field.value) }}</pre>
                    <div v-else class="min-h-11 break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold leading-5 text-slate-700">
                      {{ detailFieldText(field.value) }}
                    </div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'meta'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.bundleUlid }}
                    <input v-model="form.bundle_ulid" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.bundleGpath }}
                    <input v-model="form.bundle_gpath" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.name }}
                    <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.thumbnailObjectKey }}
                    <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    {{ copy.fields.description }}
                    <textarea v-model="form.description" class="min-h-28 rounded-xl border border-slate-200 p-4" maxlength="1200" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    {{ copy.fields.thumbnailFileHash }}
                    <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMeta">
                    <Save class="h-4 w-4" />
                    {{ copy.saveMeta }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'pricing'" class="space-y-5">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.jsonValidateHint }}
                </div>
                <section class="rounded-2xl border border-sky-200 bg-sky-50 p-5">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div>
                      <h3 class="text-lg font-black text-slate-950">{{ copy.relink.title }}</h3>
                      <p class="mt-1 text-sm font-semibold text-slate-600">{{ copy.relink.description }}</p>
                    </div>
                    <button class="rounded-xl border bg-white px-4 py-2 text-sm font-bold disabled:opacity-50" type="button" :disabled="targetOptionsLoading" @click="loadCreateTargetOptions">
                      <Loader2 v-if="targetOptionsLoading" class="mr-2 inline h-4 w-4 animate-spin" />
                      {{ copy.relink.reloadTargets }}
                    </button>
                  </div>
                  <div class="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_auto]">
                    <div class="rounded-xl bg-white p-4">
                      <div class="text-xs font-black uppercase text-slate-400">{{ copy.relink.currentPipeline }}</div>
                      <div v-if="currentPipelineRef" class="mt-2 break-all font-mono text-xs font-bold text-blue-700">{{ currentPipelineRef }}</div>
                      <div v-else class="mt-2 text-sm font-semibold text-slate-500">{{ copy.relink.noPipelineBinding }}</div>
                    </div>
                    <label class="grid gap-2 text-sm font-bold">
                      {{ copy.relink.newPipeline }}
                      <select v-model="replacementPipelineId" class="h-12 rounded-xl border border-slate-200 bg-white px-4" :disabled="!currentPipelineRef || targetOptionsLoading">
                        <option value="">{{ copy.relink.selectNewPipeline }}</option>
                        <option v-for="option in replacementPipelineOptions" :key="option.id" :value="option.id">
                          {{ option.title }} · {{ option.subtitle }}
                        </option>
                      </select>
                    </label>
                    <div class="flex flex-wrap items-end gap-2">
                      <button class="h-12 rounded-xl border bg-white px-4 text-sm font-bold disabled:opacity-50" type="button" :disabled="!currentPipelineRef || !replacementPipelineId || saving" @click="replacePipelineBindingInForm">
                        {{ copy.relink.apply }}
                      </button>
                      <button class="h-12 rounded-xl bg-blue-700 px-4 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="!currentPipelineRef || !replacementPipelineId || saving" @click="replaceAndSavePipelineBinding">
                        <Loader2 v-if="saving" class="mr-2 inline h-4 w-4 animate-spin" />
                        {{ copy.relink.save }}
                      </button>
                    </div>
                  </div>
                  <div v-if="missingUnlockPipelineRefs.length" class="mt-4 rounded-2xl border border-amber-200 bg-amber-50 p-4">
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <h4 class="font-black text-amber-950">{{ copy.relink.unlockMissingTitle }}</h4>
                        <p class="mt-1 text-sm font-semibold text-amber-800">{{ copy.relink.unlockMissingDescription }}</p>
                      </div>
                      <button class="rounded-xl bg-amber-600 px-4 py-2 text-sm font-black text-white disabled:opacity-50" type="button" :disabled="saving" @click="ensurePipelineUnlocksInForm()">
                        {{ copy.relink.fillUnlockPrice }}
                      </button>
                    </div>
                    <div class="mt-3 grid gap-2">
                      <div v-for="id in missingUnlockPipelineRefs" :key="id" class="break-all rounded-xl bg-white px-3 py-2 font-mono text-xs font-bold text-amber-900">
                        {{ id }}
                      </div>
                    </div>
                  </div>
                </section>
                <section class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h3 class="text-lg font-black text-slate-950">{{ copy.pricingPreview.title }}</h3>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.pricingPreview.description }}</p>
                    </div>
                    <span class="rounded-full bg-white px-3 py-1 text-xs font-black text-slate-600">
                      {{ copy.summary.displayPrice }}: {{ selected ? displayPrice(selected) : "-" }}
                    </span>
                  </div>

                  <div class="mt-5 grid gap-4 xl:grid-cols-2">
                    <div class="rounded-2xl border border-slate-200 bg-white p-4">
                      <h4 class="font-black text-slate-900">{{ copy.pricingPreview.linkedItems }}</h4>
                      <div v-if="linkedItemsPreview.length" class="mt-3 grid gap-2">
                        <div v-for="item in linkedItemsPreview" :key="`${item.type}-${item.ref}`" class="rounded-xl bg-slate-50 p-3 text-sm">
                          <div class="font-bold text-slate-600">{{ copy.pricingPreview.itemType }}: {{ item.type }}</div>
                          <div class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ copy.pricingPreview.itemRef }}: {{ item.ref }}</div>
                        </div>
                      </div>
                      <div v-else class="mt-3 rounded-xl border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
                        {{ copy.pricingPreview.emptyJson }}
                      </div>
                    </div>

                    <div class="rounded-2xl border border-slate-200 bg-white p-4">
                      <h4 class="font-black text-slate-900">{{ copy.pricingPreview.otherConfig }}</h4>
                      <div v-if="pricingPreview" class="mt-3 grid gap-2 text-sm">
                        <div class="rounded-xl bg-slate-50 p-3">
                          <span class="font-bold text-slate-500">{{ copy.pricingPreview.packageCoupon }}</span>
                          <div class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ pricingPreview.packageCoupon }}</div>
                        </div>
                        <div class="grid gap-2 sm:grid-cols-2">
                          <div class="rounded-xl bg-slate-50 p-3">
                            <span class="font-bold text-slate-500">{{ copy.pricingPreview.memberships }}</span>
                            <div class="mt-1 text-lg font-black">{{ pricingPreview.memberships }}</div>
                          </div>
                          <div class="rounded-xl bg-slate-50 p-3">
                            <span class="font-bold text-slate-500">{{ copy.pricingPreview.qualReviews }}</span>
                            <div class="mt-1 text-lg font-black">{{ pricingPreview.qualReviews }}</div>
                          </div>
                        </div>
                      </div>
                      <div v-else class="mt-3 rounded-xl border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
                        {{ copy.pricingPreview.invalidJson }}
                      </div>
                    </div>
                  </div>

                  <div v-if="pricingPreview" class="mt-4 grid gap-4">
                    <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white">
                      <div class="border-b border-slate-200 px-4 py-3 font-black">{{ copy.pricingPreview.unitsTitle }}</div>
                      <div v-if="pricingPreview.units.length" class="divide-y divide-slate-100">
                        <div v-for="unit in pricingPreview.units" :key="unit.unitId" class="grid gap-3 p-4 xl:grid-cols-[240px_1fr]">
                          <div>
                            <div class="text-xs font-black uppercase text-slate-400">{{ copy.pricingPreview.unitId }}</div>
                            <div class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ unit.unitId }}</div>
                          </div>
                          <div v-if="unit.prices.length" class="grid gap-3 lg:grid-cols-3">
                            <div v-for="price in unit.prices" :key="`${unit.unitId}-${price.label}`" class="rounded-xl bg-slate-50 p-3">
                              <div class="font-black text-slate-900">{{ price.label }}</div>
                              <div class="mt-2 text-xs font-bold text-slate-500">{{ copy.pricingPreview.priceId }}</div>
                              <div class="break-all font-mono text-xs text-blue-700">{{ price.priceId }}</div>
                              <div class="mt-2 text-xs font-bold text-slate-500">{{ copy.pricingPreview.productId }}</div>
                              <div class="break-all font-mono text-xs text-slate-600">{{ price.productId }}</div>
                            </div>
                          </div>
                          <div v-else class="rounded-xl border border-dashed border-slate-200 p-4 text-sm text-slate-500">
                            {{ copy.pricingPreview.notConfigured }}
                          </div>
                        </div>
                      </div>
                      <div v-else class="p-5 text-center text-sm text-slate-500">{{ copy.pricingPreview.notConfigured }}</div>
                    </div>

                    <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white">
                      <div class="border-b border-slate-200 px-4 py-3 font-black">{{ copy.pricingPreview.unlocksTitle }}</div>
                      <div v-if="pricingPreview.unlocks.length" class="divide-y divide-slate-100">
                        <div v-for="unlock in pricingPreview.unlocks" :key="unlock.targetId" class="grid gap-3 p-4 md:grid-cols-3">
                          <div>
                            <div class="text-xs font-black uppercase text-slate-400">{{ copy.pricingPreview.unlockTarget }}</div>
                            <div class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ unlock.targetId }}</div>
                          </div>
                          <div>
                            <div class="text-xs font-black uppercase text-slate-400">{{ copy.pricingPreview.priceId }}</div>
                            <div class="mt-1 break-all font-mono text-xs text-blue-700">{{ unlock.priceId }}</div>
                          </div>
                          <div>
                            <div class="text-xs font-black uppercase text-slate-400">{{ copy.pricingPreview.productId }}</div>
                            <div class="mt-1 break-all font-mono text-xs text-slate-600">{{ unlock.productId }}</div>
                          </div>
                        </div>
                      </div>
                      <div v-else class="p-5 text-center text-sm text-slate-500">{{ copy.pricingPreview.notConfigured }}</div>
                    </div>
                  </div>
                </section>
                <div class="grid gap-4 xl:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.itemsJson }}
                    <textarea v-model="form.items_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.pricingJson }}
                    <textarea v-model="form.pricing_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="savePricing">
                    <Send class="h-4 w-4" />
                    {{ copy.savePricing }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'schema'" class="space-y-4">
                <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="loadSchemas">
                  <FileJson class="h-4 w-4" />
                  {{ copy.loadSchema }}
                </button>
                <JsonPreview
                  v-if="schemas"
                  :title="copy.schemaJson"
                  :text="schemasJson"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                  max-height="620px"
                />
                <div v-else class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">{{ copy.emptySchema }}</div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-5">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <h3 class="font-black">{{ copy.actionsTitle }}</h3>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.actionsDescription }}</p>
                  <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="duplicateBundle">
                      <Loader2 v-if="duplicating" class="h-4 w-4 animate-spin" />
                      <Copy v-else class="h-4 w-4" />
                      {{ duplicating ? copy.duplicating : copy.duplicateDraft }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="publish">
                      <Loader2 v-if="publishing" class="h-4 w-4 animate-spin" />
                      {{ publishing ? copy.publishing : copy.publish }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="deprecate">
                      <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
                      {{ deprecating ? copy.deprecating : copy.deprecate }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 text-sm font-bold text-white shadow-sm shadow-red-200 transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="showDeleteConfirm = true">
                      <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
                      <Trash2 v-else class="h-4 w-4" />
                      {{ deleting ? copy.deleting : copy.delete }}
                    </button>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.rawReadonlyHint }}
                </div>
                <JsonPreview
                  :title="copy.rawJson"
                  :text="selectedJson"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                  max-height="620px"
                />
              </div>
            </main>
          </div>
        </template>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showDeleteConfirm" class="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/50 p-6">
        <div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
          <h2 class="text-2xl font-black">{{ copy.deleteConfirmTitle }}</h2>
          <p class="mt-3 text-sm text-slate-600">{{ copy.deleteConfirmDescription }}</p>
          <div class="mt-5 rounded-2xl bg-slate-50 p-4">
            <div class="font-black">{{ bundleName(selected) }}</div>
            <div class="mt-1 break-all text-xs text-slate-500">{{ selectedId }}</div>
          </div>
          <div class="mt-6 flex justify-end gap-3">
            <button class="rounded-xl border px-5 py-3 font-bold disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="showDeleteConfirm = false">{{ copy.cancel }}</button>
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="removeBundle">
              <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
              {{ deleting ? copy.deleting : copy.confirmDelete }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>
