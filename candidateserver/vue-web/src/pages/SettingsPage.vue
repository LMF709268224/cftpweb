<script setup lang="ts">
import { onMounted, reactive, ref } from "vue"
import { useRoute } from "vue-router"
import { toast } from "vue-sonner"
import { Loader2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { getMessage } from "@/lib/messages"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const { t, lang } = useTranslation()
const activeTab = ref(String(route.query.tab || "profile"))
const profile = reactive({ name: "", displayName: "", email: "", affiliation: "", title: "", realName: "", bio: "", gender: "", birthday: "", education: "" })
const password = reactive({ oldPassword: "", newPassword: "", confirmPassword: "" })
const isProfileLoading = ref(false)
const isPasswordLoading = ref(false)

onMounted(async () => {
  try {
    const payload = await apiClient("/api/user/me")
    if (payload) {
      profile.name = payload.name || ""
      profile.displayName = payload.display_name || ""
      profile.email = payload.email || ""
      profile.affiliation = payload.affiliation || ""
      profile.title = payload.title || ""
      profile.realName = payload.real_name || ""
      profile.bio = payload.bio || ""
      profile.gender = payload.gender || ""
      profile.birthday = payload.birthday || ""
      profile.education = payload.education || ""
    }
  } catch {
    // apiClient handles toast.
  }
})

async function handleUpdateProfile() {
  isProfileLoading.value = true
  try {
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify({
        display_name: profile.displayName,
        email: profile.email,
        affiliation: profile.affiliation,
        title: profile.title,
        real_name: profile.realName,
        bio: profile.bio,
        gender: profile.gender,
        birthday: profile.birthday,
        education: profile.education,
      }),
    })
    toast.success(getMessage("PROFILE_UPDATE_SUCCESS", lang.value))
    if (profile.displayName) {
      localStorage.setItem("user_name", profile.displayName)
      window.dispatchEvent(new Event("storage"))
    }
  } finally {
    isProfileLoading.value = false
  }
}

async function handleUpdatePassword() {
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
    localStorage.removeItem("is_authenticated")
    localStorage.removeItem("user_name")
    setTimeout(() => { window.location.href = "/login" }, 1500)
  } finally {
    isPasswordLoading.value = false
  }
}
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 flex items-center justify-between space-y-2">
      <h1 class="text-3xl font-bold tracking-tight">{{ t.settings.title }}</h1>
    </div>
    <div class="space-y-4">
      <div class="rounded-md bg-white px-8 pt-6">
        <div class="flex flex-wrap gap-10 border-b border-[#edf0f2]">
          <button
            :class="['relative cursor-pointer px-1 pb-7 text-base font-medium transition-colors duration-200', activeTab === 'profile' ? 'text-primary' : 'text-[#111827] hover:text-primary']"
            @click="activeTab = 'profile'"
          >
            {{ t.settings.profileTab }}
            <span v-if="activeTab === 'profile'" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
          <button
            :class="['relative cursor-pointer px-1 pb-7 text-base font-medium transition-colors duration-200', activeTab === 'account' ? 'text-primary' : 'text-[#111827] hover:text-primary']"
            @click="activeTab = 'account'"
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
        <form class="max-w-2xl space-y-4" @submit.prevent="handleUpdateProfile">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.loginId }}</span><input v-model="profile.name" class="input bg-muted" disabled /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.email }}</span><input v-model="profile.email" class="input bg-muted" disabled /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.displayName }}</span><input v-model="profile.displayName" class="input" :placeholder="t.settings.displayNamePlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.realName }}</span><input v-model="profile.realName" class="input" :placeholder="t.settings.realNamePlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.gender }}</span><input v-model="profile.gender" class="input" :placeholder="t.settings.genderPlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.birthday }}</span><input v-model="profile.birthday" class="input" type="date" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.affiliation }}</span><input v-model="profile.affiliation" class="input" :placeholder="t.settings.affiliationPlaceholder" /></label>
            <label class="space-y-2"><span class="text-sm font-medium">{{ t.settings.jobTitle }}</span><input v-model="profile.title" class="input" :placeholder="t.settings.jobTitlePlaceholder" /></label>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium">{{ t.settings.education }}</span><input v-model="profile.education" class="input" :placeholder="t.settings.educationPlaceholder" /></label>
            <label class="space-y-2 md:col-span-2"><span class="text-sm font-medium">{{ t.settings.bio }}</span><textarea v-model="profile.bio" class="textarea" :placeholder="t.settings.bioPlaceholder" rows="3" /></label>
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
        <form class="max-w-xl space-y-4" @submit.prevent="handleUpdatePassword">
          <label class="block space-y-2"><span class="text-sm font-medium">{{ t.settings.currentPassword }}</span><input v-model="password.oldPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium">{{ t.settings.newPassword }}</span><input v-model="password.newPassword" class="input" type="password" required /></label>
          <label class="block space-y-2"><span class="text-sm font-medium">{{ t.settings.confirmNewPassword }}</span><input v-model="password.confirmPassword" class="input" type="password" required /></label>
          <button class="btn btn-primary" :disabled="isPasswordLoading"><Loader2 v-if="isPasswordLoading" class="h-4 w-4 animate-spin" /> {{ t.settings.updatePasswordBtn }}</button>
        </form>
        </div>
      </div>
    </div>
  </AppShell>
</template>
