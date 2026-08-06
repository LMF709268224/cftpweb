<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, ArrowRight, ClipboardList, Loader2, Send, Check, CheckCircle2, CircleAlert, Clock, UploadCloud } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import LocalizedDatePicker from "@/components/LocalizedDatePicker.vue"
import LoadingState from "@/components/LoadingState.vue"
import CheckoutPaymentPanel from "@/components/CheckoutPaymentPanel.vue"
import { ApiClientError, apiClient } from "@/lib/apiClient"
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
  getProvinceOptions,
  getStateCityOptions,
  loadLocationData,
  normalizeAddressCountryCode,
  normalizeLocationForSubmission,
  type CountryOption,
} from "@/lib/locationOptions"
import { GENDER_OPTIONS, PROFILE_TEXT_LIMITS, isValidEmail, isValidInternationalPhone, isValidPostalCode, normalizeGender, normalizeInternationalPhone, normalizePostalCode, trimToMax } from "@/lib/profileFormValidation"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, statusEnumNameForStatus } from "@/lib/status-labels"
import { getFileConstraintInfo } from "@/lib/fileConstraints"
import { sha256Hex, uploadWithTimeout } from "@/lib/upload"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const { currentUser, fetchUser } = useUser()
const bundleId = String(route.params.bundleId || route.query.bundleId || "")
const TEMPORARY_IMPLICIT_UNLOCK_BUNDLE_GPATH = "/gcc/pipeline/core/cftp"
const currentStep = ref(1)
const bundleData = ref<any>(null)
const pricingDetail = ref<any>(null)
const paymentMode = ref("FULL_PIPELINE")
const paymentPreview = ref<any>(null)
const exemptionStages = ref<any[]>([])
const selectedExemptionUnitIds = ref<Record<string, boolean>>({})
const activeOrderId = ref("")
const activeOrderAction = ref<"purchase" | "unlock" | "credential_application">("purchase")
const activeCredentialQualIds = ref<string[]>([])
const activeCredentialUnitId = ref("")
const credentialApplicationLoadingUnitId = ref("")
const qualificationApplications = ref<Record<string, any>>({})
const qualificationDefinitions = ref<Record<string, any>>({})
const expandedQualificationUnitIds = ref<Record<string, boolean>>({})
const qualificationUploadedFiles = ref<Record<string, Record<string, { name: string; url: string; ext: string; hash: string; size: number }>>>({})
const qualificationUploadingKey = ref("")
const qualificationSubmittingUnitId = ref("")
const levelPlaceholder = "{" + "{level}}"
const exemptionDeclarationChecked = ref(false)
const isExemptionSelected = computed(() => Object.values(selectedExemptionUnitIds.value).some(Boolean))

function selectedExemptedUnitIds() {
  return new Set(
    Object.entries(selectedExemptionUnitIds.value)
      .filter(([, selected]) => selected)
      .map(([unitId]) => unitId),
  )
}

function includedPurchaseUnitIds() {
  const stages = [...(bundleData.value?.stages || [])]
    .sort((left: any, right: any) => Number(left?.sort_order || 0) - Number(right?.sort_order || 0))
  const exemptedUnitIds = selectedExemptedUnitIds()
  const includedStages = paymentMode.value === "BY_STAGE"
    ? stages.slice(0, stages.findIndex((stage: any) =>
      (stage?.units || []).some((unit: any) => !exemptedUnitIds.has(String(unit?.unit_id || ""))),
    ) + 1)
    : stages

  return new Set(
    includedStages.flatMap((stage: any) =>
      (stage?.units || [])
        .map((unit: any) => String(unit?.unit_id || "").trim())
        .filter(Boolean),
    ),
  )
}

