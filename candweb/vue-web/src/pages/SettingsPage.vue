<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { Loader2, Settings } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import LocalizedDatePicker from "@/components/LocalizedDatePicker.vue"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession } from "@/lib/authStorage"
import { getMessage } from "@/lib/messages"
import { useTranslation } from "@/lib/language"
import {
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
  resolvePhoneCountryCode,
} from "@/lib/locationOptions"
import { GENDER_OPTIONS, PROFILE_TEXT_LIMITS, isValidInternationalPhone, isValidPostalCode, normalizeGender, normalizeInternationalPhone, normalizePostalCode, trimToMax } from "@/lib/profileFormValidation"
import { useUser } from "@/lib/user"

const route = useRoute()
const router = useRouter()
const { t, lang, changeLanguage } = useTranslation()
const { fetchUser } = useUser()
const activeTab = ref(String(route.query.tab || "profile"))
const profile = reactive({
  email: "",
  avatar: "",
  firstName: "",
  lastName: "",
  homePhone: "",
  phoneCountryCode: "",
  phone: "",
  gender: "",
  birthday: "",
  country: "",
  province: "",
  city: "",
  address: "",
  postalCode: "",
  affiliation: "",
  title: "",
  bio: "",
  education: "",
  homepage: "",
  linkedin: "",
  twitter: "",
})
const password = reactive({ oldPassword: "", newPassword: "", confirmPassword: "" })
const emailChange = reactive({ newEmail: "", oldCode: "", newCode: "", oldCodeSent: false, newCodeSent: false })
const isEmailCodeLoading = ref(false)
const isNewEmailCodeLoading = ref(false)
const isEmailConfirmLoading = ref(false)
const isProfileLoading = ref(false)
const isPasswordLoading = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarFile = ref<File | null>(null)
const avatarCanvas = ref<HTMLCanvasElement | null>(null)
const avatarImage = ref<HTMLImageElement | null>(null)
const avatarPreviewURL = ref("")
const avatarZoom = ref(1)
const avatarX = ref(50)
const avatarY = ref(50)
const isAvatarUploading = ref(false)

function renderAvatarCrop() {
  const image = avatarImage.value
  const canvas = avatarCanvas.value
  if (!image || !canvas) return
  const ctx = canvas.getContext("2d")
  if (!ctx) return
  const side = Math.min(image.naturalWidth, image.naturalHeight) / avatarZoom.value
  const left = (image.naturalWidth - side) * avatarX.value / 100
  const top = (image.naturalHeight - side) * avatarY.value / 100
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  ctx.drawImage(image, left, top, side, side, 0, 0, canvas.width, canvas.height)
}

function resetAvatarSelection() {
  if (avatarPreviewURL.value) URL.revokeObjectURL(avatarPreviewURL.value)
  avatarPreviewURL.value = ""
  avatarFile.value = null
  avatarImage.value = null
  if (avatarInput.value) avatarInput.value.value = ""
}

function selectAvatar(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (!["image/jpeg", "image/png"].includes(file.type) || file.size > 2 * 1024 * 1024) {
    toast.error(t.value.settings.avatarInvalid)
    resetAvatarSelection()
    return
  }
  resetAvatarSelection()
  avatarFile.value = file
  avatarPreviewURL.value = URL.createObjectURL(file)
  avatarZoom.value = 1
  avatarX.value = 50
  avatarY.value = 50
  const image = new Image()
  const previewURL = avatarPreviewURL.value
  image.onload = () => {
    if (avatarPreviewURL.value !== previewURL) return
    avatarImage.value = image
    renderAvatarCrop()
  }
  image.onerror = () => {
    if (avatarPreviewURL.value !== previewURL) return
    toast.error(t.value.settings.avatarInvalid)
    resetAvatarSelection()
  }
  image.src = previewURL
}

