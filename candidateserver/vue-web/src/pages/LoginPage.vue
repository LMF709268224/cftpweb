<script setup lang="ts">
import { onMounted } from "vue"
import { useTranslation } from "@/lib/language"
import { apiClient } from "@/lib/apiClient"

const { t } = useTranslation()

onMounted(async () => {
  try {
    const callbackUrl = encodeURIComponent(window.location.origin + "/callback")
    const resData = await apiClient(`/api/auth/login-url?callback=${callbackUrl}`)
    if (resData?.url) {
      window.location.href = resData.url
      return
    }
    throw new Error("AUTH_FAILED")
  } catch (err) {
    console.error(err)
  }
})
</script>

<template>
  <div class="flex min-h-screen w-full items-center justify-center bg-slate-950 text-slate-50">
    <div class="flex flex-col items-center gap-4">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-indigo-500 border-r-transparent" />
      <p class="text-slate-400">{{ t.loginPage.connecting }}...</p>
    </div>
  </div>
</template>
