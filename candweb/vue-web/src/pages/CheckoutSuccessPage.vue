<script setup lang="ts">
import { computed, onMounted } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { CheckCircle2, ChevronRight, Home } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { useTranslation } from "@/lib/language"
import { useUser } from "@/lib/user"
import { apiClient } from "@/lib/apiClient"

const route = useRoute()
const { t } = useTranslation()
const { currentUser } = useUser()

const orderId = computed(() => route.params.orderId as string)
const candUlid = computed(() => currentUser.value?.cand_ulid || currentUser.value?.ulid || "")

onMounted(async () => {
  if (!orderId.value) return
  const evidenceStr = sessionStorage.getItem(`bundle_evidence_${orderId.value}`)
  if (!evidenceStr) return
  
  try {
    const evidence = JSON.parse(evidenceStr)
    const { files, unitQualMap } = evidence
    
    // Process each unit's evidence
    const promises = Object.keys(unitQualMap).map(async (unitId) => {
      const qualId = unitQualMap[unitId]
      const fileData = files[unitId]
      // reasons[unitId] is available if backend supports it
      if (!qualId || !fileData) return
      
      // Find the application created by the bundle purchase
      const res = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`)
      const app = (res?.applications || [])[0]
      if (!app?.app_ulid) return
      
      // Attach the evidence
      const payload = {
        app_id: app.app_ulid,
        files: [{
          file_name: fileData.name,
          file_url: fileData.url,
          file_hash: fileData.hash,
          file_ext: fileData.ext,
          file_size: fileData.size,
          file_usage: "EXEMPTION_EVIDENCE",
          file_type: 1
        }]
      }
      await apiClient("/api/credentials/update", { method: "PUT", body: JSON.stringify(payload) })
    })
    
    await Promise.all(promises)
    sessionStorage.removeItem(`bundle_evidence_${orderId.value}`)
  } catch (err) {
    console.error("Failed to auto-submit evidence:", err)
    // Don't toast error here, it might just mean the application isn't ready yet or already submitted
  }
})
</script>

<template>
  <AppShell>
    <main class="flex min-h-[70vh] items-center justify-center p-6">
      <div class="w-full max-w-lg rounded-[24px] bg-white p-10 text-center shadow-[0_12px_40px_rgba(15,74,82,0.06)]">
        <div class="mx-auto mb-6 flex h-24 w-24 items-center justify-center rounded-full bg-emerald-50 text-emerald-500">
          <CheckCircle2 class="h-12 w-12" />
        </div>
        
        <h1 class="mb-3 text-3xl font-bold tracking-tight text-slate-900">{{ t.common?.success || 'Payment Successful' }}</h1>
        <p class="mb-8 text-slate-500 text-lg">
          Your order has been placed.
        </p>

        <div class="mb-8 rounded-2xl border border-slate-100 bg-slate-50 p-6 text-left">
          <div class="flex flex-col gap-4">
            <div>
              <div class="text-sm font-medium text-slate-500">Order ID</div>
              <div class="mt-1 font-mono text-slate-900">{{ orderId }}</div>
            </div>
            <div v-if="candUlid">
              <div class="text-sm font-medium text-slate-500">Candidate ID</div>
              <div class="mt-1 font-mono text-slate-900 font-semibold">{{ candUlid }}</div>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-3">
          <RouterLink to="/my-certifications" class="btn btn-primary w-full text-base h-12">
            View My Certifications <ChevronRight class="ml-2 h-5 w-5" />
          </RouterLink>
          <RouterLink to="/courses" class="btn btn-outline w-full text-base h-12">
            <Home class="mr-2 h-4 w-4" /> Go to Mall
          </RouterLink>
        </div>
      </div>
    </main>
  </AppShell>
</template>
