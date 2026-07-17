<script setup lang="ts">
import { Boxes, Copy, Loader2, Plus, RefreshCw, Save, Send, Trash2, UploadCloud, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"

import PricingEditor from "@/components/PricingEditor.vue"
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

type DetailTab = "summary" | "meta" | "pricing" | "schema" | "actions"
type Mode = "detail" | "create"
type BundleItemType = "pipeline" | "membership"
type CreateBundleItem = {
  key: number
  item_type: BundleItemType
  ref_ulid: string
}
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
type PricingSelectOption = {
  id: string
  label: string
  subtitle?: string
  durationMonths?: number
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
const createItems = ref<CreateBundleItem[]>([])
const pipelineOptions = ref<JsonRecord[]>([])
const membershipOptions = ref<JsonRecord[]>([])
const linkedUnitOptions = ref<PricingSelectOption[]>([])
const targetOptionsLoading = ref(false)
const pricingTargetsLoading = ref(false)
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const deprecating = ref(false)
const deleting = ref(false)
const duplicating = ref(false)
const detailOpen = ref(false)
const statusFilter = ref("")
const offset = ref(0)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)

const activeTab = ref<DetailTab>("summary")
const mode = ref<Mode>("detail")
const showDeleteConfirm = ref(false)
const replacementPipelineId = ref("")
const limit = 10
let nextCreateItemKey = 1
const { t } = useAdminLanguage()
const copy = computed(() => t.value.bundlesAdmin)

const currentPage = computed(() => Math.floor(offset.value / limit) + 1)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / limit)))
const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => hasMore.value)
const statusActionBusy = computed(() => publishing.value || deprecating.value || deleting.value || duplicating.value)
const selectedId = computed(() => selected.value ? bundleUlid(selected.value) : "")
const selectedStatusKey = computed(() => bundleStatusKey(selected.value))
const canCloneSelectedBundle = computed(() => !!selectedId.value)
const canPublishSelectedBundle = computed(() => !!selectedId.value && selectedStatusKey.value === "draft")
const canDeprecateSelectedBundle = computed(() => !!selectedId.value && selectedStatusKey.value === "published")
const canDeleteSelectedBundle = computed(() => !!selectedId.value && selectedStatusKey.value === "draft")
const statusActionCount = computed(() => [
  canCloneSelectedBundle.value,
  canPublishSelectedBundle.value,
  canDeprecateSelectedBundle.value,
  canDeleteSelectedBundle.value,
].filter(Boolean).length)

const createItemTypeOptions = computed(() => [
  { value: "pipeline" as const, label: copy.value.createItemTypes.pipeline },
  { value: "membership" as const, label: copy.value.createItemTypes.membership },
])
const replacementPipelineOptions = computed<TargetOption[]>(() => pipelineOptions.value.map(pipelineTargetOption).filter(hasTargetId))
const createItemsJson = computed(() => {
  return JSON.stringify(createItems.value.map((item) => ({
    item_type: item.item_type,
    ref_ulid: item.ref_ulid.trim(),
  })), null, 2)
})
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "meta" as const, title: copy.value.tabs.meta, count: 1 },
  { key: "pricing" as const, title: copy.value.tabs.pricing, count: 2 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: statusActionCount.value },
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
const currentMembershipRefs = computed(() => membershipRefsFromItemsJson(form.value.items_json))
const linkedPipelinePricingOptions = computed<PricingSelectOption[]>(() => currentPipelineRefs.value.map((id) => {
  const target = pipelineOptions.value.find((item) => pipelineTargetOption(item).id === id)
  const option = target ? pipelineTargetOption(target) : null
  return {
    id,
    label: option?.title || id,
    subtitle: option?.subtitle || id,
  }
}))
const linkedMembershipPricingOptions = computed<PricingSelectOption[]>(() => currentMembershipRefs.value.map((id) => {
  const target = membershipOptions.value.find((item) => membershipTargetOption(item).id === id)
  const option = target ? membershipTargetOption(target) : null
  return {
    id,
    label: option?.title || id,
    subtitle: option?.subtitle || id,
    durationMonths: Number(pickFirst(target || {}, ["duration_in_months", "duration_months"])) || 0,
  }
}))
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
    .filter(([key]) => !["version", "items_json", "pricing_json"].includes(key))
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

function bundleStatusKey(bundle: JsonRecord | null | undefined) {
  const normalized = String(bundleStatus(bundle) || "").trim().toLowerCase()
  if (normalized === "active" || normalized === "published") return "published"
  if (normalized === "deprecated" || normalized === "deprecate") return "deprecated"
  if (normalized === "draft" || normalized === "inactive") return "draft"
  return normalized || "draft"
}

function bundleStatusLabel(bundle: JsonRecord | null | undefined) {
  const key = bundleStatusKey(bundle)
  return copy.value.statusLabels[key as keyof typeof copy.value.statusLabels] || String(bundleStatus(bundle) || "-")
}

function bundleStatusBadgeValue(bundle: JsonRecord | null | undefined) {
  const key = bundleStatusKey(bundle)
  if (key === "published") return "COMPLETED"
  if (key === "deprecated") return "DEPRECATED"
  return "PENDING"
}

function normalizeItemType(value: unknown) {
  return String(value || "").trim().toLowerCase().replace(/-/g, "_")
}

function linkedItemTypeLabel(value: unknown) {
  const type = normalizeItemType(value)
  if (type.includes("pipeline")) return copy.value.createItemTypes.pipeline
  if (type.includes("membership")) return copy.value.createItemTypes.membership
  return String(value || "-")
}

function itemReference(record: JsonRecord) {
  return String(pickFirst(record, ["ref_ulid", "ref_id", "ulid", "id", "item_id", "pipeline_id", "pipeline_cc_ulid", "membership_id"]) || "").trim()
}

function isPipelineItem(record: JsonRecord) {
  const type = normalizeItemType(pickFirst(record, ["item_type", "type", "itemType", "kind"]))
  return type.includes("pipeline") || !!String(pickFirst(record, ["pipeline_id", "pipeline_cc_ulid"]) || "").trim()
}

