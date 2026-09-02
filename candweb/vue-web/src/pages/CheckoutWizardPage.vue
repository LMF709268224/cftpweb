<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, ArrowRight, ClipboardList, Loader2, Send, CheckCircle2, CircleAlert, Clock, UploadCloud, X } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import CredentialAttachmentList from "@/components/CredentialAttachmentList.vue"
import LocalizedDatePicker from "@/components/LocalizedDatePicker.vue"
import LoadingState from "@/components/LoadingState.vue"
import CheckoutPaymentPanel from "@/components/CheckoutPaymentPanel.vue"
import { ApiClientError, apiClient } from "@/lib/apiClient"
import { isSystemCredentialDefinition } from "@/lib/credentialDefinitions"
import { formatCurrencyMinorAmount } from "@/lib/display"
import { useTranslation } from "@/lib/language"
import { useUser } from "@/lib/user"
import {
  CN_CITY_LABELS,
  CN_STATE_LABELS,
  countryUsesCityField,
  countryUsesProvinceField,
  getCachedCountries,
  getChinaCityOptions,
  getCountryCityOptions,
  getCountryOptions,
  getOrganizationPhonePrefixes,
  getProvinceOptions,
  getStateCityOptions,
  loadLocationData,
  normalizeAddressCountryCode,
  normalizeLocationForSubmission,
  type CountryOption,
  type PhonePrefixOption,
  resolvePhoneCountryCode,
} from "@/lib/locationOptions"
import { GENDER_OPTIONS, PROFILE_TEXT_LIMITS, isValidEmail, isValidInternationalPhone, isValidPostalCode, normalizeGender, normalizeInternationalPhone, normalizePostalCode, trimToMax } from "@/lib/profileFormValidation"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, statusEnumNameForStatus } from "@/lib/status-labels"
import { getFileConstraintInfo } from "@/lib/fileConstraints"
import { sha256Hex, uploadWithTimeout } from "@/lib/upload"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const { currentUser, fetchUser } = useUser()
type EligibilityBlocker = { blocker_type?: string; description?: string }
const HARD_ELIGIBILITY_BLOCKER_TYPES = new Set([
  "ALREADY_PURCHASED",
  "MISSING_PREREQUISITE_QUALIFICATION",
  "MISSING_UNLOCK_QUALIFICATION",
  "FORBIDDEN_QUALIFICATION",
  "CONFLICT_PIPELINE_IN_PROGRESS",
  "CONFLICT_CHECK_UNAVAILABLE",
  "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
  "EXEMPTION_UNDER_REVIEW",
])
const bundleId = String(route.params.bundleId || route.query.bundleId || "")
const currentStep = ref(1)
const bundleData = ref<any>(null)
const pricingDetail = ref<any>(null)
const pricingEvaluation = ref<any>(null)
const paymentMode = ref("FULL_PIPELINE")
const paymentPreview = ref<any>(null)
const rawExemptionStages = ref<any[]>([])
const exemptionStages = ref<any[]>([])
const systemManagedExemptionUnitIds = ref<Record<string, boolean>>({})
const selectedExemptionUnitIds = ref<Record<string, boolean>>({})
const waivedExemptionUnitIds = ref<Record<string, boolean>>({})
const selectedQualificationIdsByUnit = ref<Record<string, string>>({})
const activeOrderId = ref("")
const activeOrderAction = ref<"purchase" | "credential_application">("purchase")
const activeCredentialQualIds = ref<string[]>([])
const activeCredentialUnitIds = ref<string[]>([])
const activeCredentialApplicationOrder = ref<any>(null)
const credentialApplicationOrders = ref<any[]>([])
const credentialApplicationOrderLoading = ref(false)
const qualificationOrderTargetUnitId = ref("")
const qualificationApplications = ref<Record<string, any>>({})
const qualificationDefinitions = ref<Record<string, any>>({})
const expandedQualificationUnitIds = ref<Record<string, boolean>>({})
const qualificationUploadedFiles = ref<Record<string, Record<string, { name: string; url: string; ext: string; hash: string; size: number }>>>({})
const qualificationUploadingKey = ref("")
const qualificationSubmittingUnitId = ref("")
const levelPlaceholder = "{" + "{level}}"

function selectedExemptedUnitIds() {
  const selectedUnitIds = new Set(
    Object.entries(selectedExemptionUnitIds.value)
      .filter(([, selected]) => selected)
      .map(([unitId]) => unitId),
  )

  // System-managed qualifications are hidden from the candidate decision UI and
  // submitted as waivers for the service to resolve. Keep the service's qualified
  // result in the display calculation so an automatic exemption is not charged.
  for (const stage of rawExemptionStages.value) {
    for (const unit of stage?.units || []) {
      const unitId = String(unit?.unit_id || "").trim()
      if (unitId && unit?.qualified) selectedUnitIds.add(unitId)
    }
  }

  return selectedUnitIds
}

function includedPurchaseUnitIds() {
  const stages = [...(bundleData.value?.stages || [])]
    .sort((left: any, right: any) => Number(left?.sort_order || 0) - Number(right?.sort_order || 0))
  const exemptedUnitIds = selectedExemptedUnitIds()
  const firstActiveStageIndex = stages.findIndex((stage: any) =>
    (stage?.units || []).some((unit: any) => !exemptedUnitIds.has(String(unit?.unit_id || ""))),
  )
  const includedStages = paymentMode.value === "BY_STAGE" && firstActiveStageIndex >= 0
    ? stages.slice(0, firstActiveStageIndex + 1)
    : stages

  return new Set(
    includedStages.flatMap((stage: any) =>
      (stage?.units || [])
        .map((unit: any) => String(unit?.unit_id || "").trim())
        .filter(Boolean),
    ),
  )
}

const pricingDetailCurrency = computed(() => {
  if (!pricingDetail.value) return "USD"

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value
    const prices = [
      ...(detail?.pipelines || []).map((item: any) => item?.enrollment_fee),
      ...(detail?.units || []).flatMap((item: any) => [item?.access, item?.exemption]),
      ...(detail?.memberships || []).map((item: any) => item?.price),
    ]
    return String(prices.find((price: any) => price?.currency)?.currency || "USD")
  } catch {
    return "USD"
  }
})

const dynamicPaymentPreview = computed(() => {
  const amount = pricingEvaluation.value?.preview_pay_amount
  if (typeof amount !== "number") return paymentPreview.value

  return {
    total: amount,
    subtotal: amount,
    currency: pricingDetailCurrency.value,
    pay_amount_label: "",
    amount_label: "",
    discount_total: 0,
  }
})

type PurchaseLineItem = {
  key: string
  itemId: string
  name: string
  stageName: string
  amount: number
  currency: string
  kind: "pipeline" | "course" | "membership"
}

const purchaseLineItems = computed<PurchaseLineItem[]>(() => {
  if (!pricingDetail.value) return []

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value
    const includedUnitIds = includedPurchaseUnitIds()
    const exemptedUnitIds = selectedExemptedUnitIds()
    const unitDetails = new Map<string, { name: string, stageName: string }>()

    for (const stage of bundleData.value?.stages || []) {
      const stageName = String(stage?.name || "").trim()
      for (const unit of stage?.units || []) {
        const unitId = String(unit?.unit_id || "").trim()
        if (!unitId) continue
        unitDetails.set(unitId, {
          name: String(unit?.unit_name || unit?.name || unitId).trim() || unitId,
          stageName,
        })
      }
    }

    const items: PurchaseLineItem[] = []
    if (Array.isArray(detail?.pipelines)) {
      for (const pipeline of detail.pipelines) {
        const configuredPipelineId = String(pipeline?.pipeline_id || "").trim()
        const amount = pipeline?.enrollment_fee?.amount
        if (!configuredPipelineId || typeof amount !== "number") continue
        items.push({
          key: `pipeline:${configuredPipelineId}`,
          itemId: configuredPipelineId,
          name: t.value.checkoutWizard.pipelineEnrollmentFee,
          stageName: String(bundleData.value?.name || "").trim(),
          amount,
          currency: String(pipeline?.enrollment_fee?.currency || "USD"),
          kind: "pipeline",
        })
      }
    }

    if (Array.isArray(detail?.units)) {
      for (const unit of detail.units) {
        const unitId = String(unit?.unit_id || "").trim()
        if (!unitId || !includedUnitIds.has(unitId)) continue
        const exempted = exemptedUnitIds.has(unitId)
        const price = exempted ? unit?.exemption : unit?.access
        const amount = price?.amount
        if (typeof amount !== "number") continue
        const unitDetail = unitDetails.get(unitId)
        items.push({
          key: `course:${unitId}`,
          itemId: unitId,
          name: exempted
            ? `${unitDetail?.name || unitId} · ${t.value.checkoutWizard.exemptionRecognitionFee}`
            : unitDetail?.name || unitId,
          stageName: unitDetail?.stageName || "",
          amount,
          currency: String(price?.currency || unit?.access?.currency || "USD"),
          kind: "course",
        })
      }
    }

    if (Array.isArray(detail?.memberships)) {
      for (const membership of detail.memberships) {
        const membershipId = String(membership?.membership_id || "").trim()
        const amount = membership?.price?.amount
        if (!membershipId || typeof amount !== "number") continue
        const configuredMembershipId = String(bundleData.value?.membership_id || "").trim()
        const configuredMembershipName = membershipId === configuredMembershipId
          ? String(bundleData.value?.membership_name || "").trim()
          : ""
        items.push({
          key: `membership:${membershipId}`,
          itemId: membershipId,
          name: configuredMembershipName || membershipId,
          stageName: "",
          amount,
          currency: String(membership?.price?.currency || "USD"),
          kind: "membership",
        })
      }
    }

    return items
  } catch (err) {
    console.error("Failed to build purchase line items", err)
    return []
  }
})

const unitPriceDisplay = computed<Record<string, { accessAmount?: number, exemptionAmount?: number, currency: string }>>(() => {
  if (!pricingDetail.value) return {}

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value
    const display: Record<string, { accessAmount?: number, exemptionAmount?: number, currency: string }> = {}

    for (const stage of exemptionStages.value) {
      for (const unit of stage.units || []) {
        const pricingUnit = Array.isArray(detail.units)
          ? detail.units.find((item: any) => item.unit_id === unit.unit_id)
          : null
        const exemptionAmount = typeof pricingUnit?.exemption?.amount === "number"
          ? pricingUnit.exemption.amount
          : 0
        const currency = pricingUnit?.exemption?.currency || pricingUnit?.access?.currency || "USD"

        display[unit.unit_id] = {
          accessAmount: typeof pricingUnit?.access?.amount === "number" ? pricingUnit.access.amount : undefined,
          exemptionAmount,
          currency,
        }
      }
    }

    return display
  } catch {
    return {}
  }
})

const isMultiStage = computed(() => {
  return (bundleData.value?.stages?.length || 0) > 1
})
const hasExpandedQualificationEditors = computed(() =>
  Object.values(expandedQualificationUnitIds.value).some(Boolean)
)

const isMembershipBundle = computed(() => {
  if (!bundleData.value) return false
  const itemTypes = bundleData.value.bundle_item_types || bundleData.value.item_types || []
  return itemTypes.some((type: string) => String(type).includes("membership"))
})

const registrationTitle = computed(() => {
  if (!bundleData.value) return t.value.checkoutWizard.checkoutTitle

  const bundleName = String(bundleData.value.name || "").trim()
  const subject = /CFtP/i.test(bundleName) ? "CFtP®Level 1" : bundleName
  if (!subject) return t.value.checkoutWizard.examRegistrationTitle

  return t.value.checkoutWizard.namedExamRegistrationTitle.replace("{{name}}", subject)
})

const loading = ref(false)
const cancellingPaymentOrder = ref(false)
const paymentEditDialogOpen = ref(false)
const qualificationOrderConfirmDialogOpen = ref(false)
const paymentReturnStep = ref(3)
const initialLoading = ref(true)
const pipelineId = computed(() =>
  String(bundleData.value?.pipeline_id || bundleData.value?.pipeline_cc_ulid || "").trim()
)
const paymentBizType = computed(() => {
  if (activeOrderAction.value === "credential_application") return "CREDENTIAL_APPLICATION"
  return "BUNDLE_PURCHASE"
})
const paymentReturnPath = computed(() => {
  if (activeOrderAction.value === "credential_application") return route.path
  return `/checkout/success/${activeOrderId.value}`
})
const paymentReturnParams = computed(() => {
  if (activeOrderAction.value === "credential_application") {
    return {
      qual_ulids: activeCredentialQualIds.value.join(","),
      qualification_unit_ids: activeCredentialUnitIds.value.join(","),
    }
  }
  return {
    bundle_id: bundleId,
    pipeline_id: pipelineId.value,
  }
})
const selectedCountryCode = ref("")
const selectedProvinceCode = ref("")
const countryOptions = ref<CountryOption[]>([])
const provinceOptions = ref<any[]>([])
const cityOptions = ref<any[]>([])
const showProvinceField = computed(() => !selectedCountryCode.value || countryUsesProvinceField(selectedCountryCode.value))
const showCityField = computed(() => !selectedCountryCode.value || countryUsesCityField(selectedCountryCode.value))
const locationGridClass = computed(() => {
  const fieldCount = 1 + Number(showProvinceField.value) + Number(showCityField.value)
  return fieldCount === 3 ? "sm:grid-cols-3" : fieldCount === 2 ? "sm:grid-cols-2" : "sm:grid-cols-1"
})
const orgPhonePrefixes = ref<PhonePrefixOption[]>([])
const genderOptions = GENDER_OPTIONS
const formData = reactive({
  first_name: "",
  middle_name: "",
  last_name: "",
  email: "",
  gender: "",
  birthdate: "",
  country: "",
  province: "",
  city: "",
  address: "",
  postal_code: "",
  phone_country_code: "",
  phone: "",
  agreement: false,
})
function localizedProvinceName(province: any) {
  return lang.value === "zh" && selectedCountryCode.value === "CN" ? CN_STATE_LABELS[province.isoCode] || province.name : province.name
}

function localizedCityName(city: any) {
  if (typeof city?.localizedName === "string") return city.localizedName
  return lang.value === "zh" && selectedCountryCode.value === "CN" ? CN_CITY_LABELS[selectedProvinceCode.value]?.[city.name] || city.name : city.name
}

function normalizeLocationText(value: unknown) {
  return typeof value === "string" ? value.trim().toLowerCase() : ""
}

function normalizeProvinceText(value: unknown) {
  return normalizeLocationText(value)
    .replace(/\s+(province|state|autonomous region|special administrative region)$/i, "")
    .replace(/(壮族自治区|回族自治区|维吾尔自治区|特别行政区|自治区|省|市)$/u, "")
}

function provinceMatchValues(province: any) {
  const values = [province.name, province.isoCode, localizedProvinceName(province)]
  if (selectedCountryCode.value === "CN") {
    values.push(CN_STATE_LABELS[province.isoCode] || "")
  }
  return values
}

