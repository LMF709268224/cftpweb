<script setup lang="ts">
import { CheckCircle2, GraduationCap, Loader2, ShieldAlert } from "lucide-vue-next"
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
const layoutCopy = computed(() => t.value.layout)

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
    const payload = await apiClient<{ token?: string; user?: { name?: string } }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
    })
    if (!payload.token) {
      throw new Error("missing token")
    }

    setAuthSession(payload.token, payload.user?.name)
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
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-canvas px-6">
    <!-- Soft aurora glow backdrop -->
    <div
      class="pointer-events-none absolute -top-40 left-1/2 h-[420px] w-[720px] -translate-x-1/2 rounded-full opacity-40 blur-3xl"
      style="background: linear-gradient(135deg, #e0e7ff, #ede9fe)"
      aria-hidden="true"
    />

    <div class="animate-rise relative w-full max-w-sm">
      <div class="ui-card p-8 text-center">
        <div class="mx-auto mb-6 flex items-center justify-center gap-2.5">
          <div class="flex h-9 w-9 items-center justify-center rounded-[10px] text-white shadow-sm" style="background: linear-gradient(135deg, #6366f1, #8b5cf6)">
            <GraduationCap class="h-5 w-5" />
          </div>
          <span class="text-sm font-semibold tracking-tight text-ink">CFTP</span>
        </div>
        <div
          class="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl"
          :class="status === 'error' ? 'bg-danger-50 text-danger-600' : status === 'success' ? 'bg-success-50 text-success-600' : 'bg-primary-50 text-primary-600'"
        >
          <Loader2 v-if="status === 'loading'" class="h-7 w-7 animate-spin" />
          <CheckCircle2 v-else-if="status === 'success'" class="h-7 w-7" />
          <ShieldAlert v-else class="h-7 w-7" />
        </div>
        <h1 class="text-xl font-semibold tracking-tight text-ink">
          {{ status === "loading" ? copy.loadingTitle : status === "success" ? copy.successTitle : copy.errorTitle }}
        </h1>
        <p class="mt-3 text-sm leading-6 text-slate-500">
          {{ status === "loading" ? copy.loadingDescription : status === "success" ? copy.successDescription : error }}
        </p>
      </div>
      <p class="mt-6 text-center text-xs text-slate-400">CFTP · {{ layoutCopy.systemName }}</p>
    </div>
  </div>
</template>