function isMembershipItem(record: JsonRecord) {
  const type = normalizeItemType(pickFirst(record, ["item_type", "type", "itemType", "kind"]))
  return type.includes("membership") || !!String(pickFirst(record, ["membership_id", "membership_ulid"]) || "").trim()
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

function membershipRefsFromItemsJson(value: string) {
  const parsed = parseJsonSilently(value)
  const items = Array.isArray(parsed) ? parsed : parsed && typeof parsed === "object" ? [parsed] : []
  const refs = new Set<string>()
  for (const item of items) {
    const record = asRecord(item)
    if (!record || !isMembershipItem(record)) continue
    const ref = itemReference(record)
    if (ref) refs.add(ref)
  }
  return Array.from(refs)
}

function pipelineConfigRecord(value: unknown) {
  const record = asRecord(value)
  if (!record) return null
  const pipeline = asRecord(record.pipeline)
  if (pipeline) return asRecord(pipeline.config) || pipeline
  return asRecord(record.config) || record
}

function pipelineUnitIds(value: unknown) {
  const config = pipelineConfigRecord(value)
  const stages = Array.isArray(config?.stages) ? config.stages : []
  const unitIds: string[] = []
  for (const stage of stages) {
    const stageRecord = asRecord(stage)
    if (!Array.isArray(stageRecord?.units)) continue
    for (const item of stageRecord.units) {
      const unit = asRecord(item)
      const unitId = String(pickFirst(unit || {}, ["unit_ulid", "unit_id"]) || "").trim()
      if (unitId) unitIds.push(unitId)
    }
  }
  return unitIds
}

function bundleTargetSummary(bundle: JsonRecord | null | undefined) {
  const directPipelineId = String(pickFirst(bundle || {}, ["pipeline_id", "pipeline_cc_ulid"]) || "")
  const directMembershipId = String(pickFirst(bundle || {}, ["membership_id", "membership_ulid"]) || "")
  const directSummaries = [
    directPipelineId ? `${copy.value.createItemTypes.pipeline} · ${directPipelineId}` : "",
    directMembershipId ? `${copy.value.createItemTypes.membership} · ${directMembershipId}` : "",
  ].filter(Boolean)
  if (directSummaries.length) return directSummaries.join(" + ")

  const parsed = parseJsonSilently(String(bundle?.items_json || ""))
  const items = Array.isArray(parsed) ? parsed : parsed && typeof parsed === "object" ? [parsed] : []
  const summaries: string[] = []
  for (const item of items) {
    const record = asRecord(item)
    if (!record) continue
    const itemType = String(record.item_type || record.type || record.itemType || record.kind || "").toLowerCase()
    const ref = String(record.ref_ulid || record.item_id || record.id || record.pipeline_id || record.pipeline_cc_ulid || record.membership_id || "").trim()
    if (!ref) continue
    if (itemType.includes("pipeline")) summaries.push(`${copy.value.createItemTypes.pipeline} · ${ref}`)
    else if (itemType.includes("membership")) summaries.push(`${copy.value.createItemTypes.membership} · ${ref}`)
    else summaries.push(`${itemType || copy.value.fields.linkedTarget} · ${ref}`)
  }
  if (summaries.length <= 2) return summaries.join(" + ")
  return `${summaries.slice(0, 2).join(" + ")} + ${copy.value.moreLinkedItems(summaries.length - 2)}`
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

function createTargetOptions(itemType: BundleItemType) {
  if (itemType === "membership") return membershipOptions.value.map(membershipTargetOption).filter(hasTargetId)
  return pipelineOptions.value.map(pipelineTargetOption).filter(hasTargetId)
}

function selectedCreateTarget(item: CreateBundleItem) {
  return createTargetOptions(item.item_type).find((option) => option.id === item.ref_ulid) || null
}

function createBundleItem(itemType: BundleItemType = "pipeline"): CreateBundleItem {
  const item = {
    key: nextCreateItemKey,
    item_type: itemType,
    ref_ulid: "",
  }
  nextCreateItemKey += 1
  return item
}

function addCreateItem() {
  if (createItems.value.length >= 2) return
  const itemType: BundleItemType = createItems.value.some((item) => item.item_type === "pipeline") ? "membership" : "pipeline"
  createItems.value.push(createBundleItem(itemType))
}

function removeCreateItem(index: number) {
  if (createItems.value.length <= 1) return
  createItems.value.splice(index, 1)
}

function changeCreateItemType(item: CreateBundleItem) {
  item.ref_ulid = ""
}

function createItemTypeDisabled(item: CreateBundleItem, itemType: BundleItemType) {
  return createItems.value.some((candidate) => candidate.key !== item.key && candidate.item_type === itemType)
}

function isStructuredValue(value: unknown) {
  return Array.isArray(value) || (!!value && typeof value === "object")
}

function jsonText(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2)
}

function detailFieldText(key: string, value: unknown) {
  if (key === "status" || key === "raw_status") return bundleStatusLabel({ [key]: value } as JsonRecord)
  if (key.endsWith("_at")) return formatDate(String(value || "")) || "-"
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

  const seen = new Set<string>()
  let pipelineCount = 0
  let membershipCount = 0
  for (const [index, item] of parsed.entries()) {
    const record = asRecord(item)
    if (!record || isBlank(record.item_type) || isBlank(record.ref_ulid)) {
      toast.error(copy.value.toasts.itemRequired(index + 1))
      return false
    }
    const itemType = normalizeItemType(record.item_type)
    const ref = String(record.ref_ulid).trim()
    if (itemType !== "pipeline" && itemType !== "membership") {
      toast.error(copy.value.toasts.itemTypeUnsupported(index + 1))
      return false
    }
    if (itemType === "pipeline") pipelineCount += 1
    if (itemType === "membership") membershipCount += 1
    const key = `${itemType}:${ref}`
    if (seen.has(key)) {
      toast.error(copy.value.toasts.itemDuplicate(index + 1))
      return false
    }
    seen.add(key)
  }
  if (pipelineCount > 1) {
    toast.error(copy.value.toasts.pipelineLimit)
    return false
  }
  if (membershipCount > 1) {
    toast.error(copy.value.toasts.membershipLimit)
    return false
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
      if (isBlank(targetId)) {
        toast.error(copy.value.toasts.unlockTargetRequired)
        return false
      }
      if (!currentPipelineRefs.value.includes(targetId)) {
        toast.error(copy.value.toasts.unlockTargetNotLinked(targetId))
        return false
      }
      if (!validatePriceObject(value, copy.value.toasts.unlockPriceLabel(targetId))) return false
    }
  }

  if (Array.isArray(pricing.memberships)) {
    if (pricing.memberships.length > 1) {
      toast.error(copy.value.toasts.membershipPricingLimit)
      return false
    }
    for (const [index, value] of pricing.memberships.entries()) {
      const membership = asRecord(value)
      if (!membership || isBlank(membership.membership_id) || Number(membership.duration_months) < 1) {
        toast.error(copy.value.toasts.membershipPricingRequired(index + 1))
        return false
      }
      if (!currentMembershipRefs.value.includes(String(membership.membership_id).trim())) {
        toast.error(copy.value.toasts.membershipPricingNotLinked(index + 1))
        return false
      }
      if (!validatePriceObject(membership, copy.value.toasts.membershipPriceLabel(index + 1))) return false
    }
  }

  if (Array.isArray(pricing.qual_reviews)) {
    for (const [index, value] of pricing.qual_reviews.entries()) {
      const review = asRecord(value)
      if (!review || isBlank(review.qual_id)) {
        toast.error(copy.value.toasts.qualReviewRequired(index + 1))
        return false
      }
      if (!validatePriceObject(review, copy.value.toasts.qualReviewPriceLabel(index + 1))) return false
    }
  }

  return true
}

