<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { CheckCircle2, Loader2, ShieldAlert } from "lucide-vue-next"
import { getErrorMessage } from "@/lib/errorCodes"
import { ApiClientError, apiClient } from "@/lib/apiClient"
import { consumePostLoginRedirect, setAuthSession } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const status = ref<"loading" | "success" | "error">("loading")
const errorMsg = ref("")

onMounted(async () => {
  const code = String(route.query.code || "")
  const state = String(route.query.state || "")
  const currentLang = lang.value

  if (!code || !state) {
    status.value = "error"
    errorMsg.value = getErrorMessage("INVALID_REQUEST", currentLang)
    setTimeout(() => router.push("/login"), 3000)
    return
  }

  try {
    const payload = await apiClient("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
    })
    setAuthSession(payload.user?.name)
    status.value = "success"
    setTimeout(() => router.push(consumePostLoginRedirect() || "/"), 1000)
  } catch (err: any) {
    status.value = "error"
    errorMsg.value = getErrorMessage(err instanceof ApiClientError ? err.errorCode || "AUTH_FAILED" : err?.message || "AUTH_FAILED", currentLang)
    setTimeout(() => router.push("/login"), 3000)
  }
})
</script>

<template>
  <div class="relative flex min-h-screen w-full flex-col items-center justify-center overflow-hidden bg-slate-950 text-slate-50">
    <div class="pointer-events-none absolute left-1/2 top-1/2 h-[800px] w-[800px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-indigo-600/10 blur-[150px]" />
    <div class="relative z-10 mx-4 flex w-full max-w-sm flex-col items-center rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur-xl">
      <template v-if="status === 'loading'">
        <Loader2 class="h-16 w-16 animate-spin text-indigo-400" />
        <h2 class="mt-8 text-xl font-semibold tracking-tight text-white">{{ t.callbackPage.loadingTitle }}</h2>
        <p class="mt-2 text-center text-sm text-slate-400">{{ t.callbackPage.loadingDesc }}</p>
      </template>
      <template v-else-if="status === 'success'">
        <CheckCircle2 class="h-16 w-16 text-emerald-400" />
        <h2 class="mt-8 text-xl font-semibold tracking-tight text-white">{{ t.callbackPage.successTitle }}</h2>
        <p class="mt-2 text-sm text-slate-400">{{ t.callbackPage.successDesc }}</p>
      </template>
      <template v-else>
        <ShieldAlert class="h-16 w-16 text-red-400" />
        <h2 class="mt-8 text-xl font-semibold tracking-tight text-white">{{ t.callbackPage.errorTitle }}</h2>
        <p class="mt-2 w-full rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-center text-sm text-red-300">{{ errorMsg }}</p>
        <p class="mt-4 text-xs text-slate-500">{{ t.callbackPage.redirectLoginHint }}</p>
      </template>
    </div>
  </div>
</template>