async function uploadAvatar() {
  if (!avatarFile.value || isAvatarUploading.value) return
  isAvatarUploading.value = true
  try {
    const canvas = avatarCanvas.value
    if (!canvas || !avatarImage.value) return
    const cropped = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/png"))
    if (!cropped) throw new Error("failed to crop avatar")
    const body = new FormData()
    body.append("avatar", cropped, "avatar.png")
    const result = await apiClient("/api/user/avatar", { method: "POST", body })
    profile.avatar = String(result.avatar || "")
    await fetchUser(true)
    resetAvatarSelection()
    toast.success(t.value.settings.avatarUpdated)
  } catch {
    // apiClient displays the error.
  } finally {
    isAvatarUploading.value = false
  }
}

function updatePreferredLanguage(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (value === "zh" || value === "en") changeLanguage(value)
}

onBeforeUnmount(resetAvatarSelection)

const selectedCountryCode = ref("")
const selectedProvinceCode = ref("")
const organizationPhoneCountryCodes = ref<unknown>([])
const phonePrefixLocale = computed(() => lang.value === "zh" ? "zh-CN" : "en")
const orgPhonePrefixes = computed(() => getOrganizationPhonePrefixes(
  organizationPhoneCountryCodes.value,
  phonePrefixLocale.value,
))
const countryOptions = ref<CountryOption[]>([])
const provinceOptions = ref<any[]>([])
const cityOptions = ref<any[]>([])
const showProvinceField = computed(() => !selectedCountryCode.value || countryUsesProvinceField(selectedCountryCode.value))
const showCityField = computed(() => !selectedCountryCode.value || countryUsesCityField(selectedCountryCode.value))
const genderOptions = GENDER_OPTIONS

function normalizeDate(value: unknown) {
  return typeof value === "string" ? value.split("T")[0] : ""
}

function normalizeAddress(value: unknown, fallback: unknown) {
  if (typeof value === "string") return value
  if (Array.isArray(fallback)) return fallback.join(", ")
  if (typeof fallback === "string") return fallback
  return ""
}

function localizedProvinceName(province: any) {
  return lang.value === "zh" && selectedCountryCode.value === "CN" ? CN_STATE_LABELS[province.isoCode] || province.name : province.name
}

