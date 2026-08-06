<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, ClipboardList, Loader2, Send } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import LocalizedDatePicker from "@/components/LocalizedDatePicker.vue"
import { apiClient } from "@/lib/apiClient"
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

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const { currentUser, fetchUser } = useUser()
const unitId = String(route.query.unitId || "")
const pipelineId = String(route.query.pipelineId || "")
const courseId = String(route.query.courseId || "")
const loading = ref(false)
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
})
const returnTo = computed(() => {
  const value = route.query.returnTo
  return Array.isArray(value) ? String(value[0] || "") : String(value || "")
})
const shouldReturnToExams = computed(() => returnTo.value === "/exams")
const backLink = computed(() => {
  if (shouldReturnToExams.value) return "/exams"
  if (pipelineId && courseId) return `/certifications/${encodeURIComponent(pipelineId)}/learn/${encodeURIComponent(courseId)}`
  if (pipelineId) return `/certifications/${encodeURIComponent(pipelineId)}`
  return "/exams"
})
const backLabel = computed(() => {
  if (shouldReturnToExams.value || (!pipelineId && !courseId)) return t.value.examsPage.backToExams
  if (pipelineId && courseId) return t.value.examSignup.backToLearning
  return t.value.examSignup.backToCertification
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

async function handleSubmit() {
  if (!unitId) {
    toast.error(t.value.common.error)
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
    if (!String(formData[key]).trim()) {
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
    await apiClient(`/api/exams/units/${encodeURIComponent(unitId)}/signup`, { method: "POST", body: JSON.stringify(formData) })
    toast.success(t.value.examSignup.success, {
      description: t.value.examSignup.successDescription,
      duration: 6000,
    })
    router.push("/exams")
  } catch (err) {
    // apiClient already displays the localized API error; keep the form open.
    console.error("Exam signup failed", err)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <ClipboardList class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.examSignup.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <RouterLink :to="backLink" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" /> {{ backLabel }}
        </RouterLink>
        <div class="mb-8 max-w-2xl">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.examSignup.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.examSignup.subtitle }}</p>
        </div>
        <div class="max-w-2xl rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <form data-testid="exam-signup-form" class="space-y-6" novalidate @submit.prevent="handleSubmit">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formFirstName }}</span><input v-model="formData.first_name" data-testid="exam-signup-first-name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formLastName }}</span><input v-model="formData.last_name" data-testid="exam-signup-last-name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formMiddleName }}</span><input v-model="formData.middle_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formEmail }}</span><input v-model="formData.email" data-testid="exam-signup-email" class="input" type="email" :maxlength="PROFILE_TEXT_LIMITS.short" required /></label>
          <label class="space-y-2">
            <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formGender }}</span>
            <select v-model="formData.gender" data-testid="exam-signup-gender" class="input cursor-pointer" required>
              <option value="" disabled>{{ t.examSignup.formGender }}</option>
              <option v-for="option in genderOptions" :key="option" :value="option">{{ t.common.genderOptions[option] }}</option>
            </select>
          </label>
        </div>
        <label class="block space-y-2">
          <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formBirthdate }}</span>
          <LocalizedDatePicker
            v-model="formData.birthdate"
            data-testid="exam-signup-birthdate"
            :language="lang"
            :placeholder="lang === 'zh' ? '日/月/年' : 'DD/MM/YYYY'"
            :aria-label="t.examSignup.formBirthdate"
          />
        </label>
        <div class="grid gap-4" :class="locationGridClass">
          <label class="space-y-2">
            <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formCountry }}</span>
            <select v-model="selectedCountryCode" data-testid="exam-signup-country" class="input cursor-pointer" required @change="handleCountryChange">
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
        <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formAddress }}</span><input v-model="formData.address" data-testid="exam-signup-address" class="input" :maxlength="PROFILE_TEXT_LIMITS.address" required /></label>
        <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formPostalCode }}</span><input v-model="formData.postal_code" data-testid="exam-signup-postal-code" class="input" :maxlength="PROFILE_TEXT_LIMITS.postalCode" pattern="[A-Za-z0-9][A-Za-z0-9 -]*[A-Za-z0-9]" required @blur="formData.postal_code = normalizePostalCode(formData.postal_code)" /></label>
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
        <div class="flex justify-end pt-2">
          <button type="submit" data-testid="exam-signup-submit" class="btn btn-primary w-full sm:w-auto" :disabled="loading">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.examSignup.submitting }}</template>
            <template v-else><Send class="mr-2 h-4 w-4" /> {{ t.examSignup.submit }}</template>
          </button>
        </div>
      </form>
    </div>
      </main>
    </div>
  </AppShell>
</template>