const dynamicPaymentPreview = computed(() => {
  if (!pricingDetail.value) {
    return paymentPreview.value
  }

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value

    let total = 0
    let currency = "USD"
    let subtotal = 0

    // Match gmall bundle line-item selection: unlock and qualification review
    // orders are separate, while bundle purchases charge non-exempted units.
    const includedUnitIds = includedPurchaseUnitIds()
    const exemptedUnitIds = selectedExemptedUnitIds()

    // 1. Units
    if (Array.isArray(detail.units)) {
      for (const u of detail.units) {
        const unitId = String(u?.unit_id || "").trim()
        if (!includedUnitIds.has(unitId) || exemptedUnitIds.has(unitId)) continue
        if (u.access) {
          total += u.access.amount
          subtotal += u.access.amount
          if (u.access.currency) currency = u.access.currency
        }
      }
    }

    // 2. Memberships
    if (Array.isArray(detail.memberships)) {
       for (const m of detail.memberships) {
          if (m.price) {
             total += m.price.amount
             subtotal += m.price.amount
             if (m.price.currency) currency = m.price.currency
          }
       }
    }

    return {
      total,
      subtotal,
      currency,
      pay_amount_label: "",
      amount_label: "",
      discount_total: 0 // Assume no complex discounts for dynamic preview on this page if not provided
    }

  } catch (err) {
    console.error("Failed to calculate dynamic pricing", err)
    return paymentPreview.value
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
        let exemptionAmount = 0
        let hasExemptionAmount = false
        let currency = pricingUnit?.access?.currency || "USD"

        for (const qualification of unit.exemption_quals || []) {
          const qualificationId = String(qualification.qual_id || "").trim()
          const review = Array.isArray(detail.qual_reviews)
            ? detail.qual_reviews.find((item: any) => item.qual_id === qualificationId)
            : null
          if (typeof review?.price?.amount === "number") {
            exemptionAmount += review.price.amount
            hasExemptionAmount = true
            currency = review.price.currency || currency
          }
        }

        display[unit.unit_id] = {
          accessAmount: typeof pricingUnit?.access?.amount === "number" ? pricingUnit.access.amount : undefined,
          exemptionAmount: hasExemptionAmount ? exemptionAmount : undefined,
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
const initialLoading = ref(true)
const pipelineId = computed(() =>
  String(bundleData.value?.pipeline_id || bundleData.value?.pipeline_cc_ulid || "").trim()
)
const paymentBizType = computed(() => {
  if (activeOrderAction.value === "unlock") return "PIPELINE_UNLOCK"
  if (activeOrderAction.value === "credential_application") return "CREDENTIAL_APPLICATION"
  return "BUNDLE_PURCHASE"
})
const paymentReturnPath = computed(() => {
  if (activeOrderAction.value === "unlock") return "/my-certifications"
  if (activeOrderAction.value === "credential_application") return route.path
  return `/checkout/success/${activeOrderId.value}`
})
const paymentReturnParams = computed(() => {
  if (activeOrderAction.value === "credential_application") {
    return {
      qual_ulids: activeCredentialQualIds.value.join(","),
      qualification_unit_id: activeCredentialUnitId.value,
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
const orgPhonePrefixes = ref<{ code: string, dialCode: string, name: string }[]>([])
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
      const allCountries = getCachedCountries()
      orgPhonePrefixes.value = configRes.country_codes.map((code: string) => {
        const country = allCountries.find((c) => c.isoCode === code)
        return {
          code,
          dialCode: country ? `+${country.phonecode}` : code,
          name: country ? country.name : code,
        }
      })
      if (!formData.phone_country_code && orgPhonePrefixes.value.length > 0) {
        formData.phone_country_code = orgPhonePrefixes.value[0].code
      }
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
  void loadProfile()
  void loadLocationData()
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
})

async function syncSignupToProfile() {
  try {
    const current = await fetchUser(true)
    if (!current) return
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify(buildProfilePayload(current || {})),
      suppressErrorToast: true,
    })
    await fetchUser(true)
  } catch (err) {
    console.warn("Failed to sync signup form to profile", err)
  }
}

function applyBundleInfo(response: any) {
  bundleData.value = response
  const purchaseState = response?.purchase_state || response
  paymentPreview.value = purchaseState?.payment_preview || null

  const stages = purchaseState?.exemption_options?.stages || []
  exemptionStages.value = stages.filter((stage: any) => (stage.units?.length || 0) > 0)
  syncQualifiedExemptionSelections(exemptionStages.value)

  if (exemptionStages.value.length === 0 && currentStep.value === 1) {
    currentStep.value = 2
  }
}

function syncQualifiedExemptionSelections(stages: any[]) {
  const previousSelections = selectedExemptionUnitIds.value
  const nextSelections: Record<string, boolean> = {}

  for (const stage of stages) {
    for (const unit of stage.units || []) {
      const unitId = String(unit?.unit_id || "").trim()
      if (!unitId || !unit?.qualified) continue

      nextSelections[unitId] = Object.prototype.hasOwnProperty.call(previousSelections, unitId)
        ? Boolean(previousSelections[unitId])
        : true
    }
  }

  selectedExemptionUnitIds.value = nextSelections
}

async function fetchBundlePayload() {
  return apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}`, {
    suppressErrorToast: true,
  })
}

function bundlePipelineId(response: any) {
  return String(response?.pipeline_id || response?.pipeline_cc_ulid || "").trim()
}

function shouldImplicitlyUnlockCftp(response: any) {
  return String(response?.bundle_gpath || "").trim() === TEMPORARY_IMPLICIT_UNLOCK_BUNDLE_GPATH
    && Boolean(getEligibility(response)?.can_unlock)
    && Boolean(bundlePipelineId(response))
}

async function completeTemporaryCftpUnlock(response: any) {
  if (!shouldImplicitlyUnlockCftp(response)) return response

  // TEMP: Remove after gmall makes qualification-only CFtP bundles directly purchasable.
  const unlockResponse = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/unlock`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      pipeline_cc_ulid: bundlePipelineId(response),
    }),
  })
  const orderStatus = unlockResponse?.order_status || unlockResponse?.status
  if (!isCompletedStatus(orderStatus)) {
    throw new Error(t.value.checkoutWizard.implicitUnlockFailed)
  }

  const refreshedBundle = await fetchBundlePayload()
  if (!getEligibility(refreshedBundle)?.can_purchase) {
    throw new Error(t.value.checkoutWizard.implicitUnlockFailed)
  }
  return refreshedBundle
}

