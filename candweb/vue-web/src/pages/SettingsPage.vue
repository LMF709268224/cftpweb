<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
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
  CN_CITY_OPTIONS_BY_STATE,
  CN_STATE_LABELS,
  countryUsesCityField,
  getCachedCountries,
  getCountryCityOptions,
  getCountryOptions,
  getProvinceOptions,
  getStateCityOptions,
  loadLocationData,
  type CountryOption,
} from "@/lib/locationOptions"
import { GENDER_OPTIONS, PROFILE_TEXT_LIMITS, isValidEmail, isValidInternationalPhone, isValidPostalCode, normalizeGender, normalizeInternationalPhone, normalizePostalCode, trimToMax } from "@/lib/profileFormValidation"
import { useUser } from "@/lib/user"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const { fetchUser } = useUser()
const activeTab = ref(String(route.query.tab || "profile"))
const profile = reactive({
  name: "",
  displayName: "",
  email: "",
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
  realName: "",
  bio: "",
  education: "",
})
const password = reactive({ oldPassword: "", newPassword: "", confirmPassword: "" })
const emailUpdate = reactive({ newEmail: "", verificationCode: "" })
const isProfileLoading = ref(false)
const isPasswordLoading = ref(false)
const isEmailUpdating = ref(false)
const isEmailCodeSending = ref(false)
const emailCodeCountdown = ref(0)
const resendCodeText = computed(() => t.value.settings.resendCode.replace('{{seconds}}', String(emailCodeCountdown.value)))
let emailCodeInterval: number | undefined
const selectedCountryCode = ref("")
const selectedProvinceCode = ref("")
const orgPhonePrefixes = ref<{ code: string, dialCode: string, name: string }[]>([])
const countryOptions = ref<CountryOption[]>([])
const provinceOptions = ref<any[]>([])
const cityOptions = ref<any[]>([])
const showProvinceField = computed(() => !selectedCountryCode.value || provinceOptions.value.length > 0)
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
    if (lang.value === "zh" && selectedCountryCode.value === "CN" && CN_CITY_OPTIONS_BY_STATE[selectedProvinceCode.value]) {
      cityOptions.value = CN_CITY_OPTIONS_BY_STATE[selectedProvinceCode.value].map((name) => ({ name, localizedName: name }))
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
  selectedCountryCode.value = matchedCountry?.isoCode || ""
  refreshProvinceOptions()
  if (selectedCountryCode.value && provinceOptions.value.length === 0) {
    profile.province = ""
  }

  const provinceText = normalizeLocationText(profile.province)
  const matchedProvince = selectedCountryCode.value
    ? provinceOptions.value.find((state) => provinceMatchValues(state).some((value) => normalizeProvinceText(value) === normalizeProvinceText(provinceText)))
    : undefined
  selectedProvinceCode.value = matchedProvince?.isoCode || ""
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
  profile.displayName = trimToMax(profile.displayName, PROFILE_TEXT_LIMITS.name)
  profile.realName = trimToMax(profile.realName, PROFILE_TEXT_LIMITS.name)
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
      profile.name = payload.name || ""
      profile.displayName = payload.display_name || ""
      profile.email = payload.email || ""
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
      profile.realName = payload.real_name || ""
      profile.bio = payload.bio || ""
      profile.education = payload.education || ""
      await locationReady
      syncLocationSelectionFromProfile()
      
      const configRes = await apiClient("/api/public/config/organization")
      if (configRes && configRes.country_codes) {
        const allCountries = getCachedCountries()
        orgPhonePrefixes.value = configRes.country_codes.map((code: string) => {
          const country = allCountries.find(c => c.isoCode === code)
          return {
            code,
            dialCode: country ? `+${country.phonecode}` : code,
            name: country ? country.name : code,
          }
        })
        if (!profile.phoneCountryCode && orgPhonePrefixes.value.length > 0) {
          profile.phoneCountryCode = orgPhonePrefixes.value[0].code
        }
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

  const requiredFields = [
    [profile.email, t.value.settings.email],
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
  isProfileLoading.value = true
  try {
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify({
        display_name: profile.displayName,
        email: profile.email,
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
        real_name: profile.realName,
        bio: profile.bio,
        education: profile.education,
      }),
    })
    await fetchUser(true)
    toast.success(getMessage("PROFILE_UPDATE_SUCCESS", lang.value))
    if (profile.displayName) {
      localStorage.setItem("user_name", profile.displayName)
      window.dispatchEvent(new Event("storage"))
    }
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

async function handleSendEmailCode() {
  if (!emailUpdate.newEmail) {
    toast.error(t.value.settings.validationRequired.replace("{{field}}", t.value.settings.newEmail))
    return
  }
  if (!isValidEmail(emailUpdate.newEmail)) {
    toast.error(t.value.settings.validationInvalidEmail)
    return
  }
  isEmailCodeSending.value = true
  try {
    await apiClient("/api/user/profile/email/send-code", {
      method: "POST",
      body: JSON.stringify({ email: emailUpdate.newEmail, lang: lang.value }),
    })
    toast.success(t.value.settings.codeSent)
    emailCodeCountdown.value = 60
    emailCodeInterval = window.setInterval(() => {
      emailCodeCountdown.value--
      if (emailCodeCountdown.value <= 0) {
        clearInterval(emailCodeInterval)
      }
    }, 1000)
  } catch (e) {
    // Error is handled by apiClient toast
  } finally {
    isEmailCodeSending.value = false
  }
}

async function handleUpdateEmail() {
  if (!emailUpdate.newEmail || !emailUpdate.verificationCode) {
    toast.error(t.value.settings.validationRequired.replace("{{field}}", !emailUpdate.newEmail ? t.value.settings.newEmail : t.value.settings.verificationCode))
    return
  }
  if (!isValidEmail(emailUpdate.newEmail)) {
    toast.error(t.value.settings.validationInvalidEmail)
    return
  }
  isEmailUpdating.value = true
  try {
    await apiClient("/api/user/profile/email", {
      method: "PUT",
      body: JSON.stringify({ email: emailUpdate.newEmail, verification_code: emailUpdate.verificationCode }),
    })
    toast.success(t.value.settings.updateEmailSuccess)
    emailUpdate.newEmail = ""
    emailUpdate.verificationCode = ""
    clearInterval(emailCodeInterval)
    emailCodeCountdown.value = 0
    await fetchUser() // fetch latest profile to update global user
    const payload = await apiClient("/api/user/me")
    if (payload && payload.email) profile.email = payload.email
  } catch (e) {
    // Error is handled by apiClient toast
  } finally {
    isEmailUpdating.value = false
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

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6 flex items-center justify-between space-y-2">
          <h1 class="text-3xl font-bold tracking-tight">{{ t.settings.title }}</h1>
        </div>
    <div class="space-y-4">
      <div class="rounded-[14px] bg-white px-5 pt-4 shadow-[0_10px_24px_rgba(15,74,82,0.04)] md:px-6">
        <div class="flex flex-wrap gap-x-8 gap-y-2 border-b border-border">
          <button
            :class="['relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === 'profile' ? 'text-primary' : 'text-foreground hover:text-primary']"
            @click="setActiveTab('profile')"
          >
            {{ t.settings.profileTab }}
            <span v-if="activeTab === 'profile'" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
          <button
            :class="['relative inline-flex cursor-pointer items-center whitespace-nowrap px-1 pb-5 text-base font-medium transition-colors duration-200', activeTab === 'account' ? 'text-primary' : 'text-foreground hover:text-primary']"
            @click="setActiveTab('account')"
          >
            {{ t.settings.accountTab }}
            <span v-if="activeTab === 'account'" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
        </div>
      </div>
      <div v-if="activeTab === 'profile'" class="rounded-md bg-white text-card-foreground">
        <div class="flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.profileTab }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.profileDesc }}</p>
        </div>
        <div class="p-6 pt-0">
        <form class="max-w-2xl space-y-4" novalidate @submit.prevent="handleUpdateProfile">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.loginId }}</span><input v-model="profile.name" class="input bg-muted" disabled /></label>
            <label class="space-y-2">
              <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.email }}</span>
              <input v-model="profile.email" class="input bg-muted" disabled required />
              <p class="text-xs text-muted-foreground">{{ t.settings.emailChangeHint }}</p>
            </label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.displayName }}</span><input v-model="profile.displayName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.displayNamePlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.realName }}</span><input v-model="profile.realName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.realNamePlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.firstName }}</span><input v-model="profile.firstName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.firstNamePlaceholder" required /></label>
            <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.lastName }}</span><input v-model="profile.lastName" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" :placeholder="t.settings.lastNamePlaceholder" required /></label>
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
                <select v-if="orgPhonePrefixes.length > 0" v-model="profile.phoneCountryCode" class="input cursor-pointer w-28 shrink-0">
                  <option v-for="prefix in orgPhonePrefixes" :key="prefix.code" :value="prefix.code">{{ prefix.dialCode }}</option>
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
          </div>
          <button class="btn btn-primary" :disabled="isProfileLoading"><Loader2 v-if="isProfileLoading" class="h-4 w-4 animate-spin" /> {{ t.common.save }}</button>
        </form>
        </div>
      </div>
      <div v-if="activeTab === 'account'" class="rounded-md bg-white text-card-foreground">
        <div class="flex flex-col space-y-1.5 p-6">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.updatePassword }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.updatePasswordDesc }}</p>
        </div>
        <div class="p-6 pt-0">
        <form class="max-w-xl space-y-4" novalidate @submit.prevent="handleUpdatePassword">
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.currentPassword }}</span><input v-model="password.oldPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.newPassword }}</span><input v-model="password.newPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.confirmNewPassword }}</span><input v-model="password.confirmPassword" class="input" type="password" required /></label>
          <button class="btn btn-primary" :disabled="isPasswordLoading"><Loader2 v-if="isPasswordLoading" class="h-4 w-4 animate-spin mr-2" /> {{ t.settings.updatePasswordBtn }}</button>
        </form>
        </div>

        <div class="flex flex-col space-y-1.5 p-6 mt-6 border-t border-border">
          <h2 class="text-xl font-semibold leading-none tracking-tight">{{ t.settings.updateEmail }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.settings.updateEmailDesc }}</p>
        </div>
        <div class="p-6 pt-0">
          <form class="max-w-xl space-y-4" novalidate @submit.prevent="handleUpdateEmail">
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.newEmail }}</span><input v-model="emailUpdate.newEmail" class="input" type="email" required /></label>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.settings.verificationCode }}</span>
              <div class="flex gap-2">
                <input v-model="emailUpdate.verificationCode" class="input flex-1" type="text" required />
                <button type="button" class="btn btn-outline" :disabled="isEmailCodeSending || emailCodeCountdown > 0" @click="handleSendEmailCode">
                  <Loader2 v-if="isEmailCodeSending" class="h-4 w-4 animate-spin mr-2" />
                  {{ emailCodeCountdown > 0 ? resendCodeText : t.settings.sendCode }}
                </button>
              </div>
            </label>
            <button class="btn btn-primary" :disabled="isEmailUpdating"><Loader2 v-if="isEmailUpdating" class="h-4 w-4 animate-spin mr-2" /> {{ t.settings.updateEmailBtn }}</button>
          </form>
        </div>
      </div>
    </div>
      </main>
    </div>
  </AppShell>
</template>
