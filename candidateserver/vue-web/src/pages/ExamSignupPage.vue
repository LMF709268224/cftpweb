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

onMounted(async () => {
  try {
    const res = await apiClient("/api/user/me")
    if (res) {
      formData.email = res.email || formData.email
      formData.gender = res.gender || formData.gender
      formData.birthdate = res.birthday ? res.birthday.split("T")[0] : formData.birthdate
      formData.first_name = res.first_name || (res.real_name ? res.real_name.split(" ")[0] : formData.first_name)
      formData.last_name = res.last_name || (res.real_name?.includes(" ") ? res.real_name.split(" ").slice(1).join(" ") : formData.last_name)
      formData.home_phone = res.phone || formData.home_phone
      formData.country = res.region || formData.country
      formData.city = res.location || formData.city
      formData.address = res.address?.length > 0 ? res.address.join(", ") : formData.address
    }
  } catch (err) {
    console.error("Failed to load user profile", err)
  }
})

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
    ["work_phone", t.value.examSignup.formWorkPhone],
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
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formWorkPhone }} *</span><input v-model="formData.work_phone" class="input" type="tel" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formHomePhone }} *</span><input v-model="formData.home_phone" class="input" type="tel" required /></label>
        </div>
        <div class="flex justify-end pt-4">
          <button class="btn btn-primary w-full sm:w-auto" :disabled="loading">
            <template v-if="loading">{{ t.examSignup.submitting }}</template>
            <template v-else><Send class="mr-2 h-4 w-4" /> {{ t.examSignup.submit }}</template>
          </button>
        </div>
      </form>
    </div>
  </AppShell>
</template>
