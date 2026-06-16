<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, Send } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const unitId = String(route.query.unitId || "")
const pipelineId = String(route.query.pipelineId || "")
const courseId = String(route.query.courseId || "")
const loading = ref(false)
const syncLoading = ref(false)
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
  home_phone: "",
  work_phone: "",
})
const backLink = computed(() => pipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : courseId ? `/courses/learn?courseId=${encodeURIComponent(courseId)}` : "/courses")

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
  formData.gender = profile.gender || formData.gender
  formData.birthdate = normalizeDate(profile.birthday) || formData.birthdate
  formData.first_name = profile.first_name || realName.firstName || formData.first_name
  formData.last_name = profile.last_name || realName.lastName || formData.last_name
  formData.home_phone = profile.home_phone || profile.phone || formData.home_phone
  formData.work_phone = profile.work_phone || formData.work_phone
  formData.country = profile.country || profile.region || formData.country
  formData.province = profile.province || formData.province
  formData.city = profile.city || profile.location || formData.city
  formData.address = normalizeAddress(profile.address_text, profile.address) || formData.address
  formData.postal_code = profile.postal_code || formData.postal_code
}

function firstFilled(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim()
  }
  return ""
}

function fillOnlyWhenEmpty(current: unknown, next: unknown) {
  const currentValue = firstFilled(current)
  return currentValue || firstFilled(next)
}

function buildProfilePayload(current: any) {
  const currentAddress = normalizeAddress(current.address_text, current.address)
  return {
    display_name: current.display_name || "",
    email: fillOnlyWhenEmpty(current.email, formData.email),
    first_name: fillOnlyWhenEmpty(current.first_name, formData.first_name),
    last_name: fillOnlyWhenEmpty(current.last_name, formData.last_name),
    home_phone: fillOnlyWhenEmpty(current.home_phone || current.phone, formData.home_phone),
    work_phone: fillOnlyWhenEmpty(current.work_phone, formData.work_phone),
    gender: fillOnlyWhenEmpty(current.gender, formData.gender),
    birthday: fillOnlyWhenEmpty(normalizeDate(current.birthday), formData.birthdate),
    country: fillOnlyWhenEmpty(current.country || current.region, formData.country),
    province: fillOnlyWhenEmpty(current.province, formData.province),
    city: fillOnlyWhenEmpty(current.city || current.location, formData.city),
    address: fillOnlyWhenEmpty(currentAddress, formData.address),
    postal_code: fillOnlyWhenEmpty(current.postal_code, formData.postal_code),
    affiliation: current.affiliation || "",
    title: current.title || "",
    real_name: current.real_name || "",
    bio: current.bio || "",
    education: current.education || "",
  }
}

onMounted(async () => {
  try {
    const res = await apiClient("/api/user/me")
    if (res) {
      applyProfileToForm(res)
    }
  } catch (err) {
    console.error("Failed to load user profile", err)
  }
})

async function handleSyncToProfile() {
  syncLoading.value = true
  try {
    const current = await apiClient("/api/user/me")
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify(buildProfilePayload(current || {})),
    })
    toast.success(t.value.examSignup.syncProfileSuccess)
  } finally {
    syncLoading.value = false
  }
}

async function handleSubmit() {
  if (!unitId) {
    toast.error(t.value.common.error)
    return
  }
  const requiredFields = [
    ["first_name", t.value.examSignup.formFirstName],
    ["last_name", t.value.examSignup.formLastName],
    ["email", t.value.examSignup.formEmail],
    ["gender", t.value.examSignup.formGender],
    ["birthdate", t.value.examSignup.formBirthdate],
    ["country", t.value.examSignup.formCountry],
    ["province", t.value.examSignup.formProvince],
    ["city", t.value.examSignup.formCity],
    ["address", t.value.examSignup.formAddress],
    ["postal_code", t.value.examSignup.formPostalCode],
    ["home_phone", t.value.examSignup.formHomePhone],
  ] as const
  for (const [key, label] of requiredFields) {
    if (!String(formData[key]).trim()) {
      toast.error(t.value.examSignup.validationRequired.replace("{{field}}", label))
      return
    }
  }
  loading.value = true
  try {
    await apiClient(`/api/exams/units/${encodeURIComponent(unitId)}/signup`, { method: "POST", body: JSON.stringify(formData) })
    toast.success(t.value.examSignup.success)
    router.push("/exams")
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppShell content-class="p-4">
    <RouterLink :to="backLink" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" /> {{ t.examSignup.backToCourse }}
    </RouterLink>
    <div class="mb-8 max-w-2xl">
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.examSignup.title }}</h1>
      <p class="mt-2 text-muted-foreground">{{ t.examSignup.subtitle }}</p>
    </div>
    <div class="max-w-2xl rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <form class="space-y-6" @submit.prevent="handleSubmit">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formFirstName }} *</span><input v-model="formData.first_name" class="input" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formLastName }} *</span><input v-model="formData.last_name" class="input" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formMiddleName }}</span><input v-model="formData.middle_name" class="input" /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formEmail }} *</span><input v-model="formData.email" class="input" type="email" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formGender }} *</span><input v-model="formData.gender" class="input" placeholder="Male / Female" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formBirthdate }} *</span><input v-model="formData.birthdate" class="input" type="date" required /></label>
        <div class="grid gap-4 sm:grid-cols-3">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formCountry }} *</span><input v-model="formData.country" class="input" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formProvince }} *</span><input v-model="formData.province" class="input" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formCity }} *</span><input v-model="formData.city" class="input" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formAddress }} *</span><input v-model="formData.address" class="input" required /></label>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formPostalCode }} *</span><input v-model="formData.postal_code" class="input" required /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formWorkPhone }}</span><input v-model="formData.work_phone" class="input" type="tel" /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formHomePhone }} *</span><input v-model="formData.home_phone" class="input" type="tel" required /></label>
        </div>
        <div class="rounded-xl border border-emerald-100 bg-emerald-50/60 p-4 text-sm text-muted-foreground">
          <p>{{ t.examSignup.syncToProfileHint }}</p>
          <button type="button" class="btn btn-outline mt-3" :disabled="syncLoading || loading" @click="handleSyncToProfile">
            <template v-if="syncLoading">{{ t.examSignup.syncingToProfile }}</template>
            <template v-else>{{ t.examSignup.syncToProfile }}</template>
          </button>
        </div>
        <div class="flex justify-end pt-2">
          <button class="btn btn-primary w-full sm:w-auto" :disabled="loading">
            <template v-if="loading">{{ t.examSignup.submitting }}</template>
            <template v-else><Send class="mr-2 h-4 w-4" /> {{ t.examSignup.submit }}</template>
          </button>
        </div>
      </form>
    </div>
  </AppShell>
</template>