function localizedCityName(city: any) {
  if (typeof city?.localizedName === "string") return city.localizedName
  return city.name
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
  const cityText = normalizeLocationText(profile.city)
  if (!cityText) return
  const exists = cityOptions.value.some((city) =>
    [city.name, localizedCityName(city)].some((value) => normalizeLocationText(value) === cityText),
  )
  if (!exists) {
    cityOptions.value = [{ name: profile.city, localizedName: profile.city }, ...cityOptions.value]
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

function syncLocationSelectionFromProfile() {
  const allCountries = getCachedCountries()
  if (allCountries.length === 0) return
  const countryText = normalizeLocationText(profile.country)
  const zhRegionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" })
  const matchedCountry = allCountries.find((country) =>
    [country.name, country.isoCode, country.phonecode].some((value) => normalizeLocationText(value) === countryText) ||
    normalizeLocationText(zhRegionNames.of(country.isoCode)) === countryText,
  )
  const normalizedCountry = normalizeAddressCountryCode(matchedCountry?.isoCode || "")
  selectedCountryCode.value = normalizedCountry.countryCode
  if (normalizedCountry.provinceCode) {
    profile.country = allCountries.find((country) => country.isoCode === "CN")?.name || "China"
  }
  refreshProvinceOptions()
  if (selectedCountryCode.value && !countryUsesProvinceField(selectedCountryCode.value)) {
    profile.province = ""
  }

  const provinceText = normalizeLocationText(profile.province)
  const matchedProvince = selectedCountryCode.value
    ? provinceOptions.value.find((state) =>
        state.isoCode === normalizedCountry.provinceCode ||
        provinceMatchValues(state).some((value) => normalizeProvinceText(value) === normalizeProvinceText(provinceText)),
      )
    : undefined
  selectedProvinceCode.value = matchedProvince?.isoCode || ""
  if (normalizedCountry.provinceCode && matchedProvince) {
    profile.province = localizedProvinceName(matchedProvince)
  }
  if (selectedCountryCode.value && !countryUsesCityField(selectedCountryCode.value)) {
    profile.city = ""
  }
  refreshCityOptions()
  ensureCurrentCityOption()
}

function handleCountryChange() {
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  profile.country = country?.name || ""
  profile.province = ""
  profile.city = ""
  selectedProvinceCode.value = ""
  refreshProvinceOptions()
  refreshCityOptions()
}

function handleProvinceChange() {
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  profile.province = province ? localizedProvinceName(province) : ""
  profile.city = ""
  refreshCityOptions()
}

function sanitizeProfileForm() {
  profile.firstName = trimToMax(profile.firstName, PROFILE_TEXT_LIMITS.name)
  profile.lastName = trimToMax(profile.lastName, PROFILE_TEXT_LIMITS.name)
  profile.gender = normalizeGender(profile.gender)
  profile.country = trimToMax(profile.country, PROFILE_TEXT_LIMITS.short)
  profile.province = trimToMax(profile.province, PROFILE_TEXT_LIMITS.short)
  profile.city = trimToMax(profile.city, PROFILE_TEXT_LIMITS.short)
  profile.address = trimToMax(profile.address, PROFILE_TEXT_LIMITS.address)
  profile.postalCode = normalizePostalCode(profile.postalCode)
  profile.affiliation = trimToMax(profile.affiliation, PROFILE_TEXT_LIMITS.short)
  profile.title = trimToMax(profile.title, PROFILE_TEXT_LIMITS.short)
  profile.education = trimToMax(profile.education, PROFILE_TEXT_LIMITS.short)
  profile.bio = trimToMax(profile.bio, PROFILE_TEXT_LIMITS.bio)
  profile.homePhone = normalizeInternationalPhone(profile.homePhone)
  profile.phone = normalizeInternationalPhone(profile.phone)
  profile.homepage = profile.homepage.trim()
  profile.linkedin = profile.linkedin.trim()
  profile.twitter = profile.twitter.trim()
}

function validProfileLink(value: string, domains: string[] = []) {
  if (!value) return true
  if (value.length > 100) return false
  try {
    const parsed = new URL(value)
    if (!["http:", "https:"].includes(parsed.protocol) || !parsed.hostname || parsed.username || parsed.password) return false
    return domains.length === 0 || domains.some((domain) => parsed.hostname === domain || parsed.hostname.endsWith(`.${domain}`))
  } catch {
    return false
  }
}

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = tab === "account" ? "account" : "profile"
  },
  { immediate: true },
)

function setActiveTab(tab: "profile" | "account") {
  activeTab.value = tab
  void router.replace({ query: { ...route.query, tab } })
}

onMounted(async () => {
  const locationReady = loadLocationData()
    .then(() => {
      refreshCountryOptions()
    })
    .catch((err: unknown) => {
      console.error("Failed to load location data", err)
      toast.error(t.value.common.locationDataLoadFailed, { id: "location-data-load-failed" })
    })

  try {
    const payload = await apiClient("/api/user/me")
    if (payload) {
      profile.email = payload.email || ""
      profile.avatar = payload.avatar || ""
      profile.firstName = payload.first_name || ""
      profile.lastName = payload.last_name || ""
      profile.homePhone = payload.home_phone || payload.phone || ""
      profile.phoneCountryCode = payload.phone_country_code || ""
      profile.phone = payload.phone || ""
      profile.gender = normalizeGender(payload.gender)
      profile.birthday = normalizeDate(payload.birthday)
      profile.country = payload.country || payload.region || ""
      profile.province = payload.province || ""
      profile.city = payload.city || payload.location || ""
      profile.address = normalizeAddress(payload.address_text, payload.address)
      profile.postalCode = payload.postal_code || ""
      profile.affiliation = payload.affiliation || ""
      profile.title = payload.title || ""
      profile.bio = payload.bio || ""
      profile.education = payload.education || ""
      profile.homepage = payload.homepage || ""
      profile.linkedin = payload.linkedin || ""
      profile.twitter = payload.twitter || ""
      await locationReady
      syncLocationSelectionFromProfile()
      
      const configRes = await apiClient("/api/public/config/organization")
      if (configRes && configRes.country_codes) {
        organizationPhoneCountryCodes.value = configRes.country_codes
        profile.phoneCountryCode = resolvePhoneCountryCode(
          profile.phoneCountryCode,
          orgPhonePrefixes.value,
          selectedCountryCode.value,
        )
      }
    }
  } catch {
    // apiClient handles toast.
    await locationReady
  }
})

