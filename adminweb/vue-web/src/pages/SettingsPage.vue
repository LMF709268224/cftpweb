<script setup lang="ts">
import { Loader2, RefreshCw, Save, Shield, UserRound } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession, setAuthSession } from "@/lib/authStorage"
import type { JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

type ProfileForm = {
  name: string
  display_name: string
  email: string
  affiliation: string
  title: string
  real_name: string
  bio: string
  gender: string
  birthday: string
  education: string
}

type SettingsSection = "profile" | "password"

const GENDER_MALE = "\u7537"
const GENDER_FEMALE = "\u5973"
const BIO_MAX_LENGTH = 1000

const router = useRouter()
const { t } = useAdminLanguage()
const copy = computed(() => t.value.settings)
const activeSection = ref<SettingsSection>("profile")
const profileLoading = ref(false)
const profileSaving = ref(false)
const passwordSaving = ref(false)

const profile = ref<ProfileForm>({
  name: "",
  display_name: "",
  email: "",
  affiliation: "",
  title: "",
  real_name: "",
  bio: "",
  gender: "",
  birthday: "",
  education: "",
})

const oldPassword = ref("")
const newPassword = ref("")
const confirmPassword = ref("")

const sections = computed(() => [
  {
    key: "profile" as const,
    title: copy.value.sections.profile.title,
    description: copy.value.sections.profile.description,
    count: profile.value.display_name || profile.value.name ? 1 : 0,
  },
  {
    key: "password" as const,
    title: copy.value.sections.password.title,
    description: copy.value.sections.password.description,
    count: 1,
  },
])

function normalizeGender(value: unknown) {
  const text = String(value || "").trim().toLowerCase()
  if (text === "male" || text === "m" || text === GENDER_MALE) return GENDER_MALE
  if (text === "female" || text === "f" || text === GENDER_FEMALE) return GENDER_FEMALE
  return ""
}

function normalizeBirthday(value: unknown) {
  const text = String(value || "").trim()
  if (!text) return ""
  return text.slice(0, 10)
}

function applyProfile(data: JsonRecord) {
  profile.value = {
    name: String(data.name || ""),
    display_name: String(data.display_name || ""),
    email: String(data.email || ""),
    affiliation: String(data.affiliation || ""),
    title: String(data.title || ""),
    real_name: String(data.real_name || ""),
    bio: String(data.bio || ""),
    gender: normalizeGender(data.gender),
    birthday: normalizeBirthday(data.birthday),
    education: String(data.education || ""),
  }
}

async function loadProfile() {
  profileLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/user/me")
    applyProfile(data || {})
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.profileLoadFailed)
  } finally {
    profileLoading.value = false
  }
}

async function saveProfile() {
  profileSaving.value = true
  try {
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify({
        display_name: profile.value.display_name.trim(),
        affiliation: profile.value.affiliation.trim(),
        title: profile.value.title.trim(),
        real_name: profile.value.real_name.trim(),
        bio: profile.value.bio.trim(),
        gender: profile.value.gender,
        birthday: profile.value.birthday,
        education: profile.value.education.trim(),
      }),
    })
    setAuthSession("", profile.value.display_name.trim() || profile.value.name || "Admin")
    window.dispatchEvent(new Event("storage"))
    toast.success(copy.value.toasts.profileSaved)
    await loadProfile()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.profileSaveFailed))
  } finally {
    profileSaving.value = false
  }
}

async function savePassword() {
  if (!oldPassword.value || !newPassword.value) {
    toast.error(copy.value.toasts.passwordRequired)
    return
  }
  if (newPassword.value.length < 8) {
    toast.error(copy.value.toasts.passwordTooShort)
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error(copy.value.toasts.passwordMismatch)
    return
  }

  passwordSaving.value = true
  try {
    await apiClient("/api/user/password", {
      method: "PUT",
      body: JSON.stringify({
        old_password: oldPassword.value,
        new_password: newPassword.value,
      }),
    })
    toast.success(copy.value.toasts.passwordUpdated)
    clearAuthSession()
    setTimeout(() => router.push("/login"), 800)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.passwordUpdateFailed))
  } finally {
    passwordSaving.value = false
  }
}