function validateStructureJson() {
  return validateItemsJson() && validatePricingJson()
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
      page_size: String(limit),
    })

    let cursor = ""

    if (currentPage.value > lastPage.value) {

      cursor = nextCursor.value

    } else if (currentPage.value < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mall/bundles?${params}`)
    total.value = Number(data.total) || 0
    const list = Array.isArray(data.bundles) ? data.bundles : []

    bundles.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || 0)
    const isBackward = currentPage.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = currentPage.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = currentPage.value
    if (!selected.value && bundles.value.length) {
      await selectBundle(bundles.value[0], false)
    }
  } catch (err) {
    console.error(err)
    bundles.value = []
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

function prevPage() {
  offset.value = Math.max(0, offset.value - limit)
}

function nextPage() {
  if (!canNext.value) return
  offset.value += limit
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
    const [pipelines, memberships] = await Promise.all([
      loadRecords("pipelines", apiClient<JsonRecord>("/api/pipelines?page_size=100&only_current=true"), ["pipelines", "items"]),
      loadRecords("memberships", apiClient<JsonRecord>("/api/memberships?page=1&page_size=100"), ["memberships", "membership_configs", "items"]),
    ])
    pipelineOptions.value = pipelines.filter(targetUsable)
    membershipOptions.value = memberships.filter(targetUsable)
  } finally {
    if (failures.length) {
      console.warn("Some product target options failed to load", failures)
      toast.error(copy.value.toasts.targetOptionsLoadFailed)
    }
    targetOptionsLoading.value = false
  }
}

async function loadPricingTargets() {
  const pipelineRefs = currentPipelineRefs.value
  linkedUnitOptions.value = []
  if (!pipelineRefs.length) return

  pricingTargetsLoading.value = true
  try {
    const results = await Promise.allSettled(
      pipelineRefs.map((pipelineId) => apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(pipelineId)}`)),
    )
    const options: PricingSelectOption[] = []
    results.forEach((result, index) => {
      if (result.status !== "fulfilled") {
        console.error("Failed to load linked pipeline pricing targets", result.reason)
        return
      }
      const pipelineId = pipelineRefs[index]
      const pipelineTarget = pipelineOptions.value.find((item) => pipelineTargetOption(item).id === pipelineId)
      const pipelineLabel = pipelineTarget ? pipelineTargetOption(pipelineTarget).title : pipelineId
      for (const unitId of pipelineUnitIds(result.value)) {
        options.push({
          id: unitId,
          label: `${pipelineLabel} · ${unitId}`,
          subtitle: pipelineId,
        })
      }
    })
    linkedUnitOptions.value = options
  } finally {
    pricingTargetsLoading.value = false
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
  createItems.value = [createBundleItem()]
  form.value = { ...emptyForm }
  void loadCreateTargetOptions()
}

function closeDetail() {
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

const uploadingThumbnail = ref(false)
const thumbnailFileInput = ref<HTMLInputElement | null>(null)

watch(() => form.value.name, (newName, oldName) => {
  if (mode.value !== "create") return
  const generatePath = (t: string) => t ? `/bundle/${t.trim().toLowerCase().replace(/\s+/g, '_')}` : ''
  const newPath = generatePath(newName)
  const oldPath = generatePath(oldName)
  
  if (!form.value.bundle_gpath || form.value.bundle_gpath === oldPath) {
    form.value.bundle_gpath = newPath
  }
})

async function handleThumbnailUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!form.value.bundle_ulid) {
    if (mode.value === "create") {
      await createBundle()
      if (!form.value.bundle_ulid) {
        if (thumbnailFileInput.value) thumbnailFileInput.value.value = ""
        return
      }
    } else {
      toast.error("状态错误，请先刷新页面")
      if (thumbnailFileInput.value) thumbnailFileInput.value.value = ""
      return
    }
  }

  uploadingThumbnail.value = true
  try {
    const arrayBuffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')

    const uploadUrlReq = {
      bundle_ulid: form.value.bundle_ulid,
      file_name: file.name,
      content_type: file.type || "application/octet-stream",
      file_hash: hashHex,
    }
    const uploadUrlRes = await apiClient<JsonRecord>(`/api/mall/bundles/upload-url`, {
      method: "POST",
      body: JSON.stringify(uploadUrlReq),
    })

    const uploadUrl = String(uploadUrlRes.upload_url || "")
    const objectKey = String(uploadUrlRes.object_key || "")
    if (!uploadUrl || !objectKey) throw new Error("Invalid upload URL response")

    const uploadRes = await fetch(uploadUrl, {
      method: "PUT",
      body: file,
      headers: uploadUrlRes.signed_headers as Record<string, string> || {},
    })

    if (!uploadRes.ok) {
      throw new Error(`Upload failed with status ${uploadRes.status}`)
    }

    form.value.thumbnail_object_key = objectKey
    form.value.thumbnail_file_hash = hashHex
    toast.success("上传成功")
  } catch (err: any) {
    console.error(err)
    toast.error(apiErrorMessage(err, "上传失败"))
  } finally {
    uploadingThumbnail.value = false
    if (thumbnailFileInput.value) thumbnailFileInput.value.value = ""
  }
}