watch(lang, () => {
  refreshCountryOptions()
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  if (country) profile.country = country.name
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  if (province) profile.province = localizedProvinceName(province)
  refreshCityOptions()
  ensureCurrentCityOption()
})

async function handleUpdateProfile() {
  sanitizeProfileForm()
  Object.assign(profile, normalizeLocationForSubmission(
    selectedCountryCode.value,
    profile.country,
    profile.province,
    profile.city,
  ))

  const requiredFields = [
    [profile.firstName, t.value.settings.firstName],
    [profile.lastName, t.value.settings.lastName],
    [profile.gender, t.value.settings.gender],
    [profile.birthday, t.value.settings.birthday],
    [profile.country, t.value.settings.country],
    ...(showProvinceField.value ? [[profile.province, t.value.settings.province] as const] : []),
    ...(showCityField.value ? [[profile.city, t.value.settings.city] as const] : []),
    [profile.address, t.value.settings.address],
    [profile.postalCode, t.value.settings.postalCode],
  ] as const
  for (const [value, label] of requiredFields) {
    if (!String(value).trim()) {
      toast.error(t.value.settings.validationRequired.replace("{{field}}", label))
      return
    }
  }
  if (!isValidInternationalPhone(profile.phone)) {
    toast.error(t.value.settings.validationInvalidPhone.replace("{{field}}", t.value.settings.workPhone))
    return
  }
  if (!isValidPostalCode(profile.postalCode, true)) {
    toast.error(t.value.settings.validationInvalidPostalCode)
    return
  }
  for (const link of [
    { label: t.value.settings.website, value: profile.homepage, domains: [] },
    { label: t.value.settings.linkedin, value: profile.linkedin, domains: ["linkedin.com"] },
    { label: t.value.settings.xProfile, value: profile.twitter, domains: ["x.com", "twitter.com"] },
  ]) {
    if (!validProfileLink(link.value, link.domains)) {
      toast.error(t.value.settings.validationInvalidLink.replace("{{field}}", link.label))
      return
    }
  }
  isProfileLoading.value = true
  try {
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify({
        first_name: profile.firstName,
        last_name: profile.lastName,
        home_phone: profile.homePhone,
        phone_country_code: profile.phoneCountryCode,
        phone: profile.phone,
        gender: profile.gender,
        birthday: profile.birthday,
        country: profile.country,
        province: profile.province,
        city: profile.city,
        address: profile.address,
        postal_code: profile.postalCode,
        affiliation: profile.affiliation,
        title: profile.title,
        bio: profile.bio,
        education: profile.education,
        homepage: profile.homepage,
        linkedin: profile.linkedin,
        twitter: profile.twitter,
      }),
    })
    await fetchUser(true)
    toast.success(getMessage("PROFILE_UPDATE_SUCCESS", lang.value))
  } catch (e) {
    // Error is handled by apiClient toast
  } finally {
    isProfileLoading.value = false
  }
}

