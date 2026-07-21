<script setup lang="ts">
import { CheckCircle2, Loader2, ShieldAlert } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ApiError, apiClient } from "@/lib/apiClient"
import { setAuthSession } from "@/lib/authStorage"
import { useAdminLanguage } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const status = ref<"loading" | "success" | "error">("loading")
const error = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.callback)

function friendlyAuthError(err: unknown) {
  if (err instanceof ApiError) {
    const payload = err.payload as { error_code?: string; message?: string } | null
    const code = String(payload?.error_code || "").toUpperCase()
    const message = String(payload?.message || err.message || "")
    if (code === "AUTH_FAILED" || message.toLowerCase().includes("admin")) {
      return copy.value.notAdmin
    }
    if (code === "INVALID_TOKEN" || message.toLowerCase().includes("application")) {
      return copy.value.wrongApplication
    }
    return message || copy.value.authFailed
  }
  return err instanceof Error && err.message ? err.message : copy.value.authFailed
}

onMounted(async () => {
  const code = String(route.query.code || "")
  const state = String(route.query.state || "")
  if (!code || !state) {
    status.value = "error"
    error.value = copy.value.missingParams
    setTimeout(() => router.push("/login"), 2500)
    return
  }

  try {
    const payload = await apiClient<{ user?: { name?: string } }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
    })

    setAuthSession(payload.user?.name)
    status.value = "success"
    setTimeout(() => router.push("/lms"), 800)
  } catch (err) {
    console.error(err)
    status.value = "error"
    error.value = friendlyAuthError(err)
    setTimeout(() => router.push("/login"), 2500)
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-950 px-6 text-white">
    <div class="w-full max-w-sm rounded-3xl border border-white/10 bg-white/5 p-8 text-center shadow-2xl backdrop-blur">
      <div class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl" :class="status === 'error' ? 'bg-red-500/20 text-red-300' : status === 'success' ? 'bg-emerald-500/20 text-emerald-300' : 'bg-sky-500/20 text-sky-300'">
        <Loader2 v-if="status === 'loading'" class="h-8 w-8 animate-spin" />
        <CheckCircle2 v-else-if="status === 'success'" class="h-8 w-8" />
        <ShieldAlert v-else class="h-8 w-8" />
      </div>
      <h1 class="text-2xl font-black">
        {{ status === "loading" ? copy.loadingTitle : status === "success" ? copy.successTitle : copy.errorTitle }}
      </h1>
      <p class="mt-3 text-sm leading-6 text-slate-300">
        {{ status === "loading" ? copy.loadingDescription : status === "success" ? copy.successDescription : error }}
      </p>
    </div>
  </div>
</template>