function ensureCurrentCityOption() {
  const cityText = normalizeLocationText(formData.city)
  if (!cityText) return
  const exists = cityOptions.value.some((city) =>
    [city.name, localizedCityName(city)].some((value) => normalizeLocationText(value) === cityText),
  )
  if (!exists) {
    cityOptions.value = [{ name: formData.city, localizedName: formData.city }, ...cityOptions.value]
  }
}

function refreshCountryOptions() {
  countryOptions.value = getCountryOptions(lang.value === "zh" ? "zh-CN" : "en")
}

function refreshProvinceOptions() {
  provinceOptions.value = selectedCountryCode.value ? getProvinceOptions(selectedCountryCode.value) : []
}

function refreshCityOptions() {
  if (!selectedCountryCode.value) {
    cityOptions.value = []
    return
  }
  if (!countryUsesCityField(selectedCountryCode.value)) {
    cityOptions.value = []
    return
  }
  if (selectedProvinceCode.value) {
    const chinaCityOptions = selectedCountryCode.value === "CN"
      ? getChinaCityOptions(selectedProvinceCode.value, lang.value)
      : null
    if (chinaCityOptions) {
      cityOptions.value = chinaCityOptions
      return
    }
    cityOptions.value = getStateCityOptions(selectedCountryCode.value, selectedProvinceCode.value)
    return
  }
  cityOptions.value = provinceOptions.value.length === 0 ? getCountryCityOptions(selectedCountryCode.value) : []
}

function syncLocationSelectionFromForm() {
  const allCountries = getCachedCountries()
  if (allCountries.length === 0) return
  const countryText = normalizeLocationText(formData.country)
  const zhRegionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" })
  const matchedCountry = allCountries.find((country: any) =>
    [country.name, country.isoCode, country.phonecode].some((value) => normalizeLocationText(value) === countryText) ||
    normalizeLocationText(zhRegionNames.of(country.isoCode)) === countryText,
  )
  const normalizedCountry = normalizeAddressCountryCode(matchedCountry?.isoCode || "")
  selectedCountryCode.value = normalizedCountry.countryCode
  if (normalizedCountry.provinceCode) {
    formData.country = allCountries.find((country: any) => country.isoCode === "CN")?.name || "China"
  }
  refreshProvinceOptions()
  if (selectedCountryCode.value && !countryUsesProvinceField(selectedCountryCode.value)) {
    formData.province = ""
  }

  const provinceText = normalizeLocationText(formData.province)
  const matchedProvince = selectedCountryCode.value
    ? provinceOptions.value.find((state) =>
        state.isoCode === normalizedCountry.provinceCode ||
        provinceMatchValues(state).some((value) => normalizeProvinceText(value) === normalizeProvinceText(provinceText)),
      )
    : undefined
  selectedProvinceCode.value = matchedProvince?.isoCode || ""
  if (normalizedCountry.provinceCode && matchedProvince) {
    formData.province = localizedProvinceName(matchedProvince)
  }
  if (selectedCountryCode.value && !countryUsesCityField(selectedCountryCode.value)) {
    formData.city = ""
  }
  refreshCityOptions()
  ensureCurrentCityOption()
}

function handleCountryChange() {
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  formData.country = country?.name || ""
  formData.province = ""
  formData.city = ""
  selectedProvinceCode.value = ""
  refreshProvinceOptions()
  refreshCityOptions()
}

function handleProvinceChange() {
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  formData.province = province ? localizedProvinceName(province) : ""
  formData.city = ""
  refreshCityOptions()
}

function sanitizeSignupForm() {
  formData.first_name = trimToMax(formData.first_name, PROFILE_TEXT_LIMITS.name)
  formData.middle_name = trimToMax(formData.middle_name, PROFILE_TEXT_LIMITS.name)
  formData.last_name = trimToMax(formData.last_name, PROFILE_TEXT_LIMITS.name)
  formData.email = trimToMax(formData.email, PROFILE_TEXT_LIMITS.short)
  formData.gender = normalizeGender(formData.gender)
  formData.country = trimToMax(formData.country, PROFILE_TEXT_LIMITS.short)
  formData.province = trimToMax(formData.province, PROFILE_TEXT_LIMITS.short)
  formData.city = trimToMax(formData.city, PROFILE_TEXT_LIMITS.short)
  formData.address = trimToMax(formData.address, PROFILE_TEXT_LIMITS.address)
  formData.postal_code = normalizePostalCode(formData.postal_code)
  formData.phone = normalizeInternationalPhone(formData.phone)
}

function normalizeDate(value: unknown) {
  return typeof value === "string" ? value.split("T")[0] : ""
}

function normalizeAddress(value: unknown, fallback: unknown) {
  if (typeof value === "string") return value
  if (Array.isArray(fallback)) return fallback.join(", ")
  if (typeof fallback === "string") return fallback
  return ""
}

function splitRealName(value: unknown) {
  if (typeof value !== "string") return { firstName: "", lastName: "" }
  const parts = value.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return { firstName: "", lastName: "" }
  return {
    firstName: parts[0] || "",
    lastName: parts.length > 1 ? parts.slice(1).join(" ") : "",
  }
}

function applyProfileToForm(profile: any) {
  const realName = splitRealName(profile.real_name)
  formData.email = profile.email || formData.email
  formData.gender = normalizeGender(profile.gender) || formData.gender
  formData.birthdate = normalizeDate(profile.birthday) || formData.birthdate
  formData.first_name = profile.first_name || realName.firstName || formData.first_name
  formData.middle_name = profile.middle_name || formData.middle_name
  formData.last_name = profile.last_name || realName.lastName || formData.last_name
  formData.phone_country_code = profile.phone_country_code || formData.phone_country_code
  formData.phone = profile.phone || formData.phone
  formData.country = profile.country || profile.region || formData.country
  formData.province = profile.province || formData.province
  formData.city = profile.city || profile.location || formData.city
  formData.address = normalizeAddress(profile.address_text, profile.address) || formData.address
  formData.postal_code = profile.postal_code || formData.postal_code
  syncLocationSelectionFromForm()
}

function firstFilled(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim()
  }
  return ""
}

function takeFormValue(current: unknown, formValue: unknown) {
  const fv = firstFilled(formValue)
  return fv || firstFilled(current)
}

function buildProfilePayload(current: any) {
  const currentAddress = normalizeAddress(current.address_text, current.address)
  return {
    display_name: firstFilled(current.display_name),
    email: takeFormValue(current.email, formData.email),
    first_name: takeFormValue(current.first_name, formData.first_name),
    last_name: takeFormValue(current.last_name, formData.last_name),
    home_phone: current.home_phone || current.phone || "",
    phone_country_code: takeFormValue(current.phone_country_code, formData.phone_country_code),
    phone: takeFormValue(current.phone, formData.phone),
    gender: takeFormValue(current.gender, formData.gender),
    birthday: takeFormValue(normalizeDate(current.birthday), formData.birthdate),
    country: takeFormValue(current.country || current.region, formData.country),
    province: takeFormValue(current.province, formData.province),
    city: takeFormValue(current.city || current.location, formData.city),
    address: takeFormValue(currentAddress, formData.address),
    postal_code: takeFormValue(current.postal_code, formData.postal_code),
    affiliation: current.affiliation || "",
    title: current.title || "",
    real_name: current.real_name || "",
    bio: current.bio || "",
    education: current.education || "",
  }
}

async function loadProfile() {
  try {
    const res = currentUser.value || await fetchUser()
    if (res) {
      applyProfileToForm(res)
    }
  } catch (err) {
    console.error("Failed to load user profile", err)
  }
}

async function fetchOrgConfig() {
  try {
    const configRes = await apiClient("/api/public/config/organization")
    if (configRes && configRes.country_codes) {
      orgPhonePrefixes.value = getOrganizationPhonePrefixes(configRes.country_codes)
      formData.phone_country_code = resolvePhoneCountryCode(
        formData.phone_country_code,
        orgPhonePrefixes.value,
        selectedCountryCode.value,
      )
    }
  } catch (err) {
    console.error("Failed to load organization config", err)
  }
}

onMounted(() => {
  void (async () => {
    await fetchBundleInfo()
    await resumeQualificationUploadAfterPayment()
  })()
  void Promise.all([loadProfile(), loadLocationData()])
    .then(() => {
      refreshCountryOptions()
      syncLocationSelectionFromForm()
      void fetchOrgConfig()
    })
    .catch((err: unknown) => {
      console.error("Failed to load location data", err)
      toast.error(t.value.common.locationDataLoadFailed, { id: "location-data-load-failed" })
    })
})

watch(lang, () => {
  const previousCity = formData.city
  refreshCountryOptions()
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  if (country) formData.country = country.name
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  if (province) formData.province = localizedProvinceName(province)
  refreshCityOptions()
  const city = cityOptions.value.find((item) => [item.name, localizedCityName(item)].includes(formData.city))
  if (city) formData.city = localizedCityName(city)
  else if (lang.value === "zh" && selectedCountryCode.value === "CN" && selectedProvinceCode.value) {
    const mappedCity = Object.entries(CN_CITY_LABELS[selectedProvinceCode.value] || {}).find(([english, chinese]) => english === previousCity || chinese === previousCity)
    if (mappedCity) formData.city = mappedCity[1]
  }
  ensureCurrentCityOption()
  void refreshLocalizedCheckoutContent()
})

watch(paymentMode, () => {
  if (!bundleData.value) return
  void refreshPricingEvaluation().catch((error) => {
    console.error("Failed to refresh pricing after payment mode change", error)
    toast.error(t.value.checkoutWizard.pricingRefreshFailed)
  })
})

async function syncSignupToProfile() {
  const current = await fetchUser(true)
  if (!current) throw new Error("Unable to load the current profile")
  await apiClient("/api/user/profile", {
    method: "PUT",
    body: JSON.stringify(buildProfilePayload(current)),
  })
  await fetchUser(true)
}

function applyBundleInfo(response: any) {
  bundleData.value = response
  const purchaseState = response?.purchase_state || response
  paymentPreview.value = purchaseState?.payment_preview || null

  const stages = purchaseState?.exemption_options?.stages || []
  rawExemptionStages.value = stages.filter((stage: any) => (stage.units?.length || 0) > 0)
  exemptionStages.value = rawExemptionStages.value
  syncQualifiedExemptionSelections(exemptionStages.value)

  if (exemptionStages.value.length === 0 && currentStep.value === 1) {
    currentStep.value = 2
  }
}

function qualificationIdsForStages(stages: any[]) {
  return Array.from(new Set(
    stages
      .flatMap((stage: any) => stage.units || [])
      .flatMap((unit: any) => unit.exemption_quals || [])
      .map((qualification: any) => String(qualification?.qual_id || "").trim())
      .filter(Boolean),
  ))
}

function applyQualificationDefinitionsToStages(definitions: Record<string, any>) {
  const systemManagedUnits: Record<string, boolean> = {}
  for (const stage of rawExemptionStages.value) {
    for (const unit of stage.units || []) {
      const qualifications = unit.exemption_quals || []
      const isSystemManaged = qualifications.length > 0 && qualifications.every((qualification: any) => {
        const qualificationId = String(qualification?.qual_id || "").trim()
        const definition = definitions[qualificationId]
        return Boolean(definition) && isSystemCredentialDefinition(definition)
      })
      const unitId = String(unit?.unit_id || "").trim()
      if (isSystemManaged && unitId) systemManagedUnits[unitId] = true
    }
  }
  systemManagedExemptionUnitIds.value = systemManagedUnits

  exemptionStages.value = rawExemptionStages.value
    .map((stage: any) => ({
      ...stage,
      units: (stage.units || [])
        .map((unit: any) => ({
          ...unit,
          exemption_quals: (unit.exemption_quals || [])
            .filter((qualification: any) => {
              const qualificationId = String(qualification?.qual_id || "").trim()
              const definition = definitions[qualificationId]
              return !definition || !isSystemCredentialDefinition(definition)
            })
            .map((qualification: any) => {
              const qualificationId = String(qualification?.qual_id || "").trim()
              const definition = definitions[qualificationId]
              if (!definition) return qualification
              return {
                ...qualification,
                name: definition.name || qualification.name,
                description: definition.description || qualification.description,
                acquisition_method: definition.acquisition_method || qualification.acquisition_method,
                file_constraints: definition.file_constraints || qualification.file_constraints,
              }
            }),
        }))
        .filter((unit: any) => (unit.exemption_quals?.length || 0) > 0),
    }))
    .filter((stage: any) => (stage.units?.length || 0) > 0)

  syncQualifiedExemptionSelections(exemptionStages.value)
  if (exemptionStages.value.length === 0 && currentStep.value === 1) currentStep.value = 2
}

async function refreshCheckoutQualificationDefinitions() {
  const qualificationIds = qualificationIdsForStages(rawExemptionStages.value)
  if (qualificationIds.length === 0) {
    qualificationDefinitions.value = {}
    return
  }

  const response = await apiClient(
    `/api/credentials/definitions?qual_ulids=${encodeURIComponent(qualificationIds.join(","))}`,
    { suppressErrorToast: true },
  )
  const definitions = Array.isArray(response?.definitions) ? response.definitions : []
  const definitionsById = Object.fromEntries(
    definitions
      .map((definition: any) => [qualificationDefinitionId(definition), definition] as const)
      .filter(([qualificationId]: readonly [string, any]) => Boolean(qualificationId)),
  )
  qualificationDefinitions.value = definitionsById
  applyQualificationDefinitionsToStages(definitionsById)
}

async function applyBundleInfoWithQualificationDefinitions(response: any) {
  applyBundleInfo(response)
  applyQualificationDefinitionsToStages(qualificationDefinitions.value)
  await refreshCheckoutQualificationDefinitions()
}

function syncQualifiedExemptionSelections(stages: any[]) {
  const validUnitIds = new Set<string>()
  const nextSelections: Record<string, boolean> = {}
  const nextWaivers: Record<string, boolean> = {}

  for (const stage of stages) {
    for (const unit of stage.units || []) {
      const unitId = String(unit?.unit_id || "").trim()
      if (!unitId) continue
      validUnitIds.add(unitId)
      if (unit?.qualified) {
        nextSelections[unitId] = true
        continue
      }
      if (selectedExemptionUnitIds.value[unitId]) nextSelections[unitId] = true
      if (waivedExemptionUnitIds.value[unitId]) nextWaivers[unitId] = true
    }
  }

  selectedExemptionUnitIds.value = nextSelections
  waivedExemptionUnitIds.value = Object.fromEntries(
    Object.entries(nextWaivers).filter(([unitId]) => validUnitIds.has(unitId) && !nextSelections[unitId]),
  )
}

