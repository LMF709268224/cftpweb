<script setup lang="ts">
import { LockKeyhole } from "lucide-vue-next"
import { RouterLink } from "vue-router"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
</script>

<template>
  <div data-testid="resource-access-notice" class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-left text-amber-950" role="status">
    <div class="flex items-center gap-2 font-semibold">
      <LockKeyhole class="h-4 w-4 shrink-0" aria-hidden="true" />
      {{ t.resourceAccess.title }}
    </div>
    <p class="mt-2 text-sm leading-6">{{ t.resourceAccess.description }}</p>
    <RouterLink to="/membership" class="mt-3 inline-flex min-h-9 items-center rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90">
      {{ t.resourceAccess.membershipAction }}
    </RouterLink>
  </div>
</template>