async function createBundle() {
  form.value.items_json = createItemsJson.value
  if (!form.value.bundle_gpath.trim() || !form.value.name.trim()) {
    toast.error(copy.value.toasts.createRequired)
    return
  }
  if (!createItems.value.length || createItems.value.some((item) => !item.ref_ulid.trim())) {
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

async function persistPricing(showToast = true) {
  if (!selectedId.value) return
  saving.value = true
  try {
    const bundleId = selectedId.value
    const data = await apiClient<JsonRecord>("/api/mall/bundles/pricing", {
      method: "PUT",
      body: JSON.stringify({
        bundle_ulid: bundleId,
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
      }),
    })
    const actualBundle = (data.bundle && typeof data.bundle === "object" ? data.bundle : data) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
    await syncBundleDisplayPricing(bundleId)
    await load()
    await refreshSelectedBundleDetail(bundleId)
    if (showToast) toast.success(copy.value.toasts.pricingSaved)
    return true
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
    return false
  } finally {
    saving.value = false
  }
}

async function savePricing() {
  if (!selectedId.value) return
  if (!validateStructureJson()) return
  await persistPricing()
}

async function duplicateBundle() {
  if (!selected.value || !canCloneSelectedBundle.value || statusActionBusy.value) return
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

async function replacePipelineBindingInForm() {
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

  saving.value = true
  try {
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
      pricingRecord.unlocks = unlocks
      changed = true
    }

    if (Array.isArray(pricingRecord.units) && pricingRecord.units.length > 0) {
      try {
        const [oldRes, newRes] = await Promise.all([
          apiClient<JsonRecord>(`/api/pipelines/${fromId}`),
          apiClient<JsonRecord>(`/api/pipelines/${toId}`)
        ])
        const oldUnits = pipelineUnitIds(oldRes)
        const newUnits = pipelineUnitIds(newRes)
        
        if (oldUnits.length > 0 && newUnits.length > 0) {
          const mapping = new Map<string, string>()
          for (let i = 0; i < Math.min(oldUnits.length, newUnits.length); i++) {
            mapping.set(oldUnits[i], newUnits[i])
          }
          for (const value of pricingRecord.units) {
            const u = asRecord(value)
            if (u && typeof u.unit_id === "string" && mapping.has(u.unit_id)) {
              u.unit_id = mapping.get(u.unit_id)
              changed = true
            }
          }
        } else {
          toast.error(copy.value.toasts.relinkUnitMapFailed)
          return false
        }
      } catch (err) {
        console.error("Failed to auto-map units:", err)
        toast.error(copy.value.toasts.relinkUnitMapFailed)
        return false
      }
    }

    if (!changed) {
      toast.error(copy.value.toasts.relinkNoChange)
      return false
    }

    form.value.items_json = JSON.stringify(items, null, 2)
    form.value.pricing_json = JSON.stringify(pricingRecord, null, 2)
    return true
  } finally {
    saving.value = false
  }
}

async function replaceAndSavePipelineBinding() {
  if (!await replacePipelineBindingInForm()) return
  await savePricing()
}

async function publish() {
  if (!canPublishSelectedBundle.value || statusActionBusy.value) return
  if (!validateStructureJson()) return
  publishing.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/publish`, { method: "POST" })
    toast.success(copy.value.toasts.published)
    await load()
    await refreshSelectedBundleDetail()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.publishFailed))
  } finally {
    publishing.value = false
  }
}

async function deprecate() {
  if (!canDeprecateSelectedBundle.value || statusActionBusy.value) return
  deprecating.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/deprecate`, { method: "POST" })
    toast.success(copy.value.toasts.deprecated)
    await load()
    await refreshSelectedBundleDetail()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deprecateFailed))
  } finally {
    deprecating.value = false
  }
}