async function fetchBundlePayload() {
  return apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}`, {
    suppressErrorToast: true,
  })
}

function bundlePipelineId(response: any) {
  return String(response?.pipeline_id || response?.pipeline_cc_ulid || "").trim()
}

async function refreshLocalizedCheckoutContent() {
  if (!bundleId) return
  try {
    const response = await fetchBundlePayload()
    await applyBundleInfoWithQualificationDefinitions(response)
  } catch (error) {
    console.warn("Failed to refresh localized checkout content", error)
  }
}

async function loadPurchaseReadyBundleInfo() {
  const response = await fetchBundlePayload()
  await applyBundleInfoWithQualificationDefinitions(response)

  try {
    await refreshPricingEvaluation()
  } catch (e) {
    console.error("Failed to load pricing detail", e)
  }

  await refreshQualificationApplications()
  await refreshActiveCredentialApplicationOrder()
  return response
}

async function fetchBundleInfo() {
  if (!bundleId) {
    initialLoading.value = false
    return
  }
  loading.value = true
  try {
    await loadPurchaseReadyBundleInfo()
  } catch (err) {
    console.error(err)
    toast.error(err instanceof Error && err.message
      ? err.message
      : t.value.common.error)
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

function buildSelectedExemptionsJson() {
  const pipelineId = bundlePipelineId(bundleData.value)
  if (!pipelineId) return JSON.stringify({})

  const pipelineSelections = {
    stages: rawExemptionStages.value.map((stage: any) => ({
      stage_cc_ulid: String(stage?.stage_id || stage?.stage_cc_ulid || "").trim(),
      exempted_unit_cc_ulids: (stage?.units || [])
        .map((unit: any) => String(unit?.unit_id || "").trim())
        .filter((unitId: string) => unitId && selectedExemptionUnitIds.value[unitId]),
      waived_unit_cc_ulids: (stage?.units || [])
        .map((unit: any) => String(unit?.unit_id || "").trim())
        .filter((unitId: string) => unitId
          && !selectedExemptionUnitIds.value[unitId]
          && (waivedExemptionUnitIds.value[unitId] || systemManagedExemptionUnitIds.value[unitId])),
    })).filter((stage: any) => stage.stage_cc_ulid),
  }
  return JSON.stringify({ [pipelineId]: pipelineSelections })
}

async function fetchPricingEvaluation() {
  const params = new URLSearchParams()
  params.set("selected_exemptions_json", buildSelectedExemptionsJson())
  params.set("payment_mode", paymentMode.value)
  const response = await apiClient(
    `/api/mall/bundles/${encodeURIComponent(bundleId)}/pricing-detail?${params.toString()}`,
    { suppressErrorToast: true },
  )
  return response
}

let pricingEvaluationRequestId = 0

async function refreshPricingEvaluation() {
  const requestId = ++pricingEvaluationRequestId
  const response = await fetchPricingEvaluation()
  if (requestId !== pricingEvaluationRequestId) return response

  pricingEvaluation.value = response
  if (response?.pricing_detail_json) pricingDetail.value = response.pricing_detail_json
  return response
}



function normalizedCredentialApplicationStatus(status: unknown) {
  const enumName = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status as string)
  return String(enumName || status || "").trim().toUpperCase()
}

function isApplicationPendingStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING"
}

function isApplicationPendingUploadStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING_UPLOAD"
}

function isApplicationApprovedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_APPROVED"
}

function isApplicationRejectedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_REJECTED"
}

function isApplicationResubmitStatus(status: unknown) {
  const value = normalizedCredentialApplicationStatus(status)
  return value === "APPLICATION_STATUS_RESUBMIT" || value === "APPLICATION_STATUS_REUPLOAD"
}

function qualificationIdsForUnit(unit: any) {
  return (unit?.exemption_quals || [])
    .map((qual: any) => String(qual?.qual_id || "").trim())
    .filter(Boolean)
}

function qualificationApplicationEntriesForUnit(unit: any) {
  return qualificationIdsForUnit(unit)
    .map((qualId: string) => ({
      qualId,
      option: (unit?.exemption_quals || []).find(
        (qualification: any) => String(qualification?.qual_id || "").trim() === qualId,
      ) || null,
      definition: qualificationDefinitions.value[qualId] || null,
      application: qualificationApplications.value[qualId] || null,
    }))
}

function prioritizedQualificationApplicationEntry(unit: any) {
  const entries = qualificationApplicationEntriesForUnit(unit)
    .filter((entry: any) => Boolean(entry.application))
  return entries.find((entry: any) => isApplicationPendingUploadStatus(entry.application?.status))
    || entries.find((entry: any) => isApplicationResubmitStatus(entry.application?.status))
    || entries.find((entry: any) => isApplicationPendingStatus(entry.application?.status))
    || entries.find((entry: any) => isApplicationRejectedStatus(entry.application?.status))
    || entries.find((entry: any) => isApplicationApprovedStatus(entry.application?.status))
    || entries[0]
    || null
}

function selectedQualificationIdForUnit(unit: any) {
  const unitId = String(unit?.unit_id || "").trim()
  const qualificationIds = qualificationIdsForUnit(unit)
  const explicitSelection = selectedQualificationIdsByUnit.value[unitId]
  if (explicitSelection && qualificationIds.includes(explicitSelection)) return explicitSelection

  const activeOrderSelection = String(activeCredentialApplicationOrderItemForUnit(unit)?.qual_id || "").trim()
  if (activeOrderSelection && qualificationIds.includes(activeOrderSelection)) return activeOrderSelection

  const applicationSelection = prioritizedQualificationApplicationEntry(unit)?.qualId || ""
  if (applicationSelection) return applicationSelection

  const activeQualification = (unit?.exemption_quals || []).find((qualification: any) =>
    qualification?.eligible
    || String(qualification?.credential_status || "").trim().toUpperCase() === "CREDENTIAL_STATUS_ACTIVE"
  )
  const activeQualificationId = String(activeQualification?.qual_id || "").trim()
  if (activeQualificationId) return activeQualificationId
  return qualificationIds.length === 1 ? qualificationIds[0] : ""
}

function qualificationApplicationForUnit(unit: any) {
  const selectedQualificationId = selectedQualificationIdForUnit(unit)
  if (selectedQualificationId && qualificationApplications.value[selectedQualificationId]) {
    return qualificationApplications.value[selectedQualificationId]
  }
  return prioritizedQualificationApplicationEntry(unit)?.application || null
}

async function latestCredentialApplication(qualId: string) {
  const response = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`, {
    suppressErrorToast: true,
  })
  return (response?.applications || [])[0] || null
}

async function refreshQualificationApplications() {
  const qualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage: any) => stage.units || [])
      .flatMap((unit: any) => qualificationIdsForUnit(unit)),
  ))
  const validQualificationIds = new Set(qualIds)
  const next: Record<string, any> = Object.fromEntries(
    Object.entries(qualificationApplications.value)
      .filter(([qualId]) => validQualificationIds.has(qualId)),
  )
  await Promise.all(qualIds.map(async (qualId) => {
    try {
      const application = await latestCredentialApplication(qualId)
      if (application) next[qualId] = application
      else delete next[qualId]
    } catch (error) {
      console.warn(`Failed to load credential application ${qualId}`, error)
    }
  }))
  qualificationApplications.value = next
  const nextSelections = { ...selectedExemptionUnitIds.value }
  const nextWaivers = { ...waivedExemptionUnitIds.value }
  for (const stage of exemptionStages.value) {
    for (const unit of stage.units || []) {
      const unitId = String(unit?.unit_id || "").trim()
      const state = exemptionCredentialState(unit)
      if (!unitId || !["active", "pending", "pending_upload", "resubmit"].includes(state)) continue
      nextSelections[unitId] = true
      delete nextWaivers[unitId]
    }
  }
  selectedExemptionUnitIds.value = nextSelections
  waivedExemptionUnitIds.value = nextWaivers
}

const QUALIFICATION_POLL_INTERVAL_MS = 30_000
let pollingTimer: ReturnType<typeof setInterval> | null = null
let qualificationPollingInFlight = false

function hasPendingQualificationApplications() {
  return exemptionStages.value.some((stage: any) =>
    (stage.units || []).some((unit: any) => exemptionCredentialState(unit) === "pending")
  )
}

async function pollQualificationApplications() {
  if (qualificationPollingInFlight || currentStep.value !== 1 || document.visibilityState === "hidden") return

  qualificationPollingInFlight = true
  const hadPendingApplications = hasPendingQualificationApplications()
  try {
    await refreshQualificationApplications()
    await refreshActiveCredentialApplicationOrder()

    if (hadPendingApplications && !hasPendingQualificationApplications()) {
      const response = await fetchBundlePayload()
      await applyBundleInfoWithQualificationDefinitions(response)
    }
  } catch {
    // Ignore background polling errors. The next interval will retry.
  } finally {
    qualificationPollingInFlight = false
  }
}

function checkPolling() {
  const needsPolling = hasPendingQualificationApplications()

  if (needsPolling && !pollingTimer && currentStep.value === 1) {
    pollingTimer = setInterval(() => {
      if (currentStep.value !== 1) {
        stopPolling()
        return
      }
      void pollQualificationApplications()
    }, QUALIFICATION_POLL_INTERVAL_MS)
  } else if (!needsPolling && pollingTimer) {
    stopPolling()
  }
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

onUnmounted(() => {
  stopPolling()
})

watch([qualificationApplications, currentStep], checkPolling, { deep: true })

function qualificationDefinitionId(definition: any) {
  return String(definition?.cred_def_id || definition?.cred_def_ulid || "").trim()
}

function qualificationApplicationId(application: any) {
  return String(application?.app_id || application?.app_ulid || "").trim()
}

function exemptionUnitById(unitId: string) {
  return exemptionStages.value
    .flatMap((stage: any) => stage.units || [])
    .find((unit: any) => String(unit?.unit_id || "") === unitId)
}

function exemptionUnitByQualId(qualId: string) {
  return exemptionStages.value
    .flatMap((stage: any) => stage.units || [])
    .find((unit: any) => qualificationIdsForUnit(unit).includes(qualId))
}

function qualificationDefinitionForUnit(unit: any) {
  const qualId = selectedQualificationIdForUnit(unit)
  return qualificationDefinitions.value[qualId] || null
}

function qualificationFilesForUnit(unitId: string) {
  return qualificationUploadedFiles.value[unitId] || {}
}

async function loadQualificationDefinition(qualId: string) {
  if (qualificationDefinitions.value[qualId]) return qualificationDefinitions.value[qualId]
  const response = await apiClient(`/api/credentials/definitions?qual_ulids=${encodeURIComponent(qualId)}`)
  const definitions = Array.isArray(response?.definitions) ? response.definitions : []
  const definition = definitions.find((item: any) => qualificationDefinitionId(item) === qualId) || definitions[0]
  if (!definition) {
    throw new Error(t.value.credentialsPage.materialRequirementsUnavailable)
  }
  qualificationDefinitions.value = {
    ...qualificationDefinitions.value,
    [qualId]: definition,
  }
  return definition
}

async function openQualificationEditor(unit: any, qualId = selectedQualificationIdForUnit(unit)) {
  const unitId = String(unit?.unit_id || "").trim()
  if (!unitId || !qualId || !qualificationIdsForUnit(unit).includes(qualId)) return
  selectedQualificationIdsByUnit.value = {
    ...selectedQualificationIdsByUnit.value,
    [unitId]: qualId,
  }
  await loadQualificationDefinition(qualId)
  expandedQualificationUnitIds.value = {
    ...expandedQualificationUnitIds.value,
    [unitId]: true,
  }
}

async function selectQualificationForUnit(unit: any, qualificationId: unknown) {
  const unitId = String(unit?.unit_id || "").trim()
  const qualId = String(qualificationId || "").trim()
  if (!unitId || !qualificationIdsForUnit(unit).includes(qualId)) return
  selectedQualificationIdsByUnit.value = {
    ...selectedQualificationIdsByUnit.value,
    [unitId]: qualId,
  }
  qualificationUploadedFiles.value = {
    ...qualificationUploadedFiles.value,
    [unitId]: {},
  }
  try {
    await openQualificationEditor(unit, qualId)
  } catch (error) {
    console.error(error)
    toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
  }
}

function closeQualificationEditor(unitId: string) {
  const next = { ...expandedQualificationUnitIds.value }
  delete next[unitId]
  expandedQualificationUnitIds.value = next
}

function isQualificationEditorExpanded(unitId: string) {
  return Boolean(expandedQualificationUnitIds.value[unitId])
}

function qualificationConstraintInputId(unitId: string, constraintName: string) {
  return `qualification-file-${unitId}-${constraintName}`
}

function triggerQualificationFileInput(unitId: string, constraintName: string) {
  document.getElementById(qualificationConstraintInputId(unitId, constraintName))?.click()
}

function qualificationFormatHint(constraint: any) {
  const info = getFileConstraintInfo(constraint?.type)
  const extText = info.extLabel === "Any" ? t.value.credentialsPage.anyFileType : info.extLabel
  return t.value.credentialsPage.supportedFormats
    .replace("{{exts}}", extText)
    .replace("{{limit}}", info.maxLabel)
}

function qualificationConstraintDisplayName(constraint: any) {
  return String(constraint?.display_name || constraint?.name || "")
}

function qualificationUploadSuccessText(fileName: string) {
  return t.value.credentialsPage.uploadSuccess.replace("{{fileName}}", fileName)
}

async function onQualificationFileChange(event: Event, unit: any, constraint: any) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) await uploadQualificationFile(unit, constraint, file)
  input.value = ""
}

async function uploadQualificationFile(unit: any, constraint: any, file: File) {
  const unitId = String(unit?.unit_id || "")
  const qualId = selectedQualificationIdForUnit(unit)
  const constraintName = String(constraint?.name || "").trim()
  const uploadingKey = `${unitId}:${constraintName}`
  if (!unitId || !qualId || !constraintName || qualificationUploadingKey.value) return

  const info = getFileConstraintInfo(constraint?.type)
  const fileExt = file.name.includes(".") ? `.${file.name.split(".").pop()?.toLowerCase()}` : ""
  if (info.maxSize && file.size > info.maxSize) {
    toast.error(t.value.credentialsPage.fileSizeLimitError.replace("{{limit}}", info.maxLabel))
    return
  }
  if (info.exts.length > 0 && !info.exts.includes(fileExt)) {
    toast.error(t.value.credentialsPage.fileTypeError.replace("{{exts}}", info.extLabel))
    return
  }

  qualificationUploadingKey.value = uploadingKey
  try {
    const fileHash = await sha256Hex(file)
    const contentType = file.type || "application/octet-stream"
    const upload = await apiClient("/api/credentials/upload-url", {
      method: "POST",
      body: JSON.stringify({
        cred_def_ulid: qualId,
        file_name: file.name,
        file_ext: fileExt,
        file_hash: fileHash,
        content_type: contentType,
        file_usage: constraintName,
      }),
    })
    const uploadResponse = await uploadWithTimeout(upload.upload_url, {
      method: "PUT",
      headers: new Headers(upload.signed_headers || {}),
      body: file,
    })
    if (!uploadResponse.ok) {
      throw new Error(`S3 upload failed: ${uploadResponse.status} ${uploadResponse.statusText}`)
    }
    qualificationUploadedFiles.value = {
      ...qualificationUploadedFiles.value,
      [unitId]: {
        ...qualificationFilesForUnit(unitId),
        [constraintName]: {
          name: file.name,
          url: upload.file_key,
          ext: fileExt,
          hash: fileHash,
          size: file.size,
        },
      },
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error || "")
    toast.error(`${t.value.credentialsPage.uploadFailed}: ${message}`)
  } finally {
    qualificationUploadingKey.value = ""
  }
}

