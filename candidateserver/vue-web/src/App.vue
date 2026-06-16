<script setup lang="ts">
import { Toaster } from "vue-sonner"
import { useUser } from "@/lib/user"
import { onErrorCaptured, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { getAccessToken } from "@/lib/authStorage"

const { fetchUser } = useUser()
const route = useRoute()
const router = useRouter()
const appError = ref("")

function shouldFetchUser() {
  return Boolean(getAccessToken()) && route.path !== "/login" && route.path !== "/callback"
}

onMounted(() => {
  if (shouldFetchUser()) void fetchUser()
})

onErrorCaptured((err) => {
  console.error("Unhandled page error:", err)
  appError.value = "页面加载失败，请重试或返回首页。"
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
      <h1 class="text-xl font-bold text-slate-900">页面暂时无法显示</h1>
      <p class="mt-2 text-sm text-slate-600">{{ appError }}</p>
      <div class="mt-5 flex justify-center gap-3">
        <button class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50" @click="goHome">
          返回首页
        </button>
        <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-600" @click="reloadPage">
          重新加载
        </button>
      </div>
    </div>
  </div>
  <Toaster rich-colors position="top-center" />
</template>
