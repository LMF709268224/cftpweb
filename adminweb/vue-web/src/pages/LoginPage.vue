<script setup lang="ts">
import { GraduationCap, Loader2, ShieldCheck } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"

const error = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.login)
const layoutCopy = computed(() => t.value.layout)

function reload() {
  window.location.reload()
}

onMounted(async () => {
  try {
    const callback = encodeURIComponent(`${window.location.origin}/callback`)
    const data = await apiClient<{ url?: string }>(`/api/auth/login-url?callback=${callback}`)
    if (data?.url) {
      window.location.href = data.url
      return
    }
    error.value = copy.value.missingUrl
  } catch (err) {
    console.error(err)
    error.value = copy.value.initFailed
  }
})
</script>

<template>
  <div class="flex min-h-screen bg-canvas">
    <!-- Brand panel -->
    <div
      class="relative hidden w-[46%] flex-col justify-between overflow-hidden p-12 text-white lg:flex"
      style="background: linear-gradient(135deg, #4f46e5 0%, #6366f1 45%, #8b5cf6 100%)"
    >
      <!-- Decorative grid -->
      <svg class="pointer-events-none absolute inset-0 h-full w-full opacity-[0.14]" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <defs>
          <pattern id="aurora-grid" width="48" height="48" patternUnits="userSpaceOnUse">
            <path d="M 48 0 L 0 0 0 48" fill="none" stroke="white" stroke-width="1" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#aurora-grid)" />
      </svg>
      <!-- Glow accents -->
      <div class="pointer-events-none absolute -top-32 -right-32 h-96 w-96 rounded-full bg-white/15 blur-3xl" aria-hidden="true" />
      <div class="pointer-events-none absolute -bottom-40 -left-24 h-[420px] w-[420px] rounded-full bg-indigo-300/20 blur-3xl" aria-hidden="true" />

      <div class="animate-rise relative flex items-center gap-3" style="animation-delay: 60ms">
        <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-white/15 shadow-sm backdrop-blur">
          <GraduationCap class="h-6 w-6" />
        </div>
        <div class="leading-tight">
          <div class="text-lg font-semibold tracking-tight">CFTP</div>
          <div class="text-xs text-white/70">{{ layoutCopy.systemName }}</div>
        </div>
      </div>

      <div class="animate-rise relative" style="animation-delay: 160ms">
        <h2 class="max-w-md text-4xl font-semibold leading-tight tracking-tight">
          CFTP
          <span class="mt-2 block text-white/85">{{ layoutCopy.systemName }}</span>
        </h2>
        <div class="mt-8 h-1 w-16 rounded-full bg-white/40" aria-hidden="true" />
      </div>

      <div class="animate-rise relative text-xs text-white/60" style="animation-delay: 260ms">
        CFTP · {{ layoutCopy.systemName }}
      </div>
    </div>

    <!-- Auth panel -->
    <div class="flex flex-1 items-center justify-center px-6 py-12">
      <div class="animate-rise w-full max-w-sm">
        <div class="ui-card p-8 text-center">
          <div class="mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded-xl text-white shadow-sm lg:hidden" style="background: linear-gradient(135deg, #6366f1, #8b5cf6)">
            <GraduationCap class="h-6 w-6" />
          </div>
          <div
            class="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl"
            :class="error ? 'bg-danger-50 text-danger-600' : 'bg-primary-50 text-primary-600'"
          >
            <ShieldCheck v-if="error" class="h-7 w-7" />
            <Loader2 v-else class="h-7 w-7 animate-spin" />
          </div>
          <h1 class="text-xl font-semibold tracking-tight text-ink">
            {{ error ? copy.errorTitle : copy.loadingTitle }}
          </h1>
          <p class="mt-3 text-sm leading-6 text-slate-500">
            {{ error || copy.loadingDescription }}
          </p>
          <button
            v-if="error"
            class="ui-btn ui-btn-primary mt-6 w-full"
            type="button"
            @click="reload"
          >
            {{ copy.retry }}
          </button>
        </div>
        <p class="mt-6 text-center text-xs text-slate-400">CFTP · {{ layoutCopy.systemName }}</p>
      </div>
    </div>
  </div>
</template>