async function submitQualificationApplication(unit: any) {
  const unitId = String(unit?.unit_id || "")
  const qualId = selectedQualificationIdForUnit(unit)
  const definition = qualificationDefinitionForUnit(unit)
  const constraints = definition?.file_constraints
  const uploadedFiles = qualificationFilesForUnit(unitId)
  if (!unitId || !qualId || !Array.isArray(constraints)) {
    toast.error(t.value.credentialsPage.materialRequirementsUnavailable)
    return
  }
  if (Object.keys(uploadedFiles).length === 0
    || constraints.some((constraint: any) => constraint.is_required && !uploadedFiles[constraint.name])) {
    toast.error(t.value.credentialsPage.requiredMaterialsMissing)
    return
  }

  const evidenceFiles = Object.keys(uploadedFiles).map((constraintName) => ({
    file_name: uploadedFiles[constraintName].name,
    file_url: uploadedFiles[constraintName].url,
    file_hash: uploadedFiles[constraintName].hash,
    file_ext: uploadedFiles[constraintName].ext,
    file_size: uploadedFiles[constraintName].size,
    file_usage: constraintName,
    file_type: constraints.find((constraint: any) => constraint.name === constraintName)?.type || 1,
  }))
  const existingApplication = qualificationApplications.value[qualId]
  qualificationSubmittingUnitId.value = unitId
  try {
    if (isApplicationResubmitStatus(existingApplication?.status)) {
      const appId = qualificationApplicationId(existingApplication)
      if (!appId) throw new Error(t.value.credentialsPage.submitFailed)
      await apiClient("/api/credentials/update", {
        method: "PUT",
        body: JSON.stringify({ app_id: appId, files: evidenceFiles }),
      })
    } else {
      await apiClient("/api/credentials/submit", {
        method: "POST",
        body: JSON.stringify({ cred_def_ulid: qualId, files: evidenceFiles }),
      })
    }
    toast.success(t.value.credentialsPage.submitSuccess)
    closeQualificationEditor(unitId)
    qualificationUploadedFiles.value = {
      ...qualificationUploadedFiles.value,
      [unitId]: {},
    }
    await refreshQualificationApplications()
  } catch (error) {
    console.error(error)
    toast.error(t.value.credentialsPage.submitFailed)
  } finally {
    qualificationSubmittingUnitId.value = ""
  }
}

async function resumeQualificationUploadAfterPayment() {
  const paymentAction = String(route.query.payment_action || "")
  const paymentStatus = String(route.query.payment_status || "")
  if (paymentAction !== "credential_application" || paymentStatus !== "success") return
  const qualIds = String(route.query.qual_ulids || "").split(",").map((value) => value.trim()).filter(Boolean)
  const unitIds = String(route.query.qualification_unit_ids || route.query.qualification_unit_id || "")
    .split(",").map((value) => value.trim()).filter(Boolean)
  currentStep.value = 1
  const nextQualificationSelections = { ...selectedQualificationIdsByUnit.value }
  qualIds.forEach((qualId, index) => {
    const unitId = unitIds[index] || ""
    if (unitId) nextQualificationSelections[unitId] = qualId
  })
  selectedQualificationIdsByUnit.value = nextQualificationSelections

  const expectedQualificationIds = Array.from(new Set(qualIds))
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await refreshQualificationApplications()
    if (expectedQualificationIds.every((qualId) => Boolean(qualificationApplications.value[qualId]))) break
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  await Promise.all(qualIds.map(async (qualId, index) => {
    const unit = exemptionUnitById(unitIds[index] || "") || exemptionUnitByQualId(qualId)
    if (!unit) return
    try {
      const application = qualificationApplications.value[qualId]
      if (!isApplicationPendingUploadStatus(application?.status) && !isApplicationResubmitStatus(application?.status)) return
      await openQualificationEditor(unit, qualId)
    } catch (error) {
      console.error(error)
      toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    }
  }))
  const nextQuery = { ...route.query }
  delete nextQuery.payment_status
  delete nextQuery.payment_action
  delete nextQuery.order_id
  delete nextQuery.qual_ulids
  delete nextQuery.qualification_unit_id
  delete nextQuery.qualification_unit_ids
  await router.replace({ path: route.path, query: nextQuery })

  const missingQualificationIds = expectedQualificationIds.filter(
    (qualId) => !qualificationApplications.value[qualId],
  )
  if (missingQualificationIds.length > 0) {
    toast.info(t.value.checkoutWizard.qualificationApplicationsPreparing)
    await router.push({
      path: "/credentials",
      query: { qual_ulids: expectedQualificationIds.join(",") },
    })
  }
}

function isUploadReadyStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("UPLOAD_READY")
}

function isCredentialApplicationPaymentStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("WAIT_REVIEW_FEE_PAYMENT")
}

function isCredentialApplicationUnderReviewStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("UNDER_REVIEW")
}

function isCredentialApplicationResolvedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("RESOLVED")
}

function credentialApplicationOrderStatus(order: any) {
  return String(order?.order_status || "").trim().toUpperCase()
}

function credentialApplicationOrderIsTerminal(order: any) {
  return ["RESOLVED", "FAILED", "CANCELLED"].includes(credentialApplicationOrderStatus(order))
}

function credentialApplicationOrderForUnit(unit: any) {
  const qualificationIds = new Set(qualificationIdsForUnit(unit))
  const matchingOrders = credentialApplicationOrders.value.filter((order: any) =>
    (order?.items || []).some((item: any) => qualificationIds.has(String(item?.qual_id || "").trim())),
  )
  return matchingOrders.find((order: any) => !credentialApplicationOrderIsTerminal(order)) || matchingOrders[0] || null
}

function activeCredentialApplicationOrderItemForUnit(unit: any) {
  const qualificationIds = new Set(qualificationIdsForUnit(unit))
  const order = credentialApplicationOrderForUnit(unit)
  return (order?.items || []).find((item: any) =>
    qualificationIds.has(String(item?.qual_id || "").trim()),
  ) || null
}

function activeCredentialApplicationOrderItemStatus(unit: any) {
  return String(activeCredentialApplicationOrderItemForUnit(unit)?.item_status || "").trim().toUpperCase()
}

function activeOrderIncludesUnit(unit: any) {
  return Boolean(activeCredentialApplicationOrderItemForUnit(unit))
}

function activeOrderLocksDecisionForUnit(unit: any) {
  const order = credentialApplicationOrderForUnit(unit)
  if (!order || credentialApplicationOrderIsTerminal(order)) return false
  return true
}

function activeOrderUnitStatusLabel(unit: any) {
  const order = credentialApplicationOrderForUnit(unit)
  if (!order) return ""
  const itemStatus = activeCredentialApplicationOrderItemStatus(unit)
  if (itemStatus === "APPROVED") return t.value.checkoutWizard.automaticExemptionApplied
  if (itemStatus === "REJECTED") return t.value.checkoutWizard.qualificationReviewRejected
  if (itemStatus === "SUBMITTED") return t.value.checkoutWizard.qualificationUnderReview
  if (credentialApplicationOrderIsTerminal(order)) return t.value.checkoutWizard.qualificationOrderClosed
  const status = credentialApplicationOrderStatus(order)
  if (isCredentialApplicationPaymentStatus(status)) return t.value.checkoutWizard.qualificationPaymentPending
  if (isUploadReadyStatus(status)) return t.value.checkoutWizard.qualificationUploadReady
  if (isCredentialApplicationUnderReviewStatus(status)) return t.value.checkoutWizard.qualificationUnderReview
  return ""
}

async function refreshActiveCredentialApplicationOrder() {
  const response = await apiClient("/api/credentials/application-orders", {
    suppressErrorToast: true,
  })
  credentialApplicationOrders.value = Array.isArray(response?.orders) ? response.orders : []

  if (activeOrderId.value) {
    activeCredentialApplicationOrder.value = credentialApplicationOrders.value.find(
      (order: any) => String(order?.application_order_ulid || "").trim() === activeOrderId.value,
    ) || null
  }

  const nextSelections = { ...selectedExemptionUnitIds.value }
  const nextWaivers = { ...waivedExemptionUnitIds.value }
  const nextQualificationSelections = { ...selectedQualificationIdsByUnit.value }
  const uploadReadyEntries: Array<{ unit: any; qualId: string }> = []

  for (const unit of allExemptionUnits()) {
    const order = credentialApplicationOrderForUnit(unit)
    const item = activeCredentialApplicationOrderItemForUnit(unit)
    const unitId = String(unit?.unit_id || "").trim()
    const qualId = String(item?.qual_id || "").trim()
    if (!order || !item || !unitId || !qualId) continue

    nextQualificationSelections[unitId] = qualId
    const itemStatus = String(item?.item_status || "").trim().toUpperCase()
    const status = credentialApplicationOrderStatus(order)
    if (["PENDING", "SUBMITTED", "APPROVED"].includes(itemStatus)
      || isCredentialApplicationPaymentStatus(status)
      || isUploadReadyStatus(status)
      || isCredentialApplicationUnderReviewStatus(status)) {
      nextSelections[unitId] = true
      delete nextWaivers[unitId]
    }
    const applicationStatus = qualificationApplications.value[qualId]?.status
    if ((isUploadReadyStatus(status) || isCredentialApplicationUnderReviewStatus(status))
      && (isApplicationPendingUploadStatus(applicationStatus) || isApplicationResubmitStatus(applicationStatus))) {
      uploadReadyEntries.push({ unit, qualId })
    }
  }

  selectedExemptionUnitIds.value = nextSelections
  waivedExemptionUnitIds.value = nextWaivers
  selectedQualificationIdsByUnit.value = nextQualificationSelections
  await Promise.all(uploadReadyEntries.map(({ unit, qualId }) => openQualificationEditor(unit, qualId)))
}

function allExemptionUnits() {
  return exemptionStages.value.flatMap((stage: any) => stage.units || [])
}

function exemptionDecision(unit: any): "exempt" | "waive" | "" {
  const unitId = String(unit?.unit_id || "").trim()
  if (!unitId) return ""
  if (selectedExemptionUnitIds.value[unitId]) return "exempt"
  if (waivedExemptionUnitIds.value[unitId]) return "waive"
  return ""
}

function canUploadQualificationForUnit(unit: any) {
  const status = qualificationApplicationForUnit(unit)?.status
  return isApplicationPendingUploadStatus(status) || isApplicationResubmitStatus(status)
}

async function setExemptionDecision(unit: any, decision: "exempt" | "waive") {
  const unitId = String(unit?.unit_id || "").trim()
  if (!unitId || ["active", "pending", "pending_upload", "resubmit"].includes(exemptionCredentialState(unit)) || activeOrderLocksDecisionForUnit(unit)) return
  const nextSelections = { ...selectedExemptionUnitIds.value }
  const nextWaivers = { ...waivedExemptionUnitIds.value }
  if (decision === "exempt") {
    nextSelections[unitId] = true
    delete nextWaivers[unitId]
    const qualificationIds = qualificationIdsForUnit(unit)
    if (qualificationIds.length === 1) {
      try {
        await openQualificationEditor(unit, qualificationIds[0])
      } catch (error) {
        console.error(error)
        toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
      }
    } else {
      expandedQualificationUnitIds.value = {
        ...expandedQualificationUnitIds.value,
        [unitId]: true,
      }
    }
  } else {
    nextWaivers[unitId] = true
    delete nextSelections[unitId]
    closeQualificationEditor(unitId)
    const nextQualificationSelections = { ...selectedQualificationIdsByUnit.value }
    const nextUploadedFiles = { ...qualificationUploadedFiles.value }
    delete nextQualificationSelections[unitId]
    delete nextUploadedFiles[unitId]
    selectedQualificationIdsByUnit.value = nextQualificationSelections
    qualificationUploadedFiles.value = nextUploadedFiles
  }
  selectedExemptionUnitIds.value = nextSelections
  waivedExemptionUnitIds.value = nextWaivers
}

function selectedUnitsNeedingApplication() {
  return allExemptionUnits().filter((unit: any) => {
    const unitId = String(unit?.unit_id || "").trim()
    if (!unitId || !selectedExemptionUnitIds.value[unitId]) return false
    if (activeOrderLocksDecisionForUnit(unit)) return false
    const state = exemptionCredentialState(unit)
    return !["active", "pending", "pending_upload", "resubmit"].includes(state) && !canUploadQualificationForUnit(unit)
  })
}

const hasSelectedUnitsNeedingApplication = computed(() => selectedUnitsNeedingApplication().length > 0)
const qualificationOrderTargetUnit = computed(() => allExemptionUnits().find(
  (unit: any) => String(unit?.unit_id || "").trim() === qualificationOrderTargetUnitId.value,
) || null)
const qualificationOrderConfirmItems = computed(() => {
  const unit = qualificationOrderTargetUnit.value
  if (!unit) return []
  return [{
    unitId: String(unit?.unit_id || "").trim(),
    unitName: String(unit?.unit_name || unit?.name || "").trim(),
    qualificationName: String(
      qualificationDefinitionForUnit(unit)?.name
      || (unit?.exemption_quals || []).find(
        (qualification: any) => String(qualification?.qual_id || "").trim() === selectedQualificationIdForUnit(unit),
      )?.name
      || "",
    ).trim(),
  }]
})
const hasUndecidedExemptionUnits = computed(() => allExemptionUnits().some((unit: any) => !exemptionDecision(unit)))

function canRequestQualificationApplicationForUnit(unit: any) {
  const unitId = String(unit?.unit_id || "").trim()
  if (!unitId || !selectedExemptionUnitIds.value[unitId] || !selectedQualificationIdForUnit(unit)) return false
  const state = exemptionCredentialState(unit)
  if (["active", "pending", "pending_upload", "resubmit"].includes(state)) return false
  const existingOrder = credentialApplicationOrderForUnit(unit)
  return !existingOrder || credentialApplicationOrderIsTerminal(existingOrder) || isCredentialApplicationPaymentStatus(credentialApplicationOrderStatus(existingOrder))
}

function qualificationApplicationActionLabel(unit: any) {
  const existingOrder = credentialApplicationOrderForUnit(unit)
  return existingOrder && isCredentialApplicationPaymentStatus(credentialApplicationOrderStatus(existingOrder))
    ? t.value.checkoutWizard.continueQualificationPayment
    : t.value.checkoutWizard.applyThisExemption
}