async function loadBundleInfo() {
  const response = await fetchBundlePayload()
  applyBundleInfo(response)
  await refreshQualificationApplications()
  return response
}

async function loadPurchaseReadyBundleInfo() {
  const response = await fetchBundlePayload()
  const purchaseReadyBundle = await completeTemporaryCftpUnlock(response)
  applyBundleInfo(purchaseReadyBundle)

  try {
    const pricingRes = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/pricing-detail`, { suppressErrorToast: true })
    if (pricingRes && pricingRes.pricing_detail_json) {
      pricingDetail.value = pricingRes.pricing_detail_json
    }
  } catch (e) {
    console.error("Failed to load pricing detail", e)
  }

  await refreshQualificationApplications()
  return purchaseReadyBundle
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
  const stages = exemptionStages.value
    .map((stage) => {
      const exemptedUnitIds = (stage.units || [])
        .filter((unit: any) => unit.qualified && unit.unit_id && selectedExemptionUnitIds.value[unit.unit_id])
        .map((unit: any) => unit.unit_id)
      return {
        index: stage.index,
        stage_cc_ulid: stage.stage_id,
        exempted_unit_cc_ulids: exemptedUnitIds,
      }
    })
    .filter((stage) => stage.exempted_unit_cc_ulids.length > 0)

  const pipelineId = bundleData.value?.pipeline_id || bundleData.value?.pipeline_cc_ulid || ""
  return JSON.stringify({
    [pipelineId]: {
      stages
    }
  })
}



function normalizedCredentialApplicationStatus(status: unknown) {
  const enumName = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status as string)
  return String(enumName || status || "").trim().toUpperCase()
}

function isApplicationPendingStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING"
}

function isApplicationApprovedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_APPROVED"
}

function isApplicationRejectedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_REJECTED"
}

function isApplicationResubmitStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_RESUBMIT"
}

function qualificationIdsForUnit(unit: any) {
  return (unit?.exemption_quals || [])
    .map((qual: any) => String(qual?.qual_id || "").trim())
    .filter(Boolean)
}

function qualificationOrderQualIds(primaryQualId: string) {
  const allQualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage: any) => stage.units || [])
      .filter((unit: any) => !unit?.qualified)
      .flatMap((unit: any) => qualificationIdsForUnit(unit)),
  ))
  return [
    primaryQualId,
    ...allQualIds.filter((qualId) => qualId !== primaryQualId),
  ].filter(Boolean)
}

function qualificationApplicationForUnit(unit: any) {
  const applications = qualificationIdsForUnit(unit)
    .map((qualId: string) => qualificationApplications.value[qualId])
    .filter(Boolean)
  return applications.find((application: any) => isApplicationPendingStatus(application?.status))
    || applications.find((application: any) => isApplicationResubmitStatus(application?.status))
    || applications.find((application: any) => isApplicationRejectedStatus(application?.status))
    || applications.find((application: any) => isApplicationApprovedStatus(application?.status))
    || applications[0]
    || null
}

async function latestCredentialApplication(qualId: string) {
  const response = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`, {
    suppressErrorToast: true,
  })
  return (response?.applications || [])[0] || null
}