async function handleUpdatePassword() {
  const requiredFields = [
    [password.oldPassword, t.value.settings.currentPassword],
    [password.newPassword, t.value.settings.newPassword],
    [password.confirmPassword, t.value.settings.confirmNewPassword],
  ] as const
  for (const [value, label] of requiredFields) {
    if (!value) {
      toast.error(t.value.settings.validationRequired.replace("{{field}}", label))
      return
    }
  }
  if (password.newPassword !== password.confirmPassword) {
    toast.error(getMessage("PASSWORD_MISMATCH", lang.value))
    return
  }
  isPasswordLoading.value = true
  try {
    await apiClient("/api/user/password", {
      method: "PUT",
      body: JSON.stringify({ old_password: password.oldPassword, new_password: password.newPassword }),
    })
    toast.success(getMessage("PASSWORD_UPDATE_SUCCESS", lang.value))
    clearAuthSession()
    setTimeout(() => { window.location.href = "/login" }, 1500)
  } catch (e) {
    // Error is handled by apiClient toast
  } finally {
    isPasswordLoading.value = false
  }
}

async function sendEmailChangeCode() {
  if (isEmailCodeLoading.value) return
  isEmailCodeLoading.value = true
  try {
    await apiClient("/api/user/email/send-code", { method: "POST" })
    emailChange.oldCodeSent = true
    toast.success(t.value.settings.emailCodeSent)
  } catch {
    // apiClient displays the error.
  } finally {
    isEmailCodeLoading.value = false
  }
}

async function sendNewEmailCode() {
  if (isNewEmailCodeLoading.value) return
  if (!emailChange.newEmail.trim()) {
    toast.error(t.value.settings.emailChangeInvalid)
    return
  }
  isNewEmailCodeLoading.value = true
  try {
    await apiClient("/api/user/email/send-new-code", {
      method: "POST",
      body: JSON.stringify({ new_email: emailChange.newEmail.trim() }),
    })
    emailChange.newCodeSent = true
    toast.success(t.value.settings.newEmailCodeSent)
  } catch {
    // apiClient displays the error.
  } finally {
    isNewEmailCodeLoading.value = false
  }
}

watch(() => emailChange.newEmail, () => {
  emailChange.newCode = ""
  emailChange.newCodeSent = false
})

async function confirmEmailChange() {
  if (isEmailConfirmLoading.value) return
  if (!emailChange.newEmail.trim() || !/^\d{6}$/.test(emailChange.oldCode) || !/^\d{6}$/.test(emailChange.newCode)) {
    toast.error(t.value.settings.emailChangeInvalid)
    return
  }
  isEmailConfirmLoading.value = true
  try {
    await apiClient("/api/user/email/confirm", {
      method: "POST",
      body: JSON.stringify({ new_email: emailChange.newEmail.trim(), old_code: emailChange.oldCode, new_code: emailChange.newCode }),
    })
    clearAuthSession()
    toast.success(t.value.settings.emailChanged)
    await router.replace("/login")
  } catch {
    // apiClient displays the error.
  } finally {
    isEmailConfirmLoading.value = false
  }
}