async function startSelectedQualificationApplications() {
  const unit = qualificationOrderTargetUnit.value
  if (!unit || !pipelineId.value || !bundleId) {
    toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    return
  }

  credentialApplicationOrderLoading.value = true
  try {
    const unitId = String(unit?.unit_id || "").trim()
    const qualId = selectedQualificationIdForUnit(unit)
    if (!unitId || !qualId) {
      toast.error(t.value.checkoutWizard.exemptionQualificationRequired)
      return
    }

    const existingApplication = qualificationApplications.value[qualId] || await latestCredentialApplication(qualId)
    if (existingApplication) {
      qualificationApplications.value = { ...qualificationApplications.value, [qualId]: existingApplication }
      if (isApplicationPendingStatus(existingApplication.status)) {
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      if (isApplicationApprovedStatus(existingApplication.status)) {
        await loadPurchaseReadyBundleInfo()
        return
      }
      if (isApplicationPendingUploadStatus(existingApplication.status) || isApplicationResubmitStatus(existingApplication.status)) {
        await openQualificationEditor(unit, qualId)
        return
      }
    }

    let order
    try {
      order = await apiClient("/api/credentials/application-orders", {
        method: "POST",
        suppressErrorToast: true,
        body: JSON.stringify({
          pipeline_cc_ulid: pipelineId.value,
          bundle_ulid: bundleId,
          qual_ulids: [qualId],
        }),
      })
    } catch (error) {
      const message = error instanceof ApiClientError
        ? error.rawMessage || error.errorCode || ""
        : error instanceof Error ? error.message : ""
      if (message.includes("in-progress credential application") || message.includes("进行中") || message.includes("请先处理")) {
        await refreshQualificationApplications()
        await refreshActiveCredentialApplicationOrder()
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      throw error
    }

    const orderId = String(order?.application_order_ulid || "").trim()
    const orderStatus = String(order?.order_status || "")
    activeCredentialApplicationOrder.value = {
      found: true,
      application_order_ulid: orderId,
      order_status: orderStatus,
      pay_order_ulid: String(order?.pay_order_ulid || "").trim(),
      items: [{ qual_id: qualId }],
    }
    credentialApplicationOrders.value = [
      activeCredentialApplicationOrder.value,
      ...credentialApplicationOrders.value.filter(
        (existingOrder: any) => String(existingOrder?.application_order_ulid || "").trim() !== orderId,
      ),
    ]
    if (isUploadReadyStatus(orderStatus)) {
      await refreshQualificationApplications()
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      if (canUploadQualificationForUnit(unit)) await openQualificationEditor(unit, qualId)
      return
    }
    if (isCredentialApplicationUnderReviewStatus(orderStatus)) {
      await refreshQualificationApplications()
      toast.info(t.value.checkoutWizard.qualificationUnderReview)
      return
    }
    if (isCredentialApplicationResolvedStatus(orderStatus)) {
      await loadPurchaseReadyBundleInfo()
      return
    }
    if (isCredentialApplicationPaymentStatus(orderStatus) || order?.payment_key) {
      if (!orderId) {
        throw new Error(t.value.checkoutWizard.qualificationApplicationFailed)
      }
      activeCredentialQualIds.value = [qualId]
      activeCredentialUnitIds.value = [unitId]
      activeOrderAction.value = "credential_application"
      activeOrderId.value = orderId
      currentStep.value = 4
      return
    }
    toast.info(t.value.checkoutWizard.qualificationApplicationCreated)
  } catch (error) {
    console.error(error)
    toast.error(error instanceof Error && error.message
      ? error.message
      : t.value.checkoutWizard.qualificationApplicationFailed)
  } finally {
    credentialApplicationOrderLoading.value = false
  }
}

function requestSelectedQualificationApplications(unit: any) {
  const unitId = String(unit?.unit_id || "").trim()
  if (!unitId || !selectedExemptionUnitIds.value[unitId]) {
    toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    return
  }
  if (!selectedQualificationIdForUnit(unit)) {
    toast.error(t.value.checkoutWizard.exemptionQualificationRequired)
    return
  }
  const existingOrder = credentialApplicationOrderForUnit(unit)
  if (existingOrder && !credentialApplicationOrderIsTerminal(existingOrder)) {
    const status = credentialApplicationOrderStatus(existingOrder)
    if (isCredentialApplicationPaymentStatus(status)) {
      activeCredentialApplicationOrder.value = existingOrder
      activeCredentialQualIds.value = [selectedQualificationIdForUnit(unit)]
      activeCredentialUnitIds.value = [unitId]
      activeOrderAction.value = "credential_application"
      activeOrderId.value = String(existingOrder?.application_order_ulid || "").trim()
      currentStep.value = 4
      return
    }
    if (canUploadQualificationForUnit(unit)) {
      void openQualificationEditor(unit, selectedQualificationIdForUnit(unit))
      return
    }
    toast.info(activeOrderUnitStatusLabel(unit) || t.value.checkoutWizard.qualificationUnderReview)
    return
  }
  qualificationOrderTargetUnitId.value = unitId
  qualificationOrderConfirmDialogOpen.value = true
}

function closeQualificationOrderConfirmDialog() {
  if (credentialApplicationOrderLoading.value) return
  qualificationOrderConfirmDialogOpen.value = false
  qualificationOrderTargetUnitId.value = ""
}

async function confirmSelectedQualificationApplications() {
  qualificationOrderConfirmDialogOpen.value = false
  await startSelectedQualificationApplications()
  qualificationOrderTargetUnitId.value = ""
}

async function nextFromStep1() {
  if (hasUndecidedExemptionUnits.value) {
    toast.info(t.value.checkoutWizard.exemptionDecisionRequired)
    return
  }
  if (hasSelectedUnitsNeedingApplication.value) {
    toast.info(t.value.checkoutWizard.applySelectedExemptionsFirst)
    return
  }
  try {
    const evaluation = await refreshPricingEvaluation()
    if (evaluation?.can_checkout === false) {
      toast.info(
        evaluation?.checkout_blocker_reason === "EXEMPTIONS_UNCONFIRMED_WAIVER"
          ? t.value.checkoutWizard.exemptionDecisionRequired
          : evaluation?.checkout_blocker_reason || t.value.checkoutWizard.checkoutBlockedByExemption,
      )
      return
    }
  } catch (error) {
    console.error(error)
    toast.error(t.value.checkoutWizard.pricingRefreshFailed)
    return
  }
  currentStep.value = 2
}

function formatMoney(amount?: number, currency = "usd") {
  return formatCurrencyMinorAmount(amount, currency) || "-"
}

type ExemptionCredentialState = "active" | "pending" | "pending_upload" | "resubmit" | "rejected" | "expired" | "revoked" | "missing" | "unavailable"

function exemptionCredentialState(unit: any): ExemptionCredentialState {
  const qualifications = unit?.exemption_quals || []
  if (unit?.qualified || qualifications.some((qual: any) =>
    qual?.eligible || String(qual?.credential_status || "").toUpperCase() === "CREDENTIAL_STATUS_ACTIVE"
  )) {
    return "active"
  }

  const application = qualificationApplicationForUnit(unit)
  if (isApplicationPendingUploadStatus(application?.status)) return "pending_upload"
  if (isApplicationPendingStatus(application?.status)) return "pending"
  if (isApplicationResubmitStatus(application?.status)) return "resubmit"
  if (isApplicationRejectedStatus(application?.status)) return "rejected"
  if (isApplicationApprovedStatus(application?.status)) return "active"

  const statuses = qualifications
    .map((qual: any) => String(qual?.credential_status || "").trim().toUpperCase())
    .filter(Boolean)
  if (statuses.includes("CREDENTIAL_STATUS_EXPIRED")) return "expired"
  if (statuses.includes("CREDENTIAL_STATUS_REVOKED")) return "revoked"
  if (statuses.includes("CREDENTIAL_STATUS_UNSPECIFIED")) return "missing"
  return "unavailable"
}

function exemptionCredentialLabel(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "active":
      return t.value.checkoutWizard.statusApproved
    case "pending":
      return t.value.checkoutWizard.statusPending
    case "pending_upload":
      return t.value.credentialsPage.appStatusPendingUpload
    case "resubmit":
      return t.value.checkoutWizard.statusResubmit
    case "rejected":
      return t.value.checkoutWizard.statusRejected
    case "expired":
      return t.value.checkoutWizard.statusExpired
    case "revoked":
      return t.value.checkoutWizard.statusRevoked
    case "missing":
      return t.value.checkoutWizard.statusMissing
    default:
      return t.value.checkoutWizard.statusUnavailable
  }
}

function exemptionCredentialBadgeClass(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "active":
      return "bg-emerald-100 text-emerald-800"
    case "pending":
      return "bg-blue-100 text-blue-800"
    case "pending_upload":
      return "bg-sky-100 text-sky-800"
    case "resubmit":
      return "bg-amber-100 text-amber-800"
    case "rejected":
      return "bg-rose-100 text-rose-800"
    case "expired":
      return "bg-amber-100 text-amber-800"
    case "revoked":
      return "bg-rose-100 text-rose-800"
    default:
      return "bg-slate-100 text-slate-700"
  }
}

function qualificationStatusHint(unit: any) {
  const activeOrderHint = activeOrderUnitStatusLabel(unit)
  if (activeOrderHint) return activeOrderHint
  switch (exemptionCredentialState(unit)) {
    case "active":
      return ""
    case "pending":
      return t.value.checkoutWizard.qualificationPendingHint
    case "pending_upload":
      return t.value.checkoutWizard.qualificationUploadReady
    case "resubmit":
      return t.value.checkoutWizard.qualificationResubmitHint
    default:
      return t.value.checkoutWizard.qualificationSubmitHint
  }
}

function qualificationStatusHintClass(unit: any) {
  if (activeOrderIncludesUnit(unit)) return "border-blue-200 bg-blue-50 text-blue-800"
  switch (exemptionCredentialState(unit)) {
    case "pending":
      return "border-blue-200 bg-blue-50 text-blue-800"
    case "pending_upload":
      return "border-sky-200 bg-sky-50 text-sky-800"
    case "resubmit":
      return "border-amber-200 bg-amber-50 text-amber-800"
    default:
      return "border-slate-200 bg-white text-slate-700"
  }
}

