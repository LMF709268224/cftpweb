<script setup lang="ts">
import { TriangleAlert } from "lucide-vue-next"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
</script>

<template>
  <div role="note" class="flex gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-amber-950">
    <TriangleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-700" aria-hidden="true" />
    <div class="min-w-0 space-y-1">
      <p class="text-sm font-semibold">{{ t.settings.identityNameNoticeTitle }}</p>
      <p class="text-sm leading-6">{{ t.settings.identityNameNoticeBody }}</p>
    </div>
  </div>
</template>
