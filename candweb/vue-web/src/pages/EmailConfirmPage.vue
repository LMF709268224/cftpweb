<script setup lang="ts">
import { onMounted, ref } from "vue"
import { CheckCircle2, Loader2, ShieldAlert } from "lucide-vue-next"
import { useRoute } from "vue-router"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const { t } = useTranslation()
const status = ref<"loading" | "pending" | "updated" | "error">("loading")

onMounted(async () => {
  const id = String(route.query.id || "")
  const side = String(route.query.side || "")
  const token = String(route.query.token || "")
  window.history.replaceState(window.history.state, "", "/email-confirm")
  if (!id || !token || !["old", "new"].includes(side)) {
    status.value = "error"
    return
  }
  try {
    const result = await apiClient("/api/auth/email/confirm", {
      method: "POST",
      body: JSON.stringify({ id, side, token }),
      suppressAuthRedirect: true,
      suppressErrorToast: true,
    })
    status.value = result.status === "updated" ? "updated" : "pending"
    if (status.value === "updated") clearAuthSession()
  } catch {
    status.value = "error"
  }
})
</script>

<template>
  <main class="candidate-portal app-min-viewport-height app-safe-area-screen flex w-full flex-col items-center justify-center bg-[#edeef2] px-4 text-[#1a2233]">
    <div class="w-full max-w-sm rounded-md border border-slate-200 bg-white p-8 text-center">
      <Loader2 v-if="status === 'loading'" class="mx-auto h-8 w-8 animate-spin text-primary" />
      <CheckCircle2 v-else-if="status !== 'error'" class="mx-auto h-8 w-8 text-emerald-700" />
      <ShieldAlert v-else class="mx-auto h-8 w-8 text-red-700" />
      <h1 class="mt-5 text-lg font-semibold">{{ t.settings.changeEmail }}</h1>
      <p class="mt-3 text-sm text-muted-foreground" role="status">
        {{ status === 'loading' ? '...' : status === 'updated' ? t.settings.emailConfirmUpdated : status === 'pending' ? t.settings.emailConfirmPending : t.settings.emailConfirmFailed }}
      </p>
      <a v-if="status !== 'loading'" :href="status === 'updated' ? '/login' : '/'" class="btn btn-primary mt-6 inline-flex">
        {{ status === 'updated' ? t.settings.emailConfirmSignIn : t.settings.emailConfirmBack }}
      </a>
    </div>
  </main>
</template>