async function nextFromStep2() {
  if (!isMembershipBundle.value && !formData.agreement) {
    toast.error(t.value.examSignup.agreementRequired)
    return
  }
  sanitizeSignupForm()
  Object.assign(formData, normalizeLocationForSubmission(
    selectedCountryCode.value,
    formData.country,
    formData.province,
    formData.city,
  ))
  const requiredFields = [
    ["first_name", t.value.examSignup.formFirstName],
    ["last_name", t.value.examSignup.formLastName],
    ["email", t.value.examSignup.formEmail],
    ["gender", t.value.examSignup.formGender],
    ["birthdate", t.value.examSignup.formBirthdate],
    ["country", t.value.examSignup.formCountry],
    ...(showProvinceField.value ? [["province", t.value.examSignup.formProvince] as const] : []),
    ...(showCityField.value ? [["city", t.value.examSignup.formCity] as const] : []),
    ["address", t.value.examSignup.formAddress],
    ["postal_code", t.value.examSignup.formPostalCode],
  ] as const
  for (const [key, label] of requiredFields) {
    if (!String(formData[key as keyof typeof formData]).trim()) {
      toast.error(t.value.examSignup.validationRequired.replace("{{field}}", label))
      return
    }
  }
  if (!isValidEmail(formData.email)) {
    toast.error(t.value.examSignup.validationInvalidEmail)
    return
  }
  if (!isValidInternationalPhone(formData.phone)) {
    toast.error(t.value.examSignup.validationInvalidPhone.replace("{{field}}", t.value.examSignup.formWorkPhone))
    return
  }
  if (!isValidPostalCode(formData.postal_code, true)) {
    toast.error(t.value.examSignup.validationInvalidPostalCode)
    return
  }
  loading.value = true
  try {
    await syncSignupToProfile()
    currentStep.value = 3
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function normalizedOrderStatus(status: unknown) {
  const value = String(status || "").trim().toUpperCase()
  switch (value) {
    case "1":
      return "PENDING_CREATE"
    case "2":
      return "PENDING_PAYMENT"
    case "3":
      return "COMPLETED"
    case "4":
      return "CANCELLED"
    case "5":
      return "FAILED"
    default:
      return value
  }
}

function isCompletedStatus(status: unknown) {
  return normalizedOrderStatus(status).includes("COMPLETED")
}

function isFailedStatus(status: unknown) {
  const value = normalizedOrderStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
}

function getEligibility(response: any) {
  return response?.purchase_state?.eligibility || response?.eligibility || {}
}

const hardEligibilityBlockers = computed<EligibilityBlocker[]>(() => {
  const blockers = getEligibility(bundleData.value)?.blockers
  if (!Array.isArray(blockers)) return []
  return blockers.filter((blocker: EligibilityBlocker) => HARD_ELIGIBILITY_BLOCKER_TYPES.has(String(blocker?.blocker_type || "")))
})

const checkoutHardBlocked = computed(() => hardEligibilityBlockers.value.length > 0)

function eligibilityBlockerMessage(blocker?: EligibilityBlocker) {
  const copy = t.value.purchaseDialog
  if (!blocker) return t.value.checkoutWizard.purchaseUnavailable
  if (blocker?.blocker_type === "MISSING_PREREQUISITE_QUALIFICATION") return copy.missingQualification
  if (blocker?.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return copy.missingMembershipQualification
  if (blocker?.blocker_type === "ALREADY_PURCHASED") {
    return isMembershipBundle.value ? copy.alreadyPurchasedMembership : copy.alreadyPurchased
  }
  if (blocker?.blocker_type === "FORBIDDEN_QUALIFICATION") return copy.forbiddenQualification
  if (blocker?.blocker_type === "CONFLICT_PIPELINE_IN_PROGRESS") return copy.conflictPipelineInProgress
  if (blocker?.blocker_type === "CONFLICT_CHECK_UNAVAILABLE") return copy.conflictCheckUnavailable
  if (blocker?.blocker_type === "EXEMPTION_DOCUMENTS_PENDING_UPLOAD") return copy.exemptionDocumentsPendingUpload
  if (blocker?.blocker_type === "EXEMPTION_UNDER_REVIEW") return copy.exemptionUnderReview
  return blocker?.description || copy.unknownBlocker || t.value.checkoutWizard.purchaseUnavailable
}

async function createPurchaseOrder() {
  const response = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/purchase`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      payment_mode: paymentMode.value,
      selected_exemptions_json: buildSelectedExemptionsJson(),
    }),
  })
  const orderId = String(response?.bundle_order_ulid || response?.order_id || "").trim()
  const orderStatus = response?.order_status || response?.status

  if (isFailedStatus(orderStatus)) {
    throw new Error(response?.message || t.value.checkoutWizard.orderCreationFailed)
  }
  if (!orderId) {
    throw new Error(t.value.checkoutWizard.orderCreationFailed)
  }

  if (isCompletedStatus(orderStatus)) {
    toast.success(t.value.checkoutWizard.purchaseCompleted)
    await router.push(`/checkout/success/${encodeURIComponent(orderId)}`)
    return
  }

  activeOrderAction.value = "purchase"
  activeOrderId.value = orderId
  currentStep.value = 4
}

async function confirmAndPay() {
  loading.value = true
  try {
    const latestBundle = await loadPurchaseReadyBundleInfo()
    const eligibility = getEligibility(latestBundle)

    if (eligibility?.can_purchase) {
      await createPurchaseOrder()
      return
    }
    if (latestBundle?.purchase_state?.active_order) {
      toast.info(t.value.checkoutWizard.continueExistingOrder)
      await router.push("/orders")
      return
    }

    throw new Error(eligibilityBlockerMessage(hardEligibilityBlockers.value[0]))
  } catch (err) {
    console.error(err)
    toast.error(err instanceof Error && err.message
      ? err.message
      : t.value.checkoutWizard.orderCreationFailed)
  } finally {
    loading.value = false
  }
}

async function cancelPaymentOrderAndReturn() {
  const orderId = activeOrderId.value.trim()
  if (!orderId || cancellingPaymentOrder.value) return

  const action = activeOrderAction.value
  cancellingPaymentOrder.value = true
  try {
    const response = await apiClient("/api/orders/cancel", {
      method: "POST",
      body: JSON.stringify({
        biz_type: paymentBizType.value,
        biz_ref_ulid: orderId,
      }),
    })
    if (response?.success === false) throw new Error(response?.message || t.value.checkoutWizard.cancelPaymentOrderFailed)

    activeOrderId.value = ""
    if (action === "credential_application") {
      activeCredentialQualIds.value = []
      activeCredentialUnitIds.value = []
      currentStep.value = 1
    } else {
      currentStep.value = paymentReturnStep.value === 1 && exemptionStages.value.length === 0
        ? 2
        : paymentReturnStep.value
    }
    activeOrderAction.value = "purchase"
    paymentEditDialogOpen.value = false
    toast.success(t.value.checkoutWizard.cancelPaymentOrderSuccess)
  } catch (error) {
    console.error(error)
    toast.error(error instanceof Error && error.message
      ? error.message
      : t.value.checkoutWizard.cancelPaymentOrderFailed)
  } finally {
    cancellingPaymentOrder.value = false
  }
}

function checkoutStepLabel(step: number) {
  const labels = [
    t.value.checkoutWizard.step1,
    t.value.checkoutWizard.step2,
    t.value.checkoutWizard.step3,
    t.value.checkoutWizard.step4,
  ]
  return String(labels[step - 1] || "").replace(/^\d+\s*/, "")
}

function requestPaymentStepEdit(step: number) {
  if (currentStep.value !== 4 || step < 1 || step > 3) return
  paymentReturnStep.value = step
  paymentEditDialogOpen.value = true
}

function closePaymentEditDialog() {
  if (cancellingPaymentOrder.value) return
  paymentEditDialogOpen.value = false
}
</script>

<template>
  <AppShell content-class="p-0">
    <div class="checkout-page page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <ClipboardList class="mr-4 h-4 w-4 text-slate-700" />
        <span
          class="checkout-header-title text-sm font-medium text-foreground"
          :aria-busy="initialLoading"
        >
          <span v-if="initialLoading" class="checkout-header-title-skeleton" aria-hidden="true"></span>
          <span v-else>{{ registrationTitle }}</span>
        </span>
      </header>

      <main class="checkout-content px-5 py-8 md:px-8 lg:px-10">
        <div class="checkout-heading mb-8 max-w-5xl">
          <h1
            class="checkout-page-title text-3xl font-bold tracking-tight text-foreground"
            :aria-busy="initialLoading"
          >
            <span v-if="initialLoading" class="checkout-title-skeleton" aria-hidden="true"></span>
            <span v-else>{{ registrationTitle }}</span>
          </h1>
          <div class="checkout-progress" aria-label="Checkout progress">
            <button
              v-for="step in 4"
              :key="step"
              :data-testid="`checkout-progress-step-${step}`"
              class="checkout-progress-step"
              :class="{ active: currentStep === step, actionable: currentStep === 4 && step < 4 }"
              type="button"
              :disabled="currentStep !== 4 || step === 4 || cancellingPaymentOrder"
              :aria-current="currentStep === step ? 'step' : undefined"
              @click="requestPaymentStepEdit(step)"
            >
              <span class="checkout-progress-node">{{ step }}</span>
              <span class="checkout-progress-label">{{ checkoutStepLabel(step) }}</span>
            </button>
          </div>
        </div>
        
        <LoadingState
          v-if="initialLoading"
          class="checkout-loading-state"
          :label="t.common.loading"
          variant="section"
          :rows="4"
        />

        <template v-else>
        <div class="checkout-card max-w-5xl rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div
            v-if="checkoutHardBlocked"
            data-testid="checkout-eligibility-blockers"
            class="mb-6 flex gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-950"
            role="alert"
          >
            <CircleAlert class="mt-0.5 h-5 w-5 shrink-0" />
            <div class="min-w-0">
              <h2 class="text-sm font-semibold">{{ t.purchaseDialog.blockersTitle }}</h2>
              <ul class="mt-2 space-y-1 text-sm">
                <li v-for="(blocker, index) in hardEligibilityBlockers" :key="`${blocker.blocker_type || 'blocker'}-${index}`">
                  {{ eligibilityBlockerMessage(blocker) }}
                </li>
              </ul>
            </div>
          </div>
          <!-- Step 1: Selection -->
          <div v-if="currentStep === 1" data-testid="checkout-step-selection" class="checkout-step-one space-y-8">
            <div class="checkout-step-one-title mb-4">
              <h2 class="text-2xl font-bold">{{ t.checkoutWizard.yourLevel1Paper.replace(levelPlaceholder, "1") }}</h2>
            </div>
            
            <div v-if="exemptionStages.length > 0" class="checkout-stage-list space-y-6">
              <div v-for="stage in exemptionStages" :key="stage.stage_id || stage.index" class="checkout-stage space-y-6">
                <div class="checkout-unit-grid grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div
                    v-for="unit in stage.units"
                    :key="unit.unit_id"
                    :class="[
                      'checkout-unit-card group relative flex flex-col justify-between overflow-hidden rounded-2xl border p-5 transition-all duration-300',
                      isQualificationEditorExpanded(unit.unit_id) ? 'md:col-span-2' : '',
                      selectedExemptionUnitIds[unit.unit_id]
                        ? 'border-emerald-400 bg-emerald-50/40 shadow-md ring-1 ring-emerald-400'
                        : unit.qualified
                          ? 'cursor-pointer border-border hover:border-emerald-200 hover:shadow-sm'
                          : 'cursor-pointer border-border bg-slate-50/70 hover:border-blue-200 hover:shadow-sm',
                    ]"
                  >
                    <div class="checkout-unit-main mb-4">
                      <div class="checkout-unit-id mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{{ unit.unit_id }}</div>
                      <h3 class="checkout-unit-title text-xl font-bold text-slate-800">{{ unit.unit_name || unit.unit_id }}</h3>
                      <p v-if="unit.exemption_quals?.[0]?.description" class="checkout-unit-description mt-2 text-sm text-slate-500">{{ unit.exemption_quals[0].description }}</p>
                      
                      <div :class="['checkout-unit-badge mt-3 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium', exemptionCredentialBadgeClass(unit)]">
                        <CheckCircle2 v-if="exemptionCredentialState(unit) === 'active'" class="mr-1 h-3.5 w-3.5" />
                        <Clock v-else-if="['pending', 'expired'].includes(exemptionCredentialState(unit))" class="mr-1 h-3.5 w-3.5" />
                        <CircleAlert v-else class="mr-1 h-3.5 w-3.5" />
                        {{ exemptionCredentialLabel(unit) }}
                      </div>

                      <div
                        v-if="qualificationStatusHint(unit)"
                        :class="['mt-3 flex items-start gap-2 rounded-xl border px-3 py-2.5 text-sm leading-5', qualificationStatusHintClass(unit)]"
                      >
                        <Clock
                          v-if="['pending', 'pending_upload'].includes(exemptionCredentialState(unit))"
                          class="mt-0.5 h-4 w-4 shrink-0"
                        />
                        <CircleAlert
                          v-else-if="exemptionCredentialState(unit) === 'resubmit'"
                          class="mt-0.5 h-4 w-4 shrink-0"
                        />
                        <UploadCloud v-else class="mt-0.5 h-4 w-4 shrink-0" />
                        <span>{{ qualificationStatusHint(unit) }}</span>
                      </div>
                    </div>
                    
                    <div class="checkout-unit-footer mt-auto border-t border-slate-100 pt-4">
                      <div v-if="activeOrderLocksDecisionForUnit(unit)" class="flex items-center justify-between gap-3">
                        <span class="checkout-unit-action font-medium text-slate-700">
                          {{ activeOrderUnitStatusLabel(unit) }}
                        </span>
                      </div>
                      <div v-else-if="['active', 'pending', 'pending_upload', 'resubmit'].includes(exemptionCredentialState(unit))" class="flex items-center justify-between gap-3">
                        <span class="checkout-unit-action font-medium text-slate-700">
                          {{ exemptionCredentialState(unit) === 'active'
                            ? t.checkoutWizard.automaticExemptionApplied
                            : exemptionCredentialState(unit) === 'pending'
                              ? t.checkoutWizard.qualificationUnderReview
                              : exemptionCredentialState(unit) === 'pending_upload'
                                ? t.credentialsPage.uploadMaterials
                                : t.checkoutWizard.statusResubmit }}
                        </span>
                      </div>
                      <div v-else class="grid gap-2 sm:grid-cols-2" role="group" :aria-label="t.checkoutWizard.exemptionDecisionLabel">
                        <button
                          type="button"
                          data-testid="checkout-exemption-apply"
                          :data-unit-id="unit.unit_id"
                          :class="[
                            'btn min-h-11 justify-center rounded-lg border px-3 text-sm',
                            exemptionDecision(unit) === 'exempt'
                              ? 'border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700'
                              : 'border-slate-300 bg-white text-slate-700 hover:border-emerald-500 hover:text-emerald-700',
                          ]"
                          @click="setExemptionDecision(unit, 'exempt')"
                        >
                          <CheckCircle2 class="h-4 w-4" />
                          {{ t.checkoutWizard.applyExemptionDecision }}
                        </button>
                        <button
                          type="button"
                          data-testid="checkout-exemption-waive"
                          :data-unit-id="unit.unit_id"
                          :class="[
                            'btn min-h-11 justify-center rounded-lg border px-3 text-sm',
                            exemptionDecision(unit) === 'waive'
                              ? 'border-blue-600 bg-blue-600 text-white hover:bg-blue-700'
                              : 'border-slate-300 bg-white text-slate-700 hover:border-blue-500 hover:text-blue-700',
                          ]"
                          @click="setExemptionDecision(unit, 'waive')"
                        >
                          {{ t.checkoutWizard.waiveExemptionDecision }}
                        </button>
                      </div>
                      <div class="mt-3 flex items-center justify-end">
                        <span v-if="selectedExemptionUnitIds[unit.unit_id]" class="checkout-unit-selected-price">
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.exemptionAmount || 0, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </span>
                        <strong v-else-if="unitPriceDisplay[unit.unit_id]?.accessAmount !== undefined" class="checkout-unit-default-price">
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.accessAmount, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </strong>
                      </div>
                    </div>

                    <div
                      v-if="isQualificationEditorExpanded(unit.unit_id) && !unit.qualified"
                      class="mt-5 border-t border-blue-100 pt-5"
                    >
                      <div
                        v-if="qualificationIdsForUnit(unit).length > 1"
                        class="mb-4 rounded-xl border border-blue-100 bg-white p-4"
                      >
                        <label :for="`exemption-qualification-${unit.unit_id}`" class="text-sm font-semibold text-slate-800">
                          {{ t.checkoutWizard.exemptionQualificationLabel }}
                        </label>
                        <select
                          :id="`exemption-qualification-${unit.unit_id}`"
                          data-testid="checkout-exemption-qualification-select"
                          :data-unit-id="unit.unit_id"
                          class="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm text-slate-900"
                          :value="selectedQualificationIdForUnit(unit)"
                          @change="selectQualificationForUnit(unit, ($event.target as HTMLSelectElement).value)"
                        >
                          <option value="">{{ t.checkoutWizard.exemptionQualificationPlaceholder }}</option>
                          <option
                            v-for="qualification in qualificationApplicationEntriesForUnit(unit)"
                            :key="qualification.qualId"
                            :value="qualification.qualId"
                          >
                            {{ qualification.definition?.name || qualification.option?.name || qualification.qualId }}
                          </option>
                        </select>
                        <p class="mt-2 text-xs leading-5 text-slate-500">
                          {{ t.checkoutWizard.exemptionQualificationHelp }}
                        </p>
                      </div>
                      <div class="rounded-2xl border border-blue-100 bg-blue-50/70 p-4 sm:p-5">
                        <div
                          v-if="!selectedQualificationIdForUnit(unit)"
                          class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900"
                        >
                          {{ t.checkoutWizard.exemptionQualificationRequired }}
                        </div>
                        <div class="flex items-start gap-3">
                          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white text-blue-600 shadow-sm">
                            <UploadCloud class="h-5 w-5" />
                          </div>
                          <div>
                            <h4 class="font-semibold text-slate-900">
                              {{ qualificationDefinitionForUnit(unit)?.name || t.credentialsPage.uploadMaterials }}
                            </h4>
                            <p class="mt-1 text-sm leading-6 text-slate-600">
                              {{ qualificationDefinitionForUnit(unit)?.description || t.credentialsPage?.description }}
                            </p>
                          </div>
                        </div>

                        <CredentialAttachmentList
                          :attachments="qualificationDefinitionForUnit(unit)?.attachments"
                          class="mt-5 border-t border-blue-100 pt-5"
                        />

                        <div v-if="Array.isArray(qualificationDefinitionForUnit(unit)?.file_constraints)" class="mt-5 grid gap-4 sm:grid-cols-2">
                          <div
                            v-for="constraint in qualificationDefinitionForUnit(unit)?.file_constraints || []"
                            :key="constraint.name"
                            class="rounded-xl border border-white bg-white p-4 shadow-sm"
                          >
                            <div class="flex items-center gap-1 text-sm font-semibold text-slate-800">
                              <span v-if="constraint.is_required" class="text-rose-500">*</span>
                              <span>{{ qualificationConstraintDisplayName(constraint) }}</span>
                            </div>
                            <p class="mt-1 text-xs text-slate-500">{{ qualificationFormatHint(constraint) }}</p>
                            <div v-if="canUploadQualificationForUnit(unit)" class="mt-3 flex flex-wrap items-center gap-3">
                              <button
                                type="button"
                                class="btn btn-outline h-9 rounded-lg px-3 text-xs"
                                :disabled="Boolean(qualificationUploadingKey) || qualificationSubmittingUnitId === unit.unit_id"
                                @click="triggerQualificationFileInput(unit.unit_id, constraint.name)"
                              >
                                <Loader2
                                  v-if="qualificationUploadingKey === `${unit.unit_id}:${constraint.name}`"
                                  class="h-4 w-4 animate-spin"
                                />
                                <UploadCloud v-else class="h-4 w-4" />
                                {{ t.credentialsPage.chooseFile }}
                              </button>
                              <span
                                class="max-w-[260px] truncate text-sm text-slate-500"
                                :title="qualificationFilesForUnit(unit.unit_id)[constraint.name]?.name || ''"
                              >
                                {{ qualificationFilesForUnit(unit.unit_id)[constraint.name]?.name || t.credentialsPage.noFileChosen }}
                              </span>
                              <input
                                :id="qualificationConstraintInputId(unit.unit_id, constraint.name)"
                                type="file"
                                class="hidden"
                                :accept="getFileConstraintInfo(constraint.type).acceptStr"
                                @change="onQualificationFileChange($event, unit, constraint)"
                              />
                            </div>
                            <p
                              v-if="qualificationFilesForUnit(unit.unit_id)[constraint.name]"
                              class="mt-3 flex items-center gap-1 text-xs font-medium text-emerald-600"
                            >
                              <CheckCircle2 class="h-3.5 w-3.5" />
                              {{ qualificationUploadSuccessText(qualificationFilesForUnit(unit.unit_id)[constraint.name].name) }}
                            </p>
                          </div>
                        </div>

                        <div v-if="canUploadQualificationForUnit(unit)" class="mt-5 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
                          <button
                            type="button"
                            class="btn btn-outline"
                            :disabled="qualificationSubmittingUnitId === unit.unit_id"
                            @click="closeQualificationEditor(unit.unit_id)"
                          >
                            {{ t.common.cancel }}
                          </button>
                          <button
                            type="button"
                            class="btn bg-emerald-600 text-white hover:bg-emerald-700"
                            :disabled="Boolean(qualificationUploadingKey) || qualificationSubmittingUnitId === unit.unit_id"
                            @click="submitQualificationApplication(unit)"
                          >
                            <Loader2 v-if="qualificationSubmittingUnitId === unit.unit_id" class="h-4 w-4 animate-spin" />
                            {{ qualificationSubmittingUnitId === unit.unit_id
                              ? t.credentialsPage.submitting
                              : t.credentialsPage.submitApplication }}
                          </button>
                        </div>
                        <div v-if="!canUploadQualificationForUnit(unit)" class="mt-5 rounded-xl border border-blue-200 bg-white px-4 py-3 text-sm leading-6 text-blue-900">
                          {{ t.checkoutWizard.uploadAfterPaymentHint }}
                        </div>
                        <div v-if="canRequestQualificationApplicationForUnit(unit)" class="mt-5 flex justify-end">
                          <button
                            type="button"
                            data-testid="checkout-create-qualification-order"
                            :data-unit-id="unit.unit_id"
                            class="btn min-h-11 rounded-lg bg-emerald-600 px-5 text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
                            :disabled="credentialApplicationOrderLoading"
                            @click="requestSelectedQualificationApplications(unit)"
                          >
                            <Loader2 v-if="credentialApplicationOrderLoading && qualificationOrderTargetUnitId === unit.unit_id" class="h-4 w-4 animate-spin" />
                            {{ qualificationApplicationActionLabel(unit) }}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="bundleData" class="checkout-step-actions mt-6 flex flex-col items-stretch">
                <section
                  v-if="purchaseLineItems.length > 0"
                  class="checkout-included-items"
                  data-testid="checkout-included-items"
                >
                  <h3 class="checkout-included-items-title">{{ t.checkoutWizard.includedItems }}</h3>
                  <div class="checkout-included-items-list">
                    <div
                      v-for="item in purchaseLineItems"
                      :key="item.key"
                      class="checkout-included-item"
                      data-testid="checkout-included-item"
                      :data-item-id="item.itemId"
                    >
                      <div class="min-w-0">
                        <div class="checkout-included-item-name">{{ item.name }}</div>
                        <div v-if="item.stageName" class="checkout-included-item-stage">{{ item.stageName }}</div>
                      </div>
                      <div class="checkout-included-item-price">{{ formatMoney(item.amount, item.currency) }}</div>
                    </div>
                  </div>
                </section>
                <div class="checkout-total text-lg font-bold text-slate-900">
                  <template v-if="dynamicPaymentPreview">
                    {{ t.checkoutWizard.baseTotal }} {{ formatMoney(dynamicPaymentPreview.total, dynamicPaymentPreview.currency) }}
                  </template>
                </div>
              </div>
            </div>
          </div>

          <!-- Step 2: Registration -->
          <form id="checkout-registration-form" v-if="currentStep === 2" data-testid="checkout-step-registration" class="space-y-6" novalidate @submit.prevent="nextFromStep2">
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formFirstName }}</span><input v-model="formData.first_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formLastName }}</span><input v-model="formData.last_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
            </div>
            <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formMiddleName }}</span><input v-model="formData.middle_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" /></label>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formEmail }}</span><input v-model="formData.email" class="input" type="email" :maxlength="PROFILE_TEXT_LIMITS.short" required /></label>
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formGender }}</span>
                <select v-model="formData.gender" class="input cursor-pointer" required>
                  <option value="" disabled>{{ t.examSignup.formGender }}</option>
                  <option v-for="option in genderOptions" :key="option" :value="option">{{ t.common.genderOptions[option] }}</option>
                </select>
              </label>
            </div>
            <label class="block space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formBirthdate }}</span>
              <LocalizedDatePicker
                v-model="formData.birthdate"
                :language="lang"
                :placeholder="lang === 'zh' ? '日/月/年' : 'DD/MM/YYYY'"
                :aria-label="t.examSignup.formBirthdate"
              />
            </label>
            <div class="grid gap-4" :class="locationGridClass">
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formCountry }}</span>
                <select v-model="selectedCountryCode" class="input cursor-pointer" required @change="handleCountryChange">
                  <option value="" disabled>{{ t.examSignup.formCountry }}</option>
                  <option v-for="country in countryOptions" :key="country.code" :value="country.code">{{ country.displayName }}</option>
                </select>
              </label>
              <label v-if="showProvinceField" class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formProvince }}</span>
                <select v-if="provinceOptions.length > 0" v-model="selectedProvinceCode" class="input cursor-pointer" required @change="handleProvinceChange">
                  <option value="" disabled>{{ t.examSignup.formProvince }}</option>
                  <option v-for="province in provinceOptions" :key="province.isoCode" :value="province.isoCode">{{ localizedProvinceName(province) }}</option>
                </select>
                <input v-else v-model="formData.province" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" required />
              </label>
              <label v-if="showCityField" class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formCity }}</span>
                <select v-if="cityOptions.length > 0" v-model="formData.city" class="input cursor-pointer" required>
                  <option value="" disabled>{{ t.examSignup.formCity }}</option>
                  <option v-for="city in cityOptions" :key="`${city.name}-${city.latitude}-${city.longitude}`" :value="localizedCityName(city)">{{ localizedCityName(city) }}</option>
                </select>
                <input v-else v-model="formData.city" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" required />
              </label>
            </div>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formAddress }}</span><input v-model="formData.address" class="input" :maxlength="PROFILE_TEXT_LIMITS.address" required /></label>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formPostalCode }}</span><input v-model="formData.postal_code" class="input" :maxlength="PROFILE_TEXT_LIMITS.postalCode" pattern="[A-Za-z0-9][A-Za-z0-9 -]*[A-Za-z0-9]" required @blur="formData.postal_code = normalizePostalCode(formData.postal_code)" /></label>
            
            <div class="grid gap-4 sm:grid-cols-1">
              <label class="space-y-2">
                <span class="text-sm font-medium">{{ t.examSignup.formWorkPhone }}</span>
                <div class="flex gap-2">
                  <select v-if="orgPhonePrefixes.length > 0" v-model="formData.phone_country_code" class="input cursor-pointer w-28 shrink-0">
                    <option v-for="prefix in orgPhonePrefixes" :key="prefix.code" :value="prefix.code">{{ prefix.dialCode }} · {{ prefix.name }}</option>
                  </select>
                  <input
                    id="exam-signup-work-phone"
                    v-model="formData.phone"
                    class="input flex-1"
                    type="tel"
                    inputmode="tel"
                    autocomplete="tel"
                    maxlength="24"
                    :placeholder="t.examSignup.formWorkPhonePlaceholder"
                  />
                </div>
              </label>
            </div>

            <div v-if="!isMembershipBundle" class="mt-6 border-t border-border pt-6">
              <label class="flex items-center gap-3">
                <input v-model="formData.agreement" data-testid="checkout-agreement" type="checkbox" class="h-4 w-4 shrink-0 rounded border-gray-300 text-emerald-600 focus:ring-emerald-500" :required="!isMembershipBundle" />
                <span class="text-sm font-medium text-foreground">{{ t.examSignup.agreement }}</span>
              </label>
            </div>

          </form>

          <!-- Step 3: Review -->
          <div v-if="currentStep === 3" data-testid="checkout-step-review" class="space-y-6">
            <h2 class="text-xl font-semibold">{{ t.checkoutWizard.review }}</h2>
            <div class="rounded-lg border border-border p-4 text-sm space-y-2">
              <div class="grid grid-cols-3 gap-2">
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewName }}</div>
                <div class="col-span-2 font-medium">{{ formData.first_name }} {{ formData.last_name }}</div>
                
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewEmail }}</div>
                <div class="col-span-2 font-medium">{{ formData.email }}</div>
                
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewLocation }}</div>
                <div class="col-span-2 font-medium">{{ formData.city }}, {{ formData.province }}, {{ formData.country }}</div>
              </div>
            </div>

            <!-- PAYMENT MODE SELECTION -->
            <div v-if="isMultiStage" class="rounded-lg border border-border p-4 text-sm space-y-4">
              <div class="mb-2 text-sm font-semibold">{{ t.checkoutWizard.paymentModeTitle }}</div>
              
              <label class="flex items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-slate-50 cursor-pointer" :class="{ 'border-emerald-500 bg-emerald-50/30': paymentMode === 'FULL_PIPELINE', 'border-border': paymentMode !== 'FULL_PIPELINE' }">
                <input type="radio" v-model="paymentMode" value="FULL_PIPELINE" class="mt-1 h-4 w-4 text-emerald-600 focus:ring-emerald-500" />
                <div>
                  <div class="font-medium text-slate-900">{{ t.checkoutWizard.modeFullPipeline }}</div>
                  <div class="text-xs text-slate-500 mt-1">{{ t.checkoutWizard.modeFullPipelineDesc }}</div>
                </div>
              </label>

              <label class="flex items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-slate-50 cursor-pointer" :class="{ 'border-emerald-500 bg-emerald-50/30': paymentMode === 'BY_STAGE', 'border-border': paymentMode !== 'BY_STAGE' }">
                <input type="radio" v-model="paymentMode" value="BY_STAGE" class="mt-1 h-4 w-4 text-emerald-600 focus:ring-emerald-500" />
                <div>
                  <div class="font-medium text-slate-900">{{ t.checkoutWizard.modeByStage }}</div>
                  <div class="text-xs text-slate-500 mt-1">{{ t.checkoutWizard.modeByStageDesc }}</div>
                </div>
              </label>
            </div>

              <div v-if="dynamicPaymentPreview" class="rounded-lg bg-muted/30 p-4 border border-border">
              <div class="mb-3 text-sm font-semibold">{{ t.checkoutWizard.priceSummary }}</div>
              <div class="space-y-2 text-sm">
                <div class="flex justify-between">
                  <span class="text-muted-foreground">{{ t.checkoutWizard.subtotal }}</span>
                  <span class="font-medium">{{ dynamicPaymentPreview.amount_label || formatMoney(dynamicPaymentPreview.subtotal, dynamicPaymentPreview.currency) }}</span>
                </div>
                <div v-if="dynamicPaymentPreview.discount_total" class="flex justify-between">
                  <span class="text-muted-foreground">{{ t.checkoutWizard.discount }}</span>
                  <span class="font-medium">-{{ formatMoney(dynamicPaymentPreview.discount_total, dynamicPaymentPreview.currency) }}</span>
                </div>
                <div class="mt-2 flex justify-between border-t border-border pt-2">
                  <span class="font-semibold text-foreground">{{ t.checkoutWizard.total }}</span>
                  <span class="text-lg font-bold text-foreground">{{ dynamicPaymentPreview.pay_amount_label || formatMoney(dynamicPaymentPreview.total, dynamicPaymentPreview.currency) }}</span>
                </div>
              </div>
            </div>

          </div>

          <!-- Step 4: Payment -->
          <div v-if="currentStep === 4" data-testid="checkout-step-payment" class="space-y-6">
            <div>
              <h2 class="text-xl font-semibold">
                {{ activeOrderAction === "credential_application"
                  ? t.checkoutWizard.qualificationPaymentTitle
                  : t.checkoutWizard.payment }}
              </h2>
              <p v-if="activeOrderAction === 'credential_application'" class="mt-2 text-sm text-muted-foreground">
                {{ t.checkoutWizard.qualificationPaymentDesc }}
              </p>
            </div>
            <CheckoutPaymentPanel
              v-if="activeOrderId"
              :biz-type="paymentBizType"
              :biz-ref-ulid="activeOrderId"
              :order-id="activeOrderId"
              :source="activeOrderAction"
              :return-path="paymentReturnPath"
              :extra-return-params="paymentReturnParams"
              min-height-class="min-h-[420px]"
            />
          </div>
        </div>

        <div
          v-if="currentStep === 1 && exemptionStages.length > 0 && bundleData"
          class="checkout-step-footer"
        >
          <button
            data-testid="checkout-selection-next"
            class="checkout-next-button btn rounded-full px-8 py-3 text-white disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="checkoutHardBlocked || hasExpandedQualificationEditors || Boolean(qualificationSubmittingUnitId)"
            @click="nextFromStep1"
          >
            {{ t.checkoutWizard.saveAndContinue }}
          </button>
        </div>

        <div v-else-if="currentStep === 2" class="checkout-step-footer checkout-form-actions flex items-center">
          <button v-if="exemptionStages.length > 0" type="button" class="checkout-back-button btn btn-outline" @click="currentStep = 1">
            <ArrowLeft class="h-4 w-4" />
            {{ t.checkoutWizard.back }}
          </button>
          <button form="checkout-registration-form" data-testid="checkout-next" type="submit" class="checkout-form-next-button btn rounded-full px-8 text-white" :disabled="loading || checkoutHardBlocked">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.examSignup.submitting }}</template>
            <template v-else>{{ t.checkoutWizard.next }} <Send class="ml-2 h-4 w-4" /></template>
          </button>
        </div>

        <div v-else-if="currentStep === 3" class="checkout-step-footer checkout-review-actions flex items-center">
          <button type="button" class="checkout-back-button btn btn-outline" @click="currentStep = 2" :disabled="loading">
            <ArrowLeft class="h-4 w-4" />
            {{ t.checkoutWizard.back }}
          </button>
          <button data-testid="checkout-confirm-pay" class="checkout-form-next-button btn btn-vivid" :disabled="loading || checkoutHardBlocked" @click="confirmAndPay">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.checkoutWizard.processing }}</template>
            <template v-else>{{ t.checkoutWizard.confirmAndPay }} <ArrowRight class="h-4 w-4" /></template>
          </button>
        </div>
        </template>
      </main>

      <Teleport to="body">
        <div v-if="qualificationOrderConfirmDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4">
          <section
            v-modal-dialog="closeQualificationOrderConfirmDialog"
            data-testid="checkout-qualification-order-confirm-dialog"
            class="w-full max-w-[620px] overflow-hidden rounded-lg bg-white shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-labelledby="checkout-qualification-order-confirm-title"
          >
            <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
              <div class="flex min-w-0 items-start gap-3">
                <CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
                <div>
                  <h2 id="checkout-qualification-order-confirm-title" class="text-lg font-black text-slate-950">
                    {{ t.checkoutWizard.qualificationOrderConfirmTitle }}
                  </h2>
                  <p class="mt-2 text-sm leading-6 text-slate-600">
                    {{ t.checkoutWizard.qualificationOrderConfirmDescription }}
                  </p>
                </div>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 hover:bg-slate-50"
                type="button"
                :aria-label="t.common.close"
                :disabled="credentialApplicationOrderLoading"
                @click="closeQualificationOrderConfirmDialog"
              >
                <X class="h-4 w-4" />
              </button>
            </header>

            <div class="space-y-4 px-5 py-5">
              <div>
                <h3 class="text-sm font-bold text-slate-900">{{ t.checkoutWizard.qualificationOrderConfirmItemsTitle }}</h3>
                <ul class="mt-3 divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
                  <li
                    v-for="item in qualificationOrderConfirmItems"
                    :key="item.unitId"
                    data-testid="checkout-qualification-order-confirm-item"
                    class="px-4 py-3"
                  >
                    <div class="font-bold text-slate-900">{{ item.unitName }}</div>
                    <div v-if="item.qualificationName" class="mt-1 text-sm text-slate-600">{{ item.qualificationName }}</div>
                  </li>
                </ul>
              </div>
              <div class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-900">
                <p class="font-bold">{{ t.checkoutWizard.qualificationOrderConfirmWarningTitle }}</p>
                <p class="mt-1">{{ t.checkoutWizard.qualificationOrderConfirmWarning }}</p>
                <p class="mt-1">{{ t.checkoutWizard.qualificationOrderConfirmNextStep }}</p>
              </div>
            </div>

            <footer class="flex flex-col-reverse gap-3 bg-slate-50 px-5 py-4 sm:flex-row sm:justify-end">
              <button
                data-testid="checkout-cancel-qualification-order"
                class="inline-flex min-h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 font-bold text-slate-700 hover:bg-slate-100"
                type="button"
                :disabled="credentialApplicationOrderLoading"
                @click="closeQualificationOrderConfirmDialog"
              >
                {{ t.common.cancel }}
              </button>
              <button
                data-testid="checkout-confirm-qualification-order"
                class="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 font-bold text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
                type="button"
                :disabled="credentialApplicationOrderLoading"
                @click="confirmSelectedQualificationApplications"
              >
                <Loader2 v-if="credentialApplicationOrderLoading" class="h-4 w-4 animate-spin" />
                {{ t.checkoutWizard.qualificationOrderConfirmAction }}
              </button>
            </footer>
          </section>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="paymentEditDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4">
          <section
            v-modal-dialog="closePaymentEditDialog"
            class="w-full max-w-[560px] overflow-hidden rounded-lg bg-white shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-labelledby="checkout-payment-edit-title"
          >
            <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
              <div class="flex min-w-0 items-start gap-3">
                <CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
                <div>
                  <h2 id="checkout-payment-edit-title" class="text-lg font-black text-slate-950">{{ t.checkoutWizard.paymentEditDialogTitle }}</h2>
                  <p class="mt-2 text-sm leading-6 text-slate-600">{{ t.checkoutWizard.paymentOrderLockedHint }}</p>
                </div>
              </div>
              <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 hover:bg-slate-50" type="button" :aria-label="t.common.close" :disabled="cancellingPaymentOrder" @click="closePaymentEditDialog">
                <X class="h-4 w-4" />
              </button>
            </header>
            <footer class="flex flex-col-reverse gap-3 bg-slate-50 px-5 py-4 sm:flex-row sm:justify-end">
              <button class="inline-flex min-h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 font-bold text-slate-700 hover:bg-slate-100" type="button" :disabled="cancellingPaymentOrder" @click="closePaymentEditDialog">
                {{ t.checkoutWizard.continuePayment }}
              </button>
              <button
                data-testid="checkout-cancel-and-edit"
                class="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg bg-red-600 px-4 font-bold text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
                type="button"
                :disabled="cancellingPaymentOrder"
                @click="cancelPaymentOrderAndReturn"
              >
                <Loader2 v-if="cancellingPaymentOrder" class="h-4 w-4 animate-spin" />
                {{ t.checkoutWizard.cancelPaymentOrderAndEdit }}
              </button>
            </footer>
          </section>
        </div>
      </Teleport>
    </div>
  </AppShell>
</template>

<style scoped>
.checkout-page {
  background-color: #edeef2 !important;
}

.checkout-content {
  width: 100%;
  max-width: 1080px;
  margin: 0 auto;
  padding: 24px 32px 64px !important;
}

.checkout-heading,
.checkout-card,
.checkout-loading-state,
.checkout-step-footer {
  width: 100%;
  max-width: none;
}

.checkout-loading-state {
  min-height: 320px;
}

.checkout-heading {
  margin-bottom: 18px;
}

.checkout-heading h1 {
  font-size: 28px;
  line-height: 1.25;
  letter-spacing: 0;
}

.checkout-page-title {
  min-height: 35px;
}

.checkout-header-title {
  display: inline-flex;
  min-height: 20px;
  align-items: center;
}

.checkout-header-title-skeleton {
  display: block;
  width: 112px;
  height: 14px;
  border-radius: 3px;
  background: #e2e7ef;
  animation: checkout-title-pulse 1.2s ease-in-out infinite;
}

.checkout-title-skeleton {
  display: block;
  width: min(220px, 60vw);
  height: 35px;
  border-radius: 4px;
  background: #e2e7ef;
  animation: checkout-title-pulse 1.2s ease-in-out infinite;
}

@keyframes checkout-title-pulse {
  0%,
  100% {
    opacity: 0.55;
  }

  50% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .checkout-header-title-skeleton,
  .checkout-title-skeleton {
    animation: none;
  }
}

.checkout-progress {
  display: grid;
  width: min(460px, 100%);
  margin: 18px auto 0;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.checkout-progress-step {
  position: relative;
  display: flex;
  min-width: 0;
  align-items: center;
  flex-direction: column;
  gap: 7px;
  padding: 0;
  border: 0;
  color: #52617a;
  background: transparent;
  font: inherit;
}

.checkout-progress-step.actionable {
  cursor: pointer;
}

.checkout-progress-step.actionable:hover .checkout-progress-node {
  background: #cbdcf8;
}

.checkout-progress-step.actionable:focus-visible {
  border-radius: 6px;
  outline: 2px solid #0957f9;
  outline-offset: 4px;
}

.checkout-progress-step:not(:last-child)::after {
  position: absolute;
  z-index: 0;
  top: 14px;
  left: calc(50% + 19px);
  width: calc(100% - 38px);
  border-top: 2px dotted #cbd8e9;
  content: "";
}

.checkout-progress-node {
  position: relative;
  z-index: 1;
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #52617a;
  background: #e4ebf5;
  font-size: 13px;
  font-weight: 700;
}

.checkout-progress-label {
  overflow-wrap: anywhere;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
  text-align: center;
}

.checkout-progress-step.active {
  color: #002a66;
}

.checkout-progress-step.active .checkout-progress-node {
  color: #fff;
  background: #0957f9;
}

.checkout-card {
  padding: 26px;
  border: 1px solid rgba(0, 42, 102, 0.16);
  border-radius: 8px;
  box-shadow: none;
}

.checkout-step-one > :not([hidden]) ~ :not([hidden]),
.checkout-stage-list > :not([hidden]) ~ :not([hidden]),
.checkout-stage > :not([hidden]) ~ :not([hidden]) {
  margin-top: 18px;
}

.checkout-step-one-title {
  margin-bottom: 0;
}

.checkout-step-one-title h2 {
  color: #002a66;
  font-size: 20px;
  line-height: 1.35;
}

.checkout-unit-grid {
  gap: 12px;
}

.checkout-unit-card {
  min-height: 132px;
  padding: 14px;
  border-color: rgba(0, 42, 102, 0.18);
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.checkout-unit-card:hover {
  border-color: #0957f9;
  box-shadow: none;
}

.checkout-unit-card.ring-1 {
  border-color: #0957f9;
  background: #fff;
  box-shadow: none;
}

.checkout-unit-main {
  margin-bottom: 7px;
}

.checkout-unit-id {
  display: none;
}

.checkout-unit-title {
  color: #002a66;
  font-size: 15px;
  line-height: 1.4;
}

.checkout-unit-description {
  display: none;
}

.checkout-unit-badge {
  margin-top: 7px;
  padding: 2px 9px;
  border-color: transparent;
  font-size: 11px;
}

.checkout-unit-badge.bg-emerald-100 {
  color: #9a6500;
  background: #fff3d8;
}

.checkout-unit-badge.bg-emerald-100 svg {
  display: none;
}

.checkout-unit-footer {
  padding-top: 5px;
  border-top: 0;
}

.checkout-unit-option {
  display: grid;
  align-items: center;
  grid-template-columns: 17px minmax(0, 1fr);
  column-gap: 8px;
  row-gap: 6px;
}

.checkout-unit-checkbox {
  width: 17px;
  height: 17px;
  border-width: 1px;
  border-radius: 3px;
}

.checkout-unit-footer input.peer:checked + .checkout-unit-checkbox {
  border-color: #0957f9;
  background: #0957f9;
}

.checkout-unit-action {
  color: var(--gfi-ink-soft, #2a4575);
  font-size: 13px;
}

.checkout-unit-default-price,
.checkout-unit-selected-price {
  grid-column: 1 / -1;
  width: fit-content;
  line-height: 1.35;
}

.checkout-unit-default-price {
  color: #002a66;
  font-size: 15px;
  font-weight: 700;
}

.checkout-unit-selected-price {
  padding: 3px 11px;
  border-radius: 999px;
  color: #078653;
  background: #e2f5eb;
  font-size: 12px;
  font-weight: 700;
}

.checkout-declaration {
  margin-top: 16px;
  padding: 14px 16px;
  border-radius: 8px;
}

.checkout-declaration input.peer:checked + div {
  border-color: #0957f9;
  background: #0957f9;
}

.checkout-step-actions {
  margin-top: 20px;
  padding: 13px 0 0;
  border: 0;
  border-top: 1px solid rgba(0, 42, 102, 0.18);
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.checkout-included-items {
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(0, 42, 102, 0.12);
}

.checkout-included-items-title {
  margin: 0 0 8px;
  color: #002a66;
  font-size: 15px;
  font-weight: 700;
}

.checkout-included-items-list {
  display: grid;
  gap: 0;
}

.checkout-included-item {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 9px 0;
  border-top: 1px solid rgba(148, 163, 184, 0.2);
}

.checkout-included-item:first-child {
  border-top: 0;
}

.checkout-included-item-name {
  overflow-wrap: anywhere;
  color: #0f294d;
  font-size: 14px;
  font-weight: 600;
}

.checkout-included-item-stage {
  margin-top: 2px;
  color: #64748b;
  font-size: 12px;
}

.checkout-included-item-price {
  flex: none;
  color: #002a66;
  font-size: 14px;
  font-weight: 700;
  white-space: nowrap;
}

.checkout-total:empty {
  display: none;
}

.checkout-total {
  margin-top: 13px;
  color: #002a66;
  font-size: 18px;
  line-height: 1.4;
  text-align: right;
}

.checkout-step-footer {
  margin-top: 18px;
}

.checkout-next-button {
  min-width: 136px;
  min-height: 44px;
  padding: 10px 24px;
  border-radius: 8px;
  background: #0957f9;
  box-shadow: none;
}

.checkout-next-button:hover {
  background: #0045d8;
}

.checkout-form-actions {
  margin-top: 22px;
  padding-top: 0;
  justify-content: flex-start;
  gap: 10px;
  border-top: 0;
}

.checkout-review-actions {
  margin-top: 22px;
  padding-top: 0;
  justify-content: flex-start;
  gap: 10px;
}

.checkout-back-button,
.checkout-form-next-button {
  min-height: 44px;
  padding: 10px 24px;
  border-radius: 8px;
  box-shadow: none;
  font-weight: 600;
}

.checkout-back-button {
  min-width: 96px;
  border-color: #002a66;
  color: #002a66;
  background: #fff;
}

.checkout-back-button:hover {
  border-color: #0957f9;
  color: #0957f9;
  background: #f3f7fc;
}

.checkout-form-next-button {
  min-width: 136px;
  background: #0957f9;
}

.checkout-form-next-button:hover {
  background: #0045d8;
}

@media (max-width: 767px) {
  .checkout-content {
    padding: 20px 16px 48px !important;
  }

  .checkout-card {
    padding: 18px;
  }

  .checkout-heading h1 {
    font-size: 25px;
  }

  .checkout-progress {
    width: 100%;
    margin-top: 16px;
  }

  .checkout-progress-step:not(:last-child)::after {
    left: calc(50% + 17px);
    width: calc(100% - 34px);
  }

  .checkout-progress-label {
    font-size: 11px;
  }

  .checkout-step-actions {
    align-items: stretch;
  }

  .checkout-included-item {
    align-items: flex-start;
    gap: 12px;
  }

  .checkout-total {
    text-align: left;
  }

  .checkout-form-actions,
  .checkout-review-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .checkout-back-button,
  .checkout-form-next-button {
    width: 100%;
  }

  .checkout-next-button {
    width: auto;
  }
}
</style>