</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Settings class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.settings.title }}</span>
      </header>

      <main class="settings-main px-5 py-8 md:px-8 lg:px-10">
        <div class="settings-title-row mb-6 flex items-center justify-between space-y-2">
          <h1 class="settings-title text-3xl font-bold tracking-tight">{{ t.settings.title }}</h1>
        </div>
    <div class="settings-stack space-y-4">
      <div class="settings-tabs-shell rounded-[14px] bg-white px-5 pt-4 shadow-[0_10px_24px_rgba(15,74,82,0.04)] md:px-6">
        <div class="settings-tabs flex flex-wrap gap-x-8 gap-y-2 border-b border-border">
          <button
            :class="['settings-tab relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === 'profile' ? 'text-primary' : 'text-foreground hover:text-primary']"
            @click="setActiveTab('profile')"
          >
            {{ t.settings.profileTab }}
            <span v-if="activeTab === 'profile'" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
          <button
            :class="['settings-tab relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === 'account' ? 'text-primary' : 'text-foreground hover:text-primary']"
            @click="setActiveTab('account')"
          >
            {{ t.settings.accountTab }}
            <span v-if="activeTab === 'account'" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
        </div>
      </div>
      <div v-if="activeTab === 'profile'" class="settings-panel rounded-md bg-white text-card-foreground">
        <div class="settings-section-header flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.profileTab }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.profileDesc }}</p>
        </div>
    <div class="settings-section-content p-6 pt-0">
        <div class="mb-6 flex flex-wrap items-center gap-4 border-b border-border pb-6">
          <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary/10 text-2xl font-semibold text-primary">
            <img v-if="profile.avatar" :src="profile.avatar" class="h-full w-full object-cover" :alt="t.settings.avatar" />
            <span v-else>{{ (profile.firstName || profile.lastName || '?').charAt(0).toUpperCase() }}</span>
          </div>
          <div class="flex min-w-0 flex-col gap-2">
            <span class="text-sm font-medium">{{ t.settings.avatar }}</span>
            <input ref="avatarInput" class="sr-only" type="file" accept="image/png,image/jpeg" :aria-label="t.settings.avatar" @change="selectAvatar" />
            <div class="flex flex-wrap gap-2">
              <button type="button" class="btn btn-outline" @click="avatarInput?.click()">{{ t.settings.chooseAvatar }}</button>
              <button v-if="avatarFile" type="button" class="btn btn-primary" :disabled="isAvatarUploading || !avatarImage" @click="uploadAvatar">
                <Loader2 v-if="isAvatarUploading" class="h-4 w-4 animate-spin" />{{ t.settings.saveAvatar }}
              </button>
            </div>
            <span class="text-xs text-muted-foreground">{{ t.settings.avatarHint }}</span>
          </div>
        </div>
        <div v-show="avatarFile" class="mb-6 max-w-sm space-y-3 border-b border-border pb-6">
          <canvas ref="avatarCanvas" width="256" height="256" class="aspect-square w-48 border border-border object-contain" :aria-label="t.settings.avatarCropPreview" />
          <label class="block space-y-1 text-sm"><span>{{ t.settings.avatarZoom }}</span><input v-model.number="avatarZoom" class="w-full accent-primary" type="range" min="1" max="3" step="0.05" @input="renderAvatarCrop" /></label>
          <label class="block space-y-1 text-sm"><span>{{ t.settings.avatarHorizontal }}</span><input v-model.number="avatarX" class="w-full accent-primary" type="range" min="0" max="100" step="1" @input="renderAvatarCrop" /></label>
          <label class="block space-y-1 text-sm"><span>{{ t.settings.avatarVertical }}</span><input v-model.number="avatarY" class="w-full accent-primary" type="range" min="0" max="100" step="1" @input="renderAvatarCrop" /></label>
        </div>
        <form class="settings-form max-w-2xl space-y-4" novalidate @submit.prevent="handleUpdateProfile">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.firstName }}</span><input v-model="profile.firstName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.firstNamePlaceholder" required /></label>
            <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.lastName }}</span><input v-model="profile.lastName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.lastNamePlaceholder" required /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.languagePreference }}</span>
              <select class="input cursor-pointer" :value="lang" @change="updatePreferredLanguage">
                <option value="zh">中文</option><option value="en">English</option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.gender }}</span>
              <select v-model="profile.gender" class="input cursor-pointer" required>
                <option value="">{{ t.settings.genderPlaceholder }}</option>
                <option v-for="option in genderOptions" :key="option" :value="option">{{ t.common.genderOptions[option] }}</option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.birthday }}</span>
              <LocalizedDatePicker
                v-model="profile.birthday"
                :language="lang"
                :placeholder="t.settings.birthdayPlaceholder"
                :aria-label="t.settings.birthday"
              />
            </label>

            <label class="space-y-2">
              <span class="text-sm font-medium">{{ t.settings.workPhone }}</span>
              <div class="flex gap-2">
                <select v-if="orgPhonePrefixes.length > 0" v-model="profile.phoneCountryCode" data-testid="settings-phone-country-code" class="input cursor-pointer w-28 shrink-0">
                  <option v-for="prefix in orgPhonePrefixes" :key="prefix.code" :value="prefix.code">{{ prefix.dialCode }} · {{ prefix.name }}</option>
                </select>
                <input
                  id="settings-work-phone"
                  v-model="profile.phone"
                  class="input flex-1"
                  type="tel"
                  inputmode="tel"
                  autocomplete="tel"
                  maxlength="24"
                  :placeholder="t.settings.workPhonePlaceholder"
                />
              </div>
            </label>
            <label class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.country }}</span>
              <select v-model="selectedCountryCode" class="input cursor-pointer" required @change="handleCountryChange">
                <option value="">{{ t.settings.countryPlaceholder }}</option>
                <option v-for="country in countryOptions" :key="country.code" :value="country.code">{{ country.displayName }}</option>
              </select>
            </label>
            <label v-if="showProvinceField" class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.province }}</span>
              <select v-if="provinceOptions.length > 0" v-model="selectedProvinceCode" class="input cursor-pointer" required @change="handleProvinceChange">
                <option value="">{{ t.settings.provincePlaceholder }}</option>
                <option v-for="province in provinceOptions" :key="province.isoCode" :value="province.isoCode">{{ localizedProvinceName(province) }}</option>
              </select>
              <input v-else v-model="profile.province" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" :placeholder="t.settings.provincePlaceholder" required />
            </label>
            <label v-if="showCityField" class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.city }}</span>
              <select v-if="cityOptions.length > 0" v-model="profile.city" class="input cursor-pointer" required>
                <option value="">{{ t.settings.cityPlaceholder }}</option>
                <option v-for="city in cityOptions" :key="`${city.name}-${city.latitude}-${city.longitude}`" :value="localizedCityName(city)">{{ localizedCityName(city) }}</option>
              </select>
              <input v-else v-model="profile.city" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" :placeholder="t.settings.cityPlaceholder" required />
            </label>
            <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.postalCode }}</span><input v-model="profile.postalCode" class="input" :maxlength="PROFILE_TEXT_LIMITS.postalCode" pattern="[A-Za-z0-9][A-Za-z0-9 -]*[A-Za-z0-9]" :placeholder="t.settings.postalCodePlaceholder" required @blur="profile.postalCode = normalizePostalCode(profile.postalCode)" /></label>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.address }}</span><input v-model="profile.address" class="input" :maxlength="PROFILE_TEXT_LIMITS.address" :placeholder="t.settings.addressPlaceholder" required /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.affiliation }}</span><input v-model="profile.affiliation" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" :placeholder="t.settings.affiliationPlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.jobTitle }}</span><input v-model="profile.title" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" :placeholder="t.settings.jobTitlePlaceholder" /></label>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium">{{ t.settings.education }}</span><input v-model="profile.education" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" :placeholder="t.settings.educationPlaceholder" /></label>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium">{{ t.settings.bio }}</span><textarea v-model="profile.bio" class="textarea" :maxlength="PROFILE_TEXT_LIMITS.bio" :placeholder="t.settings.bioPlaceholder" rows="3" /></label>
            <div class="md:col-span-2 border-t border-border pt-5">
              <h3 class="text-base font-semibold">{{ t.settings.personalLinks }}</h3>
            </div>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium">{{ t.settings.website }}</span><input v-model="profile.homepage" class="input" type="url" maxlength="100" placeholder="https://example.com" autocomplete="url" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.linkedin }}</span><input v-model="profile.linkedin" class="input" type="url" maxlength="100" placeholder="https://www.linkedin.com/in/" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.xProfile }}</span><input v-model="profile.twitter" class="input" type="url" maxlength="100" placeholder="https://x.com/" /></label>
          </div>
          <button class="btn btn-primary" :disabled="isProfileLoading"><Loader2 v-if="isProfileLoading" class="h-4 w-4 animate-spin" /> {{ t.common.save }}</button>
        </form>
        </div>
      </div>
      <div v-if="activeTab === 'account'" class="settings-panel rounded-md bg-white text-card-foreground">
        <div class="settings-section-header flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.email }}</h2>
          <p class="text-sm text-muted-foreground">{{ profile.email || '-' }}</p>
        </div>
        <div class="settings-section-header flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none">{{ t.settings.changeEmail }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.changeEmailDesc }}</p>
        </div>
        <div class="settings-section-content p-6 pt-0">
          <form class="settings-form max-w-xl space-y-4" @submit.prevent="confirmEmailChange">
            <label class="block space-y-2"><span class="text-sm font-medium">{{ t.settings.newEmail }}</span><input v-model="emailChange.newEmail" class="input" type="email" autocomplete="email" required /></label>
            <div class="flex flex-wrap items-end gap-3">
              <label class="min-w-[180px] flex-1 space-y-2"><span class="text-sm font-medium">{{ t.settings.oldEmailCode }}</span><input v-model="emailChange.oldCode" class="input" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="one-time-code" required /></label>
              <button type="button" class="btn btn-outline" :disabled="isEmailCodeLoading" @click="sendEmailChangeCode"><Loader2 v-if="isEmailCodeLoading" class="mr-2 h-4 w-4 animate-spin" />{{ t.settings.sendOldEmailCode }}</button>
            </div>
            <div class="flex flex-wrap items-end gap-3">
              <label class="min-w-[180px] flex-1 space-y-2"><span class="text-sm font-medium">{{ t.settings.newEmailCode }}</span><input v-model="emailChange.newCode" class="input" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="one-time-code" required /></label>
              <button type="button" class="btn btn-outline" :disabled="isNewEmailCodeLoading || !emailChange.newEmail.trim()" @click="sendNewEmailCode"><Loader2 v-if="isNewEmailCodeLoading" class="mr-2 h-4 w-4 animate-spin" />{{ t.settings.sendNewEmailCode }}</button>
            </div>
            <button class="btn btn-primary" :disabled="isEmailConfirmLoading || !emailChange.oldCodeSent || !emailChange.newCodeSent"><Loader2 v-if="isEmailConfirmLoading" class="mr-2 h-4 w-4 animate-spin" />{{ t.settings.confirmEmailChange }}</button>
          </form>
        </div>
        <div class="settings-section-header flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.updatePassword }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.updatePasswordDesc }}</p>
        </div>
        <div class="settings-section-content p-6 pt-0">
        <form class="settings-form max-w-xl space-y-4" novalidate @submit.prevent="handleUpdatePassword">
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.currentPassword }}</span><input v-model="password.oldPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.newPassword }}</span><input v-model="password.newPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.confirmNewPassword }}</span><input v-model="password.confirmPassword" class="input" type="password" required /></label>
          <button class="btn btn-primary" :disabled="isPasswordLoading"><Loader2 v-if="isPasswordLoading" class="h-4 w-4 animate-spin mr-2" /> {{ t.settings.updatePasswordBtn }}</button>
        </form>
        </div>

      </div>
    </div>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .settings-main {
    padding: 20px 12px;
  }

  .settings-title-row {
    margin-bottom: 16px;
  }

  .settings-title {
    font-size: 24px;
    line-height: 32px;
  }

  .settings-stack > :not([hidden]) ~ :not([hidden]) {
    margin-top: 12px;
  }

  .settings-tabs-shell {
    padding: 8px 12px 0;
  }

  .settings-tabs {
    column-gap: 24px;
    row-gap: 0;
  }

  .settings-tab {
    height: 42px;
    padding: 0 4px;
  }

  .settings-panel {
    border-radius: 8px;
  }

  .settings-section-header {
    gap: 4px;
    padding: 20px 16px 12px;
  }

  .settings-section-header h2 {
    font-size: 18px;
    line-height: 24px;
  }

  .settings-section-header p {
    line-height: 20px;
  }

  .settings-section-content {
    padding: 0 16px 20px;
  }

  .settings-form > :not([hidden]) ~ :not([hidden]) {
    margin-top: 12px;
  }

  .settings-form .grid {
    gap: 12px;
  }

  .settings-form label > * + * {
    margin-top: 6px;
  }

}
</style>
