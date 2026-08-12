<script setup lang="ts">
import { computed } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { CheckCircle2, ChevronRight, Home } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { useTranslation } from "@/lib/language"
import { useUser } from "@/lib/user"

const route = useRoute()
const { t } = useTranslation()
const { currentUser } = useUser()

const orderId = computed(() => route.params.orderId as string)
const candUlid = computed(() => currentUser.value?.cand_ulid || currentUser.value?.ulid || "")

</script>

<template>
  <AppShell>
    <main class="checkout-success-page flex min-h-[70vh] items-center justify-center p-6">
      <div class="checkout-success-card w-full max-w-lg rounded-[24px] bg-white p-10 text-center shadow-[0_12px_40px_rgba(15,74,82,0.06)]">
        <div class="checkout-success-icon mx-auto mb-6 flex h-24 w-24 items-center justify-center rounded-full bg-emerald-50 text-emerald-500">
          <CheckCircle2 class="h-12 w-12" />
        </div>
        
        <h1 class="checkout-success-title mb-3 text-3xl font-bold tracking-tight text-slate-900">{{ t.checkoutSuccess.title }}</h1>
        <p class="checkout-success-description mb-8 text-slate-500 text-lg">
          {{ t.checkoutSuccess.orderPlaced }}
        </p>

        <div class="checkout-success-details mb-8 rounded-2xl border border-slate-100 bg-slate-50 p-6 text-left">
          <div class="flex flex-col gap-4">
            <div>
              <div class="text-sm font-medium text-slate-500">{{ t.checkoutSuccess.orderId }}</div>
              <div class="checkout-success-id mt-1 font-mono text-slate-900">{{ orderId }}</div>
            </div>
            <div v-if="candUlid">
              <div class="text-sm font-medium text-slate-500">{{ t.checkoutSuccess.candidateId }}</div>
              <div class="checkout-success-id mt-1 font-mono text-slate-900 font-semibold">{{ candUlid }}</div>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-3">
          <RouterLink to="/my-certifications" class="btn btn-primary w-full text-base h-12">
            {{ t.checkoutSuccess.viewCertifications }} <ChevronRight class="ml-2 h-5 w-5" />
          </RouterLink>
          <RouterLink to="/courses" class="btn btn-outline w-full text-base h-12">
            <Home class="mr-2 h-4 w-4" /> {{ t.checkoutSuccess.goToMall }}
          </RouterLink>
        </div>
      </div>
    </main>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .checkout-success-page {
    min-height: calc(100vh - 32px);
    min-height: calc(100dvh - 32px);
    padding: 8px 0;
  }

  .checkout-success-card {
    padding: 20px;
    border: 1px solid var(--border);
    border-radius: 12px;
    box-shadow: none;
  }

  .checkout-success-icon {
    width: 64px;
    height: 64px;
    margin-bottom: 16px;
  }

  .checkout-success-icon > svg {
    width: 32px;
    height: 32px;
  }

  .checkout-success-title {
    margin-bottom: 8px;
  }

  .checkout-success-description {
    margin-bottom: 24px;
    font-size: 16px;
    line-height: 24px;
  }

  .checkout-success-details {
    margin-bottom: 24px;
    padding: 16px;
    border-radius: 8px;
  }

  .checkout-success-id {
    overflow-wrap: anywhere;
    word-break: break-word;
    font-size: 13px;
    line-height: 20px;
  }
}
</style>
