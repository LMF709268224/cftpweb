<script setup lang="ts">
import { Ellipsis } from "lucide-vue-next"
import { ref } from "vue"
import { useAdminLanguage } from "@/lib/language"

defineProps<{ canViewEnrollments: boolean }>()
const emit = defineEmits<{ details: []; enrollments: [] }>()
const { t } = useAdminLanguage()
const disclosure = ref<HTMLDetailsElement | null>(null)
const trigger = ref<HTMLElement | null>(null)

function close(event: KeyboardEvent) {
  if (!disclosure.value?.open) return
  event.stopPropagation()
  disclosure.value.open = false
  trigger.value?.focus()
}
</script>

<template>
  <details ref="disclosure" class="w-fit max-w-full" @keydown.esc="close">
    <summary ref="trigger" class="flex min-h-10 w-fit cursor-pointer list-none items-center gap-2 rounded-lg border border-slate-200 px-3 text-sm font-bold text-blue-700 hover:bg-blue-50 [&::-webkit-details-marker]:hidden" :aria-label="t.dashboard.table.actions">
      <Ellipsis class="h-5 w-5" aria-hidden="true" />
      <span class="sr-only">{{ t.dashboard.table.actions }}</span>
    </summary>
    <div class="mt-2 grid gap-1 rounded-xl border border-slate-200 bg-white p-1">
      <button class="min-h-10 rounded-lg px-3 py-2 text-left text-sm font-bold text-blue-700 hover:bg-blue-50" type="button" @click="emit('details')">{{ t.dashboard.viewDetails }}</button>
      <button v-if="canViewEnrollments" class="min-h-10 rounded-lg px-3 py-2 text-left text-sm font-bold text-blue-700 hover:bg-blue-50" type="button" @click="emit('enrollments')">{{ t.dashboard.enrollments.view }}</button>
    </div>
  </details>
</template>
