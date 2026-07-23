<script setup lang="ts">
import { Toaster } from "vue-sonner"
import { useUser } from "@/lib/user"
import { onErrorCaptured, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { isAuthenticated } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

const { fetchUser } = useUser()
const { t } = useTranslation()
const route = useRoute()
const router = useRouter()
const appError = ref("")

function pageTitle() {
  const titleKey = String(route.meta.titleKey || "home")
  const pageTitles = t.value.pageTitles as Record<string, string>
  const title = pageTitles[titleKey] || t.value.app.defaultPageTitle
  return `${title} - ${t.value.app.titleSuffix}`
}

watch(
  () => [route.fullPath, t.value] as const,
  () => {
    document.title = pageTitle()
    document.documentElement.lang = t.value.app.htmlLang
  },
  { immediate: true },
)

function shouldFetchUser() {
  return isAuthenticated() && route.path !== "/login" && route.path !== "/callback"
}

onMounted(() => {
  if (shouldFetchUser()) void fetchUser()
})

onErrorCaptured((err) => {
  console.error("Unhandled page error:", err)
  appError.value = t.value.app.pageErrorMessage
  return false
})

watch(
  () => route.fullPath,
  () => {
    appError.value = ""
    if (shouldFetchUser()) void fetchUser()
  },
)

function reloadPage() {
  window.location.reload()
}

function goHome() {
  appError.value = ""
  router.push("/")
}
</script>

<template>
  <RouterView v-if="!appError" />
  <div v-else class="flex min-h-screen items-center justify-center bg-[#eef8f7] p-6">
    <div class="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 text-center shadow-[0_20px_45px_rgba(15,74,82,0.12)]">
      <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-amber-50 text-2xl text-amber-600">!</div>
      <h1 class="text-xl font-bold text-slate-900">{{ t.app.pageErrorTitle }}</h1>
      <p class="mt-2 text-sm text-slate-600">{{ appError }}</p>
      <div class="mt-5 flex justify-center gap-3">
        <button class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50" @click="goHome">
          {{ t.app.backHome }}
        </button>
        <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-600" @click="reloadPage">
          {{ t.app.reload }}
        </button>
      </div>
    </div>
  </div>
  <Toaster rich-colors position="top-center" />
</template>
