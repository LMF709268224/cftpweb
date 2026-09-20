<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue"
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
let redirectTimer: ReturnType<typeof setTimeout> | undefined
let isUnmounted = false

function scheduleRedirect(path: string, delay: number) {
  if (isUnmounted) return

  if (redirectTimer !== undefined) clearTimeout(redirectTimer)
  redirectTimer = setTimeout(() => {
    redirectTimer = undefined
    void router.replace(path)
  }, delay)
}

onBeforeUnmount(() => {
  isUnmounted = true
  if (redirectTimer !== undefined) clearTimeout(redirectTimer)
  redirectTimer = undefined
})

onMounted(async () => {
  const code = String(route.query.code || "")
  const state = String(route.query.state || "")
  const currentLang = lang.value

  if (!code || !state) {
    status.value = "error"
    errorMsg.value = getErrorMessage("INVALID_REQUEST", currentLang)
    consumePostLoginRedirect()
    scheduleRedirect("/", 3000)
    return
  }

  try {
    const payload = await apiClient("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
      suppressAuthRedirect: true,
    })
    setAuthSession(payload.user?.name)
    const postLoginRedirect = consumePostLoginRedirect()
    const redirectPath = postLoginRedirect || "/dashboard"
    status.value = "success"
    scheduleRedirect(redirectPath, 1000)
  } catch (err: any) {
    status.value = "error"
    errorMsg.value = getErrorMessage(err instanceof ApiClientError ? err.errorCode || "AUTH_FAILED" : err?.message || "AUTH_FAILED", currentLang)
    consumePostLoginRedirect()
    scheduleRedirect("/", 3000)
  }
})
</script>

<template>
  <div class="candidate-portal app-min-viewport-height app-safe-area-screen flex w-full flex-col items-center justify-center bg-[#edeef2] text-[#1a2233]">
    <div class="mx-4 flex w-full max-w-sm flex-col items-center rounded-[14px] border border-[#002a66]/15 bg-white p-8 text-center">
      <template v-if="status === 'loading'">
        <Loader2 class="h-8 w-8 animate-spin text-[#0957f9]" :stroke-width="1.5" />
        <h2 class="mt-6 text-xl font-semibold text-[#002a66]">{{ t.callbackPage.loadingTitle }}</h2>
        <p class="mt-2 text-sm text-[#525e70]">{{ t.callbackPage.loadingDesc }}</p>
      </template>
      <template v-else-if="status === 'success'">
        <CheckCircle2 class="h-8 w-8 text-[#2f7d4f]" :stroke-width="1.5" />
        <h2 class="mt-6 text-xl font-semibold text-[#002a66]">{{ t.callbackPage.successTitle }}</h2>
        <p class="mt-2 text-sm text-[#525e70]">{{ t.callbackPage.successDesc }}</p>
      </template>
      <template v-else>
        <ShieldAlert class="h-8 w-8 text-[#b3372f]" :stroke-width="1.5" />
        <h2 class="mt-6 text-xl font-semibold text-[#002a66]">{{ t.callbackPage.errorTitle }}</h2>
        <p class="mt-3 w-full rounded-lg border border-[#b3372f]/20 bg-[#fbedeb] p-3 text-sm text-[#b3372f]" role="alert">{{ errorMsg }}</p>
        <p class="mt-4 text-xs text-[#5b6b87]">{{ t.callbackPage.redirectLoginHint }}</p>
      </template>
    </div>
  </div>
</template>