async function hasQualificationUploadPermission(qualId: string) {
  const response = await apiClient(`/api/credentials/upload-permission?cred_def_ulid=${encodeURIComponent(qualId)}`, {
    suppressErrorToast: true,
  })
  return response?.granted === true
}

async function refreshQualificationApplications() {
  const qualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage: any) => stage.units || [])
      .flatMap((unit: any) => qualificationIdsForUnit(unit)),
  ))
  const next: Record<string, any> = {}
  await Promise.all(qualIds.map(async (qualId) => {
    try {
      const application = await latestCredentialApplication(qualId)
      if (application) next[qualId] = application
    } catch (error) {
      console.warn(`Failed to load credential application ${qualId}`, error)
    }
  }))
  qualificationApplications.value = next
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

    if (hadPendingApplications && !hasPendingQualificationApplications()) {
      const response = await fetchBundlePayload()
      const purchaseReadyBundle = await completeTemporaryCftpUnlock(response)
      applyBundleInfo(purchaseReadyBundle)
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
  const qualId = qualificationIdsForUnit(unit)[0] || ""
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

async function openQualificationEditor(unit: any, qualId = qualificationIdsForUnit(unit)[0] || "") {
  if (!unit?.unit_id || !qualId) return
  await loadQualificationDefinition(qualId)
  expandedQualificationUnitIds.value = {
    ...expandedQualificationUnitIds.value,
    [unit.unit_id]: true,
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
  const qualId = qualificationIdsForUnit(unit)[0] || ""
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
  const qualId = qualificationIdsForUnit(unit)[0] || ""
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
  const qualId = String(route.query.qual_ulids || "").split(",")[0]?.trim() || ""
  const unitId = String(route.query.qualification_unit_id || "").trim()
  const unit = exemptionUnitById(unitId) || exemptionUnitByQualId(qualId)
  if (unit && qualId) {
    currentStep.value = 1
    try {
      await openQualificationEditor(unit, qualId)
    } catch (error) {
      console.error(error)
      toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    }
  }
  const nextQuery = { ...route.query }
  delete nextQuery.payment_status
  delete nextQuery.payment_action
  delete nextQuery.order_id
  delete nextQuery.qual_ulids
  delete nextQuery.qualification_unit_id
  await router.replace({ path: route.path, query: nextQuery })
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

async function startQualificationApplication(unit: any) {
  const qualId = qualificationIdsForUnit(unit)[0] || ""
  const orderQualIds = qualificationOrderQualIds(qualId)
  if (!unit?.unit_id || !qualId || orderQualIds.length === 0 || !pipelineId.value || !bundleId) {
    toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    return
  }

  credentialApplicationLoadingUnitId.value = unit.unit_id
  try {
    const existingApplication = qualificationApplications.value[qualId] || await latestCredentialApplication(qualId)
    if (existingApplication) {
      qualificationApplications.value = {
        ...qualificationApplications.value,
        [qualId]: existingApplication,
      }
      if (isApplicationPendingStatus(existingApplication.status)) {
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      if (isApplicationApprovedStatus(existingApplication.status)) {
        toast.success(t.value.checkoutWizard.qualificationAlreadyApproved)
        await loadPurchaseReadyBundleInfo()
        return
      }
      if (isApplicationResubmitStatus(existingApplication.status)) {
        await openQualificationEditor(unit, qualId)
        return
      }
    }

    if (await hasQualificationUploadPermission(qualId)) {
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      await openQualificationEditor(unit, qualId)
      return
    }

    let order
    try {
      order = await apiClient("/api/credentials/application-orders", {
        method: "POST",
        suppressErrorToast: true,
        body: JSON.stringify({
          pipeline_cc_ulid: pipelineId.value,
          bundle_ulid: bundleId,
          qual_ulids: orderQualIds,
        }),
      })
    } catch (error) {
      const message = error instanceof ApiClientError
        ? error.rawMessage || error.errorCode || ""
        : error instanceof Error ? error.message : ""
      if (message.includes("in-progress credential application") || message.includes("进行中") || message.includes("请先处理")) {
        await refreshQualificationApplications()
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      throw error
    }

    const orderId = String(order?.application_order_ulid || "").trim()
    const orderStatus = String(order?.order_status || "")
    if (isUploadReadyStatus(orderStatus)) {
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      await openQualificationEditor(unit, qualId)
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
      activeCredentialQualIds.value = orderQualIds
      activeCredentialUnitId.value = unit.unit_id
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
    credentialApplicationLoadingUnitId.value = ""
  }
}

async function onExemptionToggle(unit: any, event: Event) {
  const input = event.target as HTMLInputElement | null
  if (!unit?.unit_id) return
  if (!unit.qualified) {
    if (input?.checked) await startQualificationApplication(unit)
    else closeQualificationEditor(unit.unit_id)
    return
  }
  selectedExemptionUnitIds.value = {
    ...selectedExemptionUnitIds.value,
    [unit.unit_id]: Boolean(input?.checked),
  }
}

async function nextFromStep1() {
  currentStep.value = 2
}

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

type ExemptionCredentialState = "active" | "pending" | "resubmit" | "rejected" | "expired" | "revoked" | "missing" | "unavailable"

function exemptionCredentialState(unit: any): ExemptionCredentialState {
  const qualifications = unit?.exemption_quals || []
  if (unit?.qualified || qualifications.some((qual: any) =>
    qual?.eligible || String(qual?.credential_status || "").toUpperCase() === "CREDENTIAL_STATUS_ACTIVE"
  )) {
    return "active"
  }

  const application = qualificationApplicationForUnit(unit)
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

function qualificationActionLabel(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "pending":
      return t.value.checkoutWizard.statusPending
    case "resubmit":
      return t.value.checkoutWizard.resubmitQualification
    default:
      return t.value.checkoutWizard.applyQualification
  }
}

function qualificationStatusHint(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "active":
      return ""
    case "pending":
      return t.value.checkoutWizard.qualificationPendingHint
    case "resubmit":
      return t.value.checkoutWizard.qualificationResubmitHint
    default:
      return t.value.checkoutWizard.qualificationSubmitHint
  }
}

function qualificationStatusHintClass(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "pending":
      return "border-blue-200 bg-blue-50 text-blue-800"
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

async function createUnlockOrder() {
  if (!pipelineId.value) {
    throw new Error(t.value.checkoutWizard.missingPipeline)
  }

  const hadExemptionOptions = exemptionStages.value.length > 0
  const response = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/unlock`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      pipeline_cc_ulid: pipelineId.value,
    }),
  })
  const orderId = String(response?.pipeline_unlock_order_ulid || response?.order_id || "").trim()
  const orderStatus = response?.order_status || response?.status

  if (isFailedStatus(orderStatus)) {
    throw new Error(response?.message || t.value.checkoutWizard.orderCreationFailed)
  }

  if (isCompletedStatus(orderStatus)) {
    toast.success(t.value.checkoutWizard.unlockCompleted)
    const refreshedBundle = await loadBundleInfo()
    if (!getEligibility(refreshedBundle)?.can_purchase) {
      return
    }

    if (!hadExemptionOptions && exemptionStages.value.length > 0) {
      currentStep.value = 1
      return
    }

    await createPurchaseOrder()
    return
  }

  if (!orderId) {
    throw new Error(t.value.checkoutWizard.orderCreationFailed)
  }

  activeOrderAction.value = "unlock"
  activeOrderId.value = orderId
  currentStep.value = 4
}

async function confirmAndPay() {
  loading.value = true
  try {
    const latestBundle = await loadPurchaseReadyBundleInfo()
    const eligibility = getEligibility(latestBundle)

    if (eligibility?.can_unlock) {
      await createUnlockOrder()
      return
    }
    if (eligibility?.can_purchase) {
      await createPurchaseOrder()
      return
    }
    if (latestBundle?.purchase_state?.active_order) {
      toast.info(t.value.checkoutWizard.continueExistingOrder)
      await router.push("/orders")
      return
    }

    throw new Error(t.value.checkoutWizard.purchaseUnavailable)
  } catch (err) {
    console.error(err)
    toast.error(err instanceof Error && err.message
      ? err.message
      : t.value.checkoutWizard.orderCreationFailed)
  } finally {
    loading.value = false
  }
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
            <div class="checkout-progress-step" :class="{ active: currentStep === 1 }" :aria-current="currentStep === 1 ? 'step' : undefined">
              <span class="checkout-progress-node">1</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step1.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 2 }" :aria-current="currentStep === 2 ? 'step' : undefined">
              <span class="checkout-progress-node">2</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step2.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 3 }" :aria-current="currentStep === 3 ? 'step' : undefined">
              <span class="checkout-progress-node">3</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step3.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 4 }" :aria-current="currentStep === 4 ? 'step' : undefined">
              <span class="checkout-progress-node">4</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step4.replace(/^\d+\s*/, "") }}</span>
            </div>
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
                          v-if="exemptionCredentialState(unit) === 'pending'"
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
                    
                    <div class="checkout-unit-footer mt-auto pt-4 flex items-center justify-between border-t border-slate-100">
                      <label class="checkout-unit-option cursor-pointer">
                        <div class="relative flex items-center justify-center">
                          <input
                            data-testid="checkout-exemption-toggle"
                            :data-unit-id="unit.unit_id"
                            type="checkbox"
                            class="peer sr-only"
                            :checked="unit.qualified ? Boolean(selectedExemptionUnitIds[unit.unit_id]) : isQualificationEditorExpanded(unit.unit_id)"
                            :disabled="credentialApplicationLoadingUnitId === unit.unit_id || (!unit.qualified && exemptionCredentialState(unit) === 'pending')"
                            @change="onExemptionToggle(unit, $event)"
                          />
                          <div class="checkout-unit-checkbox h-6 w-6 rounded-md border-2 border-slate-300 bg-white transition-all peer-checked:border-emerald-500 peer-checked:bg-emerald-500"></div>
                          <Loader2 v-if="credentialApplicationLoadingUnitId === unit.unit_id" class="pointer-events-none absolute h-4 w-4 animate-spin text-blue-600" />
                          <Check v-else class="pointer-events-none absolute h-4 w-4 text-white opacity-0 transition-opacity peer-checked:opacity-100" />
                        </div>
                        <span class="checkout-unit-action font-medium text-slate-700">
                          {{ unit.qualified ? t.checkoutWizard.applyForExemption : qualificationActionLabel(unit) }}
                        </span>
                        <span
                          v-if="selectedExemptionUnitIds[unit.unit_id]"
                          class="checkout-unit-selected-price"
                        >
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.exemptionAmount ?? 0, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </span>
                        <strong
                          v-else-if="unitPriceDisplay[unit.unit_id]?.accessAmount !== undefined"
                          class="checkout-unit-default-price"
                        >
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.accessAmount, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </strong>
                      </label>
                    </div>

                    <div
                      v-if="isQualificationEditorExpanded(unit.unit_id) && !unit.qualified"
                      class="mt-5 border-t border-blue-100 pt-5"
                    >
                      <div class="rounded-2xl border border-blue-100 bg-blue-50/70 p-4 sm:p-5">
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

                        <div
                          v-if="Array.isArray(qualificationDefinitionForUnit(unit)?.file_constraints)"
                          class="mt-5 grid gap-4 sm:grid-cols-2"
                        >
                          <div
                            v-for="constraint in qualificationDefinitionForUnit(unit)?.file_constraints || []"
                            :key="constraint.name"
                            class="rounded-xl border border-white bg-white p-4 shadow-sm"
                          >
                            <div class="flex items-center gap-1 text-sm font-semibold text-slate-800">
                              <span v-if="constraint.is_required" class="text-rose-500">*</span>
                              <span>{{ constraint.name }}</span>
                            </div>
                            <p class="mt-1 text-xs text-slate-500">{{ qualificationFormatHint(constraint) }}</p>
                            <div class="mt-3 flex flex-wrap items-center gap-3">
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

                        <div class="mt-5 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
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
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="isExemptionSelected" class="checkout-declaration mt-8 rounded-xl border border-blue-200 bg-blue-50/50 p-5 transition-all">
                <label class="flex cursor-pointer items-start gap-3">
                  <div class="relative mt-0.5 flex shrink-0 items-center justify-center">
                    <input
                      v-model="exemptionDeclarationChecked"
                      type="checkbox"
                      class="peer sr-only"
                    />
                    <div class="h-5 w-5 rounded border border-slate-300 bg-white transition-all peer-checked:border-emerald-500 peer-checked:bg-emerald-500"></div>
                    <Check class="pointer-events-none absolute h-3.5 w-3.5 text-white opacity-0 transition-opacity peer-checked:opacity-100" />
                  </div>
                  <span class="text-sm font-medium leading-relaxed text-slate-700">
                    {{ t.checkoutWizard.declarationText }}
                  </span>
                </label>
              </div>

              <div v-if="bundleData" class="checkout-step-actions mt-6 flex items-center justify-end">
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
                    <option v-for="prefix in orgPhonePrefixes" :key="prefix.code" :value="prefix.code">{{ prefix.dialCode }}</option>
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
            :disabled="hasExpandedQualificationEditors || Boolean(qualificationSubmittingUnitId) || (isExemptionSelected && !exemptionDeclarationChecked)"
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
          <button form="checkout-registration-form" data-testid="checkout-next" type="submit" class="checkout-form-next-button btn rounded-full px-8 text-white" :disabled="loading">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.examSignup.submitting }}</template>
            <template v-else>{{ t.checkoutWizard.next }} <Send class="ml-2 h-4 w-4" /></template>
          </button>
        </div>

        <div v-else-if="currentStep === 3" class="checkout-step-footer checkout-review-actions flex items-center">
          <button type="button" class="checkout-back-button btn btn-outline" @click="currentStep = 2" :disabled="loading">
            <ArrowLeft class="h-4 w-4" />
            {{ t.checkoutWizard.back }}
          </button>
          <button data-testid="checkout-confirm-pay" class="checkout-form-next-button btn btn-vivid" :disabled="loading" @click="confirmAndPay">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.checkoutWizard.processing }}</template>
            <template v-else>{{ t.checkoutWizard.confirmAndPay }} <ArrowRight class="h-4 w-4" /></template>
          </button>
        </div>
        </template>
      </main>
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
  color: #52617a;
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

.checkout-total:empty {
  display: none;
}

.checkout-total {
  color: #002a66;
  font-size: 18px;
  line-height: 1.4;
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
    justify-content: flex-start;
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
