<script setup lang="ts">
import { computed } from "vue"
import { Loader2 } from "lucide-vue-next"

const props = withDefaults(defineProps<{
  label?: string
  variant?: "page" | "section" | "inline"
  rows?: number
}>(), {
  label: "Loading...",
  variant: "section",
  rows: 3,
})

const skeletonRows = computed(() => Array.from({ length: Math.max(1, props.rows) }, (_, index) => index))
</script>

<template>
  <div
    v-if="variant === 'inline'"
    class="flex items-center justify-center gap-2 text-sm text-muted-foreground"
    role="status"
    aria-live="polite"
  >
    <Loader2 class="h-4 w-4 animate-spin text-primary" />
    <span>{{ label }}</span>
  </div>

  <div
    v-else
    :class="[
      'overflow-hidden rounded-md border border-slate-100 bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]',
      variant === 'page' ? 'p-5 md:p-6' : 'p-4',
    ]"
    role="status"
    aria-live="polite"
    aria-busy="true"
  >
    <span class="sr-only">{{ label }}</span>
    <div class="animate-pulse">
      <div class="flex items-start gap-4">
        <div
          :class="[
            'shrink-0 rounded-full bg-slate-100',
            variant === 'page' ? 'h-12 w-12' : 'h-9 w-9',
          ]"
        />
        <div class="min-w-0 flex-1 space-y-3">
          <div class="h-4 w-44 max-w-full rounded-full bg-slate-100" />
          <div class="h-3 w-3/4 rounded-full bg-slate-100" />
          <div v-if="variant === 'page'" class="flex flex-wrap gap-2 pt-1">
            <div class="h-6 w-20 rounded-full bg-slate-100" />
            <div class="h-6 w-24 rounded-full bg-slate-100" />
            <div class="h-6 w-16 rounded-full bg-slate-100" />
          </div>
        </div>
      </div>

      <div v-if="variant === 'page'" class="mt-6 grid gap-3 md:grid-cols-4">
        <div v-for="item in 4" :key="item" class="h-20 rounded-md bg-slate-50" />
      </div>

      <div class="mt-5 space-y-3">
        <div
          v-for="row in skeletonRows"
          :key="row"
          :class="[
            'rounded-md bg-slate-50',
            variant === 'page' ? 'h-14' : 'h-10',
          ]"
        />
      </div>
    </div>
  </div>
</template>