async function removeBundle() {
  if (!canDeleteSelectedBundle.value || statusActionBusy.value) {
    showDeleteConfirm.value = false
    return
  }
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

async function refreshSelectedBundleDetail(bundleId = selectedId.value) {
  if (!bundleId) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/mall/bundles/${encodeURIComponent(bundleId)}`)
    const actualBundle = (detail.bundle && typeof detail.bundle === "object" ? detail.bundle : detail) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
  } catch (err) {
    console.error(err)
  }
}

async function syncBundleDisplayPricing(bundleId = selectedId.value) {
  await apiClient("/api/mall/bundles/sync-display-pricing", {
    method: "POST",
    body: JSON.stringify({ bundle_ulid: bundleId || undefined }),
  })
}

async function syncDisplayPricing() {
  await syncBundleDisplayPricing()
  toast.success(copy.value.toasts.displayPricingSynced)
  await load()
  await refreshSelectedBundleDetail()
}



watch(statusFilter, () => {
  selected.value = null
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
  if (offset.value !== 0) {
    offset.value = 0
    return
  }
  void load()
})
watch(offset, () => {
  selected.value = null
  void load()
})
watch(createItemsJson, (value) => {
  if (mode.value === "create") form.value.items_json = value
})
watch(activeTab, (tab) => {
  if (tab !== "pricing") return
  void loadCreateTargetOptions().then(loadPricingTargets)
})
watch(() => form.value.items_json, () => {
  if (mode.value === "detail" && activeTab.value === "pricing") void loadPricingTargets()
})
onMounted(load)
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-primary" type="button" @click="newBundle">
          <Plus class="h-4 w-4" />
          {{ copy.newBundle }}
        </button>
        <button class="ui-btn ui-btn-secondary" type="button" @click="syncDisplayPricing">
          <RefreshCw class="h-4 w-4 text-slate-400" />
          {{ copy.syncDisplayPricing }}
        </button>
        <button class="ui-btn ui-btn-secondary" type="button" @click="load">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
          <div class="flex min-w-0 flex-wrap items-center gap-3">
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
          </div>
          <select v-model="statusFilter" class="ui-input md:w-64">
            <option value="">{{ copy.allStatus }}</option>
            <option value="Draft">{{ copy.statusOptions.Draft }}</option>
            <option value="Active">{{ copy.statusOptions.Active }}</option>
            <option value="Deprecated">{{ copy.statusOptions.Deprecated }}</option>
          </select>
        </div>
        <div v-if="loading" class="divide-y divide-hairline">
          <div
            v-for="i in 5"
            :key="`sk-${i}`"
            class="grid grid-cols-1 gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] md:items-center md:gap-4 md:px-6 md:py-3.5"
          >
            <div class="space-y-2">
              <div class="skeleton h-4 w-2/5" />
              <div class="skeleton h-3.5 w-4/5" />
            </div>
            <div class="skeleton h-5 w-16 rounded-full md:justify-self-center" />
            <div class="skeleton hidden h-4 w-10 md:block md:justify-self-center" />
            <div class="skeleton hidden h-4 w-20 md:block md:justify-self-end" />
            <div class="skeleton hidden h-4 w-14 md:block md:justify-self-end" />
          </div>
        </div>
        <template v-else>
          <div class="ui-table-head hidden grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 px-4 py-3 md:grid md:px-6">
            <span>{{ copy.columns.bundle }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-center">{{ copy.columns.version }}</span>
            <span class="text-right">{{ copy.columns.updatedAt }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <div class="divide-y divide-hairline">
            <div
              v-for="bundle in bundles"
              :key="bundleUlid(bundle)"
              class="ui-table-row flex w-full cursor-pointer flex-col gap-3 px-4 py-4 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/20 md:grid md:grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] md:gap-4 md:px-6 md:py-3.5"
              :class="mode === 'detail' && selectedId === bundleUlid(bundle) ? 'bg-primary-50/60' : ''"
              role="button"
              tabindex="0"
              @click="selectBundle(bundle)"
              @keydown.enter.prevent="selectBundle(bundle)"
              @keydown.space.prevent="selectBundle(bundle)"
            >
              <div class="min-w-0">
                <div class="break-words font-medium text-ink md:truncate">{{ bundleName(bundle) }}</div>
                <div class="mt-0.5 line-clamp-1 text-sm text-slate-500">{{ bundle.description || copy.noDescription }}</div>
                <div class="mt-2 flex flex-wrap gap-2 text-xs font-medium text-slate-600">
                  <span class="tnum max-w-full break-all rounded-full bg-slate-100 px-2.5 py-1">{{ copy.displayPricePrefix }}{{ displayPrice(bundle) }}</span>
                  <span v-if="bundleTargetSummary(bundle)" class="tnum max-w-full break-all rounded-full bg-slate-100 px-2.5 py-1">{{ copy.linkedTargetPrefix }}{{ bundleTargetSummary(bundle) }}</span>
                  <span class="tnum max-w-full break-all rounded-full bg-slate-100 px-2.5 py-1">{{ copy.fields.idPrefix }}{{ bundleUlid(bundle) || "-" }}</span>
                </div>
              </div>
              <span class="flex items-center justify-between gap-3 md:block md:self-center md:justify-self-center">
                <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
                <span class="ui-badge" :class="badgeClass(bundleStatusBadgeValue(bundle))">{{ bundleStatusLabel(bundle) }}</span>
              </span>
              <span class="flex items-center justify-between gap-3 md:block md:self-center md:text-center">
                <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.version }}</span>
                <span class="tnum text-sm text-slate-600">{{ copy.fields.versionPrefix }} {{ bundle.version || 0 }}</span>
              </span>
              <span class="flex items-center justify-between gap-3 md:block md:self-center md:justify-self-end md:text-right">
                <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.updatedAt }}</span>
                <span class="tnum text-sm text-slate-500">{{ formatDate(String(bundle.updated_at || bundle.created_at || "")) }}</span>
              </span>
              <button
                class="inline-flex h-8 items-center justify-center rounded-lg text-sm font-medium text-primary-700 transition-colors hover:text-primary-600 md:self-center md:justify-self-end"
                type="button"
                @click.stop="selectBundle(bundle)"
              >
                {{ copy.viewDetails }}
              </button>
            </div>
          </div>
        </template>
        <div v-if="!loading && !bundles.length" class="ui-empty">
          <div class="ui-empty-icon"><Boxes class="h-6 w-6" /></div>
          <p class="ui-empty-title">{{ copy.empty }}</p>
        </div>
        <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
          <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(currentPage, totalPages, total) }}</span>
          <div class="flex flex-col gap-2 sm:flex-row">
            <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="prevPage">{{ copy.prev }}</button>
            <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="nextPage">{{ copy.next }}</button>
          </div>
        </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1280px]">
        <template v-if="mode === 'create'">
          <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ copy.createTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.createDescription }}</p>
            </div>
            <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="closeDetail">
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
            <section class="rounded-2xl border border-edge p-4 md:p-5">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="text-base font-semibold text-ink">{{ copy.createSections.linkTarget }}</h3>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.createSections.linkTargetDesc }}</p>
                </div>
                <div class="flex items-center gap-3">
                  <Loader2 v-if="targetOptionsLoading" class="h-4 w-4 animate-spin text-slate-400" />
                  <button class="ui-btn ui-btn-primary" type="button" :disabled="createItems.length >= 2" @click="addCreateItem">
                    <Plus class="h-4 w-4" />
                    {{ copy.addLinkedItem }}
                  </button>
                </div>
              </div>
              <div class="mt-4 grid gap-4">
                <div v-for="(item, index) in createItems" :key="item.key" class="rounded-2xl border border-edge bg-white p-4">
                  <div class="grid gap-4 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.4fr)_auto]">
                    <label class="grid gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.itemType }}</span>
                      <select v-model="item.item_type" class="ui-input" @change="changeCreateItemType(item)">
                        <option v-for="option in createItemTypeOptions" :key="option.value" :value="option.value" :disabled="createItemTypeDisabled(item, option.value)">
                          {{ option.label }}
                        </option>
                      </select>
                      <p class="text-xs text-slate-400">{{ copy.itemTypeHint }}</p>
                    </label>
                    <label class="grid gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.linkedTarget }}</span>
                      <select v-model="item.ref_ulid" class="ui-input" :disabled="targetOptionsLoading || !createTargetOptions(item.item_type).length">
                        <option value="" disabled>{{ targetOptionsLoading ? copy.loadingTargets : copy.selectLinkedTarget }}</option>
                        <option v-for="option in createTargetOptions(item.item_type)" :key="option.id" :value="option.id">{{ option.title }} · {{ option.subtitle }}</option>
                      </select>
                      <p class="text-xs text-slate-400">{{ createTargetOptions(item.item_type).length ? copy.linkedTargetHint : copy.noLinkedTargets }}</p>
                    </label>
                    <button class="mt-[26px] inline-flex h-10 w-10 items-center justify-center rounded-[10px] border border-slate-200 text-slate-400 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40" type="button" :disabled="createItems.length <= 1" :aria-label="copy.removeLinkedItem" @click="removeCreateItem(index)">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </div>
                  <div v-if="selectedCreateTarget(item)" class="mt-4 rounded-[10px] bg-primary-50 p-4">
                    <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.selectedTarget }}</div>
                    <div class="mt-2 text-sm font-semibold text-ink">{{ selectedCreateTarget(item)?.title }}</div>
                    <div class="tnum mt-1 break-all text-xs text-slate-500">{{ selectedCreateTarget(item)?.subtitle }}</div>
                  </div>
                </div>
              </div>
            </section>

            <section class="rounded-2xl border border-edge p-4 md:p-5">
              <h3 class="text-base font-semibold text-ink">{{ copy.createSections.basicInfo }}</h3>
              <p class="mt-1 text-sm text-slate-500">{{ copy.createSections.basicInfoDesc }}</p>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <label class="grid grid-rows-[auto_auto_1.25rem] gap-1.5 md:col-span-2">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.name }}</span>
                  <input v-model="form.name" class="ui-input" maxlength="160" :placeholder="copy.placeholders.name" />
                  <span aria-hidden="true"></span>
                </label>
                <label class="grid grid-rows-[auto_auto_1.25rem] gap-1.5">
                  <span class="ui-label">{{ copy.fields.thumbnailObjectKey }}</span>
                  <input v-model="form.thumbnail_object_key" class="ui-input" :placeholder="copy.placeholders.thumbnailObjectKey" />
                  <p class="text-xs text-slate-400">{{ copy.optionalImageHint }}</p>
                </label>
                <label class="grid grid-rows-[auto_auto_1.25rem] gap-1.5">
                  <span class="ui-label">{{ copy.fields.thumbnailFileHash }}</span>
                  <input v-model="form.thumbnail_file_hash" class="ui-input" :placeholder="copy.placeholders.thumbnailFileHash" />
                  <span aria-hidden="true"></span>
                </label>
                <div class="md:col-span-2 mt-2 flex gap-3">
                  <input type="file" ref="thumbnailFileInput" class="hidden" accept="image/*" @change="handleThumbnailUpload" />
                  <button type="button" class="ui-btn ui-btn-secondary w-full sm:w-auto" :disabled="uploadingThumbnail" @click="thumbnailFileInput?.click()">
                    <Loader2 v-if="uploadingThumbnail" class="h-4 w-4 animate-spin" />
                    <UploadCloud v-else class="h-4 w-4 text-slate-400" />
                    {{ uploadingThumbnail ? (copy as any).uploading || "上传中..." : (copy as any).uploadThumbnail || "上传封面图" }}
                  </button>
                </div>
                <label class="grid gap-1.5 md:col-span-2">
                  <span class="ui-label">{{ copy.fields.description }}</span>
                  <textarea v-model="form.description" class="min-h-24 w-full rounded-[10px] border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" maxlength="1200" :placeholder="copy.placeholders.description" />
                </label>
              </div>
            </section>

            <details class="rounded-2xl border border-edge bg-white p-4 md:p-5 mt-4">
              <summary class="cursor-pointer text-sm font-medium text-slate-700">{{ (copy as any).advancedConfig || "高级配置" }}</summary>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <label class="grid grid-rows-[auto_auto_1.25rem] gap-1.5 md:col-span-2">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.bundleGpath }}</span>
                  <input v-model="form.bundle_gpath" class="ui-input" :placeholder="copy.placeholders.bundleGpath" />
                  <p class="text-xs text-slate-400">{{ copy.bundleGpathHint }}</p>
                </label>
                <label class="grid gap-1.5 md:col-span-2 xl:col-span-1">
                  <span class="ui-label">{{ copy.fields.itemsJson }}</span>
                  <textarea :value="createItemsJson" readonly class="tnum min-h-[180px] w-full rounded-[10px] border border-slate-200 bg-slate-50 p-3 font-mono text-xs leading-6 text-slate-600 focus:outline-none" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label">{{ copy.fields.pricingJson }}</span>
                  <textarea v-model="form.pricing_json" class="tnum min-h-[180px] w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-xs leading-6 text-slate-900 transition-colors focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                  <p class="text-xs text-slate-400">{{ copy.pricingJsonCreateHint }}</p>
                </label>
              </div>
            </details>
          </div>
          <div class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
            <button class="ui-btn ui-btn-primary w-full min-w-[180px] sm:w-auto" type="button" :disabled="saving" @click="createBundle">
              <Plus class="h-4 w-4" />
              {{ copy.createDraft }}
            </button>
          </div>
        </template>

        <div v-else-if="!selected" class="flex items-start justify-between gap-4 p-4 md:p-6">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
          </div>
          <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <template v-else>
          <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ bundleName(selected) }}</h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">{{ selectedId }}</p>
              </div>
              <div class="flex items-center gap-3">
                <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>

          <div class="flex gap-5 overflow-x-auto border-b border-hairline px-4 md:px-6">
            <button
              v-for="tab in detailTabs"
              :key="tab.key"
              class="ui-tab"
              :class="activeTab === tab.key ? 'ui-tab-active' : ''"
              type="button"
              @click="activeTab = tab.key"
            >
              {{ tab.title }}
              <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden">
            <main class="h-full min-h-0 max-h-none min-w-0 overflow-y-auto p-4 md:h-[60vh] md:min-h-[360px] md:max-h-[620px] md:p-6">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl border border-edge p-4">
                    <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</div>
                    <div class="tnum mt-2 break-all text-sm font-medium text-ink">{{ field.value }}</div>
                  </div>
                </div>
                <div class="rounded-2xl border border-edge bg-white p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summary.description }}</div>
                  <p class="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-600">{{ form.description || "-" }}</p>
                </div>
                <div class="rounded-2xl border border-edge bg-white p-4">
                  <div class="mb-3 text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.pricingPreview.linkedItems }}</div>
                  <div v-if="linkedItemsPreview.length" class="grid gap-2 md:grid-cols-2">
                    <div v-for="item in linkedItemsPreview" :key="`${item.type}-${item.ref}`" class="rounded-[10px] bg-slate-50 p-3 text-sm">
                        <div class="font-medium text-slate-600">{{ copy.pricingPreview.itemType }}: {{ linkedItemTypeLabel(item.type) }}</div>
                      <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ copy.pricingPreview.itemRef }}: {{ item.ref }}</div>
                    </div>
                  </div>
                  <div v-else class="rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
                    {{ copy.pricingPreview.emptyJson }}
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in selectedFields" :key="field.key" class="grid gap-1.5" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                    <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                    <pre
                      v-if="isStructuredValue(field.value)"
                      class="tnum max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-[10px] border border-slate-200 bg-slate-50 px-3 py-2.5 font-mono text-xs leading-5 text-slate-600"
                    >{{ jsonText(field.value) }}</pre>
                    <div v-else class="tnum min-h-10 break-words rounded-[10px] border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm leading-5 text-slate-700">
                      {{ detailFieldText(field.key, field.value) }}
                    </div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'meta'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-1.5 md:col-span-2">
                    <span class="ui-label">{{ copy.fields.bundleUlid }}</span>
                    <input v-model="form.bundle_ulid" disabled class="ui-input tnum" />
                  </label>
                  <label class="grid gap-1.5 md:col-span-2">
                    <span class="ui-label">{{ copy.fields.name }}</span>
                    <input v-model="form.name" class="ui-input" maxlength="160" />
                  </label>
                  <label class="grid gap-1.5">
                    <span class="ui-label">{{ copy.fields.thumbnailObjectKey }}</span>
                    <input v-model="form.thumbnail_object_key" class="ui-input" />
                  </label>
                  <label class="grid gap-1.5 md:col-span-2">
                    <span class="ui-label">{{ copy.fields.description }}</span>
                    <textarea v-model="form.description" class="min-h-28 w-full rounded-[10px] border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" maxlength="1200" />
                  </label>
                  <label class="grid gap-1.5 md:col-span-2">
                    <span class="ui-label">{{ copy.fields.thumbnailFileHash }}</span>
                    <input v-model="form.thumbnail_file_hash" class="ui-input tnum" />
                  </label>
                  <div class="md:col-span-2 mt-2 flex gap-3">
                    <input type="file" ref="thumbnailFileInput" class="hidden" accept="image/*" @change="handleThumbnailUpload" />
                    <button type="button" class="ui-btn ui-btn-secondary w-full sm:w-auto" :disabled="uploadingThumbnail" @click="thumbnailFileInput?.click()">
                      <Loader2 v-if="uploadingThumbnail" class="h-4 w-4 animate-spin" />
                      <UploadCloud v-else class="h-4 w-4 text-slate-400" />
                      {{ uploadingThumbnail ? (copy as any).uploading || "上传中..." : (copy as any).uploadThumbnail || "上传封面图" }}
                    </button>
                  </div>
                </div>

                <details class="rounded-2xl border border-edge bg-white p-4 md:p-5 mt-6 mb-4">
                  <summary class="cursor-pointer text-sm font-medium text-slate-700">{{ (copy as any).advancedConfig || "高级配置" }}</summary>
                  <div class="mt-4 grid gap-4">
                    <label class="grid gap-1.5">
                      <span class="ui-label">{{ copy.fields.bundleGpath }}</span>
                      <input v-model="form.bundle_gpath" disabled class="ui-input" />
                    </label>
                  </div>
                </details>

                <div class="flex justify-stretch sm:justify-end">
                  <button class="ui-btn ui-btn-primary w-full sm:w-auto" type="button" :disabled="saving" @click="saveMeta">
                    <Save class="h-4 w-4" />
                    {{ copy.saveMeta }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'pricing'" class="space-y-5">
                <section class="rounded-2xl bg-primary-50 p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div>
                      <h3 class="text-base font-semibold text-ink">{{ copy.relink.title }}</h3>
                      <p class="mt-1 text-sm text-slate-600">{{ copy.relink.description }}</p>
                    </div>
                    <button class="ui-btn ui-btn-secondary" type="button" :disabled="targetOptionsLoading" @click="loadCreateTargetOptions">
                      <Loader2 v-if="targetOptionsLoading" class="h-4 w-4 animate-spin text-slate-400" />
                      {{ copy.relink.reloadTargets }}
                    </button>
                  </div>
                  <div class="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_auto]">
                    <div class="rounded-[10px] bg-white p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.relink.currentPipeline }}</div>
                      <div v-if="currentPipelineRef" class="tnum mt-2 break-all font-mono text-xs text-primary-700">{{ currentPipelineRef }}</div>
                      <div v-else class="mt-2 text-sm text-slate-500">{{ copy.relink.noPipelineBinding }}</div>
                    </div>
                    <label class="grid gap-1.5">
                      <span class="ui-label">{{ copy.relink.newPipeline }}</span>
                      <select v-model="replacementPipelineId" class="ui-input" :disabled="!currentPipelineRef || targetOptionsLoading">
                        <option value="">{{ copy.relink.selectNewPipeline }}</option>
                        <option v-for="option in replacementPipelineOptions" :key="option.id" :value="option.id">
                          {{ option.title }} · {{ option.subtitle }}
                        </option>
                      </select>
                    </label>
                    <div class="flex flex-wrap items-end">
                      <button class="ui-btn ui-btn-primary" type="button" :disabled="!currentPipelineRef || !replacementPipelineId || saving" @click="replaceAndSavePipelineBinding">
                        <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                        {{ copy.relink.save }}
                      </button>
                    </div>
                  </div>
                </section>
                <section class="rounded-2xl border border-edge p-4 md:p-5">
                  <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h3 class="text-base font-semibold text-ink">{{ copy.pricingEditorTitle }}</h3>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.pricingEditorDescription }}</p>
                    </div>
                    <div v-if="pricingTargetsLoading" class="inline-flex items-center gap-2 text-xs text-slate-400">
                      <Loader2 class="h-4 w-4 animate-spin" />
                      {{ copy.loadingPricingTargets }}
                    </div>
                  </div>
                  <PricingEditor
                    v-model="form.pricing_json"
                    :unit-options="linkedUnitOptions"
                    :pipeline-options="linkedPipelinePricingOptions"
                    :membership-options="linkedMembershipPricingOptions"
                  />
                </section>
                <section class="rounded-2xl border border-edge p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h3 class="text-base font-semibold text-ink">{{ copy.pricingPreview.title }}</h3>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.pricingPreview.description }}</p>
                    </div>
                    <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">
                      {{ copy.summary.displayPrice }}: {{ selected ? displayPrice(selected) : "-" }}
                    </span>
                  </div>

                  <div class="mt-5 grid gap-4 xl:grid-cols-2">
                    <div class="rounded-2xl border border-edge bg-white p-4">
                      <h4 class="text-sm font-semibold text-ink">{{ copy.pricingPreview.linkedItems }}</h4>
                      <div v-if="linkedItemsPreview.length" class="mt-3 grid gap-2">
                        <div v-for="item in linkedItemsPreview" :key="`${item.type}-${item.ref}`" class="rounded-[10px] bg-slate-50 p-3 text-sm">
                            <div class="font-medium text-slate-600">{{ copy.pricingPreview.itemType }}: {{ linkedItemTypeLabel(item.type) }}</div>
                          <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ copy.pricingPreview.itemRef }}: {{ item.ref }}</div>
                        </div>
                      </div>
                      <div v-else class="mt-3 rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
                        {{ copy.pricingPreview.emptyJson }}
                      </div>
                    </div>

                    <div class="rounded-2xl border border-edge bg-white p-4">
                      <h4 class="text-sm font-semibold text-ink">{{ copy.pricingPreview.otherConfig }}</h4>
                      <div v-if="pricingPreview" class="mt-3 grid gap-2 text-sm">
                        <div class="rounded-[10px] bg-slate-50 p-3">
                          <span class="font-medium text-slate-500">{{ copy.pricingPreview.packageCoupon }}</span>
                          <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ pricingPreview.packageCoupon }}</div>
                        </div>
                        <div class="grid gap-2 sm:grid-cols-2">
                          <div class="rounded-[10px] bg-slate-50 p-3">
                            <span class="font-medium text-slate-500">{{ copy.pricingPreview.memberships }}</span>
                            <div class="tnum mt-1 text-lg font-semibold text-ink">{{ pricingPreview.memberships }}</div>
                          </div>
                          <div class="rounded-[10px] bg-slate-50 p-3">
                            <span class="font-medium text-slate-500">{{ copy.pricingPreview.qualReviews }}</span>
                            <div class="tnum mt-1 text-lg font-semibold text-ink">{{ pricingPreview.qualReviews }}</div>
                          </div>
                        </div>
                      </div>
                      <div v-else class="mt-3 rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
                        {{ copy.pricingPreview.invalidJson }}
                      </div>
                    </div>
                  </div>

                  <div v-if="pricingPreview" class="mt-4 grid gap-4">
                    <div class="overflow-hidden rounded-2xl border border-edge bg-white">
                      <div class="border-b border-hairline px-4 py-3 text-sm font-semibold text-ink">{{ copy.pricingPreview.unitsTitle }}</div>
                      <div v-if="pricingPreview.units.length" class="divide-y divide-hairline">
                        <div v-for="unit in pricingPreview.units" :key="unit.unitId" class="grid gap-3 p-4 xl:grid-cols-[240px_1fr]">
                          <div>
                            <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.pricingPreview.unitId }}</div>
                            <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ unit.unitId }}</div>
                          </div>
                          <div v-if="unit.prices.length" class="grid gap-3 lg:grid-cols-3">
                            <div v-for="price in unit.prices" :key="`${unit.unitId}-${price.label}`" class="rounded-[10px] bg-slate-50 p-3">
                              <div class="text-sm font-medium text-ink">{{ price.label }}</div>
                              <div class="mt-2 text-xs font-medium text-slate-500">{{ copy.pricingPreview.priceId }}</div>
                              <div class="tnum break-all font-mono text-xs text-primary-700">{{ price.priceId }}</div>
                              <div class="mt-2 text-xs font-medium text-slate-500">{{ copy.pricingPreview.productId }}</div>
                              <div class="tnum break-all font-mono text-xs text-slate-600">{{ price.productId }}</div>
                            </div>
                          </div>
                          <div v-else class="rounded-[10px] border border-dashed border-slate-200 p-4 text-sm text-slate-500">
                            {{ copy.pricingPreview.notConfigured }}
                          </div>
                        </div>
                      </div>
                      <div v-else class="px-4 py-6 text-center text-sm text-slate-500">{{ copy.pricingPreview.notConfigured }}</div>
                    </div>

                    <div class="overflow-hidden rounded-2xl border border-edge bg-white">
                      <div class="border-b border-hairline px-4 py-3 text-sm font-semibold text-ink">{{ copy.pricingPreview.unlocksTitle }}</div>
                      <div v-if="pricingPreview.unlocks.length" class="divide-y divide-hairline">
                        <div v-for="unlock in pricingPreview.unlocks" :key="unlock.targetId" class="grid gap-3 p-4 md:grid-cols-3">
                          <div>
                            <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.pricingPreview.unlockTarget }}</div>
                            <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ unlock.targetId }}</div>
                          </div>
                          <div>
                            <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.pricingPreview.priceId }}</div>
                            <div class="tnum mt-1 break-all font-mono text-xs text-primary-700">{{ unlock.priceId }}</div>
                          </div>
                          <div>
                            <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.pricingPreview.productId }}</div>
                            <div class="tnum mt-1 break-all font-mono text-xs text-slate-600">{{ unlock.productId }}</div>
                          </div>
                        </div>
                      </div>
                      <div v-else class="px-4 py-6 text-center text-sm text-slate-500">{{ copy.pricingPreview.notConfigured }}</div>
                    </div>
                  </div>

                  <details class="mt-5 rounded-2xl border border-edge bg-white p-4 md:p-5">
                    <summary class="cursor-pointer text-sm font-medium text-slate-700">{{ copy.advancedJsonTitle }}</summary>
                    <p class="mt-2 text-sm text-slate-500">{{ copy.advancedJsonHint }}</p>
                    <div class="mt-4 grid gap-4 xl:grid-cols-2">
                      <label class="grid gap-1.5">
                        <span class="ui-label">{{ copy.fields.itemsJson }}</span>
                        <textarea v-model="form.items_json" class="tnum min-h-[180px] w-full rounded-[10px] border border-slate-200 bg-slate-50 p-3 font-mono text-xs leading-6 text-slate-700 transition-colors focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                      </label>
                      <label class="grid gap-1.5">
                        <span class="ui-label">{{ copy.fields.pricingJson }}</span>
                        <textarea v-model="form.pricing_json" class="tnum min-h-[180px] w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-xs leading-6 text-slate-900 transition-colors focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                        <p class="text-xs text-slate-400">{{ copy.pricingJsonCreateHint }}</p>
                      </label>
                    </div>
                  </details>
                </section>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-5">
                  <div class="rounded-2xl border border-edge p-4 md:p-5">
                  <h3 class="text-base font-semibold text-ink">{{ copy.actionsTitle }}</h3>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.actionsDescription }}</p>
                  <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <button v-if="canCloneSelectedBundle" class="ui-btn ui-btn-secondary" type="button" :disabled="statusActionBusy" @click="duplicateBundle">
                      <Loader2 v-if="duplicating" class="h-4 w-4 animate-spin" />
                      <Copy v-else class="h-4 w-4 text-slate-400" />
                      {{ duplicating ? copy.duplicating : copy.duplicateDraft }}
                    </button>
                    <button v-if="canPublishSelectedBundle" class="ui-btn ui-btn-secondary" type="button" :disabled="statusActionBusy" @click="publish">
                      <Loader2 v-if="publishing" class="h-4 w-4 animate-spin" />
                      {{ publishing ? copy.publishing : copy.publish }}
                    </button>
                    <button v-if="canDeprecateSelectedBundle" class="ui-btn ui-btn-secondary" type="button" :disabled="statusActionBusy" @click="deprecate">
                      <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
                      {{ deprecating ? copy.deprecating : copy.deprecate }}
                    </button>
                    <button v-if="canDeleteSelectedBundle" class="ui-btn ui-btn-danger" type="button" :disabled="statusActionBusy" @click="showDeleteConfirm = true">
                      <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
                      <Trash2 v-else class="h-4 w-4" />
                      {{ deleting ? copy.deleting : copy.delete }}
                    </button>
                  </div>
                </div>
              </div>

            </main>
          </div>
          <div v-if="activeTab === 'pricing'" class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
            <button class="ui-btn ui-btn-primary w-full min-w-[180px] sm:w-auto" type="button" :disabled="saving" @click="savePricing">
              <Send class="h-4 w-4" />
              {{ copy.savePricing }}
            </button>
          </div>
        </template>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showDeleteConfirm && canDeleteSelectedBundle" class="ui-modal-mask z-50">
          <section class="ui-modal-panel md:max-w-md">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.deleteConfirmTitle }}</h2>
            </div>
            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <p class="text-sm text-slate-600">{{ copy.deleteConfirmDescription }}</p>
              <div class="mt-4 rounded-[10px] bg-slate-50 p-4">
                <div class="text-sm font-semibold text-ink">{{ bundleName(selected) }}</div>
                <div class="tnum mt-1 break-all text-xs text-slate-500">{{ selectedId }}</div>
              </div>
            </main>
            <div class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
              <button class="ui-btn ui-btn-secondary min-w-[96px]" type="button" :disabled="deleting" @click="showDeleteConfirm = false">{{ copy.cancel }}</button>
              <button class="ui-btn ui-btn-danger min-w-[112px]" type="button" :disabled="deleting" @click="removeBundle">
                <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
                {{ deleting ? copy.deleting : copy.confirmDelete }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>