onMounted(() => {
  const params = new URLSearchParams(window.location.search)
  if (params.get("tab") === "password" || params.get("tab") === "account") activeSection.value = "password"
  void loadProfile()
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadProfile">
        <RefreshCw class="h-4 w-4" :class="profileLoading ? 'animate-spin' : ''" />
        {{ copy.reload }}
      </button>
    </header>

    <div class="flex flex-col gap-5 md:gap-6">
      <section class="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm md:rounded-3xl">
        <div class="grid gap-3 md:grid-cols-2">
        <button
          v-for="section in sections"
          :key="section.key"
          class="grid min-h-20 w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 rounded-2xl border border-transparent px-4 py-4 text-left transition hover:bg-sky-50 md:px-5"
          :class="activeSection === section.key ? 'border-sky-200 bg-sky-50 shadow-sm shadow-sky-100' : ''"
          type="button"
          @click="activeSection = section.key"
        >
          <div class="min-w-0">
            <div class="break-words font-black">{{ section.title }}</div>
            <div class="mt-1 break-words text-sm text-slate-500">{{ section.description }}</div>
          </div>
          <span class="h-fit shrink-0 rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">{{ section.count }}</span>
        </button>
        </div>
      </section>

      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
        <div class="flex items-center justify-between gap-4 border-b border-slate-100 px-4 py-4 md:px-5">
          <div class="min-w-0">
            <h2 class="text-xl font-black">{{ activeSection === "profile" ? copy.sections.profile.title : copy.sections.password.title }}</h2>
            <p class="mt-1 text-sm text-slate-500">
              {{ activeSection === "profile" ? copy.sections.profile.detail : copy.sections.password.detail }}
            </p>
          </div>
          <UserRound v-if="activeSection === 'profile'" class="h-5 w-5 shrink-0 text-blue-600" />
          <Shield v-else class="h-5 w-5 shrink-0 text-blue-600" />
        </div>

        <form v-if="activeSection === 'profile'" class="space-y-5 p-4 md:space-y-6 md:p-5" @submit.prevent="saveProfile">
          <div v-if="profileLoading" class="rounded-2xl bg-slate-50 px-4 py-8 text-center text-slate-500 md:p-8">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loadingProfile }}
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.loginId }}</span>
              <input v-model="profile.name" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" disabled />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.email }}</span>
              <input v-model="profile.email" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" disabled />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.displayName }}</span>
              <input v-model.trim="profile.display_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="80" :placeholder="copy.placeholders.displayName" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.realName }}</span>
              <input v-model.trim="profile.real_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="80" :placeholder="copy.placeholders.realName" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.gender }}</span>
              <select v-model="profile.gender" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                <option value="">{{ copy.placeholders.selectGender }}</option>
                <option :value="GENDER_MALE">{{ copy.genders.male }}</option>
                <option :value="GENDER_FEMALE">{{ copy.genders.female }}</option>
              </select>
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.birthday }}</span>
              <input v-model="profile.birthday" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="date" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.affiliation }}</span>
              <input v-model.trim="profile.affiliation" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="160" :placeholder="copy.placeholders.affiliation" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.labels.title }}</span>
              <input v-model.trim="profile.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="120" :placeholder="copy.placeholders.title" />
            </label>
            <label class="block md:col-span-2">
              <span class="text-sm font-bold">{{ copy.labels.education }}</span>
              <input v-model.trim="profile.education" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="200" :placeholder="copy.placeholders.education" />
            </label>
            <label class="block md:col-span-2">
              <span class="text-sm font-bold">{{ copy.labels.bio }}</span>
              <textarea v-model.trim="profile.bio" class="mt-2 min-h-28 w-full rounded-xl border border-slate-200 p-4" :maxlength="BIO_MAX_LENGTH" :placeholder="copy.placeholders.bio" />
              <span class="mt-1 block text-right text-xs text-slate-400">{{ copy.bioCount(profile.bio.length, BIO_MAX_LENGTH) }}</span>
            </label>
          </div>

          <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50 md:w-auto" :disabled="profileSaving" type="submit">
            <Loader2 v-if="profileSaving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ profileSaving ? copy.saving : copy.saveProfile }}
          </button>
        </form>

        <form v-else class="space-y-5 p-4 md:space-y-6 md:p-5" @submit.prevent="savePassword">
          <div class="grid gap-4 md:grid-cols-2">
            <label class="block">
              <span class="inline-flex items-center text-sm font-bold">
                <span class="mr-1 text-red-500" aria-hidden="true">*</span>
                {{ copy.labels.oldPassword }}
              </span>
              <input v-model="oldPassword" autocomplete="current-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" required type="password" />
            </label>
            <label class="block">
              <span class="inline-flex items-center text-sm font-bold">
                <span class="mr-1 text-red-500" aria-hidden="true">*</span>
                {{ copy.labels.newPassword }}
              </span>
              <input v-model="newPassword" autocomplete="new-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" minlength="8" required type="password" />
              <span class="mt-1 block text-xs text-slate-400">{{ copy.passwordHint }}</span>
            </label>
            <label class="block md:col-span-2">
              <span class="inline-flex items-center text-sm font-bold">
                <span class="mr-1 text-red-500" aria-hidden="true">*</span>
                {{ copy.labels.confirmPassword }}
              </span>
              <input v-model="confirmPassword" autocomplete="new-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" minlength="8" required type="password" />
            </label>
          </div>

          <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50 md:w-auto" :disabled="passwordSaving" type="submit">
            <Loader2 v-if="passwordSaving" class="h-4 w-4 animate-spin" />
            <Shield v-else class="h-4 w-4" />
            {{ passwordSaving ? copy.updating : copy.updatePassword }}
          </button>
        </form>
      </section>
    </div>
  </section>
</template>
